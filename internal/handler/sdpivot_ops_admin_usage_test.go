package handler

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGetUsageStatsAggregatesByDateUserAndDepartment(t *testing.T) {
	db, mock := newOpsAdminSQLMock(t)
	h := NewSDPivotOpsAdminHandler(db)

	mock.ExpectExec(regexp.QuoteMeta("SET LOCAL row_security = off")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT count\(\*\) FROM \(SELECT 1 FROM token_usage tu LEFT JOIN users u ON u.id = tu.user_id AND u.deleted_at IS NULL LEFT JOIN departments d ON d.id = u.department_id AND d.deleted_at IS NULL WHERE tu.created_at >= \$1 AND tu.created_at < \(\$2::date \+ INTERVAL '1 day'\) AND u.department_id = \$3 GROUP BY TO_CHAR\(tu.created_at, 'YYYY-MM-DD'\), tu.tenant_id, tu.user_id, u.department_id\) AS usage_groups`).
		WithArgs("2026-07-01", "2026-07-31", "dept-1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT\s+TO_CHAR\(tu.created_at, 'YYYY-MM-DD'\) AS date,\s+tu.tenant_id,\s+COALESCE\(tu.user_id, ''\) AS user_id,\s+COALESCE\(u.username, ''\) AS username,\s+COALESCE\(u.department_id, ''\) AS department_id,\s+COALESCE\(d.name, ''\) AS department_name,\s+COALESCE\(SUM\(tu.input_tokens\), 0\) AS prompt_tokens,\s+COALESCE\(SUM\(tu.output_tokens\), 0\) AS completion_tokens,\s+COALESCE\(SUM\(tu.input_tokens \+ tu.output_tokens\), 0\) AS total_tokens,\s+COUNT\(\*\) AS request_count FROM token_usage tu LEFT JOIN users u ON u.id = tu.user_id AND u.deleted_at IS NULL LEFT JOIN departments d ON d.id = u.department_id AND d.deleted_at IS NULL WHERE tu.created_at >= \$1 AND tu.created_at < \(\$2::date \+ INTERVAL '1 day'\) AND u.department_id = \$3 GROUP BY TO_CHAR\(tu.created_at, 'YYYY-MM-DD'\), tu.tenant_id, tu.user_id, u.username, u.department_id, d.name ORDER BY date DESC, total_tokens DESC, user_id ASC LIMIT \$4`).
		WithArgs("2026-07-01", "2026-07-31", "dept-1", 20).
		WillReturnRows(sqlmock.NewRows([]string{"date", "tenant_id", "user_id", "username", "department_id", "department_name", "prompt_tokens", "completion_tokens", "total_tokens", "request_count"}).
			AddRow("2026-07-24", 7, "user-1", "alice", "dept-1", "Research", 120, 30, 150, 2))
	mock.ExpectQuery(`SELECT\s+COALESCE\(SUM\(tu.input_tokens\), 0\) AS total_prompt_tokens,\s+COALESCE\(SUM\(tu.output_tokens\), 0\) AS total_completion_tokens,\s+COALESCE\(SUM\(tu.input_tokens \+ tu.output_tokens\), 0\) AS total_tokens,\s+COUNT\(\*\) AS request_count FROM token_usage tu LEFT JOIN users u ON u.id = tu.user_id AND u.deleted_at IS NULL LEFT JOIN departments d ON d.id = u.department_id AND d.deleted_at IS NULL WHERE tu.created_at >= \$1 AND tu.created_at < \(\$2::date \+ INTERVAL '1 day'\) AND u.department_id = \$3`).
		WithArgs("2026-07-01", "2026-07-31", "dept-1").
		WillReturnRows(sqlmock.NewRows([]string{"total_prompt_tokens", "total_completion_tokens", "total_tokens", "request_count"}).AddRow(120, 30, 150, 2))

	c, w := newOpsAdminContext(http.MethodGet, "/ops/usage-stats?start_date=2026-07-01&end_date=2026-07-31&department_id=dept-1")
	h.GetUsageStats(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Stats   []opsUsageStatRow `json:"stats"`
		Summary map[string]int64  `json:"summary"`
		Total   int64             `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Total != 1 || len(body.Stats) != 1 || body.Stats[0].TotalTokens != 150 {
		t.Fatalf("unexpected usage response: %+v", body)
	}
	if body.Summary["total_tokens"] != 150 || body.Summary["request_count"] != 2 {
		t.Fatalf("unexpected usage summary: %+v", body.Summary)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}

func TestExportUsageStatsWritesCSV(t *testing.T) {
	db, mock := newOpsAdminSQLMock(t)
	h := NewSDPivotOpsAdminHandler(db)

	mock.ExpectExec(regexp.QuoteMeta("SET LOCAL row_security = off")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT\s+TO_CHAR\(tu.created_at, 'YYYY-MM-DD'\) AS date,.*FROM token_usage tu.*WHERE tu.user_id = \$1.*ORDER BY date DESC, total_tokens DESC, user_id ASC`).
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{"date", "tenant_id", "user_id", "username", "department_id", "department_name", "prompt_tokens", "completion_tokens", "total_tokens", "request_count"}).
			AddRow("2026-07-24", 7, "user-1", "alice", "dept-1", "Research", 120, 30, 150, 2))

	c, w := newOpsAdminContext(http.MethodGet, "/ops/usage-stats/export?user_id=user-1")
	h.ExportUsageStats(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Content-Disposition"); got != "attachment; filename=usage_statistics.csv" {
		t.Fatalf("unexpected content disposition: %q", got)
	}
	records, err := csv.NewReader(strings.NewReader(w.Body.String())).ReadAll()
	if err != nil {
		t.Fatalf("read CSV: %v", err)
	}
	if len(records) != 2 || records[1][0] != "2026-07-24" || records[1][8] != "150" {
		t.Fatalf("unexpected CSV records: %#v", records)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}

func TestGetUsageStatsRejectsInvalidDateRange(t *testing.T) {
	db, mock := newOpsAdminSQLMock(t)
	h := NewSDPivotOpsAdminHandler(db)
	mock.ExpectExec(regexp.QuoteMeta("SET LOCAL row_security = off")).WillReturnResult(sqlmock.NewResult(0, 0))

	c, w := newOpsAdminContext(http.MethodGet, "/ops/usage-stats?start_date=2026-08-01&end_date=2026-07-01")
	h.GetUsageStats(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}
