package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
)

// SmartKnoraWritingHandler handles AI writing assistant.
type SmartKnoraWritingHandler struct {
	db *gorm.DB
}

// NewSmartKnoraWritingHandler creates a new writing handler.
func NewSmartKnoraWritingHandler(db *gorm.DB) *SmartKnoraWritingHandler {
	return &SmartKnoraWritingHandler{db: db}
}

// RegisterRoutes registers writing and ops routes.
func (h *SmartKnoraWritingHandler) RegisterRoutes(rg *gin.RouterGroup) {
	w := rg.Group("/writing")
	{
		w.POST("/drafts", h.CreateDraft)
		w.GET("/drafts", h.ListDrafts)
		w.GET("/drafts/:id", h.GetDraft)
		w.PUT("/drafts/:id", h.UpdateDraft)
		w.DELETE("/drafts/:id", h.DeleteDraft)
		w.POST("/generate", h.GenerateContent)
		w.POST("/drafts/:id/export", h.ExportDraft)
	}

	ops := rg.Group("/ops")
	{
		ops.GET("/dashboard", h.GetOpsDashboard)
		ops.GET("/tenants", h.ListTenants)
		ops.GET("/audit-log", h.GetAuditLog)
		ops.POST("/announcements", h.CreateAnnouncement)
		ops.GET("/announcements", h.ListAnnouncements)
	}
}

// CreateDraft creates a new writing draft.
func (h *SmartKnoraWritingHandler) CreateDraft(c *gin.Context) {
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	var req struct {
		Title    string `json:"title"`
		Category string `json:"category"`
		SpaceID  string `json:"space_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	draft := types.WritingDraft{
		ID:        uuid.New().String(),
		UserID:    userID,
		TenantID:  tenantID,
		Title:     req.Title,
		Category:  req.Category,
		SpaceID:   req.SpaceID,
		Status:    "draft",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	h.db.Create(&draft)
	c.JSON(http.StatusCreated, gin.H{"draft": draft})
}

// ListDrafts lists writing drafts.
func (h *SmartKnoraWritingHandler) ListDrafts(c *gin.Context) {
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	var drafts []types.WritingDraft
	h.db.Where("user_id = ? AND tenant_id = ?", userID, tenantID).
		Order("updated_at DESC").Find(&drafts)

	c.JSON(http.StatusOK, gin.H{"drafts": drafts})
}

// GetDraft gets a specific draft.
func (h *SmartKnoraWritingHandler) GetDraft(c *gin.Context) {
	draftID := c.Param("id")
	userID := middleware.GetUserID(c)

	var draft types.WritingDraft
	if err := h.db.Where("id = ? AND user_id = ?", draftID, userID).First(&draft).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"draft": draft})
}

// UpdateDraft updates a draft.
func (h *SmartKnoraWritingHandler) UpdateDraft(c *gin.Context) {
	draftID := c.Param("id")
	userID := middleware.GetUserID(c)

	var req struct {
		Title   *string `json:"title"`
		Content *string `json:"content"`
		Status  *string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{"updated_at": time.Now()}
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Content != nil {
		updates["content"] = *req.Content
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	h.db.Model(&types.WritingDraft{}).Where("id = ? AND user_id = ?", draftID, userID).Updates(updates)
	c.JSON(http.StatusOK, gin.H{"message": "draft updated"})
}

// DeleteDraft deletes a draft.
func (h *SmartKnoraWritingHandler) DeleteDraft(c *gin.Context) {
	draftID := c.Param("id")
	userID := middleware.GetUserID(c)
	h.db.Where("id = ? AND user_id = ?", draftID, userID).Delete(&types.WritingDraft{})
	c.JSON(http.StatusOK, gin.H{"message": "draft deleted"})
}

// GenerateContent generates AI content for a draft.
func (h *SmartKnoraWritingHandler) GenerateContent(c *gin.Context) {
	var req struct {
		Category string `json:"category" binding:"required"`
		Prompt   string `json:"prompt" binding:"required"`
		SpaceID  string `json:"space_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Call LLM service for content generation
	// For now, return placeholder
	c.JSON(http.StatusOK, gin.H{
		"content": "AI 内容生成占位符。集成 LLM 服务后将返回真实生成内容。\n\n类别: " + req.Category + "\n提示: " + req.Prompt,
		"category": req.Category,
	})
}

// ExportDraft exports a draft to PDF/DOCX/Markdown.
func (h *SmartKnoraWritingHandler) ExportDraft(c *gin.Context) {
	draftID := c.Param("id")

	var req struct {
		Format string `json:"format" binding:"required,oneof=pdf docx markdown"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var draft types.WritingDraft
	if err := h.db.Where("id = ?", draftID).First(&draft).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
		return
	}

	// TODO: Implement actual export (Pandoc integration)
	c.JSON(http.StatusOK, gin.H{
		"message": "export initiated",
		"format":  req.Format,
		"status":  "pending",
	})
}

// GetOpsDashboard returns operations dashboard data.
func (h *SmartKnoraWritingHandler) GetOpsDashboard(c *gin.Context) {
	role, _ := c.Get("role")
	if role != "admin" && role != "owner" {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}
	var tenantCount int64
	h.db.Table("org_ext").Count(&tenantCount)

	var userCount int64
	h.db.Table("users").Count(&userCount)

	var docCount int64
	h.db.Table("documents").Count(&docCount)

	c.JSON(http.StatusOK, gin.H{
		"tenant_count": tenantCount,
		"user_count":   userCount,
		"document_count": docCount,
	})
}

// ListTenants lists all tenants for operations management.
func (h *SmartKnoraWritingHandler) ListTenants(c *gin.Context) {
	role, _ := c.Get("role")
	if role != "admin" && role != "owner" {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}
	// TODO: Implement tenant listing with pagination
	c.JSON(http.StatusOK, gin.H{"tenants": []interface{}{}, "message": "tenant listing placeholder"})
}

// GetAuditLog returns audit log entries.
func (h *SmartKnoraWritingHandler) GetAuditLog(c *gin.Context) {
	role, _ := c.Get("role")
	if role != "admin" && role != "owner" {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}
	// TODO: Implement audit log
	c.JSON(http.StatusOK, gin.H{"logs": []interface{}{}, "message": "audit log placeholder"})
}

// CreateAnnouncement creates a system announcement.
func (h *SmartKnoraWritingHandler) CreateAnnouncement(c *gin.Context) {
	role, _ := c.Get("role")
	if role != "admin" && role != "owner" {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}
	userID := middleware.GetUserID(c)

	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	announcement := types.Announcement{
		ID:        uuid.New().String(),
		Title:     req.Title,
		Content:   req.Content,
		Status:    "published",
		CreatedBy: userID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	h.db.Create(&announcement)
	c.JSON(http.StatusCreated, gin.H{"announcement": announcement})
}

// ListAnnouncements lists system announcements.
func (h *SmartKnoraWritingHandler) ListAnnouncements(c *gin.Context) {
	var announcements []types.Announcement
	h.db.Order("created_at DESC").Limit(50).Find(&announcements)

	c.JSON(http.StatusOK, gin.H{"announcements": announcements})
}
