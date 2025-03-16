package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOwnerRouteRedirectWithFlashMessage tests that when an unauthenticated user
// tries to access the /owner route, they are redirected to login with a flash message
func TestOwnerRouteRedirectWithFlashMessage(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create a new test server
	server := NewTestServer()

	// Create a router with all routes registered
	router := server.RegisterRoutes().(*gin.Engine)

	// Create a request to the owner page
	req, err := http.NewRequest("GET", "/owner", nil)
	require.NoError(t, err)

	// Create a response recorder
	resp := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(resp, req)

	// Check that we get redirected to login
	assert.Equal(t, http.StatusFound, resp.Code)
	assert.Equal(t, "/login", resp.Header().Get("Location"))

	// Check for the flash message in the session cookie
	cookies := resp.Result().Cookies()
	var flashCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "flash" || cookie.Name == "session" {
			flashCookie = cookie
			break
		}
	}

	// Assert that we have a flash or session cookie
	assert.NotNil(t, flashCookie, "Flash or session cookie should be set")
}

// NewTestServer creates a new test server with mocked dependencies
func NewTestServer() *Server {
	// Create a test database service
	db := testutils.NewTestService()

	// Create a new server with the test database
	return &Server{
		db: db,
	}
}
