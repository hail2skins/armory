package integration

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/testutils"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/hail2skins/armory/internal/testutils/testhelper"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// IntegrationSuite provides a base test suite for integration tests
type IntegrationSuite struct {
	suite.Suite
	DB              *gorm.DB
	Service         database.Service
	Helper          *testhelper.ControllerTestHelper
	Router          *gin.Engine
	MockEmail       *mocks.MockEmailService
	AuthController  *controller.AuthController
	HomeController  *controller.HomeController
	OwnerController *controller.OwnerController
}

// SetupSuite runs once before all tests in the suite
func (s *IntegrationSuite) SetupSuite() {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a test database service
	s.Service = testutils.SharedTestService()
	s.DB = s.Service.GetDB()

	// Create mock for email service only (as it's an external service)
	s.MockEmail = new(mocks.MockEmailService)

	// Create controllers with real DB service
	s.AuthController = controller.NewAuthController(s.Service)
	s.HomeController = controller.NewHomeController(s.Service)
	s.OwnerController = controller.NewOwnerController(s.Service)

	// Set up email service in auth controller using reflection
	s.setEmailService(s.AuthController, s.MockEmail)
}

// TearDownSuite runs once after all tests in the suite
func (s *IntegrationSuite) TearDownSuite() {
	// Clean up database connection
	if s.Service != nil {
		s.Service.Close()
	}
}

// SetupTest runs before each test
func (s *IntegrationSuite) SetupTest() {
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
			decodedValue, decodeErr := url.QueryUnescape(cookie)
			if decodeErr == nil {
				email = decodedValue
			} else {
				email = cookie
			}
			s.T().Logf("Found verification_email cookie with value: %s", email)
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

	// Set up owner routes if needed for integration tests
	s.Router.GET("/owner/guns/arsenal", s.OwnerController.Arsenal)
}

// TearDownTest runs after each test
func (s *IntegrationSuite) TearDownTest() {
	// Reset mock expectations for external services only
	s.MockEmail.ExpectedCalls = nil

	// Clean up any test data created during tests
	// This would ideally use transactions or specific cleanup based on test markers
}

// Helper methods

// setupFlashMiddleware sets up middleware for flash messages
func (s *IntegrationSuite) setupFlashMiddleware() {
	s.Router.Use(func(c *gin.Context) {
		// Set flash function
		c.Set("setFlash", func(msg string) {
			c.SetCookie("flash", msg, 10, "/", "", false, false)
		})

		// Get flash message from cookie
		if flash, err := c.Cookie("flash"); err == nil && flash != "" {
			// Decode cookie value
			decodedFlash, err := url.QueryUnescape(flash)
			if err == nil {
				flash = decodedFlash
			}

			// Set in context and clear cookie
			c.Set("flash", flash)
			c.SetCookie("flash", "", -1, "/", "", false, false)

			// Log for debugging
			s.T().Logf("Found flash cookie: %s", flash)
		}

		c.Next()
	})
}

// setupRenderFunctions sets up render functions for testing
func (s *IntegrationSuite) setupRenderFunctions() {
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
func (s *IntegrationSuite) setupHomeHandler() {
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

		// Handle flash messages with multiple approaches for reliability
		flashMsg := ""

		// Try getting from context first
		if flash, exists := c.Get("flash"); exists {
			if flashStr, ok := flash.(string); ok && flashStr != "" {
				flashMsg = flashStr
				s.T().Logf("Found flash in context: %s", flashMsg)
			}
		}

		// If not in context, try from cookie
		if flashMsg == "" {
			if cookie, err := c.Cookie("flash"); err == nil && cookie != "" {
				// Try to decode the cookie value
				decodedValue, err := url.QueryUnescape(cookie)
				if err == nil {
					flashMsg = decodedValue
				} else {
					flashMsg = cookie
				}
				// Clear the flash cookie after using it
				c.SetCookie("flash", "", -1, "/", "", false, false)
				s.T().Logf("Found flash in cookie: %s", flashMsg)
			}
		}

		// Add flash message to page if it exists
		if flashMsg != "" {
			html += `<div class="flash">` + flashMsg + `</div>`
			s.T().Logf("Added flash message to page: %s", flashMsg)
		} else {
			s.T().Log("No flash message found to display")
		}

		html += `<p>Welcome to the Armory!</p></body></html>`
		c.String(http.StatusOK, html)
	})
}

// setEmailService sets the email service in the auth controller using reflection
func (s *IntegrationSuite) setEmailService(authController *controller.AuthController, mockEmail *mocks.MockEmailService) {
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

// CreateTestUser creates a test user in the database and returns it
func (s *IntegrationSuite) CreateTestUser(email, password string, verified bool) *database.User {
	user := &database.User{
		Email:    email,
		Password: password, // Will be hashed by BeforeCreate hook
		Verified: verified,
	}

	err := s.DB.Create(user).Error
	s.Require().NoError(err, "Failed to create test user")

	return user
}

// CleanupTestUser removes a test user from the database
func (s *IntegrationSuite) CleanupTestUser(user *database.User) {
	if user == nil || user.ID == 0 {
		return
	}

	err := s.DB.Unscoped().Delete(user).Error
	s.NoError(err, "Failed to clean up test user")
}

// MakeRequest is a helper to make HTTP requests to the test server
func (s *IntegrationSuite) MakeRequest(method, path string, body io.Reader) *http.Response {
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
