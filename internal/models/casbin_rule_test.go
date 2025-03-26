package models

import (
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/stretchr/testify/assert"
)

func TestCasbinRuleModel(t *testing.T) {
	// Get test database
	db := GetTestDB()

	// Auto migrate the CasbinRule table - this should be done by the adapter
	err := db.AutoMigrate(&CasbinRule{})
	assert.NoError(t, err)

	// Create a DB adapter manually for testing
	adapter := NewCasbinDBAdapter(db)

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
	assert.NoError(t, err)

	// Create enforcer using the model and adapter
	enforcer, err := casbin.NewEnforcer(m, adapter)
	assert.NoError(t, err)

	// Test policy management
	t.Run("Policy Management", func(t *testing.T) {
		// Add a policy
		added, err := enforcer.AddPolicy("admin", "resource", "read")
		assert.NoError(t, err)
		assert.True(t, added)

		// Check if policy exists
		hasPolicy, err := enforcer.HasPolicy("admin", "resource", "read")
		assert.NoError(t, err)
		assert.True(t, hasPolicy)

		// Add a role for a user
		added, err = enforcer.AddGroupingPolicy("user@example.com", "admin")
		assert.NoError(t, err)
		assert.True(t, added)

		// Check if user has role
		hasRole, err := enforcer.HasGroupingPolicy("user@example.com", "admin")
		assert.NoError(t, err)
		assert.True(t, hasRole)

		// Check if user has permission via role
		allowed, err := enforcer.Enforce("user@example.com", "resource", "read")
		assert.NoError(t, err)
		assert.True(t, allowed)

		// Remove policy
		removed, err := enforcer.RemovePolicy("admin", "resource", "read")
		assert.NoError(t, err)
		assert.True(t, removed)

		// Check policy is gone
		hasPolicy, err = enforcer.HasPolicy("admin", "resource", "read")
		assert.NoError(t, err)
		assert.False(t, hasPolicy)

		// Remove role
		removed, err = enforcer.RemoveGroupingPolicy("user@example.com", "admin")
		assert.NoError(t, err)
		assert.True(t, removed)

		// Check role is gone
		hasRole, err = enforcer.HasGroupingPolicy("user@example.com", "admin")
		assert.NoError(t, err)
		assert.False(t, hasRole)
	})

	t.Run("Import Existing Policies from CSV", func(t *testing.T) {
		// Clear existing policies first
		enforcer.ClearPolicy()

		// Import policies from the current CSV
		err := enforcer.LoadPolicy()
		assert.NoError(t, err)

		// Add policies from the current CSV format
		_, err = enforcer.AddPolicy("admin", "*", "*")
		assert.NoError(t, err)
		_, err = enforcer.AddPolicy("admin", "manufacturers", "*")
		assert.NoError(t, err)
		_, err = enforcer.AddPolicy("editor", "manufacturers", "read")
		assert.NoError(t, err)
		_, err = enforcer.AddPolicy("viewer", "manufacturers", "read")
		assert.NoError(t, err)
		_, err = enforcer.AddGroupingPolicy("test@example.com", "admin")
		assert.NoError(t, err)

		// Verify policies were added
		hasPolicy, err := enforcer.HasPolicy("admin", "*", "*")
		assert.NoError(t, err)
		assert.True(t, hasPolicy)

		hasPolicy, err = enforcer.HasPolicy("admin", "manufacturers", "*")
		assert.NoError(t, err)
		assert.True(t, hasPolicy)

		hasPolicy, err = enforcer.HasPolicy("editor", "manufacturers", "read")
		assert.NoError(t, err)
		assert.True(t, hasPolicy)

		hasPolicy, err = enforcer.HasPolicy("viewer", "manufacturers", "read")
		assert.NoError(t, err)
		assert.True(t, hasPolicy)

		hasRole, err := enforcer.HasGroupingPolicy("test@example.com", "admin")
		assert.NoError(t, err)
		assert.True(t, hasRole)

		// Save policies to DB
		err = enforcer.SavePolicy()
		assert.NoError(t, err)
	})

	t.Run("Role Assignment and Validation", func(t *testing.T) {
		// Clear policies first from the enforcer
		enforcer.ClearPolicy()

		// Also clear the database directly to avoid unique constraint violations
		err = db.Exec("DELETE FROM casbin_rule").Error
		assert.NoError(t, err)

		// Create roles and assign permissions
		_, err := enforcer.AddPolicy("admin", "*", "*")
		assert.NoError(t, err)
		_, err = enforcer.AddPolicy("editor", "resource", "write")
		assert.NoError(t, err)
		_, err = enforcer.AddPolicy("viewer", "resource", "read")
		assert.NoError(t, err)

		// Assign users to roles
		_, err = enforcer.AddGroupingPolicy("admin@example.com", "admin")
		assert.NoError(t, err)
		_, err = enforcer.AddGroupingPolicy("editor@example.com", "editor")
		assert.NoError(t, err)
		_, err = enforcer.AddGroupingPolicy("viewer@example.com", "viewer")
		assert.NoError(t, err)

		// Test admin permissions
		allowed, err := enforcer.Enforce("admin@example.com", "anything", "anything")
		assert.NoError(t, err)
		assert.True(t, allowed)

		// Test editor permissions
		allowed, err = enforcer.Enforce("editor@example.com", "resource", "write")
		assert.NoError(t, err)
		assert.True(t, allowed)

		// Editor shouldn't have admin permissions
		allowed, err = enforcer.Enforce("editor@example.com", "anything", "anything")
		assert.NoError(t, err)
		assert.False(t, allowed)

		// Test viewer permissions
		allowed, err = enforcer.Enforce("viewer@example.com", "resource", "read")
		assert.NoError(t, err)
		assert.True(t, allowed)

		// Viewer shouldn't have write permissions
		allowed, err = enforcer.Enforce("viewer@example.com", "resource", "write")
		assert.NoError(t, err)
		assert.False(t, allowed)
	})
}

func TestRBACMiddleware(t *testing.T) {
	// Get test database
	db := GetTestDB()

	// Auto migrate the CasbinRule table
	err := db.AutoMigrate(&CasbinRule{})
	assert.NoError(t, err)

	// Create a new RBAC service that uses our adapter
	adapter := NewCasbinDBAdapter(db)

	// Cleanup any previous test data
	err = db.Exec("DELETE FROM casbin_rule").Error
	assert.NoError(t, err)

	// Create enforcer
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
	assert.NoError(t, err)

	enforcer, err := casbin.NewEnforcer(m, adapter)
	assert.NoError(t, err)

	// Add some policies
	_, err = enforcer.AddPolicy("admin", "*", "*")
	assert.NoError(t, err)
	_, err = enforcer.AddPolicy("editor", "manufacturers", "read")
	assert.NoError(t, err)
	_, err = enforcer.AddGroupingPolicy("test@example.com", "admin")
	assert.NoError(t, err)
	_, err = enforcer.AddGroupingPolicy("editor@example.com", "editor")
	assert.NoError(t, err)

	// Verify the rules are in the database
	var rules []CasbinRule
	err = db.Find(&rules).Error
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(rules), 4) // Should have at least 4 rules

	// Check specific rules
	var adminRule CasbinRule
	err = db.Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", "p", "admin", "*", "*").First(&adminRule).Error
	assert.NoError(t, err)

	var groupingRule CasbinRule
	err = db.Where("ptype = ? AND v0 = ? AND v1 = ?", "g", "test@example.com", "admin").First(&groupingRule).Error
	assert.NoError(t, err)
}
