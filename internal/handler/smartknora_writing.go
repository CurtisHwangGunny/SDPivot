package handler

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/application/service"
	"github.com/Tencent/WeKnora/internal/config"
	infra_web_search "github.com/Tencent/WeKnora/internal/infrastructure/web_search"
	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// SmartKnoraWritingHandler handles AI writing assistant.
type SmartKnoraWritingHandler struct {
	db                    *gorm.DB
	llm                   *SmartKnoraLLMService
	webSearchService      interfaces.WebSearchService
	webSearchProviderRepo interfaces.WebSearchProviderRepository
}

// NewSmartKnoraWritingHandler creates a new writing handler.
func NewSmartKnoraWritingHandler(db *gorm.DB) *SmartKnoraWritingHandler {
	registry := infra_web_search.NewRegistry()
	registerSmartKnoraWebSearchProviders(registry)
	providerRepo := repository.NewWebSearchProviderRepository(db)
	webSearchService, err := service.NewWebSearchService(&config.Config{}, registry, providerRepo)
	if err != nil {
		webSearchService = nil
	}
	return &SmartKnoraWritingHandler{
		db:                    db,
		llm:                   NewSmartKnoraLLMService(db),
		webSearchService:      webSearchService,
		webSearchProviderRepo: providerRepo,
	}
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
	tenantDB := middleware.TenantDB(c, h.db)
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	var req struct {
		Title            string `json:"title"`
		Category         string `json:"category"`
		SpaceID          string `json:"space_id"`
		SourceType       string `json:"source_type"`
		WebSearchEnabled bool   `json:"web_search_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	req.SourceType, req.WebSearchEnabled = normalizeWritingSource(req.SourceType, req.WebSearchEnabled)
	if req.SpaceID != "" {
		if _, ok := authorizeSpace(c, tenantDB, req.SpaceID, spaceAccessView); !ok {
			return
		}
	}

	draft := types.WritingDraft{
		ID:               uuid.New().String(),
		UserID:           userID,
		TenantID:         tenantID,
		Title:            req.Title,
		Category:         req.Category,
		SpaceID:          req.SpaceID,
		SourceType:       req.SourceType,
		WebSearchEnabled: req.WebSearchEnabled,
		Status:           "draft",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	tenantDB.Create(&draft)
	c.JSON(http.StatusCreated, gin.H{"draft": draft})
}

// ListDrafts lists writing drafts.
func (h *SmartKnoraWritingHandler) ListDrafts(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	var drafts []types.WritingDraft
	tenantDB.Where("user_id = ? AND tenant_id = ?", userID, tenantID).
		Order("updated_at DESC").Find(&drafts)

	c.JSON(http.StatusOK, gin.H{"drafts": drafts})
}

// GetDraft gets a specific draft.
func (h *SmartKnoraWritingHandler) GetDraft(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	draftID := c.Param("id")
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	var draft types.WritingDraft
	if err := tenantDB.Where("id = ? AND user_id = ? AND tenant_id = ?", draftID, userID, tenantID).First(&draft).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"draft": draft})
}

// UpdateDraft updates a draft.
func (h *SmartKnoraWritingHandler) UpdateDraft(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	draftID := c.Param("id")
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

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

	tenantDB.Model(&types.WritingDraft{}).Where("id = ? AND user_id = ? AND tenant_id = ?", draftID, userID, tenantID).Updates(updates)
	c.JSON(http.StatusOK, gin.H{"message": "draft updated"})
}

// DeleteDraft deletes a draft.
func (h *SmartKnoraWritingHandler) DeleteDraft(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	draftID := c.Param("id")
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)
	tenantDB.Where("id = ? AND user_id = ? AND tenant_id = ?", draftID, userID, tenantID).Delete(&types.WritingDraft{})
	c.JSON(http.StatusOK, gin.H{"message": "draft deleted"})
}

// GenerateContent generates AI content for a draft.
func (h *SmartKnoraWritingHandler) GenerateContent(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	var req struct {
		Category         string `json:"category" binding:"required"`
		Prompt           string `json:"prompt" binding:"required"`
		SpaceID          string `json:"space_id"`
		SourceType       string `json:"source_type"`
		WebSearchEnabled bool   `json:"web_search_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tenantID := middleware.GetTenantID(c)
	req.SourceType, req.WebSearchEnabled = normalizeWritingSource(req.SourceType, req.WebSearchEnabled)
	if req.SpaceID == "" {
		req.SpaceID = h.defaultWritingSpaceID(tenantDB, tenantID, req.Category)
	}
	if req.SpaceID != "" {
		if _, ok := authorizeSpace(c, tenantDB, req.SpaceID, spaceAccessView); !ok {
			return
		}
	}
	chunks, err := h.searchRelevantWritingChunks(tenantDB, tenantID, middleware.GetUserID(c), req.SpaceID, req.Prompt, 5)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search knowledge base"})
		return
	}
	var webResults []*types.WebSearchResult
	if req.WebSearchEnabled {
		webResults, err = h.searchWritingWebResults(c.Request.Context(), tenantID, req.Prompt)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to search internet", "detail": err.Error()})
			return
		}
	}
	categoryLabel := smartKnoraWritingCategoryLabel(req.Category)
	systemPrompt := "你是 SmartKnora 的企业写作助手。请根据用户写作要求和知识库参考内容生成结构清晰、可直接编辑的中文 Markdown 文稿。不要编造参考资料中没有的事实；如资料不足，请在文末列出需要补充的信息。"
	userPrompt := buildSmartKnoraWritingPrompt(categoryLabel, req.Prompt, req.SourceType, req.WebSearchEnabled, chunks, webResults)
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
	c.JSON(http.StatusOK, gin.H{"content": content, "category": req.Category, "source_type": req.SourceType, "web_search_enabled": req.WebSearchEnabled, "sources_count": len(chunks) + len(webResults), "knowledge_sources_count": len(chunks), "web_sources_count": len(webResults), "model_id": llmResult.ModelID, "model": llmResult.ModelName})
}

func smartKnoraWritingCategoryLabel(category string) string {
	categoryTemplates := map[string]string{
		"notice":                "通知",
		"announcement":          "公告",
		"tech_doc":              "技术文档",
		"meeting_minutes":       "会议纪要",
		"policy_interpretation": "制度解读",
		"report":                "报告",
		"work_summary":          "工作总结",
		"research_report":       "研究报告",
	}
	if label := categoryTemplates[category]; label != "" {
		return label
	}
	return category
}

func normalizeWritingSource(sourceType string, webSearchEnabled bool) (string, bool) {
	switch sourceType {
	case "knowledge_plus_web", "web_search":
		return "knowledge_plus_web", true
	default:
		if webSearchEnabled {
			return "knowledge_plus_web", true
		}
		return "knowledge_base", false
	}
}

func (h *SmartKnoraWritingHandler) defaultWritingSpaceID(tenantDB *gorm.DB, tenantID uint64, category string) string {
	var cfg types.WriteCategoryConfig
	if err := tenantDB.Where("tenant_id = ? AND category = ?", tenantID, category).First(&cfg).Error; err == nil {
		return cfg.DefaultSpaceID
	}
	return ""
}

func buildSmartKnoraWritingPrompt(categoryLabel string, prompt string, sourceType string, webSearchEnabled bool, chunks []types.SmartKnoraDocumentChunk, webResults []*types.WebSearchResult) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("写作类型:%s\n", categoryLabel))
	b.WriteString("写作要求:\n")
	b.WriteString(prompt)
	b.WriteString("\n\n知识来源:")
	if webSearchEnabled || sourceType == "knowledge_plus_web" {
		b.WriteString("知识库 + 互联网搜索\n")
	} else {
		b.WriteString("仅知识库\n")
	}
	b.WriteString("\n知识库参考内容:\n")
	if len(chunks) == 0 {
		b.WriteString("（未检索到相关知识库内容）\n")
	} else {
		for i, chunk := range chunks {
			b.WriteString(fmt.Sprintf("[%d] 文档ID:%s\n%s\n\n", i+1, chunk.DocumentID, chunk.Content))
		}
	}
	if webSearchEnabled || sourceType == "knowledge_plus_web" {
		b.WriteString("互联网搜索参考内容:\n")
		if len(webResults) == 0 {
			b.WriteString("（未检索到相关互联网搜索结果）\n")
		} else {
			for i, result := range webResults {
				b.WriteString(fmt.Sprintf("[W%d] 标题:%s\n来源:%s\n链接:%s\n摘要:%s\n\n", i+1, result.Title, result.Source, result.URL, resultSnippet(result)))
			}
		}
	}
	b.WriteString("请综合知识库参考内容与互联网搜索参考内容，优先采用知识库中已有的内部事实；互联网搜索内容仅作为补充背景和公开信息来源。请生成一份结构完整、标题清晰、段落可读的 Markdown 文稿。")
	return b.String()
}

func registerSmartKnoraWebSearchProviders(registry *infra_web_search.Registry) {
	registry.Register(string(types.WebSearchProviderTypeDuckDuckGo), infra_web_search.NewDuckDuckGoProvider)
	registry.Register(string(types.WebSearchProviderTypeGoogle), infra_web_search.NewGoogleProvider)
	registry.Register(string(types.WebSearchProviderTypeBing), infra_web_search.NewBingProvider)
	registry.Register(string(types.WebSearchProviderTypeTavily), infra_web_search.NewTavilyProvider)
	registry.Register(string(types.WebSearchProviderTypeOllama), infra_web_search.NewOllamaProvider)
	registry.Register(string(types.WebSearchProviderTypeBaidu), infra_web_search.NewBaiduProvider)
	registry.Register(string(types.WebSearchProviderTypeSearxng), infra_web_search.NewSearxngProvider)
}

func (h *SmartKnoraWritingHandler) searchWritingWebResults(ctx context.Context, tenantID uint64, query string) ([]*types.WebSearchResult, error) {
	if h.webSearchService == nil {
		return nil, fmt.Errorf("web search service is not available")
	}
	searchCtx := context.WithValue(ctx, types.TenantIDContextKey, tenantID)
	cfg := types.DefaultWebSearchConfig()
	cfg.MaxResults = 5
	cfg.IncludeDate = true
	providerID := ""
	if h.webSearchProviderRepo != nil {
		provider, err := h.webSearchProviderRepo.GetDefault(searchCtx, tenantID)
		if err != nil {
			return nil, fmt.Errorf("failed to load default web search provider: %w", err)
		}
		if provider != nil {
			providerID = provider.ID
		}
	}
	if providerID == "" {
		provider, err := infra_web_search.NewDuckDuckGoProvider(types.WebSearchProviderParameters{})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize DuckDuckGo fallback: %w", err)
		}
		return provider.Search(searchCtx, query, cfg.MaxResults, cfg.IncludeDate)
	}
	return h.webSearchService.Search(searchCtx, providerID, cfg, query)
}

func resultSnippet(result *types.WebSearchResult) string {
	if result == nil {
		return ""
	}
	if strings.TrimSpace(result.Snippet) != "" {
		return result.Snippet
	}
	if strings.TrimSpace(result.Content) != "" {
		content := strings.TrimSpace(result.Content)
		runes := []rune(content)
		if len(runes) > 180 {
			return string(runes[:180])
		}
		return content
	}
	return ""
}

func (h *SmartKnoraWritingHandler) searchRelevantWritingChunks(tenantDB *gorm.DB, tenantID uint64, userID string, spaceID string, query string, topK int) ([]types.SmartKnoraDocumentChunk, error) {
	if topK <= 0 || topK > 20 {
		topK = 5
	}
	search := escapeWritingQuery(query)
	db := tenantDB.Model(&types.SmartKnoraDocumentChunk{}).
		Joins("JOIN documents ON documents.id = document_chunks.document_id AND documents.tenant_id = document_chunks.tenant_id").
		Where("document_chunks.tenant_id = ? AND documents.deleted_at IS NULL AND documents.parse_status = ? AND document_chunks.content ILIKE ?", tenantID, "completed", "%"+search+"%")
	if spaceID != "" {
		db = db.Where("documents.space_id = ?", spaceID)
	} else {
		db = db.Where("documents.space_id IN (?)", visibleSpaceIDsQuery(tenantDB, tenantID, userID))
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
	tenantDB := middleware.TenantDB(c, h.db)
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
	if err := tenantDB.Where("id = ? AND user_id = ? AND tenant_id = ?", draftID, userID, tenantID).First(&draft).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
		return
	}
	filename := sanitizeExportFilename(draft.Title)
	if filename == "" {
		filename = "smartknora-draft"
	}
	content := draft.Content
	if strings.TrimSpace(content) == "" {
		content = fmt.Sprintf("# %s\n\n", draft.Title)
	}
	if req.Format == "pdf" {
		if !strings.HasSuffix(strings.ToLower(filename), ".pdf") {
			filename += ".pdf"
		}
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		c.Data(http.StatusOK, "application/pdf", buildSimplePDF(content))
		return
	}
	if req.Format == "docx" {
		if !strings.HasSuffix(strings.ToLower(filename), ".docx") {
			filename += ".docx"
		}
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", buildSimpleDocx(content))
		return
	}
	if !strings.HasSuffix(strings.ToLower(filename), ".md") {
		filename += ".md"
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

func buildSimpleDocx(markdown string) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	writeZipFile := func(name, content string) {
		w, _ := zw.Create(name)
		_, _ = w.Write([]byte(content))
	}
	writeZipFile("[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`)
	writeZipFile("_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`)
	var body strings.Builder
	for _, line := range strings.Split(markdown, "\n") {
		line = strings.TrimSpace(strings.TrimLeft(line, "#"))
		if line == "" {
			body.WriteString("<w:p/>")
			continue
		}
		body.WriteString("<w:p><w:r><w:t xml:space=\"preserve\">")
		body.WriteString(html.EscapeString(line))
		body.WriteString("</w:t></w:r></w:p>")
	}
	writeZipFile("word/document.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`+body.String()+`<w:sectPr><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440"/></w:sectPr></w:body></w:document>`)
	_ = zw.Close()
	return buf.Bytes()
}

func buildSimplePDF(markdown string) []byte {
	lines := strings.Split(markdown, "\n")
	contentLines := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(strings.TrimLeft(line, "#"))
		if trimmed == "" {
			contentLines = append(contentLines, "")
			continue
		}
		for len(trimmed) > 72 {
			contentLines = append(contentLines, trimmed[:72])
			trimmed = trimmed[72:]
		}
		contentLines = append(contentLines, trimmed)
	}
	if len(contentLines) == 0 {
		contentLines = []string{"SmartKnora Draft"}
	}

	var stream strings.Builder
	stream.WriteString("BT\n/F1 12 Tf\n50 792 Td\n14 TL\n")
	first := true
	for _, line := range contentLines {
		escaped := escapePDFText(line)
		if first {
			stream.WriteString(fmt.Sprintf("(%s) Tj\n", escaped))
			first = false
			continue
		}
		stream.WriteString("T*\n")
		stream.WriteString(fmt.Sprintf("(%s) Tj\n", escaped))
	}
	stream.WriteString("ET")
	streamStr := stream.String()

	objects := []string{
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n",
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n",
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>\nendobj\n",
		"4 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n",
		fmt.Sprintf("5 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n", len(streamStr), streamStr),
	}

	var pdf strings.Builder
	pdf.WriteString("%PDF-1.4\n")
	offsets := make([]int, 0, len(objects)+1)
	for _, obj := range objects {
		offsets = append(offsets, pdf.Len())
		pdf.WriteString(obj)
	}
	xrefStart := pdf.Len()
	pdf.WriteString(fmt.Sprintf("xref\n0 %d\n", len(objects)+1))
	pdf.WriteString("0000000000 65535 f \n")
	for _, offset := range offsets {
		pdf.WriteString(fmt.Sprintf("%010d 00000 n \n", offset))
	}
	pdf.WriteString(fmt.Sprintf("trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", len(objects)+1, xrefStart))
	return []byte(pdf.String())
}

func escapePDFText(input string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "(", "\\(", ")", "\\)")
	return replacer.Replace(input)
}
