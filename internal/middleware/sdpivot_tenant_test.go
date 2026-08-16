package middleware

import (
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	savepointSQL   = "SAVEPOINT sdpivot_tenant_context"
	rollbackToSQL  = "ROLLBACK TO SAVEPOINT sdpivot_tenant_context"
	releaseSQL     = "RELEASE SAVEPOINT sdpivot_tenant_context"
	primarySQL     = "SELECT set_tenant_context($1, $2)"
	fallbackSQL    = "SELECT set_config('app.current_tenant_id', $1, true), set_config('app.is_ops_admin', 'false', true)"
	databaseErrMsg = "database context failure"
)

func newTenantContextMock(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm postgres connection: %v", err)
	}
	return db, mock
}

func expectExec(mock sqlmock.Sqlmock, query string) *sqlmock.ExpectedExec {
	return mock.ExpectExec(regexp.QuoteMeta(query))
}

func requireMockExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestSDPivotTenantFallbackContextIsTransactionLocalAndNeverElevated(t *testing.T) {
	query, args := sdPivotTenantFallbackContext(uint64(42))

	want := "SELECT set_config('app.current_tenant_id', ?, true), set_config('app.is_ops_admin', 'false', true)"
	if query != want {
		t.Fatalf("fallback query = %q, want %q", query, want)
	}
	if len(args) != 1 {
		t.Fatalf("fallback args = %#v, want tenant ID only", args)
	}
	if got, ok := args[0].(string); !ok || got != "42" {
		t.Fatalf("fallback tenant arg = %#v, want string 42", args[0])
	}
	if strings.Contains(query, "app.is_ops_admin', ?") {
		t.Fatal("fallback must not bind the request role to the operations context")
	}
}

func TestConfigureSDPivotTenantContextSavepointCreationFailure(t *testing.T) {
	db, mock := newTenantContextMock(t)
	expectExec(mock, savepointSQL).WillReturnError(errors.New(databaseErrMsg))

	err := configureSDPivotTenantContext(db, 42, true)
	if err == nil || !strings.Contains(err.Error(), "create tenant context savepoint") {
		t.Fatalf("savepoint creation error = %v", err)
	}
	requireMockExpectations(t, mock)
}

func TestConfigureSDPivotTenantContextPrimarySuccess(t *testing.T) {
	db, mock := newTenantContextMock(t)
	expectExec(mock, savepointSQL).WillReturnResult(sqlmock.NewResult(0, 0))
	expectExec(mock, primarySQL).WithArgs(uint64(42), true).WillReturnResult(sqlmock.NewResult(0, 1))
	expectExec(mock, releaseSQL).WillReturnResult(sqlmock.NewResult(0, 0))

	if err := configureSDPivotTenantContext(db, 42, true); err != nil {
		t.Fatalf("configure primary context: %v", err)
	}
	requireMockExpectations(t, mock)
}

func TestGetAccessRoleNormalizesLegacyOpsAdminForTenantContext(t *testing.T) {
	c, _ := gin.CreateTestContext(nil)
	c.Set("role", "ops_admin")
	if got := GetAccessRole(c); got != types.AccessRoleSuperAdmin {
		t.Fatalf("access role = %q, want %q", got, types.AccessRoleSuperAdmin)
	}
}

func TestConfigureSDPivotTenantContextRecoversBeforeFallback(t *testing.T) {
	db, mock := newTenantContextMock(t)
	expectExec(mock, savepointSQL).WillReturnResult(sqlmock.NewResult(0, 0))
	expectExec(mock, primarySQL).WithArgs(uint64(42), true).WillReturnError(errors.New(databaseErrMsg))
	expectExec(mock, rollbackToSQL).WillReturnResult(sqlmock.NewResult(0, 0))
	expectExec(mock, fallbackSQL).WithArgs("42").WillReturnResult(sqlmock.NewResult(0, 1))
	expectExec(mock, releaseSQL).WillReturnResult(sqlmock.NewResult(0, 0))

	if err := configureSDPivotTenantContext(db, 42, true); err != nil {
		t.Fatalf("configure fallback context: %v", err)
	}
	requireMockExpectations(t, mock)
}

func TestConfigureSDPivotTenantContextRollbackToFailure(t *testing.T) {
	db, mock := newTenantContextMock(t)
	expectExec(mock, savepointSQL).WillReturnResult(sqlmock.NewResult(0, 0))
	expectExec(mock, primarySQL).WithArgs(uint64(42), false).WillReturnError(errors.New(databaseErrMsg))
	expectExec(mock, rollbackToSQL).WillReturnError(errors.New(databaseErrMsg))

	err := configureSDPivotTenantContext(db, 42, false)
	if err == nil || !strings.Contains(err.Error(), "restore tenant context savepoint") {
		t.Fatalf("rollback-to error = %v", err)
	}
	requireMockExpectations(t, mock)
}

func TestConfigureSDPivotTenantContextFallbackFailure(t *testing.T) {
	db, mock := newTenantContextMock(t)
	expectExec(mock, savepointSQL).WillReturnResult(sqlmock.NewResult(0, 0))
	expectExec(mock, primarySQL).WithArgs(uint64(42), true).WillReturnError(errors.New(databaseErrMsg))
	expectExec(mock, rollbackToSQL).WillReturnResult(sqlmock.NewResult(0, 0))
	expectExec(mock, fallbackSQL).WithArgs("42").WillReturnError(errors.New(databaseErrMsg))

	err := configureSDPivotTenantContext(db, 42, true)
	if err == nil || !strings.Contains(err.Error(), "set fallback tenant context") {
		t.Fatalf("fallback error = %v", err)
	}
	requireMockExpectations(t, mock)
}

func TestConfigureSDPivotTenantContextPrimaryReleaseFailure(t *testing.T) {
	db, mock := newTenantContextMock(t)
	expectExec(mock, savepointSQL).WillReturnResult(sqlmock.NewResult(0, 0))
	expectExec(mock, primarySQL).WithArgs(uint64(42), false).WillReturnResult(sqlmock.NewResult(0, 1))
	expectExec(mock, releaseSQL).WillReturnError(errors.New(databaseErrMsg))

	err := configureSDPivotTenantContext(db, 42, false)
	if err == nil || !strings.Contains(err.Error(), "release tenant context savepoint") {
		t.Fatalf("primary release error = %v", err)
	}
	requireMockExpectations(t, mock)
}

func TestConfigureSDPivotTenantContextFallbackReleaseFailure(t *testing.T) {
	db, mock := newTenantContextMock(t)
	expectExec(mock, savepointSQL).WillReturnResult(sqlmock.NewResult(0, 0))
	expectExec(mock, primarySQL).WithArgs(uint64(42), true).WillReturnError(errors.New(databaseErrMsg))
	expectExec(mock, rollbackToSQL).WillReturnResult(sqlmock.NewResult(0, 0))
	expectExec(mock, fallbackSQL).WithArgs("42").WillReturnResult(sqlmock.NewResult(0, 1))
	expectExec(mock, releaseSQL).WillReturnError(errors.New(databaseErrMsg))

	err := configureSDPivotTenantContext(db, 42, true)
	if err == nil || !strings.Contains(err.Error(), "release fallback tenant context savepoint") {
		t.Fatalf("fallback release error = %v", err)
	}
	requireMockExpectations(t, mock)
}

func TestSDPivotTenantContextConfigurationFailureRollsBackAndHidesDetails(t *testing.T) {
	content, err := os.ReadFile("sdpivot_tenant.go")
	if err != nil {
		t.Fatalf("read tenant middleware: %v", err)
	}
	source := strings.Join(strings.Fields(string(content)), " ")
	failureFlow := `if err := configureSDPivotTenantContext(tx, tenantID, isOpsAdmin); err != nil {
		_ = tx.Rollback().Error
		log.Printf("failed to set tenant database context: %v", err)
		c.JSON(500, gin.H{"error": "failed to set tenant database context"})
		c.Abort()
		return
	}`
	if !strings.Contains(source, strings.Join(strings.Fields(failureFlow), " ")) {
		t.Fatal("configuration failure must roll back before returning the generic client error")
	}
	if strings.Contains(source, `"detail"`) {
		t.Fatal("tenant context failures must not expose database details to clients")
	}
}
