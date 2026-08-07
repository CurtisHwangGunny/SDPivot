package handler

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/middleware"
)

const maxSDPivotSearchHistory = 10

type sdpivotSearchHistoryEntry struct {
	Query      string    `json:"query"`
	SearchedAt time.Time `json:"searched_at"`
}

type sdpivotSearchResult struct {
	ChunkID       string `json:"chunk_id"`
	ChunkIndex    int    `json:"chunk_index"`
	Content       string `json:"content"`
	DocumentID    string `json:"document_id"`
	DocumentTitle string `json:"document_title"`
	SpaceID       string `json:"space_id"`
	SpaceName     string `json:"space_name"`
}

// SDPivotSearchHandler handles cross-space knowledge search.
type SDPivotSearchHandler struct {
	db      *gorm.DB
	history map[string][]sdpivotSearchHistoryEntry
	mu      sync.RWMutex
}

// NewSDPivotSearchHandler creates a global search handler.
func NewSDPivotSearchHandler(db *gorm.DB) *SDPivotSearchHandler {
	return &SDPivotSearchHandler{db: db, history: make(map[string][]sdpivotSearchHistoryEntry)}
}

// RegisterRoutes registers global search routes with RBAC guards.
func (h *SDPivotSearchHandler) RegisterRoutes(rg *gin.RouterGroup) {
	read := rg.Group("", middleware.RequirePermission(middleware.PermissionKnowledgeRead))
	read.GET("/search", h.Search)
	read.GET("/search/history", h.GetHistory)
	read.POST("/search/history", h.AddHistory)
	read.GET("/knowledge/search", h.Search)

	write := rg.Group("", middleware.RequirePermission(middleware.PermissionKnowledgeWrite))
	write.POST("/search/reindex", h.Reindex)
}

// Search performs an ILIKE search over chunks in spaces visible to the caller.
func (h *SDPivotSearchHandler) Search(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "q is required"})
		return
	}

	limit := 20
	if value, err := strconv.Atoi(c.Query("limit")); err == nil && value > 0 {
		limit = value
	}
	if limit > 100 {
		limit = 100
	}

	tenantDB := middleware.TenantDB(c, h.db)
	tenantID := middleware.GetTenantID(c)
	search := "%" + escapeILike(query) + "%"
	base := tenantDB.Table("document_chunks").
		Joins("JOIN documents ON documents.id = document_chunks.document_id AND documents.tenant_id = document_chunks.tenant_id").
		Joins("JOIN knowledge_spaces ON knowledge_spaces.id = documents.space_id AND knowledge_spaces.tenant_id = documents.tenant_id").
		Where("document_chunks.tenant_id = ? AND documents.deleted_at IS NULL AND knowledge_spaces.deleted_at IS NULL AND documents.parse_status = ?", tenantID, "completed").
		Where("documents.space_id IN (?)", visibleSpaceIDsQuery(tenantDB, tenantID, middleware.GetUserID(c))).
		Where("document_chunks.content ILIKE ?", search)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search knowledge"})
		return
	}

	var results []sdpivotSearchResult
	if err := base.Select(`
		document_chunks.id AS chunk_id,
		document_chunks.chunk_index,
		document_chunks.content,
		documents.id AS document_id,
		documents.title AS document_title,
		knowledge_spaces.id AS space_id,
		knowledge_spaces.name AS space_name`).
		Order("documents.updated_at DESC, document_chunks.chunk_index ASC").
		Limit(limit).
		Scan(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search knowledge"})
		return
	}

	h.remember(c, query)
	c.JSON(http.StatusOK, gin.H{"query": query, "results": results, "total": total})
}

// GetHistory returns the caller's recent search terms.
func (h *SDPivotSearchHandler) GetHistory(c *gin.Context) {
	h.mu.RLock()
	history := append([]sdpivotSearchHistoryEntry(nil), h.history[h.historyKey(c)]...)
	h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"history": history})
}

// AddHistory stores a search term for the current caller.
func (h *SDPivotSearchHandler) AddHistory(c *gin.Context) {
	var req struct {
		Query string `json:"query"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Query = strings.TrimSpace(req.Query)
	if req.Query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query is required"})
		return
	}
	h.remember(c, req.Query)
	c.JSON(http.StatusCreated, gin.H{"message": "search history saved"})
}

// Reindex acknowledges a reindex request until the parsing queue is wired here.
func (h *SDPivotSearchHandler) Reindex(c *gin.Context) {
	// TODO: enqueue visible documents for asynchronous re-parsing and chunk rebuilding.
	c.JSON(http.StatusAccepted, gin.H{"status": "accepted", "message": "reindex request accepted"})
}

func (h *SDPivotSearchHandler) remember(c *gin.Context, query string) {
	key := h.historyKey(c)
	entry := sdpivotSearchHistoryEntry{Query: query, SearchedAt: time.Now()}

	h.mu.Lock()
	defer h.mu.Unlock()
	existing := h.history[key]
	next := make([]sdpivotSearchHistoryEntry, 0, maxSDPivotSearchHistory)
	next = append(next, entry)
	for _, item := range existing {
		if strings.EqualFold(item.Query, query) {
			continue
		}
		next = append(next, item)
		if len(next) == maxSDPivotSearchHistory {
			break
		}
	}
	h.history[key] = next
}

func (h *SDPivotSearchHandler) historyKey(c *gin.Context) string {
	return strconv.FormatUint(middleware.GetTenantID(c), 10) + ":" + middleware.GetUserID(c)
}
