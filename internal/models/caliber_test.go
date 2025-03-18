package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCaliberModel(t *testing.T) {
	// Get a shared database instance for testing
	db := GetTestDB()

	// Clear any existing test calibers
	db.Exec("DELETE FROM calibers WHERE caliber LIKE 'Test Custom%'")

	// Test creating a caliber
	caliber := Caliber{
		Caliber:    "Test Custom 9mm",
		Nickname:   "Test Custom Nine",
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
	assert.Equal(t, "Test Custom 9mm", retrievedCaliber.Caliber)
	assert.Equal(t, "Test Custom Nine", retrievedCaliber.Nickname)
	assert.Equal(t, 10, retrievedCaliber.Popularity)

	// Test updating the caliber
	retrievedCaliber.Caliber = "Test Custom 9x19mm"
	retrievedCaliber.Nickname = "Test Custom Parabellum"
	result = db.Save(&retrievedCaliber)
	assert.NoError(t, result.Error)

	// Verify update
	var updatedCaliber Caliber
	result = db.First(&updatedCaliber, caliber.ID)
	assert.NoError(t, result.Error)
	assert.Equal(t, "Test Custom 9x19mm", updatedCaliber.Caliber)
	assert.Equal(t, "Test Custom Parabellum", updatedCaliber.Nickname)

	// Test deleting the caliber
	result = db.Delete(&updatedCaliber)
	assert.NoError(t, result.Error)

	// Verify deletion
	result = db.First(&Caliber{}, caliber.ID)
	assert.Error(t, result.Error)
	assert.True(t, result.Error.Error() == "record not found")
}
