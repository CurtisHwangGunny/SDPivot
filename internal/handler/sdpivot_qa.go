package handler

import (
	"encoding/json"
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

// SDPivotQAHandler handles AI Q&A sessions.
type SDPivotQAHandler struct {
	db  *gorm.DB
	llm *SDPivotLLMService
}

// NewSDPivotQAHandler creates a new QA handler.
func NewSDPivotQAHandler(db *gorm.DB) *SDPivotQAHandler {
	return &SDPivotQAHandler{db: db, llm: NewSDPivotLLMService(db)}
}

// RegisterRoutes registers Q&A routes.
func (h *SDPivotQAHandler) RegisterRoutes(rg *gin.RouterGroup) {
	q := rg.Group("/qa")
	{
		q.GET("/models", h.ListModels)
		q.POST("/sessions", h.CreateSession)
		q.GET("/sessions", h.ListSessions)
		q.GET("/sessions/:id", h.GetSession)
		q.POST("/sessions/:id/messages", h.SendMessage)
		q.GET("/sessions/:id/messages", h.GetMessages)
		q.DELETE("/sessions/:id", h.DeleteSession)
	}

	admin := rg.Group("/admin", middleware.RequirePermission(middleware.PermissionUserRoleAssign))
	{
		admin.GET("/members", h.ListAllMembers)
		admin.GET("/stats", h.GetAdminStats)
		admin.GET("/spaces", h.ListAllSpaces)
	}
}

// ListModels returns the active chat models visible to the current tenant.
func (h *SDPivotQAHandler) ListModels(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	tenantID := middleware.GetTenantID(c)
	models := make([]types.Model, 0)
	seen := make(map[string]struct{})
	appendModels := func(scope string, args ...interface{}) error {
		var batch []types.Model
		err := tenantDB.Model(&types.Model{}).
			Where("(tenant_id = ? OR is_builtin = true) AND deleted_at IS NULL AND status = ?", tenantID, types.ModelStatusActive).
			Where("type IN ?", []types.ModelType{types.ModelTypeKnowledgeQA, types.ModelTypeVLLM, types.ModelType("llm")}).
			Where(scope, args...).
			Order("updated_at DESC").
			Find(&batch).Error
		if err != nil {
			return err
		}
		for _, model := range batch {
			if _, exists := seen[model.ID]; exists {
				continue
			}
			seen[model.ID] = struct{}{}
			models = append(models, model)
		}
		return nil
	}
	modelScopes := []struct {
		query string
		args  []interface{}
	}{
		{query: "tenant_id = ? AND is_default = true", args: []interface{}{tenantID}},
		{query: "is_builtin = true AND is_default = true"},
		{query: "tenant_id = ?", args: []interface{}{tenantID}},
		{query: "is_builtin = true"},
	}
	for _, scope := range modelScopes {
		if err := appendModels(scope.query, scope.args...); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list qa models"})
			return
		}
	}
	type availableModel struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
		IsDefault   bool   `json:"is_default"`
	}
	result := make([]availableModel, 0, len(models))
	for _, model := range models {
		result = append(result, availableModel{
			ID:          model.ID,
			Name:        model.Name,
			DisplayName: model.DisplayName,
			IsDefault:   model.IsDefault,
		})
	}
	c.JSON(http.StatusOK, gin.H{"models": result})
}

// CreateSession creates a new Q&A session.
func (h *SDPivotQAHandler) CreateSession(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	var req struct {
		Title   string `json:"title"`
		SpaceID string `json:"space_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if req.Title == "" {
		req.Title = "新对话"
	}
	if req.SpaceID != "" {
		if _, ok := authorizeSpace(c, tenantDB, req.SpaceID, spaceAccessView); !ok {
			return
		}
	}

	session := types.QASession{
		ID:        uuid.New().String(),
		UserID:    userID,
		TenantID:  tenantID,
		SpaceID:   req.SpaceID,
		Title:     req.Title,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := tenantDB.Create(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session", "detail": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"session": session})
}

// ListSessions lists Q&A sessions for the current user.
func (h *SDPivotQAHandler) ListSessions(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	var sessions []types.QASession
	tenantDB.Where("user_id = ? AND tenant_id = ?", userID, tenantID).
		Order("updated_at DESC").Limit(50).Find(&sessions)

	c.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

// GetSession gets a specific Q&A session.
func (h *SDPivotQAHandler) GetSession(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	sessionID := c.Param("id")
	userID := middleware.GetUserID(c)

	var session types.QASession
	if err := tenantDB.Where("id = ? AND user_id = ? AND tenant_id = ?", sessionID, userID, middleware.GetTenantID(c)).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"session": session})
}

// SendMessage sends a message in a Q&A session.
func (h *SDPivotQAHandler) SendMessage(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	sessionID := c.Param("id")
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	var session types.QASession
	if err := tenantDB.Where("id = ? AND user_id = ? AND tenant_id = ?", sessionID, userID, tenantID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	var req struct {
		Content  string   `json:"content" binding:"required"`
		ModelID  string   `json:"model_id" binding:"omitempty,max=64"`
		SpaceID  string   `json:"space_id"`
		SpaceIDs []string `json:"space_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	useSessionSpace := req.SpaceIDs == nil && strings.TrimSpace(req.SpaceID) == "" && session.SpaceID != ""
	if useSessionSpace {
		if _, ok := authorizeSpace(c, tenantDB, session.SpaceID, spaceAccessView); !ok {
			return
		}
	}
	now := time.Now()
	userMsg := types.QAMessage{ID: uuid.New().String(), SessionID: sessionID, TenantID: tenantID, Role: "user", Content: req.Content, Sources: "[]", CreatedAt: now}
	if err := tenantDB.Create(&userMsg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user message", "detail": err.Error()})
		return
	}

	spaceIDs := resolveQASpaceIDs(req.SpaceIDs, req.SpaceID, session.SpaceID)
	chunks, err := h.searchRelevantChunks(tenantDB, tenantID, userID, spaceIDs, req.Content, 5)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search knowledge base"})
		return
	}

	sources, retrievalStatus := buildSDPivotQASources(chunks)

	systemPrompt := "你是 SDPivot 的企业知识库问答助手。请严格基于给定参考资料回答；如果参考资料不足，请说明缺少哪些信息。回答要准确、简洁，并优先使用中文。"
	userPrompt := buildSDPivotQAPrompt(req.Content, chunks)
	llmResult, err := h.llm.GenerateWithModel(c.Request.Context(), tenantID, req.ModelID, systemPrompt, userPrompt, 1400)
	if err != nil {
		status := http.StatusBadGateway
		message := "failed to call configured llm"
		if errors.Is(err, ErrSDPivotLLMNotConfigured) {
			status = http.StatusPreconditionFailed
			message = "llm model is not configured. Please configure a KnowledgeQA model in WeKnora model settings first"
		} else if errors.Is(err, ErrSDPivotLLMModelNotAvailable) {
			status = http.StatusBadRequest
			message = "selected llm model is not available"
		}
		c.JSON(status, gin.H{"error": message, "detail": err.Error()})
		return
	}
	aiContent := llmResult.Content
	if strings.TrimSpace(aiContent) == "" {
		aiContent = "模型未返回有效内容，请稍后重试。"
	}
	h.llm.recordUsage(tenantID, userID, llmResult.ModelID, "/api/v1/sdp/qa/sessions/:id/messages", llmResult.PromptTokens, llmResult.CompletionTokens, llmResult.TotalTokens)

	aiMsg := types.QAMessage{ID: uuid.New().String(), SessionID: sessionID, TenantID: tenantID, Role: "assistant", Content: aiContent, Sources: sources, CreatedAt: now}
	if err := tenantDB.Create(&aiMsg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create assistant message", "detail": err.Error()})
		return
	}
	tenantDB.Model(&types.QASession{}).Where("id = ? AND tenant_id = ?", sessionID, tenantID).Update("updated_at", now)
	c.JSON(http.StatusOK, gin.H{
		"user_message":      userMsg,
		"assistant_message": aiMsg,
		"model_id":          llmResult.ModelID,
		"model":             llmResult.ModelName,
		"retrieval_status":  retrievalStatus,
	})
}

type sdpivotQAChunk struct {
	types.SDPivotDocumentChunk
	DocumentTitle string `json:"document_title"`
	SpaceID       string `json:"space_id"`
	SpaceName     string `json:"space_name"`
}

type sdpivotQASource struct {
	DocumentID    string `json:"document_id"`
	DocumentTitle string `json:"document_title"`
	SpaceID       string `json:"space_id"`
	SpaceName     string `json:"space_name"`
}

func resolveQASpaceIDs(spaceIDs []string, spaceID, sessionSpaceID string) []string {
	if spaceIDs != nil {
		return uniqueStrings(spaceIDs)
	}
	if spaceID = strings.TrimSpace(spaceID); spaceID != "" {
		return []string{spaceID}
	}
	if sessionSpaceID = strings.TrimSpace(sessionSpaceID); sessionSpaceID != "" {
		return []string{sessionSpaceID}
	}
	return nil
}

func buildSDPivotQASources(chunks []sdpivotQAChunk) (string, string) {
	if len(chunks) == 0 {
		return "[]", "no_match"
	}
	sourceSet := make(map[string]struct{}, len(chunks))
	sourceList := make([]sdpivotQASource, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk.DocumentID == "" {
			continue
		}
		if _, exists := sourceSet[chunk.DocumentID]; exists {
			continue
		}
		sourceSet[chunk.DocumentID] = struct{}{}
		sourceList = append(sourceList, sdpivotQASource{
			DocumentID:    chunk.DocumentID,
			DocumentTitle: chunk.DocumentTitle,
			SpaceID:       chunk.SpaceID,
			SpaceName:     chunk.SpaceName,
		})
	}
	if len(sourceList) == 0 {
		return "[]", "no_match"
	}
	sources, err := json.Marshal(sourceList)
	if err != nil {
		return "[]", "no_match"
	}
	return string(sources), "matched"
}

func (h *SDPivotQAHandler) searchRelevantChunks(tenantDB *gorm.DB, tenantID uint64, userID string, spaceIDs []string, query string, topK int) ([]sdpivotQAChunk, error) {
	if topK <= 0 || topK > 20 {
		topK = 5
	}
	keywords := extractQAKeywords(query)
	if len(keywords) == 0 {
		return nil, nil
	}
	db := tenantDB.Model(&types.SDPivotDocumentChunk{}).
		Joins("JOIN documents ON documents.id = document_chunks.document_id AND documents.tenant_id = document_chunks.tenant_id").
		Joins("JOIN knowledge_spaces ON knowledge_spaces.id = documents.space_id AND knowledge_spaces.tenant_id = documents.tenant_id").
		Where("document_chunks.tenant_id = ? AND documents.deleted_at IS NULL AND knowledge_spaces.deleted_at IS NULL AND documents.parse_status = ?", tenantID, "completed").
		Where("documents.space_id IN (?)", visibleSpaceIDsQuery(tenantDB, tenantID, userID))
	orConditions := make([]string, 0, len(keywords))
	orArgs := make([]interface{}, 0, len(keywords))
	for _, keyword := range keywords {
		orConditions = append(orConditions, "document_chunks.content ILIKE ?")
		orArgs = append(orArgs, "%"+escapeQAQuery(keyword)+"%")
	}
	db = db.Where("("+strings.Join(orConditions, " OR ")+")", orArgs...)
	if len(spaceIDs) > 0 {
		db = db.Where("documents.space_id IN ?", spaceIDs)
	}
	var chunks []sdpivotQAChunk
	err := db.Select(`document_chunks.*,
		documents.title AS document_title,
		knowledge_spaces.id AS space_id,
		knowledge_spaces.name AS space_name`).
		Order("document_chunks.created_at DESC").Limit(topK).Scan(&chunks).Error
	return chunks, err
}

func extractQAKeywords(query string) []string {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}

	cleaned := strings.NewReplacer(
		"？", " ", "?", " ", "。", " ", ".", " ",
		"，", " ", ",", " ", "、", " ", "：", " ", ":", " ",
		"的", " ", "了", " ", "是", " ", "在", " ",
		"包含", " ", "包括", " ", "哪些", " ", "什么", " ",
		"如何", " ", "怎么", " ", "请", " ", "根据", " ",
		"知识库", " ", "回答", " ", "引用", " ",
	).Replace(query)

	keywords := make([]string, 0, 8)
	seen := make(map[string]struct{})
	add := func(keyword string) {
		keyword = strings.TrimSpace(keyword)
		if len([]rune(keyword)) < 2 {
			return
		}
		if _, exists := seen[keyword]; exists {
			return
		}
		seen[keyword] = struct{}{}
		keywords = append(keywords, keyword)
	}
	for _, part := range strings.Fields(cleaned) {
		add(part)
		if len(keywords) == 8 {
			return keywords
		}
	}
	if len(keywords) > 0 {
		return keywords
	}

	runes := []rune(query)
	for i := 0; i+2 <= len(runes) && len(keywords) < 8; i++ {
		ngram := string(runes[i : i+2])
		if !isStopQANgram(ngram) {
			add(ngram)
		}
	}
	return keywords
}

func isStopQANgram(ngram string) bool {
	switch ngram {
	case "请根", "根据", "据知", "识库", "库回", "回答", "答请", "请引", "引用", "包含", "括哪", "哪些", "些内":
		return true
	default:
		return false
	}
}

func escapeQAQuery(input string) string {
	value := strings.ReplaceAll(input, `\`, `\\`)
	value = strings.ReplaceAll(value, "%", `\%`)
	value = strings.ReplaceAll(value, "_", `\_`)
	return value
}

func buildSDPivotQAPrompt(question string, chunks []sdpivotQAChunk) string {
	var b strings.Builder
	b.WriteString("用户问题:\n")
	b.WriteString(question)
	b.WriteString("\n\n参考资料:\n")
	if len(chunks) == 0 {
		b.WriteString("（未检索到相关知识库内容）\n")
	} else {
		for i, chunk := range chunks {
			b.WriteString(fmt.Sprintf("[%d] 空间:%s（%s） 文档:%s（%s）\n%s\n\n", i+1, chunk.SpaceName, chunk.SpaceID, chunk.DocumentTitle, chunk.DocumentID, chunk.Content))
		}
	}
	b.WriteString("请输出最终回答，并在必要时说明依据来自哪些参考资料编号。")
	return b.String()
}

// GetMessages gets messages in a Q&A session.
func (h *SDPivotQAHandler) GetMessages(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	sessionID := c.Param("id")
	userID := middleware.GetUserID(c)

	// Verify session belongs to user
	var session types.QASession
	if err := tenantDB.Where("id = ? AND user_id = ? AND tenant_id = ?", sessionID, userID, middleware.GetTenantID(c)).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	var messages []types.QAMessage
	tenantDB.Where("session_id = ?", sessionID).Order("created_at ASC").Find(&messages)

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

// DeleteSession deletes a Q&A session.
func (h *SDPivotQAHandler) DeleteSession(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	sessionID := c.Param("id")
	userID := middleware.GetUserID(c)

	// Verify session belongs to user
	var session types.QASession
	if err := tenantDB.Where("id = ? AND user_id = ? AND tenant_id = ?", sessionID, userID, middleware.GetTenantID(c)).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	tenantDB.Where("session_id = ?", sessionID).Delete(&types.QAMessage{})
	tenantDB.Where("id = ? AND user_id = ? AND tenant_id = ?", sessionID, userID, middleware.GetTenantID(c)).Delete(&types.QASession{})

	c.JSON(http.StatusOK, gin.H{"message": "session deleted"})
}

// ListAllMembers lists all members across organizations (admin view).
func (h *SDPivotQAHandler) ListAllMembers(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	if !middleware.HasPermission(middleware.GetRole(c), middleware.PermissionUserRoleAssign) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}
	tenantID := middleware.GetTenantID(c)

	var members []types.SDPivotOrgMember
	tenantDB.Joins("JOIN org_ext ON org_ext.org_id = org_members.org_id").
		Where("org_ext.tenant_id = ?", tenantID).
		Find(&members)

	c.JSON(http.StatusOK, gin.H{"members": members})
}

// GetAdminStats returns admin dashboard statistics.
func (h *SDPivotQAHandler) GetAdminStats(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	if !middleware.HasPermission(middleware.GetRole(c), middleware.PermissionUserRoleAssign) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}
	tenantID := middleware.GetTenantID(c)

	var spaceCount int64
	tenantDB.Model(&types.KnowledgeSpace{}).Where("tenant_id = ?", tenantID).Count(&spaceCount)

	var docCount int64
	tenantDB.Model(&types.SDPivotDocument{}).Where("tenant_id = ?", tenantID).Count(&docCount)

	var memberCount int64
	tenantDB.Model(&types.SDPivotOrgMember{}).
		Joins("JOIN org_ext ON org_ext.org_id = org_members.org_id").
		Where("org_ext.tenant_id = ?", tenantID).
		Count(&memberCount)

	c.JSON(http.StatusOK, gin.H{
		"space_count":    spaceCount,
		"document_count": docCount,
		"member_count":   memberCount,
	})
}

// ListAllSpaces lists all spaces in the tenant (admin view).
func (h *SDPivotQAHandler) ListAllSpaces(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	if !middleware.HasPermission(middleware.GetRole(c), middleware.PermissionUserRoleAssign) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}
	tenantID := middleware.GetTenantID(c)

	spaces, err := listKnowledgeSpaces(tenantDB, tenantID, middleware.GetUserID(c), true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list spaces"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"spaces": spaces})
}
