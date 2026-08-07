package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/middleware"
)

type sdpivotAdminAPIToken struct {
	ID         string          `json:"id" gorm:"type:varchar(36);primaryKey"`
	UserID     string          `json:"-" gorm:"column:user_id"`
	TenantID   uint64          `json:"-"`
	Name       string          `json:"name"`
	TokenHash  string          `json:"-"`
	Prefix     string          `json:"prefix"`
	Scopes     json.RawMessage `json:"scopes" gorm:"type:jsonb"`
	ExpiresAt  *time.Time      `json:"expires_at,omitempty"`
	LastUsedAt *time.Time      `json:"last_used_at,omitempty"`
	RevokedAt  *time.Time      `json:"revoked_at,omitempty"`
	CreatedBy  string          `json:"created_by"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

func (sdpivotAdminAPIToken) TableName() string { return "api_tokens" }

type SDPivotAPITokenHandler struct{ db *gorm.DB }

func NewSDPivotAPITokenHandler(db *gorm.DB) *SDPivotAPITokenHandler {
	return &SDPivotAPITokenHandler{db: db}
}

func (h *SDPivotAPITokenHandler) RegisterRoutes(rg *gin.RouterGroup) {
	admin := rg.Group("/admin/api-tokens", middleware.RequirePermission(middleware.PermissionDepartmentManage))
	admin.POST("", h.Create)
	admin.GET("", h.List)
	admin.PUT("/:id", h.Update)
	admin.DELETE("/:id", h.Revoke)
	admin.POST("/:id/regenerate", h.Regenerate)
}

type sdpivotAPITokenRequest struct {
	Name      string     `json:"name" binding:"required,max=100"`
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at"`
}

func generateSDPivotAPIToken() (string, string, string, error) {
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", "", "", err
	}
	raw := "sdp_" + base64.RawURLEncoding.EncodeToString(random)
	hash := sha256.Sum256([]byte(raw))
	return raw, raw[:6], hex.EncodeToString(hash[:]), nil
}

func (h *SDPivotAPITokenHandler) Create(c *gin.Context) {
	var req sdpivotAPITokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ExpiresAt != nil && !req.ExpiresAt.After(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "expires_at must be in the future"})
		return
	}
	raw, prefix, hash, err := generateSDPivotAPIToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate api token"})
		return
	}
	scopes, err := json.Marshal(req.Scopes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scopes"})
		return
	}
	now := time.Now()
	record := sdpivotAdminAPIToken{
		ID: uuid.NewString(), UserID: middleware.GetUserID(c), TenantID: middleware.GetTenantID(c),
		Name: strings.TrimSpace(req.Name), TokenHash: hash, Prefix: prefix, Scopes: scopes,
		ExpiresAt: req.ExpiresAt, CreatedBy: middleware.GetUserID(c), CreatedAt: now, UpdatedAt: now,
	}
	if err := middleware.TenantDB(c, h.db).Create(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create api token"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"token": raw, "data": record})
}

func (h *SDPivotAPITokenHandler) List(c *gin.Context) {
	rows := make([]sdpivotAdminAPIToken, 0)
	if err := middleware.TenantDB(c, h.db).Where("tenant_id = ?", middleware.GetTenantID(c)).Order("created_at DESC").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list api tokens"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tokens": rows})
}

func (h *SDPivotAPITokenHandler) Update(c *gin.Context) {
	var req struct {
		Name   string   `json:"name" binding:"required,max=100"`
		Scopes []string `json:"scopes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	scopes, _ := json.Marshal(req.Scopes)
	result := middleware.TenantDB(c, h.db).Model(&sdpivotAdminAPIToken{}).
		Where("id = ? AND tenant_id = ? AND revoked_at IS NULL", c.Param("id"), middleware.GetTenantID(c)).
		Updates(map[string]interface{}{"name": strings.TrimSpace(req.Name), "scopes": scopes, "updated_at": time.Now()})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update api token"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "api token not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "api token updated"})
}

func (h *SDPivotAPITokenHandler) Revoke(c *gin.Context) {
	now := time.Now()
	result := middleware.TenantDB(c, h.db).Model(&sdpivotAdminAPIToken{}).
		Where("id = ? AND tenant_id = ? AND revoked_at IS NULL", c.Param("id"), middleware.GetTenantID(c)).
		Updates(map[string]interface{}{"revoked_at": now, "updated_at": now})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke api token"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "api token not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "api token revoked"})
}

func (h *SDPivotAPITokenHandler) Regenerate(c *gin.Context) {
	db := middleware.TenantDB(c, h.db)
	var record sdpivotAdminAPIToken
	if err := db.Where("id = ? AND tenant_id = ?", c.Param("id"), middleware.GetTenantID(c)).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "api token not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load api token"})
		}
		return
	}
	raw, prefix, hash, err := generateSDPivotAPIToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate api token"})
		return
	}
	now := time.Now()
	if err := db.Model(&record).Updates(map[string]interface{}{
		"token_hash": hash, "prefix": prefix, "revoked_at": nil, "last_used_at": nil, "updated_at": now,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to regenerate api token"})
		return
	}
	record.TokenHash, record.Prefix, record.RevokedAt, record.LastUsedAt, record.UpdatedAt = hash, prefix, nil, nil, now
	c.JSON(http.StatusOK, gin.H{"token": raw, "data": record})
}
