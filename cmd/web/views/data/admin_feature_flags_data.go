package data

import (
	"github.com/hail2skins/armory/internal/models"
)

// FeatureFlagsViewData contains data for the feature flags index view
type FeatureFlagsViewData struct {
	// All feature flags in the system
	FeatureFlags []models.FeatureFlag

	// Map of feature flag ID to roles
	FlagRoles map[uint][]string
}

// FeatureFlagFormData contains data for the feature flag create/edit forms
type FeatureFlagFormData struct {
	// The feature flag being edited (nil for create)
	FeatureFlag *models.FeatureFlag

	// Available roles for assignment
	AvailableRoles []string

	// Roles already assigned to this feature flag (for edit form)
	AssignedRoles []string
}

// FeatureFlagRoleData contains data for the role assignment form
type FeatureFlagRoleData struct {
	// The feature flag ID
	FeatureFlagID uint

	// The feature flag name
	FeatureFlagName string

	// Available roles for assignment
	AvailableRoles []string

	// Currently assigned roles
	AssignedRoles []string
}
