package models

import (
	"testing"
	"time"

	"github.com/hail2skins/armory/internal/testutils/testhelper"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestCreateGun tests the CreateGun function
func TestCreateGun(t *testing.T) {
	// Setup in-memory SQLite database
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	// Migrate the schema
	err = db.AutoMigrate(&Gun{}, &WeaponType{}, &Caliber{}, &Manufacturer{})
	assert.NoError(t, err)

	// Seed test data
	testhelper.SeedTestData(db)

	// Get test data from seeded data
	var weaponType WeaponType
	err = db.Where("type = ?", "Test Rifle").First(&weaponType).Error
	assert.NoError(t, err)

	var caliber Caliber
	err = db.Where("caliber = ?", "Test .223").First(&caliber).Error
	assert.NoError(t, err)

	var manufacturer Manufacturer
	err = db.Where("name = ?", "Test Manufacturer").First(&manufacturer).Error
	assert.NoError(t, err)

	// Test creating a gun
	acquired := time.Now()
	gun := &Gun{
		Name:           "Test Gun",
		SerialNumber:   "123456",
		Acquired:       &acquired,
		WeaponTypeID:   weaponType.ID,
		CaliberID:      caliber.ID,
		ManufacturerID: manufacturer.ID,
		OwnerID:        1, // Test owner ID
	}

	// Call the function being tested
	err = CreateGun(db, gun)
	assert.NoError(t, err)

	// Verify the gun was created
	var createdGun Gun
	err = db.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").First(&createdGun, gun.ID).Error
	assert.NoError(t, err)

	// Verify the gun data
	assert.Equal(t, "Test Gun", createdGun.Name)
	assert.Equal(t, "123456", createdGun.SerialNumber)
	assert.Equal(t, weaponType.ID, createdGun.WeaponTypeID)
	assert.Equal(t, weaponType.Type, createdGun.WeaponType.Type)
	assert.Equal(t, caliber.ID, createdGun.CaliberID)
	assert.Equal(t, caliber.Caliber, createdGun.Caliber.Caliber)
	assert.Equal(t, manufacturer.ID, createdGun.ManufacturerID)
	assert.Equal(t, manufacturer.Name, createdGun.Manufacturer.Name)
	assert.Equal(t, uint(1), createdGun.OwnerID)
}

// TestFindGunsByOwner tests the FindGunsByOwner function
func TestFindGunsByOwner(t *testing.T) {
	// Setup in-memory SQLite database
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	// Migrate the schema
	err = db.AutoMigrate(&Gun{}, &WeaponType{}, &Caliber{}, &Manufacturer{})
	assert.NoError(t, err)

	// Seed test data
	testhelper.SeedTestData(db)

	// Get test data from seeded data
	var weaponType WeaponType
	err = db.Where("type = ?", "Test Rifle").First(&weaponType).Error
	assert.NoError(t, err)

	var caliber Caliber
	err = db.Where("caliber = ?", "Test .223").First(&caliber).Error
	assert.NoError(t, err)

	var manufacturer Manufacturer
	err = db.Where("name = ?", "Test Manufacturer").First(&manufacturer).Error
	assert.NoError(t, err)

	// Clear any existing guns for owner 1
	db.Where("owner_id = ?", 1).Delete(&Gun{})

	// Create test guns for owner 1
	for i := 0; i < 3; i++ {
		acquired := time.Now()
		gun := &Gun{
			Name:           "Test Gun " + string(rune(i+49)), // "Test Gun 1", "Test Gun 2", etc.
			SerialNumber:   "12345" + string(rune(i+49)),
			Acquired:       &acquired,
			WeaponTypeID:   weaponType.ID,
			CaliberID:      caliber.ID,
			ManufacturerID: manufacturer.ID,
			OwnerID:        1, // Test owner ID
		}
		err = db.Create(gun).Error
		assert.NoError(t, err)
	}

	// Create a gun for a different owner
	acquired := time.Now()
	otherGun := &Gun{
		Name:           "Other Gun",
		SerialNumber:   "654321",
		Acquired:       &acquired,
		WeaponTypeID:   weaponType.ID,
		CaliberID:      caliber.ID,
		ManufacturerID: manufacturer.ID,
		OwnerID:        2, // Different owner ID
	}
	err = db.Create(otherGun).Error
	assert.NoError(t, err)

	// Call the function being tested
	guns, err := FindGunsByOwner(db, 1)
	assert.NoError(t, err)

	// Verify the correct guns were returned
	assert.Equal(t, 3, len(guns))
	for _, gun := range guns {
		assert.Equal(t, uint(1), gun.OwnerID)
	}

	// Test with a different owner
	guns, err = FindGunsByOwner(db, 2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(guns))
	assert.Equal(t, "Other Gun", guns[0].Name)
}

// TestFindGunByID tests the FindGunByID function
func TestFindGunByID(t *testing.T) {
	// Setup in-memory SQLite database
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	// Migrate the schema
	err = db.AutoMigrate(&Gun{}, &WeaponType{}, &Caliber{}, &Manufacturer{})
	assert.NoError(t, err)

	// Seed test data
	testhelper.SeedTestData(db)

	// Get test data from seeded data
	var weaponType WeaponType
	err = db.Where("type = ?", "Test Rifle").First(&weaponType).Error
	assert.NoError(t, err)

	var caliber Caliber
	err = db.Where("caliber = ?", "Test .223").First(&caliber).Error
	assert.NoError(t, err)

	var manufacturer Manufacturer
	err = db.Where("name = ?", "Test Manufacturer").First(&manufacturer).Error
	assert.NoError(t, err)

	// Create a test gun
	acquired := time.Now()
	gun := &Gun{
		Name:           "Test Gun",
		SerialNumber:   "123456",
		Acquired:       &acquired,
		WeaponTypeID:   weaponType.ID,
		CaliberID:      caliber.ID,
		ManufacturerID: manufacturer.ID,
		OwnerID:        1, // Test owner ID
	}
	err = db.Create(gun).Error
	assert.NoError(t, err)

	// Call the function being tested
	foundGun, err := FindGunByID(db, gun.ID, 1)
	assert.NoError(t, err)
	assert.NotNil(t, foundGun)

	// Verify the gun data
	assert.Equal(t, "Test Gun", foundGun.Name)
	assert.Equal(t, "123456", foundGun.SerialNumber)
	assert.Equal(t, weaponType.ID, foundGun.WeaponTypeID)
	assert.Equal(t, weaponType.Type, foundGun.WeaponType.Type)
	assert.Equal(t, caliber.ID, foundGun.CaliberID)
	assert.Equal(t, caliber.Caliber, foundGun.Caliber.Caliber)
	assert.Equal(t, manufacturer.ID, foundGun.ManufacturerID)
	assert.Equal(t, manufacturer.Name, foundGun.Manufacturer.Name)
	assert.Equal(t, uint(1), foundGun.OwnerID)

	// Test with wrong owner ID
	foundGun, err = FindGunByID(db, gun.ID, 2)
	assert.Error(t, err)
	assert.Nil(t, foundGun)

	// Test with non-existent gun ID
	foundGun, err = FindGunByID(db, 999, 1)
	assert.Error(t, err)
	assert.Nil(t, foundGun)
}
