package postgresbootstrap

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

const bootstrapPGContractEnv = "SDPIVOT_PG_TEST_DSN"

func requireBootstrapPGContractDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv(bootstrapPGContractEnv)
	if dsn == "" {
		t.Skipf("set %s to run PostgreSQL integration contracts", bootstrapPGContractEnv)
	}
	return dsn
}

func withBootstrapContractDatabase(t *testing.T, fn func(context.Context, *pgx.Conn)) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	config, err := pgx.ParseConfig(requireBootstrapPGContractDSN(t))
	if err != nil {
		t.Fatalf("parse PostgreSQL contract DSN: %v", err)
	}
	admin, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		t.Fatalf("connect PostgreSQL contract server: %v", err)
	}
	defer admin.Close(context.Background())
	database := fmt.Sprintf("sdpivot_bootstrap_%d", time.Now().UnixNano())
	identifier := pgx.Identifier{database}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+identifier); err != nil {
		t.Fatalf("create bootstrap contract database: %v", err)
	}
	defer func() { _, _ = admin.Exec(context.Background(), "DROP DATABASE "+identifier+" WITH (FORCE)") }()
	testConfig := config.Copy()
	testConfig.Database = database
	conn, err := pgx.ConnectConfig(ctx, testConfig)
	if err != nil {
		t.Fatalf("connect bootstrap contract database: %v", err)
	}
	defer conn.Close(context.Background())
	fn(ctx, conn)
}

func execBootstrapSQL(ctx context.Context, conn *pgx.Conn, sql string) error {
	results, err := conn.PgConn().Exec(ctx, sql).ReadAll()
	if err != nil {
		return err
	}
	for _, result := range results {
		if result.Err != nil {
			return result.Err
		}
	}
	return nil
}

func extractBootstrapTable(t *testing.T, migration, table string) string {
	t.Helper()
	pattern := regexp.MustCompile(`(?is)CREATE TABLE IF NOT EXISTS ` + regexp.QuoteMeta(table) + `\s*\(.*?\n\);`)
	definition := pattern.FindString(migration)
	if definition == "" {
		t.Fatalf("cannot extract %s fixture from bootstrap migration", table)
	}
	return definition
}

func historicalUUIDDefinition(t *testing.T, definition string) string {
	t.Helper()
	pattern := regexp.MustCompile(`(?m)^(\s*id\s+)VARCHAR\(36\)\s+PRIMARY KEY DEFAULT gen_random_uuid\(\)::TEXT,`)
	converted := pattern.ReplaceAllString(definition, `${1}UUID PRIMARY KEY DEFAULT gen_random_uuid(),`)
	if converted == definition {
		t.Fatal("fixture table ID definition was not converted to UUID")
	}
	return converted
}

func historicalBootstrapFixture(t *testing.T, migration string, mixed, currentAudit bool) string {
	t.Helper()
	parts := []string{`
		CREATE TABLE users (
			id VARCHAR(36) PRIMARY KEY,
			email VARCHAR(255) NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			tenant_id BIGINT NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			is_system_admin BOOLEAN NOT NULL DEFAULT FALSE,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE TABLE organizations (
			id VARCHAR(36) PRIMARY KEY,
			owner_id VARCHAR(36) NOT NULL,
			owner_tenant_id BIGINT NOT NULL
		);
		CREATE TABLE audit_logs (
			id BIGSERIAL PRIMARY KEY,
			tenant_id BIGINT NOT NULL,
			actor_user_id VARCHAR(36) NOT NULL DEFAULT '',
			actor_role VARCHAR(32) NOT NULL DEFAULT '',
			action VARCHAR(64) NOT NULL,
			target_type VARCHAR(32) NOT NULL DEFAULT '',
			target_id VARCHAR(64) NOT NULL DEFAULT '',
			target_user_id VARCHAR(36) NOT NULL DEFAULT '',
			request_path VARCHAR(512) NOT NULL DEFAULT '',
			request_method VARCHAR(16) NOT NULL DEFAULT '',
			outcome VARCHAR(16) NOT NULL DEFAULT 'success',
			details JSONB NOT NULL DEFAULT '{}'::JSONB,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX idx_audit_logs_tenant_id_desc ON audit_logs (tenant_id, id DESC);
		CREATE INDEX idx_audit_logs_actor ON audit_logs (actor_user_id);
		CREATE INDEX idx_audit_logs_tenant_action ON audit_logs (tenant_id, action);
		CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at);`}
	if currentAudit {
		parts = append(parts, `
			CREATE INDEX idx_audit_logs_tenant_created_at ON audit_logs (tenant_id, created_at DESC);
			ALTER TABLE audit_logs
				ADD COLUMN user_id VARCHAR(36) NOT NULL DEFAULT '',
				ADD COLUMN resource_type VARCHAR(32) NOT NULL DEFAULT '',
				ADD COLUMN resource_id VARCHAR(64) NOT NULL DEFAULT '',
				ADD COLUMN ip_address VARCHAR(45) NOT NULL DEFAULT '';
			CREATE INDEX idx_audit_logs_user_id ON audit_logs (user_id);`)
	}
	for _, table := range []string{"org_ext", "knowledge_spaces", "documents", "document_chunks", "document_versions", "chunk_strategies"} {
		definition := extractBootstrapTable(t, migration, table)
		if table == "document_chunks" || table == "document_versions" || (table == "chunk_strategies" && !mixed) {
			definition = historicalUUIDDefinition(t, definition)
		}
		parts = append(parts, definition)
	}
	return strings.Join(parts, "\n")
}

func TestSDPivotBootstrapAuditLogsPostgreSQLContract(t *testing.T) {
	migrationBytes, err := os.ReadFile("000012_sdpivot_op_baseline.up.sql")
	if err != nil {
		t.Fatalf("read bootstrap migration: %v", err)
	}
	migration := string(migrationBytes)

	tests := []struct {
		name         string
		currentAudit bool
		mutate       string
		wantError    string
		wantColumn   int
	}{
		{name: "accepts exact migration 44 audit schema", wantColumn: 19},
		{name: "accepts exact current Core audit schema", currentAudit: true, wantColumn: 21},
		{name: "rejects missing audit table", mutate: "DROP TABLE audit_logs", wantError: "requires exact Core migration 44 audit_logs contract"},
		{name: "rejects wrong Core column type", mutate: "ALTER TABLE audit_logs ALTER COLUMN actor_user_id TYPE TEXT", wantError: "requires exact Core migration 44 audit_logs contract"},
		{name: "rejects partial canonical projection", mutate: "ALTER TABLE audit_logs ADD COLUMN user_id VARCHAR(36)", wantError: "requires exact Core migration 44 audit_logs contract"},
		{name: "rejects wrong canonical projection type", mutate: `
			ALTER TABLE audit_logs ADD COLUMN user_id VARCHAR(36) NOT NULL DEFAULT '';
			ALTER TABLE audit_logs ADD COLUMN resource_type VARCHAR(32) NOT NULL DEFAULT '';
			ALTER TABLE audit_logs ADD COLUMN resource_id VARCHAR(64) NOT NULL DEFAULT '';
			ALTER TABLE audit_logs ADD COLUMN ip_address VARCHAR(46) NOT NULL DEFAULT '';`, wantError: "requires exact Core migration 44 audit_logs contract"},
		{name: "rejects wrong legacy projection type", currentAudit: true, mutate: `
			ALTER TABLE audit_logs ADD COLUMN username VARCHAR(100);
			ALTER TABLE audit_logs ADD COLUMN resource VARCHAR(100);
			ALTER TABLE audit_logs ADD COLUMN detail VARCHAR(255);
			ALTER TABLE audit_logs ADD COLUMN ip VARCHAR(50);`, wantError: "requires exact Core migration 44 audit_logs contract"},
		{name: "rejects unknown audit column", currentAudit: true, mutate: "ALTER TABLE audit_logs ADD COLUMN unexpected TEXT", wantError: "requires exact Core migration 44 audit_logs contract"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withBootstrapContractDatabase(t, func(ctx context.Context, conn *pgx.Conn) {
				if err := execBootstrapSQL(ctx, conn, historicalBootstrapFixture(t, migration, false, tt.currentAudit)); err != nil {
					t.Fatalf("create Core audit fixture: %v", err)
				}
				if tt.mutate != "" {
					if err := execBootstrapSQL(ctx, conn, tt.mutate); err != nil {
						t.Fatalf("mutate Core audit fixture: %v", err)
					}
				}

				err := execBootstrapSQL(ctx, conn, migration)
				if tt.wantError != "" {
					if err == nil {
						t.Fatalf("bootstrap accepted incompatible audit_logs fixture")
					}
					if !strings.Contains(strings.ToLower(err.Error()), tt.wantError) {
						t.Fatalf("bootstrap failed for unexpected reason: %v", err)
					}
					return
				}
				if err != nil {
					t.Fatalf("apply bootstrap to exact Core audit fixture: %v", err)
				}
				var count int
				if err := conn.QueryRow(ctx, `
					SELECT count(*)
					FROM information_schema.columns
					WHERE table_schema = 'public' AND table_name = 'audit_logs'`).Scan(&count); err != nil {
					t.Fatalf("count bootstrapped audit columns: %v", err)
				}
				if count != tt.wantColumn {
					t.Fatalf("audit_logs column count = %d, want %d", count, tt.wantColumn)
				}
			})
		})
	}
}

func TestSDPivotBootstrapHistoricalUUIDPostgreSQLContract(t *testing.T) {
	migrationBytes, err := os.ReadFile("000012_sdpivot_op_baseline.up.sql")
	if err != nil {
		t.Fatalf("read bootstrap migration: %v", err)
	}
	migration := string(migrationBytes)

	t.Run("preserves complete historical UUID profile", func(t *testing.T) {
		withBootstrapContractDatabase(t, func(ctx context.Context, conn *pgx.Conn) {
			if err := execBootstrapSQL(ctx, conn, historicalBootstrapFixture(t, migration, false, false)); err != nil {
				t.Fatalf("create historical UUID fixture: %v", err)
			}
			if err := execBootstrapSQL(ctx, conn, migration); err != nil {
				t.Fatalf("apply bootstrap to historical UUID fixture: %v", err)
			}
			rows, err := conn.Query(ctx, `
				SELECT table_name, data_type
				FROM information_schema.columns
				WHERE table_schema = 'public'
				  AND (table_name, column_name) IN (
					('org_ext', 'org_id'), ('knowledge_spaces', 'id'), ('documents', 'id'),
					('document_chunks', 'id'), ('document_versions', 'id'), ('chunk_strategies', 'id'))`)
			if err != nil {
				t.Fatalf("query historical ID profile: %v", err)
			}
			got := map[string]string{}
			for rows.Next() {
				var table, dataType string
				if err := rows.Scan(&table, &dataType); err != nil {
					rows.Close()
					t.Fatalf("scan historical ID profile: %v", err)
				}
				got[table] = dataType
			}
			rows.Close()
			if err := rows.Err(); err != nil {
				t.Fatalf("read historical ID profile: %v", err)
			}
			want := map[string]string{
				"org_ext": "character varying", "knowledge_spaces": "character varying", "documents": "character varying",
				"document_chunks": "uuid", "document_versions": "uuid", "chunk_strategies": "uuid",
			}
			for table, dataType := range want {
				if got[table] != dataType {
					t.Errorf("%s ID type = %q, want %q", table, got[table], dataType)
				}
			}
		})
	})

	t.Run("fails closed for mixed ID profile", func(t *testing.T) {
		withBootstrapContractDatabase(t, func(ctx context.Context, conn *pgx.Conn) {
			if err := execBootstrapSQL(ctx, conn, historicalBootstrapFixture(t, migration, true, false)); err != nil {
				t.Fatalf("create mixed ID fixture: %v", err)
			}
			err := execBootstrapSQL(ctx, conn, migration)
			if err == nil {
				t.Fatal("bootstrap accepted mixed UUID and varchar ID profile")
			}
			if !strings.Contains(strings.ToLower(err.Error()), "incompatible id type profile") {
				t.Fatalf("bootstrap failed for unexpected reason: %v", err)
			}
		})
	})
}
