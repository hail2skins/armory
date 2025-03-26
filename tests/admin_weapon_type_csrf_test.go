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

// AdminWeaponTypeCSRFSuite is a test suite for verifying CSRF tokens in weapon type templates
type AdminWeaponTypeCSRFSuite struct {
	suite.Suite
	Router         *gin.Engine
	MockDB         *mocks.MockDB
	Controller     *controller.AdminWeaponTypeController
	Recorder       *httptest.ResponseRecorder
	TestWeaponType *models.WeaponType
}

// SetupTest is called before each test
func (s *AdminWeaponTypeCSRFSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	s.Router = gin.Default()
	s.MockDB = &mocks.MockDB{}
	s.Controller = controller.NewAdminWeaponTypeController(s.MockDB)
	s.Recorder = httptest.NewRecorder()

	// Create a test weapon type for use in tests
	s.TestWeaponType = &models.WeaponType{
		Model:      gorm.Model{ID: 1},
		Type:       "Test Type",
		Nickname:   "Test",
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
	s.Router.GET("/admin/weapon_types/new", s.Controller.New)
	s.Router.GET("/admin/weapon_types/:id/edit", s.Controller.Edit)
	s.Router.GET("/admin/weapon_types/:id", s.Controller.Show)
	s.Router.GET("/admin/weapon_types", s.Controller.Index)
}

// TestAdminWeaponTypeCSRFSuite runs the test suite
func TestAdminWeaponTypeCSRFSuite(t *testing.T) {
	suite.Run(t, new(AdminWeaponTypeCSRFSuite))
}

// TestNewFormCSRFToken tests that the new weapon type form includes a CSRF token
func (s *AdminWeaponTypeCSRFSuite) TestNewFormCSRFToken() {
	// Send request
	req, _ := http.NewRequest("GET", "/admin/weapon_types/new", nil)
	s.Router.ServeHTTP(s.Recorder, req)

	// Assert response
	s.Equal(http.StatusOK, s.Recorder.Code)
	s.Contains(s.Recorder.Body.String(), "New Weapon Type")

	// Check for CSRF token in form
	s.Contains(s.Recorder.Body.String(), `<input type="hidden" name="csrf_token" value=`)
}

// TestEditFormCSRFToken tests that the edit weapon type form includes a CSRF token
func (s *AdminWeaponTypeCSRFSuite) TestEditFormCSRFToken() {
	// Mock the FindWeaponTypeByID method
	s.MockDB.On("FindWeaponTypeByID", uint(1)).Return(s.TestWeaponType, nil)

	// Send request
	req, _ := http.NewRequest("GET", "/admin/weapon_types/1/edit", nil)
	s.Router.ServeHTTP(s.Recorder, req)

	// Assert response
	s.Equal(http.StatusOK, s.Recorder.Code)
	s.Contains(s.Recorder.Body.String(), "Edit Weapon Type")

	// Check for CSRF token in form
	s.Contains(s.Recorder.Body.String(), `<input type="hidden" name="csrf_token" value=`)
}

// TestShowCSRFToken tests that the delete form on the show page includes a CSRF token
func (s *AdminWeaponTypeCSRFSuite) TestShowCSRFToken() {
	// Mock the FindWeaponTypeByID method
	s.MockDB.On("FindWeaponTypeByID", uint(1)).Return(s.TestWeaponType, nil)

	// Send request
	req, _ := http.NewRequest("GET", "/admin/weapon_types/1", nil)
	s.Router.ServeHTTP(s.Recorder, req)

	// Assert response
	s.Equal(http.StatusOK, s.Recorder.Code)
	s.Contains(s.Recorder.Body.String(), "Weapon Type Details")

	// Check for CSRF token in delete form
	s.Contains(s.Recorder.Body.String(), `<form action="/admin/weapon_types/1/delete" method="post"`)
	s.Contains(s.Recorder.Body.String(), `<input type="hidden" name="csrf_token" value=`)
}

// TestIndexCSRFToken tests that the delete forms on the index page include CSRF tokens
func (s *AdminWeaponTypeCSRFSuite) TestIndexCSRFToken() {
	// Mock the FindAllWeaponTypes method
	s.MockDB.On("FindAllWeaponTypes").Return([]models.WeaponType{*s.TestWeaponType}, nil)

	// Send request
	req, _ := http.NewRequest("GET", "/admin/weapon_types", nil)
	s.Router.ServeHTTP(s.Recorder, req)

	// Assert response
	s.Equal(http.StatusOK, s.Recorder.Code)
	s.Contains(s.Recorder.Body.String(), "Weapon Types")

	// Check for CSRF token in delete form
	s.Contains(s.Recorder.Body.String(), `<form action="/admin/weapon_types/1/delete" method="post"`)
	s.Contains(s.Recorder.Body.String(), `<input type="hidden" name="csrf_token" value=`)
}
