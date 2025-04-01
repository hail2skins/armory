package tests

import (
	"testing"
	"time"

	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromotionSubscriptionExpiration(t *testing.T) {
	// Setup test database
	db := testutils.NewTestDB()
	defer db.Close()

	// Create a test user
	user := &database.User{
		Email:    "promotion_expiration@test.com",
		Password: "$2a$10$rsCOHaTqI2/rKZ7ADj/JseKuuKp1/L7Tcn08KgRmfQbvd9tZZjFGq", // bcrypted "password"
		Verified: true,
	}
	err := db.DB.Create(user).Error
	require.NoError(t, err, "Failed to create test user")

	// Create a promotion
	promotion := models.Promotion{
		Name:        "Test Expiration Promotion",
		Type:        "free_trial",
		Active:      true,
		StartDate:   time.Now().AddDate(0, 0, -10), // 10 days ago
		EndDate:     time.Now().AddDate(0, 0, 10),  // 10 days from now
		BenefitDays: 5,                             // 5-day trial
	}
	err = db.DB.Create(&promotion).Error
	require.NoError(t, err, "Failed to create test promotion")

	// Apply promotion to user with an already-expired subscription
	user.SubscriptionTier = "promotion"
	user.SubscriptionStatus = "active"                      // Initially active
	user.SubscriptionEndDate = time.Now().AddDate(0, 0, -2) // Expired 2 days ago
	user.PromotionID = promotion.ID

	err = db.DB.Save(user).Error
	require.NoError(t, err, "Failed to update user with promotion")

	// Create the database service using the test database
	dbService := testutils.NewTestService(db.DB)

	// Call the CheckExpiredPromotionSubscription function
	updated, err := dbService.CheckExpiredPromotionSubscription(user)
	require.NoError(t, err, "Error checking promotion expiration")
	assert.True(t, updated, "User subscription should be updated")

	// Refresh user from database
	err = db.DB.First(user, user.ID).Error
	require.NoError(t, err, "Failed to refresh user from database")

	// Verify subscription status is now expired
	assert.Equal(t, "expired", user.SubscriptionStatus, "Subscription status should be 'expired'")
	assert.Equal(t, "free", user.SubscriptionTier, "Subscription tier should be reset to 'free'")
	assert.True(t, user.SubscriptionEndDate.IsZero(), "Subscription end date should be cleared")
}

func TestActivePromotionSubscription(t *testing.T) {
	// Setup test database
	db := testutils.NewTestDB()
	defer db.Close()

	// Create a test user
	user := &database.User{
		Email:    "active_promotion@test.com",
		Password: "$2a$10$rsCOHaTqI2/rKZ7ADj/JseKuuKp1/L7Tcn08KgRmfQbvd9tZZjFGq", // bcrypted "password"
		Verified: true,
	}
	err := db.DB.Create(user).Error
	require.NoError(t, err, "Failed to create test user")

	// Create a promotion
	promotion := models.Promotion{
		Name:        "Test Active Promotion",
		Type:        "free_trial",
		Active:      true,
		StartDate:   time.Now().AddDate(0, 0, -10), // 10 days ago
		EndDate:     time.Now().AddDate(0, 0, 10),  // 10 days from now
		BenefitDays: 30,                            // 30-day trial
	}
	err = db.DB.Create(&promotion).Error
	require.NoError(t, err, "Failed to create test promotion")

	// Apply promotion to user with a still-active subscription
	user.SubscriptionTier = "promotion"
	user.SubscriptionStatus = "active"
	user.SubscriptionEndDate = time.Now().AddDate(0, 0, 5) // Expires 5 days from now
	user.PromotionID = promotion.ID

	err = db.DB.Save(user).Error
	require.NoError(t, err, "Failed to update user with promotion")

	// Create the database service using the test database
	dbService := testutils.NewTestService(db.DB)

	// Call the CheckExpiredPromotionSubscription function
	updated, err := dbService.CheckExpiredPromotionSubscription(user)
	require.NoError(t, err, "Error checking promotion expiration")
	assert.False(t, updated, "User subscription should not be updated")

	// Refresh user from database
	err = db.DB.First(user, user.ID).Error
	require.NoError(t, err, "Failed to refresh user from database")

	// Verify subscription status is still active
	assert.Equal(t, "active", user.SubscriptionStatus, "Subscription status should still be 'active'")
	assert.Equal(t, "promotion", user.SubscriptionTier, "Subscription tier should still be 'promotion'")
	assert.False(t, user.SubscriptionEndDate.IsZero(), "Subscription end date should be populated")
}
