package tests

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// AdminUserSuite is a test suite for the admin user controller
type AdminUserSuite struct {
	suite.Suite
	Router     *gin.Engine
	MockDB     *mocks.MockDB
	Controller *controller.AdminUserController
	Recorder   *httptest.ResponseRecorder
}

// SetupTest is called before each test
func (s *AdminUserSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	s.Router = gin.Default()
	s.MockDB = &mocks.MockDB{}
	s.Controller = controller.NewAdminUserController(s.MockDB)
	s.Recorder = httptest.NewRecorder()

	// Setup mock authentication middleware
	s.Router.Use(func(c *gin.Context) {
		c.Set("authData", map[string]interface{}{
			"Authenticated": true,
			"Email":         "admin@example.com",
			"Roles":         []string{"admin"},
			"IsCasbinAdmin": true,
		})
		c.Next()
	})

	// Add test routes
	s.Router.GET("/admin/users", s.Controller.Index)
	s.Router.GET("/admin/users/:id", s.Controller.Show)
	s.Router.GET("/admin/users/:id/edit", s.Controller.Edit)
	s.Router.POST("/admin/users/:id", s.Controller.Update)
	s.Router.POST("/admin/users/:id/delete", s.Controller.Delete)
	s.Router.POST("/admin/users/:id/restore", s.Controller.Restore)
}

// TestAdminUserSuite runs the test suite
func TestAdminUserSuite(t *testing.T) {
	suite.Run(t, new(AdminUserSuite))
}

// TestIndex tests the Index method
func (s *AdminUserSuite) TestIndex() {
	// Skip test until proper DB mock implementation is in place
	s.T().Skip("Skipping test until proper DB mock implementation is in place")

	// Mock the FindRecentUsers method
	users := []database.User{
		{
			Model: gorm.Model{ID: 1},
			Email: "user1@example.com",
		},
		{
			Model: gorm.Model{ID: 2},
			Email: "user2@example.com",
		},
	}
	s.MockDB.On("CountUsers").Return(int64(2), nil)
	s.MockDB.On("FindRecentUsers", 0, 50, "created_at", "desc").Return(users, nil)

	// Send request
	req, _ := http.NewRequest("GET", "/admin/users?q=test", nil)
	s.Router.ServeHTTP(s.Recorder, req)

	// Assert response
	s.Equal(http.StatusOK, s.Recorder.Code)
	s.Contains(s.Recorder.Body.String(), "User Management")
	s.Contains(s.Recorder.Body.String(), "user1@example.com")
	s.Contains(s.Recorder.Body.String(), "user2@example.com")
}

// TestShow tests the Show method
func (s *AdminUserSuite) TestShow() {
	// Mock the GetUserByID method
	s.MockDB.On("GetUserByID", uint(1)).Return(&database.User{
		Model:            gorm.Model{ID: 1},
		Email:            "user@example.com",
		SubscriptionTier: "monthly",
		Verified:         true,
	}, nil)

	// Send request
	req, _ := http.NewRequest("GET", "/admin/users/1", nil)
	s.Router.ServeHTTP(s.Recorder, req)

	// Assert response
	s.Equal(http.StatusOK, s.Recorder.Code)
	s.Contains(s.Recorder.Body.String(), "User Details")
	s.Contains(s.Recorder.Body.String(), "user@example.com")
	s.Contains(s.Recorder.Body.String(), "Monthly")
}

// TestEdit tests the Edit method
func (s *AdminUserSuite) TestEdit() {
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
	s.Contains(s.Recorder.Body.String(), "user@example.com")
	s.Contains(s.Recorder.Body.String(), "Monthly")
}

// TestUpdate tests the Update method
func (s *AdminUserSuite) TestUpdate() {
	// Mock the GetUserByID and UpdateUser methods
	s.MockDB.On("GetUserByID", uint(1)).Return(&database.User{
		Model:            gorm.Model{ID: 1},
		Email:            "user@example.com",
		SubscriptionTier: "monthly",
		Verified:         true,
	}, nil)
	s.MockDB.On("UpdateUser", mock.Anything, mock.Anything).Return(nil)

	// Prepare form data
	form := url.Values{}
	form.Add("email", "updated@example.com")
	form.Add("subscription_tier", "yearly")
	form.Add("verified", "on")

	// Send request
	req, _ := http.NewRequest("POST", "/admin/users/1", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	s.Router.ServeHTTP(s.Recorder, req)

	// Assert response
	s.Equal(http.StatusSeeOther, s.Recorder.Code)
	s.Equal("/admin/users/1?success=User+updated+successfully", s.Recorder.Header().Get("Location"))
}

// TestDelete tests the Delete method
func (s *AdminUserSuite) TestDelete() {
	// Skip test until proper DB mock implementation is in place
	s.T().Skip("Skipping test until proper DB mock implementation is in place")

	// Mock the GetUserByID method
	user := &database.User{
		Model: gorm.Model{ID: 1},
		Email: "user@example.com",
	}
	s.MockDB.On("GetUserByID", uint(1)).Return(user, nil)
	// Additional mocking would be needed for the delete operation

	// Send request
	req, _ := http.NewRequest("POST", "/admin/users/1/delete", nil)
	s.Router.ServeHTTP(s.Recorder, req)

	// Assert response
	s.Equal(http.StatusSeeOther, s.Recorder.Code)
	s.Equal("/admin/users?success=User+deleted+successfully", s.Recorder.Header().Get("Location"))
}

// TestRestore tests the Restore method
func (s *AdminUserSuite) TestRestore() {
	// Skip test until proper DB mock implementation is in place
	s.T().Skip("Skipping test until proper DB mock implementation is in place")

	// Send request
	req, _ := http.NewRequest("POST", "/admin/users/1/restore", nil)
	s.Router.ServeHTTP(s.Recorder, req)

	// Assert response
	s.Equal(http.StatusSeeOther, s.Recorder.Code)
	s.Equal("/admin/users?success=User+restored+successfully", s.Recorder.Header().Get("Location"))
}
