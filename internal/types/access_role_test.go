package types

import "testing"

func TestNormalizeAccessRole(t *testing.T) {
	tests := map[string]AccessRole{"ops_admin": AccessRoleSuperAdmin, "system_admin": AccessRoleSuperAdmin, "owner": AccessRoleDepartmentAdmin, "admin": AccessRoleDepartmentAdmin, "contributor": AccessRoleKnowledgeEditor, "viewer": AccessRoleKnowledgeViewer, "user": AccessRoleUser}
	for input, want := range tests {
		if got := NormalizeAccessRole(input); got != want {
			t.Fatalf("NormalizeAccessRole(%q) = %q, want %q", input, got, want)
		}
	}
}
