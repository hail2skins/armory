package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// CSRFAdminTemplatesSuite is a test suite for verifying CSRF tokens in admin templates
type CSRFAdminTemplatesSuite struct {
	suite.Suite
	Router     *gin.Engine
	MockDB     *mocks.MockDB
	Controller *controller.AdminUserController
	Recorder   *httptest.ResponseRecorder
}

// SetupTest is called before each test
func (s *CSRFAdminTemplatesSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	s.Router = gin.Default()
	s.MockDB = &mocks.MockDB{}
	s.Controller = controller.NewAdminUserController(s.MockDB)
	s.Recorder = httptest.NewRecorder()

	// Setup mock authentication middleware with CSRF token
	s.Router.Use(func(c *gin.Context) {
		c.Set("authData", map[string]interface{}{
			"Authenticated": true,
			"Email":         "admin@example.com",
			"Roles":         []string{"admin"},
			"IsCasbinAdmin": true,
			"CSRFToken":     "mX0OwCuPLFmTs4Og0tANOmccR6NpB6OsM1XfoDa3VWQ=", // More realistic CSRF token
		})
		c.Next()
	})

	// Add test routes
	s.Router.GET("/admin/users/:id/edit", s.Controller.Edit)
	s.Router.GET("/admin/users/:id/grant-subscription", s.Controller.ShowGrantSubscription)
}

// TestCSRFAdminTemplatesSuite runs the test suite
func TestCSRFAdminTemplatesSuite(t *testing.T) {
	suite.Run(t, new(CSRFAdminTemplatesSuite))
}

// TestUserEditCSRFToken tests that the user edit form includes a CSRF token
func (s *CSRFAdminTemplatesSuite) TestUserEditCSRFToken() {
	// Mock the GetUserByID method
	s.MockDB.On("GetUserByID", uint(1)).Return(&database.User{
		Model:            gorm.Model{ID: 1},
		Email:            "user@example.com",
		SubscriptionTier: "monthly",
		Verified:         true,
	}, nil)

	// Send request
	req, _ := http.NewRequest("GET", "/admin/users/1/edit", nil)
	s.Router.ServeHTTP(s.Recorder, req)

	// Assert response
	s.Equal(http.StatusOK, s.Recorder.Code)
	s.Contains(s.Recorder.Body.String(), "Edit User")

	// Check for CSRF token in form
	s.Contains(s.Recorder.Body.String(), `<input type="hidden" name="csrf_token" value="`)
}

// TestUserGrantSubscriptionCSRFToken tests that the user grant subscription form includes a CSRF token
func (s *CSRFAdminTemplatesSuite) TestUserGrantSubscriptionCSRFToken() {
	// Mock the GetUserByID method
	s.MockDB.On("GetUserByID", uint(1)).Return(&database.User{
		Model:            gorm.Model{ID: 1},
		Email:            "user@example.com",
		SubscriptionTier: "free",
		Verified:         true,
	}, nil)

	// Send request
	req, _ := http.NewRequest("GET", "/admin/users/1/grant-subscription", nil)
	s.Router.ServeHTTP(s.Recorder, req)

	// Assert response
	s.Equal(http.StatusOK, s.Recorder.Code)
	s.Contains(s.Recorder.Body.String(), "Grant Subscription")

	// Check for CSRF token in form
	s.Contains(s.Recorder.Body.String(), `<input type="hidden" name="csrf_token" value="`)
}
