package handler

import (
	"encoding/json"
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

// SmartKnoraQAHandler handles AI Q&A sessions.
type SmartKnoraQAHandler struct {
	db *gorm.DB
}

// NewSmartKnoraQAHandler creates a new QA handler.
func NewSmartKnoraQAHandler(db *gorm.DB) *SmartKnoraQAHandler {
	return &SmartKnoraQAHandler{db: db}
}

// RegisterRoutes registers Q&A routes.
func (h *SmartKnoraQAHandler) RegisterRoutes(rg *gin.RouterGroup) {
	q := rg.Group("/qa")
	{
		q.POST("/sessions", h.CreateSession)
		q.GET("/sessions", h.ListSessions)
		q.GET("/sessions/:id", h.GetSession)
		q.POST("/sessions/:id/messages", h.SendMessage)
		q.GET("/sessions/:id/messages", h.GetMessages)
		q.DELETE("/sessions/:id", h.DeleteSession)
	}

	admin := rg.Group("/admin")
	{
		admin.GET("/members", h.ListAllMembers)
		admin.GET("/stats", h.GetAdminStats)
		admin.GET("/spaces", h.ListAllSpaces)
	}
}

// CreateSession creates a new Q&A session.
func (h *SmartKnoraQAHandler) CreateSession(c *gin.Context) {
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

	session := types.QASession{
		ID:        uuid.New().String(),
		UserID:    userID,
		TenantID:  tenantID,
		SpaceID:   req.SpaceID,
		Title:     req.Title,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	h.db.Create(&session)
	c.JSON(http.StatusCreated, gin.H{"session": session})
}

// ListSessions lists Q&A sessions for the current user.
func (h *SmartKnoraQAHandler) ListSessions(c *gin.Context) {
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	var sessions []types.QASession
	h.db.Where("user_id = ? AND tenant_id = ?", userID, tenantID).
		Order("updated_at DESC").Limit(50).Find(&sessions)

	c.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

// GetSession gets a specific Q&A session.
func (h *SmartKnoraQAHandler) GetSession(c *gin.Context) {
	sessionID := c.Param("id")
	userID := middleware.GetUserID(c)

	var session types.QASession
	if err := h.db.Where("id = ? AND user_id = ? AND tenant_id = ?", sessionID, userID, middleware.GetTenantID(c)).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"session": session})
}

// SendMessage sends a message in a Q&A session.
func (h *SmartKnoraQAHandler) SendMessage(c *gin.Context) {
	sessionID := c.Param("id")
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	var session types.QASession
	if err := h.db.Where("id = ? AND user_id = ? AND tenant_id = ?", sessionID, userID, tenantID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	now := time.Now()
	userMsg := types.QAMessage{ID: uuid.New().String(), SessionID: sessionID, Role: "user", Content: req.Content, CreatedAt: now}
	h.db.Create(&userMsg)

	chunks, err := h.searchRelevantChunks(tenantID, session.SpaceID, req.Content, 5)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search knowledge base"})
		return
	}

	sources := "[]"
	aiContent := "暂未找到相关知识内容。请先在知识空间中导入文档。"
	if len(chunks) > 0 {
		aiContent = "根据知识库中的内容，为您找到以下相关信息：\n\n"
		sourceSet := map[string]struct{}{}
		sourceList := []string{}
		for i, chunk := range chunks {
			aiContent += fmt.Sprintf("%d. %s\n\n", i+1, chunk.Content)
			if _, ok := sourceSet[chunk.DocumentID]; !ok {
				sourceSet[chunk.DocumentID] = struct{}{}
				sourceList = append(sourceList, chunk.DocumentID)
			}
		}
		sourcesBytes, _ := json.Marshal(sourceList)
		sources = string(sourcesBytes)
	}

	aiMsg := types.QAMessage{ID: uuid.New().String(), SessionID: sessionID, Role: "assistant", Content: aiContent, Sources: sources, CreatedAt: now}
	h.db.Create(&aiMsg)
	h.db.Model(&types.QASession{}).Where("id = ? AND tenant_id = ?", sessionID, tenantID).Update("updated_at", now)
	c.JSON(http.StatusOK, gin.H{"user_message": userMsg, "assistant_message": aiMsg})
}

func (h *SmartKnoraQAHandler) searchRelevantChunks(tenantID uint64, spaceID string, query string, topK int) ([]types.SmartKnoraDocumentChunk, error) {
	if topK <= 0 || topK > 20 {
		topK = 5
	}
	search := escapeQAQuery(query)
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

func escapeQAQuery(input string) string {
	value := strings.ReplaceAll(input, `\`, `\\`)
	value = strings.ReplaceAll(value, "%", `\%`)
	value = strings.ReplaceAll(value, "_", `\_`)
	return value
}

// GetMessages gets messages in a Q&A session.
func (h *SmartKnoraQAHandler) GetMessages(c *gin.Context) {
	sessionID := c.Param("id")
	userID := middleware.GetUserID(c)

	// Verify session belongs to user
	var session types.QASession
	if err := h.db.Where("id = ? AND user_id = ? AND tenant_id = ?", sessionID, userID, middleware.GetTenantID(c)).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	var messages []types.QAMessage
	h.db.Where("session_id = ?", sessionID).Order("created_at ASC").Find(&messages)

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

// DeleteSession deletes a Q&A session.
func (h *SmartKnoraQAHandler) DeleteSession(c *gin.Context) {
	sessionID := c.Param("id")
	userID := middleware.GetUserID(c)

	// Verify session belongs to user
	var session types.QASession
	if err := h.db.Where("id = ? AND user_id = ? AND tenant_id = ?", sessionID, userID, middleware.GetTenantID(c)).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	h.db.Where("session_id = ?", sessionID).Delete(&types.QAMessage{})
	h.db.Where("id = ? AND user_id = ? AND tenant_id = ?", sessionID, userID, middleware.GetTenantID(c)).Delete(&types.QASession{})

	c.JSON(http.StatusOK, gin.H{"message": "session deleted"})
}

// ListAllMembers lists all members across organizations (admin view).
func (h *SmartKnoraQAHandler) ListAllMembers(c *gin.Context) {
	role, _ := c.Get("role")
	if role != "admin" && role != "owner" {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}
	tenantID := middleware.GetTenantID(c)

	var members []types.SmartKnoraOrgMember
	h.db.Joins("JOIN org_ext ON org_ext.org_id = org_members.org_id").
		Where("org_ext.tenant_id = ?", tenantID).
		Find(&members)

	c.JSON(http.StatusOK, gin.H{"members": members})
}

// GetAdminStats returns admin dashboard statistics.
func (h *SmartKnoraQAHandler) GetAdminStats(c *gin.Context) {
	role, _ := c.Get("role")
	if role != "admin" && role != "owner" {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}
	tenantID := middleware.GetTenantID(c)

	var spaceCount int64
	h.db.Model(&types.KnowledgeSpace{}).Where("tenant_id = ?", tenantID).Count(&spaceCount)

	var docCount int64
	h.db.Model(&types.SmartKnoraDocument{}).Where("tenant_id = ?", tenantID).Count(&docCount)

	var memberCount int64
	h.db.Model(&types.SmartKnoraOrgMember{}).
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
func (h *SmartKnoraQAHandler) ListAllSpaces(c *gin.Context) {
	role, _ := c.Get("role")
	if role != "admin" && role != "owner" {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}
	tenantID := middleware.GetTenantID(c)

	var spaces []types.KnowledgeSpace
	h.db.Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&spaces)

	c.JSON(http.StatusOK, gin.H{"spaces": spaces})
}
