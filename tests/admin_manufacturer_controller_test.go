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

// AdminManufacturerControllerTestSuite is a test suite for the AdminManufacturerController
type AdminManufacturerControllerTestSuite struct {
	ControllerTestSuite
	mockManufacturer *models.Manufacturer
}

// SetupTest sets up each test
func (s *AdminManufacturerControllerTestSuite) SetupTest() {
	// Run the parent's SetupTest
	s.ControllerTestSuite.SetupTest()

	// Create a mock manufacturer for testing
	s.mockManufacturer = &models.Manufacturer{
		Model:      gorm.Model{ID: 1},
		Name:       "Test Manufacturer",
		Nickname:   "Test Nickname",
		Country:    "Test Country",
		Popularity: 1,
	}
}

// CreateAdminManufacturerController creates and returns an AdminManufacturerController
func (s *AdminManufacturerControllerTestSuite) CreateAdminManufacturerController() *controller.AdminManufacturerController {
	if ctl, ok := s.Controllers["adminManufacturer"]; ok {
		return ctl.(*controller.AdminManufacturerController)
	}

	adminManufacturerController := controller.NewAdminManufacturerController(s.MockDB)
	s.Controllers["adminManufacturer"] = adminManufacturerController
	return adminManufacturerController
}

// TestIndexRoute tests the index route
func (s *AdminManufacturerControllerTestSuite) TestIndexRoute() {
	// Set up expectations for this specific test
	s.MockDB.On("FindAllManufacturers").Return([]models.Manufacturer{*s.mockManufacturer}, nil).Once()

	// Create the controller
	adminController := s.CreateAdminManufacturerController()

	// Register routes
	s.Router.GET("/admin/manufacturers", adminController.Index)

	// Create a test request
	req, _ := http.NewRequest("GET", "/admin/manufacturers", nil)
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response
	s.Equal(http.StatusOK, resp.Code)
	s.Contains(resp.Body.String(), "Manufacturers")

	// Verify mock expectations
	s.MockDB.AssertExpectations(s.T())
}

// TestNewRoute tests the new route
func (s *AdminManufacturerControllerTestSuite) TestNewRoute() {
	// Create the controller
	adminController := s.CreateAdminManufacturerController()

	// Register routes
	s.Router.GET("/admin/manufacturers/new", adminController.New)

	// Create a test request
	req, _ := http.NewRequest("GET", "/admin/manufacturers/new", nil)
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response
	s.Equal(http.StatusOK, resp.Code)
	s.Contains(resp.Body.String(), "New Manufacturer")
}

// TestShowRoute tests the show route
func (s *AdminManufacturerControllerTestSuite) TestShowRoute() {
	// Set up expectations for this specific test
	s.MockDB.On("FindManufacturerByID", uint(1)).Return(s.mockManufacturer, nil).Once()

	// Create the controller
	adminController := s.CreateAdminManufacturerController()

	// Register routes
	s.Router.GET("/admin/manufacturers/:id", adminController.Show)

	// Create a test request
	req, _ := http.NewRequest("GET", "/admin/manufacturers/1", nil)
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response
	s.Equal(http.StatusOK, resp.Code)
	s.Contains(resp.Body.String(), "Manufacturer Details")
	s.Contains(resp.Body.String(), "Test Manufacturer")

	// Verify mock expectations
	s.MockDB.AssertExpectations(s.T())
}

// TestEditRoute tests the edit route
func (s *AdminManufacturerControllerTestSuite) TestEditRoute() {
	// Set up expectations for this specific test
	s.MockDB.On("FindManufacturerByID", uint(1)).Return(s.mockManufacturer, nil).Once()

	// Create the controller
	adminController := s.CreateAdminManufacturerController()

	// Register routes
	s.Router.GET("/admin/manufacturers/:id/edit", adminController.Edit)

	// Create a test request
	req, _ := http.NewRequest("GET", "/admin/manufacturers/1/edit", nil)
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response
	s.Equal(http.StatusOK, resp.Code)
	s.Contains(resp.Body.String(), "Edit Manufacturer")
	s.Contains(resp.Body.String(), "Test Manufacturer")

	// Verify mock expectations
	s.MockDB.AssertExpectations(s.T())
}

// TestCreateRoute tests the create route
func (s *AdminManufacturerControllerTestSuite) TestCreateRoute() {
	// Setup expectations for create
	s.MockDB.On("CreateManufacturer", mock.AnythingOfType("*models.Manufacturer")).Return(nil).Once()

	// Create the controller
	adminController := s.CreateAdminManufacturerController()

	// Register routes
	s.Router.POST("/admin/manufacturers", adminController.Create)

	// Create form data
	form := url.Values{}
	form.Add("name", "New Test Manufacturer")
	form.Add("nickname", "New Test Nickname")
	form.Add("country", "New Test Country")

	// Create a test request
	req, _ := http.NewRequest("POST", "/admin/manufacturers", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response - should be a redirect
	s.Equal(http.StatusSeeOther, resp.Code)
	s.Equal("/admin/manufacturers?success=Manufacturer created successfully", resp.Header().Get("Location"))

	// Verify mock expectations
	s.MockDB.AssertExpectations(s.T())
}

// TestUpdateRoute tests the update route
func (s *AdminManufacturerControllerTestSuite) TestUpdateRoute() {
	// Setup expectations for update
	s.MockDB.On("FindManufacturerByID", uint(1)).Return(s.mockManufacturer, nil).Once()
	s.MockDB.On("UpdateManufacturer", mock.AnythingOfType("*models.Manufacturer")).Return(nil).Once()

	// Create the controller
	adminController := s.CreateAdminManufacturerController()

	// Register routes
	s.Router.POST("/admin/manufacturers/:id", adminController.Update)

	// Create form data
	form := url.Values{}
	form.Add("name", "Updated Test Manufacturer")
	form.Add("nickname", "Updated Test Nickname")
	form.Add("country", "Updated Test Country")

	// Create a test request
	req, _ := http.NewRequest("POST", "/admin/manufacturers/1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response - should be a redirect
	s.Equal(http.StatusSeeOther, resp.Code)
	s.Equal("/admin/manufacturers/1?success=Manufacturer updated successfully", resp.Header().Get("Location"))

	// Verify mock expectations
	s.MockDB.AssertExpectations(s.T())
}

// TestDeleteRoute tests the delete route
func (s *AdminManufacturerControllerTestSuite) TestDeleteRoute() {
	// Setup expectations for delete
	s.MockDB.On("DeleteManufacturer", uint(1)).Return(nil).Once()

	// Create the controller
	adminController := s.CreateAdminManufacturerController()

	// Register routes
	s.Router.POST("/admin/manufacturers/:id/delete", adminController.Delete)

	// Create a test request
	req, _ := http.NewRequest("POST", "/admin/manufacturers/1/delete", nil)
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response - should be a redirect
	s.Equal(http.StatusSeeOther, resp.Code)
	s.Equal("/admin/manufacturers?success=Manufacturer deleted successfully", resp.Header().Get("Location"))

	// Verify mock expectations
	s.MockDB.AssertExpectations(s.T())
}

// Run the tests
func TestAdminManufacturerControllerSuite(t *testing.T) {
	suite.Run(t, new(AdminManufacturerControllerTestSuite))
}
