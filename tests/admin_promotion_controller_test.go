package tests

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/models"
	"github.com/stretchr/testify/mock"
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

// TestCreatePromotion tests the create action for promotions
func (s *AdminPromotionControllerTestSuite) TestCreatePromotion() {
	// Create the controller
	adminController := s.CreateAdminPromotionController()

	// Mock DB save behavior
	s.MockDB.On("CreatePromotion", mock.AnythingOfType("*models.Promotion")).Return(nil).Once()

	// Register routes
	s.Router.POST("/admin/promotions", adminController.Create)

	// Create form data
	startDate := time.Now().Format("2006-01-02")
	endDate := time.Now().AddDate(0, 1, 0).Format("2006-01-02")

	formData := url.Values{
		"name":          {"Test Free Trial"},
		"type":          {"free_trial"},
		"active":        {"true"},
		"startDate":     {startDate},
		"endDate":       {endDate},
		"benefitDays":   {"30"},
		"displayOnHome": {"true"},
		"description":   {"Test promotion description"},
		"banner":        {"/images/banners/test-promotion.jpg"},
	}

	// Create a test request
	req, _ := http.NewRequest("POST", "/admin/promotions", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert redirect response (assuming successful creation redirects)
	s.Equal(http.StatusSeeOther, resp.Code)
	s.Equal("/admin/promotions?success=Promotion created successfully", resp.Header().Get("Location"))

	// Assert that CreatePromotion was called
	s.MockDB.AssertExpectations(s.T())
}

// TestCreatePromotionValidationErrors tests validation errors during promotion creation
func (s *AdminPromotionControllerTestSuite) TestCreatePromotionValidationErrors() {
	// Create the controller
	adminController := s.CreateAdminPromotionController()

	// Register routes
	s.Router.POST("/admin/promotions", adminController.Create)

	// Create form data with missing required fields
	formData := url.Values{
		"type":   {"free_trial"},
		"active": {"true"},
		// Missing name and dates
	}

	// Create a test request
	req, _ := http.NewRequest("POST", "/admin/promotions", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert that we're shown the form again with errors
	s.Equal(http.StatusOK, resp.Code)
	s.Contains(resp.Body.String(), "New Promotion")
	s.Contains(resp.Body.String(), "bg-red-100") // Check for error styling class
}

// Run the tests
func TestAdminPromotionControllerSuite(t *testing.T) {
	suite.Run(t, new(AdminPromotionControllerTestSuite))
}
