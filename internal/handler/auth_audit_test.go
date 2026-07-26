package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type auditLoginUserService struct {
	interfaces.UserService
	response *types.LoginResponse
}

func (s *auditLoginUserService) Login(context.Context, *types.LoginRequest) (*types.LoginResponse, error) {
	return s.response, nil
}

type auditCaptureService struct {
	interfaces.AuditLogService
	entries []*types.AuditLog
}

func (s *auditCaptureService) Log(_ context.Context, entry *types.AuditLog) error {
	s.entries = append(s.entries, entry)
	return nil
}

func TestLoginAuditRecordsSuccessfulUserTenantAndIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	audit := &auditCaptureService{}
	h := NewAuthHandler(
		&config.Config{Auth: &config.AuthConfig{}},
		&auditLoginUserService{response: &types.LoginResponse{
			Success:      true,
			User:         &types.User{ID: "u1"},
			ActiveTenant: &types.Tenant{ID: 7},
		}},
		nil, nil, nil, audit,
	)
	r := gin.New()
	r.POST("/auth/login", h.Login)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"a@example.com","password":"secret"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.0.2.8:1234"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if len(audit.entries) != 1 {
		t.Fatalf("expected one audit event, got %d", len(audit.entries))
	}
	got := audit.entries[0]
	if got.Action != types.AuditActionLogin || got.Outcome != types.AuditOutcomeSuccess || got.ActorUserID != "u1" || got.TenantID != 7 || got.IPAddress != "192.0.2.8" {
		t.Fatalf("unexpected audit event: %+v", got)
	}
	var details map[string]string
	if err := json.Unmarshal(got.Details, &details); err != nil || details["mechanism"] != "password" {
		t.Fatalf("unexpected details: %s err=%v", got.Details, err)
	}
}

func TestLoginAuditDoesNotPersistSubmittedIdentityOnFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	audit := &auditCaptureService{}
	h := NewAuthHandler(
		&config.Config{Auth: &config.AuthConfig{}},
		&auditLoginUserService{response: &types.LoginResponse{Success: false, Message: "invalid"}},
		nil, nil, nil, audit,
	)
	r := gin.New()
	r.POST("/auth/login", h.Login)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"secret@example.com","password":"secret"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized || len(audit.entries) != 1 {
		t.Fatalf("status=%d audit_entries=%d", w.Code, len(audit.entries))
	}
	got := audit.entries[0]
	if got.ActorUserID != "" || strings.Contains(string(got.Details), "secret@example.com") {
		t.Fatalf("failed login leaked submitted identity: %+v", got)
	}
}
