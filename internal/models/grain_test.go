package models_test

import (
	"testing"

	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/testutils"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// TestCreateGrain tests the CreateGrain function
func TestCreateGrain(t *testing.T) {
	db := testutils.SharedTestService().GetDB()

	grain := models.Grain{
		Weight:     999, // Using an uncommon weight for testing
		Popularity: 50,
	}

	defer func() {
		var createdGrain models.Grain
		if err := db.Where("weight = ?", grain.Weight).First(&createdGrain).Error; err == nil {
			db.Unscoped().Delete(&createdGrain)
		}
	}()

	err := models.CreateGrain(db, &grain)
	assert.NoError(t, err)
	assert.Greater(t, grain.ID, uint(0))

	var retrievedGrain models.Grain
	err = db.First(&retrievedGrain, grain.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, 999, retrievedGrain.Weight)
	assert.Equal(t, 50, retrievedGrain.Popularity)
}

// TestFindAllGrains tests the FindAllGrains function
func TestFindAllGrains(t *testing.T) {
	db := testutils.SharedTestService().GetDB()

	grain1 := models.Grain{Weight: 111, Popularity: 100}
	grain2 := models.Grain{Weight: 222, Popularity: 95}
	grain3 := models.Grain{Weight: 333, Popularity: 95}
	models.CreateGrain(db, &grain1)
	models.CreateGrain(db, &grain2)
	models.CreateGrain(db, &grain3)

	defer func() {
		db.Unscoped().Delete(&grain1)
		db.Unscoped().Delete(&grain2)
		db.Unscoped().Delete(&grain3)
	}()

	grains, err := models.FindAllGrains(db)
	assert.NoError(t, err)

	found111 := false
	found222 := false
	found333 := false
	grain222Index, grain333Index := -1, -1

	for i, g := range grains {
		if g.ID == grain1.ID {
			found111 = true
		} else if g.ID == grain2.ID {
			found222 = true
			grain222Index = i
		} else if g.ID == grain3.ID {
			found333 = true
			grain333Index = i
		}
	}

	assert.True(t, found111, "Weight 111 not found")
	assert.True(t, found222, "Weight 222 not found")
	assert.True(t, found333, "Weight 333 not found")

	// Same popularity, so the order is determined by Weight
	assert.Less(t, grain222Index, grain333Index, "Weight 222 should come before Weight 333 numerically")
}

// TestFindGrainByID tests the FindGrainByID function
func TestFindGrainByID(t *testing.T) {
	db := testutils.SharedTestService().GetDB()

	grain := models.Grain{Weight: 444, Popularity: 60}
	models.CreateGrain(db, &grain)
	defer db.Unscoped().Delete(&grain)

	foundGrain, err := models.FindGrainByID(db, grain.ID)
	assert.NoError(t, err)
	assert.NotNil(t, foundGrain)
	assert.Equal(t, 444, foundGrain.Weight)
	assert.Equal(t, 60, foundGrain.Popularity)

	_, err = models.FindGrainByID(db, 999999)
	assert.Error(t, err)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

// TestFindGrainByWeight tests the FindGrainByWeight function
func TestFindGrainByWeight(t *testing.T) {
	db := testutils.SharedTestService().GetDB()

	grain := models.Grain{Weight: 555, Popularity: 35}
	models.CreateGrain(db, &grain)
	defer db.Unscoped().Delete(&grain)

	foundGrain, err := models.FindGrainByWeight(db, 555)
	assert.NoError(t, err)
	assert.NotNil(t, foundGrain)
	assert.Equal(t, grain.ID, foundGrain.ID)
	assert.Equal(t, 35, foundGrain.Popularity)

	_, err = models.FindGrainByWeight(db, 999999)
	assert.Error(t, err)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

// TestUpdateGrain tests the UpdateGrain function
func TestUpdateGrain(t *testing.T) {
	db := testutils.SharedTestService().GetDB()

	grain := models.Grain{Weight: 666, Popularity: 10}
	models.CreateGrain(db, &grain)
	defer db.Unscoped().Delete(&grain)

	grain.Weight = 667
	grain.Popularity = 15
	err := models.UpdateGrain(db, &grain)
	assert.NoError(t, err)

	updatedGrain, _ := models.FindGrainByID(db, grain.ID)
	assert.Equal(t, 667, updatedGrain.Weight)
	assert.Equal(t, 15, updatedGrain.Popularity)
}

// TestDeleteGrain tests the DeleteGrain function
func TestDeleteGrain(t *testing.T) {
	db := testutils.SharedTestService().GetDB()

	grain := models.Grain{Weight: 777, Popularity: 5}
	models.CreateGrain(db, &grain)

	err := models.DeleteGrain(db, grain.ID)
	assert.NoError(t, err)

	_, err = models.FindGrainByID(db, grain.ID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
