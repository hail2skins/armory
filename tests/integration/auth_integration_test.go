package integration

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/middleware"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// AuthIntegrationTest extends the base IntegrationSuite
type AuthIntegrationTest struct {
	IntegrationSuite
	testUser *database.User
}

// SetupTest runs before each test in the suite
func (s *AuthIntegrationTest) SetupTest() {
	// Call the parent SetupTest
	s.IntegrationSuite.SetupTest()

	// Enable CSRF test mode for this test suite
	middleware.EnableTestMode()

	// Create a test user for login tests
	s.testUser = s.CreateTestUser("test@example.com", "Password123!", true)

	// Set up email mock with wildcards for any token
	s.MockEmail.On("SendVerificationEmail",
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string")).Return(nil)
}

// TearDownTest runs after each test in the suite
func (s *AuthIntegrationTest) TearDownTest() {
	// Clean up test user
	s.CleanupTestUser(s.testUser)

	// Disable CSRF test mode
	middleware.DisableTestMode()

	// Call the parent TearDownTest
	s.IntegrationSuite.TearDownTest()
}

// TestLoginFlow tests the full login flow
func (s *AuthIntegrationTest) TestLoginFlow() {
	// Test 1: Unauthenticated user sees login link in navbar
	s.Run("Unauthenticated user sees login link in navbar", func() {
		req, _ := http.NewRequest("GET", "/", nil)
		resp := httptest.NewRecorder()
		s.Router.ServeHTTP(resp, req)

		s.Equal(http.StatusOK, resp.Code)
		s.Contains(resp.Body.String(), "Login") // Nav bar should contain Login link
	})

	// Test 2: User can access the login page
	s.Run("User can access login page", func() {
		req, _ := http.NewRequest("GET", "/login", nil)
		resp := httptest.NewRecorder()
		s.Router.ServeHTTP(resp, req)

		s.Equal(http.StatusOK, resp.Code)
		s.Contains(resp.Body.String(), "Login")
		s.Contains(resp.Body.String(), "Email")
		s.Contains(resp.Body.String(), "Password")
		s.Contains(resp.Body.String(), "Reset Password") // Reset password link
	})

	// Test 3: Failed login shows error flash message
	s.Run("Failed login shows error message", func() {
		// Create login form data with wrong credentials
		form := url.Values{}
		form.Add("email", "wrong@example.com")
		form.Add("password", "wrongpassword")

		// Submit login request
		req, _ := http.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		resp := httptest.NewRecorder()
		s.Router.ServeHTTP(resp, req)

		// Check response contains error message
		s.Equal(http.StatusOK, resp.Code) // Form reloads with error
		s.Contains(resp.Body.String(), "Invalid email or password")
	})

	// Test 4: Successful login redirects to owner page with welcome flash
	s.Run("Successful login redirects to owner page with welcome flash", func() {
		// Use our helper to login
		cookies := s.LoginUser("test@example.com", "Password123!")

		// Follow redirect to owner page
		ownerResp := s.MakeAuthenticatedRequest("GET", "/owner", cookies)

		// Check we're on owner page and see expected content
		s.Equal(http.StatusOK, ownerResp.Code)
		s.Contains(ownerResp.Body.String(), "Welcome to Your Virtual Armory")
		s.Contains(ownerResp.Body.String(), "My Armory")
	})

	// Test 5: After login, nav bar should change to show real UI elements
	s.Run("After login, nav bar shows correct authenticated elements", func() {
		// Use our helper to login
		cookies := s.LoginUser("test@example.com", "Password123!")

		// Check the nav bar on the home page
		homeResp := s.MakeAuthenticatedRequest("GET", "/", cookies)
		s.Equal(http.StatusOK, homeResp.Code)

		// Check for authenticated nav elements
		homeContent := homeResp.Body.String()
		s.Contains(homeContent, "My Armory", "Nav bar should contain My Armory for authenticated user")
		s.Contains(homeContent, "Logout", "Nav bar should contain Logout link")
		s.NotContains(homeContent, "Login", "Nav bar should not contain Login link")
		s.NotContains(homeContent, "Register", "Nav bar should not contain Register link")
	})
}

// TestLogoutFlow tests the full logout flow
func (s *AuthIntegrationTest) TestLogoutFlow() {
	// Login using our helper
	cookies := s.LoginUser("test@example.com", "Password123!")

	// Make a direct request to /logout
	logoutReq, _ := http.NewRequest("GET", "/logout", nil)
	for _, cookie := range cookies {
		logoutReq.AddCookie(cookie)
	}
	logoutResp := httptest.NewRecorder()
	s.Router.ServeHTTP(logoutResp, logoutReq)

	// Verify we see the logout.templ content
	s.Equal(http.StatusOK, logoutResp.Code)
	s.Contains(logoutResp.Body.String(), "You have been logged out")

	// Verify the nav bar no longer shows authenticated elements
	s.Contains(logoutResp.Body.String(), "Login")
	s.Contains(logoutResp.Body.String(), "Register")
	s.NotContains(logoutResp.Body.String(), "My Armory")

	// Verify we can't access protected pages after logout
	ownerReq, _ := http.NewRequest("GET", "/owner", nil)
	// Use cookies from the logout response
	for _, cookie := range logoutResp.Result().Cookies() {
		ownerReq.AddCookie(cookie)
	}
	ownerResp := httptest.NewRecorder()
	s.Router.ServeHTTP(ownerResp, ownerReq)

	// Should redirect to login
	s.Equal(http.StatusSeeOther, ownerResp.Code)
	s.Equal("/login", ownerResp.Header().Get("Location"))
}

// TestRegistrationFlow tests the registration functionality
func (s *AuthIntegrationTest) TestRegistrationFlow() {
	// Create a test email that's not already registered
	testEmail := "newuser@example.com"
	testPassword := "Password123!"

	s.Run("Registration Page UI Elements", func() {
		// Request the register page
		resp := s.MakeRequest(http.MethodGet, "/register", nil)
		defer resp.Body.Close()

		// Check response status
		s.Equal(http.StatusOK, resp.StatusCode)

		// Parse the response body
		body, err := io.ReadAll(resp.Body)
		s.NoError(err)
		bodyStr := string(body)

		// Check for expected content
		s.Contains(bodyStr, "Create Account")
		s.Contains(bodyStr, "Already have an account?")
		s.Contains(bodyStr, "/login")

		// Check for form fields
		s.Contains(bodyStr, "Email")
		s.Contains(bodyStr, "Password")
		s.Contains(bodyStr, "Confirm Password")
	})

	s.Run("Successful Registration", func() {
		// First get the registration page to get CSRF token
		regPageReq, _ := http.NewRequest(http.MethodGet, "/register", nil)
		regPageResp := httptest.NewRecorder()
		s.Router.ServeHTTP(regPageResp, regPageReq)

		// Extract CSRF token if present
		csrfToken := s.extractCSRFToken(regPageResp)

		// Prepare form data
		form := url.Values{}
		form.Add("email", testEmail)
		form.Add("password", testPassword)
		form.Add("password_confirm", testPassword)

		// Add CSRF token if we have one
		if csrfToken != "" {
			form.Add("csrf_token", csrfToken)
		}

		// Get the session cookie from the registration page
		var sessionCookie *http.Cookie
		for _, cookie := range regPageResp.Result().Cookies() {
			if cookie.Name == "armory-session" {
				sessionCookie = cookie
				break
			}
		}

		// Submit the registration form
		req, err := http.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
		s.Require().NoError(err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		// Add the session cookie to maintain session across requests
		if sessionCookie != nil {
			req.AddCookie(sessionCookie)
		}

		resp := httptest.NewRecorder()
		s.Router.ServeHTTP(resp, req)

		// Should redirect to verification-sent page
		s.Equal(http.StatusSeeOther, resp.Code)
		s.Equal("/verification-sent", resp.Header().Get("Location"))

		// Follow the redirect manually with session cookie
		redirectReq, err := http.NewRequest(http.MethodGet, "/verification-sent", nil)
		s.Require().NoError(err)

		// Get session cookie from registration response
		var newSessionCookie *http.Cookie
		for _, cookie := range resp.Result().Cookies() {
			if cookie.Name == "armory-session" {
				newSessionCookie = cookie
				break
			}
		}
		s.NotNil(newSessionCookie, "Session cookie should be present")
		redirectReq.AddCookie(newSessionCookie)

		redirectResp := httptest.NewRecorder()
		s.Router.ServeHTTP(redirectResp, redirectReq)
		s.Equal(http.StatusOK, redirectResp.Code)

		// Parse the response body
		body := redirectResp.Body.String()
		s.T().Logf("Verification page content: %s", body)

		// Expected content on verification-sent page
		s.Contains(body, "verification email has been sent")
		s.Contains(body, "Important:</span> The verification link will expire in 60 minutes")
		s.Contains(body, "Didn't receive the email? Check your spam folder or request a new verification email")
		s.Contains(body, "Resend Verification Email")
		s.Contains(body, "Return to Login")

		// Clean up the created user
		var createdUser database.User
		result := s.DB.Where("email = ?", testEmail).First(&createdUser)
		if result.Error == nil {
			s.DB.Unscoped().Delete(&createdUser)
		}
	})

	s.Run("Go to Login from Verification Page", func() {
		// Get the verification page first - no manual cookie setting
		req, err := http.NewRequest(http.MethodGet, "/verification-sent", nil)
		s.Require().NoError(err)
		resp := httptest.NewRecorder()
		s.Router.ServeHTTP(resp, req)

		s.Equal(http.StatusOK, resp.Code)

		// Parse the body to find the login link
		bodyStr := resp.Body.String()

		// Verify login link exists
		s.Contains(bodyStr, "Return to Login")
		s.Contains(bodyStr, "/login")

		// Now follow the login link
		req, err = http.NewRequest(http.MethodGet, "/login", nil)
		s.Require().NoError(err)
		resp = httptest.NewRecorder()
		s.Router.ServeHTTP(resp, req)

		s.Equal(http.StatusOK, resp.Code)

		// Verify we're at the login page
		bodyStr = resp.Body.String()
		s.Contains(bodyStr, "Login")
	})
}

// TestAuthIntegration runs the auth integration test suite
func TestAuthIntegration(t *testing.T) {
	suite.Run(t, new(AuthIntegrationTest))
}
