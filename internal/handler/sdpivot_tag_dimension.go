package handler

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var tagDimensionCodePattern = regexp.MustCompile(`[^a-z0-9_]+`)

type SDPivotTagDimensionHandler struct{ db *gorm.DB }

func NewSDPivotTagDimensionHandler(db *gorm.DB) *SDPivotTagDimensionHandler {
	return &SDPivotTagDimensionHandler{db: db}
}
func (h *SDPivotTagDimensionHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/admin/tag-dimensions", h.List)
	rg.POST("/admin/tag-dimensions", h.Create)
	rg.PUT("/admin/tag-dimensions/:id", h.Update)
	rg.DELETE("/admin/tag-dimensions/:id", h.Delete)
}
func (h *SDPivotTagDimensionHandler) List(c *gin.Context) {
	var rows []types.SDPivotTagDimension
	err := h.db.WithContext(c.Request.Context()).Where("deleted_at IS NULL AND (tenant_id IS NULL OR tenant_id = ?)", tenantID(c)).Order("sort_order ASC, code ASC").Find(&rows).Error
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to list tag dimensions"})
		return
	}
	c.JSON(200, gin.H{"dimensions": rows})
}

type tagDimensionRequest struct {
	Code        string `json:"code"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Enabled     *bool  `json:"enabled"`
	SortOrder   int    `json:"sort_order"`
}

func normalizeTagDimensionRequest(req *tagDimensionRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.ToLower(strings.TrimSpace(req.Code))
	req.Description = strings.TrimSpace(req.Description)
	if req.Name == "" || len(req.Name) > 128 {
		return errors.New("invalid dimension name")
	}
	if req.Code == "" {
		req.Code = tagDimensionCodePattern.ReplaceAllString(strings.ToLower(req.Name), "_")
	}
	req.Code = strings.Trim(tagDimensionCodePattern.ReplaceAllString(req.Code, "_"), "_")
	if req.Code == "" || len(req.Code) > 64 {
		return errors.New("invalid dimension code")
	}
	return nil
}
func (h *SDPivotTagDimensionHandler) Create(c *gin.Context) {
	var req tagDimensionRequest
	if c.ShouldBindJSON(&req) != nil || normalizeTagDimensionRequest(&req) != nil {
		c.JSON(400, gin.H{"error": "invalid dimension"})
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	now := time.Now()
	row := types.SDPivotTagDimension{ID: uuid.NewString(), TenantID: ptrTenant(tenantID(c)), Code: req.Code, Name: req.Name, Description: req.Description, Enabled: enabled, SortOrder: req.SortOrder, CreatedAt: now, UpdatedAt: now}
	if err := h.db.WithContext(c.Request.Context()).Create(&row).Error; err != nil {
		c.JSON(409, gin.H{"error": "dimension code already exists"})
		return
	}
	c.JSON(201, row)
}
func (h *SDPivotTagDimensionHandler) Update(c *gin.Context) {
	var req tagDimensionRequest
	if c.ShouldBindJSON(&req) != nil || normalizeTagDimensionRequest(&req) != nil {
		c.JSON(400, gin.H{"error": "invalid dimension"})
		return
	}
	var row types.SDPivotTagDimension
	db := h.db.WithContext(c.Request.Context())
	if err := db.Where("id = ? AND deleted_at IS NULL AND (tenant_id IS NULL OR tenant_id = ?)", c.Param("id"), tenantID(c)).First(&row).Error; err != nil {
		c.JSON(404, gin.H{"error": "tag dimension not found"})
		return
	}
	updates := map[string]any{"code": req.Code, "name": req.Name, "description": req.Description, "sort_order": req.SortOrder, "updated_at": time.Now()}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if err := db.Model(&row).Updates(updates).Error; err != nil {
		c.JSON(409, gin.H{"error": "failed to update tag dimension"})
		return
	}
	db.First(&row, "id = ?", row.ID)
	c.JSON(200, row)
}
func (h *SDPivotTagDimensionHandler) Delete(c *gin.Context) {
	result := h.db.WithContext(c.Request.Context()).Where("id = ? AND (tenant_id IS NULL OR tenant_id = ?)", c.Param("id"), tenantID(c)).Delete(&types.SDPivotTagDimension{})
	if result.Error != nil {
		c.JSON(409, gin.H{"error": "failed to delete tag dimension"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(404, gin.H{"error": "tag dimension not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
