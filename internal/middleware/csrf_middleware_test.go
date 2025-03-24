package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

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

	// Add the CSRF middleware
	r.Use(CSRFMiddleware())

	// Test route
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "test")
	})

	// Create a test request
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify that the CSRF cookie is set
	cookies := w.Result().Cookies()
	var csrfCookieFound bool
	for _, cookie := range cookies {
		if cookie.Name == "csrf_token" {
			csrfCookieFound = true
			break
		}
	}
	assert.True(t, csrfCookieFound, "CSRF cookie should be set")

	// Verify that the CSRF token is available in the context
	// This is an indirect test that nosurf middleware was correctly applied
	// The token would be accessible in a handler via c.GetString("csrf_token")
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

	// Add the CSRF middleware
	r.Use(CSRFMiddleware())

	// Test route
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "test")
	})

	// Create a test request
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify that the CSRF cookie is set even with default secret
	cookies := w.Result().Cookies()
	var csrfCookieFound bool
	for _, cookie := range cookies {
		if cookie.Name == "csrf_token" {
			csrfCookieFound = true
			break
		}
	}
	assert.True(t, csrfCookieFound, "CSRF cookie should be set even with default secret")
}
