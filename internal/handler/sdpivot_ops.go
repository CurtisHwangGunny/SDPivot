package handler

import (
	"net/http"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/auth"
	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
)

// SDPivotOpsHandler handles operations admin authentication (PRD 1.1.4).
type SDPivotOpsHandler struct {
	db         *gorm.DB
	jwtManager *auth.JWTManager
}

// NewSDPivotOpsHandler creates a new ops handler.
func NewSDPivotOpsHandler(db *gorm.DB, jwtManager *auth.JWTManager) *SDPivotOpsHandler {
	return &SDPivotOpsHandler{db: db, jwtManager: jwtManager}
}

// RegisterRoutes registers ops auth routes.
func (h *SDPivotOpsHandler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	ops := rg.Group("/ops")
	{
		ops.POST("/login", h.OpsLogin)
		ops.POST("/refresh", h.OpsRefreshToken)
	}
}

func (h *SDPivotOpsHandler) RegisterProtectedRoutes(rg *gin.RouterGroup) {
	ops := rg.Group("/ops")
	{
		ops.POST("/change-password", h.OpsChangePassword)
		ops.GET("/check-first-login", h.CheckFirstLogin)
	}
}

// OpsLogin handles operations admin login (PRD 1.1.4).
func (h *SDPivotOpsHandler) OpsLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find ops admin user
	var user types.User
	if err := h.db.Where("email = ? AND is_ops_admin = true AND is_active = true", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Check password expiry (90 days)
	if user.PasswordExpiresAt != nil && time.Now().After(*user.PasswordExpiresAt) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":                "password expired",
			"must_change_password": true,
		})
		return
	}

	// Generate tokens with ops-admin role
	now := time.Now()
	accessToken, _, err := h.jwtManager.GenerateAccessToken(user.ID, user.TenantID, string(types.AccessRoleSuperAdmin))
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
	h.db.Create(&rt)

	// Update last login
	h.db.Model(&user).Update("updated_at", now)

	c.JSON(http.StatusOK, gin.H{
		"access_token":         accessToken,
		"refresh_token":        refreshToken,
		"expires_in":           h.jwtManager.AccessExpirySeconds(),
		"must_change_password": user.MustChangePassword,
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"role":  types.AccessRoleSuperAdmin,
		},
	})
}

// OpsRefreshToken refreshes operations admin tokens with rotation.
func (h *SDPivotOpsHandler) OpsRefreshToken(c *gin.Context) {
	var req types.SDPivotRefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tokenHash := auth.HashRefreshToken(req.RefreshToken)
	var rt types.RefreshToken
	if err := h.db.Where("token_hash = ? AND expires_at > ?", tokenHash, time.Now()).First(&rt).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired refresh token"})
		return
	}
	var user types.User
	if err := h.db.Where("id = ? AND is_ops_admin = true AND is_active = true", rt.UserID).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "ops admin not found"})
		return
	}
	now := time.Now()
	accessToken, _, err := h.jwtManager.GenerateAccessToken(user.ID, user.TenantID, string(types.AccessRoleSuperAdmin))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}
	newRefreshToken, newTokenHash, err := h.jwtManager.GenerateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}
	if err := h.db.Delete(&rt).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to rotate refresh token"})
		return
	}
	newRT := types.RefreshToken{UserID: user.ID, TokenHash: newTokenHash, Family: rt.Family, ExpiresAt: now.Add(h.jwtManager.RefreshExpiry()), CreatedAt: now}
	if err := h.db.Create(&newRT).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create new refresh token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"access_token": accessToken, "refresh_token": newRefreshToken, "expires_in": h.jwtManager.AccessExpirySeconds()})
}

// OpsChangePassword handles forced password change (PRD 1.1.4).
func (h *SDPivotOpsHandler) OpsChangePassword(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if !middleware.HasPermission(middleware.GetRole(c), middleware.PermissionUserRoleAssign) {
		c.JSON(http.StatusForbidden, gin.H{"error": "ops admin access required"})
		return
	}

	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate password complexity (PRD: ≥8位，大小写+数字+特殊字符)
	if !validatePasswordComplexity(req.NewPassword) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "password must be ≥8 characters with uppercase, lowercase, digit, and special character",
		})
		return
	}

	// Find user
	var user types.User
	if err := h.db.Where("id = ? AND is_ops_admin = true", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "old password is incorrect"})
		return
	}

	// Check new password is different
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.NewPassword)); err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "new password must be different from old password"})
		return
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	// Update password with expiry policy (90 days)
	now := time.Now()
	expiresAt := now.Add(90 * 24 * time.Hour)
	updates := map[string]interface{}{
		"password_hash":        string(hash),
		"must_change_password": false,
		"password_changed_at":  now,
		"password_expires_at":  expiresAt,
		"updated_at":           now,
	}
	h.db.Model(&user).Updates(updates)

	c.JSON(http.StatusOK, gin.H{
		"message":             "password changed successfully",
		"password_expires_at": expiresAt,
		"days_remaining":      90,
	})
}

// CheckFirstLogin checks if ops admin needs to change password.
func (h *SDPivotOpsHandler) CheckFirstLogin(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if !middleware.HasPermission(middleware.GetRole(c), middleware.PermissionUserRoleAssign) {
		c.JSON(http.StatusForbidden, gin.H{"error": "ops admin access required"})
		return
	}

	var user types.User
	if err := h.db.Select("must_change_password, password_expires_at").
		Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	mustChange := user.MustChangePassword
	expired := user.PasswordExpiresAt != nil && time.Now().After(*user.PasswordExpiresAt)

	c.JSON(http.StatusOK, gin.H{
		"must_change_password": mustChange || expired,
	})
}

// validatePasswordComplexity checks PRD-required password rules:
// ≥8 characters, at least one uppercase, one lowercase, one digit, one special char.
func validatePasswordComplexity(pw string) bool {
	if len(pw) < 8 {
		return false
	}
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range pw {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}
	return hasUpper && hasLower && hasDigit && hasSpecial
}
