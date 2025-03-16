package testutils

import (
	"testing"

	"github.com/hail2skins/armory/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestNewTestDB(t *testing.T) {
	// Create a new test database
	testDB := NewTestDB()
	defer testDB.Close()

	// Verify that the database was created
	assert.NotNil(t, testDB.DB)

	// Test creating a gun
	gun := models.Gun{
		Name:           "Test Gun",
		Description:    "A test gun",
		SerialNumber:   "123456",
		WeaponTypeID:   1,
		CaliberID:      1,
		ManufacturerID: 1,
		OwnerID:        1,
	}

	// Save to database
	result := testDB.DB.Create(&gun)
	assert.NoError(t, result.Error)
	assert.NotZero(t, gun.ID)

	// Test retrieving the gun
	var retrievedGun models.Gun
	result = testDB.DB.First(&retrievedGun, gun.ID)
	assert.NoError(t, result.Error)
	assert.Equal(t, "Test Gun", retrievedGun.Name)
	assert.Equal(t, "A test gun", retrievedGun.Description)
	assert.Equal(t, "123456", retrievedGun.SerialNumber)
	assert.Equal(t, uint(1), retrievedGun.OwnerID)
}
