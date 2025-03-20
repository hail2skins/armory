package tests

import (
	"testing"
	"time"

	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/services"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// PromotionServiceTestSuite is a test suite for the PromotionService
type PromotionServiceTestSuite struct {
	suite.Suite
	MockDB *mocks.MockDB
}

// SetupTest sets up the test suite
func (s *PromotionServiceTestSuite) SetupTest() {
	s.MockDB = new(mocks.MockDB)
}

// TestGetActivePromotionsWhenNoneActive tests the case when no promotions are active
func (s *PromotionServiceTestSuite) TestGetActivePromotionsWhenNoneActive() {
	// Set up the mock DB to return empty promotions
	s.MockDB.On("FindActivePromotions").Return([]models.Promotion{}, nil)

	// Create the service
	service := services.NewPromotionService(s.MockDB)

	// Get active promotions
	promotions, err := service.GetActivePromotions()

	// Assert expectations
	s.NoError(err)
	s.Empty(promotions)
	s.MockDB.AssertExpectations(s.T())
}

// TestGetActivePromotionsSingleActive tests when there is a single active promotion
func (s *PromotionServiceTestSuite) TestGetActivePromotionsSingleActive() {
	// Create a mock active promotion
	now := time.Now()
	activePromotion := models.Promotion{
		Model:         gorm.Model{ID: 1},
		Name:          "Test Promotion",
		Type:          "free_trial",
		Active:        true,
		StartDate:     now.AddDate(0, 0, -1), // Started yesterday
		EndDate:       now.AddDate(0, 0, 5),  // Ends in 5 days
		BenefitDays:   30,
		DisplayOnHome: true,
		Description:   "Test promotion description",
		Banner:        "/images/test-banner.jpg",
	}

	// Set up the mock DB to return our active promotion
	s.MockDB.On("FindActivePromotions").Return([]models.Promotion{activePromotion}, nil)

	// Create the service
	service := services.NewPromotionService(s.MockDB)

	// Get active promotions
	promotions, err := service.GetActivePromotions()

	// Assert expectations
	s.NoError(err)
	s.Len(promotions, 1)
	s.Equal(uint(1), promotions[0].ID)
	s.Equal("Test Promotion", promotions[0].Name)
	s.MockDB.AssertExpectations(s.T())
}

// TestGetActivePromotionsMultipleActive tests when there are multiple active promotions
func (s *PromotionServiceTestSuite) TestGetActivePromotionsMultipleActive() {
	// Create multiple mock active promotions
	now := time.Now()
	activePromotions := []models.Promotion{
		{
			Model:         gorm.Model{ID: 1},
			Name:          "First Promotion",
			Type:          "free_trial",
			Active:        true,
			StartDate:     now.AddDate(0, 0, -2), // Started 2 days ago
			EndDate:       now.AddDate(0, 0, 5),  // Ends in 5 days
			BenefitDays:   30,
			DisplayOnHome: true,
			Description:   "First promotion description",
			Banner:        "/images/first-banner.jpg",
		},
		{
			Model:         gorm.Model{ID: 2},
			Name:          "Second Promotion",
			Type:          "discount",
			Active:        true,
			StartDate:     now.AddDate(0, 0, -1), // Started yesterday
			EndDate:       now.AddDate(0, 0, 10), // Ends in 10 days
			BenefitDays:   45,
			DisplayOnHome: false,
			Description:   "Second promotion description",
			Banner:        "/images/second-banner.jpg",
		},
	}

	// Set up the mock DB to return our active promotions
	s.MockDB.On("FindActivePromotions").Return(activePromotions, nil)

	// Create the service
	service := services.NewPromotionService(s.MockDB)

	// Get active promotions
	promotions, err := service.GetActivePromotions()

	// Assert expectations
	s.NoError(err)
	s.Len(promotions, 2)
	s.MockDB.AssertExpectations(s.T())
}

// TestGetBestActivePromotion tests prioritization logic for multiple active promotions
func (s *PromotionServiceTestSuite) TestGetBestActivePromotion() {
	// Create multiple mock active promotions
	now := time.Now()
	activePromotions := []models.Promotion{
		{
			Model:         gorm.Model{ID: 1},
			Name:          "Less Beneficial Promotion",
			Type:          "free_trial",
			Active:        true,
			StartDate:     now.AddDate(0, 0, -2), // Started 2 days ago
			EndDate:       now.AddDate(0, 0, 5),  // Ends in 5 days
			BenefitDays:   30,
			DisplayOnHome: true,
			Description:   "First promotion description",
			Banner:        "/images/first-banner.jpg",
		},
		{
			Model:         gorm.Model{ID: 2},
			Name:          "More Beneficial Promotion",
			Type:          "discount",
			Active:        true,
			StartDate:     now.AddDate(0, 0, -1), // Started yesterday
			EndDate:       now.AddDate(0, 0, 10), // Ends in 10 days
			BenefitDays:   45,                    // More benefit days
			DisplayOnHome: false,
			Description:   "Second promotion description",
			Banner:        "/images/second-banner.jpg",
		},
	}

	// Set up the mock DB to return our active promotions
	s.MockDB.On("FindActivePromotions").Return(activePromotions, nil)

	// Create the service
	service := services.NewPromotionService(s.MockDB)

	// Get the best active promotion
	bestPromotion, err := service.GetBestActivePromotion()

	// Assert expectations
	s.NoError(err)
	s.NotNil(bestPromotion)
	s.Equal(uint(2), bestPromotion.ID) // Should select the one with more benefit days
	s.Equal(45, bestPromotion.BenefitDays)
	s.MockDB.AssertExpectations(s.T())
}

// TestGetActivePromotionsDBError tests handling of database errors
func (s *PromotionServiceTestSuite) TestGetActivePromotionsDBError() {
	// Set up the mock DB to return an error
	dbError := gorm.ErrInvalidDB
	s.MockDB.On("FindActivePromotions").Return([]models.Promotion{}, dbError)

	// Create the service
	service := services.NewPromotionService(s.MockDB)

	// Get active promotions
	promotions, err := service.GetActivePromotions()

	// Assert expectations
	s.Error(err)
	s.Equal(dbError, err)
	s.Empty(promotions)
	s.MockDB.AssertExpectations(s.T())
}

// TestPromotionServiceSuite runs the test suite
func TestPromotionServiceSuite(t *testing.T) {
	suite.Run(t, new(PromotionServiceTestSuite))
}
