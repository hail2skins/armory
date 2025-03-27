package models

import (
	"fmt"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
)

// GetEnforcer creates a Casbin enforcer from a CasbinDBAdapter
func GetEnforcer(adapter *CasbinDBAdapter) (*casbin.Enforcer, error) {
	// Load the model configuration
	m, err := model.NewModelFromString(`
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && (r.obj == p.obj || p.obj == "*") && (r.act == p.act || p.act == "*") || g(r.sub, "admin")
`)
	if err != nil {
		return nil, err
	}

	// Create enforcer using the model and adapter
	enforcer, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, err
	}

	// Load policies from the database
	if err := enforcer.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("failed to load policies from database: %w", err)
	}

	// Enable auto-saving for any policy changes
	enforcer.EnableAutoSave(true)

	return enforcer, nil
}

// GetAllRoles returns all roles defined in the policy
func GetAllRoles(enforcer *casbin.Enforcer) []string {
	policies, _ := enforcer.GetPolicy()

	// Create a map to deduplicate roles
	rolesMap := make(map[string]bool)

	// Add all roles from policy rules
	for _, policy := range policies {
		if len(policy) >= 1 {
			rolesMap[policy[0]] = true
		}
	}

	// Convert map to slice
	roles := make([]string, 0, len(rolesMap))
	for role := range rolesMap {
		roles = append(roles, role)
	}

	return roles
}

// GetPermissionsForRole returns all permissions for a role
func GetPermissionsForRole(enforcer *casbin.Enforcer, role string) []Permission {
	policies, _ := enforcer.GetFilteredPolicy(0, role)

	// Convert policies to permissions
	permissions := make([]Permission, 0, len(policies))
	for _, policy := range policies {
		if len(policy) >= 3 {
			perm := Permission{
				Role:     policy[0],
				Resource: policy[1],
				Action:   policy[2],
			}
			permissions = append(permissions, perm)
		}
	}

	return permissions
}

// GetUsersForRole returns all users assigned to a role
func GetUsersForRole(enforcer *casbin.Enforcer, role string) []string {
	users, _ := enforcer.GetUsersForRole(role)
	return users
}

// RoleExists checks if a role exists in the policy
func RoleExists(enforcer *casbin.Enforcer, role string) bool {
	policies, _ := enforcer.GetFilteredPolicy(0, role)
	return len(policies) > 0
}

// HasRole checks if a user has a role
func HasRole(enforcer *casbin.Enforcer, user, role string) bool {
	has, _ := enforcer.HasGroupingPolicy(user, role)
	return has
}

// AddPermissionForRole adds a permission for a role
func AddPermissionForRole(enforcer *casbin.Enforcer, role, resource, action string) error {
	_, err := enforcer.AddPolicy(role, resource, action)
	return err
}

// RemovePermissionForRole removes a permission from a role
func RemovePermissionForRole(enforcer *casbin.Enforcer, role, resource, action string) error {
	_, err := enforcer.RemovePolicy(role, resource, action)
	return err
}

// DeleteRoleWithEnforcer deletes a role and all its permissions using a Casbin enforcer
func DeleteRoleWithEnforcer(enforcer *casbin.Enforcer, role string) error {
	// Remove all policies for the role
	_, err := enforcer.RemoveFilteredPolicy(0, role)
	if err != nil {
		return err
	}

	// Remove all role assignments
	users := GetUsersForRole(enforcer, role)

	for _, user := range users {
		_, err = enforcer.RemoveGroupingPolicy(user, role)
		if err != nil {
			return err
		}
	}

	return nil
}

// AssignRoleToUser assigns a role to a user
func AssignRoleToUser(enforcer *casbin.Enforcer, user, role string) error {
	_, err := enforcer.AddGroupingPolicy(user, role)
	return err
}

// RemoveRoleFromUser removes a role from a user
func RemoveRoleFromUser(enforcer *casbin.Enforcer, user, role string) error {
	_, err := enforcer.RemoveGroupingPolicy(user, role)
	return err
}

// ClearPolicies removes all policies
func ClearPolicies(enforcer *casbin.Enforcer) error {
	enforcer.ClearPolicy()
	return nil
}

// ImportDefaultPolicies imports the default set of policies
func ImportDefaultPolicies(enforcer *casbin.Enforcer) error {
	// Clear existing policies
	enforcer.ClearPolicy()

	// Add default roles and permissions
	policies := [][]string{
		// Admin role has all permissions
		{"admin", "*", "*"},

		// Editor role has read, create, update permissions for content
		{"editor", "manufacturers", "read"},
		{"editor", "manufacturers", "create"},
		{"editor", "manufacturers", "update"},
		{"editor", "calibers", "read"},
		{"editor", "calibers", "create"},
		{"editor", "calibers", "update"},
		{"editor", "weapon_types", "read"},
		{"editor", "weapon_types", "create"},
		{"editor", "weapon_types", "update"},
		{"editor", "promotions", "read"},
		// Feature flags permissions for editors
		{"editor", "feature_flags", "read"},
		{"editor", "feature_flags", "update"},

		// Viewer role has read-only permissions
		{"viewer", "manufacturers", "read"},
		{"viewer", "calibers", "read"},
		{"viewer", "weapon_types", "read"},
		// Feature flags read-only permissions for viewers
		{"viewer", "feature_flags", "read"},
	}

	// Add the policies
	for _, policy := range policies {
		if len(policy) == 3 {
			_, err := enforcer.AddPolicy(policy[0], policy[1], policy[2])
			if err != nil {
				return err
			}
		}
	}

	return nil
}
