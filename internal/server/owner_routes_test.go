package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/assert"
)

// TestSimplifiedOwnerRoutes tests that owner routes can be registered
func TestSimplifiedOwnerRoutes(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add session middleware
	store := cookie.NewStore([]byte("test-secret-key"))
	router.Use(sessions.Sessions("mysession", store))

	// Create mock DB
	mockDB := new(mocks.MockDB)

	// Create controllers
	mockAuthController := controller.NewAuthController(mockDB)
	ownerController := controller.NewOwnerController(mockDB)

	// Add auth middleware to router
	router.Use(func(c *gin.Context) {
		// Set auth controller in context
		c.Set("auth", mockAuthController)
		c.Set("authController", mockAuthController)
		c.Next()
	})

	// Register owner routes directly
	ownerGroup := router.Group("/owner")
	ownerGroup.Use(func(c *gin.Context) {
		// Check if user is authenticated
		_, authenticated := mockAuthController.GetCurrentUser(c)
		if !authenticated {
			// Redirect to login
			c.Redirect(http.StatusSeeOther, "/login")
			c.Abort()
			return
		}
		c.Next()
	})
	{
		// Owner landing page
		ownerGroup.GET("", ownerController.LandingPage)

		// Owner profile
		ownerGroup.GET("/profile", ownerController.Profile)
		ownerGroup.GET("/profile/edit", ownerController.EditProfile)
		ownerGroup.POST("/profile", ownerController.UpdateProfile)

		// Arsenal view
		ownerGroup.GET("/arsenal", ownerController.Arsenal)

		// Subscription management
		ownerGroup.GET("/subscription", ownerController.Subscription)

		// Gun routes
		gunGroup := ownerGroup.Group("/guns")
		{
			gunGroup.GET("", ownerController.Index)
			gunGroup.GET("/new", ownerController.New)
			gunGroup.POST("", ownerController.Create)
			gunGroup.GET("/:id", ownerController.Show)
			gunGroup.GET("/:id/edit", ownerController.Edit)
			gunGroup.POST("/:id", ownerController.Update)
			gunGroup.POST("/:id/delete", ownerController.Delete)
		}
	}

	// Test routes
	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/owner"},
		{"GET", "/owner/profile"},
		{"GET", "/owner/profile/edit"},
		{"POST", "/owner/profile"},
		{"GET", "/owner/arsenal"},
		{"GET", "/owner/subscription"},
		{"GET", "/owner/guns"},
		{"GET", "/owner/guns/new"},
		{"POST", "/owner/guns"},
		{"GET", "/owner/guns/1"},
		{"GET", "/owner/guns/1/edit"},
		{"POST", "/owner/guns/1"},
		{"POST", "/owner/guns/1/delete"},
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

			// Depending on authentication, we should get redirect or successful response
			statusOK := resp.Code == http.StatusOK
			statusRedirect := resp.Code == http.StatusSeeOther && resp.Header().Get("Location") == "/login"

			assert.True(t, statusOK || statusRedirect,
				"Expected either OK or redirect to login, got status %d", resp.Code)
		})
	}
}
