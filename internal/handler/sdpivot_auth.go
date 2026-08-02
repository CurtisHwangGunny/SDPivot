package handler

import (
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/auth"
	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
)

// SDPivotAuthHandler handles SDPivot-specific authentication.
// Extends WeKnora auth with phone/email login and dual-token system.
type SDPivotAuthHandler struct {
	db         *gorm.DB
	jwtManager *auth.JWTManager
	redis      *redis.Client
}

// NewSDPivotAuthHandler creates a new auth handler.
func NewSDPivotAuthHandler(db *gorm.DB, jwtManager *auth.JWTManager, redis *redis.Client) *SDPivotAuthHandler {
	return &SDPivotAuthHandler{
		db:         db,
		jwtManager: jwtManager,
		redis:      redis,
	}
}

// RegisterRoutes registers SDPivot auth routes.
func (h *SDPivotAuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.RefreshToken)
		auth.POST("/logout", h.Logout)
	}
}

// RegisterProtectedRoutes registers authenticated user profile routes.
func (h *SDPivotAuthHandler) RegisterProtectedRoutes(rg *gin.RouterGroup) {
	rg.GET("/auth/me", h.GetCurrentUser)
}

// GetCurrentUser returns the authenticated user's non-sensitive profile.
func (h *SDPivotAuthHandler) GetCurrentUser(c *gin.Context) {
	db := middleware.TenantDB(c, h.db)
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)
	type currentUser struct {
		ID                 string           `json:"id"`
		Username           string           `json:"username"`
		Email              string           `json:"email"`
		Avatar             string           `json:"avatar"`
		TenantID           uint64           `json:"tenant_id"`
		IsActive           bool             `json:"is_active"`
		AccessRole         types.AccessRole `json:"access_role"`
		DepartmentID       *string          `json:"department_id"`
		DepartmentName     string           `json:"department_name"`
		Phone              *string          `json:"phone"`
		Nickname           string           `json:"nickname"`
		MustChangePassword bool             `json:"must_change_password"`
	}

	schema := detectOpsUsageStatsSchema(db)
	departmentName := "''"
	joinDepartment := ""
	if schema.HasDepartments {
		departmentName = "COALESCE(d.name, '')"
		joinDepartment = "LEFT JOIN departments d ON d.id = NULLIF(to_jsonb(u)->>'department_id', '') AND d.tenant_id = u.tenant_id AND d.deleted_at IS NULL"
	}
	var user currentUser
	query := db.Table("users u").
		Select(`u.id, u.username, u.email, u.avatar, u.tenant_id, u.is_active,
			COALESCE(to_jsonb(u)->>'access_role', 'knowledge_viewer') AS access_role,
			NULLIF(to_jsonb(u)->>'department_id', '') AS department_id,
			` + departmentName + ` AS department_name,
			p.phone, COALESCE(p.nickname, '') AS nickname,
			COALESCE((to_jsonb(u)->>'must_change_password')::boolean, false) AS must_change_password`).
		Joins("LEFT JOIN smartknora_user_profiles p ON p.user_id = u.id AND p.deleted_at IS NULL")
	if joinDepartment != "" {
		query = query.Joins(joinDepartment)
	}
	err := query.
		Where("u.id = ? AND u.tenant_id = ? AND u.is_active = true AND u.deleted_at IS NULL", userID, tenantID).
		Scan(&user).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load current user"})
		return
	}
	if user.ID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found or inactive"})
		return
	}
	user.AccessRole = types.NormalizeAccessRole(string(user.AccessRole))
	if !user.AccessRole.IsValid() {
		user.AccessRole = types.AccessRoleKnowledgeViewer
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "user": user})
}

// Register handles user registration via phone or email.
func (h *SDPivotAuthHandler) Register(c *gin.Context) {
	var req types.SDPivotRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Phone == "" && req.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone or email is required"})
		return
	}

	// Check if user already exists
	var existingUser types.User

	// Check username conflict (phone used as username, include soft-deleted)
	if req.Phone != "" {
		var existingByName types.User
		if err := h.db.Unscoped().Where("username = ?", req.Phone).First(&existingByName).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "username already taken"})
			return
		}
	}
	if req.Phone != "" {
		// Look up by phone in smartknora_user_profiles
		var profile types.SDPivotUserProfile
		if err := h.db.Unscoped().Where("phone = ?", req.Phone).First(&profile).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "phone already registered"})
			return
		}
	}
	if req.Email != "" {
		if err := h.db.Unscoped().Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
			return
		}
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	// Create WeKnora User
	now := time.Now()
	user := types.User{
		ID:             "",
		Username:       req.Phone,
		Email:          req.Email,
		PasswordHash:   string(hash),
		IsActive:       true,
		TrialStartedAt: &now,
		TrialPhase:     "30day",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if user.Username == "" {
		user.Username = req.Email
	}
	if user.Email == "" {
		user.Email = req.Phone + "@sdpivot.local"
	}

	// Generate ID
	user.ID = uuid.New().String()

	// OP uses one canonical tenant for every locally provisioned user.
	tenant := types.Tenant{
		ID:          types.DefaultTenantID,
		Name:        "SDPivot",
		Description: "SDPivot OP tenant",
		APIKey:      uuid.New().String(),
		Status:      "active",
		Business:    "sdpivot",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	nowOrg := time.Now()
	trialExpiresAt := nowOrg.Add(30 * 24 * time.Hour)
	org := types.Organization{
		ID:         uuid.New().String(),
		Name:       "默认组织",
		OwnerID:    user.ID,
		InviteCode: generateInviteCode(),
		CreatedAt:  nowOrg,
		UpdatedAt:  nowOrg,
	}

	var phone *string
	if req.Phone != "" {
		phone = &req.Phone
	}
	profile := types.SDPivotUserProfile{
		UserID:    user.ID,
		Phone:     phone,
		Nickname:  req.Nickname,
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if profile.Nickname == "" {
		if req.Phone != "" {
			if len(req.Phone) >= 4 {
				profile.Nickname = "用户" + req.Phone[len(req.Phone)-4:]
			} else {
				profile.Nickname = "用户" + req.Phone
			}
		} else {
			profile.Nickname = "用户"
		}
	}

	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", types.DefaultTenantID).FirstOrCreate(&tenant).Error; err != nil {
			return err
		}
		user.TenantID = types.DefaultTenantID
		org.OwnerTenantID = types.DefaultTenantID

		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		if err := tx.Create(&types.TenantMember{
			UserID:    user.ID,
			TenantID:  types.DefaultTenantID,
			Role:      types.TenantRoleOwner,
			Status:    types.TenantMemberStatusActive,
			JoinedAt:  now,
			CreatedAt: now,
			UpdatedAt: now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Create(&org).Error; err != nil {
			return err
		}
		if err := tx.Create(&types.OrgExt{
			OrgID:         org.ID,
			TenantID:      types.DefaultTenantID,
			AuthStatus:    "trial",
			AuthExpiresAt: &trialExpiresAt,
			CreatedAt:     nowOrg,
			UpdatedAt:     nowOrg,
		}).Error; err != nil {
			return err
		}
		if err := tx.Create(&types.SDPivotOrgMember{
			OrgID:    org.ID,
			UserID:   user.ID,
			Role:     "owner",
			Status:   "active",
			JoinedAt: nowOrg,
		}).Error; err != nil {
			return err
		}
		return tx.Create(&profile).Error
	}); err != nil {
		log.Printf("ERROR: failed to provision tenant for user %s: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create account workspace"})
		return
	}

	// Generate tokens
	accessToken, _, err := h.jwtManager.GenerateAccessToken(user.ID, types.DefaultTenantID, h.resolveUserRole(user.ID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	refreshToken, tokenHash, err := h.jwtManager.GenerateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}

	// Store refresh token in DB
	rt := types.RefreshToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		Family:    uuid.New().String(),
		ExpiresAt: now.Add(h.jwtManager.RefreshExpiry()),
		CreatedAt: now,
	}
	if err := h.db.Create(&rt).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store refresh token"})
		return
	}

	// Update last login
	h.db.Model(&user).Update("updated_at", now)

	c.JSON(http.StatusCreated, types.SDPivotAuthResponse{
		Success:      true,
		Token:        accessToken,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    h.jwtManager.AccessExpirySeconds(),
		User:         &user,
	})
}

// Login handles user login via phone or email + password.
func (h *SDPivotAuthHandler) Login(c *gin.Context) {
	var req types.SDPivotLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Phone == "" && req.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone or email is required"})
		return
	}

	// Find user by phone or email
	var user types.User
	var found bool

	if req.Phone != "" {
		// Find by phone in profile table
		var profile types.SDPivotUserProfile
		if err := h.db.Where("phone = ? AND status = 'active'", req.Phone).First(&profile).Error; err == nil {
			if err := h.db.Where("id = ? AND is_active = true", profile.UserID).First(&user).Error; err == nil {
				found = true
			}
		}
	}

	if !found && req.Email != "" {
		if err := h.db.Where("email = ? AND is_active = true", req.Email).First(&user).Error; err == nil {
			found = true
		}
	}

	if !found {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Generate tokens
	now := time.Now()
	accessToken, _, err := h.jwtManager.GenerateAccessToken(user.ID, types.DefaultTenantID, h.resolveUserRole(user.ID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	refreshToken, tokenHash, err := h.jwtManager.GenerateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}

	// Store refresh token
	rt := types.RefreshToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		Family:    uuid.New().String(),
		ExpiresAt: now.Add(h.jwtManager.RefreshExpiry()),
		CreatedAt: now,
	}
	if err := h.db.Create(&rt).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store refresh token"})
		return
	}

	// Update last login time
	h.db.Model(&user).Update("updated_at", now)

	c.JSON(http.StatusOK, types.SDPivotAuthResponse{
		Success:      true,
		Token:        accessToken,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    h.jwtManager.AccessExpirySeconds(),
		User:         &user,
	})
}

// RefreshToken handles token refresh.
func (h *SDPivotAuthHandler) RefreshToken(c *gin.Context) {
	var req types.SDPivotRefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Hash the incoming token to find it in DB
	tokenHash := auth.HashRefreshToken(req.RefreshToken)

	var rt types.RefreshToken
	if err := h.db.Where("token_hash = ? AND expires_at > ?", tokenHash, time.Now()).First(&rt).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired refresh token"})
		return
	}

	// Find the user
	var user types.User
	if err := h.db.Where("id = ?", rt.UserID).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	// Generate new tokens
	now := time.Now()
	accessToken, _, err := h.jwtManager.GenerateAccessToken(user.ID, types.DefaultTenantID, h.resolveUserRole(user.ID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	newRefreshToken, newTokenHash, err := h.jwtManager.GenerateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}

	// Rotate: delete old, create new (token rotation for security)
	if err := h.db.Delete(&rt).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to rotate refresh token"})
		return
	}
	newRT := types.RefreshToken{
		UserID:    user.ID,
		TokenHash: newTokenHash,
		Family:    rt.Family, // Keep same family
		ExpiresAt: now.Add(h.jwtManager.RefreshExpiry()),
		CreatedAt: now,
	}
	if err := h.db.Create(&newRT).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create new refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
		"expires_in":    h.jwtManager.AccessExpirySeconds(),
	})
}

// Logout handles user logout (revoke refresh token).
func (h *SDPivotAuthHandler) Logout(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := h.db
	if refreshToken := strings.TrimSpace(req.RefreshToken); refreshToken != "" {
		query = query.Where("token_hash = ?", auth.HashRefreshToken(refreshToken))
	} else if authorization := c.GetHeader("Authorization"); strings.HasPrefix(authorization, "Bearer ") {
		claims, err := h.jwtManager.ValidateAccessToken(strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer ")))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		query = query.Where("user_id = ?", claims.UserID)
	} else {
		c.JSON(http.StatusOK, gin.H{"message": "logged out"})
		return
	}
	if err := query.Delete(&types.RefreshToken{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to log out"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// generateUUID returns a new UUID string. Uses google/uuid.
func generateUUID() string {
	return uuid.New().String()
}

// resolveUserRole determines the product role embedded in the JWT.
func (h *SDPivotAuthHandler) resolveUserRole(userID string) string {
	var user types.User
	if err := h.db.Select("is_system_admin, is_ops_admin").Where("id = ?", userID).First(&user).Error; err == nil && (user.IsSystemAdmin || user.IsOpsAdmin) {
		return string(types.AccessRoleSuperAdmin)
	}
	if err := h.db.Select("access_role").Where("id = ?", userID).First(&user).Error; err == nil {
		role := types.NormalizeAccessRole(string(user.AccessRole))
		if role.IsValid() {
			return string(role)
		}
	}
	return string(types.AccessRoleKnowledgeViewer)
}
