package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newOpsAdminScopeDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE departments (
		id text PRIMARY KEY, tenant_id integer NOT NULL, parent_id text NOT NULL,
		name text NOT NULL, deleted_at datetime
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE users (
		id text PRIMARY KEY, username text NOT NULL, email text NOT NULL,
		is_active boolean NOT NULL, is_ops_admin boolean NOT NULL,
		access_role text NOT NULL, department_id text, tenant_id integer NOT NULL,
		created_at datetime NOT NULL, updated_at datetime, deleted_at datetime
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE smartknora_user_profiles (
		user_id text PRIMARY KEY, phone text, nickname text
	)`).Error)
	return db
}

func newDepartmentAdminContext(method, target, departmentID string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, target, bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("role", string(types.AccessRoleDepartmentAdmin))
	c.Set("tenant_id", uint64(7))
	c.Set("department_id", departmentID)
	return c, w
}

func TestDepartmentAdminSeesOnlyDepartmentUsers(t *testing.T) {
	db := newOpsAdminScopeDB(t)
	require.NoError(t, db.Exec(`INSERT INTO departments (id, tenant_id, parent_id, name) VALUES
		('engineering', 7, '', 'Engineering'),
		('platform', 7, 'engineering', 'Platform'),
		('sales', 7, '', 'Sales'),
		('other-tenant', 8, '', 'Other')`).Error)
	now := time.Now()
	require.NoError(t, db.Exec(`INSERT INTO users
		(id, username, email, is_active, is_ops_admin, access_role, department_id, tenant_id, created_at) VALUES
		('root-user', 'root', 'root@example.com', true, false, 'knowledge_viewer', 'engineering', 7, ?),
		('child-user', 'child', 'child@example.com', true, false, 'knowledge_editor', 'platform', 7, ?),
		('sales-user', 'sales', 'sales@example.com', true, false, 'knowledge_viewer', 'sales', 7, ?),
		('other-user', 'other', 'other@example.com', true, false, 'knowledge_viewer', 'other-tenant', 8, ?)`, now, now, now, now).Error)

	h := NewSDPivotOpsAdminHandler(db, true)
	c, w := newDepartmentAdminContext(http.MethodGet, "/ops/users", "engineering", nil)
	h.ListUsers(c)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var response struct {
		Users []struct {
			ID string `json:"id"`
		} `json:"users"`
		Total int64 `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.EqualValues(t, 2, response.Total)
	require.ElementsMatch(t, []string{"root-user", "child-user"}, []string{response.Users[0].ID, response.Users[1].ID})
}

func TestDepartmentAdminCannotAssignSuperAdmin(t *testing.T) {
	h := NewSDPivotOpsAdminHandler(nil, true)
	c, w := newDepartmentAdminContext(http.MethodPut, "/ops/users/target/role", "engineering", []byte(`{"role":"super_admin"}`))
	c.Params = gin.Params{{Key: "id", Value: "target"}}

	h.UpdateUserRole(c)

	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
}

func TestDepartmentAdminCannotUpdateUserStatusOutsideScope(t *testing.T) {
	db := newOpsAdminScopeDB(t)
	require.NoError(t, db.Exec(`INSERT INTO departments (id, tenant_id, parent_id, name) VALUES
		('engineering', 7, '', 'Engineering'),
		('sales', 7, '', 'Sales'),
		('other', 8, '', 'Other')`).Error)
	now := time.Now()
	require.NoError(t, db.Exec(`INSERT INTO users
		(id, username, email, is_active, is_ops_admin, access_role, department_id, tenant_id, created_at) VALUES
		('sales-user', 'sales', 'sales@example.com', true, false, 'knowledge_viewer', 'sales', 7, ?),
		('other-user', 'other', 'other@example.com', true, false, 'knowledge_viewer', 'other', 8, ?)`, now, now).Error)

	for _, userID := range []string{"sales-user", "other-user"} {
		c, w := newDepartmentAdminContext(http.MethodPut, "/ops/users/"+userID+"/status", "engineering", []byte(`{"is_active":false}`))
		c.Params = gin.Params{{Key: "id", Value: userID}}
		NewSDPivotOpsAdminHandler(db, true).UpdateUserStatus(c)
		require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
		var active bool
		require.NoError(t, db.Table("users").Select("is_active").Where("id = ?", userID).Scan(&active).Error)
		require.True(t, active)
	}
}

func TestDepartmentAdminCanUpdateUserStatusInChildDepartment(t *testing.T) {
	db := newOpsAdminScopeDB(t)
	require.NoError(t, db.Exec(`INSERT INTO departments (id, tenant_id, parent_id, name) VALUES
		('engineering', 7, '', 'Engineering'),
		('platform', 7, 'engineering', 'Platform')`).Error)
	now := time.Now()
	require.NoError(t, db.Exec(`INSERT INTO users
		(id, username, email, is_active, is_ops_admin, access_role, department_id, tenant_id, created_at) VALUES
		('child-user', 'child', 'child@example.com', true, false, 'knowledge_viewer', 'platform', 7, ?)`, now).Error)

	c, w := newDepartmentAdminContext(http.MethodPut, "/ops/users/child-user/status", "engineering", []byte(`{"is_active":false}`))
	c.Params = gin.Params{{Key: "id", Value: "child-user"}}
	NewSDPivotOpsAdminHandler(db, true).UpdateUserStatus(c)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var active bool
	require.NoError(t, db.Table("users").Select("is_active").Where("id = ?", "child-user").Scan(&active).Error)
	require.False(t, active)
}
