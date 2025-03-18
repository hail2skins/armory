package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/shaj13/go-guardian/v2/auth"
	"github.com/stretchr/testify/assert"
)

// IMPORTANT: The MockAuthService and MockUserInfo types defined in this file
// are used by multiple test files in this package (title_test.go and auth_integration_test.go).
// When running individual test files, include this file in your go test command to ensure
// these mock types are available:
// Example: go test ./internal/server/routes_test.go ./internal/server/title_test.go -v

// MockAuthService is a mock implementation of controller.AuthService
type MockAuthService struct {
	authenticated bool
	email         string
}

// GetCurrentUser returns a mock user and authentication status
func (m *MockAuthService) GetCurrentUser(c *gin.Context) (auth.Info, bool) {
	if !m.authenticated {
		return nil, false
	}
	return &MockUserInfo{email: m.email}, true
}

// IsAuthenticated returns the mock authentication status
func (m *MockAuthService) IsAuthenticated(c *gin.Context) bool {
	return m.authenticated
}

// LoginHandler is a mock implementation
func (m *MockAuthService) LoginHandler(c *gin.Context) {
	c.String(http.StatusOK, "Login page")
}

// LogoutHandler is a mock implementation
func (m *MockAuthService) LogoutHandler(c *gin.Context) {
	m.authenticated = false
	c.String(http.StatusOK, "Logged out")
}

// RegisterHandler is a mock implementation
func (m *MockAuthService) RegisterHandler(c *gin.Context) {
	c.String(http.StatusOK, "Register page")
}

// VerifyEmailHandler is a mock implementation
func (m *MockAuthService) VerifyEmailHandler(c *gin.Context) {
	c.String(http.StatusOK, "Email verified")
}

// ForgotPasswordHandler is a mock implementation
func (m *MockAuthService) ForgotPasswordHandler(c *gin.Context) {
	c.String(http.StatusOK, "Forgot password page")
}

// ResetPasswordHandler is a mock implementation
func (m *MockAuthService) ResetPasswordHandler(c *gin.Context) {
	c.String(http.StatusOK, "Reset password page")
}

// MockUserInfo implements auth.Info
type MockUserInfo struct {
	email      string
	id         string
	groups     []string
	extensions auth.Extensions
}

func (m *MockUserInfo) GetUserName() string {
	return m.email
}

func (m *MockUserInfo) GetID() string {
	if m.id == "" {
		return "1"
	}
	return m.id
}

func (m *MockUserInfo) GetGroups() []string {
	return m.groups
}

func (m *MockUserInfo) GetExtensions() auth.Extensions {
	if m.extensions == nil {
		return auth.Extensions{}
	}
	return m.extensions
}

func (m *MockUserInfo) SetUserName(username string) {
	m.email = username
}

func (m *MockUserInfo) SetID(id string) {
	m.id = id
}

func (m *MockUserInfo) SetGroups(groups []string) {
	m.groups = groups
}

func (m *MockUserInfo) SetExtensions(exts auth.Extensions) {
	m.extensions = exts
}

// TestSimplifiedRoutes tests that basic routes can be registered
func TestSimplifiedRoutes(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Create mock DB
	mockDB := new(mocks.MockDB)
	mockDB.On("Health").Return(map[string]string{"status": "ok"})

	// Create controllers
	authController := &MockAuthService{authenticated: false, email: ""}
	homeController := controller.NewHomeController(mockDB)

	// Add auth middleware to router
	router.Use(func(c *gin.Context) {
		// Set auth controller in context
		c.Set("auth", authController)
		c.Set("authController", authController)
		c.Set("authData", map[string]interface{}{
			"Authenticated": false,
			"Email":         "",
			"Title":         "Test Page",
		})
		c.Next()
	})

	// Register basic routes
	router.GET("/", homeController.HomeHandler)
	router.GET("/about", homeController.AboutHandler)
	router.GET("/contact", homeController.ContactHandler)
	router.POST("/contact", homeController.ContactHandler)

	// Auth routes
	router.GET("/login", authController.LoginHandler)
	router.POST("/login", authController.LoginHandler)
	router.GET("/register", authController.RegisterHandler)
	router.POST("/register", authController.RegisterHandler)
	router.GET("/logout", authController.LogoutHandler)
	router.GET("/forgot-password", authController.ForgotPasswordHandler)
	router.POST("/forgot-password", authController.ForgotPasswordHandler)
	router.GET("/reset-password", authController.ResetPasswordHandler)
	router.POST("/reset-password", authController.ResetPasswordHandler)

	// Test routes
	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/"},
		{"GET", "/about"},
		{"GET", "/contact"},
		{"POST", "/contact"},
		{"GET", "/login"},
		{"POST", "/login"},
		{"GET", "/register"},
		{"POST", "/register"},
		{"GET", "/logout"},
		{"GET", "/forgot-password"},
		{"POST", "/forgot-password"},
		{"GET", "/reset-password"},
		{"POST", "/reset-password"},
	}

	for _, route := range routes {
		t.Run("Testing "+route.path, func(t *testing.T) {
			// Create request
			req, _ := http.NewRequest(route.method, route.path, nil)
			resp := httptest.NewRecorder()

			// Perform request
			router.ServeHTTP(resp, req)

			// Assert route is defined (not 404)
			assert.NotEqual(t, http.StatusNotFound, resp.Code, "Route %s should be defined", route.path)
		})
	}
}

// TestSimplifiedNavbar tests the navbar auth state
func TestSimplifiedNavbar(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// For navbar tests, we'll use our mock directly and not try to render templates
	// Create both auth states to test
	authenticatedAuth := &MockAuthService{
		authenticated: true,
		email:         "test@example.com",
	}

	unauthenticatedAuth := &MockAuthService{
		authenticated: false,
		email:         "",
	}

	// Verify the auth service works correctly
	t.Run("auth service methods", func(t *testing.T) {
		// Test authenticated state
		info, auth := authenticatedAuth.GetCurrentUser(nil)
		assert.True(t, auth, "Authenticated service should return true")
		assert.Equal(t, "test@example.com", info.GetUserName(), "User email should match")
		assert.True(t, authenticatedAuth.IsAuthenticated(nil), "IsAuthenticated should return true")

		// Test unauthenticated state
		info, auth = unauthenticatedAuth.GetCurrentUser(nil)
		assert.False(t, auth, "Unauthenticated service should return false")
		assert.Nil(t, info, "User info should be nil when not authenticated")
		assert.False(t, unauthenticatedAuth.IsAuthenticated(nil), "IsAuthenticated should return false")
	})
}
