package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/middleware"
)

type writingCategory struct {
	ID          string    `json:"id"`
	TenantID    uint64    `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Sort        int       `json:"sort"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (writingCategory) TableName() string { return "writing_category" }

type writingTemplate struct {
	ID         string    `json:"id"`
	TenantID   uint64    `json:"tenant_id"`
	CategoryID string    `json:"category_id"`
	Name       string    `json:"name"`
	Content    string    `json:"content"`
	IsBuiltin  bool      `json:"is_builtin"`
	Sort       int       `json:"sort"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (writingTemplate) TableName() string { return "writing_template" }

type SDPivotWritingManagementHandler struct{ db *gorm.DB }

func NewSDPivotWritingManagementHandler(db *gorm.DB) *SDPivotWritingManagementHandler {
	return &SDPivotWritingManagementHandler{db: db}
}
func (h *SDPivotWritingManagementHandler) RegisterRoutes(rg *gin.RouterGroup) {
	w := rg.Group("/writing")
	w.GET("/categories", h.ListCategories)
	w.GET("/templates", h.ListTemplates)
	w.POST("/categories", h.CreateCategory)
	w.PUT("/categories/:id", h.UpdateCategory)
	w.DELETE("/categories/:id", h.DeleteCategory)
	w.POST("/templates", h.CreateTemplate)
	w.PUT("/templates/:id", h.UpdateTemplate)
	w.DELETE("/templates/:id", h.DeleteTemplate)
}
func (h *SDPivotWritingManagementHandler) ListCategories(c *gin.Context) {
	var rows []writingCategory
	if err := middleware.TenantDB(c, h.db).Where("tenant_id = ?", writingTenantID(c)).Order("sort, created_at").Find(&rows).Error; err != nil {
		c.JSON(500, gin.H{"error": "failed to list writing categories"})
		return
	}
	c.JSON(200, gin.H{"categories": rows})
}
func (h *SDPivotWritingManagementHandler) CreateCategory(c *gin.Context) {
	var req struct {
		Name, Description string
		Sort              int
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		c.JSON(400, gin.H{"error": "name is required"})
		return
	}
	now := time.Now()
	row := writingCategory{ID: uuid.NewString(), TenantID: writingTenantID(c), Name: strings.TrimSpace(req.Name), Description: strings.TrimSpace(req.Description), Sort: req.Sort, CreatedAt: now, UpdatedAt: now}
	if err := middleware.TenantDB(c, h.db).Create(&row).Error; err != nil {
		c.JSON(409, gin.H{"error": "writing category already exists"})
		return
	}
	c.JSON(201, gin.H{"category": row})
}
func (h *SDPivotWritingManagementHandler) UpdateCategory(c *gin.Context) {
	var req struct {
		Name, Description *string
		Sort              *int
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request body"})
		return
	}
	u := map[string]interface{}{"updated_at": time.Now()}
	if req.Name != nil {
		u["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		u["description"] = strings.TrimSpace(*req.Description)
	}
	if req.Sort != nil {
		u["sort"] = *req.Sort
	}
	h.update(c, &writingCategory{}, u, "category")
}
func (h *SDPivotWritingManagementHandler) DeleteCategory(c *gin.Context) {
	db := middleware.TenantDB(c, h.db)
	var count int64
	db.Model(&writingTemplate{}).Where("tenant_id=? AND category_id=?", writingTenantID(c), c.Param("id")).Count(&count)
	if count > 0 {
		c.JSON(409, gin.H{"error": "delete templates in this category first"})
		return
	}
	h.delete(c, &writingCategory{}, "category")
}
func (h *SDPivotWritingManagementHandler) ListTemplates(c *gin.Context) {
	q := middleware.TenantDB(c, h.db).Where("tenant_id=?", writingTenantID(c))
	if id := strings.TrimSpace(c.Query("category_id")); id != "" {
		q = q.Where("category_id=?", id)
	}
	var rows []writingTemplate
	if err := q.Order("sort,created_at").Find(&rows).Error; err != nil {
		c.JSON(500, gin.H{"error": "failed to list writing templates"})
		return
	}
	c.JSON(200, gin.H{"templates": rows})
}
func (h *SDPivotWritingManagementHandler) CreateTemplate(c *gin.Context) {
	var req struct {
		CategoryID, Name, Content string
		IsBuiltin                 bool
		Sort                      int
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.CategoryID) == "" || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Content) == "" {
		c.JSON(400, gin.H{"error": "category_id, name, and content are required"})
		return
	}
	db := middleware.TenantDB(c, h.db)
	var n int64
	db.Model(&writingCategory{}).Where("id=? AND tenant_id=?", req.CategoryID, writingTenantID(c)).Count(&n)
	if n == 0 {
		c.JSON(400, gin.H{"error": "writing category not found"})
		return
	}
	now := time.Now()
	row := writingTemplate{ID: uuid.NewString(), TenantID: writingTenantID(c), CategoryID: req.CategoryID, Name: strings.TrimSpace(req.Name), Content: strings.TrimSpace(req.Content), IsBuiltin: req.IsBuiltin, Sort: req.Sort, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&row).Error; err != nil {
		c.JSON(409, gin.H{"error": "writing template already exists"})
		return
	}
	c.JSON(201, gin.H{"template": row})
}
func (h *SDPivotWritingManagementHandler) UpdateTemplate(c *gin.Context) {
	var req struct {
		CategoryID, Name, Content *string
		IsBuiltin                 *bool
		Sort                      *int
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request body"})
		return
	}
	u := map[string]interface{}{"updated_at": time.Now()}
	if req.CategoryID != nil {
		u["category_id"] = strings.TrimSpace(*req.CategoryID)
	}
	if req.Name != nil {
		u["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Content != nil {
		u["content"] = strings.TrimSpace(*req.Content)
	}
	if req.IsBuiltin != nil {
		u["is_builtin"] = *req.IsBuiltin
	}
	if req.Sort != nil {
		u["sort"] = *req.Sort
	}
	h.update(c, &writingTemplate{}, u, "template")
}
func (h *SDPivotWritingManagementHandler) DeleteTemplate(c *gin.Context) {
	h.delete(c, &writingTemplate{}, "template")
}
func (h *SDPivotWritingManagementHandler) update(c *gin.Context, model interface{}, values map[string]interface{}, name string) {
	r := middleware.TenantDB(c, h.db).Model(model).Where("id=? AND tenant_id=?", c.Param("id"), writingTenantID(c)).Updates(values)
	if r.Error != nil {
		c.JSON(409, gin.H{"error": "failed to update " + name})
		return
	}
	if r.RowsAffected == 0 {
		c.JSON(404, gin.H{"error": name + " not found"})
		return
	}
	c.JSON(200, gin.H{"message": name + " updated"})
}
func (h *SDPivotWritingManagementHandler) delete(c *gin.Context, model interface{}, name string) {
	r := middleware.TenantDB(c, h.db).Where("id=? AND tenant_id=?", c.Param("id"), writingTenantID(c)).Delete(model)
	if r.Error != nil {
		c.JSON(500, gin.H{"error": "failed to delete " + name})
		return
	}
	if r.RowsAffected == 0 {
		c.JSON(404, gin.H{"error": name + " not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
