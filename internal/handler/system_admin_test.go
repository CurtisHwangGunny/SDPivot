package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apprepo "github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type systemAdminUserServiceStub struct {
	interfaces.UserService
	getByID   func(context.Context, string) (*types.User, error)
	getByEmail func(context.Context, string) (*types.User, error)
	update     func(context.Context, *types.User) error
	revoke     func(context.Context, string, string) (*types.User, error)
	list       func(context.Context, int, int) ([]*types.User, int64, error)
}

func (s *systemAdminUserServiceStub) GetUserByID(ctx context.Context, id string) (*types.User, error) {
	return s.getByID(ctx, id)
}

func (s *systemAdminUserServiceStub) GetUserByEmail(ctx context.Context, email string) (*types.User, error) {
	return s.getByEmail(ctx, email)
}

func (s *systemAdminUserServiceStub) UpdateUser(ctx context.Context, user *types.User) error {
	return s.update(ctx, user)
}

func (s *systemAdminUserServiceStub) RevokeSystemAdmin(ctx context.Context, userID, actorID string) (*types.User, error) {
	return s.revoke(ctx, userID, actorID)
}

func (s *systemAdminUserServiceStub) ListSystemAdmins(ctx context.Context, offset, limit int) ([]*types.User, int64, error) {
	return s.list(ctx, offset, limit)
}

type systemAdminAuditStub struct {
	interfaces.AuditLogService
	calls int
}

func (s *systemAdminAuditStub) Log(_ context.Context, _ *types.AuditLog) error {
	s.calls++
	return nil
}

func newSystemAdminTestRouter(userSvc interfaces.UserService, auditSvc interfaces.AuditLogService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := &SystemHandler{userSvc: userSvc, auditSvc: auditSvc}
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.POST("/system/admin/promote", h.PromoteUserToSystemAdmin)
	r.POST("/system/admin/revoke", h.RevokeSystemAdmin)
	r.GET("/system/admin/list", h.ListSystemAdmins)
	return r
}

func performSystemAdminRequest(t *testing.T, r http.Handler, method, path, body, actorID string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if actorID != "" {
		req = req.WithContext(context.WithValue(req.Context(), types.UserIDContextKey, actorID))
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestSystemAdminPromoteHTTP(t *testing.T) {
	t.Run("promotes by trimmed email", func(t *testing.T) {
		user := &types.User{ID: "target", Username: "target-user", Email: "target@example.com"}
		audit := &systemAdminAuditStub{}
		updated := false
		svc := &systemAdminUserServiceStub{
			getByEmail: func(_ context.Context, email string) (*types.User, error) {
				if email != user.Email {
					t.Fatalf("expected trimmed email %q, got %q", user.Email, email)
				}
				return user, nil
			},
			update: func(_ context.Context, got *types.User) error {
				updated = true
				if got != user || !got.IsSystemAdmin {
					t.Fatalf("expected promoted target, got %+v", got)
				}
				return nil
			},
		}

		w := performSystemAdminRequest(t, newSystemAdminTestRouter(svc, audit), http.MethodPost,
			"/system/admin/promote", `{"email":"  target@example.com  "}`, "actor")
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
		}
		var got types.UserInfo
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if !updated || got.ID != user.ID || !got.IsSystemAdmin {
			t.Fatalf("unexpected promote result: updated=%v response=%+v", updated, got)
		}
		if audit.calls != 1 {
			t.Fatalf("expected one audit attempt, got %d", audit.calls)
		}
	})

	t.Run("user id wins and existing admin is a no-op", func(t *testing.T) {
		user := &types.User{ID: "target", Email: "actual@example.com", IsSystemAdmin: true}
		audit := &systemAdminAuditStub{}
		svc := &systemAdminUserServiceStub{
			getByID: func(_ context.Context, id string) (*types.User, error) {
				if id != user.ID {
					t.Fatalf("expected trimmed id %q, got %q", user.ID, id)
				}
				return user, nil
			},
			getByEmail: func(_ context.Context, email string) (*types.User, error) {
				t.Fatalf("email lookup must not run when user_id is supplied: %q", email)
				return nil, nil
			},
			update: func(_ context.Context, _ *types.User) error {
				t.Fatal("existing admin must not be updated")
				return nil
			},
		}

		w := performSystemAdminRequest(t, newSystemAdminTestRouter(svc, audit), http.MethodPost,
			"/system/admin/promote", `{"user_id":" target ","email":"other@example.com"}`, "actor")
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"is_system_admin":true`) {
			t.Fatalf("expected idempotent 200, got %d body=%s", w.Code, w.Body.String())
		}
		if audit.calls != 1 {
			t.Fatalf("expected one audit attempt, got %d", audit.calls)
		}
	})

	t.Run("validates selector and maps lookup and update failures", func(t *testing.T) {
		tests := []struct {
			name       string
			body       string
			svc        *systemAdminUserServiceStub
			wantStatus int
			wantError  string
		}{
			{
				name:       "missing selector",
				body:       `{"user_id":" ","email":" "}`,
				svc:        &systemAdminUserServiceStub{},
				wantStatus: http.StatusBadRequest,
				wantError:  "Either user_id or email is required",
			},
			{
				name: "lookup failure",
				body: `{"email":"missing@example.com"}`,
				svc: &systemAdminUserServiceStub{getByEmail: func(context.Context, string) (*types.User, error) {
					return nil, errors.New("database unavailable")
				}},
				wantStatus: http.StatusNotFound,
				wantError:  "User not found",
			},
			{
				name: "update failure",
				body: `{"user_id":"target"}`,
				svc: &systemAdminUserServiceStub{
					getByID: func(context.Context, string) (*types.User, error) {
						return &types.User{ID: "target"}, nil
					},
					update: func(context.Context, *types.User) error {
						return errors.New("write failed")
					},
				},
				wantStatus: http.StatusInternalServerError,
				wantError:  "Failed to promote user",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				w := performSystemAdminRequest(t, newSystemAdminTestRouter(tt.svc, nil), http.MethodPost,
					"/system/admin/promote", tt.body, "actor")
				if w.Code != tt.wantStatus || !strings.Contains(w.Body.String(), tt.wantError) {
					t.Fatalf("expected %d containing %q, got %d body=%s", tt.wantStatus, tt.wantError, w.Code, w.Body.String())
				}
			})
		}
	})
}

func TestSystemAdminRevokeHTTP(t *testing.T) {
	t.Run("revokes using the caller id", func(t *testing.T) {
		audit := &systemAdminAuditStub{}
		svc := &systemAdminUserServiceStub{revoke: func(_ context.Context, userID, actorID string) (*types.User, error) {
			if userID != "target" || actorID != "actor" {
				t.Fatalf("unexpected revoke args: target=%q actor=%q", userID, actorID)
			}
			return &types.User{ID: userID, Username: "target-user", Email: "target@example.com", IsSystemAdmin: false}, nil
		}}

		w := performSystemAdminRequest(t, newSystemAdminTestRouter(svc, audit), http.MethodPost,
			"/system/admin/revoke", `{"user_id":"target"}`, "actor")
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"is_system_admin":false`) {
			t.Fatalf("expected successful revoke, got %d body=%s", w.Code, w.Body.String())
		}
		if audit.calls != 1 {
			t.Fatalf("expected one audit attempt, got %d", audit.calls)
		}
	})

	t.Run("already non-admin is idempotent", func(t *testing.T) {
		audit := &systemAdminAuditStub{}
		svc := &systemAdminUserServiceStub{revoke: func(context.Context, string, string) (*types.User, error) {
			return &types.User{ID: "target", IsSystemAdmin: false}, apprepo.ErrUserNotSystemAdmin
		}}

		w := performSystemAdminRequest(t, newSystemAdminTestRouter(svc, audit), http.MethodPost,
			"/system/admin/revoke", `{"user_id":"target"}`, "actor")
		if w.Code != http.StatusOK {
			t.Fatalf("expected idempotent 200, got %d body=%s", w.Code, w.Body.String())
		}
		if audit.calls != 1 {
			t.Fatalf("expected one audit attempt, got %d", audit.calls)
		}
	})

	t.Run("validates input and maps safety and service errors", func(t *testing.T) {
		tests := []struct {
			name       string
			body       string
			err        error
			wantStatus int
			wantError  string
			wantCall   bool
		}{
			{name: "missing user id", body: `{}`, wantStatus: http.StatusBadRequest, wantError: "Invalid request"},
			{name: "self revoke", body: `{"user_id":"actor"}`, err: apprepo.ErrCannotRevokeSelf, wantStatus: http.StatusBadRequest, wantError: "Cannot revoke your own", wantCall: true},
			{name: "last admin", body: `{"user_id":"target"}`, err: apprepo.ErrLastSystemAdmin, wantStatus: http.StatusBadRequest, wantError: "last remaining system administrator", wantCall: true},
			{name: "unknown user", body: `{"user_id":"missing"}`, err: apprepo.ErrUserNotFound, wantStatus: http.StatusNotFound, wantError: "User not found", wantCall: true},
			{name: "unexpected failure", body: `{"user_id":"target"}`, err: errors.New("database unavailable"), wantStatus: http.StatusInternalServerError, wantError: "Failed to revoke system admin privileges", wantCall: true},
			{name: "invalid idempotent result", body: `{"user_id":"target"}`, err: apprepo.ErrUserNotSystemAdmin, wantStatus: http.StatusInternalServerError, wantError: "Failed to revoke system admin privileges", wantCall: true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				called := false
				svc := &systemAdminUserServiceStub{revoke: func(context.Context, string, string) (*types.User, error) {
					called = true
					return nil, tt.err
				}}
				w := performSystemAdminRequest(t, newSystemAdminTestRouter(svc, nil), http.MethodPost,
					"/system/admin/revoke", tt.body, "actor")
				if w.Code != tt.wantStatus || !strings.Contains(w.Body.String(), tt.wantError) {
					t.Fatalf("expected %d containing %q, got %d body=%s", tt.wantStatus, tt.wantError, w.Code, w.Body.String())
				}
				if called != tt.wantCall {
					t.Fatalf("service call mismatch: got %v want %v", called, tt.wantCall)
				}
			})
		}
	})
}

func TestSystemAdminListHTTP(t *testing.T) {
	t.Run("returns paginated admins and caps the limit", func(t *testing.T) {
		svc := &systemAdminUserServiceStub{list: func(_ context.Context, offset, limit int) ([]*types.User, int64, error) {
			if offset != 7 || limit != 200 {
				t.Fatalf("unexpected pagination: offset=%d limit=%d", offset, limit)
			}
			return []*types.User{{ID: "admin-1", Email: "admin@example.com", IsSystemAdmin: true}}, 9, nil
		}}

		w := performSystemAdminRequest(t, newSystemAdminTestRouter(svc, nil), http.MethodGet,
			"/system/admin/list?offset=7&limit=999", "", "actor")
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
		}
		var got ListSystemAdminsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.Total != 9 || len(got.Admins) != 1 || got.Admins[0].ID != "admin-1" {
			t.Fatalf("unexpected list response: %+v", got)
		}
	})

	t.Run("malformed pagination falls back and nil results serialize as an array", func(t *testing.T) {
		svc := &systemAdminUserServiceStub{list: func(_ context.Context, offset, limit int) ([]*types.User, int64, error) {
			if offset != 0 || limit != 50 {
				t.Fatalf("expected default pagination, got offset=%d limit=%d", offset, limit)
			}
			return nil, 0, nil
		}}

		w := performSystemAdminRequest(t, newSystemAdminTestRouter(svc, nil), http.MethodGet,
			"/system/admin/list?offset=-1&limit=bad", "", "actor")
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"admins":[]`) {
			t.Fatalf("expected empty admin array, got %d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("service failure returns 500", func(t *testing.T) {
		svc := &systemAdminUserServiceStub{list: func(context.Context, int, int) ([]*types.User, int64, error) {
			return nil, 0, errors.New("database unavailable")
		}}
		w := performSystemAdminRequest(t, newSystemAdminTestRouter(svc, nil), http.MethodGet,
			"/system/admin/list", "", "actor")
		if w.Code != http.StatusInternalServerError || !strings.Contains(w.Body.String(), "Failed to list system admins") {
			t.Fatalf("expected list failure, got %d body=%s", w.Code, w.Body.String())
		}
	})
}
