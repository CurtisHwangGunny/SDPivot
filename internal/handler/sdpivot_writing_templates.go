package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Tencent/WeKnora/internal/middleware"
)

type templateConfig struct {
	ID         string    `json:"id"`
	TenantID   uint64    `json:"tenant_id"`
	CategoryID string    `json:"category_id"`
	Name       string    `json:"name"`
	Content    string    `json:"content"`
	IsDefault  bool      `json:"is_default"`
	IsBuiltin  bool      `json:"is_builtin"`
	Sort       int       `json:"sort"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (templateConfig) TableName() string { return "template_configs" }

type userTemplatePref struct {
	ID                string          `json:"id"`
	UserID            string          `json:"user_id"`
	TenantID          uint64          `json:"tenant_id"`
	DefaultTemplateID *string         `json:"default_template_id"`
	CategoryOrder     json.RawMessage `json:"category_order"`
	Templates         json.RawMessage `json:"templates"`
	UpdatedAt         time.Time       `json:"updated_at"`
	CreatedAt         time.Time       `json:"created_at"`
}

func (userTemplatePref) TableName() string { return "user_template_prefs" }

func (h *SDPivotWritingManagementHandler) RegisterTieredTemplateRoutes(rg *gin.RouterGroup) {
	a := rg.Group("/admin/writing/templates")
	a.GET("", h.ListAdminTemplates)
	a.POST("", h.CreateAdminTemplate)
	a.PUT("/:id", h.UpdateAdminTemplate)
	a.DELETE("/:id", h.DeleteAdminTemplate)
	m := rg.Group("/my/writing/templates")
	m.GET("", h.GetMyWritingTemplates)
	m.POST("", h.CreateMyWritingTemplate)
	m.PUT("/:id", h.UpdateMyWritingTemplate)
	m.DELETE("/:id", h.DeleteMyWritingTemplate)
}
func (h *SDPivotWritingManagementHandler) ListAdminTemplates(c *gin.Context) {
	var rows []templateConfig
	if err := middleware.TenantDB(c, h.db).Where("tenant_id=?", writingTenantID(c)).Order("category_id,sort,created_at").Find(&rows).Error; err != nil {
		c.JSON(500, gin.H{"error": "failed to list admin writing templates"})
		return
	}
	c.JSON(200, gin.H{"templates": rows})
}
func (h *SDPivotWritingManagementHandler) CreateAdminTemplate(c *gin.Context) {
	var req struct {
		CategoryID, Name, Content string
		IsDefault, IsBuiltin      bool
		Sort                      int
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.CategoryID == "" || req.Name == "" || req.Content == "" {
		c.JSON(400, gin.H{"error": "category_id, name, and content are required"})
		return
	}
	now := time.Now()
	row := templateConfig{ID: uuid.NewString(), TenantID: writingTenantID(c), CategoryID: req.CategoryID, Name: strings.TrimSpace(req.Name), Content: strings.TrimSpace(req.Content), IsDefault: req.IsDefault, IsBuiltin: req.IsBuiltin, Sort: req.Sort, CreatedAt: now, UpdatedAt: now}
	db := middleware.TenantDB(c, h.db)
	err := db.Transaction(func(tx *gorm.DB) error {
		if row.IsDefault {
			if err := tx.Model(&templateConfig{}).Where("tenant_id=? AND category_id=?", row.TenantID, row.CategoryID).Updates(map[string]interface{}{"is_default": false}).Error; err != nil {
				return err
			}
		}
		return tx.Create(&row).Error
	})
	if err != nil {
		c.JSON(409, gin.H{"error": "failed to create admin writing template"})
		return
	}
	c.JSON(201, gin.H{"template": row})
}
func (h *SDPivotWritingManagementHandler) UpdateAdminTemplate(c *gin.Context) {
	var req struct {
		Name, Content *string
		IsDefault     *bool
		Sort          *int
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request body"})
		return
	}
	u := map[string]interface{}{"updated_at": time.Now()}
	if req.Name != nil {
		u["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Content != nil {
		u["content"] = strings.TrimSpace(*req.Content)
	}
	if req.IsDefault != nil {
		u["is_default"] = *req.IsDefault
	}
	if req.Sort != nil {
		u["sort"] = *req.Sort
	}
	h.update(c, &templateConfig{}, u, "template")
}
func (h *SDPivotWritingManagementHandler) DeleteAdminTemplate(c *gin.Context) {
	h.delete(c, &templateConfig{}, "template")
}
func (h *SDPivotWritingManagementHandler) GetMyWritingTemplates(c *gin.Context) {
	pref := h.loadPref(c)
	if c.Query("resolved") != "1" {
		c.JSON(200, gin.H{"preferences": pref})
		return
	}
	var rows []templateConfig
	_ = middleware.TenantDB(c, h.db).Where("tenant_id=? AND is_default=true", writingTenantID(c)).Find(&rows).Error
	c.JSON(200, gin.H{"preferences": pref, "resolved_templates": rows})
}
func (h *SDPivotWritingManagementHandler) CreateMyWritingTemplate(c *gin.Context) {
	var req struct {
		CategoryID, Name, Content string
		Sort                      int
		SetDefault                bool
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.CategoryID == "" || req.Name == "" || req.Content == "" {
		c.JSON(400, gin.H{"error": "category_id, name, and content are required"})
		return
	}
	pref := h.loadPref(c)
	var items []map[string]interface{}
	_ = json.Unmarshal(pref.Templates, &items)
	item := map[string]interface{}{"id": uuid.NewString(), "category_id": req.CategoryID, "name": req.Name, "content": req.Content, "sort": req.Sort, "created_at": time.Now(), "updated_at": time.Now()}
	items = append(items, item)
	if req.SetDefault {
		id := item["id"].(string)
		pref.DefaultTemplateID = &id
	}
	pref.Templates, _ = json.Marshal(items)
	if err := h.savePref(c, pref); err != nil {
		c.JSON(500, gin.H{"error": "failed to save personal writing template"})
		return
	}
	c.JSON(201, gin.H{"template": item, "preferences": pref})
}
func (h *SDPivotWritingManagementHandler) UpdateMyWritingTemplate(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "personal template update is not available in this backport"})
}
func (h *SDPivotWritingManagementHandler) DeleteMyWritingTemplate(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "personal template delete is not available in this backport"})
}
func (h *SDPivotWritingManagementHandler) loadPref(c *gin.Context) userTemplatePref {
	db := middleware.TenantDB(c, h.db)
	var p userTemplatePref
	if db.Where("tenant_id=? AND user_id=?", writingTenantID(c), writingUserID(c)).First(&p).Error != nil {
		p = userTemplatePref{ID: uuid.NewString(), TenantID: writingTenantID(c), UserID: writingUserID(c), CategoryOrder: json.RawMessage("[]"), Templates: json.RawMessage("[]"), CreatedAt: time.Now(), UpdatedAt: time.Now()}
	}
	if len(p.Templates) == 0 {
		p.Templates = json.RawMessage("[]")
	}
	return p
}
func (h *SDPivotWritingManagementHandler) savePref(c *gin.Context, p userTemplatePref) error {
	p.UpdatedAt = time.Now()
	return middleware.TenantDB(c, h.db).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "tenant_id"}, {Name: "user_id"}}, DoUpdates: clause.AssignmentColumns([]string{"default_template_id", "category_order", "templates", "updated_at"})}).Create(&p).Error
}
