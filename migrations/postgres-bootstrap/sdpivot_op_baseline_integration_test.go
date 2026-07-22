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

func historicalBootstrapFixture(t *testing.T, migration string, mixed bool) string {
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
		);`}
	for _, table := range []string{"org_ext", "knowledge_spaces", "documents", "document_chunks", "document_versions", "chunk_strategies"} {
		definition := extractBootstrapTable(t, migration, table)
		if table == "document_chunks" || table == "document_versions" || (table == "chunk_strategies" && !mixed) {
			definition = historicalUUIDDefinition(t, definition)
		}
		parts = append(parts, definition)
	}
	return strings.Join(parts, "\n")
}

func TestSDPivotBootstrapHistoricalUUIDPostgreSQLContract(t *testing.T) {
	migrationBytes, err := os.ReadFile("000012_sdpivot_op_baseline.up.sql")
	if err != nil {
		t.Fatalf("read bootstrap migration: %v", err)
	}
	migration := string(migrationBytes)

	t.Run("preserves complete historical UUID profile", func(t *testing.T) {
		withBootstrapContractDatabase(t, func(ctx context.Context, conn *pgx.Conn) {
			if err := execBootstrapSQL(ctx, conn, historicalBootstrapFixture(t, migration, false)); err != nil {
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
			if err := execBootstrapSQL(ctx, conn, historicalBootstrapFixture(t, migration, true)); err != nil {
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
