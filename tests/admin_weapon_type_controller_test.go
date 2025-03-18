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

// AdminWeaponTypeControllerTestSuite is a test suite for the AdminWeaponTypeController
type AdminWeaponTypeControllerTestSuite struct {
	ControllerTestSuite
	mockWeaponType *models.WeaponType
}

// SetupTest sets up each test
func (s *AdminWeaponTypeControllerTestSuite) SetupTest() {
	// Run the parent's SetupTest
	s.ControllerTestSuite.SetupTest()

	// Create a mock weapon type for testing
	s.mockWeaponType = &models.WeaponType{
		Model:      gorm.Model{ID: 1},
		Type:       "Test Weapon Type",
		Nickname:   "Test",
		Popularity: 1,
	}
}

// CreateAdminWeaponTypeController creates and returns an AdminWeaponTypeController
func (s *AdminWeaponTypeControllerTestSuite) CreateAdminWeaponTypeController() *controller.AdminWeaponTypeController {
	if ctl, ok := s.Controllers["adminWeaponType"]; ok {
		return ctl.(*controller.AdminWeaponTypeController)
	}

	adminWeaponTypeController := controller.NewAdminWeaponTypeController(s.MockDB)
	s.Controllers["adminWeaponType"] = adminWeaponTypeController
	return adminWeaponTypeController
}

// TestIndexRoute tests the index route
func (s *AdminWeaponTypeControllerTestSuite) TestIndexRoute() {
	// Set up expectations for this specific test
	s.MockDB.On("FindAllWeaponTypes").Return([]models.WeaponType{*s.mockWeaponType}, nil).Once()

	// Create the controller
	adminController := s.CreateAdminWeaponTypeController()

	// Register routes
	s.Router.GET("/admin/weapon_types", adminController.Index)

	// Create a test request
	req, _ := http.NewRequest("GET", "/admin/weapon_types", nil)
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response
	s.Equal(http.StatusOK, resp.Code)
	s.Contains(resp.Body.String(), "Weapon Types")

	// Verify mock expectations
	s.MockDB.AssertExpectations(s.T())
}

// TestNewRoute tests the new route
func (s *AdminWeaponTypeControllerTestSuite) TestNewRoute() {
	// Create the controller
	adminController := s.CreateAdminWeaponTypeController()

	// Register routes
	s.Router.GET("/admin/weapon_types/new", adminController.New)

	// Create a test request
	req, _ := http.NewRequest("GET", "/admin/weapon_types/new", nil)
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response
	s.Equal(http.StatusOK, resp.Code)
	s.Contains(resp.Body.String(), "New Weapon Type")
}

// TestShowRoute tests the show route
func (s *AdminWeaponTypeControllerTestSuite) TestShowRoute() {
	// Set up expectations for this specific test
	s.MockDB.On("FindWeaponTypeByID", uint(1)).Return(s.mockWeaponType, nil).Once()

	// Create the controller
	adminController := s.CreateAdminWeaponTypeController()

	// Register routes
	s.Router.GET("/admin/weapon_types/:id", adminController.Show)

	// Create a test request
	req, _ := http.NewRequest("GET", "/admin/weapon_types/1", nil)
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response
	s.Equal(http.StatusOK, resp.Code)
	s.Contains(resp.Body.String(), "Weapon Type Details")
	s.Contains(resp.Body.String(), "Test Weapon Type")

	// Verify mock expectations
	s.MockDB.AssertExpectations(s.T())
}

// TestEditRoute tests the edit route
func (s *AdminWeaponTypeControllerTestSuite) TestEditRoute() {
	// Set up expectations for this specific test
	s.MockDB.On("FindWeaponTypeByID", uint(1)).Return(s.mockWeaponType, nil).Once()

	// Create the controller
	adminController := s.CreateAdminWeaponTypeController()

	// Register routes
	s.Router.GET("/admin/weapon_types/:id/edit", adminController.Edit)

	// Create a test request
	req, _ := http.NewRequest("GET", "/admin/weapon_types/1/edit", nil)
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response
	s.Equal(http.StatusOK, resp.Code)
	s.Contains(resp.Body.String(), "Edit Weapon Type")
	s.Contains(resp.Body.String(), "Test Weapon Type")

	// Verify mock expectations
	s.MockDB.AssertExpectations(s.T())
}

// TestCreateRoute tests the create route
func (s *AdminWeaponTypeControllerTestSuite) TestCreateRoute() {
	// Setup expectations for create
	s.MockDB.On("CreateWeaponType", mock.AnythingOfType("*models.WeaponType")).Return(nil).Once()

	// Create the controller
	adminController := s.CreateAdminWeaponTypeController()

	// Register routes
	s.Router.POST("/admin/weapon_types", adminController.Create)

	// Create form data
	form := url.Values{}
	form.Add("type", "New Test Weapon Type")
	form.Add("nickname", "New Test")

	// Create a test request
	req, _ := http.NewRequest("POST", "/admin/weapon_types", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response - should be a redirect
	s.Equal(http.StatusSeeOther, resp.Code)
	s.Equal("/admin/weapon_types?success=Weapon type created successfully", resp.Header().Get("Location"))

	// Verify mock expectations
	s.MockDB.AssertExpectations(s.T())
}

// TestUpdateRoute tests the update route
func (s *AdminWeaponTypeControllerTestSuite) TestUpdateRoute() {
	// Setup expectations for update
	s.MockDB.On("FindWeaponTypeByID", uint(1)).Return(s.mockWeaponType, nil).Once()
	s.MockDB.On("UpdateWeaponType", mock.AnythingOfType("*models.WeaponType")).Return(nil).Once()

	// Create the controller
	adminController := s.CreateAdminWeaponTypeController()

	// Register routes
	s.Router.POST("/admin/weapon_types/:id", adminController.Update)

	// Create form data
	form := url.Values{}
	form.Add("type", "Updated Test Weapon Type")
	form.Add("nickname", "Updated Test")

	// Create a test request
	req, _ := http.NewRequest("POST", "/admin/weapon_types/1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response - should be a redirect
	s.Equal(http.StatusSeeOther, resp.Code)
	s.Equal("/admin/weapon_types/1?success=Weapon type updated successfully", resp.Header().Get("Location"))

	// Verify mock expectations
	s.MockDB.AssertExpectations(s.T())
}

// TestDeleteRoute tests the delete route
func (s *AdminWeaponTypeControllerTestSuite) TestDeleteRoute() {
	// Setup expectations for delete
	s.MockDB.On("DeleteWeaponType", uint(1)).Return(nil).Once()

	// Create the controller
	adminController := s.CreateAdminWeaponTypeController()

	// Register routes
	s.Router.POST("/admin/weapon_types/:id/delete", adminController.Delete)

	// Create a test request
	req, _ := http.NewRequest("POST", "/admin/weapon_types/1/delete", nil)
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response - should be a redirect
	s.Equal(http.StatusSeeOther, resp.Code)
	s.Equal("/admin/weapon_types?success=Weapon type deleted successfully", resp.Header().Get("Location"))

	// Verify mock expectations
	s.MockDB.AssertExpectations(s.T())
}

// Run the tests
func TestAdminWeaponTypeControllerSuite(t *testing.T) {
	suite.Run(t, new(AdminWeaponTypeControllerTestSuite))
}
