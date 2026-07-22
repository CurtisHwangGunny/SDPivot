package postgres

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

const pgContractEnv = "SDPIVOT_PG_TEST_DSN"

var policyWhitespace = regexp.MustCompile(`[[:space:]()]`)

func requirePGContractDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv(pgContractEnv)
	if dsn == "" {
		t.Skipf("set %s to run PostgreSQL integration contracts", pgContractEnv)
	}
	return dsn
}

func withContractDatabase(t *testing.T, fn func(context.Context, *pgx.Conn)) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	config, err := pgx.ParseConfig(requirePGContractDSN(t))
	if err != nil {
		t.Fatalf("parse PostgreSQL contract DSN: %v", err)
	}
	admin, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		t.Fatalf("connect PostgreSQL contract server: %v", err)
	}
	defer admin.Close(context.Background())

	database := fmt.Sprintf("sdpivot_contract_%d", time.Now().UnixNano())
	identifier := pgx.Identifier{database}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+identifier); err != nil {
		t.Fatalf("create contract database: %v", err)
	}
	defer func() {
		_, _ = admin.Exec(context.Background(), "DROP DATABASE "+identifier+" WITH (FORCE)")
	}()

	testConfig := config.Copy()
	testConfig.Database = database
	conn, err := pgx.ConnectConfig(ctx, testConfig)
	if err != nil {
		t.Fatalf("connect contract database: %v", err)
	}
	defer conn.Close(context.Background())
	fn(ctx, conn)
}

func execContractSQL(t *testing.T, ctx context.Context, conn *pgx.Conn, sql string) error {
	t.Helper()
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

func normalizePolicyExpression(expression string) string {
	expression = strings.ToLower(expression)
	expression = policyWhitespace.ReplaceAllString(expression, "")
	expression = strings.ReplaceAll(expression, "::text", "")
	return strings.ReplaceAll(expression, "public.", "")
}

func validateSecureTenantCatalog(ctx context.Context, conn *pgx.Conn) error {
	expected := map[string]string{
		"write_category_config": "sdpivot_secure_000014_write_category_config",
		"knowledge_spaces":      "sdpivot_secure_000014_knowledge_spaces",
		"documents":             "sdpivot_secure_000014_documents",
		"qa_sessions":           "sdpivot_secure_000014_qa_sessions",
		"qa_messages":           "sdpivot_secure_000014_qa_messages",
		"writing_drafts":        "sdpivot_secure_000014_writing_drafts",
		"announcements":         "sdpivot_secure_000014_announcements",
		"token_usage":           "sdpivot_secure_000014_token_usage",
		"document_chunks":       "sdpivot_secure_000014_document_chunks",
	}

	rows, err := conn.Query(ctx, `
		SELECT c.relname, c.relrowsecurity, c.relforcerowsecurity
		FROM pg_catalog.pg_class c
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public' AND c.relname = ANY($1)`, secureTenantTables)
	if err != nil {
		return err
	}
	seenTables := 0
	for rows.Next() {
		var table string
		var enabled, forced bool
		if err := rows.Scan(&table, &enabled, &forced); err != nil {
			rows.Close()
			return err
		}
		seenTables++
		if !enabled || (table == "document_chunks" && !forced) {
			rows.Close()
			return fmt.Errorf("invalid RLS flags for %s", table)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if seenTables != len(expected) {
		return fmt.Errorf("found %d target tables, want %d", seenTables, len(expected))
	}

	rows, err = conn.Query(ctx, `
		SELECT c.relname, p.polname, p.polcmd::text, p.polpermissive,
		       (p.polroles::oid[]) = ARRAY[0::oid],
		       COALESCE(pg_get_expr(p.polqual, p.polrelid), ''),
		       COALESCE(pg_get_expr(p.polwithcheck, p.polrelid), '')
		FROM pg_catalog.pg_policy p
		JOIN pg_catalog.pg_class c ON c.oid = p.polrelid
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public' AND c.relname = ANY($1)`, secureTenantTables)
	if err != nil {
		return err
	}
	seenPolicies := make(map[string]bool, len(expected))
	for rows.Next() {
		var table, name, command, usingExpression, checkExpression string
		var permissive, publicRole bool
		if err := rows.Scan(&table, &name, &command, &permissive, &publicRole, &usingExpression, &checkExpression); err != nil {
			rows.Close()
			return err
		}
		if !permissive {
			continue
		}
		if expected[table] != name {
			rows.Close()
			return fmt.Errorf("unexpected permissive policy %s on %s", name, table)
		}
		if seenPolicies[table] {
			rows.Close()
			return fmt.Errorf("duplicate permissive policy on %s", table)
		}
		seenPolicies[table] = true
		if command != "*" || !publicRole {
			rows.Close()
			return fmt.Errorf("invalid command or roles for %s", name)
		}
		want := "tenant_id=get_current_tenant_id"
		if table == "document_chunks" {
			want = "existsselect1fromdocumentsdwhered.id=document_chunks.document_idandd.tenant_id=document_chunks.tenant_idandd.tenant_id=get_current_tenant_id"
		}
		if normalizePolicyExpression(usingExpression) != want || normalizePolicyExpression(checkExpression) != want {
			rows.Close()
			return fmt.Errorf("invalid USING or WITH CHECK for %s", name)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(seenPolicies) != len(expected) {
		return fmt.Errorf("found %d expected permissive policies, want %d", len(seenPolicies), len(expected))
	}

	functionRows, err := conn.Query(ctx, `
		SELECT p.proname, p.prosecdef, COALESCE(array_to_string(p.proconfig, E'\n'), ''), p.prosrc,
		       CASE p.proname WHEN 'set_tenant_context' THEN p.prorettype = 'void'::regtype
		                        ELSE p.prorettype = 'boolean'::regtype END
		FROM pg_catalog.pg_proc p
		JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
		WHERE n.nspname = 'public'
		  AND p.oid IN (to_regprocedure('public.set_tenant_context(bigint,boolean)'), to_regprocedure('public.is_ops_admin_context()'))`)
	if err != nil {
		return err
	}
	seenFunctions := 0
	for functionRows.Next() {
		var name, config, source string
		var securityDefiner, returnTypeValid bool
		if err := functionRows.Scan(&name, &securityDefiner, &config, &source, &returnTypeValid); err != nil {
			functionRows.Close()
			return err
		}
		seenFunctions++
		if securityDefiner || !returnTypeValid || config != "search_path=pg_catalog, public" {
			functionRows.Close()
			return fmt.Errorf("invalid execution contract for %s", name)
		}
		normalizedSource := strings.Join(strings.Fields(strings.ToLower(source)), "")
		want := "selectfalse;"
		if name == "set_tenant_context" {
			want = "beginperformpg_catalog.set_config('app.current_tenant_id',p_tenant_id::text,true);performpg_catalog.set_config('app.is_ops_admin','false',true);end;"
		}
		if normalizedSource != want {
			functionRows.Close()
			return fmt.Errorf("invalid prosrc for %s", name)
		}
	}
	functionRows.Close()
	if err := functionRows.Err(); err != nil {
		return err
	}
	if seenFunctions != 2 {
		return fmt.Errorf("found %d secure context functions, want 2", seenFunctions)
	}
	return nil
}

func TestSecureOPTenantRLSPostgreSQLCatalogContract(t *testing.T) {
	withContractDatabase(t, func(ctx context.Context, conn *pgx.Conn) {
		fixture := `
			CREATE FUNCTION get_current_tenant_id() RETURNS BIGINT LANGUAGE sql STABLE AS $$ SELECT 1::BIGINT $$;
			CREATE TABLE documents (id VARCHAR(36), tenant_id BIGINT);
			CREATE TABLE document_chunks (id VARCHAR(36), document_id VARCHAR(36), tenant_id BIGINT);
			CREATE TABLE write_category_config (id VARCHAR(36), tenant_id BIGINT);
			CREATE TABLE knowledge_spaces (id VARCHAR(36), tenant_id BIGINT);
			CREATE TABLE qa_sessions (id VARCHAR(36), tenant_id BIGINT);
			CREATE TABLE qa_messages (id VARCHAR(36), tenant_id BIGINT);
			CREATE TABLE writing_drafts (id VARCHAR(36), tenant_id BIGINT);
			CREATE TABLE announcements (id VARCHAR(36), tenant_id BIGINT);
			CREATE TABLE token_usage (id VARCHAR(36), tenant_id BIGINT);`
		if err := execContractSQL(t, ctx, conn, fixture); err != nil {
			t.Fatalf("create v14 fixture: %v", err)
		}
		migrationBytes, err := os.ReadFile("000014_secure_op_tenant_rls.up.sql")
		if err != nil {
			t.Fatalf("read v14 migration: %v", err)
		}
		migration := string(migrationBytes)
		reset := func() {
			t.Helper()
			_ = execContractSQL(t, ctx, conn, "DROP POLICY IF EXISTS sdpivot_contract_extra ON documents; DROP ROLE IF EXISTS sdpivot_contract_role;")
			if err := execContractSQL(t, ctx, conn, migration); err != nil {
				t.Fatalf("apply v14 migration: %v", err)
			}
		}
		expectInvalid := func(name, mutation string) {
			t.Helper()
			t.Run(name, func(t *testing.T) {
				reset()
				if err := execContractSQL(t, ctx, conn, mutation); err != nil {
					t.Fatalf("apply catalog mutation: %v", err)
				}
				if err := validateSecureTenantCatalog(ctx, conn); err == nil {
					t.Fatal("catalog validator accepted an invalid runtime contract")
				}
			})
		}

		reset()
		if err := validateSecureTenantCatalog(ctx, conn); err != nil {
			t.Fatalf("valid v14 catalog rejected: %v", err)
		}

		t.Run("normalizes public casts parentheses and whitespace", func(t *testing.T) {
			reset()
			if err := execContractSQL(t, ctx, conn, `
				DROP POLICY sdpivot_secure_000014_documents ON documents;
				CREATE POLICY sdpivot_secure_000014_documents ON documents FOR ALL
				USING (((tenant_id)::text = (public.get_current_tenant_id())::text))
				WITH CHECK (((tenant_id)::text = (public.get_current_tenant_id())::text));`); err != nil {
				t.Fatalf("create normalized equivalent policy: %v", err)
			}
			if err := validateSecureTenantCatalog(ctx, conn); err != nil {
				t.Fatalf("equivalent normalized policy rejected: %v", err)
			}
		})

		expectInvalid("rejects equivalent ops bypass", `
			DROP POLICY sdpivot_secure_000014_documents ON documents;
			CREATE POLICY sdpivot_secure_000014_documents ON documents FOR ALL
			USING (tenant_id = get_current_tenant_id() OR public.is_ops_admin_context())
			WITH CHECK (tenant_id = get_current_tenant_id() OR public.is_ops_admin_context());`)
		expectInvalid("rejects extra permissive policy", `CREATE POLICY sdpivot_contract_extra ON documents FOR SELECT USING (TRUE);`)
		expectInvalid("rejects non public roles", `CREATE ROLE sdpivot_contract_role; ALTER POLICY sdpivot_secure_000014_documents ON documents TO sdpivot_contract_role;`)
		expectInvalid("rejects command other than for all", `
			DROP POLICY sdpivot_secure_000014_documents ON documents;
			CREATE POLICY sdpivot_secure_000014_documents ON documents FOR SELECT USING (tenant_id = get_current_tenant_id());`)
		expectInvalid("rejects mismatched using", `
			DROP POLICY sdpivot_secure_000014_documents ON documents;
			CREATE POLICY sdpivot_secure_000014_documents ON documents FOR ALL
			USING (tenant_id = get_current_tenant_id() + 1)
			WITH CHECK (tenant_id = get_current_tenant_id());`)
		expectInvalid("rejects mismatched with check", `
			DROP POLICY sdpivot_secure_000014_documents ON documents;
			CREATE POLICY sdpivot_secure_000014_documents ON documents FOR ALL
			USING (tenant_id = get_current_tenant_id())
			WITH CHECK (tenant_id = get_current_tenant_id() + 1);`)
		expectInvalid("rejects security definer", `ALTER FUNCTION set_tenant_context(BIGINT, BOOLEAN) SECURITY DEFINER;`)
		expectInvalid("rejects set tenant proconfig", `ALTER FUNCTION set_tenant_context(BIGINT, BOOLEAN) SET search_path = public;`)
		expectInvalid("rejects set tenant prosrc", `
			CREATE OR REPLACE FUNCTION set_tenant_context(p_tenant_id BIGINT, p_is_ops_admin BOOLEAN DEFAULT FALSE)
			RETURNS void LANGUAGE plpgsql SECURITY INVOKER SET search_path = pg_catalog, public AS $$
			BEGIN
				PERFORM pg_catalog.set_config('app.current_tenant_id', p_tenant_id::TEXT, true);
				PERFORM pg_catalog.set_config('app.is_ops_admin', 'true', true);
			END;
			$$;`)
		expectInvalid("rejects ops admin prosrc", `
			CREATE OR REPLACE FUNCTION is_ops_admin_context() RETURNS BOOLEAN LANGUAGE sql STABLE
			SECURITY INVOKER SET search_path = pg_catalog, public AS $$ SELECT TRUE; $$;`)
	})
}
