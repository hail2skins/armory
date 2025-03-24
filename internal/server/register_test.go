package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/hail2skins/armory/internal/middleware"
	"github.com/hail2skins/armory/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUserRegistration(t *testing.T) {
	// Enable CSRF test mode for this test
	middleware.EnableTestMode()
	defer middleware.DisableTestMode()

	// Setup test router with mock database and email service
	router, _, mockEmail := setupTestRouter(t)

	// Set up mock expectations for email service BEFORE making the request
	mockEmail.On("SendVerificationEmail",
		"test@example.com",
		mock.AnythingOfType("string"), // Token will be generated dynamically
		mock.AnythingOfType("string"), // baseURL will be constructed from request
	).Return(nil)

	// Create registration form data with CSRF token
	form := url.Values{}
	form.Add("email", "test@example.com") // Use the email that's already mocked
	form.Add("password", "Password123!")
	form.Add("password_confirm", "Password123!")
	form.Add("csrf_token", "test-token") // Dummy token, will be bypassed in test mode

	// Create a request to register
	req, _ := http.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Test-CSRF-Bypass", "true") // Alternative way to bypass CSRF in tests

	// Create a response recorder
	resp := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(resp, req)

	// Check that we get redirected to the verification-sent page
	assert.Equal(t, http.StatusSeeOther, resp.Code)
	assert.Equal(t, "/verification-sent", resp.Header().Get("Location"))

	// Verify that the verification email was sent
	assert.True(t, mockEmail.VerificationEmailSent, "SendVerificationEmail should have been called")
	assert.Equal(t, "test@example.com", mockEmail.LastVerificationEmail, "SendVerificationEmail should have been called with the correct email")
	assert.NotEmpty(t, mockEmail.LastVerificationToken, "SendVerificationEmail should have been called with a token")
	assert.NotEmpty(t, mockEmail.LastBaseURL, "SendVerificationEmail should have been called with a baseURL")

	// Verify all mock expectations were met
	mockEmail.AssertExpectations(t)

	// Now try to login with the new user
	loginForm := url.Values{}
	loginForm.Add("email", "test@example.com")
	loginForm.Add("password", "Password123!")
	loginForm.Add("csrf_token", "test-token") // Dummy token, will be bypassed in test mode

	loginReq, _ := http.NewRequest("POST", "/login", strings.NewReader(loginForm.Encode()))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginReq.Header.Set("X-Test-CSRF-Bypass", "true") // Bypass CSRF validation in tests
	loginResp := httptest.NewRecorder()

	// Serve the login request
	router.ServeHTTP(loginResp, loginReq)

	// Check that we get redirected to the owner page
	assert.Equal(t, http.StatusSeeOther, loginResp.Code)
	assert.Equal(t, "/owner", loginResp.Header().Get("Location"))

	// Check that the auth cookie is set after login
	loginResultCookies := loginResp.Result().Cookies()
	var loginAuthCookie *http.Cookie
	for _, cookie := range loginResultCookies {
		if cookie.Name == "auth-session" {
			loginAuthCookie = cookie
			break
		}
	}
	assert.NotNil(t, loginAuthCookie, "Auth cookie should be present after login")
	assert.NotEmpty(t, loginAuthCookie.Value, "Auth cookie should have a value after login")
}

// TestRegistrationValidation tests validation of the registration form
func TestRegistrationValidation(t *testing.T) {
	// Enable CSRF test mode for this test
	middleware.EnableTestMode()
	defer middleware.DisableTestMode()

	// IMPORTANT: Use SharedTestService to avoid repeatedly seeding the database
	// The shared database is seeded only once and reused across tests
	db := testutils.SharedTestService()
	defer db.Close() // This is a no-op for shared service

	// Create a new server
	server := &Server{
		db: db,
	}
	router := server.RegisterRoutes()

	// Create a context for database operations
	ctx := context.Background()

	// First create a user with the duplicate email
	_, err := db.CreateUser(ctx, "duplicate@example.com", "Password123!")
	require.NoError(t, err)

	testCases := []struct {
		name          string
		email         string
		password      string
		confirmPass   string
		expectedCode  int
		expectedError string
	}{
		{
			name:          "Empty email",
			email:         "",
			password:      "Password123!",
			confirmPass:   "Password123!",
			expectedCode:  http.StatusOK, // Form renders with error
			expectedError: "Invalid email format",
		},
		{
			name:          "Invalid email format",
			email:         "notanemail",
			password:      "Password123!",
			confirmPass:   "Password123!",
			expectedCode:  http.StatusOK,
			expectedError: "Invalid email format",
		},
		{
			name:          "Password too short",
			email:         "test@example.com",
			password:      "pass",
			confirmPass:   "pass",
			expectedCode:  http.StatusOK,
			expectedError: "Password must be at least 8 characters",
		},
		{
			name:          "Passwords don't match",
			email:         "test@example.com",
			password:      "Password123!",
			confirmPass:   "password456",
			expectedCode:  http.StatusOK,
			expectedError: "Passwords do not match",
		},
		{
			name:          "Duplicate email",
			email:         "duplicate@example.com",
			password:      "Password123!",
			confirmPass:   "Password123!",
			expectedCode:  http.StatusOK,
			expectedError: "Email already registered",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			form := url.Values{}
			form.Add("email", tc.email)
			form.Add("password", tc.password)
			form.Add("password_confirm", tc.confirmPass)
			form.Add("csrf_token", "test-token") // Dummy token, will be bypassed in test mode

			req, _ := http.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("X-Test-CSRF-Bypass", "true") // Bypass CSRF validation in tests
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tc.expectedCode, resp.Code)
			if tc.expectedError != "" {
				assert.Contains(t, resp.Body.String(), tc.expectedError)
			}
		})
	}
}
