package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCaliberModel(t *testing.T) {
	// Setup in-memory database for testing
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	// Auto migrate the schema
	err = db.AutoMigrate(&Caliber{})
	assert.NoError(t, err)

	// Test creating a caliber
	caliber := Caliber{
		Caliber:    "9mm",
		Nickname:   "Nine",
		Popularity: 10,
	}

	// Save to database
	result := db.Create(&caliber)
	assert.NoError(t, result.Error)
	assert.NotZero(t, caliber.ID)

	// Test retrieving the caliber
	var retrievedCaliber Caliber
	result = db.First(&retrievedCaliber, caliber.ID)
	assert.NoError(t, result.Error)
	assert.Equal(t, "9mm", retrievedCaliber.Caliber)
	assert.Equal(t, "Nine", retrievedCaliber.Nickname)
	assert.Equal(t, 10, retrievedCaliber.Popularity)

	// Test updating the caliber
	retrievedCaliber.Caliber = "9x19mm"
	retrievedCaliber.Nickname = "Parabellum"
	result = db.Save(&retrievedCaliber)
	assert.NoError(t, result.Error)

	// Verify update
	var updatedCaliber Caliber
	result = db.First(&updatedCaliber, caliber.ID)
	assert.NoError(t, result.Error)
	assert.Equal(t, "9x19mm", updatedCaliber.Caliber)
	assert.Equal(t, "Parabellum", updatedCaliber.Nickname)

	// Test deleting the caliber
	result = db.Delete(&updatedCaliber)
	assert.NoError(t, result.Error)

	// Verify deletion
	result = db.First(&Caliber{}, caliber.ID)
	assert.Error(t, result.Error)
	assert.True(t, result.Error == gorm.ErrRecordNotFound)
}
