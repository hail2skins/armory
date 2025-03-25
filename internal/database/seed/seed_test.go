package seed

import (
	"testing"

	"github.com/hail2skins/armory/internal/models"
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
	SeedManufacturers(db)

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
	SeedCalibers(db)

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
	SeedWeaponTypes(db)

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

func TestRunSeeds(t *testing.T) {
	// Setup in-memory database for testing
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Auto migrate the schema
	err = db.AutoMigrate(&models.Manufacturer{}, &models.Caliber{}, &models.WeaponType{})
	assert.NoError(t, err)

	// Run all seeds
	RunSeeds(db)

	// Check if all types of data were seeded
	var manufacturerCount, caliberCount, weaponTypeCount int64

	db.Model(&models.Manufacturer{}).Count(&manufacturerCount)
	assert.Greater(t, manufacturerCount, int64(0), "No manufacturers were seeded")

	db.Model(&models.Caliber{}).Count(&caliberCount)
	assert.Greater(t, caliberCount, int64(0), "No calibers were seeded")

	db.Model(&models.WeaponType{}).Count(&weaponTypeCount)
	assert.Greater(t, weaponTypeCount, int64(0), "No weapon types were seeded")
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
	assert.True(t, NeedsSeeding(db), "Empty database should need seeding")

	// Test 2: Add one weapon type and check that it no longer needs seeding
	weaponType := models.WeaponType{Type: "Test Type", Nickname: "Test", Popularity: 1}
	err = db.Create(&weaponType).Error
	require.NoError(t, err, "Failed to create test weapon type")

	assert.False(t, NeedsSeeding(db), "Database with data should not need seeding")

	// Test 3: Clear the database and verify it needs seeding again
	db.Exec("DELETE FROM weapon_types")
	db.Exec("DELETE FROM calibers")
	db.Exec("DELETE FROM manufacturers")

	assert.True(t, NeedsSeeding(db), "Empty database should need seeding again")

	// Test 4: Test the seed once behavior by adding only a caliber
	caliber := models.Caliber{Caliber: "Test Caliber", Nickname: "Test", Popularity: 1}
	err = db.Create(&caliber).Error
	require.NoError(t, err, "Failed to create test caliber")

	assert.False(t, NeedsSeeding(db), "Database with only calibers should not need seeding")
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
	RunSeeds(db)

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
	RunSeeds(db)

	// Verify counts have not changed
	db.Model(&models.WeaponType{}).Count(&weaponTypeCount)
	db.Model(&models.Caliber{}).Count(&caliberCount)
	db.Model(&models.Manufacturer{}).Count(&manufacturerCount)

	assert.Equal(t, initialCounts["weaponTypes"], weaponTypeCount, "Weapon type count should not change on second seed")
	assert.Equal(t, initialCounts["calibers"], caliberCount, "Caliber count should not change on second seed")
	assert.Equal(t, initialCounts["manufacturers"], manufacturerCount, "Manufacturer count should not change on second seed")
}
