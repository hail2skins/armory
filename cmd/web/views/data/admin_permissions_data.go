package data

import (
	"github.com/hail2skins/armory/internal/database"
)

// AdminPermissionsData holds data for the admin permissions page
type AdminPermissionsData struct {
	Roles []map[string]interface{} // List of roles with their permissions and users
	Users []database.User          // All users for potential role assignment
}

// AdminNewRoleData holds data for the new role form
type AdminNewRoleData struct {
	Resources []string // Available resources for permissions
	Actions   []string // Available actions for permissions
}

// AdminEditRoleData holds data for the edit role form
type AdminEditRoleData struct {
	Name              string              // Role name
	Resources         []string            // Available resources for permissions
	Actions           []string            // Available actions for permissions
	Permissions       []map[string]string // Current permissions for the role
	SelectedResources map[string]bool     // Map of selected resources
	SelectedActions   map[string]bool     // Map of selected actions
}

// Permission represents a permission rule entry
type Permission struct {
	Role     string
	Resource string
	Action   string
}

// UserRole represents a user's role assignment
type UserRole struct {
	User string
	Role string
}

// PermissionsViewData contains data for the admin permissions index view
type PermissionsViewData struct {
	// Role names available in the system
	Roles []string

	// All permissions in the system
	Permissions []Permission

	// Map of role name to permissions
	RolePermissions map[string][]Permission

	// Map of role name to users
	RoleUsers map[string][]string

	// All user role assignments
	UserRoles []UserRole
}

// CreateRoleViewData contains data for the role creation view
type CreateRoleViewData struct {
	// Available resources
	Resources []string

	// Available actions
	Actions []string
}

// EditRoleViewData contains data for the role editing view
type EditRoleViewData struct {
	// The role being edited
	Role string

	// Available resources
	Resources []string

	// Available actions
	Actions []string

	// Map of "resource:action" -> bool indicating if the permission is selected
	SelectedPermissions map[string]bool
}

// AssignRoleViewData contains data for the assign role view
type AssignRoleViewData struct {
	// Available roles
	Roles []string

	// Available users for role assignment
	Users []database.User
}
