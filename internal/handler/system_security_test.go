package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type securitySettingServiceStub struct {
	interfaces.SystemSettingService
	values map[string]any
}

func (s *securitySettingServiceStub) GetInt(_ context.Context, key, _ string, def int64) int64 {
	if value, ok := s.values[key].(int64); ok {
		return value
	}
	return def
}

func (s *securitySettingServiceStub) GetBool(_ context.Context, key, _ string, def bool) bool {
	if value, ok := s.values[key].(bool); ok {
		return value
	}
	return def
}

func (s *securitySettingServiceStub) GetStringList(_ context.Context, key, _ string, def []string) []string {
	if value, ok := s.values[key].([]string); ok {
		return append([]string(nil), value...)
	}
	return def
}

func (s *securitySettingServiceStub) Update(_ context.Context, key string, value any) (*types.SystemSetting, error) {
	s.values[key] = value
	return &types.SystemSetting{Key: key}, nil
}

func newSecuritySettingsRouter(settings interfaces.SystemSettingService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := &SystemHandler{systemSettingSvc: settings}
	r := gin.New()
	r.GET("/security/ip-whitelist", h.ListIPWhitelist)
	r.POST("/security/ip-whitelist", h.CreateIPWhitelistEntry)
	r.PUT("/security/ip-whitelist", h.UpdateIPWhitelist)
	r.DELETE("/security/ip-whitelist", h.DeleteIPWhitelistEntry)
	r.GET("/security/password-policy", h.GetPasswordPolicy)
	r.PUT("/security/password-policy", h.UpdatePasswordPolicy)
	r.GET("/security/login-lockout", h.GetLoginLockout)
	r.PUT("/security/login-lockout", h.UpdateLoginLockout)
	return r
}

func securityRequest(t *testing.T, r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestIPWhitelistCRUD(t *testing.T) {
	settings := &securitySettingServiceStub{values: map[string]any{
		ipWhitelistSettingKey: []string{"10.0.0.1"},
	}}
	r := newSecuritySettingsRouter(settings)

	w := securityRequest(t, r, http.MethodPost, "/security/ip-whitelist", `{"entry":"10.0.0.0/24"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d body=%s", w.Code, w.Body.String())
	}
	entries := settings.values[ipWhitelistSettingKey].([]string)
	if len(entries) != 2 || entries[1] != "10.0.0.0/24" {
		t.Fatalf("create stored unexpected entries: %#v", entries)
	}

	w = securityRequest(t, r, http.MethodDelete, "/security/ip-whitelist", `{"entry":"10.0.0.0/24"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	entries = settings.values[ipWhitelistSettingKey].([]string)
	if len(entries) != 1 || entries[0] != "10.0.0.1" {
		t.Fatalf("delete stored unexpected entries: %#v", entries)
	}
}

func TestSecurityPolicyEndpoints(t *testing.T) {
	settings := &securitySettingServiceStub{values: map[string]any{}}
	r := newSecuritySettingsRouter(settings)

	w := securityRequest(t, r, http.MethodGet, "/security/password-policy", "")
	if w.Code != http.StatusOK {
		t.Fatalf("get password policy: expected 200, got %d", w.Code)
	}
	var passwordPolicy PasswordPolicyResponse
	if err := json.Unmarshal(w.Body.Bytes(), &passwordPolicy); err != nil {
		t.Fatal(err)
	}
	if passwordPolicy.MinLength != 8 || !passwordPolicy.Complexity || passwordPolicy.RotationDays != 90 {
		t.Fatalf("unexpected password defaults: %+v", passwordPolicy)
	}

	w = securityRequest(t, r, http.MethodPut, "/security/password-policy", `{"min_length":12,"complexity":true,"rotation_days":90}`)
	if w.Code != http.StatusOK {
		t.Fatalf("update password policy: expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if settings.values["auth.password.min_length"] != int64(12) || settings.values["auth.password.rotation_days"] != int64(90) {
		t.Fatalf("unexpected password settings: %#v", settings.values)
	}

	w = securityRequest(t, r, http.MethodPut, "/security/login-lockout", `{"max_failed_attempts":5,"lockout_minutes":30}`)
	if w.Code != http.StatusOK {
		t.Fatalf("update login lockout: expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if settings.values["auth.login.max_failed_attempts"] != int64(5) || settings.values["auth.login.lockout_minutes"] != int64(30) {
		t.Fatalf("unexpected lockout settings: %#v", settings.values)
	}
}
