package postgres

import (
	"os"
	"strings"
	"testing"
)

func TestOPAdminAuthRBACMigrationUsesCanonicalTenantWithoutBypass(t *testing.T) {
	data, err := os.ReadFile("000017_op_admin_auth_rbac.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(data))
	for _, required := range []string{
		"values (1, 'sdpivot'",
		"add column if not exists access_role",
		"set tenant_id = 1",
		"can_access_all_tenants = false",
		"access_role = 'super_admin'",
		"insert into tenant_members",
		"'owner', 'active'",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"can_access_all_tenants = true",
		"app.is_ops_admin', 'true",
		"set_tenant_context(",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("migration restores forbidden cross-tenant bypass %q", forbidden)
		}
	}
}
