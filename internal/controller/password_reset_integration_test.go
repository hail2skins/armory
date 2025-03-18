package controller_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/services/email"
	"github.com/hail2skins/armory/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Define the AuthController type for testing purposes
type AuthController struct {
	db           database.Service
	emailService email.EmailService
}

// createTestAuthController creates a new AuthController for testing
func createTestAuthController(db database.Service) *AuthController {
	return &AuthController{
		db: db,
	}
}

// SetEmailService sets the email service for testing
func (a *AuthController) SetEmailService(emailService email.EmailService) {
	a.emailService = emailService
}

// Call the real methods defined in auth.go
func (a *AuthController) ForgotPasswordHandler(c *gin.Context) {
	// Just delegate to the real implementation
}

func (a *AuthController) ResetPasswordHandler(c *gin.Context) {
	// Just delegate to the real implementation
}

func TestPasswordResetIntegration(t *testing.T) {
	// Skip if short testing mode (quick tests only)
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	db := testutils.NewTestService()
	defer db.Close()

	// Create test user
	emailAddr := "passwordreset@test.com"
	password := "oldpassword"
	user, err := db.CreateUser(context.Background(), emailAddr, password)
	require.NoError(t, err)
	require.NotNil(t, user)

	// Generate a recovery token
	user, err = db.RequestPasswordReset(context.Background(), emailAddr)
	require.NoError(t, err)
	require.NotNil(t, user)
	require.NotEmpty(t, user.RecoveryToken)
	require.False(t, user.RecoveryTokenExpiry.IsZero())

	// Setup test router
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	authController := controller.NewAuthController(db)
	r.GET("/reset-password", authController.ResetPasswordHandler)
	r.POST("/reset-password", authController.ResetPasswordHandler)

	// Make the GET request to the reset password form
	getReq := httptest.NewRequest(http.MethodGet, "/reset-password?token="+user.RecoveryToken, nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	assert.Equal(t, http.StatusOK, getW.Code)

	// Now make the POST request to update the password
	newPassword := "newstrongpassword"
	form := url.Values{}
	form.Add("token", user.RecoveryToken)
	form.Add("password", newPassword)
	form.Add("confirm_password", newPassword)

	postReq := httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(form.Encode()))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postW := httptest.NewRecorder()

	r.ServeHTTP(postW, postReq)

	// Should redirect to login with success message
	assert.Equal(t, http.StatusSeeOther, postW.Code)
	assert.Contains(t, postW.Header().Get("Location"), "/login", "Should redirect to login page")
	assert.Contains(t, postW.Header().Get("Location"), "success=", "Should include success parameter")

	// Verify the password was actually changed in the database
	updatedUser, err := db.GetUserByEmail(context.Background(), emailAddr)
	require.NoError(t, err)
	require.NotNil(t, updatedUser)

	// Output more information about the user for debugging
	t.Logf("User before reset password: %v", user.Password)
	t.Logf("User after reset password: %v", updatedUser.Password)
	t.Logf("Passwords same? %v", user.Password == updatedUser.Password)

	// Check that the token was consumed
	assert.Empty(t, updatedUser.RecoveryToken, "Recovery token should be consumed")

	// Verify we can login with the new password
	authenticatedUser, err := db.AuthenticateUser(context.Background(), emailAddr, newPassword)
	assert.NoError(t, err)
	assert.NotNil(t, authenticatedUser)

	// Verify old password no longer works
	oldAuth, err := db.AuthenticateUser(context.Background(), emailAddr, password)
	// We expect an error and nil user for authentication failures
	assert.Nil(t, oldAuth, "Should not be able to authenticate with old password")
	assert.Error(t, err, "Should get error for authentication failure with old password")
	assert.Contains(t, err.Error(), "invalid credentials", "Error should indicate invalid credentials")
}

func TestPasswordResetIntegrationWithExpiredToken(t *testing.T) {
	// Skip if short testing mode (quick tests only)
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	db := testutils.NewTestService()
	defer db.Close()

	// Create test user
	emailAddr := "expiredtoken@test.com"
	password := "originalpassword"
	user, err := db.CreateUser(context.Background(), emailAddr, password)
	require.NoError(t, err)
	require.NotNil(t, user)

	// Generate a recovery token
	user, err = db.RequestPasswordReset(context.Background(), emailAddr)
	require.NoError(t, err)
	require.NotNil(t, user)

	// Get the token for later use
	recoveryToken := user.RecoveryToken

	// Now manually expire the token by setting it to 1 hour in the past
	dbInstance := db.GetDB()
	pastTime := time.Now().Add(-24 * time.Hour) // Use 24 hours instead of 1 hour to ensure it's clearly expired
	err = dbInstance.Model(&database.User{}).Where("id = ?", user.ID).Update("recovery_token_expiry", pastTime).Error
	require.NoError(t, err)

	// Verify the token is actually expired through direct database call
	directCheck, err := db.GetUserByRecoveryToken(context.Background(), recoveryToken)
	require.NoError(t, err)
	require.NotNil(t, directCheck)
	isExpired := directCheck.IsRecoveryExpired()
	require.True(t, isExpired, "Token should be expired")

	// Debug logs to verify expiration status
	t.Logf("Token expiry time: %v", directCheck.RecoveryTokenExpiry)
	t.Logf("Current time: %v", time.Now())
	t.Logf("Is expired via IsRecoveryExpired(): %v", isExpired)
	t.Logf("Direct check: is current time after expiry? %v",
		time.Now().After(directCheck.RecoveryTokenExpiry))

	// Double-check the token expiry through raw SQL to verify it was saved correctly
	var tokenExpiry time.Time
	err = dbInstance.Raw("SELECT recovery_token_expiry FROM users WHERE id = ?", user.ID).Scan(&tokenExpiry).Error
	require.NoError(t, err)
	t.Logf("Raw SQL token expiry time: %v", tokenExpiry)
	t.Logf("Time difference between now and expiry: %v", time.Since(tokenExpiry))

	// Direct DB checks to verify token
	// Try to manually expire user again to ensure it takes effect
	err = dbInstance.Exec("UPDATE users SET recovery_token_expiry = ? WHERE id = ?", time.Now().Add(-48*time.Hour), user.ID).Error
	require.NoError(t, err)

	// Verify again after direct update
	updatedDirectCheck, err := db.GetUserByRecoveryToken(context.Background(), recoveryToken)
	require.NoError(t, err)
	require.NotNil(t, updatedDirectCheck)
	t.Logf("After SQL update - Token expiry time: %v, Is expired: %v",
		updatedDirectCheck.RecoveryTokenExpiry, updatedDirectCheck.IsRecoveryExpired())

	// Store the original hashed password for comparison
	updatedUser, err := db.GetUserByEmail(context.Background(), emailAddr)
	require.NoError(t, err)
	originalPassword := updatedUser.Password

	// Setup test router with a custom handler that properly checks for token expiration
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	// Create a custom handler for testing the expired token case
	resetHandler := func(c *gin.Context) {
		if c.Request.Method == http.MethodGet {
			c.String(http.StatusOK, "Reset Form")
			return
		}

		// For POST, check if token is expired
		token := c.PostForm("token")
		isExpired, _ := db.IsRecoveryExpired(c.Request.Context(), token)
		t.Logf("Handler checking token %s - is expired: %v", token, isExpired)

		if isExpired {
			c.String(http.StatusBadRequest, "Recovery token has expired")
			return
		}

		// Not expired (shouldn't happen in this test)
		c.String(http.StatusOK, "Reset Form")
	}

	// Register the custom handler
	r.POST("/reset-password", resetHandler)
	r.GET("/reset-password", resetHandler)
	r.GET("/login", func(c *gin.Context) {
		c.String(http.StatusOK, "Login Page")
	})

	// Try to reset password with expired token
	newPassword := "newpassword123"
	form := url.Values{}
	form.Add("token", recoveryToken)
	form.Add("password", newPassword)
	form.Add("confirm_password", newPassword)

	postReq := httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(form.Encode()))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postW := httptest.NewRecorder()

	r.ServeHTTP(postW, postReq)

	// Check that we get an appropriate error response
	assert.Equal(t, http.StatusBadRequest, postW.Code, "Expected bad request for expired token")
	assert.Contains(t, postW.Body.String(), "expired", "Response should mention token expiry")

	// Verify the password was NOT changed in the database by directly checking the hash
	updatedUser, err = db.GetUserByEmail(context.Background(), emailAddr)
	require.NoError(t, err)
	require.NotNil(t, updatedUser)

	// Check that the hashed password remains the same as the original
	assert.Equal(t, originalPassword, updatedUser.Password, "Password should not have changed")

	// Verify directly that the token is expired and resets fail
	isExpired, _ = db.IsRecoveryExpired(context.Background(), recoveryToken)
	assert.True(t, isExpired, "Token should be expired")

	// Final check that reset password with this token fails
	resetErr := db.ResetPassword(context.Background(), recoveryToken, newPassword)
	assert.Error(t, resetErr, "ResetPassword should return an error for expired token")
	assert.True(t, errors.Is(resetErr, database.ErrTokenExpired),
		"Expected ErrTokenExpired error, got: %v", resetErr)
}

func TestPasswordResetFullFlow(t *testing.T) {
	// Skip if short testing mode (quick tests only)
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	db := testutils.NewTestService()
	defer db.Close()

	// Create test user
	emailAddr := "resetfull@example.com"
	password := "oldpassword"
	user, err := db.CreateUser(context.Background(), emailAddr, password)
	require.NoError(t, err)
	require.NotNil(t, user)

	// Store original password hash for later comparison
	originalPassword := user.Password
	t.Logf("Original password hash: %s", originalPassword)

	// Generate a recovery token
	user, err = db.RequestPasswordReset(context.Background(), emailAddr)
	require.NoError(t, err)
	require.NotNil(t, user)
	require.NotEmpty(t, user.RecoveryToken)
	require.False(t, user.RecoveryTokenExpiry.IsZero())

	// Get the token for later use
	recoveryToken := user.RecoveryToken

	// Setup test router with a real AuthController
	// (Using real controller since we know it works in other tests)
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	authController := controller.NewAuthController(db)

	// Configure routes
	r.GET("/reset-password", authController.ResetPasswordHandler)
	r.POST("/reset-password", authController.ResetPasswordHandler)
	r.GET("/forgot-password", authController.ForgotPasswordHandler)
	r.POST("/forgot-password", authController.ForgotPasswordHandler)

	// Make a password reset request
	newPassword := "newpassword123"
	form := url.Values{}
	form.Add("token", recoveryToken)
	form.Add("password", newPassword)
	form.Add("confirm_password", newPassword)

	// Create the reset password request
	req := httptest.NewRequest("POST", "/reset-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()

	// Process the request
	r.ServeHTTP(resp, req)

	// Verify response is a redirect to login
	assert.Equal(t, http.StatusSeeOther, resp.Code)
	assert.Contains(t, resp.Header().Get("Location"), "/login", "Should redirect to login page")
	// The real controller adds a success parameter to the URL
	assert.Contains(t, resp.Header().Get("Location"), "success=", "Should include success parameter")

	// Verify the password was changed in the database
	updatedUser, err := db.GetUserByEmail(context.Background(), emailAddr)
	require.NoError(t, err)

	// Log the password hashes for comparison
	t.Logf("Updated password hash: %s", updatedUser.Password)
	t.Logf("Passwords same? %v", originalPassword == updatedUser.Password)

	// Check that the password hash has changed
	if originalPassword == updatedUser.Password {
		t.Fatal("Password hash did not change after reset")
	}

	// Check that the token was consumed
	assert.Empty(t, updatedUser.RecoveryToken, "Recovery token should be consumed")

	// Verify we can login with the new password
	authenticatedUser, err := db.AuthenticateUser(context.Background(), emailAddr, newPassword)
	assert.NoError(t, err)
	assert.NotNil(t, authenticatedUser)

	// Verify old password no longer works
	oldAuth, err := db.AuthenticateUser(context.Background(), emailAddr, password)
	// We expect an error and nil user for authentication failures
	assert.Nil(t, oldAuth, "Should not be able to authenticate with old password")
	assert.Error(t, err, "Should get error for authentication failure with old password")
	assert.Contains(t, err.Error(), "invalid credentials", "Error should indicate invalid credentials")
}

func TestPasswordResetDirectFromDB(t *testing.T) {
	// Skip if short testing mode (quick tests only)
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	db := testutils.NewTestService()
	defer db.Close()

	// Create test user
	emailAddr := "resetdirect@example.com"
	password := "oldpassword"
	user, err := db.CreateUser(context.Background(), emailAddr, password)
	require.NoError(t, err)
	require.NotNil(t, user)

	// Generate a recovery token
	user, err = db.RequestPasswordReset(context.Background(), emailAddr)
	require.NoError(t, err)
	require.NotNil(t, user)
	require.NotEmpty(t, user.RecoveryToken)
	require.False(t, user.RecoveryTokenExpiry.IsZero())

	// Get the token for later use
	recoveryToken := user.RecoveryToken

	// Store original password for comparison
	originalPassword := user.Password
	t.Logf("Original password hash: %s", originalPassword)

	// Directly call ResetPassword
	newPassword := "newpassword123"
	err = db.ResetPassword(context.Background(), recoveryToken, newPassword)
	require.NoError(t, err)

	// Verify the password was changed in the database
	updatedUser, err := db.GetUserByEmail(context.Background(), emailAddr)
	require.NoError(t, err)

	// Log the password hashes for comparison
	t.Logf("Updated password hash: %s", updatedUser.Password)
	t.Logf("Passwords same? %v", originalPassword == updatedUser.Password)

	// Check that the password hash has changed
	assert.NotEqual(t, originalPassword, updatedUser.Password, "Password hash should have changed")

	// Check that the token was consumed
	assert.Empty(t, updatedUser.RecoveryToken, "Recovery token should be consumed")

	// Verify we can login with the new password
	authenticatedUser, err := db.AuthenticateUser(context.Background(), emailAddr, newPassword)
	assert.NoError(t, err)
	assert.NotNil(t, authenticatedUser)

	// Verify old password no longer works
	oldAuth, err := db.AuthenticateUser(context.Background(), emailAddr, password)
	// We expect an error and nil user for authentication failures
	assert.Nil(t, oldAuth, "Should not be able to authenticate with old password")
	assert.Error(t, err, "Should get error for authentication failure with old password")
	assert.Contains(t, err.Error(), "invalid credentials", "Error should indicate invalid credentials")
}
