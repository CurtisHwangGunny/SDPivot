package handler

import (
	"encoding/json"
	"net/http"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Tencent/WeKnora/internal/types"
)

func TestGetCurrentUserReturnsSafeProfile(t *testing.T) {
	db, mock := newOpsAdminSQLMock(t)
	h := NewSDPivotAuthHandler(db, nil, nil)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT to_regclass('departments') IS NOT NULL AS has_departments")).
		WillReturnRows(sqlmock.NewRows([]string{"has_departments"}).AddRow(true))
	mock.ExpectQuery(`SELECT u.id, u.username, u.email, u.avatar, u.tenant_id, u.is_active,.*FROM users u LEFT JOIN smartknora_user_profiles p ON.*LEFT JOIN departments d ON.*WHERE u.id = \$1 AND u.tenant_id = \$2 AND u.is_active = true AND u.deleted_at IS NULL`).
		WithArgs("user-1", uint64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "avatar", "tenant_id", "is_active", "access_role", "department_id", "department_name", "phone", "nickname", "must_change_password"}).
			AddRow("user-1", "alice", "alice@example.com", "avatar.png", 7, true, "knowledge_editor", "dept-1", "Research", "13800000000", "Alice", false))

	c, w := newOpsAdminContext(http.MethodGet, "/auth/me")
	c.Set("user_id", "user-1")
	c.Set("tenant_id", uint64(7))
	h.GetCurrentUser(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	user, ok := body["user"].(map[string]interface{})
	if !ok || user["id"] != "user-1" || user["access_role"] != string(types.AccessRoleKnowledgeEditor) {
		t.Fatalf("unexpected user response: %#v", body)
	}
	for _, sensitive := range []string{"password_hash", "failed_login_attempts", "locked_until", "password_changed_at", "password_expires_at"} {
		if _, exists := user[sensitive]; exists {
			t.Fatalf("sensitive field %q leaked in response", sensitive)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}

func TestListTagDictionaryReusesClassificationRepository(t *testing.T) {
	db, mock := newOpsAdminSQLMock(t)
	h := NewSDPivotQAHandler(db)

	mock.ExpectQuery(`SELECT \* FROM "tag_dimensions" ORDER BY sort_order ASC, code ASC`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "code", "name", "description", "sort_order"}).
			AddRow("dim-1", "topic", "Topic", "", 10))
	mock.ExpectQuery(`SELECT \* FROM "tag_dictionary" ORDER BY dimension_id ASC, sort_order ASC, name ASC`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "dimension_id", "name", "color", "sort_order"}).
			AddRow("tag-1", "dim-1", "Customer Success", "", 10))

	c, w := newOpsAdminContext(http.MethodGet, "/admin/tags")
	h.ListTagDictionary(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Dimensions []types.TagDimension  `json:"dimensions"`
		Tags       []types.TagDictionary `json:"tags"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Dimensions) != 1 || len(body.Tags) != 1 || body.Tags[0].Name != "Customer Success" {
		t.Fatalf("unexpected dictionary response: %+v", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}
