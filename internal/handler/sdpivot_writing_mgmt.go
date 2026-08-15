package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
)

type SDPivotWritingManagementHandler struct {
	db *gorm.DB
}

type writingCategory struct {
	ID          string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID    uint64    `json:"tenant_id" gorm:"not null;index"`
	Name        string    `json:"name" gorm:"type:varchar(100);not null"`
	Description string    `json:"description" gorm:"type:text"`
	Sort        int       `json:"sort" gorm:"not null;default:0"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (writingCategory) TableName() string { return "writing_category" }

type writingTemplate struct {
	ID         string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID   uint64    `json:"tenant_id" gorm:"not null;index"`
	CategoryID string    `json:"category_id" gorm:"type:varchar(36);not null;index"`
	Name       string    `json:"name" gorm:"type:varchar(100);not null"`
	Content    string    `json:"content" gorm:"type:text;not null"`
	IsBuiltin  bool      `json:"is_builtin" gorm:"not null;default:false"`
	Sort       int       `json:"sort" gorm:"not null;default:0"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (writingTemplate) TableName() string { return "writing_template" }

func NewSDPivotWritingManagementHandler(db *gorm.DB) *SDPivotWritingManagementHandler {
	return &SDPivotWritingManagementHandler{db: db}
}

func (h *SDPivotWritingManagementHandler) RegisterRoutes(rg *gin.RouterGroup) {
	read := rg.Group("/writing", middleware.RequirePermission(middleware.PermissionKnowledgeRead))
	read.GET("/categories", h.ListCategories)
	read.GET("/templates", h.ListTemplates)

	write := rg.Group("/writing", middleware.RequirePermission(middleware.PermissionKnowledgeWrite))
	write.POST("/categories", h.CreateCategory)
	write.PUT("/categories/:id", h.UpdateCategory)
	write.DELETE("/categories/:id", h.DeleteCategory)
	write.POST("/templates", h.CreateTemplate)
	write.PUT("/templates/:id", h.UpdateTemplate)
	write.DELETE("/templates/:id", h.DeleteTemplate)

	h.registerTieredTemplateRoutes(rg)
}

func (h *SDPivotWritingManagementHandler) ListCategories(c *gin.Context) {
	var items []writingCategory
	if err := middleware.TenantDB(c, h.db).Where("tenant_id = ?", middleware.GetTenantID(c)).Order("sort ASC, created_at ASC").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list writing categories"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"categories": items})
}

func (h *SDPivotWritingManagementHandler) CreateCategory(c *gin.Context) {
	if middleware.RequireRole(c, string(types.AccessRoleSuperAdmin), string(types.AccessRoleDepartmentAdmin)) {
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Sort        int    `json:"sort"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	now := time.Now()
	item := writingCategory{ID: uuid.New().String(), TenantID: middleware.GetTenantID(c), Name: strings.TrimSpace(req.Name), Description: strings.TrimSpace(req.Description), Sort: req.Sort, CreatedAt: now, UpdatedAt: now}
	if err := middleware.TenantDB(c, h.db).Create(&item).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "writing category already exists"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"category": item})
}

func (h *SDPivotWritingManagementHandler) UpdateCategory(c *gin.Context) {
	if middleware.RequireRole(c, string(types.AccessRoleSuperAdmin), string(types.AccessRoleDepartmentAdmin)) {
		return
	}
	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Sort        *int    `json:"sort"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	updates := map[string]interface{}{"updated_at": time.Now()}
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name cannot be empty"})
			return
		}
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		updates["description"] = strings.TrimSpace(*req.Description)
	}
	if req.Sort != nil {
		updates["sort"] = *req.Sort
	}
	h.updateWritingRecord(c, &writingCategory{}, updates, "category")
}

func (h *SDPivotWritingManagementHandler) DeleteCategory(c *gin.Context) {
	if middleware.RequireRole(c, string(types.AccessRoleSuperAdmin), string(types.AccessRoleDepartmentAdmin)) {
		return
	}
	db := middleware.TenantDB(c, h.db)
	tenantID := middleware.GetTenantID(c)
	var count int64
	if err := db.Model(&writingTemplate{}).Where("tenant_id = ? AND category_id = ?", tenantID, c.Param("id")).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate category"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "delete templates in this category first"})
		return
	}
	result := db.Where("id = ? AND tenant_id = ?", c.Param("id"), tenantID).Delete(&writingCategory{})
	h.writeDeleteResult(c, result, "category")
}

func (h *SDPivotWritingManagementHandler) ListTemplates(c *gin.Context) {
	db := middleware.TenantDB(c, h.db)
	query := db.Where("tenant_id = ?", middleware.GetTenantID(c))
	if categoryID := strings.TrimSpace(c.Query("category_id")); categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}
	var items []writingTemplate
	if err := query.Order("sort ASC, created_at ASC").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list writing templates"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"templates": items})
}

func (h *SDPivotWritingManagementHandler) CreateTemplate(c *gin.Context) {
	var req struct {
		CategoryID string `json:"category_id"`
		Name       string `json:"name"`
		Content    string `json:"content"`
		IsBuiltin  bool   `json:"is_builtin"`
		Sort       int    `json:"sort"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.CategoryID) == "" || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Content) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category_id, name, and content are required"})
		return
	}
	if req.IsBuiltin && !isWritingAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "only administrators can create built-in templates"})
		return
	}
	db := middleware.TenantDB(c, h.db)
	tenantID := middleware.GetTenantID(c)
	if !writingCategoryExists(db, tenantID, req.CategoryID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "writing category not found"})
		return
	}
	now := time.Now()
	item := writingTemplate{ID: uuid.New().String(), TenantID: tenantID, CategoryID: req.CategoryID, Name: strings.TrimSpace(req.Name), Content: strings.TrimSpace(req.Content), IsBuiltin: req.IsBuiltin, Sort: req.Sort, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&item).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "writing template already exists"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"template": item})
}

func (h *SDPivotWritingManagementHandler) UpdateTemplate(c *gin.Context) {
	db := middleware.TenantDB(c, h.db)
	tenantID := middleware.GetTenantID(c)
	var item writingTemplate
	if err := db.Where("id = ? AND tenant_id = ?", c.Param("id"), tenantID).First(&item).Error; err != nil {
		h.writeRecordError(c, err, "template")
		return
	}
	if item.IsBuiltin && !isWritingAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "only administrators can update built-in templates"})
		return
	}
	var req struct {
		CategoryID *string `json:"category_id"`
		Name       *string `json:"name"`
		Content    *string `json:"content"`
		IsBuiltin  *bool   `json:"is_builtin"`
		Sort       *int    `json:"sort"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	updates := map[string]interface{}{"updated_at": time.Now()}
	if req.CategoryID != nil {
		value := strings.TrimSpace(*req.CategoryID)
		if !writingCategoryExists(db, tenantID, value) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "writing category not found"})
			return
		}
		updates["category_id"] = value
	}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Content != nil {
		updates["content"] = strings.TrimSpace(*req.Content)
	}
	if req.IsBuiltin != nil {
		if !isWritingAdmin(c) {
			c.JSON(http.StatusForbidden, gin.H{"error": "only administrators can change built-in status"})
			return
		}
		updates["is_builtin"] = *req.IsBuiltin
	}
	if req.Sort != nil {
		updates["sort"] = *req.Sort
	}
	if err := db.Model(&writingTemplate{}).Where("id = ? AND tenant_id = ?", item.ID, tenantID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "failed to update template"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "template updated"})
}

func (h *SDPivotWritingManagementHandler) DeleteTemplate(c *gin.Context) {
	db := middleware.TenantDB(c, h.db)
	tenantID := middleware.GetTenantID(c)
	var item writingTemplate
	if err := db.Where("id = ? AND tenant_id = ?", c.Param("id"), tenantID).First(&item).Error; err != nil {
		h.writeRecordError(c, err, "template")
		return
	}
	if item.IsBuiltin && !isWritingAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "only administrators can delete built-in templates"})
		return
	}
	result := db.Where("id = ? AND tenant_id = ?", item.ID, tenantID).Delete(&writingTemplate{})
	h.writeDeleteResult(c, result, "template")
}

func (h *SDPivotWritingManagementHandler) updateWritingRecord(c *gin.Context, model interface{}, updates map[string]interface{}, name string) {
	result := middleware.TenantDB(c, h.db).Model(model).Where("id = ? AND tenant_id = ?", c.Param("id"), middleware.GetTenantID(c)).Updates(updates)
	if result.Error != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "failed to update " + name})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": name + " not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": name + " updated"})
}

func (h *SDPivotWritingManagementHandler) writeDeleteResult(c *gin.Context, result *gorm.DB, name string) {
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete " + name})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": name + " not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *SDPivotWritingManagementHandler) writeRecordError(c *gin.Context, err error, name string) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": name + " not found"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load " + name})
}

func writingCategoryExists(db *gorm.DB, tenantID uint64, id string) bool {
	var count int64
	return id != "" && db.Model(&writingCategory{}).Where("id = ? AND tenant_id = ?", id, tenantID).Count(&count).Error == nil && count == 1
}

func isWritingAdmin(c *gin.Context) bool {
	role := types.NormalizeAccessRole(middleware.GetRole(c))
	return role == types.AccessRoleSuperAdmin || role == types.AccessRoleDepartmentAdmin
}
