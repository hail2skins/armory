package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupGunTestDB creates a minimal test database for gun validation tests
func setupGunTestDB(t *testing.T) *gorm.DB {
	// Create an in-memory test database
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	// Migrate necessary schemas for foreign key validation
	err = db.AutoMigrate(&Gun{}, &WeaponType{}, &Caliber{}, &Manufacturer{})
	require.NoError(t, err)

	// Create single test instances for foreign key validation
	// Using unique names to avoid conflicts
	weaponType := WeaponType{Type: "Test Type " + time.Now().Format(time.RFC3339Nano), Nickname: "Test"}
	err = db.Create(&weaponType).Error
	require.NoError(t, err)

	caliber := Caliber{Caliber: "Test Caliber " + time.Now().Format(time.RFC3339Nano), Nickname: "Test"}
	err = db.Create(&caliber).Error
	require.NoError(t, err)

	manufacturer := Manufacturer{Name: "Test Manufacturer " + time.Now().Format(time.RFC3339Nano), Country: "Test Country"}
	err = db.Create(&manufacturer).Error
	require.NoError(t, err)

	return db
}

func TestGunNameValidation(t *testing.T) {
	// Test valid name (under max length)
	validGun := &Gun{
		Name: "Valid Gun Name",
	}
	err := validGun.Validate(nil) // nil db since we're only testing name length
	assert.NoError(t, err)

	// Test name at exactly max length (100 chars)
	exactLengthName := string(make([]byte, 100))
	for i := range exactLengthName {
		exactLengthName = exactLengthName[:i] + "a" + exactLengthName[i+1:]
	}
	exactLengthGun := &Gun{
		Name: exactLengthName,
	}
	err = exactLengthGun.Validate(nil)
	assert.NoError(t, err)

	// Test name exceeding max length
	tooLongName := string(make([]byte, 101))
	for i := range tooLongName {
		tooLongName = tooLongName[:i] + "a" + tooLongName[i+1:]
	}
	tooLongGun := &Gun{
		Name: tooLongName,
	}
	err = tooLongGun.Validate(nil)
	assert.Error(t, err)
	assert.Equal(t, ErrGunNameTooLong, err)
}

func TestGunPriceValidation(t *testing.T) {
	// Test valid price (positive)
	validPrice := 100.0
	validGun := &Gun{
		Name: "Valid Gun",
		Paid: &validPrice,
	}
	err := validGun.Validate(nil)
	assert.NoError(t, err)

	// Test valid price (zero)
	zeroPrice := 0.0
	zeroGun := &Gun{
		Name: "Zero Price Gun",
		Paid: &zeroPrice,
	}
	err = zeroGun.Validate(nil)
	assert.NoError(t, err)

	// Test invalid price (negative)
	negativePrice := -10.0
	negativeGun := &Gun{
		Name: "Negative Price Gun",
		Paid: &negativePrice,
	}
	err = negativeGun.Validate(nil)
	assert.Error(t, err)
	assert.Equal(t, ErrNegativePrice, err)

	// Test nil price (should be valid)
	nilPriceGun := &Gun{
		Name: "Nil Price Gun",
		Paid: nil,
	}
	err = nilPriceGun.Validate(nil)
	assert.NoError(t, err)
}

func TestGunDateValidation(t *testing.T) {
	// Test valid date (past)
	pastDate := time.Now().AddDate(0, 0, -1) // Yesterday
	validGun := &Gun{
		Name:     "Valid Date Gun",
		Acquired: &pastDate,
	}
	err := validGun.Validate(nil)
	assert.NoError(t, err)

	// Test valid date (today)
	todayDate := time.Now()
	todayGun := &Gun{
		Name:     "Today Date Gun",
		Acquired: &todayDate,
	}
	err = todayGun.Validate(nil)
	assert.NoError(t, err)

	// Test invalid date (future)
	futureDate := time.Now().AddDate(0, 0, 1) // Tomorrow
	futureGun := &Gun{
		Name:     "Future Date Gun",
		Acquired: &futureDate,
	}
	err = futureGun.Validate(nil)
	assert.Error(t, err)
	assert.Equal(t, ErrFutureDate, err)

	// Test nil date (should be valid)
	nilDateGun := &Gun{
		Name:     "Nil Date Gun",
		Acquired: nil,
	}
	err = nilDateGun.Validate(nil)
	assert.NoError(t, err)
}

func TestGunForeignKeyValidation(t *testing.T) {
	// Setup a test database
	db := setupGunTestDB(t)

	// Get sample data
	var weaponType WeaponType
	var caliber Caliber
	var manufacturer Manufacturer

	db.First(&weaponType)
	db.First(&caliber)
	db.First(&manufacturer)

	// Test valid foreign keys
	validGun := &Gun{
		Name:           "Valid FK Gun",
		WeaponTypeID:   weaponType.ID,
		CaliberID:      caliber.ID,
		ManufacturerID: manufacturer.ID,
	}
	err := validGun.Validate(db)
	assert.NoError(t, err)

	// Test invalid weapon type ID
	invalidWeaponTypeGun := &Gun{
		Name:           "Invalid Weapon Type Gun",
		WeaponTypeID:   99999, // Non-existent ID
		CaliberID:      caliber.ID,
		ManufacturerID: manufacturer.ID,
	}
	err = invalidWeaponTypeGun.Validate(db)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidWeaponType, err)

	// Test invalid caliber ID
	invalidCaliberGun := &Gun{
		Name:           "Invalid Caliber Gun",
		WeaponTypeID:   weaponType.ID,
		CaliberID:      99999, // Non-existent ID
		ManufacturerID: manufacturer.ID,
	}
	err = invalidCaliberGun.Validate(db)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidCaliber, err)

	// Test invalid manufacturer ID
	invalidManufacturerGun := &Gun{
		Name:           "Invalid Manufacturer Gun",
		WeaponTypeID:   weaponType.ID,
		CaliberID:      caliber.ID,
		ManufacturerID: 99999, // Non-existent ID
	}
	err = invalidManufacturerGun.Validate(db)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidManufacturer, err)
}
