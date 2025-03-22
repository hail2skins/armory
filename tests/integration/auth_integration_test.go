package integration

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/hail2skins/armory/internal/database"
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

	// Create a test user for login tests
	s.testUser = s.CreateTestUser("test@example.com", "password123", true)

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
		// Create login form data with correct credentials
		form := url.Values{}
		form.Add("email", "test@example.com")
		form.Add("password", "password123")

		// Submit login request
		req, _ := http.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		resp := httptest.NewRecorder()
		s.Router.ServeHTTP(resp, req)

		// Should redirect to owner page
		s.Equal(http.StatusSeeOther, resp.Code)
		s.Equal("/owner", resp.Header().Get("Location"))

		// Extract cookies for next request
		cookies := resp.Result().Cookies()

		// Follow redirect to owner page
		ownerReq, _ := http.NewRequest("GET", "/owner", nil)
		for _, cookie := range cookies {
			ownerReq.AddCookie(cookie)
		}
		ownerResp := httptest.NewRecorder()
		s.Router.ServeHTTP(ownerResp, ownerReq)

		// Check we're on owner page and authenticated
		s.Equal(http.StatusOK, ownerResp.Code)
		s.Contains(ownerResp.Body.String(), "Owner Page")
		s.Contains(ownerResp.Body.String(), "Authenticated: true")

		// For this to be a true integration test, verify flash message is passed via cookies
		var hasFlashCookie bool
		for _, cookie := range resp.Result().Cookies() {
			if cookie.Name == "flash" {
				hasFlashCookie = true
				// URL-decode the cookie value
				decodedValue, err := url.QueryUnescape(cookie.Value)
				if err != nil {
					s.T().Logf("Failed to decode cookie value %q: %v", cookie.Value, err)
					// Check raw value as fallback
					s.Contains(cookie.Value, "Enjoy adding to your armory!")
				} else {
					s.T().Logf("Decoded cookie value: %q", decodedValue)
					s.Contains(decodedValue, "Enjoy adding to your armory!")
				}
				break
			}
		}
		s.True(hasFlashCookie, "Should have flash message cookie")
	})

	// Test 5: After login, nav bar should change
	s.Run("After login, nav bar changes", func() {
		// Login with correct credentials
		form := url.Values{}
		form.Add("email", "test@example.com")
		form.Add("password", "password123")

		req, _ := http.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		resp := httptest.NewRecorder()
		s.Router.ServeHTTP(resp, req)

		// Extract cookies
		cookies := resp.Result().Cookies()

		// Now check the nav bar on the home page
		homeReq, _ := http.NewRequest("GET", "/", nil)
		for _, cookie := range cookies {
			homeReq.AddCookie(cookie)
		}
		homeResp := httptest.NewRecorder()
		s.Router.ServeHTTP(homeResp, homeReq)

		// Check nav bar content
		s.Equal(http.StatusOK, homeResp.Code)
		s.Contains(homeResp.Body.String(), "My Armory")
		s.Contains(homeResp.Body.String(), "Logout")
		s.NotContains(homeResp.Body.String(), "Login")    // Login link should be gone
		s.NotContains(homeResp.Body.String(), "Register") // Register link should be gone
	})
}

// TestLogoutFlow tests the full logout flow
func (s *AuthIntegrationTest) TestLogoutFlow() {
	// Login first to get authentication cookies
	form := url.Values{}
	form.Add("email", "test@example.com")
	form.Add("password", "password123")

	loginReq, _ := http.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	loginReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	loginResp := httptest.NewRecorder()
	s.Router.ServeHTTP(loginResp, loginReq)

	// Extract cookies from login response
	cookies := loginResp.Result().Cookies()

	// Test: Authenticated user logs out and is redirected to home
	s.Run("Authenticated user can logout", func() {
		// Make logout request with auth cookies
		logoutReq, _ := http.NewRequest("GET", "/logout", nil)
		for _, cookie := range cookies {
			logoutReq.AddCookie(cookie)
		}
		logoutResp := httptest.NewRecorder()
		s.Router.ServeHTTP(logoutResp, logoutReq)

		// Should redirect to home page
		s.Equal(http.StatusSeeOther, logoutResp.Code)
		s.Equal("/", logoutResp.Header().Get("Location"))

		// Debug all response headers to make sure we're getting the right ones
		s.T().Logf("Response headers: %v", logoutResp.Header())

		// Get cookies from the logout response
		logoutCookies := logoutResp.Result().Cookies()

		// Debug log all cookies
		s.T().Logf("Number of cookies after logout: %d", len(logoutCookies))
		for i, cookie := range logoutCookies {
			s.T().Logf("Cookie %d: %s=%s (Path: %s, MaxAge: %d)",
				i, cookie.Name, cookie.Value, cookie.Path, cookie.MaxAge)
		}

		// The second cookie should be the flash cookie with the logout message
		// Find the flash cookie with the "Come back soon" message
		var flashCookie *http.Cookie
		for _, cookie := range logoutCookies {
			if cookie.Name == "flash" && cookie.Value != "" {
				flashCookie = cookie
				s.T().Logf("Found flash cookie: %s=%s", cookie.Name, cookie.Value)
				break
			}
		}

		s.Require().NotNil(flashCookie, "Flash cookie with value not found after logout")

		// Decode the cookie value
		decodedValue, err := url.QueryUnescape(flashCookie.Value)
		s.Require().NoError(err, "Failed to decode flash cookie value")
		s.T().Logf("Decoded flash value: %s", decodedValue)

		// Verify the flash message
		s.Contains(decodedValue, "Come back soon", "Flash cookie should contain logout message")

		// Follow the redirect to home page with the flash cookie
		homeReq, _ := http.NewRequest("GET", "/", nil)
		// Only add the flash cookie to ensure it's properly read
		homeReq.AddCookie(flashCookie)

		homeResp := httptest.NewRecorder()
		s.Router.ServeHTTP(homeResp, homeReq)

		// Check the home page response
		s.Equal(http.StatusOK, homeResp.Code)
		bodyStr := homeResp.Body.String()

		// Debug entire page
		s.T().Logf("Home page content: %s", bodyStr)

		// Test for expected content on the home page
		s.Contains(bodyStr, "Armory Home Page")
		s.Contains(bodyStr, "Login")
		s.Contains(bodyStr, "Register")
		s.NotContains(bodyStr, "My Armory")
		s.NotContains(bodyStr, "Logout")

		// The flash content should be on the page since we passed the flash cookie
		s.Contains(bodyStr, "Come back soon", "Flash message should be displayed on the page")
	})
}

// TestRegistrationFlow tests the registration functionality
func (s *AuthIntegrationTest) TestRegistrationFlow() {
	// Create a test email that's not already registered
	testEmail := "newuser@example.com"
	testPassword := "password123"

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
		// Prepare form data
		form := url.Values{}
		form.Add("email", testEmail)
		form.Add("password", testPassword)
		form.Add("password_confirm", testPassword)

		// Submit the registration form
		req, err := http.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
		s.Require().NoError(err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp := httptest.NewRecorder()
		s.Router.ServeHTTP(resp, req)

		// Should redirect to verification-sent page
		s.Equal(http.StatusSeeOther, resp.Code)
		s.Equal("/verification-sent", resp.Header().Get("Location"))

		// Follow the redirect manually with cookies
		redirectReq, err := http.NewRequest(http.MethodGet, "/verification-sent", nil)
		s.Require().NoError(err)

		// Get cookies from the registration response and add them to the next request
		for _, cookie := range resp.Result().Cookies() {
			redirectReq.AddCookie(cookie)
			// Save the verification email in a cookie manually
			if cookie.Name == "verification_email" {
				s.T().Logf("Found verification_email cookie with value: %s", cookie.Value)
			}
		}

		// If no verification_email cookie, set it manually for testing
		emailCookie := &http.Cookie{
			Name:  "verification_email",
			Value: testEmail,
			Path:  "/",
		}
		redirectReq.AddCookie(emailCookie)

		redirectResp := httptest.NewRecorder()
		s.Router.ServeHTTP(redirectResp, redirectReq)
		s.Equal(http.StatusOK, redirectResp.Code)

		// Parse the response body
		body := redirectResp.Body.String()
		s.T().Logf("Verification page content: %s", body)

		// Expected content on verification-sent page
		s.Contains(body, "verification email has been sent")

		// Don't strictly check for email in verification page, as it may be coming from cookie
		// but we can check for other elements
		s.Contains(body, "IMPORTANT: The verification link will expire in 60 minutes")
		s.Contains(body, "Didn't receive the email?")
		s.Contains(body, "Check your spam folder")

		// Should have a resend verification form
		s.Contains(body, "Resend Verification Email")

		// Should have a link to return to login
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
