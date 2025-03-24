package tests

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCSRFProtection(t *testing.T) {
	// Set up Gin in test mode
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Add CSRF middleware
	r.Use(middleware.CSRFMiddleware())

	// Add a test route
	r.GET("/csrf-test", func(c *gin.Context) {
		// Return the CSRF token
		token, exists := c.Get("csrf_token")
		if !exists {
			c.String(http.StatusInternalServerError, "CSRF token not found")
			return
		}
		c.String(http.StatusOK, token.(string))
	})

	// Add a test POST route
	r.POST("/csrf-test", func(c *gin.Context) {
		c.String(http.StatusOK, "CSRF validation passed")
	})

	// First, make a GET request to get the token
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/csrf-test", nil)
	r.ServeHTTP(w, req)

	// Verify we got a 200 response
	require.Equal(t, http.StatusOK, w.Code)

	// Get the CSRF token from the response
	csrfToken := w.Body.String()
	require.NotEmpty(t, csrfToken)

	// Get cookies from the response
	cookies := w.Header().Values("Set-Cookie")
	assert.NotEmpty(t, cookies, "Should set CSRF cookie")

	// Now make a POST request with the token
	formData := url.Values{}
	formData.Set("_csrf", csrfToken)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/csrf-test", strings.NewReader(formData.Encode()))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Set the cookie from the previous response
	for _, cookie := range cookies {
		req2.Header.Add("Cookie", cookie)
	}

	r.ServeHTTP(w2, req2)

	// Verify the POST request succeeded with CSRF token
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "CSRF validation passed", w2.Body.String())

	// Now make a POST request without the token, should fail
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("POST", "/csrf-test", nil)

	// Set the cookie from the previous response
	for _, cookie := range cookies {
		req3.Header.Add("Cookie", cookie)
	}

	r.ServeHTTP(w3, req3)

	// Verify the POST request failed due to missing CSRF token
	assert.Equal(t, http.StatusForbidden, w3.Code)
}
