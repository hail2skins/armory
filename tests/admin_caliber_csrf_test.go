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

// AdminCaliberCSRFSuite is a test suite for verifying CSRF tokens in caliber templates
type AdminCaliberCSRFSuite struct {
	suite.Suite
	Router      *gin.Engine
	MockDB      *mocks.MockDB
	Controller  *controller.AdminCaliberController
	Recorder    *httptest.ResponseRecorder
	TestCaliber *models.Caliber
}

// SetupTest is called before each test
func (s *AdminCaliberCSRFSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	s.Router = gin.Default()
	s.MockDB = &mocks.MockDB{}
	s.Controller = controller.NewAdminCaliberController(s.MockDB)
	s.Recorder = httptest.NewRecorder()

	// Create a test caliber for use in tests
	s.TestCaliber = &models.Caliber{
		Model:      gorm.Model{ID: 1},
		Caliber:    "Test Caliber",
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
	s.Router.GET("/admin/calibers/new", s.Controller.New)
	s.Router.GET("/admin/calibers/:id/edit", s.Controller.Edit)
	s.Router.GET("/admin/calibers/:id", s.Controller.Show)
	s.Router.GET("/admin/calibers", s.Controller.Index)
}

// TestAdminCaliberCSRFSuite runs the test suite
func TestAdminCaliberCSRFSuite(t *testing.T) {
	suite.Run(t, new(AdminCaliberCSRFSuite))
}

// TestNewFormCSRFToken tests that the new caliber form includes a CSRF token
func (s *AdminCaliberCSRFSuite) TestNewFormCSRFToken() {
	// Send request
	req, _ := http.NewRequest("GET", "/admin/calibers/new", nil)
	s.Router.ServeHTTP(s.Recorder, req)

	// Assert response
	s.Equal(http.StatusOK, s.Recorder.Code)
	s.Contains(s.Recorder.Body.String(), "New Caliber")

	// Check for CSRF token in form
	s.Contains(s.Recorder.Body.String(), `<input type="hidden" name="csrf_token" value=`)
}

// TestEditFormCSRFToken tests that the edit caliber form includes a CSRF token
func (s *AdminCaliberCSRFSuite) TestEditFormCSRFToken() {
	// Mock the FindCaliberByID method
	s.MockDB.On("FindCaliberByID", uint(1)).Return(s.TestCaliber, nil)

	// Send request
	req, _ := http.NewRequest("GET", "/admin/calibers/1/edit", nil)
	s.Router.ServeHTTP(s.Recorder, req)

	// Assert response
	s.Equal(http.StatusOK, s.Recorder.Code)
	s.Contains(s.Recorder.Body.String(), "Edit Caliber")

	// Check for CSRF token in form
	s.Contains(s.Recorder.Body.String(), `<input type="hidden" name="csrf_token" value=`)
}

// TestShowCSRFToken tests that the delete form on the show page includes a CSRF token
func (s *AdminCaliberCSRFSuite) TestShowCSRFToken() {
	// Mock the FindCaliberByID method
	s.MockDB.On("FindCaliberByID", uint(1)).Return(s.TestCaliber, nil)

	// Send request
	req, _ := http.NewRequest("GET", "/admin/calibers/1", nil)
	s.Router.ServeHTTP(s.Recorder, req)

	// Assert response
	s.Equal(http.StatusOK, s.Recorder.Code)
	s.Contains(s.Recorder.Body.String(), "Caliber Details")

	// Check for CSRF token in delete form
	s.Contains(s.Recorder.Body.String(), `<form action="/admin/calibers/1/delete" method="post"`)
	s.Contains(s.Recorder.Body.String(), `<input type="hidden" name="csrf_token" value=`)
}

// TestIndexCSRFToken tests that the delete forms on the index page include CSRF tokens
func (s *AdminCaliberCSRFSuite) TestIndexCSRFToken() {
	// Mock the FindAllCalibers method
	s.MockDB.On("FindAllCalibers").Return([]models.Caliber{*s.TestCaliber}, nil)

	// Send request
	req, _ := http.NewRequest("GET", "/admin/calibers", nil)
	s.Router.ServeHTTP(s.Recorder, req)

	// Assert response
	s.Equal(http.StatusOK, s.Recorder.Code)
	s.Contains(s.Recorder.Body.String(), "Calibers")

	// Check for CSRF token in delete form
	s.Contains(s.Recorder.Body.String(), `<form action="/admin/calibers/1/delete" method="post"`)
	s.Contains(s.Recorder.Body.String(), `<input type="hidden" name="csrf_token" value=`)
}
