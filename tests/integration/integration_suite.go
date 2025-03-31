package integration

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/auth"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/database/seed"
	"github.com/hail2skins/armory/internal/middleware"
	"github.com/hail2skins/armory/internal/testutils"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/hail2skins/armory/internal/testutils/testhelper"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// IntegrationSuite provides a base test suite for integration tests
type IntegrationSuite struct {
	suite.Suite
	DB                *gorm.DB
	Service           database.Service
	Helper            *testhelper.ControllerTestHelper
	Router            *gin.Engine
	MockEmail         *mocks.MockEmailService
	AuthController    *controller.AuthController
	HomeController    *controller.HomeController
	OwnerController   *controller.OwnerController
	PaymentController *controller.PaymentController
}

// SetupSuite runs once before all tests in the suite
func (s *IntegrationSuite) SetupSuite() {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Enable CSRF TestMode for integration tests
	middleware.EnableTestMode()

	// Create a test database service
	s.Service = testutils.SharedTestService()
	s.DB = s.Service.GetDB()

	// Explicitly seed the database since SharedTestService no longer does it automatically
	seed.RunSeeds(s.DB)

	// Create mock for email service only (as it's an external service)
	s.MockEmail = new(mocks.MockEmailService)

	// Create controllers with real DB service
	s.AuthController = controller.NewAuthController(s.Service)
	s.HomeController = controller.NewHomeController(s.Service)
	s.OwnerController = controller.NewOwnerController(s.Service)
	s.PaymentController = controller.NewPaymentController(s.Service)

	// Set up email service in auth controller using reflection
	s.setEmailService(s.AuthController, s.MockEmail)
}

// TearDownSuite runs once after all tests in the suite
func (s *IntegrationSuite) TearDownSuite() {
	// Disable CSRF TestMode after tests are done
	middleware.DisableTestMode()

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

	// Still set up static assets - this is needed for CSS/JS
	s.Router.Static("/assets", "../../cmd/web/static")

	// Log that we're using the real controllers
	s.T().Log("Using REAL controllers and templates for integration tests - NO MOCKS")

	// Set up the sessions middleware - THIS IS CRITICAL FOR TESTS
	store := cookie.NewStore([]byte("test-secret-key"))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400, // 1 day
		HttpOnly: true,
	})
	s.Router.Use(sessions.Sessions("armory-session", store))

	// Set up flash middleware
	s.setupFlashMiddleware()

	// Set up auth controller middleware - VERY IMPORTANT
	s.setupAuthControllerMiddleware()

	// Instead of overriding render functions with mocks, set them to use the real
	// templ components from server/auth_routes.go setupAuthRenderFunctions
	s.AuthController.RenderLogin = func(c *gin.Context, d interface{}) {
		authData := d.(data.AuthData)
		// Set authentication state
		_, authenticated := s.AuthController.GetCurrentUser(c)
		authData.Authenticated = authenticated
		// Set default title if not set
		if authData.Title == "" {
			authData.Title = "Login"
		}

		// Handle flash messages
		session := sessions.Default(c)
		if flashes := session.Flashes(); len(flashes) > 0 {
			session.Save()
			for _, flash := range flashes {
				if flashMsg, ok := flash.(string); ok {
					authData = authData.WithSuccess(flashMsg)
				}
			}
		}

		auth.Login(authData).Render(c.Request.Context(), c.Writer)
	}

	s.AuthController.RenderRegister = func(c *gin.Context, d interface{}) {
		authData := d.(data.AuthData)
		// Set authentication state
		_, authenticated := s.AuthController.GetCurrentUser(c)
		authData.Authenticated = authenticated
		// Set default title if not set
		if authData.Title == "" {
			authData.Title = "Register"
		}

		// Handle flash messages
		if flash, exists := c.Get("flash"); exists && flash != nil {
			if flashStr, ok := flash.(string); ok && flashStr != "" {
				authData = authData.WithSuccess(flashStr)
			}
		}

		auth.Register(authData).Render(c.Request.Context(), c.Writer)
	}

	s.AuthController.RenderLogout = func(c *gin.Context, d interface{}) {
		authData := d.(data.AuthData)
		// Set authentication state - should be false after logout
		authData.Authenticated = false
		// Set default title if not set
		if authData.Title == "" {
			authData.Title = "Logout"
		}

		// Handle flash messages
		if flash, exists := c.Get("flash"); exists && flash != nil {
			if flashStr, ok := flash.(string); ok && flashStr != "" {
				authData = authData.WithSuccess(flashStr)
			}
		}

		auth.Logout(authData).Render(c.Request.Context(), c.Writer)
	}

	s.AuthController.RenderVerificationSent = func(c *gin.Context, d interface{}) {
		authData := d.(data.AuthData)
		// Set authentication state
		_, authenticated := s.AuthController.GetCurrentUser(c)
		authData.Authenticated = authenticated
		// Set default title if not set
		if authData.Title == "" {
			authData.Title = "Verification Email Sent"
		}

		// Handle flash messages
		if flash, exists := c.Get("flash"); exists && flash != nil {
			if flashStr, ok := flash.(string); ok && flashStr != "" {
				authData = authData.WithSuccess(flashStr)
			}
		}

		auth.VerificationSent(authData).Render(c.Request.Context(), c.Writer)
	}

	// Add rendering functions for password reset
	s.AuthController.RenderForgotPassword = func(c *gin.Context, d interface{}) {
		authData := d.(data.AuthData)
		// Set authentication state
		_, authenticated := s.AuthController.GetCurrentUser(c)
		authData.Authenticated = authenticated
		// Set default title if not set
		if authData.Title == "" {
			authData.Title = "Reset Password"
		}

		// Handle flash messages
		if flash, exists := c.Get("flash"); exists && flash != nil {
			if flashStr, ok := flash.(string); ok && flashStr != "" {
				authData = authData.WithSuccess(flashStr)
			}
		}

		auth.ResetPasswordRequest(authData).Render(c.Request.Context(), c.Writer)
	}

	s.AuthController.RenderResetPassword = func(c *gin.Context, d interface{}) {
		authData := d.(data.AuthData)
		// Set authentication state
		_, authenticated := s.AuthController.GetCurrentUser(c)
		authData.Authenticated = authenticated
		// Set default title if not set
		if authData.Title == "" {
			authData.Title = "Set New Password"
		}

		// Handle flash messages
		if flash, exists := c.Get("flash"); exists && flash != nil {
			if flashStr, ok := flash.(string); ok && flashStr != "" {
				authData = authData.WithSuccess(flashStr)
			}
		}

		auth.ResetPassword(authData).Render(c.Request.Context(), c.Writer)
	}

	// Set auth controller in context
	s.Router.Use(func(c *gin.Context) {
		c.Set("auth", s.AuthController)
		c.Set("authController", s.AuthController)
		c.Next()
	})

	// Set up auth routes
	s.Router.GET("/login", s.AuthController.LoginHandler)
	s.Router.POST("/login", s.AuthController.LoginHandler)
	s.Router.GET("/register", s.AuthController.RegisterHandler)
	s.Router.POST("/register", s.AuthController.RegisterHandler)
	s.Router.GET("/logout", s.AuthController.LogoutHandler)
	s.Router.GET("/verification-sent", func(c *gin.Context) {
		s.AuthController.RenderVerificationSent(c, data.NewAuthData())
	})
	// Add password reset routes
	s.Router.GET("/reset-password/new", s.AuthController.ForgotPasswordHandler)
	s.Router.POST("/reset-password/new", s.AuthController.ForgotPasswordHandler)
	s.Router.GET("/reset-password", s.AuthController.ResetPasswordHandler)
	s.Router.POST("/reset-password", s.AuthController.ResetPasswordHandler)

	// Set up a home route for testing
	s.Router.GET("/", func(c *gin.Context) {
		// Call the real HomeController.HomeHandler
		s.T().Log("Using the REAL HomeController.HomeHandler for integration tests")
		s.HomeController.HomeHandler(c)
	})

	// Set up the owner page route with protection
	protected := s.Router.Group("/")
	protected.Use(s.AuthController.AuthMiddleware())
	{
		protected.GET("/owner", func(c *gin.Context) {
			// Call the real OwnerController.LandingPage
			s.T().Log("Using the REAL OwnerController.LandingPage for integration tests")
			s.OwnerController.LandingPage(c)
		})

		// Add profile routes
		protected.GET("/owner/profile", func(c *gin.Context) {
			s.T().Log("Using the REAL OwnerController.Profile for integration tests")
			s.OwnerController.Profile(c)
		})

		protected.GET("/owner/profile/edit", func(c *gin.Context) {
			s.T().Log("Using the REAL OwnerController.EditProfile for integration tests")
			s.OwnerController.EditProfile(c)
		})

		protected.POST("/owner/profile/update", func(c *gin.Context) {
			s.T().Log("Using the REAL OwnerController.UpdateProfile for integration tests")
			s.OwnerController.UpdateProfile(c)
		})

		protected.GET("/owner/profile/delete", func(c *gin.Context) {
			s.T().Log("Using the REAL OwnerController.DeleteAccountConfirm for integration tests")
			s.OwnerController.DeleteAccountConfirm(c)
		})

		protected.POST("/owner/profile/delete", func(c *gin.Context) {
			s.T().Log("Using the REAL OwnerController.DeleteAccountHandler for integration tests")
			s.OwnerController.DeleteAccountHandler(c)
		})

		protected.GET("/owner/profile/subscription", func(c *gin.Context) {
			s.T().Log("Using the REAL OwnerController.Subscription for integration tests")
			s.OwnerController.Subscription(c)
		})

		// Add the payment history route with the real controller
		protected.GET("/owner/payment-history", func(c *gin.Context) {
			s.T().Log("Using the REAL PaymentController.ShowPaymentHistory for integration tests")
			s.PaymentController.ShowPaymentHistory(c)
		})

		// Gun routes nested under owner
		ownerGuns := protected.Group("/owner/guns")
		{
			// Arsenal view - shows all guns with sorting and searching
			ownerGuns.GET("/arsenal", func(c *gin.Context) {
				s.T().Log("Using the REAL OwnerController.Arsenal for integration tests")
				s.OwnerController.Arsenal(c)
			})

			// New gun form
			ownerGuns.GET("/new", func(c *gin.Context) {
				s.T().Log("Using the REAL OwnerController.New for integration tests")
				s.OwnerController.New(c)
			})

			// Create a new gun
			ownerGuns.POST("", func(c *gin.Context) {
				s.T().Log("Using the REAL OwnerController.Create for integration tests")
				s.OwnerController.Create(c)
			})

			// Show a specific gun
			ownerGuns.GET("/:id", func(c *gin.Context) {
				s.T().Log("Using the REAL OwnerController.Show for integration tests")
				s.OwnerController.Show(c)
			})

			// Edit gun form
			ownerGuns.GET("/:id/edit", func(c *gin.Context) {
				s.T().Log("Using the REAL OwnerController.Edit for integration tests")
				s.OwnerController.Edit(c)
			})

			// Update gun
			ownerGuns.POST("/:id", func(c *gin.Context) {
				s.T().Log("Using the REAL OwnerController.Update for integration tests")
				s.OwnerController.Update(c)
			})

			// Delete gun
			ownerGuns.POST("/:id/delete", func(c *gin.Context) {
				s.T().Log("Using the REAL OwnerController.Delete for integration tests")
				s.OwnerController.Delete(c)
			})
		}
	}
}

// TearDownTest runs after each test
func (s *IntegrationSuite) TearDownTest() {
	// Reset mock expectations for external services only
	s.MockEmail.ExpectedCalls = nil

	// Clean up any test data created during tests
	// This would ideally use transactions or specific cleanup based on test markers
}

// Helper methods

// setupFlashMiddleware sets up the flash middleware for testing
func (s *IntegrationSuite) setupFlashMiddleware() {
	s.Router.Use(func(c *gin.Context) {
		// Set flash function that uses session
		c.Set("setFlash", func(msg string) {
			// Set in session
			session := sessions.Default(c)
			session.AddFlash(msg)
			if err := session.Save(); err != nil {
				s.T().Logf("Error saving session flash: %v", err)
			}

			// Always set as a cookie for test compatibility
			encodedMsg := url.QueryEscape(msg)
			http.SetCookie(c.Writer, &http.Cookie{
				Name:     "flash",
				Value:    encodedMsg,
				Path:     "/",
				MaxAge:   3600,
				HttpOnly: true,
			})

			s.T().Logf("Flash middleware: Set flash cookie '%s=%s'", "flash", encodedMsg)
		})

		// Check for flash cookies first (test priority)
		for _, cookie := range c.Request.Cookies() {
			if cookie.Name == "flash" && cookie.Value != "" {
				// URL-decode the cookie value
				decodedValue, err := url.QueryUnescape(cookie.Value)
				if err == nil {
					// Set flash value in context
					c.Set("flash", decodedValue)
					s.T().Logf("Flash middleware: Found flash cookie: %s=%s (decoded: %s)",
						cookie.Name, cookie.Value, decodedValue)

					// Clear the cookie after reading it
					http.SetCookie(c.Writer, &http.Cookie{
						Name:     "flash",
						Value:    "",
						Path:     "/",
						MaxAge:   -1,
						HttpOnly: true,
					})
				}
				break
			}
		}

		// Extract flash messages from the session as backup
		session := sessions.Default(c)
		if flashes := session.Flashes(); len(flashes) > 0 {
			if err := session.Save(); err != nil {
				s.T().Logf("Error saving session after reading flashes: %v", err)
			}

			// Set in context and cookie
			if len(flashes) > 0 {
				if flashMsg, ok := flashes[0].(string); ok {
					// Set in context
					c.Set("flash", flashMsg)

					// Also set as cookie
					encodedMsg := url.QueryEscape(flashMsg)
					http.SetCookie(c.Writer, &http.Cookie{
						Name:     "flash",
						Value:    encodedMsg,
						Path:     "/",
						MaxAge:   3600,
						HttpOnly: true,
					})

					s.T().Logf("Flash middleware: Set flash from session: %s", flashMsg)
				}
			}
		}

		c.Next()
	})
}

// setupHomeHandler configures the router to use the real home page template
func (s *IntegrationSuite) setupHomeHandler() {
	// Use the REAL HomeHandler from the HomeController, not mocked HTML
	s.Router.GET("/", s.HomeController.HomeHandler)

	s.T().Log("Using the REAL HomeController.HomeHandler for integration tests")
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

// extractCSRFToken extracts the CSRF token from a response if present
func (s *IntegrationSuite) extractCSRFToken(resp *httptest.ResponseRecorder) string {
	// Try to get the token from the response body
	body := resp.Body.String()

	// Look for the CSRF token in the HTML form
	// This is a simple approach - we're looking for name="csrf_token" value="TOKEN"
	for _, line := range strings.Split(body, "\n") {
		if strings.Contains(line, `name="csrf_token"`) {
			// Parse the value
			start := strings.Index(line, `value="`)
			if start != -1 {
				start += 7 // length of 'value="'
				end := strings.Index(line[start:], `"`)
				if end != -1 {
					return line[start : start+end]
				}
			}
		}
	}

	// If we didn't find it in the body, try the cookies
	cookies := resp.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "armory-session" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		return ""
	}

	// Create a request with this cookie
	req, _ := http.NewRequest("GET", "/", nil)
	req.AddCookie(sessionCookie)

	// Create a gin context for the request
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	// Try to get the session
	session := sessions.Default(c)
	if session != nil {
		if token := session.Get("csrf_token"); token != nil {
			if tokenStr, ok := token.(string); ok {
				return tokenStr
			}
		}
	}

	// Last resort: check if our TestMode is enabled, in which case any token will work
	if middleware.TestMode {
		return "test-csrf-token"
	}

	return ""
}

// addCSRFTokenToForm adds the CSRF token to a form
func (s *IntegrationSuite) addCSRFTokenToForm(form url.Values, token string) url.Values {
	if token != "" {
		form.Set("csrf_token", token)
	}
	return form
}

// setupAuthControllerMiddleware sets up the auth controller middleware
func (s *IntegrationSuite) setupAuthControllerMiddleware() {
	s.Router.Use(func(c *gin.Context) {
		c.Set("auth", s.AuthController)
		c.Set("authController", s.AuthController)
		c.Next()
	})
}
