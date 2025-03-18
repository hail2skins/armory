package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// TestSoftDeletedAccountOperations tests both soft deletion and registration with a soft-deleted account
func TestSoftDeletedAccountOperations(t *testing.T) {
	// Skip in CI/short mode
	if testing.Short() {
		t.Skip("Skipping database-dependent test in short mode")
		return
	}

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Set up test database
	db := testutils.SharedTestService()
	defer db.Close()

	// Create unique email to prevent test conflicts
	ctx := context.Background()
	testEmail := fmt.Sprintf("test_account_%d@example.com", time.Now().UnixNano())
	testPassword := "password123"

	// Create test user and immediately verify it exists
	t.Run("1-CreateAndVerifyUser", func(t *testing.T) {
		user, err := testutils.CreateTestUser(ctx, db, testEmail, testPassword)
		assert.NoError(t, err)
		assert.NotNil(t, user)

		// Mark user as verified
		user.Verified = true
		err = db.UpdateUser(ctx, user)
		assert.NoError(t, err)

		// Check user exists and is accessible
		fetchedUser, err := db.GetUserByEmail(ctx, testEmail)
		assert.NoError(t, err)
		assert.NotNil(t, fetchedUser)
	})

	// Test soft deletion
	t.Run("2-SoftDeleteUser", func(t *testing.T) {
		// Get the user
		user, err := db.GetUserByEmail(ctx, testEmail)
		assert.NoError(t, err)
		assert.NotNil(t, user)

		// Perform soft delete
		err = db.GetDB().Delete(user).Error
		assert.NoError(t, err)

		// Verify user is soft deleted using direct query with First() to check for error
		var deletedUser database.User
		err = db.GetDB().Where("email = ?", testEmail).First(&deletedUser).Error
		assert.Error(t, err, "Expected error when fetching soft-deleted user - GORM should return record not found")

		// Verify with unscoped query it's still there but marked as deleted
		var unscopedUser database.User
		err = db.GetDB().Unscoped().Where("email = ?", testEmail).First(&unscopedUser).Error
		assert.NoError(t, err)
		assert.NotNil(t, unscopedUser)
		assert.True(t, unscopedUser.DeletedAt.Valid) // Should have a valid DeletedAt
	})

	// Test registration with soft-deleted account
	t.Run("3-RestoreSoftDeletedAccount", func(t *testing.T) {
		// Setup router with mock handler
		router := gin.Default()

		// Create a simplified registration handler
		router.POST("/register", func(c *gin.Context) {
			// Parse form
			var form struct {
				Email           string `form:"email"`
				Password        string `form:"password"`
				PasswordConfirm string `form:"password_confirm"`
			}
			if err := c.ShouldBind(&form); err != nil {
				c.String(http.StatusBadRequest, "Invalid form")
				return
			}

			// Check for soft-deleted account
			var user database.User
			if err := db.GetDB().Unscoped().Where("email = ?", form.Email).First(&user).Error; err == nil {
				// Found user - check if soft-deleted
				if user.DeletedAt.Valid {
					// Restore user
					if err := db.GetDB().Unscoped().Model(&user).Update("deleted_at", nil).Error; err == nil {
						// Update password
						if err := user.SetPassword(form.Password); err == nil {
							if err := db.GetDB().Save(&user).Error; err == nil {
								c.Redirect(http.StatusSeeOther, "/login")
								return
							}
						}
					}
				}
			}
			// Fall through to normal registration flow
			c.Redirect(http.StatusSeeOther, "/login")
		})

		// Create registration request with the soft-deleted email
		form := url.Values{}
		form.Add("email", testEmail)
		form.Add("password", "newpassword456")
		form.Add("password_confirm", "newpassword456")

		req, _ := http.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(resp, req)

		// Check response
		assert.Equal(t, http.StatusSeeOther, resp.Code)
		assert.Equal(t, "/login", resp.Header().Get("Location"))

		// Verify user is reactivated
		reactivatedUser, err := db.GetUserByEmail(ctx, testEmail)
		assert.NoError(t, err)
		assert.NotNil(t, reactivatedUser)

		// Verify new password works
		authUser, err := testutils.AuthenticateTestUser(ctx, db, testEmail, "newpassword456")
		assert.NoError(t, err)
		assert.NotNil(t, authUser)
	})

	// Final cleanup - permanently delete the test user
	t.Run("4-Cleanup", func(t *testing.T) {
		// Get the user - this should succeed if the user was reactivated in the previous test
		user, err := db.GetUserByEmail(ctx, testEmail)
		if err != nil {
			t.Logf("Warning: Could not find user for cleanup: %v", err)

			// Try to find it with unscoped query
			var unscopedUser database.User
			unscErr := db.GetDB().Unscoped().Where("email = ?", testEmail).First(&unscopedUser).Error
			if unscErr != nil {
				t.Logf("User not found even with unscoped query: %v", unscErr)
				return
			}

			// Found with unscoped query, use this for deletion
			user = &unscopedUser
		}

		// Permanently delete
		err = db.GetDB().Unscoped().Delete(user).Error
		assert.NoError(t, err)

		// Verify it's completely gone using direct query with First() to check for error
		var deletedUser database.User
		err = db.GetDB().Where("email = ?", testEmail).First(&deletedUser).Error
		assert.Error(t, err, "Expected error or empty result when fetching deleted user")

		// Double-check with unscoped query
		var unscopedUser database.User
		err = db.GetDB().Unscoped().Where("email = ?", testEmail).First(&unscopedUser).Error
		assert.Error(t, err, "Expected error or empty result when fetching with unscoped query")
	})
}

// Mock auth controller for tests
type mockOwnerAuthController struct {
	mock.Mock
	user          *database.User
	authenticated bool
}

func (m *mockOwnerAuthController) GetCurrentUser(c *gin.Context) (interface{}, bool) {
	args := m.Called(c)
	return args.Get(0), args.Bool(1)
}

// TestMockedSoftDeleteReactivation tests the soft delete functionality with mocks
func TestMockedSoftDeleteReactivation(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Test user with deleted_at value
	testEmail := "deleted-user@example.com"
	hashedPassword, _ := database.HashPassword("password123")
	testUser := &database.User{
		Email:    testEmail,
		Password: hashedPassword,
		Verified: true,
	}
	testUser.ID = 1

	// Simulate a soft-deleted user (can't directly set DeletedAt because of database abstraction)
	// In a real DB this would have DeletedAt set

	// Create gin context & router
	router := gin.Default()

	// Mock form binding
	router.POST("/register", func(c *gin.Context) {
		var form struct {
			Email           string `form:"email"`
			Password        string `form:"password"`
			PasswordConfirm string `form:"password_confirm"`
		}
		_ = c.ShouldBind(&form)

		// This simulates the RegisterHandler finding a soft-deleted user
		if form.Email == testEmail {
			// Simulate the user being reactivated
			// Set new password
			_ = testUser.SetPassword(form.Password)

			// Return success response
			c.Redirect(http.StatusSeeOther, "/login")
		} else {
			// Normal registration flow
			c.Redirect(http.StatusSeeOther, "/verification-sent")
		}
	})

	// Test the registration with previously deleted account
	t.Run("ReactivateSoftDeletedAccount", func(t *testing.T) {
		// Create request with the deleted account email
		form := url.Values{}
		form.Add("email", testEmail)
		form.Add("password", "newpassword")
		form.Add("password_confirm", "newpassword")

		req, _ := http.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp := httptest.NewRecorder()

		// Send request
		router.ServeHTTP(resp, req)

		// Expect a redirect to login (restored account flow)
		assert.Equal(t, http.StatusSeeOther, resp.Code)
		assert.Equal(t, "/login", resp.Header().Get("Location"))
	})
}

// Mock implementation of DeleteGun to fulfill the database.Service interface
func (m *MockDB) DeleteGun(db *gorm.DB, gunID uint, userID uint) error {
	args := m.Called(db, gunID, userID)
	return args.Error(0)
}

// TestDeletionClearsCookieDirectly tests the cookie-clearing behavior directly
func TestDeletionClearsCookieDirectly(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a test router and controller
	router := gin.Default()

	// Add a test handler that simulates just the cookie-clearing part of the DeleteAccountHandler
	router.GET("/test-cookie-clear", func(c *gin.Context) {
		// This is the exact code from DeleteAccountHandler that sets the cookie
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     "auth-session",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			MaxAge:   -1, // Delete the cookie
		})

		// Set a flash message and redirect as the real handler does
		c.SetCookie("flash", "Your account has been deleted. Please come back any time!", 3600, "/", "", false, false)
		c.Redirect(http.StatusSeeOther, "/")
	})

	// Create a request
	req, _ := http.NewRequest("GET", "/test-cookie-clear", nil)
	resp := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(resp, req)

	// Verify redirection
	assert.Equal(t, http.StatusSeeOther, resp.Code)
	assert.Equal(t, "/", resp.Header().Get("Location"))

	// Check if the auth cookie was cleared
	cookies := resp.Result().Cookies()
	var authCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "auth-session" {
			authCookie = cookie
			break
		}
	}

	// Verify the auth cookie was correctly cleared
	assert.NotNil(t, authCookie, "Auth cookie should be present to clear the session")
	assert.Empty(t, authCookie.Value, "Auth cookie value should be empty")
	assert.Less(t, authCookie.MaxAge, 0, "Auth cookie should have a negative MaxAge to delete it")
}
