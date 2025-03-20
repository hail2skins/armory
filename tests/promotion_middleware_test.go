package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/middleware"
	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/services"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// PromotionMiddlewareTestSuite is a test suite for the promotion middleware
type PromotionMiddlewareTestSuite struct {
	suite.Suite
	MockDB           *mocks.MockDB
	Router           *gin.Engine
	PromotionService *services.PromotionService
	MockPromotion    *models.Promotion
	recorder         *httptest.ResponseRecorder
}

// SetupTest sets up the test suite
func (s *PromotionMiddlewareTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	s.MockDB = new(mocks.MockDB)
	s.PromotionService = services.NewPromotionService(s.MockDB)
	s.Router = gin.New()
	s.recorder = httptest.NewRecorder()

	// Create a mock promotion
	now := time.Now()
	s.MockPromotion = &models.Promotion{
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
}

// TestMiddlewareWithActivePromotion tests the middleware with an active promotion
func (s *PromotionMiddlewareTestSuite) TestMiddlewareWithActivePromotion() {
	// Set up mock for active promotion
	s.MockDB.On("FindActivePromotions").Return([]models.Promotion{*s.MockPromotion}, nil)

	// Register middleware and a test route
	s.Router.Use(middleware.PromotionBanner(s.PromotionService))
	s.Router.GET("/", func(c *gin.Context) {
		// Check if the promotion was added to the context
		promotionValue, exists := c.Get("active_promotion")
		s.True(exists)

		promotion, ok := promotionValue.(*models.Promotion)
		s.True(ok)
		s.Equal(uint(1), promotion.ID)
		s.Equal("Test Promotion", promotion.Name)

		c.String(http.StatusOK, "OK")
	})

	// Make a request
	req, _ := http.NewRequest("GET", "/", nil)
	s.Router.ServeHTTP(s.recorder, req)

	// Check response
	s.Equal(http.StatusOK, s.recorder.Code)
	s.Equal("OK", s.recorder.Body.String())

	// Verify mock was called
	s.MockDB.AssertExpectations(s.T())
}

// TestMiddlewareWithNoActivePromotion tests the middleware without an active promotion
func (s *PromotionMiddlewareTestSuite) TestMiddlewareWithNoActivePromotion() {
	// Set up mock for no active promotions
	s.MockDB.On("FindActivePromotions").Return([]models.Promotion{}, nil)

	// Register middleware and a test route
	s.Router.Use(middleware.PromotionBanner(s.PromotionService))
	s.Router.GET("/", func(c *gin.Context) {
		// Check that no promotion was added to the context
		_, exists := c.Get("active_promotion")
		s.False(exists)

		c.String(http.StatusOK, "OK")
	})

	// Make a request
	req, _ := http.NewRequest("GET", "/", nil)
	s.Router.ServeHTTP(s.recorder, req)

	// Check response
	s.Equal(http.StatusOK, s.recorder.Code)
	s.Equal("OK", s.recorder.Body.String())

	// Verify mock was called
	s.MockDB.AssertExpectations(s.T())
}

// TestMiddlewareWithDatabaseError tests the middleware with a database error
func (s *PromotionMiddlewareTestSuite) TestMiddlewareWithDatabaseError() {
	// Set up mock for database error
	dbError := gorm.ErrInvalidDB
	s.MockDB.On("FindActivePromotions").Return([]models.Promotion{}, dbError)

	// Register middleware and a test route
	s.Router.Use(middleware.PromotionBanner(s.PromotionService))
	s.Router.GET("/", func(c *gin.Context) {
		// Check that no promotion was added to the context
		_, exists := c.Get("active_promotion")
		s.False(exists)

		c.String(http.StatusOK, "OK")
	})

	// Make a request
	req, _ := http.NewRequest("GET", "/", nil)
	s.Router.ServeHTTP(s.recorder, req)

	// Check response
	s.Equal(http.StatusOK, s.recorder.Code)

	// Verify mock was called
	s.MockDB.AssertExpectations(s.T())
}

// TestPromotionMiddlewareSuite runs the test suite
func TestPromotionMiddlewareSuite(t *testing.T) {
	suite.Run(t, new(PromotionMiddlewareTestSuite))
}
