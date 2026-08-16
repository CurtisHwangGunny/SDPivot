package tests

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	functionalTimeout      = 10 * time.Minute
	functionalPollInterval = 2 * time.Second
)

type apiSuite struct {
	t       *testing.T
	client  *http.Client
	baseURL string
	token   string
	userID  string
	tenant  uint64
	runID   string
}

type apiResponse struct {
	status int
	header http.Header
	body   []byte
}

type loginResponse struct {
	Token string `json:"token"`
	User  struct {
		ID            string `json:"id"`
		IsSystemAdmin bool   `json:"is_system_admin"`
	} `json:"user"`
	ActiveTenant struct {
		ID uint64 `json:"id"`
	} `json:"active_tenant"`
}

type systemSetting struct {
	ID    uint64          `json:"id"`
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

type backupSchedule struct {
	Enabled       bool   `json:"enabled"`
	Cron          string `json:"cron"`
	RetentionDays int    `json:"retention_days"`
}

type backupRecord struct {
	ID             string `json:"id"`
	Status         string `json:"status"`
	SizeBytes      int64  `json:"size_bytes"`
	ChecksumSHA256 string `json:"checksum_sha256"`
	ErrorMessage   string `json:"error_message"`
}

func TestAPIFunctional(t *testing.T) {
	if os.Getenv("WEKNORA_API_FUNCTIONAL") != "1" {
		t.Skip("set WEKNORA_API_FUNCTIONAL=1 to run live API functional tests")
	}

	email := requireEnv(t, "WEKNORA_E2E_ADMIN_EMAIL")
	password := requireEnv(t, "WEKNORA_E2E_ADMIN_PASSWORD")
	baseURL := strings.TrimRight(os.Getenv("WEKNORA_E2E_BASE_URL"), "/")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8080"
	}

	s := &apiSuite{
		t:       t,
		client:  &http.Client{Timeout: 3 * time.Minute},
		baseURL: baseURL,
		runID:   fmt.Sprintf("api-functional-%d", time.Now().UTC().UnixNano()),
	}

	ctx, cancel := context.WithTimeout(context.Background(), functionalTimeout)
	defer cancel()

	s.health(ctx)
	login := s.login(ctx, email, password)
	if !login.User.IsSystemAdmin {
		t.Fatal("WEKNORA_E2E_ADMIN_EMAIL must be a System Admin")
	}
	s.token = login.Token
	s.userID = login.User.ID
	s.tenant = login.ActiveTenant.ID
	if s.token == "" || s.userID == "" || s.tenant == 0 {
		t.Fatalf("login response is missing token, user ID, or active tenant: %+v", login)
	}

	t.Run("user CRUD and role assignment", func(t *testing.T) { s.withT(t).testUserAndRole(ctx) })
	t.Run("department CRUD", func(t *testing.T) { s.withT(t).testDepartmentCRUD(ctx) })
	t.Run("model config CRUD", func(t *testing.T) { s.withT(t).testModelConfigCRUD(ctx) })
	t.Run("system config", func(t *testing.T) { s.withT(t).testSystemConfig(ctx) })
	t.Run("tag CRUD", func(t *testing.T) { s.withT(t).testTagCRUD(ctx) })
	t.Run("backup endpoints", func(t *testing.T) { s.withT(t).testBackups(ctx) })
	t.Run("audit log query", func(t *testing.T) { s.withT(t).testAuditLogQuery(ctx) })
}

func (s *apiSuite) withT(t *testing.T) *apiSuite {
	clone := *s
	clone.t = t
	return &clone
}

func requireEnv(t *testing.T, name string) string {
	t.Helper()
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		t.Fatalf("%s is required", name)
	}
	return value
}

func (s *apiSuite) health(ctx context.Context) {
	resp := s.request(ctx, http.MethodGet, "/health", "", nil)
	s.requireStatus(resp, http.StatusOK)
	var body struct {
		Status string `json:"status"`
	}
	s.decode(resp, &body)
	if body.Status != "ok" {
		s.t.Fatalf("health status: got %q, want ok", body.Status)
	}
}

func (s *apiSuite) login(ctx context.Context, email, password string) loginResponse {
	body := mustJSON(s.t, map[string]any{"email": email, "password": password})
	resp := s.request(ctx, http.MethodPost, "/api/v1/auth/login", "", body)
	s.requireStatus(resp, http.StatusOK)
	var result loginResponse
	s.decode(resp, &result)
	return result
}

func (s *apiSuite) testUserAndRole(ctx context.Context) {
	// The public API has no global user-delete endpoint. Its deletable user
	// lifecycle is registration plus tenant membership removal.
	email := s.runID + "@example.invalid"
	password := "ApiFunctional-Aa1!" + strconv.FormatInt(time.Now().UnixNano(), 10)
	register := s.request(ctx, http.MethodPost, "/api/v1/auth/register", "", mustJSON(s.t, map[string]any{
		"username": s.runID,
		"email":    email,
		"password": password,
	}))
	s.requireStatus(register, http.StatusCreated)
	var created struct {
		Success bool `json:"success"`
		User    struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"user"`
	}
	s.decode(register, &created)
	if !created.Success || created.User.ID == "" || created.User.Email != email {
		s.t.Fatalf("unexpected registration response: %s", register.body)
	}

	userLogin := s.login(ctx, email, password)
	me := s.request(ctx, http.MethodGet, "/api/v1/auth/me", userLogin.Token, nil)
	s.requireStatus(me, http.StatusOK)
	var current struct {
		Success bool `json:"success"`
		Data    struct {
			User struct {
				ID    string `json:"id"`
				Email string `json:"email"`
			} `json:"user"`
		} `json:"data"`
	}
	s.decode(me, &current)
	if !current.Success || current.Data.User.ID != created.User.ID || current.Data.User.Email != email {
		s.t.Fatalf("unexpected current-user response: %s", me.body)
	}

	preferences := s.request(ctx, http.MethodPut, "/api/v1/auth/me/preferences", userLogin.Token,
		mustJSON(s.t, map[string]any{"enable_memory": true}))
	s.requireStatus(preferences, http.StatusOK)
	if !jsonPathBool(s.t, preferences.body, "success") {
		s.t.Fatalf("preference update did not succeed: %s", preferences.body)
	}

	memberPath := fmt.Sprintf("/api/v1/tenants/%d/members", s.tenant)
	added := s.request(ctx, http.MethodPost, memberPath, s.token,
		mustJSON(s.t, map[string]any{"email": email, "role": "viewer"}))
	s.requireStatus(added, http.StatusCreated)

	memberID := created.User.ID
	s.t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		s.request(cleanupCtx, http.MethodDelete, memberPath+"/"+url.PathEscape(memberID), s.token, nil)
	})

	listed := s.request(ctx, http.MethodGet, memberPath+"?q="+url.QueryEscape(email)+"&page=1&page_size=10", s.token, nil)
	s.requireStatus(listed, http.StatusOK)
	var members struct {
		Success bool `json:"success"`
		Data    struct {
			Members []struct {
				UserID string `json:"user_id"`
				Role   string `json:"role"`
			} `json:"members"`
		} `json:"data"`
	}
	s.decode(listed, &members)
	if !members.Success || len(members.Data.Members) != 1 || members.Data.Members[0].UserID != memberID || members.Data.Members[0].Role != "viewer" {
		s.t.Fatalf("created member not returned by list: %s", listed.body)
	}

	memberItemPath := memberPath + "/" + url.PathEscape(memberID)
	updated := s.request(ctx, http.MethodPut, memberItemPath, s.token, mustJSON(s.t, map[string]any{"role": "contributor"}))
	s.requireStatus(updated, http.StatusOK)

	listed = s.request(ctx, http.MethodGet, memberPath+"?q="+url.QueryEscape(email), s.token, nil)
	s.requireStatus(listed, http.StatusOK)
	s.decode(listed, &members)
	if len(members.Data.Members) != 1 || members.Data.Members[0].Role != "contributor" {
		s.t.Fatalf("role assignment was not persisted: %s", listed.body)
	}

	removed := s.request(ctx, http.MethodDelete, memberItemPath, s.token, nil)
	s.requireStatus(removed, http.StatusOK)
	listed = s.request(ctx, http.MethodGet, memberPath+"?q="+url.QueryEscape(email), s.token, nil)
	s.requireStatus(listed, http.StatusOK)
	s.decode(listed, &members)
	if len(members.Data.Members) != 0 {
		s.t.Fatalf("removed member is still listed: %s", listed.body)
	}
}

func (s *apiSuite) testDepartmentCRUD(ctx context.Context) {
	basePath := fmt.Sprintf("/api/v1/tenants/%d/departments", s.tenant)
	name := s.runID + " department"
	created := s.request(ctx, http.MethodPost, basePath, s.token, mustJSON(s.t, map[string]any{
		"parent_id":   "",
		"name":        name,
		"description": "API functional fixture",
		"sort_order":  900,
	}))
	s.requireStatus(created, http.StatusCreated)
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			SortOrder   int    `json:"sort_order"`
		} `json:"data"`
	}
	s.decode(created, &envelope)
	if !envelope.Success || envelope.Data.ID == "" || envelope.Data.Name != name {
		s.t.Fatalf("unexpected department create response: %s", created.body)
	}
	itemPath := basePath + "/" + url.PathEscape(envelope.Data.ID)
	s.t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		s.request(cleanupCtx, http.MethodDelete, itemPath, s.token, nil)
	})

	got := s.request(ctx, http.MethodGet, itemPath, s.token, nil)
	s.requireStatus(got, http.StatusOK)
	s.decode(got, &envelope)
	if envelope.Data.Name != name {
		s.t.Fatalf("unexpected department get response: %s", got.body)
	}

	updatedName := name + " updated"
	updated := s.request(ctx, http.MethodPut, itemPath, s.token, mustJSON(s.t, map[string]any{
		"name": updatedName, "description": "updated", "sort_order": 901,
	}))
	s.requireStatus(updated, http.StatusOK)
	s.decode(updated, &envelope)
	if envelope.Data.Name != updatedName || envelope.Data.Description != "updated" || envelope.Data.SortOrder != 901 {
		s.t.Fatalf("unexpected department update response: %s", updated.body)
	}

	list := s.request(ctx, http.MethodGet, basePath, s.token, nil)
	s.requireStatus(list, http.StatusOK)
	if !containsID(s.t, list.body, envelope.Data.ID) {
		s.t.Fatalf("department list does not contain %q: %s", envelope.Data.ID, list.body)
	}
	tree := s.request(ctx, http.MethodGet, basePath+"/tree", s.token, nil)
	s.requireStatus(tree, http.StatusOK)
	if !bytes.Contains(tree.body, []byte(envelope.Data.ID)) {
		s.t.Fatalf("department tree does not contain %q: %s", envelope.Data.ID, tree.body)
	}

	deleted := s.request(ctx, http.MethodDelete, itemPath, s.token, nil)
	s.requireStatus(deleted, http.StatusNoContent)
	s.requireStatus(s.request(ctx, http.MethodGet, itemPath, s.token, nil), http.StatusNotFound)
}

func (s *apiSuite) testModelConfigCRUD(ctx context.Context) {
	basePath := "/api/v1/system/admin/model-configs"
	name := s.runID + " model"
	created := s.request(ctx, http.MethodPost, basePath, s.token, mustJSON(s.t, map[string]any{
		"name": name, "provider": "openai", "api_key": "functional-secret",
		"temperature": 0.2, "max_tokens": 512, "top_p": 0.8,
	}))
	s.requireStatus(created, http.StatusCreated)
	var model struct {
		ID               string  `json:"id"`
		Name             string  `json:"name"`
		Provider         string  `json:"provider"`
		APIKeyConfigured bool    `json:"api_key_configured"`
		Temperature      float64 `json:"temperature"`
		MaxTokens        int     `json:"max_tokens"`
		TopP             float64 `json:"top_p"`
	}
	s.decode(created, &model)
	if model.ID == "" || model.Name != name || !model.APIKeyConfigured {
		s.t.Fatalf("unexpected model config create response: %s", created.body)
	}
	itemPath := basePath + "/" + url.PathEscape(model.ID)
	s.t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		s.request(cleanupCtx, http.MethodDelete, itemPath, s.token, nil)
	})

	s.requireStatus(s.request(ctx, http.MethodGet, itemPath, s.token, nil), http.StatusOK)
	list := s.request(ctx, http.MethodGet, basePath, s.token, nil)
	s.requireStatus(list, http.StatusOK)
	if !containsID(s.t, list.body, model.ID) {
		s.t.Fatalf("model config list does not contain %q: %s", model.ID, list.body)
	}

	updatedName := name + " updated"
	updated := s.request(ctx, http.MethodPut, itemPath, s.token, mustJSON(s.t, map[string]any{
		"name": updatedName, "provider": "compatible", "temperature": 0.4, "max_tokens": 1024, "top_p": 0.9,
	}))
	s.requireStatus(updated, http.StatusOK)
	s.decode(updated, &model)
	if model.Name != updatedName || model.Provider != "compatible" || !model.APIKeyConfigured || model.MaxTokens != 1024 {
		s.t.Fatalf("unexpected model config update response: %s", updated.body)
	}

	s.requireStatus(s.request(ctx, http.MethodDelete, itemPath, s.token, nil), http.StatusNoContent)
	s.requireStatus(s.request(ctx, http.MethodGet, itemPath, s.token, nil), http.StatusNotFound)
}

func (s *apiSuite) testSystemConfig(ctx context.Context) {
	const key = "auth.password.complexity"
	itemPath := "/api/v1/system/admin/settings/" + key
	originalResponse := s.request(ctx, http.MethodGet, itemPath, s.token, nil)
	s.requireStatus(originalResponse, http.StatusOK)
	var original systemSetting
	s.decode(originalResponse, &original)
	if original.Key != key || len(original.Value) == 0 {
		s.t.Fatalf("unexpected original system setting: %s", originalResponse.body)
	}

	s.t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if original.ID == 0 {
			s.request(cleanupCtx, http.MethodDelete, itemPath, s.token, nil)
			return
		}
		s.request(cleanupCtx, http.MethodPut, itemPath, s.token,
			mustJSON(s.t, map[string]any{"value": original.Value}))
	})

	var oldValue bool
	if err := json.Unmarshal(original.Value, &oldValue); err != nil {
		s.t.Fatalf("decode original setting value: %v", err)
	}
	updated := s.request(ctx, http.MethodPut, itemPath, s.token,
		mustJSON(s.t, map[string]any{"value": !oldValue}))
	s.requireStatus(updated, http.StatusOK)
	var changed systemSetting
	s.decode(updated, &changed)
	var changedValue bool
	if err := json.Unmarshal(changed.Value, &changedValue); err != nil || changedValue == oldValue {
		s.t.Fatalf("system setting did not change: %s", updated.body)
	}

	list := s.request(ctx, http.MethodGet, "/api/v1/system/admin/settings", s.token, nil)
	s.requireStatus(list, http.StatusOK)
	if !containsKey(s.t, list.body, key) {
		s.t.Fatalf("system settings list does not contain %q: %s", key, list.body)
	}

	s.requireStatus(s.request(ctx, http.MethodDelete, itemPath, s.token, nil), http.StatusOK)
	reset := s.request(ctx, http.MethodGet, itemPath, s.token, nil)
	s.requireStatus(reset, http.StatusOK)
	var resetSetting systemSetting
	s.decode(reset, &resetSetting)
	if resetSetting.Key != key {
		s.t.Fatalf("reset setting is unavailable: %s", reset.body)
	}

	if original.ID != 0 {
		restored := s.request(ctx, http.MethodPut, itemPath, s.token,
			mustJSON(s.t, map[string]any{"value": original.Value}))
		s.requireStatus(restored, http.StatusOK)
	}
}

func (s *apiSuite) testTagCRUD(ctx context.Context) {
	basePath := "/api/v1/system/admin/tag-dictionary"
	dimensions := s.request(ctx, http.MethodGet, "/api/v1/system/admin/tag-dimensions", s.token, nil)
	s.requireStatus(dimensions, http.StatusOK)
	var dimensionRows []struct {
		ID string `json:"id"`
	}
	s.decode(dimensions, &dimensionRows)
	if len(dimensionRows) == 0 || dimensionRows[0].ID == "" {
		s.t.Fatalf("no tag dimensions are configured: %s", dimensions.body)
	}

	name := s.runID + " tag"
	created := s.request(ctx, http.MethodPost, basePath, s.token, mustJSON(s.t, map[string]any{
		"dimension_id": dimensionRows[0].ID, "name": name, "color": "#4455AA", "sort_order": 900,
	}))
	s.requireStatus(created, http.StatusCreated)
	var tag struct {
		ID          string `json:"id"`
		DimensionID string `json:"dimension_id"`
		Name        string `json:"name"`
		Color       string `json:"color"`
		SortOrder   int    `json:"sort_order"`
	}
	s.decode(created, &tag)
	if tag.ID == "" || tag.Name != name || tag.DimensionID != dimensionRows[0].ID {
		s.t.Fatalf("unexpected tag create response: %s", created.body)
	}
	itemPath := basePath + "/" + url.PathEscape(tag.ID)
	s.t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		s.request(cleanupCtx, http.MethodDelete, itemPath, s.token, nil)
	})

	s.requireStatus(s.request(ctx, http.MethodGet, itemPath, s.token, nil), http.StatusOK)
	list := s.request(ctx, http.MethodGet, basePath, s.token, nil)
	s.requireStatus(list, http.StatusOK)
	if !containsID(s.t, list.body, tag.ID) {
		s.t.Fatalf("tag list does not contain %q: %s", tag.ID, list.body)
	}

	updatedName := name + " updated"
	updated := s.request(ctx, http.MethodPut, itemPath, s.token, mustJSON(s.t, map[string]any{
		"dimension_id": dimensionRows[0].ID, "name": updatedName, "color": "#AA5544", "sort_order": 901,
	}))
	s.requireStatus(updated, http.StatusOK)
	s.decode(updated, &tag)
	if tag.Name != updatedName || tag.Color != "#AA5544" || tag.SortOrder != 901 {
		s.t.Fatalf("unexpected tag update response: %s", updated.body)
	}

	s.requireStatus(s.request(ctx, http.MethodDelete, itemPath, s.token, nil), http.StatusNoContent)
	s.requireStatus(s.request(ctx, http.MethodGet, itemPath, s.token, nil), http.StatusNotFound)
}

func (s *apiSuite) testBackups(ctx context.Context) {
	basePath := "/api/v1/system/admin/backups"
	scheduleResponse := s.request(ctx, http.MethodGet, basePath+"/schedule", s.token, nil)
	s.requireStatus(scheduleResponse, http.StatusOK)
	var originalSchedule backupSchedule
	s.decode(scheduleResponse, &originalSchedule)
	s.t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		s.request(cleanupCtx, http.MethodPut, basePath+"/schedule", s.token, mustJSON(s.t, originalSchedule))
	})

	disabledSchedule := originalSchedule
	disabledSchedule.Enabled = false
	updatedSchedule := s.request(ctx, http.MethodPut, basePath+"/schedule", s.token, mustJSON(s.t, disabledSchedule))
	s.requireStatus(updatedSchedule, http.StatusOK)
	var gotSchedule backupSchedule
	s.decode(updatedSchedule, &gotSchedule)
	if gotSchedule != disabledSchedule {
		s.t.Fatalf("backup schedule update mismatch: got %+v want %+v", gotSchedule, disabledSchedule)
	}

	created := s.request(ctx, http.MethodPost, basePath, s.token, nil)
	s.requireStatus(created, http.StatusAccepted)
	var backup backupRecord
	s.decode(created, &backup)
	if backup.ID == "" {
		s.t.Fatalf("backup create response is missing ID: %s", created.body)
	}
	itemPath := basePath + "/" + url.PathEscape(backup.ID)
	s.t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		s.request(cleanupCtx, http.MethodDelete, itemPath, s.token, nil)
	})

	deadline := time.Now().Add(functionalTimeout)
	for time.Now().Before(deadline) {
		got := s.request(ctx, http.MethodGet, itemPath, s.token, nil)
		s.requireStatus(got, http.StatusOK)
		s.decode(got, &backup)
		switch backup.Status {
		case "succeeded":
			if backup.SizeBytes <= 0 || len(backup.ChecksumSHA256) != 64 {
				s.t.Fatalf("successful backup has invalid metadata: %s", got.body)
			}
			goto ready
		case "failed":
			s.t.Fatalf("backup failed: %s", backup.ErrorMessage)
		}
		select {
		case <-ctx.Done():
			s.t.Fatal(ctx.Err())
		case <-time.After(functionalPollInterval):
		}
	}
	s.t.Fatal("timed out waiting for backup")

ready:
	list := s.request(ctx, http.MethodGet, basePath+"?limit=10&offset=0", s.token, nil)
	s.requireStatus(list, http.StatusOK)
	if !containsID(s.t, list.body, backup.ID) {
		s.t.Fatalf("backup list does not contain %q: %s", backup.ID, list.body)
	}

	download := s.request(ctx, http.MethodGet, itemPath+"/download", s.token, nil)
	s.requireStatus(download, http.StatusOK)
	if int64(len(download.body)) != backup.SizeBytes {
		s.t.Fatalf("download size: got %d, want %d", len(download.body), backup.SizeBytes)
	}
	digest := sha256.Sum256(download.body)
	if hex.EncodeToString(digest[:]) != backup.ChecksumSHA256 {
		s.t.Fatal("download checksum does not match backup record")
	}
	if download.header.Get("Content-Type") != "application/octet-stream" {
		s.t.Fatalf("unexpected backup content type: %q", download.header.Get("Content-Type"))
	}

	invalidRestore := s.request(ctx, http.MethodPost, itemPath+"/restore", s.token,
		mustJSON(s.t, map[string]any{"confirmation": "RESTORE wrong-backup"}))
	s.requireStatus(invalidRestore, http.StatusBadRequest)

	s.requireStatus(s.request(ctx, http.MethodDelete, itemPath, s.token, nil), http.StatusNoContent)
	s.requireStatus(s.request(ctx, http.MethodGet, itemPath, s.token, nil), http.StatusNotFound)

	restoredSchedule := s.request(ctx, http.MethodPut, basePath+"/schedule", s.token, mustJSON(s.t, originalSchedule))
	s.requireStatus(restoredSchedule, http.StatusOK)
}

func (s *apiSuite) testAuditLogQuery(ctx context.Context) {
	start := time.Now().UTC().Add(-functionalTimeout).Format(time.RFC3339)
	query := url.Values{}
	query.Set("action", "system.setting_changed")
	query.Set("outcome", "success")
	query.Set("user", s.userID)
	query.Set("start_time", start)
	query.Set("end_time", time.Now().UTC().Add(time.Minute).Format(time.RFC3339))
	query.Set("limit", "100")
	path := "/api/v1/system/admin/audit-log?" + query.Encode()

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		resp := s.request(ctx, http.MethodGet, path, s.token, nil)
		s.requireStatus(resp, http.StatusOK)
		var page struct {
			Success    bool   `json:"success"`
			NextCursor uint64 `json:"next_cursor"`
			Data       []struct {
				ID          uint64 `json:"id"`
				ActorUserID string `json:"actor_user_id"`
				Action      string `json:"action"`
				Outcome     string `json:"outcome"`
				Details     struct {
					Key string `json:"key"`
				} `json:"details"`
			} `json:"data"`
		}
		s.decode(resp, &page)
		if !page.Success {
			s.t.Fatalf("audit query did not succeed: %s", resp.body)
		}
		for _, entry := range page.Data {
			if entry.ActorUserID == s.userID && entry.Action == "system.setting_changed" &&
				entry.Outcome == "success" && entry.Details.Key == "auth.password.complexity" {
				cursorQuery := url.Values{}
				cursorQuery.Set("after_id", strconv.FormatUint(entry.ID, 10))
				cursorQuery.Set("limit", "1")
				cursor := s.request(ctx, http.MethodGet, "/api/v1/system/admin/audit-log?"+cursorQuery.Encode(), s.token, nil)
				s.requireStatus(cursor, http.StatusOK)
				return
			}
		}
		select {
		case <-ctx.Done():
			s.t.Fatal(ctx.Err())
		case <-time.After(functionalPollInterval):
		}
	}
	s.t.Fatal("system setting audit entry was not returned by filtered query")
}

func (s *apiSuite) request(ctx context.Context, method, path, token string, body []byte) apiResponse {
	s.t.Helper()
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, reader)
	if err != nil {
		s.t.Fatalf("create %s %s request: %v", method, path, err)
	}
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.client.Do(req)
	if err != nil {
		s.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		s.t.Fatalf("read %s %s response: %v", method, path, err)
	}
	return apiResponse{status: resp.StatusCode, header: resp.Header.Clone(), body: responseBody}
}

func (s *apiSuite) requireStatus(resp apiResponse, expected int) {
	s.t.Helper()
	if resp.status != expected {
		s.t.Fatalf("HTTP status: got %d, want %d; body=%s", resp.status, expected, resp.body)
	}
}

func (s *apiSuite) decode(resp apiResponse, target any) {
	s.t.Helper()
	if err := json.Unmarshal(resp.body, target); err != nil {
		s.t.Fatalf("decode response: %v; body=%s", err, resp.body)
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("encode JSON: %v", err)
	}
	return body
}

func jsonPathBool(t *testing.T, body []byte, key string) bool {
	t.Helper()
	var object map[string]json.RawMessage
	if err := json.Unmarshal(body, &object); err != nil {
		t.Fatalf("decode JSON object: %v", err)
	}
	var value bool
	if err := json.Unmarshal(object[key], &value); err != nil {
		t.Fatalf("decode boolean %q: %v", key, err)
	}
	return value
}

func containsID(t *testing.T, body []byte, id string) bool {
	t.Helper()
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatalf("decode response for ID search: %v", err)
	}
	return containsStringField(value, "id", id)
}

func containsKey(t *testing.T, body []byte, key string) bool {
	t.Helper()
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatalf("decode response for key search: %v", err)
	}
	return containsStringField(value, "key", key)
}

func containsStringField(value any, field, expected string) bool {
	switch typed := value.(type) {
	case map[string]any:
		if actual, ok := typed[field].(string); ok && actual == expected {
			return true
		}
		for _, child := range typed {
			if containsStringField(child, field, expected) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if containsStringField(child, field, expected) {
				return true
			}
		}
	}
	return false
}
