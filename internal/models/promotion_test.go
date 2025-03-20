package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPromotionModel(t *testing.T) {
	// Get a shared database instance for testing
	db := GetTestDB()

	// Clear any existing test promotions
	db.Exec("DELETE FROM promotions WHERE name LIKE 'Test Promotion%'")

	// Generate dates for testing
	now := time.Now()
	startDate := now
	endDate := now.AddDate(0, 1, 0) // One month from now

	// Test creating a promotion
	promotion := Promotion{
		Name:          "Test Promotion Free Trial",
		Type:          "free_trial",
		Active:        true,
		StartDate:     startDate,
		EndDate:       endDate,
		BenefitDays:   30,
		BenefitTier:   "monthly",
		DisplayOnHome: true,
		Description:   "Test promotion description",
		Banner:        "/images/banners/test-promotion.jpg",
	}

	// Save to database
	result := db.Create(&promotion)
	assert.NoError(t, result.Error)
	assert.NotZero(t, promotion.ID)

	// Test retrieving the promotion
	var retrievedPromotion Promotion
	result = db.First(&retrievedPromotion, promotion.ID)
	assert.NoError(t, result.Error)
	assert.Equal(t, "Test Promotion Free Trial", retrievedPromotion.Name)
	assert.Equal(t, "free_trial", retrievedPromotion.Type)
	assert.True(t, retrievedPromotion.Active)
	assert.Equal(t, startDate.UTC().Truncate(time.Second), retrievedPromotion.StartDate.UTC().Truncate(time.Second))
	assert.Equal(t, endDate.UTC().Truncate(time.Second), retrievedPromotion.EndDate.UTC().Truncate(time.Second))
	assert.Equal(t, 30, retrievedPromotion.BenefitDays)
	assert.Equal(t, "monthly", retrievedPromotion.BenefitTier)
	assert.True(t, retrievedPromotion.DisplayOnHome)
	assert.Equal(t, "Test promotion description", retrievedPromotion.Description)
	assert.Equal(t, "/images/banners/test-promotion.jpg", retrievedPromotion.Banner)

	// Test updating the promotion
	retrievedPromotion.Name = "Test Promotion Updated"
	retrievedPromotion.Type = "discount"
	retrievedPromotion.Active = false
	retrievedPromotion.BenefitDays = 60
	retrievedPromotion.BenefitTier = "yearly"
	retrievedPromotion.DisplayOnHome = false
	retrievedPromotion.Description = "Updated description"

	result = db.Save(&retrievedPromotion)
	assert.NoError(t, result.Error)

	// Verify update
	var updatedPromotion Promotion
	result = db.First(&updatedPromotion, promotion.ID)
	assert.NoError(t, result.Error)
	assert.Equal(t, "Test Promotion Updated", updatedPromotion.Name)
	assert.Equal(t, "discount", updatedPromotion.Type)
	assert.False(t, updatedPromotion.Active)
	assert.Equal(t, 60, updatedPromotion.BenefitDays)
	assert.Equal(t, "yearly", updatedPromotion.BenefitTier)
	assert.False(t, updatedPromotion.DisplayOnHome)
	assert.Equal(t, "Updated description", updatedPromotion.Description)

	// Test finding active promotions
	activePromotion := Promotion{
		Name:          "Test Promotion Active",
		Type:          "discount",
		Active:        true,
		StartDate:     startDate,
		EndDate:       endDate,
		BenefitDays:   14,
		BenefitTier:   "monthly",
		DisplayOnHome: true,
		Description:   "Active promotion",
	}

	result = db.Create(&activePromotion)
	assert.NoError(t, result.Error)

	var activePromotions []Promotion
	result = db.Where("active = ?", true).Find(&activePromotions)
	assert.NoError(t, result.Error)
	assert.NotEmpty(t, activePromotions)
	assert.GreaterOrEqual(t, len(activePromotions), 1)

	// Test deleting the promotions
	result = db.Delete(&updatedPromotion)
	assert.NoError(t, result.Error)
	result = db.Delete(&activePromotion)
	assert.NoError(t, result.Error)

	// Verify deletion
	result = db.First(&Promotion{}, promotion.ID)
	assert.Error(t, result.Error)
	assert.True(t, result.Error.Error() == "record not found")
}
