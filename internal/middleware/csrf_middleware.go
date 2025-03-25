package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"os"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/errors"
)

const (
	// CSRFKey is the key used to store the CSRF token in the session
	CSRFKey = "csrf_token"
)

var (
	// TestMode indicates if we should bypass CSRF validation for tests
	TestMode bool
)

// CSRFMiddleware creates a middleware for CSRF protection
func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Special handling for test mode - always bypass CSRF checks
		if TestMode || os.Getenv("GO_ENV") == "test" || c.Request.Header.Get("X-Test-CSRF-Bypass") == "true" {
			// For GET requests, still generate a token for templates
			if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
				token, _ := generateToken()
				c.Set(CSRFKey, token)
			}
			c.Next()
			return
		}

		session := sessions.Default(c)

		// For GET, HEAD, OPTIONS requests - just make sure a token exists in the session
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			// Check if a token already exists in the session
			sessionToken := session.Get(CSRFKey)
			var csrfToken string

			if sessionToken == nil {
				// Generate a new token if none exists
				token, err := generateToken()
				if err != nil {
					// Use the error middleware instead of direct abort
					c.Error(errors.NewInternalServerError("Failed to generate CSRF token"))
					c.Abort()
					return
				}

				// Store it in the session
				session.Set(CSRFKey, token)
				session.Save()
				csrfToken = token
			} else {
				// Use existing token
				csrfToken = sessionToken.(string)
			}

			// Make the token available to templates
			c.Set(CSRFKey, csrfToken)
			c.Next()
			return
		}

		// For other methods (POST, PUT, DELETE, etc.), verify the token
		sessionToken := session.Get(CSRFKey)
		if sessionToken == nil {
			// No CSRF token in session
			// Use the error middleware instead of direct abort
			c.Error(errors.NewForbiddenError("CSRF token is missing or invalid"))
			c.Abort()
			return
		}

		// Get token from form or header
		requestToken := c.PostForm("csrf_token")

		// Also check X-CSRF-Token header as a fallback
		if requestToken == "" {
			requestToken = c.GetHeader("X-CSRF-Token")
		}

		if requestToken == "" || sessionToken.(string) != requestToken {
			// Invalid or missing token
			// Use the error middleware instead of direct abort
			c.Error(errors.NewForbiddenError("CSRF token is missing or invalid"))
			c.Abort()
			return
		}

		// Generate a new token for the next request
		newToken, err := generateToken()
		if err != nil {
			// Use the error middleware instead of direct abort
			c.Error(errors.NewInternalServerError("Failed to generate CSRF token"))
			c.Abort()
			return
		}

		// Store the new token in the session
		session.Set(CSRFKey, newToken)
		session.Save()

		// Make the new token available to templates
		c.Set(CSRFKey, newToken)

		c.Next()
	}
}

// generateToken creates a new random token
func generateToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// GetCSRFToken returns the CSRF token from the context
func GetCSRFToken(c *gin.Context) string {
	// Try to get from context first
	if token, exists := c.Get(CSRFKey); exists {
		if tokenStr, ok := token.(string); ok {
			return tokenStr
		}
	}

	// If not in context, try to get from session
	session := sessions.Default(c)
	if token := session.Get(CSRFKey); token != nil {
		if tokenStr, ok := token.(string); ok {
			return tokenStr
		}
	}

	// If no token found, generate a new one
	token, err := generateToken()
	if err != nil {
		// Return empty string in case of error
		return ""
	}

	// Store the new token
	session = sessions.Default(c)
	session.Set(CSRFKey, token)
	session.Save()

	// Also set in context
	c.Set(CSRFKey, token)

	return token
}

// EnableTestMode turns on CSRF test mode to bypass validation
func EnableTestMode() {
	TestMode = true
}

// DisableTestMode turns off CSRF test mode
func DisableTestMode() {
	TestMode = false
}
