package postgres

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"testing"
)

const legacyOpsHash = "$2a$10$L4fDCGy48S7wHDmzZqAAf.sql5NnA1.0uEwfbbVzvAsMSV0qyUGS2"

func readSQL(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return strings.ToLower(string(content))
}

func requireSQLFragments(t *testing.T, sql string, fragments ...string) {
	t.Helper()
	normalizedSQL := strings.Join(strings.Fields(sql), " ")
	for _, fragment := range fragments {
		normalizedFragment := strings.Join(strings.Fields(strings.ToLower(fragment)), " ")
		if !strings.Contains(normalizedSQL, normalizedFragment) {
			t.Errorf("migration missing %q", fragment)
		}
	}
}

func TestDisableLegacyOpsAdminUpIsStrictAndAuditable(t *testing.T) {
	sql := readSQL(t, "000013_disable_legacy_ops_admin.up.sql")
	requireSQLFragments(t, sql,
		"CREATE TABLE IF NOT EXISTS sdpivot_disable_legacy_ops_admin_000013_state",
		"user_id VARCHAR(36) PRIMARY KEY",
		"original_password_hash",
		"original_is_active",
		"original_must_change_password",
		"original_is_ops_admin",
		"original_is_system_admin",
		"disabled_password_hash",
		"WHERE email = 'admin@smartknora.com'",
		"AND password_hash = '"+legacyOpsHash+"'",
		"ON CONFLICT (user_id) DO NOTHING",
		"FROM sdpivot_disable_legacy_ops_admin_000013_state AS state",
		"target.id = state.user_id",
		"target.email = state.email",
		"target.email = 'admin@smartknora.com'",
		"target.password_hash = '"+legacyOpsHash+"'",
		"state.original_password_hash = '"+legacyOpsHash+"'",
		"state.disabled_password_hash = '!sdpivot-disabled-legacy-ops-admin:' || state.user_id",
		"state.restored_at IS NULL",
		"is_active = FALSE",
		"must_change_password = TRUE",
		"!sdpivot-disabled-legacy-ops-admin:",
	)
	if strings.Contains(sql, "where email = 'admin@smartknora.com' or") {
		t.Error("legacy account capture must require both exact email and exact password hash")
	}
}

func TestDisableLegacyOpsAdminDownIsGuardedAndRetainsEvidence(t *testing.T) {
	sql := readSQL(t, "000013_disable_legacy_ops_admin.down.sql")
	requireSQLFragments(t, sql,
		"to_regclass('public.sdpivot_disable_legacy_ops_admin_000013_state') IS NULL",
		"FROM sdpivot_disable_legacy_ops_admin_000013_state AS state",
		"target.id = state.user_id",
		"target.email = state.email",
		"target.password_hash = state.disabled_password_hash",
		"target.is_active = FALSE",
		"target.must_change_password = TRUE",
		"state.restored_at IS NULL",
		"SET restored_at = NOW()",
	)
	if strings.Contains(sql, "drop table") || strings.Contains(sql, "delete from sdpivot_disable_legacy_ops_admin_000013_state") {
		t.Error("down migration must retain state evidence")
	}
}

func TestHistoricalMigrationsRemainUnmodified(t *testing.T) {
	expected := map[string]string{
		"000001_smartknora_init.down.sql":            "b34ca003bb95db8ec2a27d30b1f3358aa86ebf0a09d14a376375d328f1f9bf19",
		"000001_smartknora_init.up.sql":              "70e60f1a0225326dc114b8c59a285b37fd6b44e50319a6719c2bf275e2ffcfcb",
		"000002_rls_policies.down.sql":               "09bb7021cb71b8b88fad5c4993db93f205d5be30fef813870afe56dc5e40eaed",
		"000002_rls_policies.up.sql":                 "ab99a3377050e3ed427a6f4f9becc50c33c098fb37da08a3ecc91413d7bf99c5",
		"000003_smartknora_extensions.down.sql":      "dcd77924c4e1444057375befa6b11ebce94c4efe42e138e40bc4dbd215355302",
		"000003_smartknora_extensions.up.sql":        "971de94a664b15607093438eaea3d54c052bbb47fec3e3c5b3f0730b639c567a",
		"000004_sprint2_sprint4.down.sql":            "584a70af8ac53594df3299777642ee2b7d4278e753e6aa6784d8effe8d37f7ce",
		"000004_sprint2_sprint4.up.sql":              "0eabe02dcb614260bccb23c9492726186a6b301f75cb4ecb66b16722669025e8",
		"000005_smartknora_rls_fixes.down.sql":       "ad40d4385464da1cbd328bb9e614c4cbac33853ba2eac5a36b347674b9261739",
		"000005_smartknora_rls_fixes.up.sql":         "fcad1fc21ff97705491b8ccb0888964a59ef891ddc4e86143cdae41b119129b4",
		"000006_smartknora_fixes.down.sql":           "f2154f3b1ab1c7a2afa58cdc593e79c33474d82f50c6a9fd37ea532cd73c32f1",
		"000006_smartknora_fixes.up.sql":             "6d8f05d816e3e48e0fa68f660514e327e96d022ef943982bb868e63dd733ca7d",
		"000007_ops_admin.up.sql":                    "c1555d088011e976da2bf1530a287041387ad6d3f1adb65ef82c6c2f0ea69613",
		"000008_smartknora_ops_fixes.up.sql":         "de50035230cbed2a37e47d136f8b2b60b055204d87a2baf4cfd15211566c291f",
		"000009_ops_modules.up.sql":                  "8b544e5129a08ac81f760f231b7942db022bdf5405e5a5894b48e55f26193025",
		"000010_smartknora_v42_alignment.down.sql":   "626f565120173c4d571a81d453f2824df42dddcb41d62642c5dd747e58b348d1",
		"000010_smartknora_v42_alignment.up.sql":     "51c0ac8a66a48bda848d0e253f2c3cd67493111ea14fd019f8d236e8db29fbff",
		"000011_sdpivot_brand_migration.down.sql":    "d1294864ab01cb052dffa1c39308433a6043001522b44bb765469be1c3796d41",
		"000011_sdpivot_brand_migration.up.sql":      "8bacb93351697a0e78d16f7a72c81469575dcfad12c510c538f5d1c8a22c3ef6",
		"000012_document_chunks_tenant_rls.down.sql": "abb1b2de46dbaefd985c29d28fb14abb68ca71f7029b182e2a86fe5f87d3c508",
		"000012_document_chunks_tenant_rls.up.sql":   "71f3b8ec2e8a80d58adc920c309470fbb6833fdca99da9f2fdf37ff76c7852f3",
	}
	for name, want := range expected {
		content, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read historical migration %s: %v", name, err)
		}
		got := fmt.Sprintf("%x", sha256.Sum256(content))
		if got != want {
			t.Errorf("historical migration %s changed: got %s want %s", name, got, want)
		}
	}
}
