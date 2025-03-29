package seed_test

import (
	"log"
	"testing"

	"github.com/hail2skins/armory/internal/database/seed"
	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSeedManufacturers(t *testing.T) {
	// Setup in-memory database for testing
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Auto migrate the schema
	err = db.AutoMigrate(&models.Manufacturer{})
	assert.NoError(t, err)

	// Run the seed function
	seed.SeedManufacturers(db)

	// Check if manufacturers were created
	var count int64
	db.Model(&models.Manufacturer{}).Count(&count)
	assert.Greater(t, count, int64(0), "No manufacturers were seeded")

	// Check for a specific manufacturer that should be in the seed data
	var manufacturer models.Manufacturer
	result := db.Where("name = ?", "Glock").First(&manufacturer)
	assert.NoError(t, result.Error)
	assert.Equal(t, "Glock", manufacturer.Name)
	assert.Equal(t, "Austria", manufacturer.Country)
}

func TestSeedCalibers(t *testing.T) {
	// Setup in-memory database for testing
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Auto migrate the schema
	err = db.AutoMigrate(&models.Caliber{})
	assert.NoError(t, err)

	// Run the seed function
	seed.SeedCalibers(db)

	// Check if calibers were created
	var count int64
	db.Model(&models.Caliber{}).Count(&count)
	assert.Greater(t, count, int64(0), "No calibers were seeded")

	// Check for a specific caliber that should be in the seed data
	var caliber models.Caliber
	result := db.Where("caliber = ?", "9mm Parabellum").First(&caliber)
	assert.NoError(t, result.Error)
	assert.Equal(t, "9mm Parabellum", caliber.Caliber)
}

func TestSeedWeaponTypes(t *testing.T) {
	// Setup in-memory database for testing
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Auto migrate the schema
	err = db.AutoMigrate(&models.WeaponType{})
	assert.NoError(t, err)

	// Run the seed function
	seed.SeedWeaponTypes(db)

	// Check if weapon types were created
	var count int64
	db.Model(&models.WeaponType{}).Count(&count)
	assert.Greater(t, count, int64(0), "No weapon types were seeded")

	// Check for a specific weapon type that should be in the seed data
	var weaponType models.WeaponType
	result := db.Where("type = ?", "Handgun").First(&weaponType)
	assert.NoError(t, result.Error)
	assert.Equal(t, "Handgun", weaponType.Type)
}

// TestRunSeeds checks if seed data is loaded correctly and is idempotent
func TestRunSeeds(t *testing.T) {
	dbService := testutils.SharedTestService()
	db := dbService.GetDB()

	// Clean up before test to ensure clean state
	db.Unscoped().Where("1 = 1").Delete(&models.Caliber{})
	db.Unscoped().Where("1 = 1").Delete(&models.Manufacturer{})
	db.Unscoped().Where("1 = 1").Delete(&models.WeaponType{})
	db.Unscoped().Where("1 = 1").Delete(&models.Casing{})

	// Initial seed run
	seed.RunSeeds(db) // Explicitly run seeds here

	// Get initial counts
	initialCounts := make(map[string]int64)
	var initialCaliberCount, initialManufacturerCount, initialWeaponTypeCount, initialCasingCount int64 // Declare variables first
	db.Model(&models.Caliber{}).Count(&initialCaliberCount)
	db.Model(&models.Manufacturer{}).Count(&initialManufacturerCount)
	db.Model(&models.WeaponType{}).Count(&initialWeaponTypeCount)
	db.Model(&models.Casing{}).Count(&initialCasingCount)

	// Store counts in the map
	initialCounts["calibers"] = initialCaliberCount
	initialCounts["manufacturers"] = initialManufacturerCount
	initialCounts["weapon_types"] = initialWeaponTypeCount
	initialCounts["casings"] = initialCasingCount

	assert.Greater(t, initialCounts["calibers"], int64(0), "Should have seeded calibers")
	assert.Greater(t, initialCounts["manufacturers"], int64(0), "Should have seeded manufacturers")
	assert.Greater(t, initialCounts["weapon_types"], int64(0), "Should have seeded weapon types")
	assert.Greater(t, initialCounts["casings"], int64(0), "Should have seeded casings")

	// Re-run seeds to check for idempotency (no duplicates, updates happen correctly)
	seed.RunSeeds(db)

	// Check counts after second seed run
	var caliberCount, manufacturerCount, weaponTypeCount, casingCount int64
	db.Model(&models.Caliber{}).Count(&caliberCount)
	db.Model(&models.Manufacturer{}).Count(&manufacturerCount)
	db.Model(&models.WeaponType{}).Count(&weaponTypeCount)
	db.Model(&models.Casing{}).Count(&casingCount) // Add casing count check

	assert.Equal(t, initialCounts["calibers"], caliberCount, "Caliber count should not change on second seed")
	assert.Equal(t, initialCounts["manufacturers"], manufacturerCount, "Manufacturer count should not change on second seed")
	assert.Equal(t, initialCounts["weapon_types"], weaponTypeCount, "Weapon type count should not change on second seed")
	assert.Equal(t, initialCounts["casings"], casingCount, "Casing count should not change on second seed") // Add casing count assertion

	// Clean up after test
	db.Unscoped().Where("1 = 1").Delete(&models.Caliber{})
	db.Unscoped().Where("1 = 1").Delete(&models.Manufacturer{})
	db.Unscoped().Where("1 = 1").Delete(&models.WeaponType{})
	db.Unscoped().Where("1 = 1").Delete(&models.Casing{})
}

// Add a new test for the NeedsSeeding function and seed once strategy
func TestNeedsSeeding(t *testing.T) {
	// Create a separate database for this test
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err, "Failed to create in-memory database")

	// Run migrations to create tables
	err = db.AutoMigrate(&models.WeaponType{}, &models.Caliber{}, &models.Manufacturer{})
	require.NoError(t, err, "Failed to run migrations")

	// Test 1: Empty database should need seeding
	assert.True(t, seed.NeedsSeeding(db), "Empty database should need seeding")

	// Test 2: Add one weapon type and check that it no longer needs seeding
	weaponType := models.WeaponType{Type: "Test Type", Nickname: "Test", Popularity: 1}
	err = db.Create(&weaponType).Error
	require.NoError(t, err, "Failed to create test weapon type")

	assert.False(t, seed.NeedsSeeding(db), "Database with data should not need seeding")

	// Test 3: Clear the database and verify it needs seeding again
	db.Exec("DELETE FROM weapon_types")
	db.Exec("DELETE FROM calibers")
	db.Exec("DELETE FROM manufacturers")

	assert.True(t, seed.NeedsSeeding(db), "Empty database should need seeding again")

	// Test 4: Test the seed once behavior by adding only a caliber
	caliber := models.Caliber{Caliber: "Test Caliber", Nickname: "Test", Popularity: 1}
	err = db.Create(&caliber).Error
	require.NoError(t, err, "Failed to create test caliber")

	assert.False(t, seed.NeedsSeeding(db), "Database with only calibers should not need seeding")
}

// Test RunSeeds with the new seed once behavior
func TestRunSeedsOnce(t *testing.T) {
	// Create a separate database for this test
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err, "Failed to create in-memory database")

	// Run migrations to create tables
	err = db.AutoMigrate(&models.WeaponType{}, &models.Caliber{}, &models.Manufacturer{})
	require.NoError(t, err, "Failed to run migrations")

	// Run seeds for the first time
	seed.RunSeeds(db)

	// Count entities to verify data was seeded
	var weaponTypeCount, caliberCount, manufacturerCount int64
	db.Model(&models.WeaponType{}).Count(&weaponTypeCount)
	db.Model(&models.Caliber{}).Count(&caliberCount)
	db.Model(&models.Manufacturer{}).Count(&manufacturerCount)

	assert.Greater(t, weaponTypeCount, int64(0), "Weapon types should be seeded")
	assert.Greater(t, caliberCount, int64(0), "Calibers should be seeded")
	assert.Greater(t, manufacturerCount, int64(0), "Manufacturers should be seeded")

	// Remember the counts
	initialCounts := map[string]int64{
		"weaponTypes":   weaponTypeCount,
		"calibers":      caliberCount,
		"manufacturers": manufacturerCount,
	}

	// Run seeds again
	seed.RunSeeds(db)

	// Verify counts have not changed
	db.Model(&models.WeaponType{}).Count(&weaponTypeCount)
	db.Model(&models.Caliber{}).Count(&caliberCount)
	db.Model(&models.Manufacturer{}).Count(&manufacturerCount)

	assert.Equal(t, initialCounts["weaponTypes"], weaponTypeCount, "Weapon type count should not change on second seed")
	assert.Equal(t, initialCounts["calibers"], caliberCount, "Caliber count should not change on second seed")
	assert.Equal(t, initialCounts["manufacturers"], manufacturerCount, "Manufacturer count should not change on second seed")
}

// TestSeedCasings tests the SeedCasings function specifically
func TestSeedCasings(t *testing.T) {
	dbService := testutils.SharedTestService()
	db := dbService.GetDB()

	// Clean up any existing casings before the test, as DB is shared
	db.Unscoped().Where("1 = 1").Delete(&models.Casing{})

	// Initial seed call for this specific seeder
	seed.SeedCasings(db) // Use seed. qualifier

	// Define expected casings
	expectedCasings := map[string]int{
		"Other":               999,
		"Brass":               100,
		"Steel":               80,
		"Nickel-Plated Brass": 70,
		"Aluminum":            50,
		"Polymer":             20,
	}

	// Verify initial seeding
	for casingType, expectedPopularity := range expectedCasings {
		var casing models.Casing
		err := db.Where("type = ?", casingType).First(&casing).Error
		assert.NoError(t, err, "Casing type '%s' should exist after initial seed", casingType)
		assert.Equal(t, expectedPopularity, casing.Popularity, "Popularity for '%s' should match initial seed", casingType)
	}

	// Check count after initial seed
	var initialCount int64
	db.Model(&models.Casing{}).Count(&initialCount)
	assert.EqualValues(t, len(expectedCasings), initialCount, "Initial casing count should match expected")

	// Modify a popularity value and re-seed to test update logic
	log.Println("Updating Brass popularity for idempotency check...")
	originalBrass, err := models.FindCasingByType(db, "Brass")
	assert.NoError(t, err)

	// Simulate a change that might happen outside the seeder
	originalBrass.Popularity = 110 // Change popularity
	err = models.UpdateCasing(db, originalBrass)
	assert.NoError(t, err)

	// Re-run the seeder
	log.Println("Re-running SeedCasings...")
	seed.SeedCasings(db) // Use seed. qualifier

	// Verify counts haven't changed (no duplicates)
	var countAfterReseed int64
	db.Model(&models.Casing{}).Count(&countAfterReseed)
	assert.EqualValues(t, len(expectedCasings), countAfterReseed, "Total number of casings should not change after re-seeding")

	// Verify the popularity was updated back by the seeder
	updatedBrass, err := models.FindCasingByType(db, "Brass")
	assert.NoError(t, err)
	assert.Equal(t, expectedCasings["Brass"], updatedBrass.Popularity, "Popularity for 'Brass' should be reset to seed value after re-seeding")
	log.Println("SeedCasings test completed.")

	// Clean up casings created by this test
	db.Unscoped().Where("1 = 1").Delete(&models.Casing{})
}
