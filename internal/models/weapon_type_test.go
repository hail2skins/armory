package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestWeaponTypeModel(t *testing.T) {
	// Setup in-memory database for testing
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	// Auto migrate the schema
	err = db.AutoMigrate(&WeaponType{})
	assert.NoError(t, err)

	// Test creating a weapon type
	weaponType := WeaponType{
		Type:       "Pistol",
		Nickname:   "Handgun",
		Popularity: 15,
	}

	// Save to database
	err = CreateWeaponType(db, &weaponType)
	assert.NoError(t, err)
	assert.NotZero(t, weaponType.ID)

	// Test retrieving the weapon type
	retrievedWeaponType, err := FindWeaponTypeByID(db, weaponType.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Pistol", retrievedWeaponType.Type)
	assert.Equal(t, "Handgun", retrievedWeaponType.Nickname)
	assert.Equal(t, 15, retrievedWeaponType.Popularity)

	// Test retrieving all weapon types
	weaponTypes, err := FindAllWeaponTypes(db)
	assert.NoError(t, err)
	assert.Len(t, weaponTypes, 1)
	assert.Equal(t, "Pistol", weaponTypes[0].Type)

	// Test updating the weapon type
	retrievedWeaponType.Type = "Semi-Auto Pistol"
	retrievedWeaponType.Nickname = "Semi-Auto"
	err = UpdateWeaponType(db, retrievedWeaponType)
	assert.NoError(t, err)

	// Verify update
	updatedWeaponType, err := FindWeaponTypeByID(db, weaponType.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Semi-Auto Pistol", updatedWeaponType.Type)
	assert.Equal(t, "Semi-Auto", updatedWeaponType.Nickname)

	// Test deleting the weapon type
	err = DeleteWeaponType(db, weaponType.ID)
	assert.NoError(t, err)

	// Verify deletion
	_, err = FindWeaponTypeByID(db, weaponType.ID)
	assert.Error(t, err)
	assert.True(t, err == gorm.ErrRecordNotFound)

	// Test table name
	assert.Equal(t, "weapon_types", WeaponType{}.TableName())
}
