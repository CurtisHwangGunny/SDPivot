package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/service"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type stubDepartmentService struct {
	interfaces.DepartmentService
	create func(context.Context, uint64, *types.CreateDepartmentRequest) (*types.Department, error)
	get    func(context.Context, uint64, string) (*types.Department, error)
	list   func(context.Context, uint64) ([]*types.Department, error)
	tree   func(context.Context, uint64) ([]*types.DepartmentTreeNode, error)
	update func(context.Context, uint64, string, *types.UpdateDepartmentRequest) (*types.Department, error)
	delete func(context.Context, uint64, string) error
}

func (s *stubDepartmentService) Create(ctx context.Context, tenantID uint64, req *types.CreateDepartmentRequest) (*types.Department, error) {
	return s.create(ctx, tenantID, req)
}

func (s *stubDepartmentService) Get(ctx context.Context, tenantID uint64, id string) (*types.Department, error) {
	return s.get(ctx, tenantID, id)
}

func (s *stubDepartmentService) List(ctx context.Context, tenantID uint64) ([]*types.Department, error) {
	return s.list(ctx, tenantID)
}

func (s *stubDepartmentService) Tree(ctx context.Context, tenantID uint64) ([]*types.DepartmentTreeNode, error) {
	return s.tree(ctx, tenantID)
}

func (s *stubDepartmentService) Update(ctx context.Context, tenantID uint64, id string, req *types.UpdateDepartmentRequest) (*types.Department, error) {
	return s.update(ctx, tenantID, id, req)
}

func (s *stubDepartmentService) Delete(ctx context.Context, tenantID uint64, id string) error {
	return s.delete(ctx, tenantID, id)
}

func departmentTestRouter(svc interfaces.DepartmentService) *gin.Engine {
	return departmentTestRouterWithContext(svc, "", "")
}

func departmentTestRouterWithContext(svc interfaces.DepartmentService, role, departmentID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	if role != "" {
		r.Use(func(c *gin.Context) {
			c.Set("role", role)
			c.Set("department_id", departmentID)
			c.Next()
		})
	}
	h := NewDepartmentHandler(svc)
	r.GET("/tenants/:id/departments", h.List)
	r.GET("/tenants/:id/departments/tree", h.Tree)
	r.GET("/tenants/:id/departments/:department_id", h.Get)
	r.POST("/tenants/:id/departments", h.Create)
	r.PUT("/tenants/:id/departments/:department_id", h.Update)
	r.DELETE("/tenants/:id/departments/:department_id", h.Delete)
	return r
}

func TestDepartmentAdminSeesOnlyDepartmentScope(t *testing.T) {
	departments := []*types.Department{
		{ID: "engineering", TenantID: 42, Name: "Engineering"},
		{ID: "platform", TenantID: 42, ParentID: "engineering", Name: "Platform"},
		{ID: "sales", TenantID: 42, Name: "Sales"},
	}
	svc := &stubDepartmentService{list: func(context.Context, uint64) ([]*types.Department, error) {
		return departments, nil
	}}
	r := departmentTestRouterWithContext(svc, string(types.AccessRoleDepartmentAdmin), "engineering")
	w := departmentRequest(t, r, http.MethodGet, "/tenants/42/departments", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var response struct {
		Data []*types.Department `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Len(t, response.Data, 2)
	require.Equal(t, []string{"engineering", "platform"}, []string{response.Data[0].ID, response.Data[1].ID})
}

func departmentRequest(t *testing.T, r http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func departmentAssertError(t *testing.T, w *httptest.ResponseRecorder, status int, code apperrors.ErrorCode, message string) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status: got %d, want %d (body=%s)", w.Code, status, w.Body.String())
	}
	var response struct {
		Success bool `json:"success"`
		Error   struct {
			Code    apperrors.ErrorCode `json:"code"`
			Message string              `json:"message"`
			Details any                 `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode error response: %v (body=%s)", err, w.Body.String())
	}
	if response.Success || response.Error.Code != code || response.Error.Message != message {
		t.Fatalf("unexpected error envelope: %+v", response)
	}
}

func TestDepartmentHandler_HappyPaths(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		svc := &stubDepartmentService{list: func(_ context.Context, tenantID uint64) ([]*types.Department, error) {
			if tenantID != 42 {
				t.Fatalf("tenant ID: got %d, want 42", tenantID)
			}
			return []*types.Department{{ID: "engineering", TenantID: 42, Name: "Engineering"}}, nil
		}}
		w := departmentRequest(t, departmentTestRouter(svc), http.MethodGet, "/tenants/42/departments", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("status: got %d, want 200 (body=%s)", w.Code, w.Body.String())
		}
		var response struct {
			Success bool                `json:"success"`
			Data    []*types.Department `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if !response.Success || len(response.Data) != 1 || response.Data[0].Name != "Engineering" {
			t.Fatalf("unexpected response: %+v", response)
		}
	})

	t.Run("tree", func(t *testing.T) {
		svc := &stubDepartmentService{tree: func(_ context.Context, tenantID uint64) ([]*types.DepartmentTreeNode, error) {
			if tenantID != 42 {
				t.Fatalf("tenant ID: got %d, want 42", tenantID)
			}
			return []*types.DepartmentTreeNode{{
				Department: types.Department{ID: "root", TenantID: 42, Name: "Root"},
				Children: []*types.DepartmentTreeNode{{
					Department: types.Department{ID: "child", TenantID: 42, ParentID: "root", Name: "Child"},
					Children:   []*types.DepartmentTreeNode{},
				}},
			}}, nil
		}}
		w := departmentRequest(t, departmentTestRouter(svc), http.MethodGet, "/tenants/42/departments/tree", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("status: got %d, want 200 (body=%s)", w.Code, w.Body.String())
		}
		var response struct {
			Success bool                        `json:"success"`
			Data    []*types.DepartmentTreeNode `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if !response.Success || len(response.Data) != 1 || len(response.Data[0].Children) != 1 || response.Data[0].Children[0].ID != "child" {
			t.Fatalf("unexpected tree response: %+v", response)
		}
	})

	t.Run("get trims department ID", func(t *testing.T) {
		svc := &stubDepartmentService{get: func(_ context.Context, tenantID uint64, id string) (*types.Department, error) {
			if tenantID != 42 || id != "finance" {
				t.Fatalf("service arguments: tenant=%d id=%q", tenantID, id)
			}
			return &types.Department{ID: id, TenantID: tenantID, Name: "Finance"}, nil
		}}
		w := departmentRequest(t, departmentTestRouter(svc), http.MethodGet, "/tenants/42/departments/%20finance%20", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("status: got %d, want 200 (body=%s)", w.Code, w.Body.String())
		}
		var response struct {
			Success bool              `json:"success"`
			Data    *types.Department `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if !response.Success || response.Data == nil || response.Data.ID != "finance" {
			t.Fatalf("unexpected response: %+v", response)
		}
	})

	t.Run("create", func(t *testing.T) {
		svc := &stubDepartmentService{create: func(_ context.Context, tenantID uint64, req *types.CreateDepartmentRequest) (*types.Department, error) {
			if tenantID != 42 || req.ParentID != "root" || req.Name != "Platform" || req.Description != "Build systems" || req.SortOrder != 3 {
				t.Fatalf("unexpected create arguments: tenant=%d req=%+v", tenantID, req)
			}
			return &types.Department{ID: "platform", TenantID: tenantID, ParentID: req.ParentID, Name: req.Name}, nil
		}}
		body := types.CreateDepartmentRequest{ParentID: "root", Name: "Platform", Description: "Build systems", SortOrder: 3}
		w := departmentRequest(t, departmentTestRouter(svc), http.MethodPost, "/tenants/42/departments", body)
		if w.Code != http.StatusCreated {
			t.Fatalf("status: got %d, want 201 (body=%s)", w.Code, w.Body.String())
		}
		var response struct {
			Success bool              `json:"success"`
			Data    *types.Department `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if !response.Success || response.Data == nil || response.Data.ID != "platform" {
			t.Fatalf("unexpected response: %+v", response)
		}
	})

	t.Run("update", func(t *testing.T) {
		svc := &stubDepartmentService{update: func(_ context.Context, tenantID uint64, id string, req *types.UpdateDepartmentRequest) (*types.Department, error) {
			if tenantID != 42 || id != "platform" || req.Name == nil || *req.Name != "Infrastructure" || req.SortOrder == nil || *req.SortOrder != 7 {
				t.Fatalf("unexpected update arguments: tenant=%d id=%q req=%+v", tenantID, id, req)
			}
			return &types.Department{ID: id, TenantID: tenantID, Name: *req.Name, SortOrder: *req.SortOrder}, nil
		}}
		w := departmentRequest(t, departmentTestRouter(svc), http.MethodPut, "/tenants/42/departments/platform", map[string]any{
			"name": "Infrastructure", "sort_order": 7,
		})
		if w.Code != http.StatusOK {
			t.Fatalf("status: got %d, want 200 (body=%s)", w.Code, w.Body.String())
		}
		var response struct {
			Success bool              `json:"success"`
			Data    *types.Department `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if !response.Success || response.Data == nil || response.Data.Name != "Infrastructure" || response.Data.SortOrder != 7 {
			t.Fatalf("unexpected response: %+v", response)
		}
	})

	t.Run("delete trims department ID", func(t *testing.T) {
		svc := &stubDepartmentService{delete: func(_ context.Context, tenantID uint64, id string) error {
			if tenantID != 42 || id != "obsolete" {
				t.Fatalf("service arguments: tenant=%d id=%q", tenantID, id)
			}
			return nil
		}}
		w := departmentRequest(t, departmentTestRouter(svc), http.MethodDelete, "/tenants/42/departments/%20obsolete%20", nil)
		if w.Code != http.StatusNoContent || w.Body.Len() != 0 {
			t.Fatalf("expected empty 204 response, got status=%d body=%q", w.Code, w.Body.String())
		}
	})
}

func TestDepartmentHandler_Validation(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		path    string
		body    any
		message string
	}{
		{name: "non-numeric tenant ID", method: http.MethodGet, path: "/tenants/nope/departments", message: "tenant id must be a positive integer"},
		{name: "zero tenant ID", method: http.MethodGet, path: "/tenants/0/departments/tree", message: "tenant id must be a positive integer"},
		{name: "create requires name", method: http.MethodPost, path: "/tenants/42/departments", body: map[string]any{"description": "missing name"}, message: "invalid department data"},
		{name: "update rejects wrong field type", method: http.MethodPut, path: "/tenants/42/departments/platform", body: map[string]any{"sort_order": "first"}, message: "invalid department data"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := departmentRequest(t, departmentTestRouter(&stubDepartmentService{}), tt.method, tt.path, tt.body)
			departmentAssertError(t, w, http.StatusBadRequest, apperrors.ErrValidation, tt.message)
		})
	}
}

func TestDepartmentHandler_ErrorMappings(t *testing.T) {
	backendErr := errors.New("database unavailable")
	tests := []struct {
		name    string
		method  string
		path    string
		body    any
		svc     *stubDepartmentService
		status  int
		code    apperrors.ErrorCode
		message string
	}{
		{
			name: "not found", method: http.MethodGet, path: "/tenants/42/departments/missing",
			svc: &stubDepartmentService{get: func(context.Context, uint64, string) (*types.Department, error) {
				return nil, fmt.Errorf("lookup: %w", service.ErrDepartmentNotFound)
			}},
			status: http.StatusNotFound, code: apperrors.ErrNotFound, message: "department not found",
		},
		{
			name: "duplicate name conflict", method: http.MethodPost, path: "/tenants/42/departments", body: map[string]any{"name": "Engineering"},
			svc: &stubDepartmentService{create: func(context.Context, uint64, *types.CreateDepartmentRequest) (*types.Department, error) {
				return nil, service.ErrDepartmentConflict
			}},
			status: http.StatusConflict, code: apperrors.ErrConflict, message: service.ErrDepartmentConflict.Error(),
		},
		{
			name: "cycle validation", method: http.MethodPut, path: "/tenants/42/departments/root", body: map[string]any{"parent_id": "child"},
			svc: &stubDepartmentService{update: func(context.Context, uint64, string, *types.UpdateDepartmentRequest) (*types.Department, error) {
				return nil, fmt.Errorf("move rejected: %w", service.ErrDepartmentCycle)
			}},
			status: http.StatusBadRequest, code: apperrors.ErrValidation, message: "move rejected: " + service.ErrDepartmentCycle.Error(),
		},
		{
			name: "non-empty conflict", method: http.MethodDelete, path: "/tenants/42/departments/root",
			svc: &stubDepartmentService{delete: func(context.Context, uint64, string) error {
				return service.ErrDepartmentNotEmpty
			}},
			status: http.StatusConflict, code: apperrors.ErrConflict, message: service.ErrDepartmentNotEmpty.Error(),
		},
		{
			name: "unexpected list failure", method: http.MethodGet, path: "/tenants/42/departments",
			svc: &stubDepartmentService{list: func(context.Context, uint64) ([]*types.Department, error) {
				return nil, backendErr
			}},
			status: http.StatusInternalServerError, code: apperrors.ErrInternalServer, message: "failed to list departments",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := departmentRequest(t, departmentTestRouter(tt.svc), tt.method, tt.path, tt.body)
			departmentAssertError(t, w, tt.status, tt.code, tt.message)
		})
	}
}
