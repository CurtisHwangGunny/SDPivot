package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
)

// SmartKnoraWritingHandler handles AI writing assistant.
type SmartKnoraWritingHandler struct {
	db  *gorm.DB
	llm *SmartKnoraLLMService
}

// NewSmartKnoraWritingHandler creates a new writing handler.
func NewSmartKnoraWritingHandler(db *gorm.DB) *SmartKnoraWritingHandler {
	return &SmartKnoraWritingHandler{db: db, llm: NewSmartKnoraLLMService(db)}
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

	// Ops routes moved to smartknora_ops_admin.go
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
	tenantID := middleware.GetTenantID(c)
	chunks, err := h.searchRelevantWritingChunks(tenantID, req.SpaceID, req.Prompt, 5)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search knowledge base"})
		return
	}
	categoryLabel := smartKnoraWritingCategoryLabel(req.Category)
	systemPrompt := "你是 SmartKnora 的企业写作助手。请根据用户写作要求和知识库参考内容生成结构清晰、可直接编辑的中文 Markdown 文稿。不要编造参考资料中没有的事实；如资料不足，请在文末列出需要补充的信息。"
	userPrompt := buildSmartKnoraWritingPrompt(categoryLabel, req.Prompt, chunks)
	llmResult, err := h.llm.Generate(c.Request.Context(), tenantID, systemPrompt, userPrompt, 2200)
	if err != nil {
		status := http.StatusBadGateway
		message := "failed to call configured llm"
		if errors.Is(err, ErrSmartKnoraLLMNotConfigured) {
			status = http.StatusPreconditionFailed
			message = "llm model is not configured. Please configure a KnowledgeQA model in WeKnora model settings first"
		}
		c.JSON(status, gin.H{"error": message, "detail": err.Error()})
		return
	}
	content := llmResult.Content
	if strings.TrimSpace(content) == "" {
		content = fmt.Sprintf("# %s\n\n模型未返回有效内容，请稍后重试。", categoryLabel)
	}
	userID := middleware.GetUserID(c)
	h.llm.recordUsage(tenantID, userID, llmResult.ModelID, "/api/v1/smartknora/writing/generate", llmResult.PromptTokens, llmResult.CompletionTokens, llmResult.TotalTokens)
	c.JSON(http.StatusOK, gin.H{"content": content, "category": req.Category, "sources_count": len(chunks), "model_id": llmResult.ModelID, "model": llmResult.ModelName})
}

func smartKnoraWritingCategoryLabel(category string) string {
	categoryTemplates := map[string]string{"work_summary": "工作总结", "research_report": "研究报告", "project_proposal": "项目方案", "meeting_minutes": "会议纪要", "tech_doc": "技术文档", "business_plan": "商业计划书", "weekly_report": "周报日报", "notice": "通知公告"}
	if label := categoryTemplates[category]; label != "" {
		return label
	}
	return category
}

func buildSmartKnoraWritingPrompt(categoryLabel string, prompt string, chunks []types.SmartKnoraDocumentChunk) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("写作类型:%s\n", categoryLabel))
	b.WriteString("写作要求:\n")
	b.WriteString(prompt)
	b.WriteString("\n\n知识库参考内容:\n")
	if len(chunks) == 0 {
		b.WriteString("（未检索到相关知识库内容）\n")
	} else {
		for i, chunk := range chunks {
			b.WriteString(fmt.Sprintf("[%d] 文档ID:%s\n%s\n\n", i+1, chunk.DocumentID, chunk.Content))
		}
	}
	b.WriteString("请生成一份结构完整、标题清晰、段落可读的 Markdown 文稿。")
	return b.String()
}

func (h *SmartKnoraWritingHandler) searchRelevantWritingChunks(tenantID uint64, spaceID string, query string, topK int) ([]types.SmartKnoraDocumentChunk, error) {
	if topK <= 0 || topK > 20 {
		topK = 5
	}
	search := escapeWritingQuery(query)
	db := h.db.Model(&types.SmartKnoraDocumentChunk{}).
		Joins("JOIN documents ON documents.id = document_chunks.document_id AND documents.tenant_id = document_chunks.tenant_id").
		Where("document_chunks.tenant_id = ? AND documents.deleted_at IS NULL AND documents.parse_status = ? AND document_chunks.content ILIKE ?", tenantID, "completed", "%"+search+"%")
	if spaceID != "" {
		db = db.Where("documents.space_id = ?", spaceID)
	}
	var chunks []types.SmartKnoraDocumentChunk
	err := db.Order("document_chunks.created_at DESC").Limit(topK).Find(&chunks).Error
	return chunks, err
}

func escapeWritingQuery(input string) string {
	value := strings.ReplaceAll(input, `\`, `\\`)
	value = strings.ReplaceAll(value, "%", `\%`)
	value = strings.ReplaceAll(value, "_", `\_`)
	return value
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
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)
	var draft types.WritingDraft
	if err := h.db.Where("id = ? AND user_id = ? AND tenant_id = ?", draftID, userID, tenantID).First(&draft).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
		return
	}
	if req.Format != "markdown" {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "only markdown export is currently supported", "format": req.Format})
		return
	}
	filename := sanitizeExportFilename(draft.Title)
	if filename == "" {
		filename = "smartknora-draft"
	}
	if !strings.HasSuffix(strings.ToLower(filename), ".md") {
		filename += ".md"
	}
	content := draft.Content
	if strings.TrimSpace(content) == "" {
		content = fmt.Sprintf("# %s\n\n", draft.Title)
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Data(http.StatusOK, "text/markdown; charset=utf-8", []byte(content))
}

func sanitizeExportFilename(name string) string {
	name = strings.TrimSpace(name)
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", "\"", "-", "<", "-", ">", "-", "|", "-", "\n", " ", "\r", " ")
	name = strings.TrimSpace(replacer.Replace(name))
	if len([]rune(name)) > 80 {
		name = string([]rune(name)[:80])
	}
	return name
}
