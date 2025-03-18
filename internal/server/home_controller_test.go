package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/assert"
)

func TestHomeControllerWithAuthService(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockDB := &mocks.MockDB{} // Use the correct mock DB implementation

	t.Run("HomeController uses AuthService correctly", func(t *testing.T) {
		// Create a new router
		router := gin.New()

		// Create a mock auth service using the implementation from routes_test.go
		mockAuth := &MockAuthService{
			authenticated: true,
			email:         "test@example.com",
		}

		// Add middleware to set auth service BEFORE adding routes
		router.Use(func(c *gin.Context) {
			c.Set("auth", mockAuth)
			c.Next()
		})

		// Create home controller
		homeController := controller.NewHomeController(mockDB)

		// Add routes after middleware
		router.GET("/", homeController.HomeHandler)

		// Create a test request
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		resp := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(resp, req)

		// Check that the response is successful
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), "The Virtual Armory")
	})

	t.Run("HomeController handles unauthenticated state", func(t *testing.T) {
		// Create a new router
		router := gin.New()

		// Create a mock auth service with unauthenticated state
		mockAuth := &MockAuthService{
			authenticated: false,
			email:         "",
		}

		// Add middleware to set auth service BEFORE adding routes
		router.Use(func(c *gin.Context) {
			c.Set("auth", mockAuth)
			c.Next()
		})

		// Create home controller
		homeController := controller.NewHomeController(mockDB)

		// Add routes after middleware
		router.GET("/", homeController.HomeHandler)

		// Create a test request
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		resp := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(resp, req)

		// Check that the response is successful
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), "The Virtual Armory")
	})
}
