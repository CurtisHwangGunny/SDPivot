package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
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
	if err := db.AutoMigrate(&systemConfigRow{}, &types.TagDimension{}, &types.TagDictionary{}, &types.DocumentTag{}); err != nil {
		t.Fatalf("migrate system_configs: %v", err)
	}
	if err := db.Create(&types.TagDimension{ID: types.DefaultTagDimensionID, Code: "topic", Name: "Topic"}).Error; err != nil {
		t.Fatalf("seed tag dimension: %v", err)
	}
	h := &SystemHandler{db: db}
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.GET("/storage", h.GetStorageConfig)
	r.PUT("/storage", h.UpdateStorageConfig)
	r.GET("/models", h.ListModelConfigs)
	r.POST("/models", h.CreateModelConfig)
	r.GET("/models/:id", h.GetModelConfig)
	r.PUT("/models/:id", h.UpdateModelConfig)
	r.DELETE("/models/:id", h.DeleteModelConfig)
	r.GET("/sms", h.GetSMSSystemConfig)
	r.PUT("/sms", h.UpdateSMSSystemConfig)
	r.GET("/wechat", h.GetWeChatLoginConfig)
	r.PUT("/wechat", h.UpdateWeChatLoginConfig)
	r.PUT("/tags", h.UpdateTagDictionary)
	r.GET("/tags", h.GetTagDictionary)
	r.POST("/tags", h.CreateTagDictionaryEntry)
	r.PUT("/tags/:id", h.UpdateTagDictionaryEntry)
	r.DELETE("/tags/:id", h.DeleteTagDictionaryEntry)
	r.GET("/params", h.GetGlobalParams)
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

func decodeSystemConfigResponse[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var response T
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, w.Body.String())
	}
	return response
}

func loadStoredSystemConfig[T any](t *testing.T, db *gorm.DB, key string) T {
	t.Helper()
	var row systemConfigRow
	if err := db.Where("key = ?", key).First(&row).Error; err != nil {
		t.Fatalf("load stored config %q: %v", key, err)
	}
	var config T
	if err := json.Unmarshal([]byte(row.Value), &config); err != nil {
		t.Fatalf("decode stored config %q: %v", key, err)
	}
	return config
}

func TestModelConfigCRUDAndSecretPreservation(t *testing.T) {
	t.Setenv("SYSTEM_AES_KEY", "0123456789abcdef0123456789abcdef")
	r, db := newSystemConfigTestRouter(t)

	w := performSystemConfigRequest(t, r, http.MethodGet, "/models", "")
	if w.Code != http.StatusOK || strings.TrimSpace(w.Body.String()) != "[]" {
		t.Fatalf("list empty model configs: status=%d body=%s", w.Code, w.Body.String())
	}

	w = performSystemConfigRequest(t, r, http.MethodPost, "/models", `{"name":"Primary","provider":"openai","api_key":"model-secret","temperature":0.4,"max_tokens":4096,"top_p":0.8}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create model config: status=%d body=%s", w.Code, w.Body.String())
	}
	created := decodeSystemConfigResponse[modelConfigResponse](t, w)
	if created.ID == "" || created.Name != "Primary" || !created.APIKeyConfigured || created.Temperature != 0.4 || created.MaxTokens != 4096 || created.TopP != 0.8 {
		t.Fatalf("unexpected created model config: %+v", created)
	}
	if strings.Contains(w.Body.String(), "model-secret") || strings.Contains(w.Body.String(), `"api_key":`) {
		t.Fatalf("model config response exposed API key: %s", w.Body.String())
	}

	stored := loadStoredSystemConfig[storedModelConfig](t, db, modelConfigKeyPrefix+created.ID)
	if stored.APIKey == "model-secret" || !strings.HasPrefix(stored.APIKey, "enc:v1:") {
		t.Fatalf("model API key was not encrypted: %q", stored.APIKey)
	}
	originalAPIKey := stored.APIKey

	w = performSystemConfigRequest(t, r, http.MethodGet, "/models/"+created.ID, "")
	if w.Code != http.StatusOK {
		t.Fatalf("get model config: status=%d body=%s", w.Code, w.Body.String())
	}
	got := decodeSystemConfigResponse[modelConfigResponse](t, w)
	if got.ID != created.ID || got.Name != "Primary" || !got.APIKeyConfigured {
		t.Fatalf("unexpected fetched model config: %+v", got)
	}

	w = performSystemConfigRequest(t, r, http.MethodPut, "/models/"+created.ID, `{"name":"Primary Updated","provider":"openai-compatible","temperature":0.6,"max_tokens":8192,"top_p":0.9}`)
	if w.Code != http.StatusOK {
		t.Fatalf("update model config: status=%d body=%s", w.Code, w.Body.String())
	}
	updated := decodeSystemConfigResponse[modelConfigResponse](t, w)
	if updated.Name != "Primary Updated" || updated.Provider != "openai-compatible" || !updated.APIKeyConfigured || updated.MaxTokens != 8192 {
		t.Fatalf("unexpected updated model config: %+v", updated)
	}
	stored = loadStoredSystemConfig[storedModelConfig](t, db, modelConfigKeyPrefix+created.ID)
	if stored.APIKey != originalAPIKey {
		t.Fatalf("omitted model API key was not preserved: before=%q after=%q", originalAPIKey, stored.APIKey)
	}

	w = performSystemConfigRequest(t, r, http.MethodGet, "/models", "")
	if w.Code != http.StatusOK {
		t.Fatalf("list model configs: status=%d body=%s", w.Code, w.Body.String())
	}
	listed := decodeSystemConfigResponse[[]modelConfigResponse](t, w)
	if len(listed) != 1 || listed[0].ID != created.ID || listed[0].Name != "Primary Updated" || !listed[0].APIKeyConfigured {
		t.Fatalf("unexpected model config list: %+v", listed)
	}

	w = performSystemConfigRequest(t, r, http.MethodDelete, "/models/"+created.ID, "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete model config: status=%d body=%s", w.Code, w.Body.String())
	}
	w = performSystemConfigRequest(t, r, http.MethodGet, "/models/"+created.ID, "")
	if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "Model configuration not found") {
		t.Fatalf("get deleted model config: status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestModelConfigDuplicateNotFoundAndValidation(t *testing.T) {
	r, _ := newSystemConfigTestRouter(t)

	w := performSystemConfigRequest(t, r, http.MethodPost, "/models", `{"name":"Duplicate","provider":"openai"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("seed model config: status=%d body=%s", w.Code, w.Body.String())
	}
	w = performSystemConfigRequest(t, r, http.MethodPost, "/models", `{"name":" duplicate ","provider":"other"}`)
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "already exists") {
		t.Fatalf("duplicate model config: status=%d body=%s", w.Code, w.Body.String())
	}

	validationCases := []struct {
		name string
		body string
	}{
		{name: "missing name", body: `{"provider":"openai"}`},
		{name: "temperature", body: `{"name":"Invalid Temperature","provider":"openai","temperature":2.1}`},
		{name: "max tokens", body: `{"name":"Invalid Tokens","provider":"openai","max_tokens":0}`},
		{name: "top p", body: `{"name":"Invalid Top P","provider":"openai","top_p":1.1}`},
		{name: "endpoint", body: `{"name":"Invalid Endpoint","provider":"openai","endpoint":"http://127.0.0.1:8080"}`},
	}
	for _, tc := range validationCases {
		t.Run(tc.name, func(t *testing.T) {
			w := performSystemConfigRequest(t, r, http.MethodPost, "/models", tc.body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected validation failure: status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}

	for _, tc := range []struct {
		method string
		body   string
	}{
		{method: http.MethodGet},
		{method: http.MethodPut, body: `{"name":"Missing","provider":"openai"}`},
		{method: http.MethodDelete},
	} {
		w = performSystemConfigRequest(t, r, tc.method, "/models/missing", tc.body)
		if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "Model configuration not found") {
			t.Fatalf("%s missing model config: status=%d body=%s", tc.method, w.Code, w.Body.String())
		}
	}
}

func TestSMSConfigDefaultsAndSecretPreservation(t *testing.T) {
	t.Setenv("SYSTEM_AES_KEY", "0123456789abcdef0123456789abcdef")
	r, db := newSystemConfigTestRouter(t)

	w := performSystemConfigRequest(t, r, http.MethodGet, "/sms", "")
	if w.Code != http.StatusOK {
		t.Fatalf("get default SMS config: status=%d body=%s", w.Code, w.Body.String())
	}
	defaults := decodeSystemConfigResponse[smsConfigResponse](t, w)
	if defaults.Provider != "custom" || defaults.TimeoutSeconds != 10 || defaults.CustomHeaders == nil {
		t.Fatalf("unexpected default SMS config: %+v", defaults)
	}

	w = performSystemConfigRequest(t, r, http.MethodPut, "/sms", `{"provider":"custom","endpoint":"https://sms.example.com/send","access_key_id":"sms-key","access_key_secret":"sms-secret","region":"us-east-1","custom_headers":{"X-Tenant":"platform"},"timeout_seconds":30}`)
	if w.Code != http.StatusOK {
		t.Fatalf("create SMS config: status=%d body=%s", w.Code, w.Body.String())
	}
	created := decodeSystemConfigResponse[smsConfigResponse](t, w)
	if !created.AccessKeyIDConfigured || !created.AccessKeySecretConfigured || created.CustomHeaders["X-Tenant"] != "platform" {
		t.Fatalf("unexpected SMS response: %+v", created)
	}
	if strings.Contains(w.Body.String(), "sms-key") || strings.Contains(w.Body.String(), "sms-secret") {
		t.Fatalf("SMS response exposed a secret: %s", w.Body.String())
	}
	stored := loadStoredSystemConfig[smsSystemConfig](t, db, smsConfigKey)
	if !strings.HasPrefix(stored.AccessKeyID, "enc:v1:") || !strings.HasPrefix(stored.AccessKeySecret, "enc:v1:") {
		t.Fatalf("SMS secrets were not encrypted: %+v", stored)
	}
	originalAccessKey, originalSecret := stored.AccessKeyID, stored.AccessKeySecret

	w = performSystemConfigRequest(t, r, http.MethodPut, "/sms", `{"provider":"custom","endpoint":"https://sms.example.com/v2","region":"us-west-2","timeout_seconds":20}`)
	if w.Code != http.StatusOK {
		t.Fatalf("update SMS config: status=%d body=%s", w.Code, w.Body.String())
	}
	stored = loadStoredSystemConfig[smsSystemConfig](t, db, smsConfigKey)
	if stored.AccessKeyID != originalAccessKey || stored.AccessKeySecret != originalSecret || stored.CustomHeaders["X-Tenant"] != "platform" {
		t.Fatalf("omitted SMS secrets or headers were not preserved: %+v", stored)
	}

	w = performSystemConfigRequest(t, r, http.MethodPut, "/sms", `{"provider":"invalid","endpoint":"https://sms.example.com"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected SMS validation error: status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestWeChatConfigDefaultsValidationAndSecretPreservation(t *testing.T) {
	t.Setenv("SYSTEM_AES_KEY", "0123456789abcdef0123456789abcdef")
	r, db := newSystemConfigTestRouter(t)

	w := performSystemConfigRequest(t, r, http.MethodGet, "/wechat", "")
	if w.Code != http.StatusOK {
		t.Fatalf("get default WeChat config: status=%d body=%s", w.Code, w.Body.String())
	}
	defaults := decodeSystemConfigResponse[wechatLoginConfigResponse](t, w)
	if defaults.Enabled || defaults.AppSecretConfigured {
		t.Fatalf("unexpected default WeChat config: %+v", defaults)
	}

	w = performSystemConfigRequest(t, r, http.MethodPut, "/wechat", `{"enabled":true}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected WeChat validation error: status=%d body=%s", w.Code, w.Body.String())
	}

	w = performSystemConfigRequest(t, r, http.MethodPut, "/wechat", `{"enabled":true,"app_id":"wechat-app","app_secret":"wechat-secret","redirect_url":"https://app.example.com/wechat/callback"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("create WeChat config: status=%d body=%s", w.Code, w.Body.String())
	}
	created := decodeSystemConfigResponse[wechatLoginConfigResponse](t, w)
	if !created.Enabled || !created.AppSecretConfigured || created.AppID != "wechat-app" {
		t.Fatalf("unexpected WeChat response: %+v", created)
	}
	if strings.Contains(w.Body.String(), "wechat-secret") || strings.Contains(w.Body.String(), `"app_secret":`) {
		t.Fatalf("WeChat response exposed app secret: %s", w.Body.String())
	}
	stored := loadStoredSystemConfig[wechatLoginConfig](t, db, wechatLoginKey)
	if !strings.HasPrefix(stored.AppSecret, "enc:v1:") {
		t.Fatalf("WeChat app secret was not encrypted: %q", stored.AppSecret)
	}
	originalSecret := stored.AppSecret

	w = performSystemConfigRequest(t, r, http.MethodPut, "/wechat", `{"enabled":true,"app_id":"wechat-app-2","redirect_url":"https://app.example.com/wechat/v2/callback"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("update WeChat config: status=%d body=%s", w.Code, w.Body.String())
	}
	stored = loadStoredSystemConfig[wechatLoginConfig](t, db, wechatLoginKey)
	if stored.AppSecret != originalSecret {
		t.Fatalf("omitted WeChat app secret was not preserved: before=%q after=%q", originalSecret, stored.AppSecret)
	}
	w = performSystemConfigRequest(t, r, http.MethodGet, "/wechat", "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"app_secret_configured":true`) || strings.Contains(w.Body.String(), "wechat-secret") {
		t.Fatalf("get saved WeChat config: status=%d body=%s", w.Code, w.Body.String())
	}
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

func TestTagDictionaryCRUD(t *testing.T) {
	r, db := newSystemConfigTestRouter(t)
	w := performSystemConfigRequest(t, r, http.MethodPost, "/tags", `{"name":"Finance","color":"#123456","sort_order":10}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create tag: status=%d body=%s", w.Code, w.Body.String())
	}
	var created types.TagDictionary
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created tag: %v", err)
	}
	if created.ID == "" || created.DimensionID != types.DefaultTagDimensionID {
		t.Fatalf("unexpected created tag: %+v", created)
	}

	w = performSystemConfigRequest(t, r, http.MethodPut, "/tags/"+created.ID, `{"name":"Legal","color":"#654321","sort_order":20}`)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"name":"Legal"`) {
		t.Fatalf("update tag: status=%d body=%s", w.Code, w.Body.String())
	}

	w = performSystemConfigRequest(t, r, http.MethodGet, "/tags", "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"name":"Legal"`) {
		t.Fatalf("list tags: status=%d body=%s", w.Code, w.Body.String())
	}

	w = performSystemConfigRequest(t, r, http.MethodDelete, "/tags/"+created.ID, "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete tag: status=%d body=%s", w.Code, w.Body.String())
	}
	var count int64
	if err := db.Model(&types.TagDictionary{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("expected empty dictionary, count=%d err=%v", count, err)
	}
}

func TestUpdateGlobalParamsValidatesAndPersists(t *testing.T) {
	r, db := newSystemConfigTestRouter(t)
	w := performSystemConfigRequest(t, r, http.MethodGet, "/params", "")
	if w.Code != http.StatusOK {
		t.Fatalf("get default global params: status=%d body=%s", w.Code, w.Body.String())
	}
	defaults := decodeSystemConfigResponse[globalParamsConfig](t, w)
	if defaults != defaultGlobalParamsConfig() {
		t.Fatalf("unexpected default global params: %+v", defaults)
	}

	w = performSystemConfigRequest(t, r, http.MethodPut, "/params", `{"chunk_size":512,"threshold":1.1,"token_limit":4096,"concurrency":32}`)
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
	w = performSystemConfigRequest(t, r, http.MethodGet, "/params", "")
	if w.Code != http.StatusOK {
		t.Fatalf("get saved global params: status=%d body=%s", w.Code, w.Body.String())
	}
	saved := decodeSystemConfigResponse[globalParamsConfig](t, w)
	if saved.ChunkSize != 1024 || saved.Threshold != 0.75 || saved.TokenLimit != 8192 || saved.Concurrency != 64 {
		t.Fatalf("unexpected saved global params: %+v", saved)
	}
}
