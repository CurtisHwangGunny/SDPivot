package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/types"
)

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
		c.Next()
	})
	h := &SDPivotDocumentHandler{db: db, uploadDir: uploadDir}
	router.POST("/documents/upload", h.UploadDocument)
	router.POST("/documents/manual", h.UploadManualDocument)
	router.POST("/documents/url", h.UploadFromURL)

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
