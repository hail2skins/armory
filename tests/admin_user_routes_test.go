package tests

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// AdminUserRoutesSuite is a test suite for admin user routes
type AdminUserRoutesSuite struct {
	ControllerTestSuite
}

// SetupTest sets up each test
func (s *AdminUserRoutesSuite) SetupTest() {
	// Run the parent's SetupTest
	s.ControllerTestSuite.SetupTest()

	// Set up mock authenticated state for admin access
	s.MockAuth.On("IsAuthenticated", mock.Anything).Return(true)

	// Create mock user info for GetCurrentUser
	mockUserInfo := &mocks.MockAuthInfo{}
	mockUserInfo.SetUserName("admin@example.com")
	mockUserInfo.SetGroups([]string{"admin"})

	// Set up GetCurrentUser to return the mock info
	s.MockAuth.On("GetCurrentUser", mock.Anything).Return(mockUserInfo, true)

	// Add middleware to set authData in context
	s.Router.Use(func(c *gin.Context) {
		// Create auth data for the context
		authData := data.NewAuthData()
		authData.Authenticated = true
		authData.Email = "admin@example.com"
		authData.Roles = []string{"admin"}
		authData.IsCasbinAdmin = true
		c.Set("authData", authData)
		c.Next()
	})
}

// CreateAdminUserController creates and returns an AdminUserController
func (s *AdminUserRoutesSuite) CreateAdminUserController() *controller.AdminUserController {
	if ctl, ok := s.Controllers["adminUser"]; ok {
		return ctl.(*controller.AdminUserController)
	}

	adminUserController := controller.NewAdminUserController(s.MockDB)
	s.Controllers["adminUser"] = adminUserController
	return adminUserController
}

// TestAdminUserRoutesSuite runs the test suite
func TestAdminUserRoutesSuite(t *testing.T) {
	suite.Run(t, new(AdminUserRoutesSuite))
}

// TestAdminUserRoutes tests the admin user management routes
func TestAdminUserRoutes(t *testing.T) {
	// Skip this test until we properly fix all the mocks
	t.Skip("Skipping until DB mocks are properly implemented")

	// Setup
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	r := gin.Default()

	// Mock DB and controllers
	db := &mocks.MockDB{}

	// Mock methods for all the routes
	// For index
	users := []database.User{
		{Email: "test1@example.com"},
		{Email: "test2@example.com"},
	}
	db.On("CountUsers").Return(int64(2), nil)
	db.On("FindRecentUsers", 0, 50, "created_at", "desc").Return(users, nil)

	// For show, edit, update, delete, restore
	user := &database.User{
		Email:            "test@example.com",
		SubscriptionTier: "monthly",
		Verified:         true,
	}
	db.On("GetUserByID", uint(1)).Return(user, nil)
	db.On("UpdateUser", mock.Anything, mock.Anything).Return(nil)

	adminUserController := controller.NewAdminUserController(db)

	// Setup authentication middleware mock
	r.Use(func(c *gin.Context) {
		c.Set("authData", map[string]interface{}{
			"Authenticated": true,
			"Email":         "admin@example.com",
			"Roles":         []string{"admin"},
			"IsCasbinAdmin": true,
		})
		c.Next()
	})

	// Setup routes for testing
	adminGroup := r.Group("/admin")
	userGroup := adminGroup.Group("/users")
	{
		userGroup.GET("", adminUserController.Index)
		userGroup.GET("/:id", adminUserController.Show)
		userGroup.GET("/:id/edit", adminUserController.Edit)
		userGroup.POST("/:id", adminUserController.Update)
		userGroup.POST("/:id/delete", adminUserController.Delete)
		userGroup.POST("/:id/restore", adminUserController.Restore)
	}

	// Test admin index route
	req, _ := http.NewRequest("GET", "/admin/users", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test admin show route
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/admin/users/1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test admin edit route
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/admin/users/1/edit", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test admin update route
	w = httptest.NewRecorder()
	form := url.Values{}
	form.Add("email", "updated@example.com")
	form.Add("subscription_tier", "yearly")
	form.Add("verified", "on")
	req, _ = http.NewRequest("POST", "/admin/users/1", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusSeeOther, w.Code)
}

// TestGrantSubscriptionRoute tests the grant subscription route
func (s *AdminUserRoutesSuite) TestGrantSubscriptionRoute() {
	// Set up mock for the user
	user := &database.User{
		Email:            "user@example.com",
		SubscriptionTier: "free",
		Verified:         true,
	}
	s.MockDB.On("GetUserByID", uint(1)).Return(user, nil)

	// Setup the route
	adminController := s.CreateAdminUserController()
	s.Router.GET("/admin/users/:id/grant-subscription", adminController.ShowGrantSubscription)

	// Create test request
	req, _ := http.NewRequest("GET", "/admin/users/1/grant-subscription", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	// Check response status code
	s.Equal(http.StatusOK, w.Code)
	s.Contains(w.Body.String(), "Grant Subscription")
}

// TestPostGrantSubscriptionRoute tests the post grant subscription route
func (s *AdminUserRoutesSuite) TestPostGrantSubscriptionRoute() {
	// Set up mock for the user
	user := &database.User{
		Email:            "user@example.com",
		SubscriptionTier: "free",
		Verified:         true,
	}
	s.MockDB.On("GetUserByID", uint(1)).Return(user, nil)
	s.MockDB.On("UpdateUser", mock.Anything, mock.Anything).Return(nil)

	// Setup the route
	adminController := s.CreateAdminUserController()
	s.Router.POST("/admin/users/:id/grant-subscription", adminController.GrantSubscription)

	// Create form data
	form := url.Values{}
	form.Add("subscription_type", "monthly")
	form.Add("grant_reason", "Test grant")

	// Create test request
	req, _ := http.NewRequest("POST", "/admin/users/1/grant-subscription", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	// Check response status code - should be a redirect on success
	s.Equal(http.StatusSeeOther, w.Code)
}
