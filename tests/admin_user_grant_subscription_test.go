package tests

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// AdminUserGrantSubscriptionSuite is a test suite for the admin user grant subscription functionality
type AdminUserGrantSubscriptionSuite struct {
	suite.Suite
	Router     *gin.Engine
	MockDB     *mocks.MockDB
	Controller *controller.AdminUserController
	Recorder   *httptest.ResponseRecorder
}

// SetupTest is called before each test
func (s *AdminUserGrantSubscriptionSuite) SetupTest() {
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
			"CSRFToken":     "mX0OwCuPLFmTs4Og0tANOmccR6NpB6OsM1XfoDa3VWQ=",
		})
		c.Next()
	})

	// Add test routes
	s.Router.GET("/admin/users/:id/grant-subscription", s.Controller.ShowGrantSubscription)
	s.Router.POST("/admin/users/:id/grant-subscription", s.Controller.GrantSubscription)
}

// TestAdminUserGrantSubscriptionSuite runs the test suite
func TestAdminUserGrantSubscriptionSuite(t *testing.T) {
	suite.Run(t, new(AdminUserGrantSubscriptionSuite))
}

// TestShowGrantSubscription tests the ShowGrantSubscription method
func (s *AdminUserGrantSubscriptionSuite) TestShowGrantSubscription() {
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
	s.Contains(s.Recorder.Body.String(), "user@example.com")
	s.Contains(s.Recorder.Body.String(), "Subscription Type")
	s.Contains(s.Recorder.Body.String(), "Admin Grant")
}

// TestGrantSubscriptionExistingType tests granting an existing subscription type
func (s *AdminUserGrantSubscriptionSuite) TestGrantSubscriptionExistingType() {
	// Mock the GetUserByID method
	user := &database.User{
		Model:            gorm.Model{ID: 1},
		Email:            "user@example.com",
		SubscriptionTier: "free",
		Verified:         true,
	}
	s.MockDB.On("GetUserByID", uint(1)).Return(user, nil)

	// Create a variable to store the captured user
	var capturedUser *database.User

	// Mock the UpdateUser method with a callback to capture the user argument
	s.MockDB.On("UpdateUser", mock.Anything, mock.AnythingOfType("*database.User")).
		Run(func(args mock.Arguments) {
			// Capture the user from the second argument
			capturedUser = args.Get(1).(*database.User)
		}).
		Return(nil)

	// Prepare form data for existing subscription type
	form := url.Values{}
	form.Add("subscription_type", "monthly")
	form.Add("grant_reason", "Test grant")
	form.Add("csrf_token", "mX0OwCuPLFmTs4Og0tANOmccR6NpB6OsM1XfoDa3VWQ=")

	// Send request
	req, _ := http.NewRequest("POST", "/admin/users/1/grant-subscription", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	s.Router.ServeHTTP(s.Recorder, req)

	// Assert response
	s.Equal(http.StatusSeeOther, s.Recorder.Code)
	s.Equal("/admin/users/1?success=Subscription+granted+successfully", s.Recorder.Header().Get("Location"))

	// Verify that the user was updated correctly
	s.MockDB.AssertCalled(s.T(), "UpdateUser", mock.Anything, mock.AnythingOfType("*database.User"))

	// Check the captured user
	s.NotNil(capturedUser, "Expected a user to be passed to UpdateUser")
	s.Equal("monthly", capturedUser.SubscriptionTier)
	s.Equal("active", capturedUser.SubscriptionStatus)
	s.Equal(uint(1), capturedUser.GrantedByID)
	s.Equal("Test grant", capturedUser.GrantReason)
	s.True(capturedUser.IsAdminGranted)
	s.False(capturedUser.SubscriptionEndDate.IsZero(), "Subscription end date should be set")
}

// TestGrantSubscriptionAdminGrant tests granting an admin subscription with custom duration
func (s *AdminUserGrantSubscriptionSuite) TestGrantSubscriptionAdminGrant() {
	// Mock the GetUserByID method
	user := &database.User{
		Model:            gorm.Model{ID: 1},
		Email:            "user@example.com",
		SubscriptionTier: "free",
		Verified:         true,
	}
	s.MockDB.On("GetUserByID", uint(1)).Return(user, nil)

	// Create a variable to store the captured user
	var capturedUser *database.User

	// Mock the UpdateUser method with a callback to capture the user argument
	s.MockDB.On("UpdateUser", mock.Anything, mock.AnythingOfType("*database.User")).
		Run(func(args mock.Arguments) {
			// Capture the user from the second argument
			capturedUser = args.Get(1).(*database.User)
		}).
		Return(nil)

	// Prepare form data for admin grant with custom duration
	form := url.Values{}
	form.Add("subscription_type", "admin_grant")
	form.Add("duration_days", "30")
	form.Add("grant_reason", "Test admin grant")
	form.Add("csrf_token", "mX0OwCuPLFmTs4Og0tANOmccR6NpB6OsM1XfoDa3VWQ=")

	// Send request
	req, _ := http.NewRequest("POST", "/admin/users/1/grant-subscription", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	s.Router.ServeHTTP(s.Recorder, req)

	// Assert response
	s.Equal(http.StatusSeeOther, s.Recorder.Code)
	s.Equal("/admin/users/1?success=Subscription+granted+successfully", s.Recorder.Header().Get("Location"))

	// Verify that the user was updated correctly
	s.MockDB.AssertCalled(s.T(), "UpdateUser", mock.Anything, mock.AnythingOfType("*database.User"))

	// Check the captured user
	s.NotNil(capturedUser, "Expected a user to be passed to UpdateUser")
	s.Equal("admin_grant", capturedUser.SubscriptionTier)
	s.Equal("active", capturedUser.SubscriptionStatus)
	s.Equal(uint(1), capturedUser.GrantedByID)
	s.Equal("Test admin grant", capturedUser.GrantReason)
	s.True(capturedUser.IsAdminGranted)

	// Check that the subscription end date is approximately 30 days from now
	expectedEndDate := time.Now().AddDate(0, 0, 30)
	s.WithinDuration(expectedEndDate, capturedUser.SubscriptionEndDate, 10*time.Second)
}

// TestGrantSubscriptionLifetime tests granting a lifetime subscription
func (s *AdminUserGrantSubscriptionSuite) TestGrantSubscriptionLifetime() {
	// Mock the GetUserByID method
	user := &database.User{
		Model:            gorm.Model{ID: 1},
		Email:            "user@example.com",
		SubscriptionTier: "free",
		Verified:         true,
	}
	s.MockDB.On("GetUserByID", uint(1)).Return(user, nil)

	// Create a variable to store the captured user
	var capturedUser *database.User

	// Mock the UpdateUser method with a callback to capture the user argument
	s.MockDB.On("UpdateUser", mock.Anything, mock.AnythingOfType("*database.User")).
		Run(func(args mock.Arguments) {
			// Capture the user from the second argument
			capturedUser = args.Get(1).(*database.User)
		}).
		Return(nil)

	// Prepare form data for admin grant with lifetime subscription
	form := url.Values{}
	form.Add("subscription_type", "admin_grant")
	form.Add("is_lifetime", "on")
	form.Add("grant_reason", "Test lifetime grant")
	form.Add("csrf_token", "mX0OwCuPLFmTs4Og0tANOmccR6NpB6OsM1XfoDa3VWQ=")

	// Send request
	req, _ := http.NewRequest("POST", "/admin/users/1/grant-subscription", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	s.Router.ServeHTTP(s.Recorder, req)

	// Assert response
	s.Equal(http.StatusSeeOther, s.Recorder.Code)
	s.Equal("/admin/users/1?success=Subscription+granted+successfully", s.Recorder.Header().Get("Location"))

	// Verify that the user was updated correctly
	s.MockDB.AssertCalled(s.T(), "UpdateUser", mock.Anything, mock.AnythingOfType("*database.User"))

	// Check the captured user
	s.NotNil(capturedUser, "Expected a user to be passed to UpdateUser")
	s.Equal("admin_grant", capturedUser.SubscriptionTier)
	s.Equal("active", capturedUser.SubscriptionStatus)
	s.Equal(uint(1), capturedUser.GrantedByID)
	s.Equal("Test lifetime grant", capturedUser.GrantReason)
	s.True(capturedUser.IsAdminGranted)
	s.True(capturedUser.IsLifetime)
	s.True(capturedUser.SubscriptionEndDate.IsZero(), "Lifetime subscription should not have end date")
}

// TestGrantSubscriptionValidationError tests validation errors when granting a subscription
func (s *AdminUserGrantSubscriptionSuite) TestGrantSubscriptionValidationError() {
	// Mock the GetUserByID method
	s.MockDB.On("GetUserByID", uint(1)).Return(&database.User{
		Model:            gorm.Model{ID: 1},
		Email:            "user@example.com",
		SubscriptionTier: "free",
		Verified:         true,
	}, nil)

	// Prepare invalid form data (missing required fields)
	form := url.Values{}
	form.Add("csrf_token", "mX0OwCuPLFmTs4Og0tANOmccR6NpB6OsM1XfoDa3VWQ=")

	// Send request
	req, _ := http.NewRequest("POST", "/admin/users/1/grant-subscription", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	s.Router.ServeHTTP(s.Recorder, req)

	// Assert response (expect a validation error)
	s.Equal(http.StatusBadRequest, s.Recorder.Code)
	s.Contains(s.Recorder.Body.String(), "Subscription type is required")
}
