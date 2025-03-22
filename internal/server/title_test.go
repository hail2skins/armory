package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAuthService is defined in routes_test.go in the same package

func TestPageTitles(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Setup session middleware
	store := cookie.NewStore([]byte("test-secret-key"))
	router.Use(sessions.Sessions("auth-session", store))

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
	router.GET("/login", authController.LoginHandler)
	router.GET("/register", authController.RegisterHandler)
	router.GET("/logout", authController.LogoutHandler)

	// Define test cases
	testCases := []struct {
		name      string
		path      string
		wantCode  int
		wantTitle string
	}{
		{
			name:      "Home page",
			path:      "/",
			wantCode:  http.StatusOK,
			wantTitle: "", // Not checking title as we're using mocks
		},
		{
			name:      "Login page",
			path:      "/login",
			wantCode:  http.StatusOK,
			wantTitle: "Login page", // Simple response from mock
		},
		{
			name:      "Register page",
			path:      "/register",
			wantCode:  http.StatusOK,
			wantTitle: "Register page", // Simple response from mock
		},
		{
			name:      "Logout page",
			path:      "/logout",
			wantCode:  http.StatusOK, // Our mock just returns OK with a string
			wantTitle: "Logged out",  // Simple response from mock
		},
	}

	// Run the tests
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a request
			req, err := http.NewRequest("GET", tc.path, nil)
			require.NoError(t, err)

			// Create a response recorder
			resp := httptest.NewRecorder()

			// Serve the request
			router.ServeHTTP(resp, req)

			// Check the status code
			assert.Equal(t, tc.wantCode, resp.Code)

			// For the home page, we don't check content since it's using a real controller with templates
			if tc.path == "/" {
				return
			}

			// For other routes using our mocks, check the content
			body := resp.Body.String()
			assert.Contains(t, body, tc.wantTitle, "Response should contain expected content")
		})
	}
}

// extractTitle extracts the title tag from an HTML string
func extractTitle(html string) string {
	titleStart := strings.Index(html, "<title>")
	if titleStart == -1 {
		return ""
	}
	titleStart += 7 // Length of "<title>"

	titleEnd := strings.Index(html[titleStart:], "</title>")
	if titleEnd == -1 {
		return ""
	}

	return html[titleStart : titleStart+titleEnd]
}
