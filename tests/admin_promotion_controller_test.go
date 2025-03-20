package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/models"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// AdminPromotionControllerTestSuite is a test suite for the AdminPromotionController
type AdminPromotionControllerTestSuite struct {
	ControllerTestSuite
	mockPromotion *models.Promotion
}

// SetupTest sets up each test
func (s *AdminPromotionControllerTestSuite) SetupTest() {
	// Run the parent's SetupTest
	s.ControllerTestSuite.SetupTest()

	// Create mock dates for testing
	now := time.Now()
	endDate := now.AddDate(0, 1, 0) // One month from now

	// Create a mock promotion for testing
	s.mockPromotion = &models.Promotion{
		Model:         gorm.Model{ID: 1},
		Name:          "Test Free Trial",
		Type:          "free_trial",
		Active:        true,
		StartDate:     now,
		EndDate:       endDate,
		BenefitDays:   30,
		BenefitTier:   "monthly",
		DisplayOnHome: true,
		Description:   "Test promotion description",
		Banner:        "/images/banners/test-promotion.jpg",
	}
}

// CreateAdminPromotionController creates and returns an AdminPromotionController
func (s *AdminPromotionControllerTestSuite) CreateAdminPromotionController() *controller.AdminPromotionController {
	if ctl, ok := s.Controllers["adminPromotion"]; ok {
		return ctl.(*controller.AdminPromotionController)
	}

	adminPromotionController := controller.NewAdminPromotionController(s.MockDB)
	s.Controllers["adminPromotion"] = adminPromotionController
	return adminPromotionController
}

// TestNewRoute tests the new route for creating promotions
func (s *AdminPromotionControllerTestSuite) TestNewRoute() {
	// Create the controller
	adminController := s.CreateAdminPromotionController()

	// Register routes
	s.Router.GET("/admin/promotions/new", adminController.New)

	// Create a test request
	req, _ := http.NewRequest("GET", "/admin/promotions/new", nil)
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response
	s.Equal(http.StatusOK, resp.Code)
	s.Contains(resp.Body.String(), "New Promotion")
}

// Run the tests
func TestAdminPromotionControllerSuite(t *testing.T) {
	suite.Run(t, new(AdminPromotionControllerTestSuite))
}
