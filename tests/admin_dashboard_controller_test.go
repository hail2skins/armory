package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// AdminDashboardControllerTestSuite is a test suite for the AdminDashboardController
type AdminDashboardControllerTestSuite struct {
	ControllerTestSuite
}

// SetupTest sets up each test
func (s *AdminDashboardControllerTestSuite) SetupTest() {
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

// CreateAdminDashboardController creates and returns an AdminDashboardController
func (s *AdminDashboardControllerTestSuite) CreateAdminDashboardController() *controller.AdminDashboardController {
	if ctl, ok := s.Controllers["adminDashboard"]; ok {
		return ctl.(*controller.AdminDashboardController)
	}

	adminDashboardController := controller.NewAdminDashboardController(s.MockDB)
	s.Controllers["adminDashboard"] = adminDashboardController
	return adminDashboardController
}

// TestDashboardRoute tests the dashboard route
func (s *AdminDashboardControllerTestSuite) TestDashboardRoute() {
	// Mock the DB methods needed for the dashboard
	s.MockDB.On("CountUsers").Return(int64(100), nil).Once()

	// Mock FindRecentUsers with proper user data
	mockUsers := []database.User{
		{
			Model:            gorm.Model{ID: 1, CreatedAt: time.Now().Add(-24 * time.Hour)},
			Email:            "user1@example.com",
			SubscriptionTier: "free",
			LastLoginAttempt: time.Now().Add(-12 * time.Hour),
		},
		{
			Model:            gorm.Model{ID: 2, CreatedAt: time.Now().Add(-48 * time.Hour)},
			Email:            "user2@example.com",
			SubscriptionTier: "monthly",
			LastLoginAttempt: time.Now().Add(-6 * time.Hour),
		},
	}
	s.MockDB.On("FindRecentUsers", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockUsers, nil).Once()

	// Create the controller
	adminController := s.CreateAdminDashboardController()

	// Register routes
	s.Router.GET("/admin/dashboard", adminController.Dashboard)

	// Create a test request
	req, _ := http.NewRequest("GET", "/admin/dashboard", nil)
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response
	s.Equal(http.StatusOK, resp.Code)
	s.Contains(resp.Body.String(), "Admin Dashboard")

	// Verify mock expectations
	s.MockDB.AssertExpectations(s.T())
}

// TestDetailedHealthRoute tests the detailed health route
func (s *AdminDashboardControllerTestSuite) TestDetailedHealthRoute() {
	// Mock the DB health method with a map
	healthMap := map[string]string{
		"database": "OK",
		"server":   "OK",
	}
	s.MockDB.On("Health").Return(healthMap).Once()

	// Create the controller
	adminController := s.CreateAdminDashboardController()

	// Register routes
	s.Router.GET("/admin/detailed-health", adminController.DetailedHealth)

	// Create a test request
	req, _ := http.NewRequest("GET", "/admin/detailed-health", nil)
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response
	s.Equal(http.StatusOK, resp.Code)
	s.Contains(resp.Body.String(), "System Health")

	// Verify mock expectations
	s.MockDB.AssertExpectations(s.T())
}

// TestErrorMetricsRoute tests the error metrics route
func (s *AdminDashboardControllerTestSuite) TestErrorMetricsRoute() {
	// Create the controller
	adminController := s.CreateAdminDashboardController()

	// Register routes
	s.Router.GET("/admin/error-metrics", adminController.ErrorMetrics)

	// Create a test request
	req, _ := http.NewRequest("GET", "/admin/error-metrics", nil)
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response
	s.Equal(http.StatusOK, resp.Code)
	s.Contains(resp.Body.String(), "Error Metrics")

	// Verify mock expectations
	s.MockDB.AssertExpectations(s.T())
}

// Run the tests
func TestAdminDashboardControllerSuite(t *testing.T) {
	suite.Run(t, new(AdminDashboardControllerTestSuite))
}
