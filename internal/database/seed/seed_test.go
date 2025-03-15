package seed

import (
	"testing"

	"github.com/hail2skins/armory/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSeedManufacturers(t *testing.T) {
	// Setup in-memory database for testing
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
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
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
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
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
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
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
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
