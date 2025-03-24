package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCSRFMiddleware(t *testing.T) {
	// Save original env and restore after test
	originalSecret := os.Getenv("CSRF_SECRET")
	defer os.Setenv("CSRF_SECRET", originalSecret)

	// Setup
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Set a test secret
	os.Setenv("CSRF_SECRET", "test-csrf-secret")

	// Initialize session middleware - required for CSRF middleware
	store := cookie.NewStore([]byte("test-session-secret"))
	r.Use(sessions.Sessions("armory-session", store))

	// Add the CSRF middleware
	r.Use(CSRFMiddleware())

	// Test route
	r.GET("/test", func(c *gin.Context) {
		token, exists := c.Get(CSRFKey)
		assert.True(t, exists, "CSRF token should exist in context")
		assert.NotEmpty(t, token, "CSRF token should not be empty")
		c.String(http.StatusOK, "test")
	})

	// Create a test request
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	// Check that a session was created with the CSRF token
	cookies := w.Result().Cookies()
	var sessionCookieFound bool
	for _, cookie := range cookies {
		if cookie.Name == "armory-session" {
			sessionCookieFound = true
			break
		}
	}
	assert.True(t, sessionCookieFound, "Session cookie should be set")
}

func TestCSRFMiddlewareWithDefaultSecret(t *testing.T) {
	// Save original env and restore after test
	originalSecret := os.Getenv("CSRF_SECRET")
	defer os.Setenv("CSRF_SECRET", originalSecret)

	// Setup
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Clear the env variable to test default behavior
	os.Setenv("CSRF_SECRET", "")

	// Initialize session middleware - required for CSRF middleware
	store := cookie.NewStore([]byte("test-session-secret"))
	r.Use(sessions.Sessions("armory-session", store))

	// Add the CSRF middleware
	r.Use(CSRFMiddleware())

	// Test route
	r.GET("/test", func(c *gin.Context) {
		token, exists := c.Get(CSRFKey)
		assert.True(t, exists, "CSRF token should exist in context")
		assert.NotEmpty(t, token, "CSRF token should not be empty")
		c.String(http.StatusOK, "test")
	})

	// Create a test request
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	// Check that a session was created with the CSRF token
	cookies := w.Result().Cookies()
	var sessionCookieFound bool
	for _, cookie := range cookies {
		if cookie.Name == "armory-session" {
			sessionCookieFound = true
			break
		}
	}
	assert.True(t, sessionCookieFound, "Session cookie should be set")
}

// Test POST request with valid CSRF token
func TestCSRFMiddlewareWithPostRequest(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Initialize session middleware
	store := cookie.NewStore([]byte("test-session-secret"))
	r.Use(sessions.Sessions("armory-session", store))

	// Add the CSRF middleware
	r.Use(CSRFMiddleware())

	// Test routes
	r.GET("/form", func(c *gin.Context) {
		// Just a page that would contain a form
		token := GetCSRFToken(c)
		c.String(http.StatusOK, token)
	})

	r.POST("/submit", func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})

	// First make a GET request to get a CSRF token
	getReq, _ := http.NewRequest("GET", "/form", nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)

	// Get the session cookie from the response
	cookies := getW.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "armory-session" {
			sessionCookie = cookie
			break
		}
	}
	assert.NotNil(t, sessionCookie, "Session cookie should exist")

	// Get the CSRF token from the response body
	csrfToken := getW.Body.String()
	assert.NotEmpty(t, csrfToken, "CSRF token should not be empty")

	// Create a POST request with the CSRF token
	postReq, _ := http.NewRequest("POST", "/submit", nil)
	postReq.Header.Set("X-CSRF-Token", csrfToken)
	postReq.AddCookie(sessionCookie)

	postW := httptest.NewRecorder()
	r.ServeHTTP(postW, postReq)

	// Check response
	assert.Equal(t, http.StatusOK, postW.Code, "POST with valid CSRF token should succeed")
}
