package models

import (
	"context"
	"errors"
	"strings"

	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"gorm.io/gorm"
)

// CasbinRule represents a rule in Casbin RBAC policy
type CasbinRule struct {
	ID    uint   `gorm:"primaryKey;autoIncrement"`
	Ptype string `gorm:"size:100;uniqueIndex:unique_index"`
	V0    string `gorm:"size:100;uniqueIndex:unique_index"`
	V1    string `gorm:"size:100;uniqueIndex:unique_index"`
	V2    string `gorm:"size:100;uniqueIndex:unique_index"`
	V3    string `gorm:"size:100;uniqueIndex:unique_index"`
	V4    string `gorm:"size:100;uniqueIndex:unique_index"`
	V5    string `gorm:"size:100;uniqueIndex:unique_index"`
}

// TableName sets the table name for CasbinRule
func (c *CasbinRule) TableName() string {
	return "casbin_rule"
}

// CasbinDBAdapter is an adapter that implements persist.Adapter interface for Casbin
type CasbinDBAdapter struct {
	db *gorm.DB
}

// NewCasbinDBAdapter creates a new adapter for Casbin using GORM
func NewCasbinDBAdapter(db *gorm.DB) *CasbinDBAdapter {
	// Ensure the CasbinRule table exists
	if err := db.AutoMigrate(&CasbinRule{}); err != nil {
		panic("failed to migrate CasbinRule: " + err.Error())
	}

	return &CasbinDBAdapter{db: db}
}

// LoadPolicy loads all policy rules from the storage
func (a *CasbinDBAdapter) LoadPolicy(model model.Model) error {
	var rules []CasbinRule
	if err := a.db.Find(&rules).Error; err != nil {
		return err
	}

	for _, rule := range rules {
		loadPolicyLine(rule, model)
	}

	return nil
}

// SavePolicy saves all policy rules to the storage
func (a *CasbinDBAdapter) SavePolicy(model model.Model) error {
	// Clear existing policies
	if err := a.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&CasbinRule{}).Error; err != nil {
		return err
	}

	var lines []CasbinRule

	// Save policy lines
	for ptype, ast := range model["p"] {
		for _, policy := range ast.Policy {
			line := savePolicyLine(ptype, policy)
			lines = append(lines, line)
		}
	}

	// Save role lines (g sections in Casbin)
	for ptype, ast := range model["g"] {
		for _, policy := range ast.Policy {
			line := savePolicyLine(ptype, policy)
			lines = append(lines, line)
		}
	}

	if len(lines) > 0 {
		return a.db.Create(&lines).Error
	}

	return nil
}

// AddPolicy adds a policy rule to the storage
func (a *CasbinDBAdapter) AddPolicy(sec string, ptype string, rule []string) error {
	line := savePolicyLine(ptype, rule)
	return a.db.Create(&line).Error
}

// RemovePolicy removes a policy rule from the storage
func (a *CasbinDBAdapter) RemovePolicy(sec string, ptype string, rule []string) error {
	line := savePolicyLine(ptype, rule)

	db := a.db.Where("ptype = ?", line.Ptype)

	if line.V0 != "" {
		db = db.Where("v0 = ?", line.V0)
	}
	if line.V1 != "" {
		db = db.Where("v1 = ?", line.V1)
	}
	if line.V2 != "" {
		db = db.Where("v2 = ?", line.V2)
	}
	if line.V3 != "" {
		db = db.Where("v3 = ?", line.V3)
	}
	if line.V4 != "" {
		db = db.Where("v4 = ?", line.V4)
	}
	if line.V5 != "" {
		db = db.Where("v5 = ?", line.V5)
	}

	return db.Delete(&CasbinRule{}).Error
}

// RemoveFilteredPolicy removes policy rules that match the filter from the storage
func (a *CasbinDBAdapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	db := a.db.Where("ptype = ?", ptype)

	if fieldIndex <= 0 && 0 < fieldIndex+len(fieldValues) && len(fieldValues) > 0 {
		db = db.Where("v0 = ?", fieldValues[0-fieldIndex])
	}
	if fieldIndex <= 1 && 1 < fieldIndex+len(fieldValues) && len(fieldValues) > 1 {
		db = db.Where("v1 = ?", fieldValues[1-fieldIndex])
	}
	if fieldIndex <= 2 && 2 < fieldIndex+len(fieldValues) && len(fieldValues) > 2 {
		db = db.Where("v2 = ?", fieldValues[2-fieldIndex])
	}
	if fieldIndex <= 3 && 3 < fieldIndex+len(fieldValues) && len(fieldValues) > 3 {
		db = db.Where("v3 = ?", fieldValues[3-fieldIndex])
	}
	if fieldIndex <= 4 && 4 < fieldIndex+len(fieldValues) && len(fieldValues) > 4 {
		db = db.Where("v4 = ?", fieldValues[4-fieldIndex])
	}
	if fieldIndex <= 5 && 5 < fieldIndex+len(fieldValues) && len(fieldValues) > 5 {
		db = db.Where("v5 = ?", fieldValues[5-fieldIndex])
	}

	return db.Delete(&CasbinRule{}).Error
}

// UpdatePolicy updates a policy rule from the storage
func (a *CasbinDBAdapter) UpdatePolicy(sec string, ptype string, oldRule, newRule []string) error {
	// Start a transaction
	return a.db.Transaction(func(tx *gorm.DB) error {
		// Delete old rule
		if err := a.RemovePolicy(sec, ptype, oldRule); err != nil {
			return err
		}

		// Add new rule
		return a.AddPolicy(sec, ptype, newRule)
	})
}

// UpdateFilteredPolicies updates a policy rule from the storage
func (a *CasbinDBAdapter) UpdateFilteredPolicies(sec string, ptype string, newPolicies [][]string, fieldIndex int, fieldValues ...string) ([][]string, error) {
	// This is a more complex operation, we need to find existing policies and replace them
	return nil, errors.New("not implemented")
}

// loadPolicyLine loads a policy rule into the model
func loadPolicyLine(line CasbinRule, model model.Model) {
	lineText := line.Ptype
	if line.V0 != "" {
		lineText += ", " + line.V0
	}
	if line.V1 != "" {
		lineText += ", " + line.V1
	}
	if line.V2 != "" {
		lineText += ", " + line.V2
	}
	if line.V3 != "" {
		lineText += ", " + line.V3
	}
	if line.V4 != "" {
		lineText += ", " + line.V4
	}
	if line.V5 != "" {
		lineText += ", " + line.V5
	}

	persist.LoadPolicyLine(lineText, model)
}

// savePolicyLine converts a policy rule to a CasbinRule entity
func savePolicyLine(ptype string, rule []string) CasbinRule {
	line := CasbinRule{
		Ptype: ptype,
	}

	if len(rule) > 0 {
		line.V0 = rule[0]
	}
	if len(rule) > 1 {
		line.V1 = rule[1]
	}
	if len(rule) > 2 {
		line.V2 = rule[2]
	}
	if len(rule) > 3 {
		line.V3 = rule[3]
	}
	if len(rule) > 4 {
		line.V4 = rule[4]
	}
	if len(rule) > 5 {
		line.V5 = rule[5]
	}

	return line
}

// FindRolesByUser returns all roles assigned to the user
func FindRolesByUser(db *gorm.DB, user string) ([]string, error) {
	var rules []CasbinRule
	if err := db.Where("ptype = ? AND v0 = ?", "g", user).Find(&rules).Error; err != nil {
		return nil, err
	}

	roles := make([]string, 0, len(rules))
	for _, rule := range rules {
		roles = append(roles, rule.V1)
	}

	return roles, nil
}

// FindUsersInRole returns all users assigned to a specific role
func FindUsersInRole(db *gorm.DB, role string) ([]string, error) {
	var rules []CasbinRule
	if err := db.Where("ptype = ? AND v1 = ?", "g", role).Find(&rules).Error; err != nil {
		return nil, err
	}

	users := make([]string, 0, len(rules))
	for _, rule := range rules {
		users = append(users, rule.V0)
	}

	return users, nil
}

// FindAllRoles returns all roles defined in the system
func FindAllRoles(db *gorm.DB) ([]string, error) {
	var rules []CasbinRule
	if err := db.Where("ptype = ?", "p").Find(&rules).Error; err != nil {
		return nil, err
	}

	// Use a map to deduplicate roles
	rolesMap := make(map[string]bool)
	for _, rule := range rules {
		rolesMap[rule.V0] = true
	}

	roles := make([]string, 0, len(rolesMap))
	for role := range rolesMap {
		roles = append(roles, role)
	}

	return roles, nil
}

// FindPermissionsForRole returns all permissions assigned to a role
func FindPermissionsForRole(db *gorm.DB, role string) ([]map[string]string, error) {
	var rules []CasbinRule
	if err := db.Where("ptype = ? AND v0 = ?", "p", role).Find(&rules).Error; err != nil {
		return nil, err
	}

	permissions := make([]map[string]string, 0, len(rules))
	for _, rule := range rules {
		perm := map[string]string{
			"object": rule.V1,
			"action": rule.V2,
		}
		if rule.V3 != "" {
			perm["domain"] = rule.V3
		}
		permissions = append(permissions, perm)
	}

	return permissions, nil
}

// AddUserToRole assigns a user to a role
func AddUserToRole(db *gorm.DB, user, role string) error {
	// Check if the role exists
	var roleCount int64
	if err := db.Model(&CasbinRule{}).Where("ptype = ? AND v0 = ?", "p", role).Count(&roleCount).Error; err != nil {
		return err
	}

	if roleCount == 0 {
		return errors.New("role does not exist")
	}

	// Check if assignment already exists
	var count int64
	if err := db.Model(&CasbinRule{}).
		Where("ptype = ? AND v0 = ? AND v1 = ?", "g", user, role).
		Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return nil // Already assigned
	}

	// Create the assignment
	rule := CasbinRule{
		Ptype: "g",
		V0:    user,
		V1:    role,
	}

	return db.Create(&rule).Error
}

// RemoveUserFromRole removes a user from a role
func RemoveUserFromRole(db *gorm.DB, user, role string) error {
	return db.Where("ptype = ? AND v0 = ? AND v1 = ?", "g", user, role).
		Delete(&CasbinRule{}).Error
}

// CreateRole creates a new role with default permissions
func CreateRole(db *gorm.DB, role string, permissions []map[string]string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// Check if the role already exists
		var count int64
		if err := tx.Model(&CasbinRule{}).
			Where("ptype = ? AND v0 = ?", "p", role).
			Count(&count).Error; err != nil {
			return err
		}

		if count > 0 {
			return errors.New("role already exists")
		}

		// Create the role with permissions
		for _, perm := range permissions {
			rule := CasbinRule{
				Ptype: "p",
				V0:    role,
				V1:    perm["object"],
				V2:    perm["action"],
			}

			if domain, ok := perm["domain"]; ok {
				rule.V3 = domain
			}

			if err := tx.Create(&rule).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// DeleteRole removes a role and all associated user assignments
func DeleteRole(db *gorm.DB, role string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// Delete all policies for the role
		if err := tx.Where("ptype = ? AND v0 = ?", "p", role).
			Delete(&CasbinRule{}).Error; err != nil {
			return err
		}

		// Delete all user role assignments
		if err := tx.Where("ptype = ? AND v1 = ?", "g", role).
			Delete(&CasbinRule{}).Error; err != nil {
			return err
		}

		return nil
	})
}

// ImportFromCSV imports policies from a CSV string
func ImportFromCSV(db *gorm.DB, csv string) error {
	lines := strings.Split(csv, "\n")

	return db.Transaction(func(tx *gorm.DB) error {
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			parts := strings.Split(line, ",")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}

			if len(parts) < 2 {
				continue
			}

			var rule CasbinRule
			rule.Ptype = parts[0]

			if len(parts) > 1 {
				rule.V0 = parts[1]
			}
			if len(parts) > 2 {
				rule.V1 = parts[2]
			}
			if len(parts) > 3 {
				rule.V2 = parts[3]
			}
			if len(parts) > 4 {
				rule.V3 = parts[4]
			}
			if len(parts) > 5 {
				rule.V4 = parts[5]
			}
			if len(parts) > 6 {
				rule.V5 = parts[6]
			}

			if err := tx.Create(&rule).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// ExportToCSV exports all policies to a CSV string
func ExportToCSV(db *gorm.DB) (string, error) {
	var rules []CasbinRule
	if err := db.Find(&rules).Error; err != nil {
		return "", err
	}

	var sb strings.Builder

	for _, rule := range rules {
		sb.WriteString(rule.Ptype)

		if rule.V0 != "" {
			sb.WriteString(", " + rule.V0)
		}
		if rule.V1 != "" {
			sb.WriteString(", " + rule.V1)
		}
		if rule.V2 != "" {
			sb.WriteString(", " + rule.V2)
		}
		if rule.V3 != "" {
			sb.WriteString(", " + rule.V3)
		}
		if rule.V4 != "" {
			sb.WriteString(", " + rule.V4)
		}
		if rule.V5 != "" {
			sb.WriteString(", " + rule.V5)
		}

		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// IsUserInRole checks if a user has a specific role
func IsUserInRole(db *gorm.DB, user, role string) (bool, error) {
	var count int64
	err := db.Model(&CasbinRule{}).
		Where("ptype = ? AND v0 = ? AND v1 = ?", "g", user, role).
		Count(&count).Error

	return count > 0, err
}

// GetUserRoles gets all roles for a user
func GetUserRoles(db *gorm.DB, user string) ([]string, error) {
	var rules []CasbinRule
	if err := db.Where("ptype = ? AND v0 = ?", "g", user).Find(&rules).Error; err != nil {
		return nil, err
	}

	roles := make([]string, 0, len(rules))
	for _, rule := range rules {
		roles = append(roles, rule.V1)
	}

	return roles, nil
}

// CasbinRBACService provides RBAC services using Casbin
type CasbinRBACService struct {
	db      *gorm.DB
	adapter *CasbinDBAdapter
}

// NewCasbinRBACService creates a new CasbinRBACService
func NewCasbinRBACService(db *gorm.DB) (*CasbinRBACService, error) {
	adapter := NewCasbinDBAdapter(db)

	// This function just creates the service, not the enforcer
	return &CasbinRBACService{
		db:      db,
		adapter: adapter,
	}, nil
}

// ImportDefaultPolicies imports the default policies from the CSV file
func (s *CasbinRBACService) ImportDefaultPolicies(ctx context.Context, csvContent string) error {
	return ImportFromCSV(s.db, csvContent)
}
