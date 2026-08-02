package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newOpsAdminSQLMock(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm postgres: %v", err)
	}
	return db, mock
}

func newOpsAdminContext(method, target string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, target, nil)
	c.Set("role", string(types.AccessRoleSuperAdmin))
	return c, w
}

func TestGetOpsDashboardAllowsLocalSuperAdmin(t *testing.T) {
	db, mock := newOpsAdminSQLMock(t)
	h := NewSDPivotOpsAdminHandler(db, false)

	mock.ExpectExec(regexp.QuoteMeta("SET LOCAL row_security = off")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT count\(\*\) FROM "organizations" WHERE deleted_at IS NULL`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT count\(\*\) FROM "users" WHERE deleted_at IS NULL`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(`SELECT count\(\*\) FROM "documents" WHERE deleted_at IS NULL`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectQuery(`SELECT count\(\*\) FROM "token_usage"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(4))
	mock.ExpectQuery(`SELECT count\(\*\) FROM "token_usage" WHERE created_at >= \$1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(file_size\), 0\) FROM "documents" WHERE deleted_at IS NULL`).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(6))

	c, w := newOpsAdminContext(http.MethodGet, "/ops/dashboard")
	h.GetOpsDashboard(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}

func TestGetAuditLogsReadsCoreFieldsAndFiltersActorUserID(t *testing.T) {
	db, mock := newOpsAdminSQLMock(t)
	h := NewSDPivotOpsAdminHandler(db, false)

	mock.ExpectExec(regexp.QuoteMeta("SET LOCAL row_security = off")).WillReturnResult(sqlmock.NewResult(0, 0))
	countSQL := `SELECT count\(\*\) FROM "audit_logs" WHERE COALESCE\(user_id, actor_user_id\) = \$1`
	mock.ExpectQuery(countSQL).WithArgs("core-actor").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	createdAt := time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC)
	selectSQL := `SELECT\s+id, tenant_id,\s+COALESCE\(user_id, actor_user_id\) AS user_id,\s+COALESCE\(username, ''::text\) AS username, action,\s+COALESCE\(resource, target_type\) AS resource,\s+COALESCE\(resource_id, target_id\) AS resource_id,\s+COALESCE\(detail, details->>'detail', ''::text\) AS detail,\s+COALESCE\(ip, ''::text\) AS ip, created_at FROM "audit_logs" WHERE COALESCE\(user_id, actor_user_id\) = \$1 ORDER BY created_at DESC LIMIT \$2`
	mock.ExpectQuery(selectSQL).WithArgs("core-actor", 20).WillReturnRows(sqlmock.NewRows([]string{
		"id", "tenant_id", "user_id", "username", "action", "resource", "resource_id", "detail", "ip", "created_at",
	}).AddRow(44, 7, "core-actor", "", "update_user_status", "user", "target-9", "is_active: false", "", createdAt))

	c, w := newOpsAdminContext(http.MethodGet, "/ops/audit-logs?user_id=core-actor")
	h.GetAuditLogs(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Logs []struct {
			UserID     string `json:"user_id"`
			Resource   string `json:"resource"`
			ResourceID string `json:"resource_id"`
			Detail     string `json:"detail"`
		} `json:"logs"`
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Total != 1 || len(body.Logs) != 1 {
		t.Fatalf("expected one Core audit row, got total=%d logs=%d", body.Total, len(body.Logs))
	}
	got := body.Logs[0]
	if got.UserID != "core-actor" || got.Resource != "user" || got.ResourceID != "target-9" || got.Detail != "is_active: false" {
		t.Fatalf("Core audit projection was not preserved: %+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}

func TestWriteAuditLogWritesActorAndLegacyUserID(t *testing.T) {
	db, mock := newOpsAdminSQLMock(t)
	h := NewSDPivotOpsAdminHandler(db, false)
	c, _ := newOpsAdminContext(http.MethodPut, "/ops/users/target-9/status")
	c.Set("user_id", "ops-user")
	c.Set("tenant_id", uint64(7))
	c.Set("username", "operator")

	insertSQL := `INSERT INTO audit_logs \(\s+tenant_id, actor_user_id, actor_role, action, target_type, target_id,\s+target_user_id, request_path, request_method, outcome, details,\s+created_at, user_id, username, resource, resource_id, detail, ip\s+\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7, \$8, \$9, \$10, CAST\(\$11 AS jsonb\), \$12, \$13, \$14, \$15, \$16, \$17, \$18\)`
	mock.ExpectExec(insertSQL).WithArgs(
		uint64(7), "ops-user", string(types.AccessRoleSuperAdmin), "update_user_status", "user", "target-9",
		"", "/ops/users/target-9/status", http.MethodPut, "success", `{"detail":"is_active: false"}`,
		sqlmock.AnyArg(), "ops-user", "operator", "user", "target-9", "is_active: false", sqlmock.AnyArg(),
	).WillReturnResult(sqlmock.NewResult(1, 1))

	h.writeAuditLog(c, "update_user_status", "user", "target-9", "is_active: false")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}

func TestCannotDisableLastSuperAdmin(t *testing.T) {
	db, mock := newOpsAdminSQLMock(t)
	h := NewSDPivotOpsAdminHandler(db, true)

	mock.ExpectQuery(`SELECT "access_role" FROM "users" WHERE id = \$1 AND "users"\."deleted_at" IS NULL ORDER BY "users"\."id" LIMIT \$2`).
		WithArgs("last-admin", 1).
		WillReturnRows(sqlmock.NewRows([]string{"access_role"}).AddRow(types.AccessRoleSuperAdmin))
	mock.ExpectQuery(`SELECT count\(\*\) FROM "users" WHERE \(access_role = \$1 AND is_active = \$2\) AND "users"\."deleted_at" IS NULL`).
		WithArgs(types.AccessRoleSuperAdmin, true).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	c, w := newOpsAdminContext(http.MethodPut, "/ops/users/last-admin/status")
	c.Params = gin.Params{{Key: "id", Value: "last-admin"}}
	c.Request = httptest.NewRequest(http.MethodPut, "/ops/users/last-admin/status", bytes.NewBufferString(`{"is_active":false}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.UpdateUserStatus(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["error"] != "不允许禁用最后一个超级管理员" {
		t.Fatalf("unexpected error: %q", body["error"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}
