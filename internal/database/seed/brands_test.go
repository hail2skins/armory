package seed_test

import (
	"testing"

	"github.com/hail2skins/armory/internal/database/seed"
	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/testutils"
	"github.com/stretchr/testify/assert"
)

func TestSeedBrands(t *testing.T) {
	// Get the test database connection
	db := testutils.SharedTestService().GetDB()

	// Ensure Brand table exists (auto-migrate)
	err := db.AutoMigrate(&models.Brand{})
	assert.NoError(t, err)

	// Cleanup - delete any existing brands
	db.Exec("DELETE FROM brands")

	// Run the seeder
	seed.SeedBrands(db)

	// Verify that we have brands in the database
	var count int64
	db.Model(&models.Brand{}).Count(&count)
	assert.Greater(t, count, int64(0), "Should have seeded some brands")

	// Check for a specific brand that should have been created - Federal
	var federal models.Brand
	err = db.Where("name = ?", "Federal Premium Ammunition").First(&federal).Error
	assert.NoError(t, err)
	assert.Equal(t, 100, federal.Popularity)
	assert.Equal(t, "Federal", federal.Nickname)

	// Check for the Other/Unknown brand
	var other models.Brand
	err = db.Where("name = ?", "Other/Unknown").First(&other).Error
	assert.NoError(t, err)
	assert.Equal(t, 999, other.Popularity)
	assert.Equal(t, "Other", other.Nickname)

	// Run the seeder again to test the update branch
	// First, change a record
	federal.Popularity = 90
	err = db.Save(&federal).Error
	assert.NoError(t, err)

	// Run the seeder again
	seed.SeedBrands(db)

	// Verify that the record was updated back to original popularity
	err = db.Where("name = ?", "Federal Premium Ammunition").First(&federal).Error
	assert.NoError(t, err)
	assert.Equal(t, 100, federal.Popularity)

	// Clean up after the test
	db.Exec("DELETE FROM brands")
}
