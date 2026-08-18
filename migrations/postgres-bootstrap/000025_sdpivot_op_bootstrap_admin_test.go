package postgresbootstrap

import (
	"os"
	"strings"
	"testing"
)

func TestOPBootstrapAdminMigrationAddsPasswordChangeFlagIdempotently(t *testing.T) {
	data, err := os.ReadFile("000025_sdpivot_op_bootstrap_admin.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(data))
	for _, want := range []string{
		"alter table users", "add column if not exists must_change_password",
		"boolean not null default false",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("migration missing %q: %s", want, sql)
		}
	}
}
