package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testErrorMiddleware is a test middleware that checks for errors and handles them
func testErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there were any errors during processing
		if len(c.Errors) > 0 {
			// Get the last error
			err := c.Errors.Last().Err

			// Check error type and respond accordingly
			switch err.(type) {
			case *errors.ForbiddenError:
				c.AbortWithStatus(http.StatusForbidden)
				return
			case *errors.InternalServerError:
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			default:
				// For any other error
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
		}
	}
}

func TestCSRFMiddleware(t *testing.T) {
	// Save original test mode and restore after test
	originalTestMode := TestMode
	defer func() { TestMode = originalTestMode }()
	TestMode = false // Ensure test mode is off to properly test security

	// Setup
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Initialize session middleware - required for CSRF middleware
	store := cookie.NewStore([]byte("test-session-secret"))
	r.Use(sessions.Sessions("armory-session", store))

	// Add our test error middleware to handle errors from CSRF middleware
	r.Use(testErrorMiddleware())

	// Add the CSRF middleware
	r.Use(CSRFMiddleware())

	// Test route that returns the CSRF token for GET
	r.GET("/form", func(c *gin.Context) {
		token, exists := c.Get(CSRFKey)
		assert.True(t, exists, "CSRF token should exist in context")
		assert.NotEmpty(t, token, "CSRF token should not be empty")
		c.String(http.StatusOK, token.(string))
	})

	// Test route for POST that requires CSRF token
	r.POST("/submit", func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})

	// Test 1: GET request should generate a CSRF token and set it in context
	t.Run("GET request generates CSRF token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/form", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Check response code
		assert.Equal(t, http.StatusOK, w.Code)

		// Token should be in the response body
		token := w.Body.String()
		assert.NotEmpty(t, token, "CSRF token should be in the response")

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
	})

	// Test 2: POST without CSRF token should be blocked (403 Forbidden)
	t.Run("POST without CSRF token is blocked", func(t *testing.T) {
		form := url.Values{}
		form.Add("foo", "bar")

		req, _ := http.NewRequest("POST", "/submit", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Should be forbidden due to missing CSRF token
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	// Test 3: POST with invalid CSRF token should be blocked
	t.Run("POST with invalid CSRF token is blocked", func(t *testing.T) {
		form := url.Values{}
		form.Add("foo", "bar")
		form.Add("csrf_token", "invalid-token")

		req, _ := http.NewRequest("POST", "/submit", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Should be forbidden due to invalid CSRF token
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	// Test 4: Complete valid CSRF flow (GET token, then POST with token)
	t.Run("Valid CSRF flow succeeds", func(t *testing.T) {
		// Step 1: Get a valid token
		getReq, _ := http.NewRequest("GET", "/form", nil)
		getW := httptest.NewRecorder()
		r.ServeHTTP(getW, getReq)

		// Check token is returned in response
		token := getW.Body.String()
		assert.NotEmpty(t, token, "CSRF token should be returned")

		// Get session cookie from response
		cookies := getW.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == "armory-session" {
				sessionCookie = cookie
				break
			}
		}
		require.NotNil(t, sessionCookie, "Session cookie should exist")

		// Step 2: Submit form with the token
		form := url.Values{}
		form.Add("foo", "bar")
		form.Add("csrf_token", token)

		postReq, _ := http.NewRequest("POST", "/submit", strings.NewReader(form.Encode()))
		postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		postReq.AddCookie(sessionCookie)
		postW := httptest.NewRecorder()
		r.ServeHTTP(postW, postReq)

		// Request should succeed
		assert.Equal(t, http.StatusOK, postW.Code)
		assert.Equal(t, "success", postW.Body.String())
	})

	// Test 5: CSRF token from header instead of form
	t.Run("CSRF token from header succeeds", func(t *testing.T) {
		// Step 1: Get a valid token
		getReq, _ := http.NewRequest("GET", "/form", nil)
		getW := httptest.NewRecorder()
		r.ServeHTTP(getW, getReq)

		token := getW.Body.String()
		assert.NotEmpty(t, token, "CSRF token should be returned")

		cookies := getW.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == "armory-session" {
				sessionCookie = cookie
				break
			}
		}
		require.NotNil(t, sessionCookie, "Session cookie should exist")

		// Step 2: Submit with token in header
		form := url.Values{}
		form.Add("foo", "bar")
		// Intentionally don't add to form: form.Add("csrf_token", token)

		postReq, _ := http.NewRequest("POST", "/submit", strings.NewReader(form.Encode()))
		postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		postReq.Header.Set("X-CSRF-Token", token) // Set in header instead
		postReq.AddCookie(sessionCookie)
		postW := httptest.NewRecorder()
		r.ServeHTTP(postW, postReq)

		// Request should succeed
		assert.Equal(t, http.StatusOK, postW.Code)
	})

	// Test 6: Token rotation after POST request
	t.Run("CSRF token is rotated after POST", func(t *testing.T) {
		// Step 1: Get first token
		getReq1, _ := http.NewRequest("GET", "/form", nil)
		getW1 := httptest.NewRecorder()
		r.ServeHTTP(getW1, getReq1)

		firstToken := getW1.Body.String()
		assert.NotEmpty(t, firstToken, "First CSRF token should be returned")

		cookies1 := getW1.Result().Cookies()
		var sessionCookie1 *http.Cookie
		for _, cookie := range cookies1 {
			if cookie.Name == "armory-session" {
				sessionCookie1 = cookie
				break
			}
		}
		require.NotNil(t, sessionCookie1, "Session cookie should exist")

		// Step 2: Submit form with first token
		form := url.Values{}
		form.Add("csrf_token", firstToken)

		postReq, _ := http.NewRequest("POST", "/submit", strings.NewReader(form.Encode()))
		postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		postReq.AddCookie(sessionCookie1)
		postW := httptest.NewRecorder()
		r.ServeHTTP(postW, postReq)

		// Save new session cookie after POST
		postCookies := postW.Result().Cookies()
		var postSessionCookie *http.Cookie
		for _, cookie := range postCookies {
			if cookie.Name == "armory-session" {
				postSessionCookie = cookie
				break
			}
		}
		require.NotNil(t, postSessionCookie, "Updated session cookie should exist")

		// Step 3: Get token again (should be different)
		getReq2, _ := http.NewRequest("GET", "/form", nil)
		getReq2.AddCookie(postSessionCookie) // Use cookie from previous POST
		getW2 := httptest.NewRecorder()
		r.ServeHTTP(getW2, getReq2)

		secondToken := getW2.Body.String()
		assert.NotEmpty(t, secondToken, "Second CSRF token should be returned")

		// Tokens should be different (rotation happened)
		assert.NotEqual(t, firstToken, secondToken, "CSRF token should have been rotated")

		// Step 4: Old token should no longer work
		oldTokenForm := url.Values{}
		oldTokenForm.Add("csrf_token", firstToken)

		oldTokenReq, _ := http.NewRequest("POST", "/submit", strings.NewReader(oldTokenForm.Encode()))
		oldTokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		oldTokenReq.AddCookie(postSessionCookie)
		oldTokenW := httptest.NewRecorder()
		r.ServeHTTP(oldTokenW, oldTokenReq)

		// Request should fail because token was rotated
		assert.Equal(t, http.StatusForbidden, oldTokenW.Code)
	})

	// Test 7: Test mode allows bypassing CSRF
	t.Run("Test mode bypasses CSRF validation", func(t *testing.T) {
		// Enable test mode
		TestMode = true
		defer func() { TestMode = false }()

		// Post without any real token
		form := url.Values{}
		form.Add("foo", "bar")
		form.Add("csrf_token", "dummy-test-token")

		req, _ := http.NewRequest("POST", "/submit", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Should succeed even though token is dummy
		assert.Equal(t, http.StatusOK, w.Code)
	})

	// Test 8: Bypass header allows skipping validation
	t.Run("X-Test-CSRF-Bypass header bypasses CSRF validation", func(t *testing.T) {
		// Ensure test mode is off
		TestMode = false

		// Post with bypass header
		form := url.Values{}
		form.Add("foo", "bar")
		// No real token

		req, _ := http.NewRequest("POST", "/submit", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Test-CSRF-Bypass", "true")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Should succeed due to bypass header
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestGetCSRFToken(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Initialize session middleware - required for CSRF middleware
	store := cookie.NewStore([]byte("test-session-secret"))
	r.Use(sessions.Sessions("armory-session", store))

	// Test route that uses GetCSRFToken helper
	r.GET("/csrf-helper", func(c *gin.Context) {
		token := GetCSRFToken(c)
		assert.NotEmpty(t, token, "CSRF token from helper should not be empty")
		c.String(http.StatusOK, token)
	})

	// Test 1: GetCSRFToken should return a token
	t.Run("GetCSRFToken returns valid token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/csrf-helper", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Check response code
		assert.Equal(t, http.StatusOK, w.Code)

		// Token should be in the response body
		token := w.Body.String()
		assert.NotEmpty(t, token, "CSRF token should be in the response")
		assert.Greater(t, len(token), 30, "Token should be at least 30 chars")
	})

	// Test 2: GetCSRFToken returns consistent token for same context
	t.Run("GetCSRFToken returns consistent token", func(t *testing.T) {
		r.GET("/csrf-multiple", func(c *gin.Context) {
			token1 := GetCSRFToken(c)
			token2 := GetCSRFToken(c)
			assert.Equal(t, token1, token2, "CSRF token should be consistent in same context")
			c.String(http.StatusOK, token1)
		})

		req, _ := http.NewRequest("GET", "/csrf-multiple", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
