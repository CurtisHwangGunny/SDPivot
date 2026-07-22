package deploy

import (
	"os"
	"strings"
	"testing"
)

func readDeployFile(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return strings.ToLower(string(data))
}

func requireOrdered(t *testing.T, text string, fragments ...string) {
	t.Helper()
	position := 0
	for _, fragment := range fragments {
		index := strings.Index(text[position:], fragment)
		if index < 0 {
			t.Fatalf("missing ordered fragment %q", fragment)
		}
		position += index + len(fragment)
	}
}

func TestMigrationImagePinsToolingAndCopiesOnlyMigrationInputs(t *testing.T) {
	dockerfile := readDeployFile(t, "migration/Dockerfile")
	for _, fragment := range []string{
		"golang_migrate_version=v4.19.1",
		"github.com/golang-migrate/migrate/v4/cmd/migrate@${golang_migrate_version}",
		"copy --chown=migration:migration migrations/versioned /migrations/versioned",
		"copy --chown=migration:migration migrations/postgres-bootstrap /migrations/postgres-bootstrap",
		"copy --chown=migration:migration migrations/postgres /migrations/postgres",
		"user migration",
	} {
		if !strings.Contains(dockerfile, fragment) {
			t.Errorf("Dockerfile must contain %q", fragment)
		}
	}
}

func TestMigrationScriptUsesTwoVersionTablesAndStrictOrder(t *testing.T) {
	script := readDeployFile(t, "migrate-op.sh")
	for _, fragment := range []string{
		"schema_migrations",
		"sdpivot_schema_migrations",
		"x-migrations-table=%s",
		"automatic force is forbidden",
		"environment is ambiguous",
		"op_database_url is required",
	} {
		if !strings.Contains(script, fragment) {
			t.Errorf("migration script must contain %q", fragment)
		}
	}

	requireOrdered(t, script,
		"run_migrate \"$op_database_url\" \"$core_migrations_dir\"",
		"run_migrate \"$sdpivot_database_url\" \"$bootstrap_migrations_dir\"",
		"assert_clean_version sdpivot_schema_migrations \"$sdpivot_baseline_version\"",
		"run_migrate \"$sdpivot_database_url\" \"$sdpivot_migrations_dir\"",
		"assert_clean_version sdpivot_schema_migrations \"$sdpivot_latest_version\"",
	)
}

func TestMigrationScriptFailsClosedWithoutForceOrCredentialLogging(t *testing.T) {
	script := readDeployFile(t, "migrate-op.sh")
	for _, forbidden := range []string{
		"migrate force",
		"set -x",
		"printf '%s' \"$op_database_url\"",
		"echo \"$op_database_url\"",
	} {
		if strings.Contains(script, forbidden) {
			t.Errorf("migration script contains forbidden fragment %q", forbidden)
		}
	}

	for _, required := range []string{
		"[[ \"$core_dirty\" == \"f\" ]]",
		"[[ \"$sdpivot_dirty\" == \"f\" ]]",
		"existing_public_tables",
		"sdpivot_object_count",
		"expected 12, 13, or 14",
	} {
		if !strings.Contains(script, required) {
			t.Errorf("migration script must contain fail-closed check %q", required)
		}
	}
}
