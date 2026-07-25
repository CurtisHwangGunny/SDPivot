package types

// AccessRole is the product-level role assigned to a user.
type AccessRole string

const (
	AccessRoleSuperAdmin      AccessRole = "super_admin"
	AccessRoleDepartmentAdmin AccessRole = "department_admin"
	AccessRoleKnowledgeEditor AccessRole = "knowledge_editor"
	AccessRoleKnowledgeViewer AccessRole = "knowledge_viewer"
)

// IsValid reports whether the role can be assigned through the user API.
func (r AccessRole) IsValid() bool {
	switch r {
	case AccessRoleSuperAdmin, AccessRoleDepartmentAdmin, AccessRoleKnowledgeEditor, AccessRoleKnowledgeViewer:
		return true
	default:
		return false
	}
}

// NormalizeAccessRole maps legacy JWT and tenant role names to product roles.
func NormalizeAccessRole(role string) AccessRole {
	switch role {
	case "ops_admin", "system_admin", string(AccessRoleSuperAdmin):
		return AccessRoleSuperAdmin
	case "owner", "admin", string(AccessRoleDepartmentAdmin):
		return AccessRoleDepartmentAdmin
	case "editor", "contributor", string(AccessRoleKnowledgeEditor):
		return AccessRoleKnowledgeEditor
	case "member", "viewer", "", string(AccessRoleKnowledgeViewer):
		return AccessRoleKnowledgeViewer
	default:
		return AccessRole(role)
	}
}
