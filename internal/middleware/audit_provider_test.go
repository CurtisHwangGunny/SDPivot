package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type operationAuditCapture struct {
	interfaces.AuditLogService
	entries []*types.AuditLog
}

func (s *operationAuditCapture) Log(_ context.Context, entry *types.AuditLog) error {
	s.entries = append(s.entries, entry)
	return nil
}

func TestAuditAdminOperationCapturesSuccessfulMutation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	audit := &operationAuditCapture{}
	r := gin.New()
	r.Use(AuditServiceProvider(audit))
	r.Use(func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), types.TenantIDContextKey, uint64(7))
		ctx = context.WithValue(ctx, types.UserIDContextKey, "u-admin")
		ctx = context.WithValue(ctx, types.TenantRoleContextKey, types.TenantRoleAdmin)
		c.Request = c.Request.WithContext(ctx)
	})
	r.PUT("/models/:id", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
		AuditAdminOperation(c, types.TenantRoleAdmin)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/models/m1", nil)
	req.RemoteAddr = "192.0.2.10:1234"
	r.ServeHTTP(w, req)

	if len(audit.entries) != 1 {
		t.Fatalf("expected one audit event, got %d", len(audit.entries))
	}
	got := audit.entries[0]
	if got.Action != types.AuditActionAdminOperation || got.TenantID != 7 || got.ActorUserID != "u-admin" || got.TargetID != "m1" || got.IPAddress != "192.0.2.10" {
		t.Fatalf("unexpected audit event: %+v", got)
	}
}
