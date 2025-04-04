package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestCreateAmmo tests the CreateAmmo function
func TestCreateAmmo(t *testing.T) {
	// Get a shared database instance for testing
	db := GetTestDB()

	// Get test data from seeded data
	var brand Brand
	err := db.Where("name = ?", "Test Federal").First(&brand).Error
	assert.NoError(t, err)

	var bulletStyle BulletStyle
	err = db.Where("type = ?", "FMJ").First(&bulletStyle).Error
	if err != nil {
		// If not found, create a test bullet style
		bulletStyle = BulletStyle{
			Type:       "FMJ",
			Nickname:   "Full Metal Jacket",
			Popularity: 100,
		}
		err = db.Create(&bulletStyle).Error
		assert.NoError(t, err)
	}

	var grain Grain
	err = db.Where("weight = ?", 115).First(&grain).Error
	if err != nil {
		// If not found, create a test grain
		grain = Grain{
			Weight:     115,
			Popularity: 85,
		}
		err = db.Create(&grain).Error
		assert.NoError(t, err)
	}

	var caliber Caliber
	err = db.Where("caliber = ?", "Test 9mm").First(&caliber).Error
	assert.NoError(t, err)

	var casing Casing
	err = db.Where("type = ?", "Brass").First(&casing).Error
	if err != nil {
		// If not found, create a test casing
		casing = Casing{
			Type:       "Brass",
			Popularity: 100,
		}
		err = db.Create(&casing).Error
		assert.NoError(t, err)
	}

	// Test creating ammo
	acquired := time.Now()
	paid := 24.99
	ammo := &Ammo{
		Name:          "Test 9mm Ammo",
		Acquired:      &acquired,
		BrandID:       brand.ID,
		BulletStyleID: bulletStyle.ID,
		GrainID:       grain.ID,
		CaliberID:     caliber.ID,
		CasingID:      casing.ID,
		OwnerID:       1, // Test owner ID
		Paid:          &paid,
		Count:         50,
	}

	// Call the function being tested
	err = CreateAmmo(db, ammo)
	assert.NoError(t, err)

	// Verify the ammo was created
	var createdAmmo Ammo
	err = db.Preload("Brand").Preload("BulletStyle").Preload("Grain").Preload("Caliber").Preload("Casing").First(&createdAmmo, ammo.ID).Error
	assert.NoError(t, err)

	// Verify the ammo data
	assert.Equal(t, "Test 9mm Ammo", createdAmmo.Name)
	assert.Equal(t, brand.ID, createdAmmo.BrandID)
	assert.Equal(t, brand.Name, createdAmmo.Brand.Name)
	assert.Equal(t, bulletStyle.ID, createdAmmo.BulletStyleID)
	assert.Equal(t, bulletStyle.Type, createdAmmo.BulletStyle.Type)
	assert.Equal(t, grain.ID, createdAmmo.GrainID)
	assert.Equal(t, grain.Weight, createdAmmo.Grain.Weight)
	assert.Equal(t, caliber.ID, createdAmmo.CaliberID)
	assert.Equal(t, caliber.Caliber, createdAmmo.Caliber.Caliber)
	assert.Equal(t, casing.ID, createdAmmo.CasingID)
	assert.Equal(t, casing.Type, createdAmmo.Casing.Type)
	assert.Equal(t, uint(1), createdAmmo.OwnerID)
	assert.Equal(t, 50, createdAmmo.Count)
	assert.Equal(t, 24.99, *createdAmmo.Paid)

	// Clean up test data
	db.Delete(&ammo)
}

// TestFindAmmoByOwner tests the FindAmmoByOwner function
func TestFindAmmoByOwner(t *testing.T) {
	// Get a shared database instance for testing
	db := GetTestDB()

	// Get test data from seeded data
	var brand Brand
	err := db.Where("name = ?", "Test Federal").First(&brand).Error
	assert.NoError(t, err)

	var caliber Caliber
	err = db.Where("caliber = ?", "Test 9mm").First(&caliber).Error
	assert.NoError(t, err)

	// We'll need a bullet style, grain, and casing
	var bulletStyle BulletStyle
	err = db.Where("type = ?", "FMJ").First(&bulletStyle).Error
	if err != nil {
		bulletStyle = BulletStyle{Type: "FMJ", Nickname: "Full Metal Jacket", Popularity: 100}
		db.Create(&bulletStyle)
	}

	var grain Grain
	err = db.Where("weight = ?", 115).First(&grain).Error
	if err != nil {
		grain = Grain{Weight: 115, Popularity: 85}
		db.Create(&grain)
	}

	var casing Casing
	err = db.Where("type = ?", "Brass").First(&casing).Error
	if err != nil {
		casing = Casing{Type: "Brass", Popularity: 100}
		db.Create(&casing)
	}

	// Clear any existing test ammo for owner 1
	db.Where("owner_id = ? AND name LIKE ?", 1, "Test Custom Ammo %").Delete(&Ammo{})

	// Create test ammo for owner 1
	var createdAmmo []Ammo
	for i := 0; i < 3; i++ {
		acquired := time.Now()
		paid := 19.99 + float64(i)
		ammo := &Ammo{
			Name:          "Test Custom Ammo " + string(rune(i+49)), // "Test Custom Ammo 1", "Test Custom Ammo 2", etc.
			Acquired:      &acquired,
			BrandID:       brand.ID,
			BulletStyleID: bulletStyle.ID,
			GrainID:       grain.ID,
			CaliberID:     caliber.ID,
			CasingID:      casing.ID,
			OwnerID:       1, // Test owner ID
			Paid:          &paid,
			Count:         50 + (i * 10),
		}
		err = db.Create(ammo).Error
		assert.NoError(t, err)
		createdAmmo = append(createdAmmo, *ammo)
	}

	// Create ammo for a different owner
	acquired := time.Now()
	paid := 29.99
	otherAmmo := &Ammo{
		Name:          "Test Custom Other Ammo",
		Acquired:      &acquired,
		BrandID:       brand.ID,
		BulletStyleID: bulletStyle.ID,
		GrainID:       grain.ID,
		CaliberID:     caliber.ID,
		CasingID:      casing.ID,
		OwnerID:       2, // Different owner ID
		Paid:          &paid,
		Count:         100,
	}
	err = db.Create(otherAmmo).Error
	assert.NoError(t, err)

	// Call the function being tested
	ammoList, err := FindAmmoByOwner(db, 1)
	assert.NoError(t, err)

	// Verify at least our test ammo was returned (there may be others)
	assert.GreaterOrEqual(t, len(ammoList), 3)

	// Count our test ammo
	var testAmmoCount int
	for _, ammo := range ammoList {
		for _, testAmmo := range createdAmmo {
			if ammo.ID == testAmmo.ID {
				testAmmoCount++
				assert.Equal(t, uint(1), ammo.OwnerID)
				break
			}
		}
	}
	assert.Equal(t, 3, testAmmoCount, "All our test ammo should be found")

	// Test with a different owner
	ammoList, err = FindAmmoByOwner(db, 2)
	assert.NoError(t, err)
	assert.Contains(t, extractAmmoNames(ammoList), "Test Custom Other Ammo")

	// Clean up test data
	for _, ammo := range createdAmmo {
		db.Delete(&ammo)
	}
	db.Delete(&otherAmmo)
}

// Helper function to extract names from ammo
func extractAmmoNames(ammoList []Ammo) []string {
	var names []string
	for _, ammo := range ammoList {
		names = append(names, ammo.Name)
	}
	return names
}

// TestFindAmmoByID tests the FindAmmoByID function
func TestFindAmmoByID(t *testing.T) {
	// Get a shared database instance for testing
	db := GetTestDB()

	// Get test data from seeded data
	var brand Brand
	err := db.Where("name = ?", "Test Federal").First(&brand).Error
	assert.NoError(t, err)

	var caliber Caliber
	err = db.Where("caliber = ?", "Test 9mm").First(&caliber).Error
	assert.NoError(t, err)

	// We'll need a bullet style, grain, and casing
	var bulletStyle BulletStyle
	err = db.Where("type = ?", "FMJ").First(&bulletStyle).Error
	if err != nil {
		bulletStyle = BulletStyle{Type: "FMJ", Nickname: "Full Metal Jacket", Popularity: 100}
		db.Create(&bulletStyle)
	}

	var grain Grain
	err = db.Where("weight = ?", 115).First(&grain).Error
	if err != nil {
		grain = Grain{Weight: 115, Popularity: 85}
		db.Create(&grain)
	}

	var casing Casing
	err = db.Where("type = ?", "Brass").First(&casing).Error
	if err != nil {
		casing = Casing{Type: "Brass", Popularity: 100}
		db.Create(&casing)
	}

	// Create a test ammo
	acquired := time.Now()
	paid := 24.99
	ammo := &Ammo{
		Name:          "Test Custom FindByID Ammo",
		Acquired:      &acquired,
		BrandID:       brand.ID,
		BulletStyleID: bulletStyle.ID,
		GrainID:       grain.ID,
		CaliberID:     caliber.ID,
		CasingID:      casing.ID,
		OwnerID:       1,
		Paid:          &paid,
		Count:         50,
	}
	err = db.Create(ammo).Error
	assert.NoError(t, err)

	// Call the function being tested
	foundAmmo, err := FindAmmoByID(db, ammo.ID, 1)
	assert.NoError(t, err)

	// Verify the ammo was found and data is correct
	assert.Equal(t, ammo.ID, foundAmmo.ID)
	assert.Equal(t, "Test Custom FindByID Ammo", foundAmmo.Name)
	assert.Equal(t, brand.ID, foundAmmo.BrandID)
	assert.Equal(t, 50, foundAmmo.Count)

	// Test with an invalid ID
	_, err = FindAmmoByID(db, 99999, 1)
	assert.Error(t, err)
	assert.True(t, err.Error() == "record not found")

	// Clean up
	db.Delete(&ammo)
}

// TestUpdateAmmo tests the UpdateAmmo function
func TestUpdateAmmo(t *testing.T) {
	// Get a shared database instance for testing
	db := GetTestDB()

	// Get test data from seeded data
	var brand Brand
	err := db.Where("name = ?", "Test Federal").First(&brand).Error
	assert.NoError(t, err)

	var caliber Caliber
	err = db.Where("caliber = ?", "Test 9mm").First(&caliber).Error
	assert.NoError(t, err)

	// We'll need a bullet style, grain, and casing
	var bulletStyle BulletStyle
	err = db.Where("type = ?", "FMJ").First(&bulletStyle).Error
	if err != nil {
		bulletStyle = BulletStyle{Type: "FMJ", Nickname: "Full Metal Jacket", Popularity: 100}
		db.Create(&bulletStyle)
	}

	var grain Grain
	err = db.Where("weight = ?", 115).First(&grain).Error
	if err != nil {
		grain = Grain{Weight: 115, Popularity: 85}
		db.Create(&grain)
	}

	var casing Casing
	err = db.Where("type = ?", "Brass").First(&casing).Error
	if err != nil {
		casing = Casing{Type: "Brass", Popularity: 100}
		db.Create(&casing)
	}

	// Create a test ammo
	acquired := time.Now()
	paid := 24.99
	ammo := &Ammo{
		Name:          "Test Update Ammo Original",
		Acquired:      &acquired,
		BrandID:       brand.ID,
		BulletStyleID: bulletStyle.ID,
		GrainID:       grain.ID,
		CaliberID:     caliber.ID,
		CasingID:      casing.ID,
		OwnerID:       1,
		Paid:          &paid,
		Count:         50,
	}
	err = db.Create(ammo).Error
	assert.NoError(t, err)

	// Update the ammo
	ammo.Name = "Test Update Ammo Modified"
	ammo.Count = 75
	newPaid := 29.99
	ammo.Paid = &newPaid

	// Call the function being tested
	err = UpdateAmmo(db, ammo)
	assert.NoError(t, err)

	// Verify the ammo was updated
	var updatedAmmo Ammo
	err = db.First(&updatedAmmo, ammo.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "Test Update Ammo Modified", updatedAmmo.Name)
	assert.Equal(t, 75, updatedAmmo.Count)
	assert.Equal(t, 29.99, *updatedAmmo.Paid)

	// Clean up
	db.Delete(&ammo)
}

// TestDeleteAmmo tests the DeleteAmmo function
func TestDeleteAmmo(t *testing.T) {
	// Get a shared database instance for testing
	db := GetTestDB()

	// Get test data from seeded data
	var brand Brand
	err := db.Where("name = ?", "Test Federal").First(&brand).Error
	assert.NoError(t, err)

	var caliber Caliber
	err = db.Where("caliber = ?", "Test 9mm").First(&caliber).Error
	assert.NoError(t, err)

	// We'll need a bullet style, grain, and casing
	var bulletStyle BulletStyle
	err = db.Where("type = ?", "FMJ").First(&bulletStyle).Error
	if err != nil {
		bulletStyle = BulletStyle{Type: "FMJ", Nickname: "Full Metal Jacket", Popularity: 100}
		db.Create(&bulletStyle)
	}

	var grain Grain
	err = db.Where("weight = ?", 115).First(&grain).Error
	if err != nil {
		grain = Grain{Weight: 115, Popularity: 85}
		db.Create(&grain)
	}

	var casing Casing
	err = db.Where("type = ?", "Brass").First(&casing).Error
	if err != nil {
		casing = Casing{Type: "Brass", Popularity: 100}
		db.Create(&casing)
	}

	// Create a test ammo
	acquired := time.Now()
	paid := 24.99
	ammo := &Ammo{
		Name:          "Test Delete Ammo",
		Acquired:      &acquired,
		BrandID:       brand.ID,
		BulletStyleID: bulletStyle.ID,
		GrainID:       grain.ID,
		CaliberID:     caliber.ID,
		CasingID:      casing.ID,
		OwnerID:       1,
		Paid:          &paid,
		Count:         50,
	}
	err = db.Create(ammo).Error
	assert.NoError(t, err)

	// Call the function being tested
	err = DeleteAmmo(db, ammo.ID, 1)
	assert.NoError(t, err)

	// Verify the ammo was deleted
	_, err = FindAmmoByID(db, ammo.ID, 1)
	assert.Error(t, err)
	assert.True(t, err.Error() == "record not found")
}

// TestCreateAmmoWithExpended tests creating ammo with valid and invalid Expended values
func TestCreateAmmoWithExpended(t *testing.T) {
	// Get a shared database instance for testing
	db := GetTestDB()

	// Get test data from seeded data
	var brand Brand
	err := db.Where("name = ?", "Test Federal").First(&brand).Error
	assert.NoError(t, err)

	var caliber Caliber
	err = db.Where("caliber = ?", "Test 9mm").First(&caliber).Error
	assert.NoError(t, err)

	// We'll need a bullet style, grain, and casing
	var bulletStyle BulletStyle
	err = db.Where("type = ?", "FMJ").First(&bulletStyle).Error
	if err != nil {
		bulletStyle = BulletStyle{Type: "FMJ", Nickname: "Full Metal Jacket", Popularity: 100}
		db.Create(&bulletStyle)
	}

	var grain Grain
	err = db.Where("weight = ?", 115).First(&grain).Error
	if err != nil {
		grain = Grain{Weight: 115, Popularity: 85}
		db.Create(&grain)
	}

	var casing Casing
	err = db.Where("type = ?", "Brass").First(&casing).Error
	if err != nil {
		casing = Casing{Type: "Brass", Popularity: 100}
		db.Create(&casing)
	}

	// Test case 1: Create ammo with valid Expended value (0)
	acquired := time.Now()
	paid := 24.99
	ammo1 := &Ammo{
		Name:          "Test Ammo With Expended Zero",
		Acquired:      &acquired,
		BrandID:       brand.ID,
		BulletStyleID: bulletStyle.ID,
		GrainID:       grain.ID,
		CaliberID:     caliber.ID,
		CasingID:      casing.ID,
		OwnerID:       1,
		Paid:          &paid,
		Count:         50,
		Expended:      0,
	}
	err = CreateAmmoWithValidation(db, ammo1)
	assert.NoError(t, err)

	// Test case 2: Create ammo with valid Expended value (equal to Count)
	ammo2 := &Ammo{
		Name:          "Test Ammo With Expended Equal",
		Acquired:      &acquired,
		BrandID:       brand.ID,
		BulletStyleID: bulletStyle.ID,
		GrainID:       grain.ID,
		CaliberID:     caliber.ID,
		CasingID:      casing.ID,
		OwnerID:       1,
		Paid:          &paid,
		Count:         50,
		Expended:      50,
	}
	err = CreateAmmoWithValidation(db, ammo2)
	assert.NoError(t, err)

	// Test case 3: Create ammo with valid Expended value (less than Count)
	ammo3 := &Ammo{
		Name:          "Test Ammo With Expended Less",
		Acquired:      &acquired,
		BrandID:       brand.ID,
		BulletStyleID: bulletStyle.ID,
		GrainID:       grain.ID,
		CaliberID:     caliber.ID,
		CasingID:      casing.ID,
		OwnerID:       1,
		Paid:          &paid,
		Count:         50,
		Expended:      25,
	}
	err = CreateAmmoWithValidation(db, ammo3)
	assert.NoError(t, err)

	// Test case 4: Try to create ammo with negative Expended value
	ammo4 := &Ammo{
		Name:          "Test Ammo With Negative Expended",
		Acquired:      &acquired,
		BrandID:       brand.ID,
		BulletStyleID: bulletStyle.ID,
		GrainID:       grain.ID,
		CaliberID:     caliber.ID,
		CasingID:      casing.ID,
		OwnerID:       1,
		Paid:          &paid,
		Count:         50,
		Expended:      -5,
	}
	err = CreateAmmoWithValidation(db, ammo4)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expended count cannot be negative")

	// Test case 5: Try to create ammo with Expended > Count
	ammo5 := &Ammo{
		Name:          "Test Ammo With Expended Greater",
		Acquired:      &acquired,
		BrandID:       brand.ID,
		BulletStyleID: bulletStyle.ID,
		GrainID:       grain.ID,
		CaliberID:     caliber.ID,
		CasingID:      casing.ID,
		OwnerID:       1,
		Paid:          &paid,
		Count:         50,
		Expended:      75,
	}
	err = CreateAmmoWithValidation(db, ammo5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expended count cannot be greater than total count")

	// Verify the created ammo have the correct Expended values
	var foundAmmo1 Ammo
	err = db.First(&foundAmmo1, ammo1.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, 0, foundAmmo1.Expended)

	var foundAmmo2 Ammo
	err = db.First(&foundAmmo2, ammo2.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, 50, foundAmmo2.Expended)

	var foundAmmo3 Ammo
	err = db.First(&foundAmmo3, ammo3.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, 25, foundAmmo3.Expended)

	// Clean up
	db.Delete(&ammo1)
	db.Delete(&ammo2)
	db.Delete(&ammo3)
}

// TestUpdateAmmoWithExpended tests updating ammo with valid and invalid Expended values
func TestUpdateAmmoWithExpended(t *testing.T) {
	// Get a shared database instance for testing
	db := GetTestDB()

	// Get test data from seeded data
	var brand Brand
	err := db.Where("name = ?", "Test Federal").First(&brand).Error
	assert.NoError(t, err)

	var caliber Caliber
	err = db.Where("caliber = ?", "Test 9mm").First(&caliber).Error
	assert.NoError(t, err)

	// We'll need a bullet style, grain, and casing
	var bulletStyle BulletStyle
	err = db.Where("type = ?", "FMJ").First(&bulletStyle).Error
	if err != nil {
		bulletStyle = BulletStyle{Type: "FMJ", Nickname: "Full Metal Jacket", Popularity: 100}
		db.Create(&bulletStyle)
	}

	var grain Grain
	err = db.Where("weight = ?", 115).First(&grain).Error
	if err != nil {
		grain = Grain{Weight: 115, Popularity: 85}
		db.Create(&grain)
	}

	var casing Casing
	err = db.Where("type = ?", "Brass").First(&casing).Error
	if err != nil {
		casing = Casing{Type: "Brass", Popularity: 100}
		db.Create(&casing)
	}

	// Create a test ammo with 0 expended
	acquired := time.Now()
	paid := 24.99
	ammo := &Ammo{
		Name:          "Test Update Ammo Expended",
		Acquired:      &acquired,
		BrandID:       brand.ID,
		BulletStyleID: bulletStyle.ID,
		GrainID:       grain.ID,
		CaliberID:     caliber.ID,
		CasingID:      casing.ID,
		OwnerID:       1,
		Paid:          &paid,
		Count:         50,
		Expended:      0,
	}
	err = db.Create(ammo).Error
	assert.NoError(t, err)

	// Test case 1: Update with valid Expended value
	ammo.Expended = 25
	err = UpdateAmmoWithValidation(db, ammo)
	assert.NoError(t, err)

	// Verify update
	var updatedAmmo Ammo
	err = db.First(&updatedAmmo, ammo.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, 25, updatedAmmo.Expended)

	// Test case 2: Update with Expended = Count
	ammo.Expended = 50
	err = UpdateAmmoWithValidation(db, ammo)
	assert.NoError(t, err)

	// Verify update
	err = db.First(&updatedAmmo, ammo.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, 50, updatedAmmo.Expended)

	// Test case 3: Try to update with negative Expended
	ammo.Expended = -10
	err = UpdateAmmoWithValidation(db, ammo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expended count cannot be negative")

	// Test case 4: Try to update with Expended > Count
	ammo.Expended = 60
	err = UpdateAmmoWithValidation(db, ammo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expended count cannot be greater than total count")

	// Verify ammo still has previous valid value
	err = db.First(&updatedAmmo, ammo.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, 50, updatedAmmo.Expended)

	// Clean up
	db.Delete(&ammo)
}
