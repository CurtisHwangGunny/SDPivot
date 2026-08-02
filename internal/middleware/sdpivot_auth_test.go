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
	expired, _, err := auth.NewJWTManager(expiredConfig).GenerateAccessToken("u1", 1, string(types.AccessRoleKnowledgeViewer), nil)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+expired)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
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
