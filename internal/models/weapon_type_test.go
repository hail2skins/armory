package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWeaponTypeModel(t *testing.T) {
	// Get a shared database instance for testing
	db := GetTestDB()

	// Clear any existing test weapon types with this specific name
	db.Exec("DELETE FROM weapon_types WHERE type LIKE 'Test Custom%'")

	// Test creating a weapon type
	weaponType := WeaponType{
		Type:       "Test Custom Pistol",
		Nickname:   "Test Custom Handgun",
		Popularity: 15,
	}

	// Save to database
	err := CreateWeaponType(db, &weaponType)
	assert.NoError(t, err)
	assert.NotZero(t, weaponType.ID)

	// Test retrieving the weapon type
	retrievedWeaponType, err := FindWeaponTypeByID(db, weaponType.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Test Custom Pistol", retrievedWeaponType.Type)
	assert.Equal(t, "Test Custom Handgun", retrievedWeaponType.Nickname)
	assert.Equal(t, 15, retrievedWeaponType.Popularity)

	// Test retrieving all weapon types (this will return more than just our test one)
	weaponTypes, err := FindAllWeaponTypes(db)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(weaponTypes), 1)

	// Find our test weapon type in the list
	var found bool
	for _, wt := range weaponTypes {
		if wt.ID == weaponType.ID {
			assert.Equal(t, "Test Custom Pistol", wt.Type)
			found = true
			break
		}
	}
	assert.True(t, found, "Our test weapon type should be in the list")

	// Test updating the weapon type
	retrievedWeaponType.Type = "Test Custom Semi-Auto Pistol"
	retrievedWeaponType.Nickname = "Test Custom Semi-Auto"
	err = UpdateWeaponType(db, retrievedWeaponType)
	assert.NoError(t, err)

	// Verify update
	updatedWeaponType, err := FindWeaponTypeByID(db, weaponType.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Test Custom Semi-Auto Pistol", updatedWeaponType.Type)
	assert.Equal(t, "Test Custom Semi-Auto", updatedWeaponType.Nickname)

	// Test deleting the weapon type
	err = DeleteWeaponType(db, weaponType.ID)
	assert.NoError(t, err)

	// Verify deletion
	_, err = FindWeaponTypeByID(db, weaponType.ID)
	assert.Error(t, err)
	assert.True(t, err.Error() == "record not found")

	// Test table name
	assert.Equal(t, "weapon_types", WeaponType{}.TableName())
}
