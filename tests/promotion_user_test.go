package tests

import (
	"context"
	"testing"
	"time"

	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// PromotionUserTestSuite is a test suite for testing promotion application to users
type PromotionUserTestSuite struct {
	suite.Suite
	MockDB        *mocks.MockDB
	TestUser      *database.User
	TestPromotion *models.Promotion
}

// SetupTest sets up the test suite
func (s *PromotionUserTestSuite) SetupTest() {
	s.MockDB = new(mocks.MockDB)

	// Create test user
	s.TestUser = &database.User{
		Model: gorm.Model{ID: 1},
		Email: "test@example.com",
	}

	// Create test promotion
	now := time.Now()
	s.TestPromotion = &models.Promotion{
		Model:         gorm.Model{ID: 1},
		Name:          "Test Promotion",
		Type:          "free_trial",
		Active:        true,
		StartDate:     now.AddDate(0, 0, -1),
		EndDate:       now.AddDate(0, 0, 5),
		BenefitDays:   30,
		DisplayOnHome: true,
		Description:   "Test promotion description",
		Banner:        "/images/test-banner.jpg",
	}
}

// TestApplyPromotionToUser tests the direct application of a promotion to a user
func (s *PromotionUserTestSuite) TestApplyPromotionToUser() {
	// Set up mock for user update
	s.MockDB.On("UpdateUser", mock.Anything, mock.MatchedBy(func(u *database.User) bool {
		// Verify user fields were updated correctly
		expectedExpiry := time.Now().AddDate(0, 0, s.TestPromotion.BenefitDays)
		expiryDiff := u.SubscriptionEndDate.Sub(expectedExpiry)

		// Allow a small time difference
		return u.ID == s.TestUser.ID &&
			u.SubscriptionTier == "promotion" &&
			u.SubscriptionStatus == "active" &&
			u.PromotionID == s.TestPromotion.ID &&
			expiryDiff > -time.Minute && expiryDiff < time.Minute
	})).Return(nil)

	// Create a helper function similar to what we'd have in the auth controller
	applyPromotionToUser := func(user *database.User, promotion *models.Promotion) error {
		user.SubscriptionTier = "promotion"
		user.SubscriptionStatus = "active"
		user.SubscriptionEndDate = time.Now().AddDate(0, 0, promotion.BenefitDays)
		user.PromotionID = promotion.ID

		return s.MockDB.UpdateUser(context.Background(), user)
	}

	// Apply the promotion to the user
	err := applyPromotionToUser(s.TestUser, s.TestPromotion)

	// Assert expectations
	s.NoError(err)
	s.Equal("promotion", s.TestUser.SubscriptionTier)
	s.Equal("active", s.TestUser.SubscriptionStatus)
	s.Equal(s.TestPromotion.ID, s.TestUser.PromotionID)

	// Verify the mock was called
	s.MockDB.AssertExpectations(s.T())
}

// TestPromotionUserSuite runs the test suite
func TestPromotionUserSuite(t *testing.T) {
	suite.Run(t, new(PromotionUserTestSuite))
}
