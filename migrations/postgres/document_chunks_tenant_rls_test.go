package postgres

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func readMigration(t *testing.T, name string) string {
	t.Helper()
	content, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return strings.ToLower(string(content))
}

func TestDocumentChunksTenantRLSUp(t *testing.T) {
	sql := readMigration(t, "000012_document_chunks_tenant_rls.up.sql")
	required := []string{
		"alter table document_chunks enable row level security",
		"alter table document_chunks force row level security",
		"create policy document_chunks_tenant_isolation_000012 on document_chunks",
		"for all",
		"using (",
		"with check (",
		"is_ops_admin_context()",
		"from documents d",
		"d.id = document_chunks.document_id",
		"d.tenant_id = document_chunks.tenant_id",
		"d.tenant_id = get_current_tenant_id()",
	}
	for _, fragment := range required {
		if !strings.Contains(sql, fragment) {
			t.Errorf("up migration missing %q", fragment)
		}
	}

	if matches := regexp.MustCompile(`(?s)using\s*\(.*?d\.tenant_id\s*=\s*get_current_tenant_id\(\).*?\)\s*with check\s*\(.*?d\.tenant_id\s*=\s*get_current_tenant_id\(\)`).FindString(sql); matches == "" {
		t.Error("both USING and WITH CHECK must constrain chunks through documents to the current tenant")
	}
}

func TestDocumentChunksTenantRLSDownRestoresPriorState(t *testing.T) {
	sql := readMigration(t, "000012_document_chunks_tenant_rls.down.sql")
	required := []string{
		"drop policy if exists document_chunks_tenant_isolation_000012 on document_chunks",
		"rls_was_enabled",
		"force_was_enabled",
		"alter table document_chunks no force row level security",
		"alter table document_chunks disable row level security",
		"drop table if exists sdpivot_rls_migration_000012_state",
	}
	for _, fragment := range required {
		if !strings.Contains(sql, fragment) {
			t.Errorf("down migration missing %q", fragment)
		}
	}
}
