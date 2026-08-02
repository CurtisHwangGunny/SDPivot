package handler

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newUsageStatsSQLiteDB(t *testing.T, departmentColumn, departmentsTable bool) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	departmentDefinition := ""
	if departmentColumn {
		departmentDefinition = ", department_id text"
	}
	require.NoError(t, db.Exec(fmt.Sprintf(`CREATE TABLE users (
		id text PRIMARY KEY, username text NOT NULL, tenant_id integer NOT NULL,
		is_active boolean NOT NULL, deleted_at datetime%s
	)`, departmentDefinition)).Error)
	require.NoError(t, db.Exec(`CREATE TABLE token_usage (
		id text PRIMARY KEY, user_id text, tenant_id integer NOT NULL,
		input_tokens integer NOT NULL, output_tokens integer NOT NULL, created_at datetime NOT NULL
	)`).Error)
	if departmentsTable {
		require.NoError(t, db.Exec(`CREATE TABLE departments (
			id text PRIMARY KEY, tenant_id integer NOT NULL, name text NOT NULL, deleted_at datetime
		)`).Error)
	}
	return db
}

func seedUsageStats(t *testing.T, db *gorm.DB, departmentColumn bool) {
	t.Helper()
	if departmentColumn {
		require.NoError(t, db.Exec(`INSERT INTO users (id, username, tenant_id, is_active, department_id) VALUES (?, ?, ?, ?, ?)`, "user-1", "alice", 7, true, "dept-1").Error)
	} else {
		require.NoError(t, db.Exec(`INSERT INTO users (id, username, tenant_id, is_active) VALUES (?, ?, ?, ?)`, "user-1", "alice", 7, true).Error)
	}
	require.NoError(t, db.Exec(`INSERT INTO token_usage (id, user_id, tenant_id, input_tokens, output_tokens, created_at) VALUES
		(?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?)`,
		"usage-1", "user-1", 7, 120, 30, time.Date(2026, 7, 24, 9, 0, 0, 0, time.UTC),
		"usage-2", "user-1", 7, 10, 5, time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC)).Error)
}

func TestGetUsageStatsAggregatesByDateUserAndDepartment(t *testing.T) {
	db := newUsageStatsSQLiteDB(t, true, true)
	seedUsageStats(t, db, true)
	require.NoError(t, db.Exec(`INSERT INTO departments (id, tenant_id, name) VALUES (?, ?, ?)`, "dept-1", 7, "Research").Error)
	h := NewSDPivotOpsAdminHandler(db, false)

	c, w := newOpsAdminContext(http.MethodGet, "/ops/usage-stats?start_date=2026-07-01&end_date=2026-07-31&department_id=dept-1")
	h.GetUsageStats(c)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct {
		Stats   []opsUsageStatRow `json:"stats"`
		Summary map[string]int64  `json:"summary"`
		Total   int64             `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.EqualValues(t, 1, body.Total)
	require.Len(t, body.Stats, 1)
	require.EqualValues(t, 165, body.Stats[0].TotalTokens)
	require.Equal(t, "dept-1", body.Stats[0].DepartmentID)
	require.Equal(t, "Research", body.Stats[0].DepartmentName)
	require.EqualValues(t, 165, body.Summary["total_tokens"])
	require.EqualValues(t, 2, body.Summary["request_count"])
}

func TestGetUsageStatsWithoutDepartmentColumnReturnsUnassigned(t *testing.T) {
	db := newUsageStatsSQLiteDB(t, false, false)
	seedUsageStats(t, db, false)
	h := NewSDPivotOpsAdminHandler(db, false)

	c, w := newOpsAdminContext(http.MethodGet, "/ops/usage-stats")
	h.GetUsageStats(c)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct {
		Stats []opsUsageStatRow `json:"stats"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Stats, 1)
	require.Empty(t, body.Stats[0].DepartmentID)
	require.Empty(t, body.Stats[0].DepartmentName)
}

func TestGetUsageStatsWithoutDepartmentsTableKeepsDepartmentID(t *testing.T) {
	db := newUsageStatsSQLiteDB(t, true, false)
	seedUsageStats(t, db, true)
	h := NewSDPivotOpsAdminHandler(db, false)

	c, w := newOpsAdminContext(http.MethodGet, "/ops/usage-stats")
	h.GetUsageStats(c)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct {
		Stats []opsUsageStatRow `json:"stats"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Stats, 1)
	require.Equal(t, "dept-1", body.Stats[0].DepartmentID)
	require.Empty(t, body.Stats[0].DepartmentName)
}

func TestExportUsageStatsWritesCSV(t *testing.T) {
	db := newUsageStatsSQLiteDB(t, false, false)
	seedUsageStats(t, db, false)
	h := NewSDPivotOpsAdminHandler(db, false)

	c, w := newOpsAdminContext(http.MethodGet, "/ops/usage-stats/export?user_id=user-1")
	h.ExportUsageStats(c)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, "attachment; filename=usage_statistics.csv", w.Header().Get("Content-Disposition"))
	records, err := csv.NewReader(strings.NewReader(w.Body.String())).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 2)
	require.Equal(t, "2026-07-24", records[1][0])
	require.Equal(t, "165", records[1][8])
}

func TestGetUsageStatsRejectsInvalidDateRange(t *testing.T) {
	db := newUsageStatsSQLiteDB(t, false, false)
	h := NewSDPivotOpsAdminHandler(db, false)

	c, w := newOpsAdminContext(http.MethodGet, "/ops/usage-stats?start_date=2026-08-01&end_date=2026-07-01")
	h.GetUsageStats(c)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}
