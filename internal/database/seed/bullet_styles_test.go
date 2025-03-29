package seed_test

import (
	"testing"

	"github.com/hail2skins/armory/internal/database/seed"
	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/testutils"
	"github.com/stretchr/testify/assert"
)

func TestSeedBulletStyles(t *testing.T) {
	// Get the test database connection
	db := testutils.SharedTestService().GetDB()

	// Ensure BulletStyle table exists (auto-migrate)
	err := db.AutoMigrate(&models.BulletStyle{})
	assert.NoError(t, err)

	// Cleanup - delete any existing bullet styles
	db.Exec("DELETE FROM bullet_styles")

	// Run the seeder
	seed.SeedBulletStyles(db)

	// Verify that we have bullet styles in the database
	var count int64
	db.Model(&models.BulletStyle{}).Count(&count)
	assert.Greater(t, count, int64(0), "Should have seeded some bullet styles")

	// Check for a specific bullet style that should have been created
	var fmj models.BulletStyle
	err = db.Where("type = ?", "Full Metal Jacket").First(&fmj).Error
	assert.NoError(t, err)
	assert.Equal(t, "FMJ", fmj.Nickname)
	assert.Equal(t, 100, fmj.Popularity)

	// Run the seeder again to test the update branch
	// First, change a record
	fmj.Popularity = 90
	err = db.Save(&fmj).Error
	assert.NoError(t, err)

	// Run the seeder again
	seed.SeedBulletStyles(db)

	// Verify that the record was updated back to original popularity
	err = db.Where("type = ?", "Full Metal Jacket").First(&fmj).Error
	assert.NoError(t, err)
	assert.Equal(t, "FMJ", fmj.Nickname)
	assert.Equal(t, 100, fmj.Popularity)

	// Clean up after the test
	db.Exec("DELETE FROM bullet_styles")
}
