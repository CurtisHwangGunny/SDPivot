package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
)

// SmartKnoraDocumentHandler handles document upload, parsing, and management.
type SmartKnoraDocumentHandler struct {
	db        *gorm.DB
	uploadDir string
}

// NewSmartKnoraDocumentHandler creates a new document handler.
func NewSmartKnoraDocumentHandler(db *gorm.DB) *SmartKnoraDocumentHandler {
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "/tmp/smartknora-uploads"
	}
	os.MkdirAll(uploadDir, 0755)
	return &SmartKnoraDocumentHandler{db: db, uploadDir: uploadDir}
}

// RegisterRoutes registers document management routes.
func (h *SmartKnoraDocumentHandler) RegisterRoutes(rg *gin.RouterGroup) {
	docs := rg.Group("/documents")
	{
		docs.POST("/upload", h.UploadDocument)
		docs.POST("/manual", h.UploadManualDocument)
		docs.POST("/url", h.UploadFromURL)
		docs.GET("", h.ListDocuments)
		docs.GET("/:id", h.GetDocument)
		docs.GET("/:id/chunks", h.GetDocumentChunks)
		docs.GET("/:id/versions", h.GetDocumentVersions)
		docs.DELETE("/:id", h.DeleteDocument)
		docs.POST("/:id/reparse", h.ReparseDocument)
	}

	chunks := rg.Group("/chunks")
	{
		chunks.GET("/:id", h.GetChunk)
		chunks.PUT("/:id", h.UpdateChunk)
	}

	strategies := rg.Group("/chunk-strategies")
	{
		strategies.POST("", h.CreateStrategy)
		strategies.GET("", h.ListStrategies)
		strategies.PUT("/:id", h.UpdateStrategy)
		strategies.DELETE("/:id", h.DeleteStrategy)
	}

	search := rg.Group("/search")
	{
		search.POST("", h.SearchDocuments)
	}
}

// UploadDocument handles file upload.
func (h *SmartKnoraDocumentHandler) UploadDocument(c *gin.Context) {
	// Limit upload size to 50MB
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 50<<20)

	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)
	spaceID := c.PostForm("space_id")
	tags := c.PostForm("tags")

	if spaceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "space_id is required"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	// Calculate file hash
	hasher := sha256.New()
	content, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read file content"})
		return
	}
	hasher.Write(content)
	contentHash := hex.EncodeToString(hasher.Sum(nil))

	// Check duplicate
	var existing types.SmartKnoraDocument
	if err := h.db.Where("content_hash = ? AND space_id = ? AND deleted_at IS NULL", contentHash, spaceID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "document already exists", "document_id": existing.ID})
		return
	}

	// Save file
	docID := uuid.New().String()
	ext := filepath.Ext(header.Filename)
	savePath := filepath.Join(h.uploadDir, tenantIDStr(tenantID), docID+ext)
	os.MkdirAll(filepath.Dir(savePath), 0755)

	dst, err := os.Create(savePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}
	defer dst.Close()

	// Reset file reader
	if _, err := file.Seek(0, 0); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset file reader"})
		return
	}
	if _, err := io.Copy(dst, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}

	// Create document record
	now := time.Now()
	doc := types.SmartKnoraDocument{
		ID:              docID,
		TenantID:        tenantID,
		SpaceID:         spaceID,
		UploaderID:      userID,
		Title:           header.Filename,
		FileName:        header.Filename,
		FileType:        ext,
		FileSize:        header.Size,
		FilePath:        savePath,
		ContentHash:     contentHash,
		ParseStatus:     "pending",
		EmbeddingStatus: "pending",
		Version:         1,
		Tags:            tags,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := h.db.Create(&doc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create document record"})
		return
	}

	// Create initial version
	version := types.SmartKnoraDocumentVersion{
		ID:         uuid.New().String(),
		DocumentID: docID,
		Version:    1,
		FilePath:   savePath,
		FileSize:   header.Size,
		CreatedAt:  now,
		CreatedBy:  userID,
	}
	if err := h.db.Create(&version).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create document version"})
		return
	}

	parseMessage := "document uploaded and parsed"
	if err := h.parseAndStoreDocument(&doc, content); err != nil {
		parseMessage = "document uploaded, parsing failed: " + err.Error()
	}

	c.JSON(http.StatusCreated, gin.H{
		"document": doc,
		"message":  parseMessage,
	})
}

// ListDocuments lists documents in a knowledge space.
func (h *SmartKnoraDocumentHandler) ListDocuments(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	var query types.DocumentListQuery
	c.ShouldBindQuery(&query)

	db := h.db.Model(&types.SmartKnoraDocument{}).Where("tenant_id = ?", tenantID)

	if query.SpaceID != "" {
		db = db.Where("space_id = ?", query.SpaceID)
	}
	if query.ParseStatus != "" {
		db = db.Where("parse_status = ?", query.ParseStatus)
	}
	if query.Search != "" {
		search := strings.ReplaceAll(strings.ReplaceAll(query.Search, "%", "\\%"), "_", "\\_")
		db = db.Where("title ILIKE ?", "%"+search+"%")
	}

	page := query.Page
	if page < 1 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	db.Count(&total)

	var docs []types.SmartKnoraDocument
	db.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&docs)

	c.JSON(http.StatusOK, gin.H{
		"documents": docs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetDocument gets a specific document.
func (h *SmartKnoraDocumentHandler) GetDocument(c *gin.Context) {
	docID := c.Param("id")
	tenantID := middleware.GetTenantID(c)

	var doc types.SmartKnoraDocument
	if err := h.db.Where("id = ? AND tenant_id = ?", docID, tenantID).First(&doc).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"document": doc})
}

// GetDocumentChunks gets chunks of a document.
func (h *SmartKnoraDocumentHandler) GetDocumentChunks(c *gin.Context) {
	docID := c.Param("id")
	tenantID := middleware.GetTenantID(c)

	var chunks []types.SmartKnoraDocumentChunk
	h.db.Where("document_id = ? AND tenant_id = ?", docID, tenantID).Order("chunk_index").Find(&chunks)

	c.JSON(http.StatusOK, gin.H{"chunks": chunks, "total": len(chunks)})
}

// GetDocumentVersions gets version history of a document.
func (h *SmartKnoraDocumentHandler) GetDocumentVersions(c *gin.Context) {
	docID := c.Param("id")
	tenantID := middleware.GetTenantID(c)

	var versions []types.SmartKnoraDocumentVersion
	h.db.Where("document_id = ? AND tenant_id = ?", docID, tenantID).Order("version DESC").Find(&versions)

	c.JSON(http.StatusOK, gin.H{"versions": versions})
}

// DeleteDocument soft-deletes a document.
func (h *SmartKnoraDocumentHandler) DeleteDocument(c *gin.Context) {
	docID := c.Param("id")
	tenantID := middleware.GetTenantID(c)

	h.db.Where("id = ? AND tenant_id = ?", docID, tenantID).Delete(&types.SmartKnoraDocument{})
	h.db.Where("document_id = ? AND tenant_id = ?", docID, tenantID).Delete(&types.SmartKnoraDocumentChunk{})

	c.JSON(http.StatusOK, gin.H{"message": "document deleted"})
}

// ReparseDocument triggers re-parsing of a document.
func (h *SmartKnoraDocumentHandler) ReparseDocument(c *gin.Context) {
	docID := c.Param("id")
	tenantID := middleware.GetTenantID(c)

	var doc types.SmartKnoraDocument
	if err := h.db.Where("id = ? AND tenant_id = ?", docID, tenantID).First(&doc).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}

	content, err := h.loadDocumentContent(&doc)
	if err != nil {
		h.db.Model(&doc).Updates(map[string]interface{}{
			"parse_status": "failed",
			"updated_at":   time.Now(),
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.parseAndStoreDocument(&doc, content); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "re-parse completed"})
}

// GetChunk gets a specific chunk.
func (h *SmartKnoraDocumentHandler) GetChunk(c *gin.Context) {
	chunkID := c.Param("id")
	tenantID := middleware.GetTenantID(c)

	var chunk types.SmartKnoraDocumentChunk
	if err := h.db.Where("id = ? AND tenant_id = ?", chunkID, tenantID).First(&chunk).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "chunk not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"chunk": chunk})
}

// UpdateChunk updates a chunk's content.
func (h *SmartKnoraDocumentHandler) UpdateChunk(c *gin.Context) {
	chunkID := c.Param("id")
	tenantID := middleware.GetTenantID(c)

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.db.Model(&types.SmartKnoraDocumentChunk{}).Where("id = ? AND tenant_id = ?", chunkID, tenantID).Update("content", req.Content)

	c.JSON(http.StatusOK, gin.H{"message": "chunk updated"})
}

// CreateStrategy creates a chunking strategy.
func (h *SmartKnoraDocumentHandler) CreateStrategy(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	var req types.ChunkStrategyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ChunkSize == 0 {
		req.ChunkSize = 512
	}
	if req.ChunkOverlap == 0 {
		req.ChunkOverlap = 50
	}

	strategy := types.SmartKnoraChunkStrategy{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		SpaceID:      req.SpaceID,
		Name:         req.Name,
		StrategyType: req.StrategyType,
		ChunkSize:    req.ChunkSize,
		ChunkOverlap: req.ChunkOverlap,
		SplitMarkers: req.SplitMarkers,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	h.db.Create(&strategy)
	c.JSON(http.StatusCreated, gin.H{"strategy": strategy})
}

// ListStrategies lists chunking strategies.
func (h *SmartKnoraDocumentHandler) ListStrategies(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	var strategies []types.SmartKnoraChunkStrategy
	h.db.Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&strategies)

	c.JSON(http.StatusOK, gin.H{"strategies": strategies})
}

// UpdateStrategy updates a chunking strategy.
func (h *SmartKnoraDocumentHandler) UpdateStrategy(c *gin.Context) {
	strategyID := c.Param("id")

	var req types.ChunkStrategyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{"updated_at": time.Now()}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.ChunkSize > 0 {
		updates["chunk_size"] = req.ChunkSize
	}
	if req.ChunkOverlap > 0 {
		updates["chunk_overlap"] = req.ChunkOverlap
	}

	h.db.Model(&types.SmartKnoraChunkStrategy{}).Where("id = ?", strategyID).Updates(updates)
	c.JSON(http.StatusOK, gin.H{"message": "strategy updated"})
}

// DeleteStrategy deletes a chunking strategy.
func (h *SmartKnoraDocumentHandler) DeleteStrategy(c *gin.Context) {
	strategyID := c.Param("id")
	h.db.Where("id = ?", strategyID).Delete(&types.SmartKnoraChunkStrategy{})
	c.JSON(http.StatusOK, gin.H{"message": "strategy deleted"})
}

// SearchDocuments performs semantic search across documents.
func (h *SmartKnoraDocumentHandler) SearchDocuments(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	var req types.SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.TopK == 0 {
		req.TopK = 10
	}

	// Simple keyword search for now (vector search requires embedding service)
	search := strings.ReplaceAll(strings.ReplaceAll(req.Query, "%", "\\%"), "_", "\\_")
	db := h.db.Model(&types.SmartKnoraDocumentChunk{}).
		Where("tenant_id = ? AND content ILIKE ?", tenantID, "%"+search+"%")

	if req.SpaceID != "" {
		// Join with documents to filter by space
		db = db.Joins("JOIN documents ON documents.id = document_chunks.document_id").
			Where("documents.space_id = ?", req.SpaceID)
	}

	var chunks []types.SmartKnoraDocumentChunk
	db.Limit(req.TopK).Find(&chunks)

	results := make([]types.SmartKnoraSearchResult, len(chunks))
	for i, chunk := range chunks {
		results[i] = types.SmartKnoraSearchResult{
			DocumentID: chunk.DocumentID,
			ChunkID:    chunk.ID,
			Content:    chunk.Content,
			Score:      1.0, // Placeholder score
		}
	}

	c.JSON(http.StatusOK, gin.H{"results": results, "total": len(results)})
}

var htmlBlockRE = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
var htmlTagRE = regexp.MustCompile(`(?s)<[^>]+>`)

func (h *SmartKnoraDocumentHandler) loadDocumentContent(doc *types.SmartKnoraDocument) ([]byte, error) {
	if strings.HasPrefix(doc.FilePath, "http://") || strings.HasPrefix(doc.FilePath, "https://") {
		resp, err := http.Get(doc.FilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch document URL")
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("document URL returned non-200 status")
		}
		return io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	}
	if doc.FilePath == "" {
		return nil, fmt.Errorf("document has no file path")
	}
	return os.ReadFile(doc.FilePath)
}

func (h *SmartKnoraDocumentHandler) parseAndStoreDocument(doc *types.SmartKnoraDocument, content []byte) error {
	now := time.Now()
	h.db.Model(doc).Updates(map[string]interface{}{"parse_status": "parsing", "updated_at": now})

	text, err := normalizeDocumentContent(doc.FileType, content)
	if err != nil {
		h.db.Model(doc).Updates(map[string]interface{}{"parse_status": "failed", "chunk_count": 0, "updated_at": time.Now()})
		return err
	}
	chunks := splitDocumentText(text, 2000, 200)
	if len(chunks) == 0 {
		h.db.Model(doc).Updates(map[string]interface{}{"parse_status": "failed", "chunk_count": 0, "updated_at": time.Now()})
		return fmt.Errorf("document content is empty")
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("document_id = ? AND tenant_id = ?", doc.ID, doc.TenantID).Delete(&types.SmartKnoraDocumentChunk{}).Error; err != nil {
			return err
		}
		for i, part := range chunks {
			chunk := types.SmartKnoraDocumentChunk{ID: uuid.New().String(), DocumentID: doc.ID, TenantID: doc.TenantID, ChunkIndex: i, Content: part, TokenCount: estimateTokenCount(part), Metadata: "{}", CreatedAt: now}
			if err := tx.Create(&chunk).Error; err != nil {
				return err
			}
		}
		return tx.Model(doc).Updates(map[string]interface{}{"parse_status": "completed", "chunk_count": len(chunks), "updated_at": time.Now()}).Error
	})
	if err != nil {
		h.db.Model(doc).Updates(map[string]interface{}{"parse_status": "failed", "updated_at": time.Now()})
		return fmt.Errorf("failed to store chunks")
	}
	return nil
}

func normalizeDocumentContent(fileType string, content []byte) (string, error) {
	ext := strings.ToLower(strings.TrimPrefix(fileType, "."))
	text := string(content)
	switch ext {
	case "md", "markdown", "txt", "text":
		return normalizeWhitespace(text), nil
	case "html", "htm":
		return normalizeWhitespace(stripHTMLTags(text)), nil
	default:
		return "", fmt.Errorf("unsupported file type for fallback parser: %s", fileType)
	}
}

func stripHTMLTags(input string) string {
	withoutBlocks := htmlBlockRE.ReplaceAllString(input, " ")
	withoutTags := htmlTagRE.ReplaceAllString(withoutBlocks, " ")
	return html.UnescapeString(withoutTags)
}

func normalizeWhitespace(input string) string { return strings.Join(strings.Fields(input), " ") }

func splitDocumentText(text string, chunkSize int, overlap int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	runes := []rune(text)
	if chunkSize <= 0 {
		chunkSize = 2000
	}
	if overlap < 0 || overlap >= chunkSize {
		overlap = 0
	}
	chunks := make([]string, 0, (len(runes)/chunkSize)+1)
	for start := 0; start < len(runes); {
		end := start + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		part := strings.TrimSpace(string(runes[start:end]))
		if part != "" {
			chunks = append(chunks, part)
		}
		if end == len(runes) {
			break
		}
		start = end - overlap
	}
	return chunks
}

func estimateTokenCount(text string) int {
	count := len([]rune(text)) / 4
	if count < 1 {
		return 1
	}
	return count
}

func escapeILike(input string) string {
	value := strings.ReplaceAll(input, `\`, `\\`)
	value = strings.ReplaceAll(value, "%", `\%`)
	value = strings.ReplaceAll(value, "_", `\_`)
	return value
}

func tenantIDStr(tenantID uint64) string {
	return strconv.FormatUint(tenantID, 10)
}

// UploadManualDocument handles manual text/markdown input.
func (h *SmartKnoraDocumentHandler) UploadManualDocument(c *gin.Context) {
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	var req struct {
		SpaceID string `json:"space_id" binding:"required"`
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
		Tags    string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	docID := uuid.New().String()
	now := time.Now()

	// Calculate content hash
	hasher := sha256.New()
	hasher.Write([]byte(req.Content))
	contentHash := hex.EncodeToString(hasher.Sum(nil))

	doc := types.SmartKnoraDocument{
		ID:              docID,
		TenantID:        tenantID,
		SpaceID:         req.SpaceID,
		UploaderID:      userID,
		Title:           req.Title,
		FileName:        req.Title + ".md",
		FileType:        ".md",
		FileSize:        int64(len(req.Content)),
		FilePath:        "",
		ContentHash:     contentHash,
		ParseStatus:     "completed",
		EmbeddingStatus: "pending",
		Version:         1,
		Tags:            req.Tags,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := h.db.Create(&doc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create document"})
		return
	}

	if err := h.parseAndStoreDocument(&doc, []byte(req.Content)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse manual document"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"document": doc,
		"message":  "manual document created and parsed",
	})
}

// UploadFromURL handles web page URL import.
func (h *SmartKnoraDocumentHandler) UploadFromURL(c *gin.Context) {
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	var req struct {
		SpaceID string `json:"space_id" binding:"required"`
		URL     string `json:"url" binding:"required"`
		Tags    string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch URL content
	resp, err := http.Get(req.URL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to fetch URL"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL returned non-200 status"})
		return
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10MB limit
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read URL content"})
		return
	}

	content := normalizeWhitespace(stripHTMLTags(string(body)))

	docID := uuid.New().String()
	now := time.Now()

	hasher := sha256.New()
	hasher.Write(body)
	contentHash := hex.EncodeToString(hasher.Sum(nil))

	// Extract domain for title
	domain := req.URL
	if idx := strings.Index(domain, "://"); idx >= 0 {
		domain = domain[idx+3:]
	}
	if idx := strings.Index(domain, "/"); idx >= 0 {
		domain = domain[:idx]
	}

	doc := types.SmartKnoraDocument{
		ID:              docID,
		TenantID:        tenantID,
		SpaceID:         req.SpaceID,
		UploaderID:      userID,
		Title:           "网页导入 - " + domain,
		FileName:        domain + ".html",
		FileType:        ".html",
		FileSize:        int64(len(body)),
		FilePath:        req.URL,
		ContentHash:     contentHash,
		ParseStatus:     "parsing",
		EmbeddingStatus: "pending",
		Version:         1,
		Tags:            req.Tags,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := h.db.Create(&doc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create document"})
		return
	}

	if err := h.parseAndStoreDocument(&doc, []byte(content)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse URL content"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"document": doc,
		"message":  "URL imported and parsed",
	})
}
