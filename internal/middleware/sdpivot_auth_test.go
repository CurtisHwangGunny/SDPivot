package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/auth"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSDPivotAuthRejectsInvalidAndExpiredTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := auth.DefaultJWTConfig("phase4-secret")
	manager := auth.NewJWTManager(config)
	r := gin.New()
	r.GET("/protected", SDPivotAuth(manager), func(c *gin.Context) { c.Status(http.StatusOK) })

	for name, token := range map[string]string{"invalid": "not-a-token"} {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}

	expiredConfig := config
	expiredConfig.AccessExpiry = -time.Minute
	expired, _, err := auth.NewJWTManager(expiredConfig).GenerateAccessToken("u1", 1, string(types.AccessRoleKnowledgeViewer), nil, false)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+expired)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSDPivotAuthEnforcesMustChangePasswordGate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := auth.NewJWTManager(auth.DefaultJWTConfig("must-change-password-secret"))
	mustChangeToken, _, err := manager.GenerateAccessToken("u1", 1, string(types.AccessRoleKnowledgeViewer), nil, true)
	require.NoError(t, err)
	normalToken, _, err := manager.GenerateAccessToken("u1", 1, string(types.AccessRoleKnowledgeViewer), nil, false)
	require.NoError(t, err)

	tests := []struct {
		name  string
		path  string
		token string
		want  int
	}{
		{name: "blocks other API", path: "/api/v1/sdp/spaces", token: mustChangeToken, want: http.StatusForbidden},
		{name: "allows current user", path: "/api/v1/sdp/auth/me", token: mustChangeToken, want: http.StatusOK},
		{name: "allows password change", path: "/api/v1/sdp/auth/change-password", token: mustChangeToken, want: http.StatusOK},
		{name: "allows logout", path: "/api/v1/sdp/auth/logout", token: mustChangeToken, want: http.StatusOK},
		{name: "allows refresh", path: "/api/v1/sdp/auth/refresh", token: mustChangeToken, want: http.StatusOK},
		{name: "allows compatibility prefix", path: "/api/v1/smartknora/auth/me", token: mustChangeToken, want: http.StatusOK},
		{name: "normal token is unrestricted", path: "/api/v1/sdp/spaces", token: normalToken, want: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(SDPivotAuth(manager))
			r.Handle(http.MethodGet, tt.path, func(c *gin.Context) { c.Status(http.StatusOK) })
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set("Authorization", "Bearer "+tt.token)
			r.ServeHTTP(w, req)
			assert.Equal(t, tt.want, w.Code)
			if tt.want == http.StatusForbidden {
				assert.JSONEq(t, `{"error":"must_change_password","message":"请先修改密码"}`, w.Body.String())
			}
		})
	}
}

func TestSDPivotAuthSetsDepartmentID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := auth.NewJWTManager(auth.DefaultJWTConfig("department-scope-secret"))
	departmentID := "engineering"
	token, _, err := manager.GenerateAccessToken("u1", 7, string(types.AccessRoleDepartmentAdmin), &departmentID, false)
	require.NoError(t, err)

	r := gin.New()
	r.GET("/protected", SDPivotAuth(manager), func(c *gin.Context) {
		assert.Equal(t, departmentID, GetDepartmentID(c))
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequirePermissionEnforcesViewerAndEditorBoundaries(t *testing.T) {
	tests := []struct {
		name       string
		role       types.AccessRole
		permission Permission
		want       int
	}{
		{"viewer can read", types.AccessRoleKnowledgeViewer, PermissionKnowledgeRead, http.StatusOK},
		{"viewer cannot edit", types.AccessRoleKnowledgeViewer, PermissionKnowledgeWrite, http.StatusForbidden},
		{"editor can edit", types.AccessRoleKnowledgeEditor, PermissionKnowledgeWrite, http.StatusOK},
		{"editor cannot admin", types.AccessRoleKnowledgeEditor, PermissionUserRoleAssign, http.StatusForbidden},
		{"department admin can manage", types.AccessRoleDepartmentAdmin, PermissionDepartmentManage, http.StatusOK},
		{"department admin cannot assign roles", types.AccessRoleDepartmentAdmin, PermissionUserRoleAssign, http.StatusForbidden},
		{"super admin can assign roles", types.AccessRoleSuperAdmin, PermissionUserRoleAssign, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(func(c *gin.Context) { c.Set("role", string(tt.role)); c.Next() })
			r.POST("/protected", RequirePermission(tt.permission), func(c *gin.Context) { c.Status(http.StatusOK) })
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/protected", nil))
			assert.Equal(t, tt.want, w.Code)
		})
	}
}

func TestRequireRoleNormalizesAndRestrictsAccessRoles(t *testing.T) {
	tests := []struct {
		name    string
		role    string
		allowed []string
		want    int
	}{
		{"super admin allowed", "super_admin", []string{"super_admin"}, http.StatusOK},
		{"legacy ops admin normalized", "ops_admin", []string{"super_admin"}, http.StatusOK},
		{"department admin allowed for user management", "department_admin", []string{"super_admin", "department_admin"}, http.StatusOK},
		{"department admin rejected from super admin action", "department_admin", []string{"super_admin"}, http.StatusForbidden},
		{"missing role rejected", "", []string{"super_admin"}, http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/protected", func(c *gin.Context) {
				c.Set("role", tt.role)
				if RequireRole(c, tt.allowed...) {
					return
				}
				c.Status(http.StatusOK)
			})
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/protected", nil))
			assert.Equal(t, tt.want, w.Code)
			if tt.want == http.StatusForbidden {
				assert.JSONEq(t, `{"error":"permission denied"}`, w.Body.String())
			}
		})
	}
}
