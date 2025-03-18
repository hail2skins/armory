package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/errors"
)

// SetupErrorHandling configures all error handling middleware for a Gin router
func SetupErrorHandling(router *gin.Engine) {
	// Set up 404 handler for undefined routes
	router.NoRoute(errors.NoRouteHandler())

	// Set up 405 handler for method not allowed
	router.NoMethod(errors.NoMethodHandler())

	// Set up panic recovery middleware
	router.Use(errors.RecoveryHandler())

	// Set up error handling middleware
	router.Use(ErrorHandler())
}

// NewRateLimiterMiddleware creates and returns a new rate limiter instance
func NewRateLimiterMiddleware() *RateLimiter {
	return NewRateLimiter()
}

// SetupRateLimiting configures rate limiting for critical endpoints
func SetupRateLimiting(router *gin.Engine) {
	rateLimiter := NewRateLimiter()

	// Create a single middleware for all routes
	router.Use(func(c *gin.Context) {
		// Get the actual request path
		path := c.Request.URL.Path

		// Apply the appropriate rate limiter based on the path
		switch {
		case path == "/login":
			// Login rate limit (5 requests per minute)
			rateLimiter.RateLimit(5, time.Minute)(c)
		case path == "/register":
			// Register rate limit (5 requests per minute)
			rateLimiter.RateLimit(5, time.Minute)(c)
		case path == "/reset-password" || path == "/reset-password/new":
			// Password reset rate limit (3 requests per hour)
			rateLimiter.RateLimit(3, time.Hour)(c)
		case path == "/webhook":
			// Webhook rate limit (10 requests per minute)
			rateLimiter.RateLimit(10, time.Minute)(c)
		default:
			// No rate limiting for other routes
			c.Next()
		}
	})
}
