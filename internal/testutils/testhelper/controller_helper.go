package testhelper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/database"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// Set Gin to test mode for all tests
func init() {
	gin.SetMode(gin.TestMode)
}

// ControllerTestHelper provides utilities for testing controllers
type ControllerTestHelper struct {
	TestData    *TestData
	Router      *gin.Engine
	AuthService AuthService
}

// NewControllerTestHelper creates a new helper with all necessary dependencies
func NewControllerTestHelper(db *gorm.DB, service database.Service) *ControllerTestHelper {
	// Initialize TestData with database connection
	testData := NewTestData(db, service)

	// Create the auth service
	authService := NewMockAuthService(service)

	// Create a basic router
	router := authService.SetupAuthMiddleware(gin.New())
	router.Use(gin.Recovery())

	return &ControllerTestHelper{
		TestData:    testData,
		Router:      router,
		AuthService: authService,
	}
}

// GetUnauthenticatedRouter returns a router with common middleware but no authentication
func (ch *ControllerTestHelper) GetUnauthenticatedRouter() *gin.Engine {
	return ch.Router
}

// GetAuthenticatedRouter returns a router with a user already authenticated
func (ch *ControllerTestHelper) GetAuthenticatedRouter(userID uint, email string) *gin.Engine {
	return ch.AuthService.SetupAuthenticatedRouter(userID, email)
}

// MakeRequest is a utility method to make a test request
func (ch *ControllerTestHelper) MakeRequest(method, path string, body string, headers map[string]string) (*httptest.ResponseRecorder, *http.Request) {
	// Create request
	req, _ := http.NewRequest(method, path, strings.NewReader(body))

	// Set content type based on body
	if body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	// Set any custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Create response recorder
	w := httptest.NewRecorder()

	return w, req
}

// SubmitForm is a utility method to submit a form and test the response
func (ch *ControllerTestHelper) SubmitForm(t *testing.T, router *gin.Engine, method, path string, form url.Values, expectedStatus int, expectedRedirect string) *httptest.ResponseRecorder {
	// Create recorder and request
	w, req := ch.MakeRequest(method, path, form.Encode(), nil)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Serve the request
	router.ServeHTTP(w, req)

	// Check status code
	assert.Equal(t, expectedStatus, w.Code)

	// Check redirect if expected
	if expectedRedirect != "" {
		assert.Equal(t, expectedRedirect, w.Header().Get("Location"))
	}

	return w
}

// AssertViewRendered is a utility to check if a view was rendered with the right data
func (ch *ControllerTestHelper) AssertViewRendered(t *testing.T, router *gin.Engine, method, path string, headers map[string]string, expectedStatus int) *httptest.ResponseRecorder {
	// Make the request
	w, req := ch.MakeRequest(method, path, "", headers)
	router.ServeHTTP(w, req)

	// Check the status
	assert.Equal(t, expectedStatus, w.Code)

	return w
}

// CreateTestUser creates a test user and returns it
func (ch *ControllerTestHelper) CreateTestUser(t *testing.T) *database.User {
	user := ch.TestData.CreateTestUser(context.Background())
	assert.NotNil(t, user, "Failed to create test user")
	return user
}

// CreateTestGun creates a test gun and returns it
func (ch *ControllerTestHelper) CreateTestGun(t *testing.T, user *database.User) *Gun {
	gun := ch.TestData.CreateTestGun(user)
	assert.NotNil(t, gun, "Failed to create test gun")
	return gun
}

// CleanupTest performs cleanup after a test
func (ch *ControllerTestHelper) CleanupTest() {
	ch.TestData.CleanupTestData()
}
