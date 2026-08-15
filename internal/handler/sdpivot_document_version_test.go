package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/types"
)

type stubSDPivotDocReader struct {
	read func(context.Context, *types.ReadRequest) (*types.ReadResult, error)
}

func (s *stubSDPivotDocReader) Read(ctx context.Context, req *types.ReadRequest) (*types.ReadResult, error) {
	return s.read(ctx, req)
}

func newSDPivotDocumentTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:sdpivot-document-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&types.KnowledgeSpace{},
		&types.SpaceMember{},
		&types.SDPivotDocument{},
		&types.SDPivotDocumentVersion{},
		&types.SDPivotDocumentChunk{},
	); err != nil {
		t.Fatalf("migrate document tables: %v", err)
	}
	return db
}

func serveSDPivotDocumentRequest(t *testing.T, db *gorm.DB, uploadDir string, tenantID uint64, method string, path string, body *bytes.Buffer, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	if err := db.Create(&types.KnowledgeSpace{ID: "space-1", TenantID: tenantID, Name: "test space", Visibility: "private"}).Error; err != nil {
		t.Fatalf("create test space: %v", err)
	}
	if err := db.Create(&types.SpaceMember{SpaceID: "space-1", UserID: "user-1", Role: "editor"}).Error; err != nil {
		t.Fatalf("create test space member: %v", err)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		c.Set("user_id", "user-1")
		c.Set("role", "knowledge_editor")
		c.Next()
	})
	h := &SDPivotDocumentHandler{db: db, uploadDir: uploadDir}
	router.POST("/documents/upload", h.UploadDocument)
	router.POST("/documents/manual", h.UploadManualDocument)
	router.POST("/documents/url", h.UploadFromURL)
	router.POST("/documents/:id/reparse", h.ReparseDocument)

	req := httptest.NewRequest(method, path, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	return response
}

func uploadSDPivotTestDocument(t *testing.T, db *gorm.DB, uploadDir string, tenantID uint64) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("space_id", "space-1"); err != nil {
		t.Fatalf("write space_id: %v", err)
	}
	part, err := writer.CreateFormFile("file", "tenant-document.md")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte("tenant scoped document content")); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	return serveSDPivotDocumentRequest(t, db, uploadDir, tenantID, http.MethodPost, "/documents/upload", &body, writer.FormDataContentType())
}

func assertSingleDocumentVersionTenant(t *testing.T, db *gorm.DB, tenantID uint64) (types.SDPivotDocument, types.SDPivotDocumentVersion) {
	t.Helper()
	var doc types.SDPivotDocument
	if err := db.First(&doc).Error; err != nil {
		t.Fatalf("load document: %v", err)
	}
	var versions []types.SDPivotDocumentVersion
	if err := db.Where("document_id = ?", doc.ID).Find(&versions).Error; err != nil {
		t.Fatalf("load document versions: %v", err)
	}
	if len(versions) != 1 {
		t.Fatalf("expected one initial document version, got %d", len(versions))
	}
	version := versions[0]
	if doc.TenantID != tenantID || version.TenantID != tenantID {
		t.Fatalf("tenant mismatch: request=%d document=%d version=%d", tenantID, doc.TenantID, version.TenantID)
	}
	if doc.Version != 1 || version.Version != 1 {
		t.Fatalf("initial version mismatch: document=%d version=%d", doc.Version, version.Version)
	}
	return doc, version
}

func TestUploadDocumentCreatesVersionWithRequestTenant(t *testing.T) {
	db := newSDPivotDocumentTestDB(t)
	tenantID := uint64(42)
	response := uploadSDPivotTestDocument(t, db, t.TempDir(), tenantID)
	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", response.Code, response.Body.String())
	}

	doc, version := assertSingleDocumentVersionTenant(t, db, tenantID)
	if version.FilePath != doc.FilePath {
		t.Fatalf("expected version file path %q, got %q", doc.FilePath, version.FilePath)
	}
}

func TestUploadManualDocumentCreatesInitialVersionWithRequestTenant(t *testing.T) {
	db := newSDPivotDocumentTestDB(t)
	tenantID := uint64(51)
	payload, err := json.Marshal(map[string]string{
		"space_id": "space-1",
		"title":    "manual document",
		"content":  "manual tenant scoped content",
	})
	if err != nil {
		t.Fatal(err)
	}
	response := serveSDPivotDocumentRequest(t, db, t.TempDir(), tenantID, http.MethodPost, "/documents/manual", bytes.NewBuffer(payload), "application/json")
	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", response.Code, response.Body.String())
	}

	doc, version := assertSingleDocumentVersionTenant(t, db, tenantID)
	if version.FilePath != "" {
		t.Fatalf("expected manual version without file path, got %q", version.FilePath)
	}
	if version.FileSize != doc.FileSize {
		t.Fatalf("expected version size %d, got %d", doc.FileSize, version.FileSize)
	}
}

func TestReparseManualDocumentUsesStoredChunks(t *testing.T) {
	db := newSDPivotDocumentTestDB(t)
	tenantID := uint64(61)
	doc := types.SDPivotDocument{
		ID:          "manual-doc",
		TenantID:    tenantID,
		SpaceID:     "space-1",
		Title:       "manual document",
		FileType:    ".md",
		ParseStatus: "completed",
	}
	if err := db.Create(&doc).Error; err != nil {
		t.Fatalf("create manual document: %v", err)
	}
	chunks := []types.SDPivotDocumentChunk{
		{ID: "chunk-1", DocumentID: doc.ID, TenantID: tenantID, ChunkIndex: 0, Content: "first section"},
		{ID: "chunk-2", DocumentID: doc.ID, TenantID: tenantID, ChunkIndex: 1, Content: "second section"},
	}
	if err := db.Create(&chunks).Error; err != nil {
		t.Fatalf("create stored chunks: %v", err)
	}

	response := serveSDPivotDocumentRequest(t, db, t.TempDir(), tenantID, http.MethodPost, "/documents/manual-doc/reparse", bytes.NewBuffer(nil), "")
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", response.Code, response.Body.String())
	}

	var reparsed []types.SDPivotDocumentChunk
	if err := db.Where("document_id = ?", doc.ID).Order("chunk_index").Find(&reparsed).Error; err != nil {
		t.Fatalf("load reparsed chunks: %v", err)
	}
	if len(reparsed) != 1 || reparsed[0].Content != "first section second section" {
		t.Fatalf("unexpected reparsed chunks: %#v", reparsed)
	}
}

func TestCreateSDPivotDocumentWithVersionRollsBackManualOrURLDocument(t *testing.T) {
	db := newSDPivotDocumentTestDB(t)
	if err := db.Exec(`CREATE TRIGGER reject_document_version BEFORE INSERT ON document_versions BEGIN SELECT RAISE(FAIL, 'forced version failure'); END`).Error; err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}
	doc := types.SDPivotDocument{ID: "doc-rollback", TenantID: 82, SpaceID: "space-1", Title: "rollback", Version: 1}
	version := types.SDPivotDocumentVersion{ID: "version-rollback", DocumentID: doc.ID, TenantID: doc.TenantID, Version: 1}
	if err := createSDPivotDocumentWithVersion(db, &doc, &version); err == nil {
		t.Fatal("expected version insert failure")
	}
	var documentCount int64
	if err := db.Model(&types.SDPivotDocument{}).Count(&documentCount).Error; err != nil {
		t.Fatalf("count documents: %v", err)
	}
	if documentCount != 0 {
		t.Fatalf("expected document rollback, got %d rows", documentCount)
	}
}

func TestUploadDocumentVersionInsertFailureRollsBackAndRemovesFile(t *testing.T) {
	db := newSDPivotDocumentTestDB(t)
	if err := db.Exec(`CREATE TRIGGER reject_document_version BEFORE INSERT ON document_versions BEGIN SELECT RAISE(FAIL, 'forced version failure'); END`).Error; err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}
	uploadDir := t.TempDir()
	response := uploadSDPivotTestDocument(t, db, uploadDir, 73)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", response.Code, response.Body.String())
	}

	var documentCount int64
	if err := db.Model(&types.SDPivotDocument{}).Count(&documentCount).Error; err != nil {
		t.Fatalf("count documents: %v", err)
	}
	if documentCount != 0 {
		t.Fatalf("expected document insert rollback, got %d rows", documentCount)
	}
	files, err := filepath.Glob(filepath.Join(uploadDir, "73", "*"))
	if err != nil {
		t.Fatalf("list uploaded files: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected failed upload file cleanup, found %v", files)
	}
	if _, err := os.Stat(filepath.Join(uploadDir, "73")); err != nil && !os.IsNotExist(err) {
		t.Fatalf("stat tenant upload directory: %v", err)
	}
}

func TestParseAndStoreDocumentUsesDocReaderForOfficeAndPDF(t *testing.T) {
	fileTypes := []string{".doc", ".docx", ".ppt", ".pptx", ".xls", ".xlsx", ".pdf"}
	for _, fileType := range fileTypes {
		t.Run(fileType, func(t *testing.T) {
			db := newSDPivotDocumentTestDB(t)
			doc := types.SDPivotDocument{
				ID:          "document-" + fileType[1:],
				TenantID:    91,
				SpaceID:     "space-1",
				Title:       "Quarterly report",
				FileName:    "report" + fileType,
				FileType:    fileType,
				ParseStatus: "pending",
			}
			if err := db.Create(&doc).Error; err != nil {
				t.Fatalf("create document: %v", err)
			}

			var received *types.ReadRequest
			h := &SDPivotDocumentHandler{
				db: db,
				documentReader: &stubSDPivotDocReader{read: func(_ context.Context, req *types.ReadRequest) (*types.ReadResult, error) {
					received = req
					return &types.ReadResult{MarkdownContent: "# Parsed report\n\nOffice content"}, nil
				}},
			}
			content := []byte("binary office content")
			if err := h.parseAndStoreDocument(context.Background(), db, &doc, content); err != nil {
				t.Fatalf("parse document: %v", err)
			}
			if received == nil {
				t.Fatal("expected document reader to be called")
			}
			if received.FileName != doc.FileName || received.FileType != fileType[1:] || received.Title != doc.Title || received.RequestID != doc.ID {
				t.Fatalf("unexpected read request: %#v", received)
			}
			if !bytes.Equal(received.FileContent, content) {
				t.Fatalf("unexpected file content: %q", received.FileContent)
			}

			var stored types.SDPivotDocument
			if err := db.First(&stored, "id = ?", doc.ID).Error; err != nil {
				t.Fatalf("load document: %v", err)
			}
			if stored.ParseStatus != "completed" || stored.ChunkCount != 1 {
				t.Fatalf("unexpected parse result: status=%s chunks=%d", stored.ParseStatus, stored.ChunkCount)
			}
			var chunk types.SDPivotDocumentChunk
			if err := db.First(&chunk, "document_id = ?", doc.ID).Error; err != nil {
				t.Fatalf("load chunk: %v", err)
			}
			if chunk.Content != "# Parsed report\n\nOffice content" {
				t.Fatalf("unexpected chunk content: %q", chunk.Content)
			}
		})
	}
}

func TestParseAndStoreDocumentKeepsTextFallback(t *testing.T) {
	db := newSDPivotDocumentTestDB(t)
	doc := types.SDPivotDocument{ID: "markdown-document", TenantID: 92, SpaceID: "space-1", FileName: "notes.md", FileType: ".md", ParseStatus: "pending"}
	if err := db.Create(&doc).Error; err != nil {
		t.Fatalf("create document: %v", err)
	}
	h := &SDPivotDocumentHandler{db: db}
	if err := h.parseAndStoreDocument(context.Background(), db, &doc, []byte("first\n\nsecond")); err != nil {
		t.Fatalf("parse markdown fallback: %v", err)
	}
	var chunk types.SDPivotDocumentChunk
	if err := db.First(&chunk, "document_id = ?", doc.ID).Error; err != nil {
		t.Fatalf("load chunk: %v", err)
	}
	if chunk.Content != "first second" {
		t.Fatalf("unexpected fallback content: %q", chunk.Content)
	}
}

func TestParseAndStoreDocumentMarksDocReaderFailure(t *testing.T) {
	db := newSDPivotDocumentTestDB(t)
	doc := types.SDPivotDocument{ID: "failed-document", TenantID: 93, SpaceID: "space-1", FileName: "report.pdf", FileType: ".pdf", ParseStatus: "pending"}
	if err := db.Create(&doc).Error; err != nil {
		t.Fatalf("create document: %v", err)
	}
	h := &SDPivotDocumentHandler{
		db: db,
		documentReader: &stubSDPivotDocReader{read: func(context.Context, *types.ReadRequest) (*types.ReadResult, error) {
			return nil, errors.New("docreader unavailable")
		}},
	}
	err := h.parseAndStoreDocument(context.Background(), db, &doc, []byte("pdf content"))
	if err == nil || !strings.Contains(err.Error(), "docreader unavailable") {
		t.Fatalf("expected clear document reader error, got %v", err)
	}
	var stored types.SDPivotDocument
	if err := db.First(&stored, "id = ?", doc.ID).Error; err != nil {
		t.Fatalf("load document: %v", err)
	}
	if stored.ParseStatus != "failed" || stored.ChunkCount != 0 {
		t.Fatalf("unexpected failed parse state: status=%s chunks=%d", stored.ParseStatus, stored.ChunkCount)
	}
}
