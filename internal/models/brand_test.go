package models_test

import (
	"testing"

	"github.com/hail2skins/armory/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestBrandModel(t *testing.T) {
	// Get a shared database instance for testing
	db := models.GetTestDB()

	// Clear any existing test brands
	db.Exec("DELETE FROM brands WHERE name LIKE 'Test Brand%'")

	// Test creating a brand
	brand := models.Brand{
		Name:       "Test Brand Model",
		Nickname:   "Test",
		Popularity: 5,
	}

	// Save to database
	result := db.Create(&brand)
	assert.NoError(t, result.Error)
	assert.NotZero(t, brand.ID)

	// Test retrieving the brand
	var retrievedBrand models.Brand
	result = db.First(&retrievedBrand, brand.ID)
	assert.NoError(t, result.Error)
	assert.Equal(t, "Test Brand Model", retrievedBrand.Name)
	assert.Equal(t, "Test", retrievedBrand.Nickname)
	assert.Equal(t, 5, retrievedBrand.Popularity)

	// Test getter methods
	assert.Equal(t, brand.ID, retrievedBrand.GetID())
	assert.Equal(t, "Test Brand Model", retrievedBrand.GetName())
	assert.Equal(t, "Test", retrievedBrand.GetNickname())

	// Test setter methods
	retrievedBrand.SetName("Updated Name")
	retrievedBrand.SetNickname("Updated")
	retrievedBrand.SetPopularity(10)

	assert.Equal(t, "Updated Name", retrievedBrand.Name)
	assert.Equal(t, "Updated", retrievedBrand.Nickname)
	assert.Equal(t, 10, retrievedBrand.Popularity)

	// Test updating the brand
	result = db.Save(&retrievedBrand)
	assert.NoError(t, result.Error)

	// Verify update
	var updatedBrand models.Brand
	result = db.First(&updatedBrand, brand.ID)
	assert.NoError(t, result.Error)
	assert.Equal(t, "Updated Name", updatedBrand.Name)
	assert.Equal(t, "Updated", updatedBrand.Nickname)
	assert.Equal(t, 10, updatedBrand.Popularity)

	// Test deleting the brand
	result = db.Delete(&updatedBrand)
	assert.NoError(t, result.Error)

	// Verify deletion
	result = db.First(&models.Brand{}, brand.ID)
	assert.Error(t, result.Error)
	assert.True(t, result.Error.Error() == "record not found")
}
