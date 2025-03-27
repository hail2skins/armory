package models

import (
	"fmt"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
)

// Permission represents a permission rule
type Permission struct {
	Role     string
	Resource string
	Action   string
}

// RBACService provides role-based access control functionality
type RBACService struct {
	enforcer *casbin.Enforcer
}

// NewRBACService creates a new RBAC service
func NewRBACService(adapter *CasbinDBAdapter) (*RBACService, error) {
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

	return &RBACService{
		enforcer: enforcer,
	}, nil
}

// GetAllRoles returns all roles in the system
func (s *RBACService) GetAllRoles() ([]string, error) {
	// Get all policies
	policies, err := s.enforcer.GetPolicy()
	if err != nil {
		return nil, err
	}

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

	return roles, nil
}

// GetPermissionsForRole returns all permissions for a role
func (s *RBACService) GetPermissionsForRole(role string) ([]Permission, error) {
	// Get all policies for the role
	policies, err := s.enforcer.GetFilteredPolicy(0, role)
	if err != nil {
		return nil, err
	}

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

	return permissions, nil
}

// GetUsersForRole returns all users assigned to a role
func (s *RBACService) GetUsersForRole(role string) ([]string, error) {
	// Get all users for the role
	users, err := s.enforcer.GetUsersForRole(role)
	if err != nil {
		return nil, err
	}

	return users, nil
}

// RoleExists checks if a role exists
func (s *RBACService) RoleExists(role string) (bool, error) {
	// Get all policies for the role
	policies, err := s.enforcer.GetFilteredPolicy(0, role)
	if err != nil {
		return false, err
	}

	// If there are any policies, the role exists
	return len(policies) > 0, nil
}

// AddPermissionForRole adds a permission for a role
func (s *RBACService) AddPermissionForRole(role, resource, action string) error {
	// Add the policy
	_, err := s.enforcer.AddPolicy(role, resource, action)
	return err
}

// RemovePermissionForRole removes a permission from a role
func (s *RBACService) RemovePermissionForRole(role, resource, action string) error {
	// Remove the policy
	_, err := s.enforcer.RemovePolicy(role, resource, action)
	return err
}

// DeleteRole deletes a role and all its permissions
func (s *RBACService) DeleteRole(role string) error {
	// Remove all policies for the role
	_, err := s.enforcer.RemoveFilteredPolicy(0, role)
	if err != nil {
		return err
	}

	// Remove all role assignments
	users, err := s.GetUsersForRole(role)
	if err != nil {
		return err
	}

	for _, user := range users {
		_, err = s.enforcer.RemoveGroupingPolicy(user, role)
		if err != nil {
			return err
		}
	}

	return nil
}

// AssignRoleToUser assigns a role to a user
func (s *RBACService) AssignRoleToUser(user, role string) error {
	// Add the role assignment
	_, err := s.enforcer.AddGroupingPolicy(user, role)
	return err
}

// RemoveRoleFromUser removes a role from a user
func (s *RBACService) RemoveRoleFromUser(user, role string) error {
	// Remove the role assignment
	_, err := s.enforcer.RemoveGroupingPolicy(user, role)
	return err
}

// HasRole checks if a user has a role
func (s *RBACService) HasRole(user, role string) (bool, error) {
	// Check if the user has the role
	return s.enforcer.HasGroupingPolicy(user, role)
}

// ClearPolicies removes all policies
func (s *RBACService) ClearPolicies() error {
	// Clear all policies
	s.enforcer.ClearPolicy()

	return nil
}

// ImportDefaultPolicies imports the default set of policies
func (s *RBACService) ImportDefaultPolicies() error {
	// Clear existing policies
	s.enforcer.ClearPolicy()

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

		// Viewer role has read-only permissions
		{"viewer", "manufacturers", "read"},
		{"viewer", "calibers", "read"},
		{"viewer", "weapon_types", "read"},
	}

	// Add the policies
	for _, policy := range policies {
		if len(policy) == 3 {
			_, err := s.enforcer.AddPolicy(policy[0], policy[1], policy[2])
			if err != nil {
				return fmt.Errorf("failed to add policy %v: %w", policy, err)
			}
		}
	}

	return nil
}
