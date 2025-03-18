package controller

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// TestOwnerProfileUpdate tests updating the owner profile
func TestOwnerProfileUpdate(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create a new mock DB
	mockDB := new(mocks.MockDB)
	mockEmailService := new(mocks.MockEmailService)

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
		Verified:            true,
	}

	// Create a copy of the user with the new email for update
	updatedUser := *user
	updatedUser.Email = "newemail@example.com"
	updatedUser.Verified = false
	updatedUser.VerificationToken = "new-token"
	updatedUser.VerificationTokenExpiry = time.Now().Add(24 * time.Hour)

	// Set up expectations
	mockAuthInfo := &mocks.MockAuthInfo{}
	mockAuthInfo.SetUserName("test@example.com")

	// Expect GetCurrentUser to be called and return the mock auth info and true
	mockAuthController.On("GetCurrentUser", mock.Anything).Return(mockAuthInfo, true)

	// Expect GetUserByEmail to be called with the user's email and return the user
	mockDB.On("GetUserByEmail", mock.Anything, "test@example.com").Return(user, nil)

	// Expect UpdateUser to be called and return no error
	mockDB.On("UpdateUser", mock.Anything, mock.AnythingOfType("*database.User")).Run(func(args mock.Arguments) {
		// Validate that the user's pending email has been updated
		updatedUser := args.Get(1).(*database.User)
		assert.Equal(t, "newemail@example.com", updatedUser.PendingEmail)
		assert.Equal(t, "test@example.com", updatedUser.Email) // Original email should remain unchanged
		assert.True(t, updatedUser.Verified)                   // User should remain verified
		assert.NotEmpty(t, updatedUser.VerificationToken)
		assert.False(t, updatedUser.VerificationTokenExpiry.IsZero())
	}).Return(nil)

	// Expect email service to send verification email
	mockEmailService.On("SendEmailChangeVerification", "newemail@example.com", mock.AnythingOfType("string")).Return(nil)

	// Create owner controller
	ownerController := NewOwnerController(mockDB)

	// Create test router
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("authController", mockAuthController)
		c.Set("emailService", mockEmailService)
	})

	// Add the update profile route
	router.POST("/owner/profile/update", ownerController.UpdateProfile)

	t.Run("Profile update with email change redirects to verification sent page", func(t *testing.T) {
		// Create form data
		form := url.Values{}
		form.Add("email", "newemail@example.com")

		// Create request
		req := httptest.NewRequest("POST", "/owner/profile/update", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(resp, req)

		// Check response - should redirect to verification-sent
		assert.Equal(t, http.StatusSeeOther, resp.Code)
		assert.Equal(t, "/verification-sent", resp.Header().Get("Location"))

		// Verify that a cookie was set with the email
		cookies := resp.Result().Cookies()
		var emailCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == "verification_email" {
				emailCookie = cookie
				break
			}
		}
		assert.NotNil(t, emailCookie, "verification_email cookie should be set")
		assert.Equal(t, "newemail%40example.com", emailCookie.Value)

		// Verify mock expectations
		mockDB.AssertExpectations(t)
		mockEmailService.AssertExpectations(t)
	})
}

// TestOwnerProfileUpdateNoChanges tests updating the owner profile without changes
func TestOwnerProfileUpdateNoChanges(t *testing.T) {
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
		Verified:            true,
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

	// Add the update profile route
	router.POST("/owner/profile/update", ownerController.UpdateProfile)

	t.Run("Profile update with no changes redirects to profile", func(t *testing.T) {
		// Create form data with the same email
		form := url.Values{}
		form.Add("email", "test@example.com")

		// Create request
		req := httptest.NewRequest("POST", "/owner/profile/update", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(resp, req)

		// Check response - should redirect to profile with success message
		assert.Equal(t, http.StatusSeeOther, resp.Code)
		assert.Equal(t, "/owner/profile", resp.Header().Get("Location"))

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})
}

// TestOwnerProfileUpdateInvalidEmail tests updating the owner profile with an invalid email
func TestOwnerProfileUpdateInvalidEmail(t *testing.T) {
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
		Verified:            true,
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

	// Add the update profile route
	router.POST("/owner/profile/update", ownerController.UpdateProfile)

	t.Run("Profile update with invalid email shows error", func(t *testing.T) {
		// Create form data with an invalid email
		form := url.Values{}
		form.Add("email", "invalid-email")

		// Create request
		req := httptest.NewRequest("POST", "/owner/profile/update", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(resp, req)

		// Check response - should remain on edit page with error
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), "Invalid email format")

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})
}
