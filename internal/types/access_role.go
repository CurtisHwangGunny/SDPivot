package types

// AccessRole is the product-level authority used by SDPivot RBAC.
// TenantRole remains available for the official multi-workspace/share model,
// but must not be used as the product permission source when this value is set.
type AccessRole string

const (
	AccessRoleSuperAdmin      AccessRole = "super_admin"
	AccessRoleOpsAdmin        AccessRole = "ops_admin"
	AccessRoleDepartmentAdmin AccessRole = "department_admin"
	AccessRoleKnowledgeEditor AccessRole = "knowledge_editor"
	AccessRoleKnowledgeViewer AccessRole = "knowledge_viewer"
	AccessRoleUser            AccessRole = "user"
	AccessRoleViewer          AccessRole = "viewer"
)

func (r AccessRole) IsValid() bool {
	switch r {
	case AccessRoleSuperAdmin, AccessRoleOpsAdmin, AccessRoleDepartmentAdmin,
		AccessRoleKnowledgeEditor, AccessRoleKnowledgeViewer, AccessRoleUser,
		AccessRoleViewer:
		return true
	default:
		return false
	}
}

// NormalizeAccessRole is deliberately the only compatibility boundary for
// product authorization. Historical tenant/system role strings are accepted
// here, never at individual permission call sites.
func NormalizeAccessRole(role string) AccessRole {
	switch role {
	case "ops_admin", "system_admin", "super_admin":
		return AccessRoleSuperAdmin
	case "owner", "admin", "department_admin":
		return AccessRoleDepartmentAdmin
	case "editor", "contributor", "knowledge_editor":
		return AccessRoleKnowledgeEditor
	case "member", "user":
		return AccessRoleUser
	case "viewer", "knowledge_viewer", "":
		return AccessRoleKnowledgeViewer
	default:
		return AccessRole(role)
	}
}

func (r AccessRole) Level() int {
	switch NormalizeAccessRole(string(r)) {
	case AccessRoleSuperAdmin:
		return 50
	case AccessRoleOpsAdmin:
		return 45
	case AccessRoleDepartmentAdmin:
		return 30
	case AccessRoleKnowledgeEditor:
		return 20
	case AccessRoleKnowledgeViewer, AccessRoleViewer, AccessRoleUser:
		return 10
	default:
		return 0
	}
}

func (r AccessRole) HasPermission(required AccessRole) bool {
	return r.Level() >= NormalizeAccessRole(string(required)).Level()
}
