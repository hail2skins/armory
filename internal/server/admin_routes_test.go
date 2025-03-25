package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/middleware"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/assert"
)

// TestAdminPromotionRoutes tests the promotion routes specifically
func TestAdminPromotionRoutes(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Create mock DB
	mockDB := new(mocks.MockDB)

	// Create controllers directly - only the one we need
	adminPromotionController := controller.NewAdminPromotionController(mockDB)

	// Create a mock auth service that is authenticated as an admin
	mockAuth := &MockAuthService{
		authenticated: true,
		email:         "admin@example.com",
	}

	// Enable test mode for CSRF middleware
	middleware.EnableTestMode()

	// Add session middleware
	store := cookie.NewStore([]byte("test-secret"))
	router.Use(sessions.Sessions("armory-session", store))

	// Add auth middleware to router
	router.Use(func(c *gin.Context) {
		// Set mock auth in context
		c.Set("auth", mockAuth)
		c.Set("authController", mockAuth)

		// Set a mock CSRF token
		c.Set("csrf_token", "test-csrf-token")

		c.Next()
	})

	// Register routes directly - only the one we're testing
	adminGroup := router.Group("/admin")
	adminGroup.GET("/promotions/new", adminPromotionController.New)

	// Test the promotion route
	req, _ := http.NewRequest("GET", "/admin/promotions/new", nil)
	resp := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(resp, req)

	// Assertions
	assert.NotEqual(t, http.StatusNotFound, resp.Code, "Route /admin/promotions/new should be defined")
	assert.Equal(t, http.StatusOK, resp.Code, "Expected OK status for /admin/promotions/new")

	// Clean up
	middleware.DisableTestMode()
}
