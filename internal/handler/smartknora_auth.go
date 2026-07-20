package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/auth"
	"github.com/Tencent/WeKnora/internal/types"
)

// SmartKnoraAuthHandler handles smartKnora-specific authentication.
// Extends WeKnora auth with phone/email login and dual-token system.
type SmartKnoraAuthHandler struct {
	db         *gorm.DB
	jwtManager *auth.JWTManager
	redis      *redis.Client
}

// NewSmartKnoraAuthHandler creates a new auth handler.
func NewSmartKnoraAuthHandler(db *gorm.DB, jwtManager *auth.JWTManager, redis *redis.Client) *SmartKnoraAuthHandler {
	return &SmartKnoraAuthHandler{
		db:         db,
		jwtManager: jwtManager,
		redis:      redis,
	}
}

// RegisterRoutes registers smartKnora auth routes.
func (h *SmartKnoraAuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.RefreshToken)
		auth.POST("/logout", h.Logout)
	}
}

// Register handles user registration via phone or email.
func (h *SmartKnoraAuthHandler) Register(c *gin.Context) {
	var req types.SmartKnoraRegisterRequest
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
		var profile types.SmartKnoraUserProfile
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
		user.Email = req.Phone + "@smartknora.local"
	}

	// Generate ID
	user.ID = uuid.New().String()

	// Provision a dedicated tenant for every self-service registration. The
	// database sequence is the source of truth, so concurrent registrations
	// cannot accidentally share a tenant ID.
	tenant := types.Tenant{
		Name:        user.Username + " 的工作区",
		Description: "SmartKnora 默认工作区",
		APIKey:      uuid.New().String(),
		Status:      "active",
		Business:    "smartknora",
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
	profile := types.SmartKnoraUserProfile{
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
		if err := tx.Create(&tenant).Error; err != nil {
			return err
		}
		user.TenantID = tenant.ID
		org.OwnerTenantID = tenant.ID

		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		if err := tx.Create(&types.TenantMember{
			UserID:    user.ID,
			TenantID:  tenant.ID,
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
			TenantID:      tenant.ID,
			AuthStatus:    "trial",
			AuthExpiresAt: &trialExpiresAt,
			CreatedAt:     nowOrg,
			UpdatedAt:     nowOrg,
		}).Error; err != nil {
			return err
		}
		if err := tx.Create(&types.SmartKnoraOrgMember{
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
	accessToken, _, err := h.jwtManager.GenerateAccessToken(user.ID, user.TenantID, h.resolveUserRole(user.ID))
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

	c.JSON(http.StatusCreated, types.SmartKnoraAuthResponse{
		Success:      true,
		Token:        accessToken,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    h.jwtManager.AccessExpirySeconds(),
		User:         &user,
	})
}

// Login handles user login via phone or email + password.
func (h *SmartKnoraAuthHandler) Login(c *gin.Context) {
	var req types.SmartKnoraLoginRequest
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
		var profile types.SmartKnoraUserProfile
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
	accessToken, _, err := h.jwtManager.GenerateAccessToken(user.ID, user.TenantID, h.resolveUserRole(user.ID))
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

	c.JSON(http.StatusOK, types.SmartKnoraAuthResponse{
		Success:      true,
		Token:        accessToken,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    h.jwtManager.AccessExpirySeconds(),
		User:         &user,
	})
}

// RefreshToken handles token refresh.
func (h *SmartKnoraAuthHandler) RefreshToken(c *gin.Context) {
	var req types.SmartKnoraRefreshRequest
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
	accessToken, _, err := h.jwtManager.GenerateAccessToken(user.ID, user.TenantID, h.resolveUserRole(user.ID))
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
func (h *SmartKnoraAuthHandler) Logout(c *gin.Context) {
	var req types.SmartKnoraRefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokenHash := auth.HashRefreshToken(req.RefreshToken)
	h.db.Where("token_hash = ?", tokenHash).Delete(&types.RefreshToken{})

	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// generateUUID returns a new UUID string. Uses google/uuid.
func generateUUID() string {
	return uuid.New().String()
}

// resolveUserRole determines the JWT role for a user.
// Returns "admin" if the user is a system admin, otherwise "member".
func (h *SmartKnoraAuthHandler) resolveUserRole(userID string) string {
	var user types.User
	if err := h.db.Select("is_system_admin").Where("id = ?", userID).First(&user).Error; err == nil && user.IsSystemAdmin {
		return "admin"
	}
	return "member"
}
