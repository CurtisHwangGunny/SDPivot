package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// SDPivotDocumentHandler handles document upload, parsing, and management.
type SDPivotDocumentHandler struct {
	db             *gorm.DB
	uploadDir      string
	documentReader interfaces.DocReader
}

// NewSDPivotDocumentHandler creates a new document handler.
func NewSDPivotDocumentHandler(db *gorm.DB, documentReader interfaces.DocReader) *SDPivotDocumentHandler {
	uploadDir := os.Getenv("SDP_UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = os.Getenv("UPLOAD_DIR")
	}
	if uploadDir == "" {
		uploadDir = "/tmp/sdpivot-uploads"
	}
	os.MkdirAll(uploadDir, 0755)
	return &SDPivotDocumentHandler{db: db, uploadDir: uploadDir, documentReader: documentReader}
}

// RegisterRoutes registers document management routes.
func (h *SDPivotDocumentHandler) RegisterRoutes(rg *gin.RouterGroup) {
	docs := rg.Group("/documents")
	{
		docs.POST("/upload", h.UploadDocument)
		docs.POST("/manual", h.UploadManualDocument)
		docs.POST("/import-url", h.UploadFromURL)
		docs.POST("/url", h.UploadFromURL)
		docs.GET("", h.ListDocuments)
		docs.GET("/:id", h.GetDocument)
		docs.GET("/:id/parse-status", h.GetDocumentParseStatus)
		docs.GET("/:id/chunks", h.GetDocumentChunks)
		docs.GET("/:id/versions", h.GetDocumentVersions)
		docs.DELETE("/:id", h.DeleteDocument)
		docs.POST("/:id/reparse", h.ReparseDocument)
	}
	thirdPartyDocs := rg.Group("/sdpivot/documents")
	thirdPartyDocs.GET("/:id/tags", h.GetDocumentTags)
	thirdPartyDocs.PUT("/:id/tags", middleware.RequirePermission(middleware.PermissionDepartmentManage), h.SyncDocumentTags)
	thirdPartyDocumentWrites := thirdPartyDocs.Group("", middleware.RequirePermission(middleware.PermissionKnowledgeWrite))
	thirdPartyDocumentWrites.POST("/import", h.ImportDocument)
	thirdPartyDocumentWrites.POST("/batch", h.BatchImportDocuments)

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
func (h *SDPivotDocumentHandler) UploadDocument(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	// Limit upload size to 50MB
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 50<<20)
	if !middleware.HasContextPermission(c, middleware.PermissionKnowledgeWrite) {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permission", "permission": "knowledge.write"})
		return
	}

	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)
	spaceID := c.PostForm("space_id")
	if spaceID == "" {
		spaceID = c.Param("spaceId")
	}
	if spaceID == "" {
		spaceID = c.Param("space_id")
	}
	tags := c.PostForm("tags")

	if spaceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "space_id is required"})
		return
	}
	if _, ok := authorizeSpace(c, tenantDB, spaceID, spaceAccessEdit); !ok {
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
	var existing types.SDPivotDocument
	if err := tenantDB.Where("content_hash = ? AND space_id = ? AND deleted_at IS NULL", contentHash, spaceID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "document already exists", "document_id": existing.ID})
		return
	}

	// Save file
	docID := uuid.New().String()
	ext := filepath.Ext(header.Filename)
	savePath := filepath.Join(h.uploadDir, tenantIDStr(tenantID), docID+ext)
	if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		savePath = filepath.Join(os.TempDir(), "sdpivot-uploads", tenantIDStr(tenantID), docID+ext)
		if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to prepare upload directory", "detail": err.Error()})
			return
		}
	}

	dst, err := os.Create(savePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}
	fileSaved := true
	defer func() {
		_ = dst.Close()
		if fileSaved {
			_ = os.Remove(savePath)
		}
	}()

	if _, err := file.Seek(0, 0); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset file reader"})
		return
	}
	if _, err := io.Copy(dst, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}
	if err := dst.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}

	now := time.Now()
	doc := types.SDPivotDocument{
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
	version := types.SDPivotDocumentVersion{
		ID:         uuid.New().String(),
		DocumentID: docID,
		TenantID:   tenantID,
		Version:    1,
		FilePath:   savePath,
		FileSize:   header.Size,
		CreatedAt:  now,
		CreatedBy:  userID,
	}
	if err := createSDPivotDocumentWithVersion(tenantDB, &doc, &version); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create document record and version", "detail": err.Error()})
		return
	}

	fileSaved = false
	parseMessage := "document uploaded and parsed"
	if err := h.parseAndStoreDocument(c.Request.Context(), tenantDB, &doc, content); err != nil {
		parseMessage = "document uploaded, parsing failed: " + err.Error()
	}
	writeSDPivotAuditLog(h.db, c, auditActionDocumentUpload, auditModuleDocument, "document", doc.ID, map[string]interface{}{
		"space_id": doc.SpaceID, "title": doc.Title, "file_size": doc.FileSize,
	})

	c.JSON(http.StatusCreated, gin.H{
		"document": doc,
		"message":  parseMessage,
	})
}

// ListDocuments lists documents in a knowledge space.
func (h *SDPivotDocumentHandler) ListDocuments(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	tenantID := middleware.GetTenantID(c)

	var query types.DocumentListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if query.SpaceID == "" {
		query.SpaceID = c.Param("spaceId")
	}
	if query.SpaceID == "" {
		query.SpaceID = c.Param("space_id")
	}

	db := tenantDB.Model(&types.SDPivotDocument{}).Where("tenant_id = ?", tenantID)

	if query.SpaceID != "" {
		if _, ok := authorizeSpace(c, tenantDB, query.SpaceID, spaceAccessView); !ok {
			return
		}
		db = db.Where("space_id = ?", query.SpaceID)
	} else {
		db = db.Where("space_id IN (?)", visibleSpaceIDsQuery(tenantDB, tenantID, middleware.GetUserID(c)))
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

	var docs []types.SDPivotDocument
	db.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&docs)

	c.JSON(http.StatusOK, gin.H{
		"documents": docs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetDocument gets a specific document.
func (h *SDPivotDocumentHandler) GetDocument(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	docID := c.Param("id")
	tenantID := middleware.GetTenantID(c)

	var doc types.SDPivotDocument
	if err := tenantDB.Where("id = ? AND tenant_id = ?", docID, tenantID).First(&doc).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}

	if _, ok := authorizeSpace(c, tenantDB, doc.SpaceID, spaceAccessView); !ok {
		return
	}

	c.JSON(http.StatusOK, gin.H{"document": doc})
}

// GetDocumentParseStatus returns a stable parsing progress projection.
func (h *SDPivotDocumentHandler) GetDocumentParseStatus(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	doc, ok := h.authorizeDocument(c, tenantDB, c.Param("id"), spaceAccessView)
	if !ok {
		return
	}

	var storedChunks int64
	if err := tenantDB.Model(&types.SDPivotDocumentChunk{}).
		Where("tenant_id = ? AND document_id = ?", doc.TenantID, doc.ID).
		Count(&storedChunks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load document parse status"})
		return
	}

	status := strings.ToLower(strings.TrimSpace(doc.ParseStatus))
	progress := 0
	errorMessage := ""
	switch status {
	case "completed":
		progress = 100
	case "parsing", "processing":
		status = "parsing"
		progress = 10
		if doc.ChunkCount > 0 {
			progress = 10 + int(storedChunks*80/int64(doc.ChunkCount))
			if progress > 90 {
				progress = 90
			}
		}
	case "failed":
		errorMessage = "document parsing failed"
	case "pending", "":
		status = "pending"
	default:
		status = "pending"
	}

	c.JSON(http.StatusOK, gin.H{"progress": progress, "status": status, "error": errorMessage})
}

// GetDocumentChunks gets chunks of a document.
func (h *SDPivotDocumentHandler) GetDocumentChunks(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	docID := c.Param("id")
	tenantID := middleware.GetTenantID(c)

	doc, ok := h.authorizeDocument(c, tenantDB, docID, spaceAccessView)
	if !ok {
		return
	}

	var chunks []types.SDPivotDocumentChunk
	if err := tenantDB.Where("document_id = ? AND tenant_id = ?", doc.ID, tenantID).Order("chunk_index").Find(&chunks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list document chunks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"chunks": chunks, "total": len(chunks)})
}

// GetDocumentVersions gets version history of a document.
func (h *SDPivotDocumentHandler) GetDocumentVersions(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	docID := c.Param("id")
	tenantID := middleware.GetTenantID(c)

	doc, ok := h.authorizeDocument(c, tenantDB, docID, spaceAccessView)
	if !ok {
		return
	}

	var versions []types.SDPivotDocumentVersion
	if err := tenantDB.Where("document_id = ? AND tenant_id = ?", doc.ID, tenantID).Order("version DESC").Find(&versions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list document versions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"versions": versions})
}

// DeleteDocument soft-deletes a document.
func (h *SDPivotDocumentHandler) DeleteDocument(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	if !middleware.HasContextPermission(c, middleware.PermissionKnowledgeWrite) {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permission", "permission": "knowledge.write"})
		return
	}
	docID := c.Param("id")
	tenantID := middleware.GetTenantID(c)

	doc, ok := h.authorizeDocument(c, tenantDB, docID, spaceAccessEdit)
	if !ok {
		return
	}
	if err := tenantDB.Where("id = ? AND tenant_id = ?", doc.ID, tenantID).Delete(&types.SDPivotDocument{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete document"})
		return
	}
	if err := tenantDB.Where("document_id = ? AND tenant_id = ?", doc.ID, tenantID).Delete(&types.SDPivotDocumentChunk{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete document chunks"})
		return
	}
	writeSDPivotAuditLog(h.db, c, auditActionDocumentDelete, auditModuleDocument, "document", doc.ID, map[string]interface{}{
		"space_id": doc.SpaceID, "title": doc.Title,
	})

	c.JSON(http.StatusOK, gin.H{"message": "document deleted"})
}

// ReparseDocument triggers re-parsing of a document.
func (h *SDPivotDocumentHandler) ReparseDocument(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	if !middleware.HasContextPermission(c, middleware.PermissionKnowledgeWrite) {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permission", "permission": "knowledge.write"})
		return
	}
	docID := c.Param("id")

	doc, ok := h.authorizeDocument(c, tenantDB, docID, spaceAccessEdit)
	if !ok {
		return
	}

	var content []byte
	var err error
	if doc.FilePath == "" {
		var chunks []types.SDPivotDocumentChunk
		err = tenantDB.Where("document_id = ? AND tenant_id = ?", doc.ID, doc.TenantID).
			Order("chunk_index").
			Find(&chunks).Error
		if err == nil {
			parts := make([]string, 0, len(chunks))
			for _, chunk := range chunks {
				parts = append(parts, chunk.Content)
			}
			content = []byte(strings.Join(parts, "\n"))
		}
	} else {
		content, err = h.loadDocumentContent(c.Request.Context(), doc)
	}
	if err != nil {
		tenantDB.Model(doc).Updates(map[string]interface{}{
			"parse_status": "failed",
			"updated_at":   time.Now(),
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.parseAndStoreDocument(c.Request.Context(), tenantDB, doc, content); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "re-parse completed"})
}

// GetChunk gets a specific chunk.
func (h *SDPivotDocumentHandler) GetChunk(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	chunkID := c.Param("id")

	chunk, ok := h.authorizeChunk(c, tenantDB, chunkID, spaceAccessView)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, gin.H{"chunk": chunk})
}

// UpdateChunk updates a chunk's content.
func (h *SDPivotDocumentHandler) UpdateChunk(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	if !middleware.HasContextPermission(c, middleware.PermissionKnowledgeWrite) {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permission", "permission": "knowledge.write"})
		return
	}
	chunkID := c.Param("id")
	tenantID := middleware.GetTenantID(c)

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	chunk, ok := h.authorizeChunk(c, tenantDB, chunkID, spaceAccessEdit)
	if !ok {
		return
	}
	if err := tenantDB.Model(&types.SDPivotDocumentChunk{}).Where("id = ? AND tenant_id = ?", chunk.ID, tenantID).Update("content", req.Content).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update chunk"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "chunk updated"})
}

// CreateStrategy creates a chunking strategy.
func (h *SDPivotDocumentHandler) CreateStrategy(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
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

	strategy := types.SDPivotChunkStrategy{
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

	tenantDB.Create(&strategy)
	c.JSON(http.StatusCreated, gin.H{"strategy": strategy})
}

// ListStrategies lists chunking strategies.
func (h *SDPivotDocumentHandler) ListStrategies(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	tenantID := middleware.GetTenantID(c)

	var strategies []types.SDPivotChunkStrategy
	tenantDB.Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&strategies)

	c.JSON(http.StatusOK, gin.H{"strategies": strategies})
}

// UpdateStrategy updates a chunking strategy.
func (h *SDPivotDocumentHandler) UpdateStrategy(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
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

	tenantDB.Model(&types.SDPivotChunkStrategy{}).Where("id = ?", strategyID).Updates(updates)
	c.JSON(http.StatusOK, gin.H{"message": "strategy updated"})
}

// DeleteStrategy deletes a chunking strategy.
func (h *SDPivotDocumentHandler) DeleteStrategy(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	strategyID := c.Param("id")
	tenantDB.Where("id = ?", strategyID).Delete(&types.SDPivotChunkStrategy{})
	c.JSON(http.StatusOK, gin.H{"message": "strategy deleted"})
}

// SearchDocuments performs semantic search across documents.
func (h *SDPivotDocumentHandler) SearchDocuments(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	tenantID := middleware.GetTenantID(c)

	var req types.SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.TopK == 0 {
		req.TopK = 10
	}

	search := strings.ReplaceAll(strings.ReplaceAll(req.Query, "%", "\\%"), "_", "\\_")
	db := tenantDB.Model(&types.SDPivotDocumentChunk{}).
		Joins("JOIN documents ON documents.id = document_chunks.document_id AND documents.tenant_id = document_chunks.tenant_id").
		Where("document_chunks.tenant_id = ? AND documents.deleted_at IS NULL AND documents.parse_status = ? AND document_chunks.content ILIKE ?", tenantID, "completed", "%"+search+"%")

	if req.SpaceID != "" {
		if _, ok := authorizeSpace(c, tenantDB, req.SpaceID, spaceAccessView); !ok {
			return
		}
		db = db.Where("documents.space_id = ?", req.SpaceID)
	} else {
		db = db.Where("documents.space_id IN (?)", visibleSpaceIDsQuery(tenantDB, tenantID, middleware.GetUserID(c)))
	}

	var chunks []types.SDPivotDocumentChunk
	if err := db.Order("document_chunks.created_at DESC").Limit(req.TopK).Find(&chunks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search documents", "detail": err.Error()})
		return
	}

	results := make([]types.SDPivotSearchResult, len(chunks))
	for i, chunk := range chunks {
		results[i] = types.SDPivotSearchResult{
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

func (h *SDPivotDocumentHandler) loadDocumentContent(ctx context.Context, doc *types.SDPivotDocument) ([]byte, error) {
	if strings.HasPrefix(doc.FilePath, "http://") || strings.HasPrefix(doc.FilePath, "https://") {
		content, _, _, err := fetchSDPivotURLDocument(ctx, doc.FilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch document URL")
		}
		return content, nil
	}
	if doc.FilePath == "" {
		return nil, fmt.Errorf("document has no file path")
	}
	return os.ReadFile(doc.FilePath)
}

func (h *SDPivotDocumentHandler) parseAndStoreDocument(ctx context.Context, tenantDB *gorm.DB, doc *types.SDPivotDocument, content []byte) error {
	now := time.Now()
	tenantDB.Model(doc).Updates(map[string]interface{}{"parse_status": "parsing", "updated_at": now})

	text, err := h.parseDocumentContent(ctx, doc, content)
	if err != nil {
		tenantDB.Model(doc).Updates(map[string]interface{}{"parse_status": "failed", "chunk_count": 0, "updated_at": time.Now()})
		return err
	}
	chunks := splitDocumentText(text, 2000, 200)
	if len(chunks) == 0 {
		tenantDB.Model(doc).Updates(map[string]interface{}{"parse_status": "failed", "chunk_count": 0, "updated_at": time.Now()})
		return fmt.Errorf("document content is empty")
	}

	err = tenantDB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("document_id = ? AND tenant_id = ?", doc.ID, doc.TenantID).Delete(&types.SDPivotDocumentChunk{}).Error; err != nil {
			return err
		}
		for i, part := range chunks {
			chunk := types.SDPivotDocumentChunk{ID: uuid.New().String(), DocumentID: doc.ID, TenantID: doc.TenantID, ChunkIndex: i, Content: part, TokenCount: estimateTokenCount(part), Metadata: "{}", CreatedAt: now}
			if err := tx.Create(&chunk).Error; err != nil {
				return err
			}
		}
		return tx.Model(doc).Updates(map[string]interface{}{"parse_status": "completed", "chunk_count": len(chunks), "updated_at": time.Now()}).Error
	})
	if err != nil {
		tenantDB.Model(doc).Updates(map[string]interface{}{"parse_status": "failed", "updated_at": time.Now()})
		return fmt.Errorf("failed to store chunks")
	}
	return nil
}

func (h *SDPivotDocumentHandler) parseDocumentContent(ctx context.Context, doc *types.SDPivotDocument, content []byte) (string, error) {
	ext := strings.ToLower(strings.TrimPrefix(doc.FileType, "."))
	switch ext {
	case "doc", "docx", "ppt", "pptx", "xls", "xlsx", "pdf":
		if h.documentReader == nil {
			return "", fmt.Errorf("document parser is unavailable for file type: %s", doc.FileType)
		}
		result, err := h.documentReader.Read(ctx, &types.ReadRequest{
			FileContent: content,
			FileName:    doc.FileName,
			FileType:    ext,
			Title:       doc.Title,
			RequestID:   doc.ID,
		})
		if err != nil {
			return "", fmt.Errorf("document parser failed for %s: %w", doc.FileType, err)
		}
		if result == nil {
			return "", fmt.Errorf("document parser returned no result for file type: %s", doc.FileType)
		}
		if result.Error != "" {
			return "", fmt.Errorf("document parser failed for %s: %s", doc.FileType, result.Error)
		}
		text := strings.TrimSpace(result.MarkdownContent)
		if text == "" {
			return "", fmt.Errorf("document parser returned empty content for file type: %s", doc.FileType)
		}
		return text, nil
	default:
		return normalizeDocumentContent(doc.FileType, content)
	}
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

func createSDPivotDocumentWithVersion(tenantDB *gorm.DB, doc *types.SDPivotDocument, version *types.SDPivotDocumentVersion) error {
	return tenantDB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(doc).Error; err != nil {
			return fmt.Errorf("create document: %w", err)
		}
		if err := tx.Create(version).Error; err != nil {
			return fmt.Errorf("create document version: %w", err)
		}
		return nil
	})
}

// UploadManualDocument handles manual text/markdown input.
func (h *SDPivotDocumentHandler) UploadManualDocument(c *gin.Context) {
	if !middleware.HasContextPermission(c, middleware.PermissionKnowledgeWrite) {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permission", "permission": "knowledge.write"})
		return
	}
	var req sdpivotManualDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	doc, status, err := h.createManualDocument(c, req)
	if err != nil {
		if status == 0 {
			return
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"document": doc,
		"message":  "manual document created and parsed",
	})
}

func (h *SDPivotDocumentHandler) ImportDocument(c *gin.Context) {
	var req sdpivotManualDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	doc, status, err := h.createManualDocument(c, req)
	if err != nil {
		if status == 0 {
			return
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"document": doc, "status": "completed", "message": "document imported and parsed"})
}

type sdpivotManualDocumentRequest struct {
	SpaceID string `json:"space_id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Tags    string `json:"tags"`
}

func (h *SDPivotDocumentHandler) createManualDocument(c *gin.Context, req sdpivotManualDocumentRequest) (*types.SDPivotDocument, int, error) {
	tenantDB := middleware.TenantDB(c, h.db)
	req.SpaceID = strings.TrimSpace(req.SpaceID)
	req.Title = strings.TrimSpace(req.Title)
	if req.SpaceID == "" || req.Title == "" || strings.TrimSpace(req.Content) == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("space_id, title and content are required")
	}
	if _, ok := authorizeSpace(c, tenantDB, req.SpaceID, spaceAccessEdit); !ok {
		return nil, 0, fmt.Errorf("space access denied")
	}
	hasher := sha256.Sum256([]byte(req.Content))
	now := time.Now()
	doc := &types.SDPivotDocument{
		ID: uuid.NewString(), TenantID: middleware.GetTenantID(c), SpaceID: req.SpaceID, UploaderID: middleware.GetUserID(c),
		Title: req.Title, FileName: req.Title + ".md", FileType: ".md", FileSize: int64(len(req.Content)),
		ContentHash: hex.EncodeToString(hasher[:]), ParseStatus: "completed", EmbeddingStatus: "pending",
		Version: 1, Tags: req.Tags, CreatedAt: now, UpdatedAt: now,
	}
	version := types.SDPivotDocumentVersion{ID: uuid.NewString(), DocumentID: doc.ID, TenantID: doc.TenantID, Version: 1, FileSize: doc.FileSize, CreatedAt: now, CreatedBy: doc.UploaderID}
	if err := createSDPivotDocumentWithVersion(tenantDB, doc, &version); err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to create document and version: %w", err)
	}
	if err := h.parseAndStoreDocument(c.Request.Context(), tenantDB, doc, []byte(req.Content)); err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to parse manual document: %w", err)
	}
	return doc, http.StatusCreated, nil
}

func (h *SDPivotDocumentHandler) BatchImportDocuments(c *gin.Context) {
	var raw json.RawMessage
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document batch"})
		return
	}
	var documents []sdpivotManualDocumentRequest
	if err := json.Unmarshal(raw, &documents); err != nil {
		var payload struct {
			Documents []sdpivotManualDocumentRequest `json:"documents"`
		}
		if err := json.Unmarshal(raw, &payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "body must be a document array or contain documents"})
			return
		}
		documents = payload.Documents
	}
	if len(documents) == 0 || len(documents) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "batch must contain 1 to 100 documents"})
		return
	}
	created := make([]*types.SDPivotDocument, 0, len(documents))
	failed := make([]gin.H, 0)
	for index, req := range documents {
		doc, _, err := h.createManualDocument(c, req)
		if err != nil {
			if c.Writer.Written() {
				return
			}
			failed = append(failed, gin.H{"index": index, "title": req.Title, "error": err.Error()})
			continue
		}
		created = append(created, doc)
	}
	c.JSON(http.StatusOK, gin.H{"total": len(documents), "imported": len(created), "failed": len(failed), "documents": created, "errors": failed})
}

func (h *SDPivotDocumentHandler) GetDocumentTags(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	doc, ok := h.authorizeDocument(c, tenantDB, c.Param("id"), spaceAccessView)
	if !ok {
		return
	}
	type tagRow struct {
		ID          string  `json:"id"`
		DimensionID string  `json:"dimension_id"`
		Name        string  `json:"name"`
		Color       string  `json:"color"`
		Source      string  `json:"source"`
		Confidence  float64 `json:"confidence"`
	}
	var tags []tagRow
	if err := tenantDB.Table("document_tags dt").
		Joins("JOIN tag_dictionary td ON td.id = dt.tag_id AND td.dimension_id = dt.dimension_id").
		Where("dt.tenant_id = ? AND dt.document_id = ? AND dt.deleted_at IS NULL", doc.TenantID, doc.ID).
		Select("td.id, td.dimension_id, td.name, td.color, dt.source, dt.confidence").Order("td.sort_order, td.name").Scan(&tags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load document tags"})
		return
	}
	if tags == nil {
		tags = make([]tagRow, 0)
	}
	c.JSON(http.StatusOK, gin.H{"document_id": doc.ID, "raw_tags": doc.Tags, "tags": tags})
}

// SyncDocumentTags replaces a document's dictionary tags for third-party callers.
func (h *SDPivotDocumentHandler) SyncDocumentTags(c *gin.Context) {
	var req struct {
		Tags []string `json:"tags" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	db := middleware.TenantDB(c, h.db)
	doc, ok := authorizeTagDocument(c, db, c.Param("id"), spaceAccessEdit)
	if !ok {
		return
	}
	values := uniqueStrings(req.Tags)
	refs, err := loadTagDictionaryRefs(db, doc.TenantID, values)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate document tags"})
		return
	}
	if len(refs) != len(values) {
		var byName []tagDictionaryRef
		if err := db.Table("tag_dictionary AS dict").Select("dict.id, dict.dimension_id").
			Joins("JOIN tag_dimensions AS dim ON dim.id = dict.dimension_id").
			Where("dict.name IN ? AND dim.deleted_at IS NULL AND dim.enabled = TRUE AND (dim.tenant_id IS NULL OR dim.tenant_id = ?)", values, doc.TenantID).
			Scan(&byName).Error; err != nil || len(byName) != len(values) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "one or more tags are invalid or ambiguous"})
			return
		}
		refs = byName
	}
	if err := replaceSDPivotDocumentTags(db, doc, refs, middleware.GetUserID(c), "third-party synchronization"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to synchronize document tags"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"document_id": doc.ID, "tag_count": len(refs), "status": "updated"})
}

// UploadFromURL handles web page URL import.
func (h *SDPivotDocumentHandler) UploadFromURL(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	if !middleware.HasContextPermission(c, middleware.PermissionKnowledgeWrite) {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permission", "permission": "knowledge.write"})
		return
	}
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
	if _, ok := authorizeSpace(c, tenantDB, req.SpaceID, spaceAccessEdit); !ok {
		return
	}

	// Fetch URL content with SSRF protection, bounded time, and bounded size.
	body, finalURL, status, err := fetchSDPivotURLDocument(c.Request.Context(), req.URL)
	if err != nil {
		switch status {
		case http.StatusRequestEntityTooLarge:
			c.JSON(status, gin.H{"error": "URL content exceeds maximum document size"})
		case http.StatusUnsupportedMediaType:
			c.JSON(status, gin.H{"error": "unsupported URL content type"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to fetch URL"})
		}
		return
	}

	content := normalizeWhitespace(stripHTMLTags(string(body)))

	docID := uuid.New().String()
	now := time.Now()

	hasher := sha256.New()
	hasher.Write(body)
	contentHash := hex.EncodeToString(hasher.Sum(nil))

	// Extract the final response domain for title.
	domain := finalURL.Hostname()
	if finalURL.Port() != "" {
		domain = finalURL.Host
	}

	doc := types.SDPivotDocument{
		ID:              docID,
		TenantID:        tenantID,
		SpaceID:         req.SpaceID,
		UploaderID:      userID,
		Title:           "网页导入 - " + domain,
		FileName:        domain + ".html",
		FileType:        ".html",
		FileSize:        int64(len(body)),
		FilePath:        finalURL.String(),
		ContentHash:     contentHash,
		ParseStatus:     "parsing",
		EmbeddingStatus: "pending",
		Version:         1,
		Tags:            req.Tags,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	version := types.SDPivotDocumentVersion{
		ID:         uuid.New().String(),
		DocumentID: docID,
		TenantID:   tenantID,
		Version:    1,
		FilePath:   finalURL.String(),
		FileSize:   int64(len(body)),
		CreatedAt:  now,
		CreatedBy:  userID,
	}
	if err := createSDPivotDocumentWithVersion(tenantDB, &doc, &version); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create document and version"})
		return
	}

	if err := h.parseAndStoreDocument(c.Request.Context(), tenantDB, &doc, []byte(content)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse URL content"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"document": doc,
		"message":  "URL imported and parsed",
	})
}

func (h *SDPivotDocumentHandler) authorizeDocument(c *gin.Context, db *gorm.DB, docID string, level spaceAccessLevel) (*types.SDPivotDocument, bool) {
	var doc types.SDPivotDocument
	if err := db.Where("id = ? AND tenant_id = ?", docID, middleware.GetTenantID(c)).First(&doc).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return nil, false
	}
	if _, ok := authorizeSpace(c, db, doc.SpaceID, level); !ok {
		return nil, false
	}
	return &doc, true
}

func (h *SDPivotDocumentHandler) authorizeChunk(c *gin.Context, db *gorm.DB, chunkID string, level spaceAccessLevel) (*types.SDPivotDocumentChunk, bool) {
	var chunk types.SDPivotDocumentChunk
	if err := db.Where("id = ? AND tenant_id = ?", chunkID, middleware.GetTenantID(c)).First(&chunk).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "chunk not found"})
		return nil, false
	}
	if _, ok := h.authorizeDocument(c, db, chunk.DocumentID, level); !ok {
		return nil, false
	}
	return &chunk, true
}
