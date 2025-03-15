package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestManufacturerModel(t *testing.T) {
	// Setup in-memory database for testing
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	// Auto migrate the schema
	err = db.AutoMigrate(&Manufacturer{})
	assert.NoError(t, err)

	// Test creating a manufacturer
	manufacturer := Manufacturer{
		Name:       "Test Manufacturer",
		Nickname:   "Test",
		Country:    "Test Country",
		Popularity: 5,
	}

	// Save to database
	result := db.Create(&manufacturer)
	assert.NoError(t, result.Error)
	assert.NotZero(t, manufacturer.ID)

	// Test retrieving the manufacturer
	var retrievedManufacturer Manufacturer
	result = db.First(&retrievedManufacturer, manufacturer.ID)
	assert.NoError(t, result.Error)
	assert.Equal(t, "Test Manufacturer", retrievedManufacturer.Name)
	assert.Equal(t, "Test", retrievedManufacturer.Nickname)
	assert.Equal(t, "Test Country", retrievedManufacturer.Country)
	assert.Equal(t, 5, retrievedManufacturer.Popularity)

	// Test getter methods
	assert.Equal(t, manufacturer.ID, retrievedManufacturer.GetID())
	assert.Equal(t, "Test Manufacturer", retrievedManufacturer.GetName())
	assert.Equal(t, "Test", retrievedManufacturer.GetNickname())
	assert.Equal(t, "Test Country", retrievedManufacturer.GetCountry())

	// Test setter methods
	retrievedManufacturer.SetName("Updated Name")
	retrievedManufacturer.SetNickname("Updated")
	retrievedManufacturer.SetCountry("Updated Country")

	assert.Equal(t, "Updated Name", retrievedManufacturer.Name)
	assert.Equal(t, "Updated", retrievedManufacturer.Nickname)
	assert.Equal(t, "Updated Country", retrievedManufacturer.Country)

	// Test updating the manufacturer
	result = db.Save(&retrievedManufacturer)
	assert.NoError(t, result.Error)

	// Verify update
	var updatedManufacturer Manufacturer
	result = db.First(&updatedManufacturer, manufacturer.ID)
	assert.NoError(t, result.Error)
	assert.Equal(t, "Updated Name", updatedManufacturer.Name)
	assert.Equal(t, "Updated", updatedManufacturer.Nickname)
	assert.Equal(t, "Updated Country", updatedManufacturer.Country)

	// Test deleting the manufacturer
	result = db.Delete(&updatedManufacturer)
	assert.NoError(t, result.Error)

	// Verify deletion
	result = db.First(&Manufacturer{}, manufacturer.ID)
	assert.Error(t, result.Error)
	assert.True(t, result.Error == gorm.ErrRecordNotFound)
}
