package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// AdminManufacturerCSRFSuite is a test suite for verifying CSRF tokens in manufacturer templates
type AdminManufacturerCSRFSuite struct {
	suite.Suite
	Router           *gin.Engine
	MockDB           *mocks.MockDB
	Controller       *controller.AdminManufacturerController
	Recorder         *httptest.ResponseRecorder
	TestManufacturer *models.Manufacturer
}

// SetupTest is called before each test
func (s *AdminManufacturerCSRFSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	s.Router = gin.Default()
	s.MockDB = &mocks.MockDB{}
	s.Controller = controller.NewAdminManufacturerController(s.MockDB)
	s.Recorder = httptest.NewRecorder()

	// Create a test manufacturer for use in tests
	s.TestManufacturer = &models.Manufacturer{
		Model:      gorm.Model{ID: 1},
		Name:       "Test Manufacturer",
		Nickname:   "Test",
		Country:    "Test Country",
		Popularity: 50,
	}

	// Setup mock authentication middleware with CSRF token
	s.Router.Use(func(c *gin.Context) {
		c.Set("authData", map[string]interface{}{
			"Authenticated": true,
			"Email":         "admin@example.com",
			"Roles":         []string{"admin"},
			"IsCasbinAdmin": true,
			"CSRFToken":     "mX0OwCuPLFmTs4Og0tANOmccR6NpB6OsM1XfoDa3VWQ=",
		})
		c.Next()
	})

	// Add test routes
	s.Router.GET("/admin/manufacturers/new", s.Controller.New)
	s.Router.GET("/admin/manufacturers/:id/edit", s.Controller.Edit)
	s.Router.GET("/admin/manufacturers/:id", s.Controller.Show)
	s.Router.GET("/admin/manufacturers", s.Controller.Index)
}

// TestAdminManufacturerCSRFSuite runs the test suite
func TestAdminManufacturerCSRFSuite(t *testing.T) {
	suite.Run(t, new(AdminManufacturerCSRFSuite))
}

// TestNewFormCSRFToken tests that the new manufacturer form includes a CSRF token
func (s *AdminManufacturerCSRFSuite) TestNewFormCSRFToken() {
	// Send request
	req, _ := http.NewRequest("GET", "/admin/manufacturers/new", nil)
	s.Router.ServeHTTP(s.Recorder, req)

	// Assert response
	s.Equal(http.StatusOK, s.Recorder.Code)
	s.Contains(s.Recorder.Body.String(), "New Manufacturer")

	// Check for CSRF token in form
	s.Contains(s.Recorder.Body.String(), `<input type="hidden" name="csrf_token" value=`)
}

// TestEditFormCSRFToken tests that the edit manufacturer form includes a CSRF token
func (s *AdminManufacturerCSRFSuite) TestEditFormCSRFToken() {
	// Mock the FindManufacturerByID method
	s.MockDB.On("FindManufacturerByID", uint(1)).Return(s.TestManufacturer, nil)

	// Send request
	req, _ := http.NewRequest("GET", "/admin/manufacturers/1/edit", nil)
	s.Router.ServeHTTP(s.Recorder, req)

	// Assert response
	s.Equal(http.StatusOK, s.Recorder.Code)
	s.Contains(s.Recorder.Body.String(), "Edit Manufacturer")

	// Check for CSRF token in form
	s.Contains(s.Recorder.Body.String(), `<input type="hidden" name="csrf_token" value=`)
}

// TestShowCSRFToken tests that the delete form on the show page includes a CSRF token
func (s *AdminManufacturerCSRFSuite) TestShowCSRFToken() {
	// Mock the FindManufacturerByID method
	s.MockDB.On("FindManufacturerByID", uint(1)).Return(s.TestManufacturer, nil)

	// Send request
	req, _ := http.NewRequest("GET", "/admin/manufacturers/1", nil)
	s.Router.ServeHTTP(s.Recorder, req)

	// Assert response
	s.Equal(http.StatusOK, s.Recorder.Code)
	s.Contains(s.Recorder.Body.String(), "Manufacturer Details")

	// Check for CSRF token in delete form
	s.Contains(s.Recorder.Body.String(), `<form action="/admin/manufacturers/1/delete" method="post"`)
	s.Contains(s.Recorder.Body.String(), `<input type="hidden" name="csrf_token" value=`)
}

// TestIndexCSRFToken tests that the delete forms on the index page include CSRF tokens
func (s *AdminManufacturerCSRFSuite) TestIndexCSRFToken() {
	// Mock the FindAllManufacturers method
	s.MockDB.On("FindAllManufacturers").Return([]models.Manufacturer{*s.TestManufacturer}, nil)

	// Send request
	req, _ := http.NewRequest("GET", "/admin/manufacturers", nil)
	s.Router.ServeHTTP(s.Recorder, req)

	// Assert response
	s.Equal(http.StatusOK, s.Recorder.Code)
	s.Contains(s.Recorder.Body.String(), "Manufacturers")

	// Check for CSRF token in delete form
	s.Contains(s.Recorder.Body.String(), `<form action="/admin/manufacturers/1/delete" method="post"`)
	s.Contains(s.Recorder.Body.String(), `<input type="hidden" name="csrf_token" value=`)
}
