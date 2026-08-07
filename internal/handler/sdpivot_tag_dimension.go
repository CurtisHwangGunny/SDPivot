package handler

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
)

var tagDimensionCodePattern = regexp.MustCompile(`[^a-z0-9_]+`)

type SDPivotTagDimensionHandler struct {
	db *gorm.DB
}

func NewSDPivotTagDimensionHandler(db *gorm.DB) *SDPivotTagDimensionHandler {
	return &SDPivotTagDimensionHandler{db: db}
}

func (h *SDPivotTagDimensionHandler) RegisterRoutes(rg *gin.RouterGroup) {
	adminRead := rg.Group("/admin", middleware.RequirePermission(middleware.PermissionKnowledgeRead))
	adminRead.GET("/tag-dimensions", h.List)

	adminWrite := rg.Group("/admin", middleware.RequirePermission(middleware.PermissionUserRoleAssign))
	adminWrite.POST("/tag-dimensions", h.Create)
	adminWrite.PUT("/tag-dimensions/:id", h.Update)
	adminWrite.DELETE("/tag-dimensions/:id", h.Delete)
}

func (h *SDPivotTagDimensionHandler) List(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	var dimensions []types.SDPivotTagDimension
	err := middleware.TenantDB(c, h.db).
		Where("deleted_at IS NULL AND (tenant_id IS NULL OR tenant_id = ?)", tenantID).
		Order("sort_order ASC, code ASC").Find(&dimensions).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tag dimensions"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"dimensions": dimensions})
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
	req.Description = strings.TrimSpace(req.Description)
	req.Code = strings.ToLower(strings.TrimSpace(req.Code))
	if req.Name == "" || len(req.Name) > 128 {
		return errors.New("dimension name is required and cannot exceed 128 characters")
	}
	if req.Code == "" {
		req.Code = tagDimensionCodePattern.ReplaceAllString(strings.ToLower(req.Name), "_")
	}
	req.Code = strings.Trim(tagDimensionCodePattern.ReplaceAllString(req.Code, "_"), "_")
	if req.Code == "" || len(req.Code) > 64 {
		return errors.New("dimension code is invalid")
	}
	return nil
}

func (h *SDPivotTagDimensionHandler) Create(c *gin.Context) {
	var req tagDimensionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := normalizeTagDimensionRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	tenantID := middleware.GetTenantID(c)
	now := time.Now()
	dimension := types.SDPivotTagDimension{ID: uuid.NewString(), TenantID: &tenantID, Code: req.Code, Name: req.Name, Description: req.Description, Enabled: enabled, SortOrder: req.SortOrder, CreatedAt: now, UpdatedAt: now}
	if err := middleware.TenantDB(c, h.db).Create(&dimension).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "dimension code already exists"})
		return
	}
	c.JSON(http.StatusCreated, dimension)
}

func (h *SDPivotTagDimensionHandler) Update(c *gin.Context) {
	var req tagDimensionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := normalizeTagDimensionRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	db := middleware.TenantDB(c, h.db)
	tenantID := middleware.GetTenantID(c)
	var dimension types.SDPivotTagDimension
	if err := db.Where("id = ? AND deleted_at IS NULL AND (tenant_id IS NULL OR tenant_id = ?)", c.Param("id"), tenantID).First(&dimension).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tag dimension not found"})
		return
	}
	updates := map[string]any{"code": req.Code, "name": req.Name, "description": req.Description, "sort_order": req.SortOrder, "updated_at": time.Now()}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if err := db.Model(&dimension).Updates(updates).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "failed to update tag dimension"})
		return
	}
	db.First(&dimension, "id = ?", dimension.ID)
	c.JSON(http.StatusOK, dimension)
}

func (h *SDPivotTagDimensionHandler) Delete(c *gin.Context) {
	db := middleware.TenantDB(c, h.db)
	tenantID := middleware.GetTenantID(c)
	result := db.Where("id = ? AND (tenant_id IS NULL OR tenant_id = ?)", c.Param("id"), tenantID).Delete(&types.SDPivotTagDimension{})
	if result.Error != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "failed to delete tag dimension"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "tag dimension not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
