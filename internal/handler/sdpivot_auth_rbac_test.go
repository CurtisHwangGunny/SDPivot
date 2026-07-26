package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/auth"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newSDPivotAuthRBACDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&types.Tenant{}, &types.TenantMember{}, &types.User{}, &types.RefreshToken{}, &types.SDPivotUserProfile{},
		&types.KnowledgeSpace{}, &types.SDPivotDocument{}, &types.SDPivotOrgMember{}, &types.OrgExt{},
		&types.Organization{},
	))
	require.NoError(t, db.Exec("ALTER TABLE org_members ADD COLUMN updated_at datetime").Error)
	return db
}

func TestSDPivotRegistrationUsesCanonicalOPTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newSDPivotAuthRBACDB(t)
	manager := auth.NewJWTManager(auth.DefaultJWTConfig("focused-register-secret"))
	handler := NewSDPivotAuthHandler(db, manager, nil)

	body := `{"email":"member@example.com","password":"ValidPass!123","nickname":"Member"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/sdp/auth/register", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	handler.Register(c)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var response types.SDPivotAuthResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	claims, err := manager.ValidateAccessToken(response.AccessToken)
	require.NoError(t, err)
	require.Equal(t, types.DefaultTenantID, claims.TenantID)

	var user types.User
	require.NoError(t, db.Where("email = ?", "member@example.com").First(&user).Error)
	require.Equal(t, types.DefaultTenantID, user.TenantID)

	var tenantCount int64
	require.NoError(t, db.Model(&types.Tenant{}).Count(&tenantCount).Error)
	require.EqualValues(t, 1, tenantCount)
	var tenant types.Tenant
	require.NoError(t, db.First(&tenant, types.DefaultTenantID).Error)
	require.Equal(t, "SDPivot", tenant.Name)
	require.Equal(t, "SDPivot OP tenant", tenant.Description)
}

func TestSDPivotLoginEmitsLocalOPClaimsWithoutSaaSBypass(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newSDPivotAuthRBACDB(t)
	manager := auth.NewJWTManager(auth.DefaultJWTConfig("focused-auth-secret"))
	handler := NewSDPivotAuthHandler(db, manager, nil)

	for _, tc := range []struct {
		name     string
		email    string
		user     types.User
		wantRole types.AccessRole
	}{
		{
			name:  "repaired OP administrator",
			email: "sysadmin@sdpivot.local",
			user: types.User{
				TenantID: types.DefaultTenantID, IsSystemAdmin: true, IsOpsAdmin: true,
				CanAccessAllTenants: false, AccessRole: types.AccessRoleSuperAdmin,
			},
			wantRole: types.AccessRoleSuperAdmin,
		},
		{
			name:     "ordinary user",
			email:    "member@example.com",
			user:     types.User{TenantID: types.DefaultTenantID, AccessRole: types.AccessRoleKnowledgeViewer},
			wantRole: types.AccessRoleKnowledgeViewer,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hash, err := bcrypt.GenerateFromPassword([]byte("ValidPass!123"), bcrypt.DefaultCost)
			require.NoError(t, err)
			user := tc.user
			user.ID = strings.ReplaceAll(tc.name, " ", "-")
			user.Username = tc.email
			user.Email = tc.email
			user.PasswordHash = string(hash)
			user.IsActive = true
			require.NoError(t, db.Create(&user).Error)

			body, err := json.Marshal(types.SDPivotLoginRequest{Email: tc.email, Password: "ValidPass!123"})
			require.NoError(t, err)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/sdp/auth/login", bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")
			handler.Login(c)
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())

			var response types.SDPivotAuthResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
			claims, err := manager.ValidateAccessToken(response.AccessToken)
			require.NoError(t, err)
			require.Equal(t, types.DefaultTenantID, claims.TenantID)
			require.Equal(t, string(tc.wantRole), claims.Role)

			parsed, _, err := jwt.NewParser().ParseUnverified(response.AccessToken, jwt.MapClaims{})
			require.NoError(t, err)
			rawClaims, ok := parsed.Claims.(jwt.MapClaims)
			require.True(t, ok)
			require.NotContains(t, rawClaims, "can_access_all_tenants")
			require.NotContains(t, rawClaims, "is_ops_admin")
			require.NotEqual(t, "ops_admin", rawClaims["role"])
		})
	}
}

func TestSDPivotAdminViewsAllowAdminsAndDenyNonAdmins(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newSDPivotAuthRBACDB(t)
	h := NewSDPivotQAHandler(db)
	now := time.Now()
	require.NoError(t, db.Create(&types.KnowledgeSpace{ID: "space-1", TenantID: types.DefaultTenantID, Name: "space", CreatedAt: now, UpdatedAt: now}).Error)
	require.NoError(t, db.Create(&types.OrgExt{OrgID: "org-1", TenantID: types.DefaultTenantID, AuthStatus: "trial", CreatedAt: now, UpdatedAt: now}).Error)
	require.NoError(t, db.Create(&types.SDPivotOrgMember{ID: "member-1", OrgID: "org-1", UserID: "user-1", Role: "owner", Status: "active", JoinedAt: now, CreatedAt: now}).Error)

	tests := []struct {
		path    string
		handler gin.HandlerFunc
	}{
		{path: "/admin/members", handler: h.ListAllMembers},
		{path: "/admin/spaces", handler: h.ListAllSpaces},
		{path: "/admin/stats", handler: h.GetAdminStats},
	}
	for _, tc := range tests {
		for _, role := range []types.AccessRole{types.AccessRoleSuperAdmin, types.AccessRoleDepartmentAdmin} {
			t.Run(tc.path+"/allows/"+string(role), func(t *testing.T) {
				c, w := newAdminViewContext(tc.path, role)
				tc.handler(c)
				require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			})
		}
		t.Run(tc.path+"/denies/non-admin", func(t *testing.T) {
			c, w := newAdminViewContext(tc.path, types.AccessRoleKnowledgeEditor)
			tc.handler(c)
			require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
		})
	}
}

func newAdminViewContext(path string, role types.AccessRole) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, path, nil)
	c.Set("user_id", "admin-user")
	c.Set("tenant_id", types.DefaultTenantID)
	c.Set("role", string(role))
	return c, w
}
