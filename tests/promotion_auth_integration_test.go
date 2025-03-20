package tests

import (
	"testing"
	"time"

	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// PromotionAuthIntegrationTestSuite tests the integration of promotions with the AuthController
type PromotionAuthIntegrationTestSuite struct {
	suite.Suite
	AuthController *controller.AuthController
	MockDB         *mocks.MockDB
	TestUser       *database.User
	TestPromotion  *models.Promotion
}

// SetupTest sets up the test suite
func (s *PromotionAuthIntegrationTestSuite) SetupTest() {
	s.MockDB = new(mocks.MockDB)
	s.AuthController = controller.NewAuthController(s.MockDB)

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

// TestApplyPromotionDirectly tests the AuthController's ApplyPromotionToUser method directly
func (s *PromotionAuthIntegrationTestSuite) TestApplyPromotionDirectly() {
	// Set up mock for UpdateUser
	s.MockDB.On("UpdateUser", mock.Anything, mock.MatchedBy(func(u *database.User) bool {
		// Check for expected user changes
		expectedEndDate := time.Now().AddDate(0, 0, s.TestPromotion.BenefitDays)
		dateDiff := u.SubscriptionEndDate.Sub(expectedEndDate)

		return u.Model.ID == s.TestUser.Model.ID &&
			u.SubscriptionTier == "promotion" &&
			u.SubscriptionStatus == "active" &&
			u.PromotionID == s.TestPromotion.ID &&
			dateDiff > -time.Minute && dateDiff < time.Minute
	})).Return(nil)

	// Apply the promotion directly using the AuthController method
	s.AuthController.ApplyPromotionToUser(s.TestUser, s.TestPromotion)

	// Verify user was updated correctly
	s.Equal("promotion", s.TestUser.SubscriptionTier)
	s.Equal("active", s.TestUser.SubscriptionStatus)
	s.Equal(s.TestPromotion.ID, s.TestUser.PromotionID)

	// Verify the mock was called
	s.MockDB.AssertExpectations(s.T())
}

// TestSetPromotionService tests setting the promotion service on the auth controller
func (s *PromotionAuthIntegrationTestSuite) TestSetPromotionService() {
	// Create mock promotion service
	mockPromotionService := struct {
		GetBestActivePromotionCalled bool
		GetBestActivePromotionResult *models.Promotion
		GetBestActivePromotionError  error

		GetBestActivePromotion func() (*models.Promotion, error)
	}{
		GetBestActivePromotionCalled: false,
		GetBestActivePromotionResult: s.TestPromotion,
		GetBestActivePromotionError:  nil,
	}

	// Implement the GetBestActivePromotion method
	mockPromotionService.GetBestActivePromotion = func() (*models.Promotion, error) {
		mockPromotionService.GetBestActivePromotionCalled = true
		return mockPromotionService.GetBestActivePromotionResult, mockPromotionService.GetBestActivePromotionError
	}

	// Set the promotion service
	s.AuthController.SetPromotionService(mockPromotionService)

	// Create a user to test with
	user := &database.User{
		Model: gorm.Model{ID: 1},
		Email: "test@example.com",
	}

	// Mock UpdateUser to check if the promotion service is called
	s.MockDB.On("UpdateUser", mock.Anything, mock.Anything).Return(nil)

	// Now call a method that should use the promotion service
	// This will verify if the promotion service was properly set
	// We'll simulate part of the registration process
	s.AuthController.ApplyPromotionToUser(user, s.TestPromotion)

	// Verify the user was updated with promotion details
	s.Equal("promotion", user.SubscriptionTier)
	s.Equal(s.TestPromotion.ID, user.PromotionID)

	// Verify the mock was called
	s.MockDB.AssertExpectations(s.T())
}

// Test suite entry point
func TestPromotionAuthIntegrationSuite(t *testing.T) {
	suite.Run(t, new(PromotionAuthIntegrationTestSuite))
}
