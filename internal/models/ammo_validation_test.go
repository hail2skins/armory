package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestAmmoValidate tests basic validation functions without involving the database
func TestAmmoValidate(t *testing.T) {
	t.Run("NameTooLong", func(t *testing.T) {
		// Create an ammo with a name that's too long
		longName := ""
		for i := 0; i < 101; i++ {
			longName += "a"
		}

		ammo := &Ammo{
			Name:      longName,
			BrandID:   1,
			CaliberID: 1,
			Count:     100,
		}

		// Call Validate with nil db to only check simple validations
		err := ammo.Validate(nil)

		// Assertions
		assert.Error(t, err)
		assert.Equal(t, ErrAmmoNameTooLong, err)
	})

	t.Run("NegativePrice", func(t *testing.T) {
		// Create an ammo with a negative price
		price := -10.0
		ammo := &Ammo{
			Name:      "Test Ammo",
			BrandID:   1,
			CaliberID: 1,
			Count:     100,
			Paid:      &price,
		}

		// Call Validate
		err := ammo.Validate(nil)

		// Assertions
		assert.Error(t, err)
		assert.Equal(t, ErrAmmoNegativePrice, err)
	})

	t.Run("NegativeCount", func(t *testing.T) {
		// Create an ammo with a negative count
		ammo := &Ammo{
			Name:      "Test Ammo",
			BrandID:   1,
			CaliberID: 1,
			Count:     -5,
		}

		// Call Validate
		err := ammo.Validate(nil)

		// Assertions
		assert.Error(t, err)
		assert.Equal(t, ErrAmmoNegativeCount, err)
	})

	t.Run("FutureDate", func(t *testing.T) {
		// Create an ammo with a future acquisition date
		futureDate := time.Now().AddDate(0, 0, 1) // Tomorrow
		ammo := &Ammo{
			Name:      "Test Ammo",
			BrandID:   1,
			CaliberID: 1,
			Count:     100,
			Acquired:  &futureDate,
		}

		// Call Validate
		err := ammo.Validate(nil)

		// Assertions
		assert.Error(t, err)
		assert.Equal(t, ErrAmmoFutureDate, err)
	})
}

// TestAmmoValidateWithDB tests validations that require database connectivity
func TestAmmoValidateWithDB(t *testing.T) {
	// Skip DB tests in short mode
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	// Get test database (uses the singleton pattern)
	db := GetTestDB()

	// Find a Brand, Caliber, BulletStyle, Grain, and Casing from the seeded data
	var brand Brand
	db.First(&brand)
	assert.NotEqual(t, uint(0), brand.ID, "No brand found in test database")

	var caliber Caliber
	db.First(&caliber)
	assert.NotEqual(t, uint(0), caliber.ID, "No caliber found in test database")

	var bulletStyle BulletStyle
	db.First(&bulletStyle)
	assert.NotEqual(t, uint(0), bulletStyle.ID, "No bullet style found in test database")

	var grain Grain
	db.First(&grain)
	assert.NotEqual(t, uint(0), grain.ID, "No grain found in test database")

	var casing Casing
	db.First(&casing)
	assert.NotEqual(t, uint(0), casing.ID, "No casing found in test database")

	t.Run("ValidAmmo", func(t *testing.T) {
		// Set up a valid ammo
		ammo := &Ammo{
			Name:          "Test Validation Ammo",
			BrandID:       brand.ID,
			CaliberID:     caliber.ID,
			BulletStyleID: bulletStyle.ID,
			GrainID:       grain.ID,
			CasingID:      casing.ID,
			Count:         100,
			OwnerID:       1, // Assuming test database has users
		}

		// Call Validate with real DB
		err := ammo.Validate(db)

		// Assertions
		assert.NoError(t, err)
	})

	t.Run("InvalidBrand", func(t *testing.T) {
		// Create an ammo with an invalid brand ID
		ammo := &Ammo{
			Name:      "Test Ammo Invalid Brand",
			BrandID:   999, // Non-existent ID
			CaliberID: caliber.ID,
			Count:     100,
			OwnerID:   1,
		}

		// Call Validate with real DB
		err := ammo.Validate(db)

		// Assertions
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidBrand, err)
	})

	t.Run("InvalidCaliber", func(t *testing.T) {
		// Create an ammo with an invalid caliber ID
		ammo := &Ammo{
			Name:      "Test Ammo Invalid Caliber",
			BrandID:   brand.ID,
			CaliberID: 999, // Non-existent ID
			Count:     100,
			OwnerID:   1,
		}

		// Call Validate with real DB
		err := ammo.Validate(db)

		// Assertions
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidCaliber, err)
	})
}

// TestCreateAmmoWithValidation tests the CreateAmmoWithValidation function
func TestCreateAmmoWithValidation(t *testing.T) {
	// TODO: Implement test for CreateAmmoWithValidation
	// This would require mocking the gorm.DB Create method
}

// TestUpdateAmmoWithValidation tests the UpdateAmmoWithValidation function
func TestUpdateAmmoWithValidation(t *testing.T) {
	// TODO: Implement test for UpdateAmmoWithValidation
	// This would require mocking several gorm.DB methods
}
