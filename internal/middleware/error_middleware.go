package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/errors"
	"github.com/hail2skins/armory/internal/logger"
)

// Simple metrics collector for errors
type errorMetricsCollector struct{}

var errorMetrics = &errorMetricsCollector{}

// Record records error metrics
func (e *errorMetricsCollector) Record(errorType string, status int, duration float64, path string) {
	// This is a placeholder for actual metrics collection
	// In a real application, this would send metrics to a monitoring system
	logger.Info("Error metrics", map[string]interface{}{
		"error_type": errorType,
		"status":     status,
		"duration":   duration,
		"path":       path,
	})
}

// ErrorHandler returns a middleware that handles errors using our custom error types and logger
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Record the start time for latency tracking
		startTime := time.Now()

		// Process the request
		c.Next()

		// Check if there were any errors during processing
		if len(c.Errors) > 0 {
			// Get the last error
			err := c.Errors.Last()

			// Log the error with our custom logger
			logger.Error("Request error", err.Err, map[string]interface{}{
				"path":    c.Request.URL.Path,
				"method":  c.Request.Method,
				"user_id": getUserID(c),
			})

			// Handle the error with our custom error handler
			errors.HandleError(c, err.Err)

			// Record error metrics
			duration := time.Since(startTime).Seconds()
			recordErrorMetrics(c, err.Err, duration)

			// Stop further handlers from executing
			c.Abort()
		}
	}
}

// getUserID attempts to get the user ID from the context
// Returns 0 if no user ID is found
func getUserID(c *gin.Context) uint {
	if user, exists := c.Get("user"); exists {
		if u, ok := user.(interface{ GetID() uint }); ok {
			return u.GetID()
		}
	}
	return 0
}

// recordErrorMetrics records error metrics for the given error
func recordErrorMetrics(c *gin.Context, err error, duration float64) {
	// Get the error type
	errorType := "internal_error" // Default type

	// Try to determine the error type
	switch e := err.(type) {
	case interface{ ErrorType() string }:
		// If the error has an ErrorType method, use that
		errorType = e.ErrorType()
	default:
		// Otherwise, use the error message
		errorType = e.Error()
	}

	// Record the error metrics
	errorMetrics.Record(
		errorType,
		c.Writer.Status(),
		duration,
		c.Request.URL.Path,
	)
}
