package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/middleware"
	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/services"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// PromotionMiddlewareUsageTestSuite tests how the promotion middleware
// is used with other controllers
type PromotionMiddlewareUsageTestSuite struct {
	suite.Suite
	Router           *gin.Engine
	MockDB           *mocks.MockDB
	PromotionService *services.PromotionService
	AuthController   *controller.AuthController
	HomeController   *controller.HomeController
	TestPromotion    *models.Promotion
}

// SetupTest sets up the test suite
func (s *PromotionMiddlewareUsageTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	s.MockDB = new(mocks.MockDB)
	s.PromotionService = services.NewPromotionService(s.MockDB)
	s.AuthController = controller.NewAuthController(s.MockDB)
	s.HomeController = controller.NewHomeController(s.MockDB)

	// Set the promotion service on the auth controller
	s.AuthController.SetPromotionService(s.PromotionService)

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

	// Set up router with middleware
	s.Router = gin.New()
}

// TestPromotionBannerInHomeController tests that the promotion middleware
// correctly adds the active promotion to the authData in the home controller
func (s *PromotionMiddlewareUsageTestSuite) TestPromotionBannerInHomeController() {
	// Mock active promotion
	s.MockDB.On("FindActivePromotions").Return([]models.Promotion{*s.TestPromotion}, nil)

	// Add middleware and routes
	// In a real app, this would be in server.go using the promotionService member
	s.Router.Use(middleware.PromotionBanner(s.PromotionService))

	// Register a handler that checks for the promotion in authData
	s.Router.GET("/", func(c *gin.Context) {
		// Create base authData
		authData := data.NewAuthData()

		// Get active promotion from context if it exists
		if promo, exists := c.Get("active_promotion"); exists {
			if promotion, ok := promo.(*models.Promotion); ok {
				// Update authData with the promotion
				authData = authData.WithActivePromotion(promotion)

				// Send success response with promotion info
				c.String(http.StatusOK, "Promotion found: %s", promotion.Name)
				return
			}
		}

		// If no promotion found, send different response
		c.String(http.StatusOK, "No active promotion")
	})

	// Make request
	req := httptest.NewRequest("GET", "/", nil)
	resp := httptest.NewRecorder()
	s.Router.ServeHTTP(resp, req)

	// Verify promotion was added
	s.Equal(http.StatusOK, resp.Code)
	s.Contains(resp.Body.String(), "Promotion found: Test Promotion")

	// Verify mock was called
	s.MockDB.AssertExpectations(s.T())
}

// TestPromotionBannerHiddenForAuthenticatedUsers tests that the promotion banner
// does not appear for authenticated users
func (s *PromotionMiddlewareUsageTestSuite) TestPromotionBannerHiddenForAuthenticatedUsers() {
	// Mock active promotion
	s.MockDB.On("FindActivePromotions").Return([]models.Promotion{*s.TestPromotion}, nil)

	// Add middleware and routes
	s.Router.Use(middleware.PromotionBanner(s.PromotionService))

	// Register a handler for the home route (which is in the whitelist)
	s.Router.GET("/", func(c *gin.Context) {
		// Create auth data with promotion
		authData := data.NewAuthData()

		// Get active promotion from context if it exists
		if promo, exists := c.Get("active_promotion"); exists {
			if promotion, ok := promo.(*models.Promotion); ok {
				// Add promotion to auth data
				authData = authData.WithActivePromotion(promotion)
			}
		}

		// Set authentication based on query parameter
		authData.Authenticated = c.Query("authenticated") == "true"

		// Return HTML that will conditionally show the banner based on authentication status
		htmlResponse := `<!DOCTYPE html><html><body>`

		// Only show banner for non-authenticated users with active promotion
		if authData.ActivePromotion != nil && !authData.Authenticated {
			htmlResponse += `<div id="promotion-banner">Promotion Banner</div>`
		}

		htmlResponse += `</body></html>`
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, htmlResponse)
	})

	// Test case 1: Non-authenticated user should see promotion banner
	req1 := httptest.NewRequest("GET", "/?authenticated=false", nil)
	resp1 := httptest.NewRecorder()
	s.Router.ServeHTTP(resp1, req1)

	s.Equal(http.StatusOK, resp1.Code)
	s.Contains(resp1.Body.String(), `<div id="promotion-banner">Promotion Banner</div>`)

	// Test case 2: Authenticated user should NOT see promotion banner
	req2 := httptest.NewRequest("GET", "/?authenticated=true", nil)
	resp2 := httptest.NewRecorder()
	s.Router.ServeHTTP(resp2, req2)

	s.Equal(http.StatusOK, resp2.Code)
	s.NotContains(resp2.Body.String(), `<div id="promotion-banner">Promotion Banner</div>`)

	// Verify mock was called (twice - once for each request)
	s.MockDB.AssertNumberOfCalls(s.T(), "FindActivePromotions", 2)
}

// Test suite entry point
func TestPromotionMiddlewareUsageSuite(t *testing.T) {
	suite.Run(t, new(PromotionMiddlewareUsageTestSuite))
}
