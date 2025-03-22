package tests

import (
	"reflect"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/suite"
)

// mockAuthData is a simple struct that mimics the AuthData struct
// used in the real application for testing purposes
type mockAuthData struct {
	Error   string
	Success string
}

func (m mockAuthData) GetError() string {
	return m.Error
}

func (m mockAuthData) GetSuccess() string {
	return m.Success
}

// Helper function to set a private field using reflection
// This allows us to set private fields for testing without modifying production code
func setField(obj interface{}, fieldName string, value interface{}) {
	field := reflect.ValueOf(obj).Elem().FieldByName(fieldName)
	if field.IsValid() && field.CanSet() {
		field.Set(reflect.ValueOf(value))
	}
}

// ControllerTestSuite is a base test suite for controller tests
type ControllerTestSuite struct {
	suite.Suite
	Router      *gin.Engine
	MockDB      *mocks.MockDB
	MockAuth    *mocks.MockAuthController
	MockEmail   *mocks.MockEmailService
	Controllers map[string]interface{}
}

// SetupSuite sets up the test suite
func (s *ControllerTestSuite) SetupSuite() {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
}

// SetupTest sets up each test
func (s *ControllerTestSuite) SetupTest() {
	// Create mocks
	s.MockDB = new(mocks.MockDB)
	s.MockAuth = new(mocks.MockAuthController)
	s.MockEmail = new(mocks.MockEmailService)

	// Create a router
	s.Router = gin.New()

	// Set up session middleware for tests
	store := cookie.NewStore([]byte("test-secret-key"))
	s.Router.Use(sessions.Sessions("auth-session", store))

	// Set up middleware - include both 'auth' and 'authController' keys since different controllers use different keys
	s.Router.Use(func(c *gin.Context) {
		c.Set("auth", s.MockAuth)
		c.Set("authController", s.MockAuth)
		c.Next()
	})

	// Initialize controllers map
	s.Controllers = make(map[string]interface{})
}

// CreateAuthController creates and returns an AuthController
func (s *ControllerTestSuite) CreateAuthController() *controller.AuthController {
	if auth, ok := s.Controllers["auth"]; ok {
		return auth.(*controller.AuthController)
	}

	auth := controller.NewAuthController(s.MockDB)

	// Set up the email service
	setField(auth, "emailService", s.MockEmail)

	// Set up mock render functions that provide enough content to test against
	mockLoginRenderer := func(c *gin.Context, data interface{}) {
		// Look for error messages in the data
		errorMsg := ""

		// Try to extract error message using our interface
		if authData, ok := data.(interface{ GetError() string }); ok && authData.GetError() != "" {
			errorMsg = authData.GetError()
		} else {
			// Try to extract using reflection as fallback
			dataVal := reflect.ValueOf(data)
			if dataVal.Kind() == reflect.Struct {
				errorField := dataVal.FieldByName("Error")
				if errorField.IsValid() && errorField.Kind() == reflect.String {
					errorMsg = errorField.String()
				}
			}
		}

		// For login errors, we'll set a 401 status for certain error messages
		if errorMsg == "Invalid email or password" {
			c.Status(401)
		}

		// Render a basic login form with any error message
		c.Header("Content-Type", "text/html")
		html := `<html><body>
			<h1>Login</h1>`

		if errorMsg != "" {
			html += `<div class="error">` + errorMsg + `</div>`
		}

		html += `<form>
				<label>Email</label>
				<input type="email" name="email" />
				<label>Password</label>
				<input type="password" name="password" />
				<button type="submit">Submit</button>
			</form>
		</body></html>`

		c.String(c.Writer.Status(), html)
	}

	mockRegisterRenderer := func(c *gin.Context, data interface{}) {
		// Look for error messages in the data
		errorMsg := ""

		// Try to extract error message using our interface
		if authData, ok := data.(interface{ GetError() string }); ok && authData.GetError() != "" {
			errorMsg = authData.GetError()
		} else {
			// Try to extract using reflection as fallback
			dataVal := reflect.ValueOf(data)
			if dataVal.Kind() == reflect.Struct {
				errorField := dataVal.FieldByName("Error")
				if errorField.IsValid() && errorField.Kind() == reflect.String {
					errorMsg = errorField.String()
				}
			}
		}

		c.Header("Content-Type", "text/html")
		html := `<html><body>
			<h1>Register</h1>`

		if errorMsg != "" {
			html += `<div class="error">` + errorMsg + `</div>`
		}

		html += `<form>
				<label>Email</label>
				<input type="email" name="email" />
				<label>Password</label>
				<input type="password" name="password" />
				<label>Confirm Password</label>
				<input type="password" name="confirm_password" />
				<button type="submit">Submit</button>
			</form>
		</body></html>`

		c.String(200, html)
	}

	// Set all the render functions
	setField(auth, "RenderLogin", mockLoginRenderer)
	setField(auth, "RenderRegister", mockRegisterRenderer)

	// For now, set other renderers to simple functions that provide recognizable output
	mockSuccessRenderer := func(c *gin.Context, data interface{}) {
		message := "Success"
		// Try to extract success message if possible
		if authData, ok := data.(interface{ GetSuccess() string }); ok && authData.GetSuccess() != "" {
			message = authData.GetSuccess()
		}
		c.String(200, `<html><body><div class="success">`+message+`</div></body></html>`)
	}

	setField(auth, "RenderVerifyEmail", mockSuccessRenderer)
	setField(auth, "RenderForgotPassword", mockSuccessRenderer)
	setField(auth, "RenderResetPassword", mockSuccessRenderer)
	setField(auth, "RenderVerificationSent", mockSuccessRenderer)
	setField(auth, "RenderLogout", mockSuccessRenderer)

	s.Controllers["auth"] = auth
	return auth
}

// CreateHomeController creates and returns a HomeController
func (s *ControllerTestSuite) CreateHomeController() *controller.HomeController {
	if home, ok := s.Controllers["home"]; ok {
		return home.(*controller.HomeController)
	}

	// Create a new HomeController with our mock DB
	home := controller.NewHomeController(s.MockDB)

	// IMPORTANT: Replace the email service with our mock
	// This is crucial because NewHomeController creates a real email service
	home.SetEmailService(s.MockEmail)

	s.Controllers["home"] = home
	return home
}

// TearDownTest cleans up after each test
func (s *ControllerTestSuite) TearDownTest() {
	// Reset controllers
	s.Controllers = make(map[string]interface{})
}

// TearDownSuite cleans up after the suite is done
func (s *ControllerTestSuite) TearDownSuite() {
	// Any global cleanup
}

// CreateTestMiddleware creates middleware that handles different auth key expectations
func (s *ControllerTestSuite) CreateTestMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set both auth keys for compatibility with different controllers
		c.Set("auth", s.MockAuth)
		c.Set("authController", s.MockAuth)

		// For controllers that do type assertions with (*AuthController)
		c.Next()
	}
}
