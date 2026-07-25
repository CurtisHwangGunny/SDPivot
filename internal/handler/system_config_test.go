package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newSystemConfigTestRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&systemConfigRow{}); err != nil {
		t.Fatalf("migrate system_configs: %v", err)
	}
	h := &SystemHandler{db: db}
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.GET("/storage", h.GetStorageConfig)
	r.PUT("/storage", h.UpdateStorageConfig)
	r.PUT("/tags", h.UpdateTagDictionary)
	r.PUT("/params", h.UpdateGlobalParams)
	return r, db
}

func performSystemConfigRequest(t *testing.T, r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	r.ServeHTTP(w, req)
	return w
}

func TestStorageConfigDefaultsAndSecretPreservation(t *testing.T) {
	r, db := newSystemConfigTestRouter(t)

	w := performSystemConfigRequest(t, r, http.MethodGet, "/storage", "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"provider":"minio"`) {
		t.Fatalf("unexpected default response: status=%d body=%s", w.Code, w.Body.String())
	}

	createBody := `{"provider":"s3","endpoint":"https://s3.example.com","region":"us-east-1","bucket":"docs","access_key_id":"key-1","secret_access_key":"secret-1","use_ssl":true,"force_path_style":false}`
	w = performSystemConfigRequest(t, r, http.MethodPut, "/storage", createBody)
	if w.Code != http.StatusOK {
		t.Fatalf("create storage config: status=%d body=%s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "key-1") || strings.Contains(w.Body.String(), "secret-1") {
		t.Fatalf("storage response exposed a secret: %s", w.Body.String())
	}

	updateBody := `{"provider":"s3","endpoint":"https://s3-2.example.com","region":"us-west-2","bucket":"docs-2","use_ssl":true,"force_path_style":true}`
	w = performSystemConfigRequest(t, r, http.MethodPut, "/storage", updateBody)
	if w.Code != http.StatusOK {
		t.Fatalf("update storage config: status=%d body=%s", w.Code, w.Body.String())
	}

	var row systemConfigRow
	if err := db.Where("key = ?", storageConfigKey).First(&row).Error; err != nil {
		t.Fatalf("load stored config: %v", err)
	}
	var stored storageConfig
	if err := json.Unmarshal([]byte(row.Value), &stored); err != nil {
		t.Fatalf("decode stored config: %v", err)
	}
	if stored.AccessKeyID != "key-1" || stored.SecretAccessKey != "secret-1" {
		t.Fatalf("omitted secrets were not preserved: %+v", stored)
	}
}

func TestUpdateTagDictionaryRejectsDuplicateNames(t *testing.T) {
	r, _ := newSystemConfigTestRouter(t)
	w := performSystemConfigRequest(t, r, http.MethodPut, "/tags", `{"tags":[{"name":"Finance"},{"name":"finance"}]}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestUpdateGlobalParamsValidatesAndPersists(t *testing.T) {
	r, db := newSystemConfigTestRouter(t)
	w := performSystemConfigRequest(t, r, http.MethodPut, "/params", `{"chunk_size":512,"threshold":1.1,"token_limit":4096,"concurrency":32}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected threshold validation error, got status=%d body=%s", w.Code, w.Body.String())
	}

	w = performSystemConfigRequest(t, r, http.MethodPut, "/params", `{"chunk_size":1024,"threshold":0.75,"token_limit":8192,"concurrency":64}`)
	if w.Code != http.StatusOK {
		t.Fatalf("save global params: status=%d body=%s", w.Code, w.Body.String())
	}
	var count int64
	if err := db.Model(&systemConfigRow{}).Where("key = ?", globalParamsKey).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("expected one global params row, count=%d err=%v", count, err)
	}
}
