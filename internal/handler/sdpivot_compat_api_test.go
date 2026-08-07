package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newCurrentUserSQLiteDB(t *testing.T, extendedSchema bool) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	extendedColumns := ""
	if extendedSchema {
		extendedColumns = ", access_role text, department_id text, must_change_password boolean"
	}
	require.NoError(t, db.Exec(fmt.Sprintf(`CREATE TABLE users (
		id text PRIMARY KEY, username text NOT NULL, email text NOT NULL, password_hash text NOT NULL,
		avatar text, tenant_id integer NOT NULL, is_active boolean NOT NULL, deleted_at datetime%s
	)`, extendedColumns)).Error)
	if extendedSchema {
		require.NoError(t, db.Exec(`CREATE TABLE smartknora_user_profiles (
			id text PRIMARY KEY, user_id text NOT NULL, phone text, nickname text, deleted_at datetime
		)`).Error)
		require.NoError(t, db.Exec(`CREATE TABLE departments (
			id text PRIMARY KEY, tenant_id integer NOT NULL, name text NOT NULL, deleted_at datetime
		)`).Error)
	}
	return db
}

func currentUserContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	c.Set("user_id", "user-1")
	c.Set("tenant_id", uint64(7))
	return c, w
}

func TestGetCurrentUserReturnsSafeProfile(t *testing.T) {
	db := newCurrentUserSQLiteDB(t, true)
	require.NoError(t, db.Exec(`INSERT INTO users
		(id, username, email, password_hash, avatar, tenant_id, is_active, access_role, department_id, must_change_password)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"user-1", "alice", "alice@example.com", "secret-hash", "avatar.png", 7, true, "knowledge_editor", "dept-1", true).Error)
	require.NoError(t, db.Exec(`INSERT INTO smartknora_user_profiles (id, user_id, phone, nickname) VALUES (?, ?, ?, ?)`, "profile-1", "user-1", "13800000000", "Alice").Error)
	require.NoError(t, db.Exec(`INSERT INTO departments (id, tenant_id, name) VALUES (?, ?, ?)`, "dept-1", 7, "Research").Error)
	h := NewSDPivotAuthHandler(db, nil, nil)

	c, w := currentUserContext()
	h.GetCurrentUser(c)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	user, ok := body["user"].(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "user-1", user["id"])
	require.Equal(t, string(types.AccessRoleKnowledgeEditor), user["access_role"])
	require.Equal(t, "Research", user["department_name"])
	require.Equal(t, "13800000000", user["phone"])
	require.Equal(t, true, user["must_change_password"])
	for _, sensitive := range []string{
		"password_hash", "token", "api_key", "secret", "failed_login_attempts", "locked_until",
		"password_changed_at", "password_expires_at", "is_ops_admin", "is_system_admin", "can_access_all_tenants",
	} {
		require.NotContains(t, user, sensitive)
	}
}

func TestGetCurrentUserSupportsLegacyOptionalSchema(t *testing.T) {
	db := newCurrentUserSQLiteDB(t, false)
	require.NoError(t, db.Exec(`INSERT INTO users
		(id, username, email, password_hash, avatar, tenant_id, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, "user-1", "legacy", "legacy@example.com", "secret-hash", "", 7, true).Error)
	h := NewSDPivotAuthHandler(db, nil, nil)

	c, w := currentUserContext()
	h.GetCurrentUser(c)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct {
		User struct {
			AccessRole         types.AccessRole `json:"access_role"`
			DepartmentID       *string          `json:"department_id"`
			DepartmentName     string           `json:"department_name"`
			Phone              *string          `json:"phone"`
			Nickname           string           `json:"nickname"`
			MustChangePassword bool             `json:"must_change_password"`
		} `json:"user"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, types.AccessRoleKnowledgeViewer, body.User.AccessRole)
	require.Nil(t, body.User.DepartmentID)
	require.Empty(t, body.User.DepartmentName)
	require.Nil(t, body.User.Phone)
	require.Empty(t, body.User.Nickname)
	require.False(t, body.User.MustChangePassword)
}

func TestListTagDictionaryReusesClassificationRepository(t *testing.T) {
	db, mock := newOpsAdminSQLMock(t)
	h := NewSDPivotAdminHandler(db)

	mock.ExpectQuery(`SELECT \* FROM "tag_dimensions" ORDER BY sort_order ASC, code ASC`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "code", "name", "description", "sort_order"}).
			AddRow("dim-1", "topic", "Topic", "", 10))
	mock.ExpectQuery(`SELECT \* FROM "tag_dictionary" ORDER BY dimension_id ASC, sort_order ASC, name ASC`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "dimension_id", "name", "color", "sort_order"}).
			AddRow("tag-1", "dim-1", "Customer Success", "", 10))

	c, w := newOpsAdminContext(http.MethodGet, "/admin/tags")
	h.ListTagDictionary(c)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body struct {
		Dimensions []types.TagDimension  `json:"dimensions"`
		Tags       []types.TagDictionary `json:"tags"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Dimensions, 1)
	require.Len(t, body.Tags, 1)
	require.Equal(t, "Customer Success", body.Tags[0].Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListTagDictionaryReturnsEmptyArrays(t *testing.T) {
	db, mock := newOpsAdminSQLMock(t)
	h := NewSDPivotAdminHandler(db)
	mock.ExpectQuery(`SELECT \* FROM "tag_dimensions" ORDER BY sort_order ASC, code ASC`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "code", "name", "description", "sort_order"}))
	mock.ExpectQuery(`SELECT \* FROM "tag_dictionary" ORDER BY dimension_id ASC, sort_order ASC, name ASC`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "dimension_id", "name", "color", "sort_order"}))

	c, w := newOpsAdminContext(http.MethodGet, "/admin/tags")
	h.ListTagDictionary(c)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.JSONEq(t, `[]`, string(body["dimensions"]))
	require.JSONEq(t, `[]`, string(body["tags"]))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminTagRouteRequiresSuperAdmin(t *testing.T) {
	for _, tc := range []struct {
		name string
		role string
		want int
	}{
		{name: "super admin", role: string(types.AccessRoleSuperAdmin), want: http.StatusOK},
		{name: "department admin", role: string(types.AccessRoleDepartmentAdmin), want: http.StatusOK},
		{name: "viewer", role: string(types.AccessRoleKnowledgeViewer), want: http.StatusOK},
		{name: "editor", role: string(types.AccessRoleKnowledgeEditor), want: http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newOpsAdminSQLMock(t)
			mock.ExpectQuery(`SELECT \* FROM "tag_dimensions" ORDER BY sort_order ASC, code ASC`).
				WillReturnRows(sqlmock.NewRows([]string{"id", "code", "name", "description", "sort_order"}))
			mock.ExpectQuery(`SELECT \* FROM "tag_dictionary" ORDER BY dimension_id ASC, sort_order ASC, name ASC`).
				WillReturnRows(sqlmock.NewRows([]string{"id", "dimension_id", "name", "color", "sort_order"}))
			r := gin.New()
			r.Use(func(c *gin.Context) {
				c.Set("role", tc.role)
				c.Next()
			})
			NewSDPivotAdminHandler(db).RegisterRoutes(r.Group(""))

			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/tags", nil))
			require.Equal(t, tc.want, w.Code, w.Body.String())
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}

	t.Run("viewer cannot write", func(t *testing.T) {
		db, mock := newOpsAdminSQLMock(t)
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("role", string(types.AccessRoleKnowledgeViewer))
			c.Next()
		})
		NewSDPivotAdminHandler(db).RegisterRoutes(r.Group(""))

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/admin/tags", nil))
		require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
