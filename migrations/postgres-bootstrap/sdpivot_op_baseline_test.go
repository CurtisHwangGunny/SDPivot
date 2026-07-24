package postgresbootstrap

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func readBaselineSQL(t *testing.T, name string) string {
	t.Helper()
	content, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return strings.ToLower(string(content))
}

func normalizeSQL(sql string) string {
	return strings.Join(strings.Fields(strings.ToLower(sql)), " ")
}

func requireFragments(t *testing.T, sql string, fragments ...string) {
	t.Helper()
	normalized := normalizeSQL(sql)
	for _, fragment := range fragments {
		if !strings.Contains(normalized, normalizeSQL(fragment)) {
			t.Errorf("migration missing %q", fragment)
		}
	}
}

func requireOrder(t *testing.T, sql string, steps ...string) {
	t.Helper()
	normalized := normalizeSQL(sql)
	previous := -1
	for _, step := range steps {
		position := strings.Index(normalized, normalizeSQL(step))
		if position < 0 {
			t.Fatalf("migration missing ordered step %q", step)
		}
		if position <= previous {
			t.Fatalf("migration step %q is out of order", step)
		}
		previous = position
	}
}

func TestBaselineFailsClosedWithoutVersionedCore(t *testing.T) {
	sql := readBaselineSQL(t, "000012_sdpivot_op_baseline.up.sql")
	requireFragments(t, sql,
		"PRECONDITION: migrations/versioned",
		"to_regclass('public.users') IS NULL",
		"to_regclass('public.organizations') IS NULL",
		"information_schema.columns",
		"('users', 'id', ARRAY['character varying', 'text'])",
		"('users', 'email', ARRAY['character varying', 'text'])",
		"('users', 'password_hash', ARRAY['character varying', 'text'])",
		"('users', 'tenant_id', ARRAY['integer', 'bigint'])",
		"('users', 'is_active', ARRAY['boolean'])",
		"('users', 'is_system_admin', ARRAY['boolean'])",
		"('organizations', 'id', ARRAY['character varying', 'text'])",
		"('organizations', 'owner_id', ARRAY['character varying', 'text'])",
		"('organizations', 'owner_tenant_id', ARRAY['bigint'])",
		"existing.data_type = ANY(required.allowed_types)",
		"pg_catalog.pg_constraint",
		"key_constraint.contype IN ('p', 'u')",
		"array_length(key_constraint.conkey, 1) = 1",
		"RAISE EXCEPTION 'SDPivot OP bootstrap requires compatible completed core migrations",
		"RAISE EXCEPTION 'SDPivot OP bootstrap requires PRIMARY KEY or UNIQUE constraints",
	)

	if regexp.MustCompile(`create\s+table\s+(if\s+not\s+exists\s+)?(users|organizations)\b`).MatchString(sql) {
		t.Error("bootstrap must not create shared core tables")
	}
	requireOrder(t, sql,
		"to_regclass('public.users') IS NULL",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS is_ops_admin",
		"CREATE TABLE IF NOT EXISTS org_ext",
	)
}

func TestBaselineChecksExistingTableCompatibilityBeforeHelpersAndPolicies(t *testing.T) {
	sql := readBaselineSQL(t, "000012_sdpivot_op_baseline.up.sql")
	requireFragments(t, sql,
		"CREATE OR REPLACE FUNCTION sdpivot_op_bootstrap_000012_assert_schema(p_require_all BOOLEAN)",
		"SDPivot OP bootstrap schema compatibility check failed",
		"p_require_all OR (required.table_name <> 'users' AND to_regclass(format('public.%I', required.table_name)) IS NOT NULL)",
		"SELECT sdpivot_op_bootstrap_000012_assert_schema(FALSE)",
		"SELECT sdpivot_op_bootstrap_000012_assert_schema(TRUE)",
		"DROP FUNCTION sdpivot_op_bootstrap_000012_assert_schema(BOOLEAN)",
		"('org_ext', 'org_id', ARRAY['character varying'])",
		"('org_ext', 'tenant_id', ARRAY['bigint'])",
		"('org_members', 'org_id', ARRAY['character varying'])",
		"('smartknora_user_profiles', 'user_id', ARRAY['character varying'])",
		"('refresh_tokens', 'user_id', ARRAY['character varying'])",
		"('knowledge_spaces', 'id', ARRAY['character varying'])",
		"('knowledge_spaces', 'tenant_id', ARRAY['bigint'])",
		"('space_members', 'space_id', ARRAY['character varying'])",
		"('documents', 'id', ARRAY['character varying'])",
		"('documents', 'tenant_id', ARRAY['bigint'])",
		"('document_chunks', 'id', ARRAY['character varying', 'uuid'])",
		"('document_versions', 'id', ARRAY['character varying', 'uuid'])",
		"('chunk_strategies', 'id', ARRAY['character varying', 'uuid'])",
		"('document_chunks', 'document_id', ARRAY['character varying'])",
		"('document_chunks', 'tenant_id', ARRAY['bigint'])",
		"('qa_messages', 'session_id', ARRAY['character varying'])",
		"('audit_logs', 'tenant_id', ARRAY['bigint'])",
	)
	requireOrder(t, sql,
		"CREATE OR REPLACE FUNCTION sdpivot_op_bootstrap_000012_assert_schema",
		"SELECT sdpivot_op_bootstrap_000012_assert_schema(FALSE)",
		"CREATE TABLE IF NOT EXISTS org_ext",
		"CREATE TABLE IF NOT EXISTS invoices",
		"SELECT sdpivot_op_bootstrap_000012_assert_schema(TRUE)",
		"DROP FUNCTION sdpivot_op_bootstrap_000012_assert_schema(BOOLEAN)",
		"CREATE OR REPLACE FUNCTION set_tenant_context",
		"CREATE POLICY sdpivot_op_bootstrap_000012_organizations",
	)
}

func TestBaselineAcceptsOnlyCompleteHistoricalOrBootstrapIDProfiles(t *testing.T) {
	sql := readBaselineSQL(t, "000012_sdpivot_op_baseline.up.sql")
	requireFragments(t, sql,
		"max(data_type) FILTER (WHERE table_name = 'org_ext' AND column_name = 'org_id') AS org_ext_id",
		"max(data_type) FILTER (WHERE table_name = 'knowledge_spaces' AND column_name = 'id') AS space_id",
		"max(data_type) FILTER (WHERE table_name = 'documents' AND column_name = 'id') AS document_id",
		"max(data_type) FILTER (WHERE table_name = 'document_chunks' AND column_name = 'id') AS chunk_id",
		"max(data_type) FILTER (WHERE table_name = 'document_versions' AND column_name = 'id') AS version_id",
		"max(data_type) FILTER (WHERE table_name = 'chunk_strategies' AND column_name = 'id') AS strategy_id",
		"id_type_profile.org_ext_id = 'character varying'",
		"id_type_profile.space_id = 'character varying'",
		"id_type_profile.document_id = 'character varying'",
		"id_type_profile.chunk_id = 'uuid'",
		"id_type_profile.version_id = 'uuid'",
		"id_type_profile.strategy_id = 'uuid'",
		"id_type_profile.chunk_id = 'character varying'",
		"id_type_profile.version_id = 'character varying'",
		"id_type_profile.strategy_id = 'character varying'",
		"RAISE EXCEPTION 'SDPivot OP bootstrap schema compatibility check failed: incompatible ID type profile'",
	)

	profileCheck := regexp.MustCompile(`(?s)and\s+not\s*\(\s*\(.*?org_ext_id\s*=\s*'character varying'.*?space_id\s*=\s*'character varying'.*?document_id\s*=\s*'character varying'.*?chunk_id\s*=\s*'uuid'.*?version_id\s*=\s*'uuid'.*?strategy_id\s*=\s*'uuid'\s*\)\s*or\s*\(.*?org_ext_id\s*=\s*'character varying'.*?space_id\s*=\s*'character varying'.*?document_id\s*=\s*'character varying'.*?chunk_id\s*=\s*'character varying'.*?version_id\s*=\s*'character varying'.*?strategy_id\s*=\s*'character varying'\s*\)\s*\)`).FindString(sql)
	if profileCheck == "" {
		t.Fatal("bootstrap compatibility assertion must accept exactly the historical and Bootstrap ID profiles and reject mixed profiles")
	}
}

func TestBaselineIndexedColumnsAreCoveredByCompatibilitySpecs(t *testing.T) {
	sql := readBaselineSQL(t, "000012_sdpivot_op_baseline.up.sql")

	assertionEnd := strings.Index(sql, "-- historical tenant helpers")
	if assertionEnd < 0 {
		t.Fatal("tenant helper section missing")
	}
	assertionSQL := sql[:assertionEnd]
	covered := make(map[string]bool)
	for _, match := range regexp.MustCompile(`\('([a-z0-9_]+)',\s*'([a-z0-9_]+)',\s*array\[`).FindAllStringSubmatch(assertionSQL, -1) {
		covered[match[1]+"."+match[2]] = true
	}

	indexPattern := regexp.MustCompile(`(?m)create\s+index\s+if\s+not\s+exists\s+[a-z0-9_]+\s+on\s+([a-z0-9_]+)\s*\(([^)]*)\)(?:\s+where\s+[^;]+)?;`)
	indexes := indexPattern.FindAllStringSubmatch(sql, -1)
	if len(indexes) == 0 {
		t.Fatal("no ordinary CREATE INDEX statements found")
	}
	for _, index := range indexes {
		table := index[1]
		for _, rawColumn := range strings.Split(index[2], ",") {
			column := strings.TrimSpace(rawColumn)
			if !regexp.MustCompile(`^[a-z_][a-z0-9_]*$`).MatchString(column) {
				t.Fatalf("expression index column %q on %s needs explicit test handling", column, table)
			}
			if !covered[table+"."+column] {
				t.Errorf("indexed column %s.%s is missing from compatibility assertion specs", table, column)
			}
		}
	}
}

func TestBaselinePreservesTenantIsolationSafetyContract(t *testing.T) {
	sql := readBaselineSQL(t, "000012_sdpivot_op_baseline.up.sql")
	requireFragments(t, sql,
		"CREATE OR REPLACE FUNCTION set_tenant_context(p_tenant_id BIGINT, p_is_ops_admin BOOLEAN DEFAULT FALSE)",
		"PERFORM set_config('app.is_ops_admin', 'false', false)",
		"LANGUAGE plpgsql SECURITY INVOKER",
		"CREATE OR REPLACE FUNCTION is_ops_admin_context() RETURNS BOOLEAN AS $$ SELECT FALSE; $$ LANGUAGE sql STABLE SECURITY INVOKER",
		"USING (owner_tenant_id = get_current_tenant_id())",
		"WITH CHECK (owner_tenant_id = get_current_tenant_id())",
		"ALTER TABLE document_chunks FORCE ROW LEVEL SECURITY",
	)
	policyStart := strings.Index(sql, "-- bootstrap policies")
	if policyStart < 0 {
		t.Fatal("bootstrap policy section missing")
	}
	if strings.Contains(sql[policyStart:], "is_ops_admin_context()") {
		t.Error("schema compatibility changes must not restore the operations bypass")
	}
}

func TestBaselineCreatesObjectsBeforePolicies(t *testing.T) {
	sql := readBaselineSQL(t, "000012_sdpivot_op_baseline.up.sql")
	for _, table := range []string{
		"org_ext", "org_members", "smartknora_user_profiles", "refresh_tokens", "token_usage",
		"knowledge_spaces", "space_members", "space_categories", "documents", "document_chunks",
		"document_versions", "chunk_strategies", "qa_sessions", "qa_messages", "writing_drafts",
		"write_category_config", "announcements", "audit_logs", "sensitive_words", "billing_plans",
		"enterprise_subscriptions", "invoices",
	} {
		requireFragments(t, sql, "CREATE TABLE IF NOT EXISTS "+table)
	}

	requireOrder(t, sql,
		"CREATE TABLE IF NOT EXISTS space_categories",
		"ALTER TABLE space_categories ENABLE ROW LEVEL SECURITY",
		"CREATE POLICY sdpivot_op_bootstrap_000012_space_categories ON space_categories",
	)
	requireOrder(t, sql,
		"CREATE TABLE IF NOT EXISTS documents",
		"CREATE TABLE IF NOT EXISTS document_chunks",
		"ALTER TABLE document_chunks ENABLE ROW LEVEL SECURITY",
		"CREATE POLICY sdpivot_op_bootstrap_000012_document_chunks ON document_chunks",
	)

	policies := regexp.MustCompile(`create\s+policy\s+(sdpivot_op_bootstrap_000012_[a-z0-9_]+)\s+on\s+([a-z0-9_]+)`).FindAllStringSubmatch(sql, -1)
	if len(policies) < 10 {
		t.Fatalf("expected identifiable bootstrap policies, found %d", len(policies))
	}
	seen := make(map[string]bool)
	for _, policy := range policies {
		if seen[policy[1]] {
			t.Errorf("duplicate bootstrap policy name %s", policy[1])
		}
		seen[policy[1]] = true
	}
}

func TestBaselineContainsNoFixedAccountRoleOrSeedData(t *testing.T) {
	sql := readBaselineSQL(t, "000012_sdpivot_op_baseline.up.sql")
	for _, forbidden := range []string{
		"admin@smartknora.com",
		"$2a$10$l4fdcgy48s7whdmzzqaaf.sql5nna1.0uewfbbvzvasmsv0qyugs2",
		"smartknora@2026",
		"insert into users",
		"create role ops_admin",
		"create role app_user",
	} {
		if strings.Contains(sql, forbidden) {
			t.Errorf("bootstrap contains forbidden account/role fragment %q", forbidden)
		}
	}

	if regexp.MustCompile(`\binsert\s+into\s+(announcements|sensitive_words|billing_plans|enterprise_subscriptions|invoices)\b`).MatchString(sql) {
		t.Error("bootstrap must not seed retired SaaS operations data")
	}
}

func TestBaselineUsesQuotedStringDefaults(t *testing.T) {
	sql := readBaselineSQL(t, "000012_sdpivot_op_baseline.up.sql")
	requireFragments(t, sql,
		"role VARCHAR(20) NOT NULL DEFAULT 'member'",
		"status VARCHAR(20) NOT NULL DEFAULT 'active'",
		"visibility VARCHAR(20) DEFAULT 'private'",
		"strategy_type VARCHAR(50) NOT NULL DEFAULT 'fixed_size'",
		"source_type VARCHAR(30) NOT NULL DEFAULT 'knowledge_base'",
	)

	unquoted := regexp.MustCompile(`(?m)\bdefault\s+(member|active|trial|free|private|viewer|pending|draft|general|fixed_size|knowledge_base)\b`).FindString(sql)
	if unquoted != "" {
		t.Errorf("string default must be quoted: %q", unquoted)
	}
}

func TestBaselineHasNoOpsAdminRLSBypass(t *testing.T) {
	sql := readBaselineSQL(t, "000012_sdpivot_op_baseline.up.sql")
	requireFragments(t, sql,
		"CREATE OR REPLACE FUNCTION set_tenant_context(p_tenant_id BIGINT, p_is_ops_admin BOOLEAN DEFAULT FALSE)",
		"PERFORM set_config('app.current_tenant_id', p_tenant_id::TEXT, false)",
		"PERFORM set_config('app.is_ops_admin', 'false', false)",
		"LANGUAGE plpgsql SECURITY INVOKER",
		"CREATE OR REPLACE FUNCTION is_ops_admin_context() RETURNS BOOLEAN AS $$ SELECT FALSE; $$ LANGUAGE sql STABLE SECURITY INVOKER",
	)
	if strings.Contains(sql, "security definer") {
		t.Error("bootstrap must not create SECURITY DEFINER functions")
	}
	if strings.Contains(sql, "current_setting('app.is_ops_admin'") {
		t.Error("bootstrap must not read the operations-admin GUC")
	}

	policyStart := strings.Index(sql, "-- bootstrap policies")
	if policyStart < 0 {
		t.Fatal("bootstrap policy section missing")
	}
	policySQL := sql[policyStart:]
	if strings.Contains(policySQL, "is_ops_admin_context()") || strings.Contains(policySQL, "app.is_ops_admin") {
		t.Error("bootstrap policies must contain only tenant isolation checks")
	}
}

func TestBaselineRLSAndSafeTenantContext(t *testing.T) {
	sql := readBaselineSQL(t, "000012_sdpivot_op_baseline.up.sql")
	requireFragments(t, sql,
		"current_setting('app.current_tenant_id', true)",
		"ALTER TABLE document_chunks ENABLE ROW LEVEL SECURITY",
		"ALTER TABLE document_chunks FORCE ROW LEVEL SECURITY",
		"CREATE POLICY sdpivot_op_bootstrap_000012_document_chunks ON document_chunks",
		"FROM documents d",
		"d.id = document_chunks.document_id",
		"d.tenant_id = document_chunks.tenant_id",
		"d.tenant_id = get_current_tenant_id()",
	)

	chunkPolicy := regexp.MustCompile(`(?s)create\s+policy\s+sdpivot_op_bootstrap_000012_document_chunks.*?using\s*\(.*?d\.tenant_id\s*=\s*document_chunks\.tenant_id.*?d\.tenant_id\s*=\s*get_current_tenant_id\(\).*?with\s+check\s*\(.*?d\.tenant_id\s*=\s*document_chunks\.tenant_id.*?d\.tenant_id\s*=\s*get_current_tenant_id\(\)`).FindString(sql)
	if chunkPolicy == "" {
		t.Error("document_chunks USING and WITH CHECK must both bind chunk tenant to its document and current tenant")
	}
}

func TestBaselineDownIsConservative(t *testing.T) {
	sql := readBaselineSQL(t, "000012_sdpivot_op_baseline.down.sql")
	requireFragments(t, sql,
		"to_regclass('public.' || target.table_name) IS NOT NULL",
		"DROP POLICY IF EXISTS %I ON %I",
		"sdpivot_op_bootstrap_000012_document_chunks",
	)

	forbidden := regexp.MustCompile(`\b(drop\s+table|drop\s+column|delete\s+from|truncate\b|alter\s+table\s+\S+\s+disable\s+row\s+level\s+security|no\s+force\s+row\s+level\s+security)`).FindString(sql)
	if forbidden != "" {
		t.Errorf("down migration contains destructive operation %q", forbidden)
	}
}
