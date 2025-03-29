package models_test

import (
	"testing"

	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/testutils"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// TestCreateCasing tests the CreateCasing function
func TestCreateCasing(t *testing.T) {
	db := testutils.SharedTestService().GetDB()

	casing := models.Casing{
		Type:       "Test Brass Create",
		Popularity: 50,
	}

	defer func() {
		var createdCasing models.Casing
		if err := db.Where("type = ?", casing.Type).First(&createdCasing).Error; err == nil {
			db.Unscoped().Delete(&createdCasing)
		}
	}()

	err := models.CreateCasing(db, &casing)
	assert.NoError(t, err)
	assert.Greater(t, casing.ID, uint(0))

	var retrievedCasing models.Casing
	err = db.First(&retrievedCasing, casing.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "Test Brass Create", retrievedCasing.Type)
	assert.Equal(t, 50, retrievedCasing.Popularity)
}

// TestFindAllCasings tests the FindAllCasings function
func TestFindAllCasings(t *testing.T) {
	db := testutils.SharedTestService().GetDB()

	casing1 := models.Casing{Type: "FindAll_Brass", Popularity: 100}
	casing2 := models.Casing{Type: "FindAll_Steel", Popularity: 80}
	casing3 := models.Casing{Type: "FindAll_Aluminum", Popularity: 80}
	models.CreateCasing(db, &casing1)
	models.CreateCasing(db, &casing2)
	models.CreateCasing(db, &casing3)

	defer func() {
		db.Unscoped().Delete(&casing1)
		db.Unscoped().Delete(&casing2)
		db.Unscoped().Delete(&casing3)
	}()

	casings, err := models.FindAllCasings(db)
	assert.NoError(t, err)

	foundBrass := false
	foundSteel := false
	foundAluminum := false
	brassIndex, steelIndex, aluminumIndex := -1, -1, -1

	for i, c := range casings {
		if c.ID == casing1.ID {
			foundBrass = true
			brassIndex = i
		} else if c.ID == casing2.ID {
			foundSteel = true
			steelIndex = i
		} else if c.ID == casing3.ID {
			foundAluminum = true
			aluminumIndex = i
		}
	}

	assert.True(t, foundBrass, "FindAll_Brass not found")
	assert.True(t, foundSteel, "FindAll_Steel not found")
	assert.True(t, foundAluminum, "FindAll_Aluminum not found")

	assert.Less(t, brassIndex, aluminumIndex, "Brass should come before Aluminum")
	assert.Less(t, brassIndex, steelIndex, "Brass should come before Steel")
	assert.Less(t, aluminumIndex, steelIndex, "Aluminum should come before Steel alphabetically")
}

// TestFindCasingByID tests the FindCasingByID function
func TestFindCasingByID(t *testing.T) {
	db := testutils.SharedTestService().GetDB()

	casing := models.Casing{Type: "FindByID_Nickel", Popularity: 70}
	models.CreateCasing(db, &casing)
	defer db.Unscoped().Delete(&casing)

	foundCasing, err := models.FindCasingByID(db, casing.ID)
	assert.NoError(t, err)
	assert.NotNil(t, foundCasing)
	assert.Equal(t, "FindByID_Nickel", foundCasing.Type)

	_, err = models.FindCasingByID(db, 999999)
	assert.Error(t, err)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

// TestFindCasingByType tests the FindCasingByType function
func TestFindCasingByType(t *testing.T) {
	db := testutils.SharedTestService().GetDB()

	casing := models.Casing{Type: "FindByType_Polymer", Popularity: 20}
	models.CreateCasing(db, &casing)
	defer db.Unscoped().Delete(&casing)

	foundCasing, err := models.FindCasingByType(db, "FindByType_Polymer")
	assert.NoError(t, err)
	assert.NotNil(t, foundCasing)
	assert.Equal(t, casing.ID, foundCasing.ID)
	assert.Equal(t, 20, foundCasing.Popularity)

	_, err = models.FindCasingByType(db, "NonExistentType")
	assert.Error(t, err)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

// TestUpdateCasing tests the UpdateCasing function
func TestUpdateCasing(t *testing.T) {
	db := testutils.SharedTestService().GetDB()

	casing := models.Casing{Type: "Update_OriginalType", Popularity: 10}
	models.CreateCasing(db, &casing)
	defer db.Unscoped().Delete(&casing)

	casing.Type = "Update_UpdatedType"
	casing.Popularity = 15
	err := models.UpdateCasing(db, &casing)
	assert.NoError(t, err)

	updatedCasing, _ := models.FindCasingByID(db, casing.ID)
	assert.Equal(t, "Update_UpdatedType", updatedCasing.Type)
	assert.Equal(t, 15, updatedCasing.Popularity)
}

// TestDeleteCasing tests the DeleteCasing function
func TestDeleteCasing(t *testing.T) {
	db := testutils.SharedTestService().GetDB()

	casing := models.Casing{Type: "Delete_ToDelete", Popularity: 5}
	models.CreateCasing(db, &casing)

	err := models.DeleteCasing(db, casing.ID)
	assert.NoError(t, err)

	_, err = models.FindCasingByID(db, casing.ID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
