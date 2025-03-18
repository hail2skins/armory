package tests

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/models"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// AdminCaliberControllerTestSuite is a test suite for the AdminCaliberController
type AdminCaliberControllerTestSuite struct {
	ControllerTestSuite
	mockCaliber *models.Caliber
}

// SetupTest sets up each test
func (s *AdminCaliberControllerTestSuite) SetupTest() {
	// Run the parent's SetupTest
	s.ControllerTestSuite.SetupTest()

	// Create a mock caliber for testing
	s.mockCaliber = &models.Caliber{
		Model:      gorm.Model{ID: 1},
		Caliber:    "Test Caliber",
		Nickname:   "Test",
		Popularity: 1,
	}
}

// CreateAdminCaliberController creates and returns an AdminCaliberController
func (s *AdminCaliberControllerTestSuite) CreateAdminCaliberController() *controller.AdminCaliberController {
	if ctl, ok := s.Controllers["adminCaliber"]; ok {
		return ctl.(*controller.AdminCaliberController)
	}

	adminCaliberController := controller.NewAdminCaliberController(s.MockDB)
	s.Controllers["adminCaliber"] = adminCaliberController
	return adminCaliberController
}

// TestIndexRoute tests the index route
func (s *AdminCaliberControllerTestSuite) TestIndexRoute() {
	// Set up expectations for this specific test
	s.MockDB.On("FindAllCalibers").Return([]models.Caliber{*s.mockCaliber}, nil).Once()

	// Create the controller
	adminController := s.CreateAdminCaliberController()

	// Register routes
	s.Router.GET("/admin/calibers", adminController.Index)

	// Create a test request
	req, _ := http.NewRequest("GET", "/admin/calibers", nil)
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response
	s.Equal(http.StatusOK, resp.Code)
	s.Contains(resp.Body.String(), "Calibers")

	// Verify mock expectations
	s.MockDB.AssertExpectations(s.T())
}

// TestNewRoute tests the new route
func (s *AdminCaliberControllerTestSuite) TestNewRoute() {
	// Create the controller
	adminController := s.CreateAdminCaliberController()

	// Register routes
	s.Router.GET("/admin/calibers/new", adminController.New)

	// Create a test request
	req, _ := http.NewRequest("GET", "/admin/calibers/new", nil)
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response
	s.Equal(http.StatusOK, resp.Code)
	s.Contains(resp.Body.String(), "New Caliber")
}

// TestShowRoute tests the show route
func (s *AdminCaliberControllerTestSuite) TestShowRoute() {
	// Set up expectations for this specific test
	s.MockDB.On("FindCaliberByID", uint(1)).Return(s.mockCaliber, nil).Once()

	// Create the controller
	adminController := s.CreateAdminCaliberController()

	// Register routes
	s.Router.GET("/admin/calibers/:id", adminController.Show)

	// Create a test request
	req, _ := http.NewRequest("GET", "/admin/calibers/1", nil)
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response
	s.Equal(http.StatusOK, resp.Code)
	s.Contains(resp.Body.String(), "Caliber Details")
	s.Contains(resp.Body.String(), "Test Caliber")

	// Verify mock expectations
	s.MockDB.AssertExpectations(s.T())
}

// TestEditRoute tests the edit route
func (s *AdminCaliberControllerTestSuite) TestEditRoute() {
	// Set up expectations for this specific test
	s.MockDB.On("FindCaliberByID", uint(1)).Return(s.mockCaliber, nil).Once()

	// Create the controller
	adminController := s.CreateAdminCaliberController()

	// Register routes
	s.Router.GET("/admin/calibers/:id/edit", adminController.Edit)

	// Create a test request
	req, _ := http.NewRequest("GET", "/admin/calibers/1/edit", nil)
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response
	s.Equal(http.StatusOK, resp.Code)
	s.Contains(resp.Body.String(), "Edit Caliber")
	s.Contains(resp.Body.String(), "Test Caliber")

	// Verify mock expectations
	s.MockDB.AssertExpectations(s.T())
}

// TestCreateRoute tests the create route
func (s *AdminCaliberControllerTestSuite) TestCreateRoute() {
	// Setup expectations for create
	s.MockDB.On("CreateCaliber", mock.AnythingOfType("*models.Caliber")).Return(nil).Once()

	// Create the controller
	adminController := s.CreateAdminCaliberController()

	// Register routes
	s.Router.POST("/admin/calibers", adminController.Create)

	// Create form data
	form := url.Values{}
	form.Add("caliber", "New Test Caliber")
	form.Add("nickname", "New Test")

	// Create a test request
	req, _ := http.NewRequest("POST", "/admin/calibers", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response - should be a redirect
	s.Equal(http.StatusSeeOther, resp.Code)
	s.Equal("/admin/calibers?success=Caliber created successfully", resp.Header().Get("Location"))

	// Verify mock expectations
	s.MockDB.AssertExpectations(s.T())
}

// TestUpdateRoute tests the update route
func (s *AdminCaliberControllerTestSuite) TestUpdateRoute() {
	// Setup expectations for update
	s.MockDB.On("FindCaliberByID", uint(1)).Return(s.mockCaliber, nil).Once()
	s.MockDB.On("UpdateCaliber", mock.AnythingOfType("*models.Caliber")).Return(nil).Once()

	// Create the controller
	adminController := s.CreateAdminCaliberController()

	// Register routes
	s.Router.POST("/admin/calibers/:id", adminController.Update)

	// Create form data
	form := url.Values{}
	form.Add("caliber", "Updated Test Caliber")
	form.Add("nickname", "Updated Test")

	// Create a test request
	req, _ := http.NewRequest("POST", "/admin/calibers/1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response - should be a redirect
	s.Equal(http.StatusSeeOther, resp.Code)
	s.Equal("/admin/calibers/1?success=Caliber updated successfully", resp.Header().Get("Location"))

	// Verify mock expectations
	s.MockDB.AssertExpectations(s.T())
}

// TestDeleteRoute tests the delete route
func (s *AdminCaliberControllerTestSuite) TestDeleteRoute() {
	// Setup expectations for delete
	s.MockDB.On("DeleteCaliber", uint(1)).Return(nil).Once()

	// Create the controller
	adminController := s.CreateAdminCaliberController()

	// Register routes
	s.Router.POST("/admin/calibers/:id/delete", adminController.Delete)

	// Create a test request
	req, _ := http.NewRequest("POST", "/admin/calibers/1/delete", nil)
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response - should be a redirect
	s.Equal(http.StatusSeeOther, resp.Code)
	s.Equal("/admin/calibers?success=Caliber deleted successfully", resp.Header().Get("Location"))

	// Verify mock expectations
	s.MockDB.AssertExpectations(s.T())
}

// Run the tests
func TestAdminCaliberControllerSuite(t *testing.T) {
	suite.Run(t, new(AdminCaliberControllerTestSuite))
}
