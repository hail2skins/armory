package seed_test

import (
	"testing"

	"github.com/hail2skins/armory/internal/database/seed"
	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/testutils"
	"github.com/stretchr/testify/assert"
)

func TestSeedGrains(t *testing.T) {
	// Get the test database connection
	db := testutils.SharedTestService().GetDB()

	// Ensure Grain table exists (auto-migrate)
	err := db.AutoMigrate(&models.Grain{})
	assert.NoError(t, err)

	// Cleanup - delete any existing grain weights
	db.Exec("DELETE FROM grains")

	// Run the seeder
	seed.SeedGrains(db)

	// Verify that we have grain weights in the database
	var count int64
	db.Model(&models.Grain{}).Count(&count)
	assert.Greater(t, count, int64(0), "Should have seeded some grain weights")

	// Check for a specific grain weight that should have been created - common 9mm weight
	var grain115 models.Grain
	err = db.Where("weight = ?", 115).First(&grain115).Error
	assert.NoError(t, err)
	assert.Equal(t, 100, grain115.Popularity)

	// Run the seeder again to test the update branch
	// First, change a record
	grain115.Popularity = 90
	err = db.Save(&grain115).Error
	assert.NoError(t, err)

	// Run the seeder again
	seed.SeedGrains(db)

	// Verify that the record was updated back to original popularity
	err = db.Where("weight = ?", 115).First(&grain115).Error
	assert.NoError(t, err)
	assert.Equal(t, 100, grain115.Popularity)

	// Clean up after the test
	db.Exec("DELETE FROM grains")
}
