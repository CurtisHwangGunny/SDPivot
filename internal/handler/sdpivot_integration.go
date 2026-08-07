package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/middleware"
	secutils "github.com/Tencent/WeKnora/internal/utils"
)

type thirdPartyConnector struct {
	ID         string          `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID   uint64          `json:"tenant_id"`
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	BaseURL    string          `json:"base_url"`
	AuthType   string          `json:"auth_type"`
	AuthToken  string          `json:"-"`
	Enabled    bool            `json:"enabled"`
	SyncRule   json.RawMessage `json:"sync_rule" gorm:"type:json"`
	LastSyncAt *time.Time      `json:"last_sync_at"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

func (thirdPartyConnector) TableName() string { return "third_party_connector" }

type thirdPartyConnectorResponse struct {
	thirdPartyConnector
	AuthConfigured bool `json:"auth_configured"`
}

type thirdPartyConnectorRequest struct {
	Name      string          `json:"name" binding:"required"`
	Type      string          `json:"type" binding:"required,oneof=users departments documents"`
	BaseURL   string          `json:"base_url" binding:"required"`
	AuthType  string          `json:"auth_type" binding:"required,oneof=none bearer basic"`
	AuthToken *string         `json:"auth_token"`
	Enabled   *bool           `json:"enabled"`
	SyncRule  json.RawMessage `json:"sync_rule"`
}

type SDPivotIntegrationHandler struct{ db *gorm.DB }

func NewSDPivotIntegrationHandler(db *gorm.DB) *SDPivotIntegrationHandler {
	return &SDPivotIntegrationHandler{db: db}
}

func (h *SDPivotIntegrationHandler) RegisterRoutes(rg *gin.RouterGroup) {
	integrations := rg.Group("/sdpivot/integrations", middleware.RequirePermission(middleware.PermissionDepartmentManage))
	integrations.GET("", h.List)
	integrations.POST("", h.Create)
	integrations.PUT("/:id", h.Update)
	integrations.DELETE("/:id", h.Delete)
	integrations.POST("/:id/sync/:resource", h.Sync)
}

func (h *SDPivotIntegrationHandler) List(c *gin.Context) {
	rows := make([]thirdPartyConnector, 0)
	if err := middleware.TenantDB(c, h.db).Where("tenant_id = ?", middleware.GetTenantID(c)).Order("created_at DESC").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load integrations"})
		return
	}
	items := make([]thirdPartyConnectorResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, thirdPartyConnectorResponse{thirdPartyConnector: row, AuthConfigured: row.AuthToken != ""})
	}
	c.JSON(http.StatusOK, gin.H{"integrations": items})
}

func (h *SDPivotIntegrationHandler) Create(c *gin.Context) {
	req, ok := bindConnectorRequest(c)
	if !ok {
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	now := time.Now()
	row := thirdPartyConnector{
		ID: uuid.NewString(), TenantID: middleware.GetTenantID(c), Name: strings.TrimSpace(req.Name),
		Type: req.Type, BaseURL: strings.TrimRight(strings.TrimSpace(req.BaseURL), "/"), AuthType: req.AuthType,
		Enabled: enabled, SyncRule: normalizedSyncRule(req.SyncRule), CreatedAt: now, UpdatedAt: now,
	}
	if req.AuthToken != nil {
		row.AuthToken = strings.TrimSpace(*req.AuthToken)
	}
	if err := middleware.TenantDB(c, h.db).Create(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create integration"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"integration": thirdPartyConnectorResponse{thirdPartyConnector: row, AuthConfigured: row.AuthToken != ""}})
}

func (h *SDPivotIntegrationHandler) Update(c *gin.Context) {
	req, ok := bindConnectorRequest(c)
	if !ok {
		return
	}
	db := middleware.TenantDB(c, h.db)
	var row thirdPartyConnector
	if err := db.Where("id = ? AND tenant_id = ?", c.Param("id"), middleware.GetTenantID(c)).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "integration not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load integration"})
		}
		return
	}
	updates := map[string]interface{}{
		"name": strings.TrimSpace(req.Name), "type": req.Type,
		"base_url": strings.TrimRight(strings.TrimSpace(req.BaseURL), "/"), "auth_type": req.AuthType,
		"sync_rule": normalizedSyncRule(req.SyncRule), "updated_at": time.Now(),
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.AuthToken != nil && strings.TrimSpace(*req.AuthToken) != "" {
		updates["auth_token"] = strings.TrimSpace(*req.AuthToken)
	}
	if req.AuthType == "none" {
		updates["auth_token"] = ""
	}
	if err := db.Model(&row).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update integration"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "integration updated"})
}

func (h *SDPivotIntegrationHandler) Delete(c *gin.Context) {
	result := middleware.TenantDB(c, h.db).Where("id = ? AND tenant_id = ?", c.Param("id"), middleware.GetTenantID(c)).Delete(&thirdPartyConnector{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete integration"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "integration not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "integration deleted"})
}

func (h *SDPivotIntegrationHandler) Sync(c *gin.Context) {
	resource := c.Param("resource")
	if resource != "users" && resource != "departments" && resource != "documents" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported sync resource"})
		return
	}
	var count int64
	if err := middleware.TenantDB(c, h.db).Model(&thirdPartyConnector{}).
		Where("id = ? AND tenant_id = ?", c.Param("id"), middleware.GetTenantID(c)).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load integration"})
		return
	}
	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "integration not found"})
		return
	}
	// TODO: enqueue the connector-specific synchronization worker.
	c.JSON(http.StatusOK, gin.H{"message": "同步任务已排队（暂未实现）", "status": "queued", "resource": resource})
}

func bindConnectorRequest(c *gin.Context) (thirdPartyConnectorRequest, bool) {
	var req thirdPartyConnectorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return req, false
	}
	req.BaseURL = strings.TrimSpace(req.BaseURL)
	if err := secutils.ValidateURLForSSRF(req.BaseURL); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or unsafe connector base_url"})
		return req, false
	}
	if len(req.SyncRule) > 0 && string(req.SyncRule) != "null" && !json.Valid(req.SyncRule) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sync_rule must be valid JSON"})
		return req, false
	}
	return req, true
}

func normalizedSyncRule(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 || string(raw) == "null" {
		return json.RawMessage([]byte("{}"))
	}
	return raw
}
