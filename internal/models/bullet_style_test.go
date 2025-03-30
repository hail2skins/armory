package models_test

import (
	"testing"

	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/testutils"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// TestCreateBulletStyle tests the CreateBulletStyle function
func TestCreateBulletStyle(t *testing.T) {
	db := testutils.SharedTestService().GetDB()

	bulletStyle := models.BulletStyle{
		Type:       "Test FMJ Create",
		Nickname:   "TFMJ",
		Popularity: 50,
	}

	defer func() {
		var createdBulletStyle models.BulletStyle
		if err := db.Where("type = ?", bulletStyle.Type).First(&createdBulletStyle).Error; err == nil {
			db.Unscoped().Delete(&createdBulletStyle)
		}
	}()

	err := models.CreateBulletStyle(db, &bulletStyle)
	assert.NoError(t, err)
	assert.Greater(t, bulletStyle.ID, uint(0))

	var retrievedBulletStyle models.BulletStyle
	err = db.First(&retrievedBulletStyle, bulletStyle.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "Test FMJ Create", retrievedBulletStyle.Type)
	assert.Equal(t, "TFMJ", retrievedBulletStyle.Nickname)
	assert.Equal(t, 50, retrievedBulletStyle.Popularity)
}

// TestFindAllBulletStyles tests the FindAllBulletStyles function
func TestFindAllBulletStyles(t *testing.T) {
	db := testutils.SharedTestService().GetDB()

	bulletStyle1 := models.BulletStyle{Type: "FindAll_FMJ", Nickname: "FMJ", Popularity: 100}
	bulletStyle2 := models.BulletStyle{Type: "FindAll_JHP", Nickname: "JHP", Popularity: 95}
	bulletStyle3 := models.BulletStyle{Type: "FindAll_SP", Nickname: "SP", Popularity: 95}
	models.CreateBulletStyle(db, &bulletStyle1)
	models.CreateBulletStyle(db, &bulletStyle2)
	models.CreateBulletStyle(db, &bulletStyle3)

	defer func() {
		db.Unscoped().Delete(&bulletStyle1)
		db.Unscoped().Delete(&bulletStyle2)
		db.Unscoped().Delete(&bulletStyle3)
	}()

	bulletStyles, err := models.FindAllBulletStyles(db)
	assert.NoError(t, err)

	foundFMJ := false
	foundJHP := false
	foundSP := false
	jhpIndex, spIndex := -1, -1

	for i, bs := range bulletStyles {
		if bs.ID == bulletStyle1.ID {
			foundFMJ = true
		} else if bs.ID == bulletStyle2.ID {
			foundJHP = true
			jhpIndex = i
		} else if bs.ID == bulletStyle3.ID {
			foundSP = true
			spIndex = i
		}
	}

	assert.True(t, foundFMJ, "FindAll_FMJ not found")
	assert.True(t, foundJHP, "FindAll_JHP not found")
	assert.True(t, foundSP, "FindAll_SP not found")

	// Same popularity, so the order is determined by Type
	assert.Less(t, jhpIndex, spIndex, "JHP should come before SP alphabetically")
}

// TestFindBulletStyleByID tests the FindBulletStyleByID function
func TestFindBulletStyleByID(t *testing.T) {
	db := testutils.SharedTestService().GetDB()

	bulletStyle := models.BulletStyle{Type: "FindByID_HPBT", Nickname: "HPBT", Popularity: 60}
	models.CreateBulletStyle(db, &bulletStyle)
	defer db.Unscoped().Delete(&bulletStyle)

	foundBulletStyle, err := models.FindBulletStyleByID(db, bulletStyle.ID)
	assert.NoError(t, err)
	assert.NotNil(t, foundBulletStyle)
	assert.Equal(t, "FindByID_HPBT", foundBulletStyle.Type)
	assert.Equal(t, "HPBT", foundBulletStyle.Nickname)

	_, err = models.FindBulletStyleByID(db, 999999)
	assert.Error(t, err)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

// TestFindBulletStyleByType tests the FindBulletStyleByType function
func TestFindBulletStyleByType(t *testing.T) {
	db := testutils.SharedTestService().GetDB()

	bulletStyle := models.BulletStyle{Type: "FindByType_Frangible", Nickname: "Frang", Popularity: 35}
	models.CreateBulletStyle(db, &bulletStyle)
	defer db.Unscoped().Delete(&bulletStyle)

	foundBulletStyle, err := models.FindBulletStyleByType(db, "FindByType_Frangible")
	assert.NoError(t, err)
	assert.NotNil(t, foundBulletStyle)
	assert.Equal(t, bulletStyle.ID, foundBulletStyle.ID)
	assert.Equal(t, "Frang", foundBulletStyle.Nickname)
	assert.Equal(t, 35, foundBulletStyle.Popularity)

	_, err = models.FindBulletStyleByType(db, "NonExistentType")
	assert.Error(t, err)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

// TestUpdateBulletStyle tests the UpdateBulletStyle function
func TestUpdateBulletStyle(t *testing.T) {
	db := testutils.SharedTestService().GetDB()

	bulletStyle := models.BulletStyle{Type: "Update_OriginalType", Nickname: "Original", Popularity: 10}
	models.CreateBulletStyle(db, &bulletStyle)
	defer db.Unscoped().Delete(&bulletStyle)

	bulletStyle.Type = "Update_UpdatedType"
	bulletStyle.Nickname = "Updated"
	bulletStyle.Popularity = 15
	err := models.UpdateBulletStyle(db, &bulletStyle)
	assert.NoError(t, err)

	updatedBulletStyle, _ := models.FindBulletStyleByID(db, bulletStyle.ID)
	assert.Equal(t, "Update_UpdatedType", updatedBulletStyle.Type)
	assert.Equal(t, "Updated", updatedBulletStyle.Nickname)
	assert.Equal(t, 15, updatedBulletStyle.Popularity)
}

// TestDeleteBulletStyle tests the DeleteBulletStyle function
func TestDeleteBulletStyle(t *testing.T) {
	db := testutils.SharedTestService().GetDB()

	bulletStyle := models.BulletStyle{Type: "Delete_ToDelete", Nickname: "Delete", Popularity: 5}
	models.CreateBulletStyle(db, &bulletStyle)

	err := models.DeleteBulletStyle(db, bulletStyle.ID)
	assert.NoError(t, err)

	_, err = models.FindBulletStyleByID(db, bulletStyle.ID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
