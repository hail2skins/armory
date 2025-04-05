package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// FeatureFlag represents a feature flag in the system
type FeatureFlag struct {
	ID           uint      `gorm:"primaryKey;autoIncrement"`
	Name         string    `gorm:"size:255;uniqueIndex;not null"`
	Enabled      bool      `gorm:"default:false"`
	PublicAccess bool      `gorm:"default:false"`
	Description  string    `gorm:"type:text"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
	// Allow optional roles to be associated with this feature flag
	Roles []FeatureFlagRole `gorm:"foreignKey:FeatureFlagID"`
}

// FeatureFlagRole defines a role that has access to a specific feature flag
type FeatureFlagRole struct {
	ID            uint   `gorm:"primaryKey;autoIncrement"`
	FeatureFlagID uint   `gorm:"index;not null"`
	Role          string `gorm:"size:100;not null"`
}

// TableName sets the table name for FeatureFlag
func (f *FeatureFlag) TableName() string {
	return "feature_flags"
}

// TableName sets the table name for FeatureFlagRole
func (r *FeatureFlagRole) TableName() string {
	return "feature_flag_roles"
}

// FeatureFlagService provides operations on feature flags
type FeatureFlagService struct {
	db *gorm.DB
}

// NewFeatureFlagService creates a new FeatureFlagService
func NewFeatureFlagService(db *gorm.DB) *FeatureFlagService {
	return &FeatureFlagService{
		db: db,
	}
}

// FindAllFeatureFlags returns all feature flags
func (s *FeatureFlagService) FindAllFeatureFlags() ([]FeatureFlag, error) {
	var flags []FeatureFlag
	err := s.db.Preload("Roles").Find(&flags).Error
	return flags, err
}

// FindFeatureFlagByID returns a feature flag by ID
func (s *FeatureFlagService) FindFeatureFlagByID(id uint) (*FeatureFlag, error) {
	var flag FeatureFlag
	err := s.db.Preload("Roles").First(&flag, id).Error
	if err != nil {
		return nil, err
	}
	return &flag, nil
}

// FindFeatureFlagByName returns a feature flag by name
func (s *FeatureFlagService) FindFeatureFlagByName(name string) (*FeatureFlag, error) {
	var flag FeatureFlag
	err := s.db.Preload("Roles").Where("name = ?", name).First(&flag).Error
	if err != nil {
		return nil, err
	}
	return &flag, nil
}

// CreateFeatureFlag creates a new feature flag
func (s *FeatureFlagService) CreateFeatureFlag(flag *FeatureFlag) error {
	return s.db.Create(flag).Error
}

// UpdateFeatureFlag updates a feature flag
func (s *FeatureFlagService) UpdateFeatureFlag(flag *FeatureFlag) error {
	return s.db.Save(flag).Error
}

// DeleteFeatureFlag deletes a feature flag
func (s *FeatureFlagService) DeleteFeatureFlag(id uint) error {
	// Start a transaction
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Delete associated roles first
		if err := tx.Where("feature_flag_id = ?", id).Delete(&FeatureFlagRole{}).Error; err != nil {
			return err
		}
		// Then delete the feature flag
		return tx.Delete(&FeatureFlag{}, id).Error
	})
}

// AddRoleToFeatureFlag adds a role to a feature flag
func (s *FeatureFlagService) AddRoleToFeatureFlag(flagID uint, role string) error {
	// Check if role exists in Casbin rules
	var roleExists bool
	err := s.db.Model(&CasbinRule{}).
		Where("ptype = ? AND v0 = ?", "p", role).
		Or("ptype = ? AND v1 = ?", "g", role).
		Select("count(*) > 0").
		Find(&roleExists).Error

	if err != nil {
		return err
	}

	if !roleExists {
		return errors.New("role does not exist")
	}

	// Check if the association already exists
	var count int64
	err = s.db.Model(&FeatureFlagRole{}).
		Where("feature_flag_id = ? AND role = ?", flagID, role).
		Count(&count).Error

	if err != nil {
		return err
	}

	if count > 0 {
		return nil // Already exists
	}

	// Create the association
	return s.db.Create(&FeatureFlagRole{
		FeatureFlagID: flagID,
		Role:          role,
	}).Error
}

// RemoveRoleFromFeatureFlag removes a role from a feature flag
func (s *FeatureFlagService) RemoveRoleFromFeatureFlag(flagID uint, role string) error {
	return s.db.Where("feature_flag_id = ? AND role = ?", flagID, role).
		Delete(&FeatureFlagRole{}).Error
}

// IsFeatureEnabled checks if a feature is enabled
func (s *FeatureFlagService) IsFeatureEnabled(featureName string) (bool, error) {
	var flag FeatureFlag
	err := s.db.Where("name = ?", featureName).First(&flag).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil // If feature flag doesn't exist, it's not enabled
		}
		return false, err
	}
	return flag.Enabled, nil
}

// CanAccessFeature checks if a user can access a feature
func CanAccessFeature(db *gorm.DB, username, featureName string) (bool, error) {
	// First, check if the feature is enabled at all
	var flag FeatureFlag
	err := db.Where("name = ?", featureName).First(&flag).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil // Feature doesn't exist, so no access
		}
		return false, err
	}

	// If the feature isn't enabled, no one can access it
	if !flag.Enabled {
		return false, nil
	}

	// If public access is enabled, everyone can access it
	if flag.PublicAccess {
		return true, nil
	}

	// Check if the feature has any role restrictions
	var roleCount int64
	err = db.Model(&FeatureFlagRole{}).
		Where("feature_flag_id = ?", flag.ID).
		Count(&roleCount).Error
	if err != nil {
		return false, err
	}

	// If there are no role restrictions, anyone can access it
	if roleCount == 0 {
		return true, nil
	}

	// Get all roles for the user
	userRoles, err := GetUserRoles(db, username)
	if err != nil {
		return false, err
	}

	// If the user has no roles, they can't access role-restricted features
	if len(userRoles) == 0 {
		return false, nil
	}

	// Check if any of the user's roles have access to this feature
	for _, role := range userRoles {
		var count int64
		err = db.Model(&FeatureFlagRole{}).
			Where("feature_flag_id = ? AND role = ?", flag.ID, role).
			Count(&count).Error
		if err != nil {
			return false, err
		}
		if count > 0 {
			return true, nil // User has a role with access to this feature
		}
	}

	// None of the user's roles have access to this feature
	return false, nil
}
