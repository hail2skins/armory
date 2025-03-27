package models_test

import (
	"errors"
	"testing"

	"github.com/hail2skins/armory/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFeatureFlagModel(t *testing.T) {
	// Set up in-memory database for testing
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	// Auto-migrate the schema
	err = db.AutoMigrate(&models.FeatureFlag{})
	assert.NoError(t, err)

	// Test creating a feature flag
	t.Run("CreateFeatureFlag", func(t *testing.T) {
		flag := models.FeatureFlag{
			Name:        "test_feature",
			Enabled:     false,
			Description: "Test feature flag",
		}

		// Save to database
		err := db.Create(&flag).Error
		assert.NoError(t, err)
		assert.NotZero(t, flag.ID, "Feature flag should have an ID after creation")
	})

	// Test retrieving a feature flag
	t.Run("GetFeatureFlag", func(t *testing.T) {
		// Create test data
		flag := models.FeatureFlag{
			Name:        "get_test_feature",
			Enabled:     true,
			Description: "Feature for retrieval test",
		}
		db.Create(&flag)

		// Retrieve the feature flag
		var retrievedFlag models.FeatureFlag
		err := db.Where("name = ?", "get_test_feature").First(&retrievedFlag).Error
		assert.NoError(t, err)
		assert.Equal(t, flag.Name, retrievedFlag.Name)
		assert.Equal(t, flag.Enabled, retrievedFlag.Enabled)
		assert.Equal(t, flag.Description, retrievedFlag.Description)
	})

	// Test updating a feature flag
	t.Run("UpdateFeatureFlag", func(t *testing.T) {
		// Create test data
		flag := models.FeatureFlag{
			Name:        "update_test_feature",
			Enabled:     false,
			Description: "Feature for update test",
		}
		db.Create(&flag)

		// Update the feature flag
		flag.Enabled = true
		flag.Description = "Updated description"
		err := db.Save(&flag).Error
		assert.NoError(t, err)

		// Verify the update
		var updatedFlag models.FeatureFlag
		db.Where("name = ?", "update_test_feature").First(&updatedFlag)
		assert.True(t, updatedFlag.Enabled)
		assert.Equal(t, "Updated description", updatedFlag.Description)
	})

	// Test deleting a feature flag
	t.Run("DeleteFeatureFlag", func(t *testing.T) {
		// Create test data
		flag := models.FeatureFlag{
			Name:        "delete_test_feature",
			Enabled:     false,
			Description: "Feature for deletion test",
		}
		db.Create(&flag)

		// Delete the feature flag
		err := db.Delete(&flag).Error
		assert.NoError(t, err)

		// Verify the deletion
		var deletedFlag models.FeatureFlag
		result := db.Where("name = ?", "delete_test_feature").First(&deletedFlag)
		assert.Error(t, result.Error)
		assert.True(t, errors.Is(result.Error, gorm.ErrRecordNotFound))
	})
}

func TestFeatureFlagWithRoles(t *testing.T) {
	// Set up in-memory database for testing
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	// Auto-migrate the schema
	err = db.AutoMigrate(&models.FeatureFlag{}, &models.FeatureFlagRole{}, &models.CasbinRule{})
	assert.NoError(t, err)

	// Test associating feature flags with roles
	t.Run("AssociateFeatureFlagWithRole", func(t *testing.T) {
		// Create a feature flag
		flag := models.FeatureFlag{
			Name:        "premium_feature",
			Enabled:     true,
			Description: "A premium feature",
		}
		db.Create(&flag)

		// Create a role association
		roleAssociation := models.FeatureFlagRole{
			FeatureFlagID: flag.ID,
			Role:          "premium_user",
		}

		err := db.Create(&roleAssociation).Error
		assert.NoError(t, err)

		// Verify the association
		var associations []models.FeatureFlagRole
		db.Where("feature_flag_id = ?", flag.ID).Find(&associations)
		assert.Len(t, associations, 1)
		assert.Equal(t, "premium_user", associations[0].Role)
	})

	// Test checking if a role has access to a feature
	t.Run("CheckRoleAccessToFeature", func(t *testing.T) {
		// Create a feature flag
		flag := models.FeatureFlag{
			Name:        "admin_feature",
			Enabled:     true,
			Description: "An admin-only feature",
		}
		db.Create(&flag)

		// Associate with admin role
		adminAssociation := models.FeatureFlagRole{
			FeatureFlagID: flag.ID,
			Role:          "admin",
		}
		db.Create(&adminAssociation)

		// Create a user with admin role
		adminRule := models.CasbinRule{
			Ptype: "g",
			V0:    "test_admin_user",
			V1:    "admin",
		}
		db.Create(&adminRule)

		// Create a regular user
		regularUserRule := models.CasbinRule{
			Ptype: "g",
			V0:    "test_regular_user",
			V1:    "viewer",
		}
		db.Create(&regularUserRule)

		// Check if admin user has access
		hasAccess, err := models.CanAccessFeature(db, "test_admin_user", "admin_feature")
		assert.NoError(t, err)
		assert.True(t, hasAccess, "Admin user should have access to admin feature")

		// Check if regular user has access
		hasAccess, err = models.CanAccessFeature(db, "test_regular_user", "admin_feature")
		assert.NoError(t, err)
		assert.False(t, hasAccess, "Regular user should not have access to admin feature")
	})
}
