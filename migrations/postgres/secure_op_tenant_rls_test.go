package postgres

import (
	"regexp"
	"strings"
	"testing"
)

var secureTenantTables = []string{
	"write_category_config",
	"knowledge_spaces",
	"documents",
	"qa_sessions",
	"qa_messages",
	"writing_drafts",
	"announcements",
	"token_usage",
	"document_chunks",
}

func TestSecureOPTenantRLSUpUsesTransactionLocalInvokerContext(t *testing.T) {
	sql := readSQL(t, "000014_secure_op_tenant_rls.up.sql")
	requireSQLFragments(t, sql,
		"CREATE OR REPLACE FUNCTION set_tenant_context(p_tenant_id BIGINT, p_is_ops_admin BOOLEAN DEFAULT FALSE)",
		"SECURITY INVOKER",
		"SET search_path = pg_catalog, public",
		"pg_catalog.set_config('app.current_tenant_id', p_tenant_id::TEXT, true)",
		"pg_catalog.set_config('app.is_ops_admin', 'false', true)",
		"CREATE OR REPLACE FUNCTION is_ops_admin_context()",
		"SELECT FALSE",
	)
	for _, forbidden := range []string{
		"security definer",
		"current_setting('app.is_ops_admin'",
		"is_ops_admin_context() or",
	} {
		if strings.Contains(sql, forbidden) {
			t.Errorf("up migration contains forbidden bypass fragment %q", forbidden)
		}
	}
}

func TestSecureOPTenantRLSUpFailsClosedAndReplacesBothPolicyOrigins(t *testing.T) {
	sql := readSQL(t, "000014_secure_op_tenant_rls.up.sql")
	requireSQLFragments(t, sql,
		"missing_tables := array_append(missing_tables, target_table)",
		"RAISE EXCEPTION 'SDPivot secure tenant RLS requires target tables",
	)

	historical := map[string][]string{
		"write_category_config": {"wcc_tenant_isolation"},
		"knowledge_spaces":      {"ks_tenant_isolation", "tenant_isolation"},
		"documents":             {"doc_tenant_isolation", "tenant_isolation"},
		"qa_sessions":           {"qs_tenant_isolation", "tenant_isolation"},
		"qa_messages":           {"qm_tenant_isolation", "qa_messages_tenant_isolation"},
		"writing_drafts":        {"wd_tenant_isolation", "tenant_isolation"},
		"announcements":         {"ann_tenant_isolation", "announcements_tenant_isolation"},
		"token_usage":           {"tu_tenant_isolation", "tenant_isolation"},
		"document_chunks":       {"tenant_isolation", "document_chunks_tenant_isolation_000012"},
	}
	for _, table := range secureTenantTables {
		for _, policy := range historical[table] {
			requireSQLFragments(t, sql, "DROP POLICY IF EXISTS "+policy+" ON "+table)
		}
		requireSQLFragments(t, sql,
			"DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_"+table+" ON "+table,
			"CREATE POLICY sdpivot_secure_000014_"+table+" ON "+table,
		)
	}
}

func TestSecureOPTenantRLSUpCreatesPureTenantPolicies(t *testing.T) {
	sql := readSQL(t, "000014_secure_op_tenant_rls.up.sql")
	for _, table := range secureTenantTables[:8] {
		requireSQLFragments(t, sql,
			"CREATE POLICY sdpivot_secure_000014_"+table+" ON "+table,
			"USING (tenant_id = get_current_tenant_id())",
			"WITH CHECK (tenant_id = get_current_tenant_id())",
		)
	}

	requireSQLFragments(t, sql,
		"ALTER TABLE document_chunks ENABLE ROW LEVEL SECURITY",
		"ALTER TABLE document_chunks FORCE ROW LEVEL SECURITY",
		"CREATE POLICY sdpivot_secure_000014_document_chunks ON document_chunks",
		"d.id = document_chunks.document_id",
		"d.tenant_id = document_chunks.tenant_id",
		"d.tenant_id = get_current_tenant_id()",
	)
	chunkPolicy := regexp.MustCompile(`(?s)create\s+policy\s+sdpivot_secure_000014_document_chunks.*?using\s*\(.*?d\.tenant_id\s*=\s*document_chunks\.tenant_id.*?d\.tenant_id\s*=\s*get_current_tenant_id\(\).*?with\s+check\s*\(.*?d\.tenant_id\s*=\s*document_chunks\.tenant_id.*?d\.tenant_id\s*=\s*get_current_tenant_id\(\)`).FindString(sql)
	if chunkPolicy == "" {
		t.Error("document_chunks USING and WITH CHECK must enforce identical document-linked tenant isolation")
	}
}

func TestSecureOPTenantRLSDownRemainsSafeAndGuarded(t *testing.T) {
	sql := readSQL(t, "000014_secure_op_tenant_rls.down.sql")
	requireSQLFragments(t, sql,
		"SECURITY INVOKER",
		"SET search_path = pg_catalog, public",
		"pg_catalog.set_config('app.current_tenant_id', p_tenant_id::TEXT, true)",
		"pg_catalog.set_config('app.is_ops_admin', 'false', true)",
		"SELECT FALSE",
		"IF to_regclass('public.' || target_table) IS NULL THEN",
		"DROP POLICY IF EXISTS %I ON %I",
		"CREATE POLICY document_chunks_tenant_isolation_000012 ON document_chunks",
		"ALTER TABLE document_chunks FORCE ROW LEVEL SECURITY",
	)
	for _, forbidden := range []string{
		"security definer",
		"current_setting('app.is_ops_admin'",
		"is_ops_admin_context() or",
		"disable row level security",
		"no force row level security",
	} {
		if strings.Contains(sql, forbidden) {
			t.Errorf("down migration restores unsafe behavior %q", forbidden)
		}
	}
}

func TestSecureOPTenantRLSHistoricalMigrationsRemainUnmodified(t *testing.T) {
	TestHistoricalMigrationsRemainUnmodified(t)
}

func TestSecureOPTenantRLSOrdersFailClosedBeforePolicyReplacement(t *testing.T) {
	sql := readSQL(t, "000014_secure_op_tenant_rls.up.sql")
	requireSQLOrder(t, sql,
		"CREATE OR REPLACE FUNCTION set_tenant_context",
		"RAISE EXCEPTION 'SDPivot secure tenant RLS requires target tables",
		"DROP POLICY IF EXISTS wcc_tenant_isolation ON write_category_config",
		"CREATE POLICY sdpivot_secure_000014_write_category_config ON write_category_config",
		"CREATE POLICY sdpivot_secure_000014_document_chunks ON document_chunks",
	)
}
