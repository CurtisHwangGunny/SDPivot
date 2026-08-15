package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Tencent/WeKnora/internal/middleware"
)

type templateConfig struct {
	ID         string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID   uint64    `json:"tenant_id" gorm:"not null;index"`
	CategoryID string    `json:"category_id" gorm:"type:varchar(36);not null;index"`
	Name       string    `json:"name" gorm:"type:varchar(100);not null"`
	Content    string    `json:"content" gorm:"type:text;not null"`
	IsDefault  bool      `json:"is_default" gorm:"not null;default:false"`
	IsBuiltin  bool      `json:"is_builtin" gorm:"not null;default:false"`
	Sort       int       `json:"sort" gorm:"not null;default:0"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (templateConfig) TableName() string { return "template_configs" }

type personalWritingTemplate struct {
	ID         string    `json:"id"`
	CategoryID string    `json:"category_id"`
	Name       string    `json:"name"`
	Content    string    `json:"content"`
	Sort       int       `json:"sort"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type userTemplatePref struct {
	ID                string                    `json:"id" gorm:"type:varchar(36);primaryKey"`
	UserID            string                    `json:"user_id" gorm:"type:varchar(36);not null;index"`
	TenantID          uint64                    `json:"tenant_id" gorm:"not null;index"`
	DefaultTemplateID *string                   `json:"default_template_id" gorm:"type:varchar(36)"`
	CategoryOrder     json.RawMessage           `json:"category_order" gorm:"type:jsonb"`
	Templates         []personalWritingTemplate `json:"templates" gorm:"-"`
	TemplatesJSON     json.RawMessage           `json:"-" gorm:"column:templates;type:jsonb"`
	CreatedAt         time.Time                 `json:"created_at"`
	UpdatedAt         time.Time                 `json:"updated_at"`
}

func (userTemplatePref) TableName() string { return "user_template_prefs" }

type resolvedWritingTemplate struct {
	ID         string `json:"id"`
	CategoryID string `json:"category_id"`
	Name       string `json:"name"`
	Content    string `json:"content"`
	Source     string `json:"source"`
}

func (h *SDPivotWritingManagementHandler) registerTieredTemplateRoutes(rg *gin.RouterGroup) {
	admin := rg.Group("/admin/writing/templates", middleware.RequirePermission(middleware.PermissionDepartmentManage))
	admin.GET("", h.ListAdminTemplates)
	admin.POST("", h.CreateAdminTemplate)
	admin.PUT("/:id", h.UpdateAdminTemplate)
	admin.DELETE("/:id", h.DeleteAdminTemplate)

	my := rg.Group("/my/writing/templates")
	my.GET("", h.GetMyWritingTemplates)
	my.POST("", h.CreateMyWritingTemplate)
	my.PUT("/:id", h.UpdateMyWritingTemplate)
	my.DELETE("/:id", h.DeleteMyWritingTemplate)
}

func (h *SDPivotWritingManagementHandler) ListAdminTemplates(c *gin.Context) {
	query := middleware.TenantDB(c, h.db).Where("tenant_id = ?", middleware.GetTenantID(c))
	if categoryID := strings.TrimSpace(c.Query("category_id")); categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}
	var items []templateConfig
	if err := query.Order("category_id, sort ASC, created_at ASC").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list admin writing templates"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"templates": items})
}

func (h *SDPivotWritingManagementHandler) CreateAdminTemplate(c *gin.Context) {
	var req struct {
		CategoryID string `json:"category_id"`
		Name       string `json:"name"`
		Content    string `json:"content"`
		IsDefault  bool   `json:"is_default"`
		IsBuiltin  bool   `json:"is_builtin"`
		Sort       int    `json:"sort"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.CategoryID) == "" || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Content) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category_id, name, and content are required"})
		return
	}
	db := middleware.TenantDB(c, h.db)
	tenantID := middleware.GetTenantID(c)
	if !writingCategoryExists(db, tenantID, strings.TrimSpace(req.CategoryID)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "writing category not found"})
		return
	}
	now := time.Now()
	item := templateConfig{ID: uuid.NewString(), TenantID: tenantID, CategoryID: strings.TrimSpace(req.CategoryID), Name: strings.TrimSpace(req.Name), Content: strings.TrimSpace(req.Content), IsDefault: req.IsDefault, IsBuiltin: req.IsBuiltin, Sort: req.Sort, CreatedAt: now, UpdatedAt: now}
	if err := db.Transaction(func(tx *gorm.DB) error {
		if item.IsDefault {
			if err := tx.Model(&templateConfig{}).Where("tenant_id = ? AND category_id = ? AND is_default = ?", tenantID, item.CategoryID, true).Updates(map[string]interface{}{"is_default": false, "updated_at": now}).Error; err != nil {
				return err
			}
		}
		return tx.Create(&item).Error
	}); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "failed to create admin writing template"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"template": item})
}

func (h *SDPivotWritingManagementHandler) UpdateAdminTemplate(c *gin.Context) {
	db := middleware.TenantDB(c, h.db)
	tenantID := middleware.GetTenantID(c)
	var item templateConfig
	if err := db.Where("id = ? AND tenant_id = ?", c.Param("id"), tenantID).First(&item).Error; err != nil {
		h.writeRecordError(c, err, "template")
		return
	}
	var req struct {
		CategoryID *string `json:"category_id"`
		Name       *string `json:"name"`
		Content    *string `json:"content"`
		IsDefault  *bool   `json:"is_default"`
		IsBuiltin  *bool   `json:"is_builtin"`
		Sort       *int    `json:"sort"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	categoryID := item.CategoryID
	updates := map[string]interface{}{"updated_at": time.Now()}
	if req.CategoryID != nil {
		categoryID = strings.TrimSpace(*req.CategoryID)
		if !writingCategoryExists(db, tenantID, categoryID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "writing category not found"})
			return
		}
		updates["category_id"] = categoryID
	}
	if req.Name != nil {
		value := strings.TrimSpace(*req.Name)
		if value == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name cannot be empty"})
			return
		}
		updates["name"] = value
	}
	if req.Content != nil {
		value := strings.TrimSpace(*req.Content)
		if value == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "content cannot be empty"})
			return
		}
		updates["content"] = value
	}
	if req.IsDefault != nil {
		updates["is_default"] = *req.IsDefault
	}
	if req.IsBuiltin != nil {
		updates["is_builtin"] = *req.IsBuiltin
	}
	if req.Sort != nil {
		updates["sort"] = *req.Sort
	}
	makeDefault := req.IsDefault != nil && *req.IsDefault
	if req.CategoryID != nil && item.IsDefault && req.IsDefault == nil {
		makeDefault = true
		updates["is_default"] = true
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		if makeDefault {
			if err := tx.Model(&templateConfig{}).Where("tenant_id = ? AND category_id = ? AND id <> ? AND is_default = ?", tenantID, categoryID, item.ID, true).Updates(map[string]interface{}{"is_default": false, "updated_at": time.Now()}).Error; err != nil {
				return err
			}
		}
		result := tx.Model(&templateConfig{}).Where("id = ? AND tenant_id = ?", item.ID, tenantID).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
			return
		}
		c.JSON(http.StatusConflict, gin.H{"error": "failed to update admin writing template"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "template updated"})
}

func (h *SDPivotWritingManagementHandler) DeleteAdminTemplate(c *gin.Context) {
	result := middleware.TenantDB(c, h.db).Where("id = ? AND tenant_id = ?", c.Param("id"), middleware.GetTenantID(c)).Delete(&templateConfig{})
	h.writeDeleteResult(c, result, "template")
}

func (h *SDPivotWritingManagementHandler) GetMyWritingTemplates(c *gin.Context) {
	db := middleware.TenantDB(c, h.db)
	pref, err := loadUserTemplatePref(db, middleware.GetTenantID(c), middleware.GetUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load personal writing templates"})
		return
	}
	if c.Query("resolved") != "1" {
		c.JSON(http.StatusOK, gin.H{"preferences": pref})
		return
	}
	resolved, err := h.resolveWritingTemplates(db, middleware.GetTenantID(c), pref)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve writing templates"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"preferences": pref, "resolved_templates": resolved})
}

func (h *SDPivotWritingManagementHandler) CreateMyWritingTemplate(c *gin.Context) {
	var req struct {
		CategoryID        string          `json:"category_id"`
		Name              string          `json:"name"`
		Content           string          `json:"content"`
		Sort              int             `json:"sort"`
		SetDefault        bool            `json:"set_default"`
		CategoryOrder     json.RawMessage `json:"category_order"`
		DefaultTemplateID *string         `json:"default_template_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.CategoryID) == "" || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Content) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category_id, name, and content are required"})
		return
	}
	db := middleware.TenantDB(c, h.db)
	tenantID := middleware.GetTenantID(c)
	if !writingCategoryExists(db, tenantID, strings.TrimSpace(req.CategoryID)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "writing category not found"})
		return
	}
	now := time.Now()
	item := personalWritingTemplate{ID: uuid.NewString(), CategoryID: strings.TrimSpace(req.CategoryID), Name: strings.TrimSpace(req.Name), Content: strings.TrimSpace(req.Content), Sort: req.Sort, CreatedAt: now, UpdatedAt: now}
	pref, err := loadUserTemplatePref(db, tenantID, middleware.GetUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load personal writing templates"})
		return
	}
	pref.Templates = append(pref.Templates, item)
	if len(req.CategoryOrder) > 0 {
		if !validJSONArray(req.CategoryOrder) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "category_order must be a JSON array"})
			return
		}
		pref.CategoryOrder = req.CategoryOrder
	}
	if req.SetDefault {
		pref.DefaultTemplateID = &item.ID
	} else if req.DefaultTemplateID != nil {
		pref.DefaultTemplateID = normalizedOptionalID(req.DefaultTemplateID)
	}
	if !personalDefaultExists(pref.Templates, pref.DefaultTemplateID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "default_template_id must reference a personal template"})
		return
	}
	if err := saveUserTemplatePref(db, pref); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save personal writing template"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"template": item, "preferences": pref})
}

func (h *SDPivotWritingManagementHandler) UpdateMyWritingTemplate(c *gin.Context) {
	var req struct {
		CategoryID        *string         `json:"category_id"`
		Name              *string         `json:"name"`
		Content           *string         `json:"content"`
		Sort              *int            `json:"sort"`
		SetDefault        *bool           `json:"set_default"`
		DefaultTemplateID *string         `json:"default_template_id"`
		CategoryOrder     json.RawMessage `json:"category_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	db := middleware.TenantDB(c, h.db)
	tenantID := middleware.GetTenantID(c)
	pref, err := loadUserTemplatePref(db, tenantID, middleware.GetUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load personal writing templates"})
		return
	}
	index := personalTemplateIndex(pref.Templates, c.Param("id"))
	if index < 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}
	item := &pref.Templates[index]
	if req.CategoryID != nil {
		value := strings.TrimSpace(*req.CategoryID)
		if !writingCategoryExists(db, tenantID, value) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "writing category not found"})
			return
		}
		item.CategoryID = value
	}
	if req.Name != nil {
		value := strings.TrimSpace(*req.Name)
		if value == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name cannot be empty"})
			return
		}
		item.Name = value
	}
	if req.Content != nil {
		value := strings.TrimSpace(*req.Content)
		if value == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "content cannot be empty"})
			return
		}
		item.Content = value
	}
	if req.Sort != nil {
		item.Sort = *req.Sort
	}
	item.UpdatedAt = time.Now()
	if req.SetDefault != nil {
		if *req.SetDefault {
			pref.DefaultTemplateID = &item.ID
		} else if pref.DefaultTemplateID != nil && *pref.DefaultTemplateID == item.ID {
			pref.DefaultTemplateID = nil
		}
	}
	if req.DefaultTemplateID != nil {
		pref.DefaultTemplateID = normalizedOptionalID(req.DefaultTemplateID)
	}
	if len(req.CategoryOrder) > 0 {
		if !validJSONArray(req.CategoryOrder) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "category_order must be a JSON array"})
			return
		}
		pref.CategoryOrder = req.CategoryOrder
	}
	if !personalDefaultExists(pref.Templates, pref.DefaultTemplateID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "default_template_id must reference a personal template"})
		return
	}
	if err := saveUserTemplatePref(db, pref); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update personal writing template"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"template": item, "preferences": pref})
}

func (h *SDPivotWritingManagementHandler) DeleteMyWritingTemplate(c *gin.Context) {
	db := middleware.TenantDB(c, h.db)
	pref, err := loadUserTemplatePref(db, middleware.GetTenantID(c), middleware.GetUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load personal writing templates"})
		return
	}
	index := personalTemplateIndex(pref.Templates, c.Param("id"))
	if index < 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}
	deletedID := pref.Templates[index].ID
	pref.Templates = append(pref.Templates[:index], pref.Templates[index+1:]...)
	if pref.DefaultTemplateID != nil && *pref.DefaultTemplateID == deletedID {
		pref.DefaultTemplateID = nil
	}
	if err := saveUserTemplatePref(db, pref); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete personal writing template"})
		return
	}
	c.Status(http.StatusNoContent)
}

func loadUserTemplatePref(db *gorm.DB, tenantID uint64, userID string) (userTemplatePref, error) {
	var pref userTemplatePref
	err := db.Where("tenant_id = ? AND user_id = ?", tenantID, userID).First(&pref).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		now := time.Now()
		return userTemplatePref{ID: uuid.NewString(), UserID: userID, TenantID: tenantID, CategoryOrder: json.RawMessage("[]"), Templates: []personalWritingTemplate{}, TemplatesJSON: json.RawMessage("[]"), CreatedAt: now, UpdatedAt: now}, nil
	}
	if err != nil {
		return pref, err
	}
	if len(pref.CategoryOrder) == 0 {
		pref.CategoryOrder = json.RawMessage("[]")
	}
	if len(pref.TemplatesJSON) > 0 {
		if err := json.Unmarshal(pref.TemplatesJSON, &pref.Templates); err != nil {
			return pref, err
		}
	}
	if pref.Templates == nil {
		pref.Templates = []personalWritingTemplate{}
	}
	return pref, nil
}

func saveUserTemplatePref(db *gorm.DB, pref userTemplatePref) error {
	templates, err := json.Marshal(pref.Templates)
	if err != nil {
		return err
	}
	if len(pref.CategoryOrder) == 0 {
		pref.CategoryOrder = json.RawMessage("[]")
	}
	pref.TemplatesJSON = templates
	pref.UpdatedAt = time.Now()
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "tenant_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"default_template_id": pref.DefaultTemplateID,
			"category_order":      pref.CategoryOrder,
			"templates":           pref.TemplatesJSON,
			"updated_at":          pref.UpdatedAt,
		}),
	}).Create(&pref).Error
}

func (h *SDPivotWritingManagementHandler) resolveWritingTemplates(db *gorm.DB, tenantID uint64, pref userTemplatePref) ([]resolvedWritingTemplate, error) {
	var categories []writingCategory
	if err := db.Where("tenant_id = ?", tenantID).Order("sort ASC, created_at ASC").Find(&categories).Error; err != nil {
		return nil, err
	}
	var admins []templateConfig
	if err := db.Where("tenant_id = ? AND is_default = ?", tenantID, true).Order("sort ASC, created_at ASC").Find(&admins).Error; err != nil {
		return nil, err
	}
	var builtins []writingTemplate
	if err := db.Where("tenant_id = ? AND is_builtin = ?", tenantID, true).Order("sort ASC, created_at ASC").Find(&builtins).Error; err != nil {
		return nil, err
	}
	personal := preferredPersonalTemplates(pref.Templates, pref.DefaultTemplateID)
	adminByCategory := make(map[string]templateConfig, len(admins))
	for _, item := range admins {
		if _, ok := adminByCategory[item.CategoryID]; !ok {
			adminByCategory[item.CategoryID] = item
		}
	}
	builtinByCategory := make(map[string]writingTemplate, len(builtins))
	for _, item := range builtins {
		if _, ok := builtinByCategory[item.CategoryID]; !ok {
			builtinByCategory[item.CategoryID] = item
		}
	}
	result := make([]resolvedWritingTemplate, 0, len(categories))
	for _, category := range categories {
		if item, ok := personal[category.ID]; ok {
			result = append(result, resolvedWritingTemplate{ID: item.ID, CategoryID: category.ID, Name: item.Name, Content: item.Content, Source: "personal"})
			continue
		}
		if item, ok := adminByCategory[category.ID]; ok {
			result = append(result, resolvedWritingTemplate{ID: item.ID, CategoryID: category.ID, Name: item.Name, Content: item.Content, Source: "admin"})
			continue
		}
		if item, ok := builtinByCategory[category.ID]; ok {
			result = append(result, resolvedWritingTemplate{ID: item.ID, CategoryID: category.ID, Name: item.Name, Content: item.Content, Source: "builtin"})
		}
	}
	return orderResolvedTemplates(result, pref.CategoryOrder), nil
}

func preferredPersonalTemplates(items []personalWritingTemplate, defaultID *string) map[string]personalWritingTemplate {
	sorted := append([]personalWritingTemplate(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Sort < sorted[j].Sort })
	result := make(map[string]personalWritingTemplate)
	for _, item := range sorted {
		if _, ok := result[item.CategoryID]; !ok {
			result[item.CategoryID] = item
		}
	}
	if defaultID != nil {
		for _, item := range items {
			if item.ID == *defaultID {
				result[item.CategoryID] = item
				break
			}
		}
	}
	return result
}

func orderResolvedTemplates(items []resolvedWritingTemplate, raw json.RawMessage) []resolvedWritingTemplate {
	var order []string
	if len(raw) == 0 || json.Unmarshal(raw, &order) != nil || len(order) == 0 {
		return items
	}
	positions := make(map[string]int, len(order))
	for i, categoryID := range order {
		positions[categoryID] = i
	}
	sort.SliceStable(items, func(i, j int) bool {
		left, leftOK := positions[items[i].CategoryID]
		right, rightOK := positions[items[j].CategoryID]
		if leftOK != rightOK {
			return leftOK
		}
		return leftOK && left < right
	})
	return items
}

func personalTemplateIndex(items []personalWritingTemplate, id string) int {
	for i := range items {
		if items[i].ID == id {
			return i
		}
	}
	return -1
}

func normalizedOptionalID(id *string) *string {
	if id == nil {
		return nil
	}
	value := strings.TrimSpace(*id)
	if value == "" {
		return nil
	}
	return &value
}

func personalDefaultExists(items []personalWritingTemplate, id *string) bool {
	return id == nil || personalTemplateIndex(items, *id) >= 0
}

func validJSONArray(raw json.RawMessage) bool {
	var value []string
	return json.Unmarshal(raw, &value) == nil
}
