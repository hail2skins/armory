package integration

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/testutils"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/hail2skins/armory/internal/testutils/testhelper"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// AuthIntegrationSuite provides a test suite for auth-related integration tests
type AuthIntegrationSuite struct {
	suite.Suite
	DB              *gorm.DB
	Service         database.Service
	Helper          *testhelper.ControllerTestHelper
	Router          *gin.Engine
	MockDB          *mocks.MockDB
	MockEmail       *mocks.MockEmailService
	AuthController  *controller.AuthController
	HomeController  *controller.HomeController
	OwnerController *controller.OwnerController
}

// SetupSuite runs once before all tests in the suite
func (s *AuthIntegrationSuite) SetupSuite() {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a test database service
	s.Service = testutils.SharedTestService()
	s.DB = s.Service.GetDB()

	// Initialize mock DB for controller interactions
	s.MockDB = new(mocks.MockDB)
	s.MockEmail = new(mocks.MockEmailService)

	// Create controllers with mocks
	s.AuthController = controller.NewAuthController(s.MockDB)
	s.HomeController = controller.NewHomeController(s.MockDB)
	s.OwnerController = controller.NewOwnerController(s.MockDB)

	// Set up email service in auth controller using reflection
	s.setEmailService(s.AuthController, s.MockEmail)
}

// TearDownSuite runs once after all tests in the suite
func (s *AuthIntegrationSuite) TearDownSuite() {
	// Clean up database connection
	if s.Service != nil {
		s.Service.Close()
	}
}

// SetupTest runs before each test
func (s *AuthIntegrationSuite) SetupTest() {
	// Create a fresh router for each test
	s.Router = gin.New()
	s.Router.Use(gin.Recovery())

	// Set up flash middleware
	s.setupFlashMiddleware()

	// Configure the auth controller with render functions
	s.setupRenderFunctions()

	// Set auth controller in context
	s.Router.Use(func(c *gin.Context) {
		c.Set("auth", s.AuthController)
		c.Set("authController", s.AuthController)
		c.Next()
	})

	// Set up routes
	s.setupHomeHandler() // Custom home handler for testing
	s.Router.GET("/login", s.AuthController.LoginHandler)
	s.Router.POST("/login", s.AuthController.LoginHandler)
	s.Router.GET("/register", s.AuthController.RegisterHandler)
	s.Router.POST("/register", s.AuthController.RegisterHandler)
	s.Router.GET("/verification-sent", func(c *gin.Context) {
		// Get email from cookie
		email := ""
		if cookie, err := c.Cookie("verification_email"); err == nil {
			email = cookie
		}

		// Create data with email
		authData := data.NewAuthData().WithTitle("Verification Email Sent")
		authData.Email = email

		// Call the render function directly
		s.AuthController.RenderVerificationSent(c, authData)
	})
	s.Router.GET("/resend-verification", s.AuthController.ResendVerificationHandler)
	s.Router.POST("/resend-verification", s.AuthController.ResendVerificationHandler)
	s.Router.GET("/logout", s.AuthController.LogoutHandler)
	s.Router.GET("/owner", func(c *gin.Context) {
		// Simple mock of the owner page
		authenticated := s.AuthController.IsAuthenticated(c)
		c.String(http.StatusOK, "Owner Page - Authenticated: %v", authenticated)
	})
}

// TearDownTest runs after each test
func (s *AuthIntegrationSuite) TearDownTest() {
	// Reset mock expectations
	s.MockDB.ExpectedCalls = nil
	s.MockEmail.ExpectedCalls = nil
}

// TestLoginFlow tests the full login flow
func (s *AuthIntegrationSuite) TestLoginFlow() {
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
		// Setup mock for failed authentication
		s.MockDB.On("AuthenticateUser", mock.Anything, "wrong@example.com", "wrongpassword").
			Return(nil, database.ErrInvalidCredentials).Once()

		// Create login form data
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
		// Create a verified user
		mockUser := &database.User{
			Email:    "test@example.com",
			Verified: true,
		}
		s.MockDB.On("AuthenticateUser", mock.Anything, "test@example.com", "password123").
			Return(mockUser, nil).Once()

		// Create login form data
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
		// First we need to login to get cookies
		mockUser := &database.User{
			Email:    "test@example.com",
			Verified: true,
		}
		s.MockDB.On("AuthenticateUser", mock.Anything, "test@example.com", "password123").
			Return(mockUser, nil).Once()

		// Login
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
func (s *AuthIntegrationSuite) TestLogoutFlow() {
	// First authenticate the user to get valid session cookies
	mockUser := &database.User{
		Email:    "test@example.com",
		Verified: true,
	}
	s.MockDB.On("AuthenticateUser", mock.Anything, "test@example.com", "password123").
		Return(mockUser, nil).Once()

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

		// Extract cookies from logout response
		logoutCookies := logoutResp.Result().Cookies()

		// Check for flash message cookie
		var hasFlashCookie bool
		for _, cookie := range logoutCookies {
			if cookie.Name == "flash" {
				hasFlashCookie = true
				decodedValue, err := url.QueryUnescape(cookie.Value)
				if err != nil {
					s.T().Logf("Failed to decode cookie value %q: %v", cookie.Value, err)
					s.Contains(cookie.Value, "Come back soon")
				} else {
					s.T().Logf("Decoded cookie value: %q", decodedValue)
					s.Contains(decodedValue, "Come back soon")
				}
				break
			}
		}
		s.True(hasFlashCookie, "Should have flash message cookie")

		// Follow redirect to home page
		homeReq, _ := http.NewRequest("GET", "/", nil)
		for _, cookie := range logoutCookies {
			homeReq.AddCookie(cookie)
		}
		homeResp := httptest.NewRecorder()
		s.Router.ServeHTTP(homeResp, homeReq)

		// Check we're on home page with correct nav bar
		s.Equal(http.StatusOK, homeResp.Code)
		s.Contains(homeResp.Body.String(), "Armory Home Page")
		s.Contains(homeResp.Body.String(), "Login")
		s.Contains(homeResp.Body.String(), "Register")
		s.NotContains(homeResp.Body.String(), "My Armory")
		s.NotContains(homeResp.Body.String(), "Logout")

		// Flash message should be displayed
		s.Contains(homeResp.Body.String(), "Come back soon")
	})
}

// TestRegistrationFlow tests the registration functionality
func (s *AuthIntegrationSuite) TestRegistrationFlow() {
	// Create a test email that's not already registered
	testEmail := "newuser@example.com"
	testPassword := "password123"

	// Setup SkipUnscopedChecks to avoid GetDB calls
	s.AuthController.SkipUnscopedChecks = true

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
		// Setup mocks for the registration process
		s.MockDB.On("GetUserByEmail", mock.Anything, testEmail).
			Return(nil, nil).Once()

		// When creating user, return a mock user with ID and verification token
		mockUser := &database.User{
			Email:              testEmail,
			VerificationToken:  "test-token",
			VerificationSentAt: time.Now(),
		}
		s.MockDB.On("CreateUser", mock.Anything, testEmail, testPassword).
			Return(mockUser, nil).Once()

		// Mock updating user after token generation
		s.MockDB.On("UpdateUser", mock.Anything, mock.AnythingOfType("*database.User")).
			Return(nil).Once()

		// Mock email sending
		s.MockEmail.On("SendVerificationEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
			Return(nil).Once()

		// Mock any promotion service calls (optional)
		s.MockDB.On("GetPromotionService").Return(nil).Maybe()

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

		// Follow the redirect naturally without artificially adding cookies
		redirectReq, err := http.NewRequest(http.MethodGet, "/verification-sent", nil)
		s.Require().NoError(err)

		// Get cookies from the registration response and add them to the next request
		for _, cookie := range resp.Result().Cookies() {
			redirectReq.AddCookie(cookie)
		}

		redirectResp := httptest.NewRecorder()
		s.Router.ServeHTTP(redirectResp, redirectReq)
		s.Equal(http.StatusOK, redirectResp.Code)

		// Parse the response body
		body := redirectResp.Body.String()

		// Expected content on verification-sent page
		s.Contains(body, "verification email has been sent")

		// Test for the CORRECT pattern with the email included (not the bug pattern)
		s.Contains(body, testEmail, "Email should be displayed in the verification message")

		s.Contains(body, "IMPORTANT: The verification link will expire in 60 minutes")
		s.Contains(body, "Didn't receive the email?")
		s.Contains(body, "Check your spam folder")

		// Should have a resend verification form
		s.Contains(body, "Resend Verification Email")

		// Should have a link to return to login
		s.Contains(body, "Return to Login")
	})

	s.Run("Resend Verification", func() {
		// Mock user for resend verification
		mockUser := &database.User{
			Email:              testEmail,
			VerificationToken:  "test-token",
			VerificationSentAt: time.Now(),
			Verified:           false,
		}

		// Mock GetUserByEmail for resend verification
		s.MockDB.On("GetUserByEmail", mock.Anything, testEmail).
			Return(mockUser, nil).Once()

		// Mock update user for new token generation
		s.MockDB.On("UpdateUser", mock.Anything, mock.AnythingOfType("*database.User")).
			Return(nil).Once()

		// Mock email sending for resend
		s.MockEmail.On("SendVerificationEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
			Return(nil).Once()

		// Prepare resend form
		form := url.Values{}
		form.Add("email", testEmail)

		// Submit the resend form
		req, err := http.NewRequest(http.MethodPost, "/resend-verification", strings.NewReader(form.Encode()))
		s.Require().NoError(err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp := httptest.NewRecorder()
		s.Router.ServeHTTP(resp, req)

		// The resend verification handler renders the result directly with 200 OK
		s.Equal(http.StatusOK, resp.Code, "Resend verification should return 200 OK")

		// Check the response body contains the success message
		bodyStr := resp.Body.String()
		s.Contains(bodyStr, "A new verification email has been sent",
			"Response should contain success message")

		// Check for the email address in the response (resend should show the email)
		s.Contains(bodyStr, testEmail,
			"Response should contain the email address")
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

// Helper methods

// setupFlashMiddleware sets up middleware for flash messages
func (s *AuthIntegrationSuite) setupFlashMiddleware() {
	s.Router.Use(func(c *gin.Context) {
		c.Set("setFlash", func(msg string) {
			c.SetCookie("flash", msg, 10, "/", "", false, false)
		})
		c.Next()
	})
}

// setupRenderFunctions sets up render functions for testing
func (s *AuthIntegrationSuite) setupRenderFunctions() {
	// Login render function
	s.AuthController.RenderLogin = func(c *gin.Context, d interface{}) {
		// Try to get the error message
		var errorMsg string

		// Try multiple ways to extract the error message
		if authData, ok := d.(data.AuthData); ok && authData.Error != "" {
			errorMsg = authData.Error
		} else if data, ok := d.(interface{ GetError() string }); ok && data.GetError() != "" {
			errorMsg = data.GetError()
		} else {
			// Try to access Error field using reflection
			v := reflect.ValueOf(d)
			if v.Kind() == reflect.Struct {
				errorField := v.FieldByName("Error")
				if errorField.IsValid() && errorField.Kind() == reflect.String {
					errorMsg = errorField.String()
				}
			}
		}

		// Render a basic HTML page
		c.Header("Content-Type", "text/html")
		html := `<!DOCTYPE html><html><body><nav>`

		// Add nav elements based on authentication status
		if s.AuthController.IsAuthenticated(c) {
			html += `<a href="/owner">My Armory</a> <a href="/logout">Logout</a>`
		} else {
			html += `<a href="/login">Login</a> <a href="/register">Register</a>`
		}

		html += `</nav><h1>Login</h1>`

		// Add error message if there is one
		if errorMsg != "" {
			html += `<div class="error">` + errorMsg + `</div>`
		}

		// The login form
		html += `<form method="post">
			<label for="email">Email</label>
			<input type="email" name="email" id="email" />
			<label for="password">Password</label>
			<input type="password" name="password" id="password" />
			<button type="submit">Login</button>
		</form>
		<a href="/reset-password">Reset Password</a>
		<p>Don't have an account? <a href="/register">Register</a></p>
		</body></html>`

		c.String(http.StatusOK, html)
	}

	// Register render function
	s.AuthController.RenderRegister = func(c *gin.Context, d interface{}) {
		// Extract error message if any
		var errorMsg string
		if authData, ok := d.(data.AuthData); ok && authData.Error != "" {
			errorMsg = authData.Error
		}

		// Render a basic HTML page
		c.Header("Content-Type", "text/html")
		html := `<!DOCTYPE html><html><body><nav>`

		// Add nav elements based on authentication status
		if s.AuthController.IsAuthenticated(c) {
			html += `<a href="/owner">My Armory</a> <a href="/logout">Logout</a>`
		} else {
			html += `<a href="/login">Login</a> <a href="/register">Register</a>`
		}

		html += `</nav><h1>Create Account</h1>`

		// Add error message if there is one
		if errorMsg != "" {
			html += `<div class="error">` + errorMsg + `</div>`
		}

		// The registration form
		html += `<form method="post">
			<label for="email">Email</label>
			<input type="email" name="email" id="email" />
			<label for="password">Password</label>
			<input type="password" name="password" id="password" />
			<label for="password_confirm">Confirm Password</label>
			<input type="password" name="password_confirm" id="password_confirm" />
			<button type="submit">Register</button>
		</form>
		<p>Already have an account? <a href="/login">Login</a></p>
		</body></html>`

		c.String(http.StatusOK, html)
	}

	// Verification sent render function
	s.AuthController.RenderVerificationSent = func(c *gin.Context, d interface{}) {
		// Extract data if any
		var email, errorMsg, successMsg string
		if authData, ok := d.(data.AuthData); ok {
			email = authData.Email
			errorMsg = authData.Error
			successMsg = authData.Success
		}

		// Render a basic HTML page
		c.Header("Content-Type", "text/html")
		html := `<!DOCTYPE html><html><body><nav>`

		// Add nav elements based on authentication status
		if s.AuthController.IsAuthenticated(c) {
			html += `<a href="/owner">My Armory</a> <a href="/logout">Logout</a>`
		} else {
			html += `<a href="/login">Login</a> <a href="/register">Register</a>`
		}

		html += `</nav><h1>Verification Email Sent</h1>`

		// Add error message if there is one
		if errorMsg != "" {
			html += `<div class="error">` + errorMsg + `</div>`
		}

		// Add success message if there is one
		if successMsg != "" {
			html += `<div class="success">` + successMsg + `</div>`
		}

		// The verification sent content
		html += `<p>A verification email has been sent to ` + email + `</p>
		<p>IMPORTANT: The verification link will expire in 60 minutes.</p>
		<p>Didn't receive the email? Check your spam folder or request a new verification email.</p>
		<form method="post" action="/resend-verification">
			<input type="hidden" name="email" value="` + email + `" />
			<button type="submit">Resend Verification Email</button>
		</form>
		<p><a href="/login">Return to Login</a></p>
		</body></html>`

		c.String(http.StatusOK, html)
	}

	// Logout render function (simplified)
	s.AuthController.RenderLogout = func(c *gin.Context, d interface{}) {
		c.Status(http.StatusOK)
		c.String(http.StatusOK, "Logout Page")
	}
}

// setupHomeHandler adds a custom home handler
func (s *AuthIntegrationSuite) setupHomeHandler() {
	s.Router.GET("/", func(c *gin.Context) {
		// Create a simple home page that changes based on auth status
		c.Header("Content-Type", "text/html")
		html := `<!DOCTYPE html><html><body><nav>`

		// Add nav elements based on authentication status
		if s.AuthController.IsAuthenticated(c) {
			html += `<a href="/owner">My Armory</a> <a href="/logout">Logout</a>`
		} else {
			html += `<a href="/login">Login</a> <a href="/register">Register</a>`
		}

		html += `</nav><h1>Armory Home Page</h1>`

		// Check for flash messages from cookie
		if flashMsg, err := c.Cookie("flash"); err == nil && flashMsg != "" {
			html += `<div class="flash">` + flashMsg + `</div>`
			// Clear the flash cookie after using it
			c.SetCookie("flash", "", -1, "/", "", false, false)
		}

		html += `<p>Welcome to the Armory!</p></body></html>`
		c.String(http.StatusOK, html)
	})
}

// setEmailService sets the email service in the auth controller using reflection
func (s *AuthIntegrationSuite) setEmailService(authController *controller.AuthController, mockEmail *mocks.MockEmailService) {
	// Use reflection to access and set the private emailService field
	val := reflect.ValueOf(authController).Elem()
	field := val.FieldByName("emailService")
	if field.IsValid() && field.CanSet() {
		field.Set(reflect.ValueOf(mockEmail))
		s.T().Log("Successfully set mockEmail service via reflection")
	} else {
		// Try the public method if reflection fails
		authController.SetEmailService(mockEmail)
		s.T().Log("Using SetEmailService method instead of reflection")
	}
}

// MakeRequest is a helper to make HTTP requests to the test server
func (s *AuthIntegrationSuite) MakeRequest(method, path string, body io.Reader) *http.Response {
	// Create a request
	req, err := http.NewRequest(method, path, body)
	s.Require().NoError(err)

	// Set appropriate headers for form submissions
	if method == http.MethodPost && body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	// Create a response recorder
	w := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(w, req)

	// Return the response
	return w.Result()
}

// TestAuthIntegrationSuite runs the suite
func TestAuthIntegrationSuite(t *testing.T) {
	suite.Run(t, new(AuthIntegrationSuite))
}
