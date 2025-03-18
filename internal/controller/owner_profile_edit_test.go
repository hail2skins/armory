package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// TestOwnerProfileEdit tests the owner profile edit page
func TestOwnerProfileEdit(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create a new mock DB
	mockDB := new(mocks.MockDB)

	// Create a new mock auth controller
	mockAuthController := new(mocks.MockAuthController)

	// Create a test user
	user := &database.User{
		Model: gorm.Model{
			ID: 1,
		},
		Email:               "test@example.com",
		SubscriptionTier:    "premium",
		SubscriptionEndDate: time.Now().Add(24 * time.Hour),
	}

	// Set up expectations
	mockAuthInfo := &mocks.MockAuthInfo{}
	mockAuthInfo.SetUserName("test@example.com")

	// Expect GetCurrentUser to be called and return the mock auth info and true
	mockAuthController.On("GetCurrentUser", mock.Anything).Return(mockAuthInfo, true)

	// Expect GetUserByEmail to be called with the user's email and return the user
	mockDB.On("GetUserByEmail", mock.Anything, "test@example.com").Return(user, nil)

	// Create owner controller
	ownerController := NewOwnerController(mockDB)

	// Create test router
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("authController", mockAuthController)
	})

	// Add the edit profile route - now using EditProfile handler
	router.GET("/owner/profile/edit", ownerController.EditProfile)

	t.Run("Edit profile page shows for authenticated user", func(t *testing.T) {
		// Create request
		req := httptest.NewRequest("GET", "/owner/profile/edit", nil)
		resp := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(resp, req)

		// Check response
		assert.Equal(t, http.StatusOK, resp.Code)

		// Check content
		responseBody := resp.Body.String()
		assert.Contains(t, responseBody, "Edit Profile")
		assert.Contains(t, responseBody, "Email Address")
		assert.Contains(t, responseBody, "test@example.com")
		assert.Contains(t, responseBody, "If you change your email, you will need to verify it again")
		assert.Contains(t, responseBody, "Password")
		assert.Contains(t, responseBody, "Reset Password")
		assert.Contains(t, responseBody, "Save Changes")
	})
}

// TestOwnerProfileEditUnauthenticated tests that unauthenticated users are redirected to login
func TestOwnerProfileEditUnauthenticated(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create a new mock DB
	mockDB := new(mocks.MockDB)

	// Create a new mock auth controller
	mockAuthController := new(mocks.MockAuthController)

	// Create a mock auth info
	mockAuthInfo := &mocks.MockAuthInfo{}

	// Expect GetCurrentUser to be called and return the mock auth info and false
	mockAuthController.On("GetCurrentUser", mock.Anything).Return(mockAuthInfo, false)

	// Create owner controller
	ownerController := NewOwnerController(mockDB)

	// Create test router
	router := gin.New()
	var capturedFlash string
	router.Use(func(c *gin.Context) {
		c.Set("authController", mockAuthController)
		c.Set("setFlash", func(msg string) {
			capturedFlash = msg
		})
		c.Next()
	})

	// Add the profile route - using EditProfile handler
	router.GET("/owner/profile/edit", ownerController.EditProfile)

	t.Run("Edit profile page redirects to login for unauthenticated user", func(t *testing.T) {
		// Create request
		req := httptest.NewRequest("GET", "/owner/profile/edit", nil)
		resp := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(resp, req)

		// Check response
		assert.Equal(t, http.StatusSeeOther, resp.Code)
		assert.Equal(t, "/login", resp.Header().Get("Location"))

		// Check that a permission message was set
		assert.Contains(t, capturedFlash, "must be logged in")
	})
}

// TestOwnerProfileEditResetPasswordLink tests that the reset password link is correct
func TestOwnerProfileEditResetPasswordLink(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create a new mock DB
	mockDB := new(mocks.MockDB)

	// Create a new mock auth controller
	mockAuthController := new(mocks.MockAuthController)

	// Create a test user
	user := &database.User{
		Model: gorm.Model{
			ID: 1,
		},
		Email:               "test@example.com",
		SubscriptionTier:    "premium",
		SubscriptionEndDate: time.Now().Add(24 * time.Hour),
	}

	// Set up expectations
	mockAuthInfo := &mocks.MockAuthInfo{}
	mockAuthInfo.SetUserName("test@example.com")

	// Expect GetCurrentUser to be called and return the mock auth info and true
	mockAuthController.On("GetCurrentUser", mock.Anything).Return(mockAuthInfo, true)

	// Expect GetUserByEmail to be called with the user's email and return the user
	mockDB.On("GetUserByEmail", mock.Anything, "test@example.com").Return(user, nil)

	// Create owner controller
	ownerController := NewOwnerController(mockDB)

	// Create test router
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("authController", mockAuthController)
	})

	// Add the edit profile route
	router.GET("/owner/profile/edit", ownerController.EditProfile)

	t.Run("Reset password link points to /reset-password/new", func(t *testing.T) {
		// Create request
		req := httptest.NewRequest("GET", "/owner/profile/edit", nil)
		resp := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(resp, req)

		// Check response
		assert.Equal(t, http.StatusOK, resp.Code)

		// Check the reset password link
		responseBody := resp.Body.String()
		assert.Contains(t, responseBody, "href=\"/reset-password/new\"")
		assert.Contains(t, responseBody, "Reset Password")
	})
}
