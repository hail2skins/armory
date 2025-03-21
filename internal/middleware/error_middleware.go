package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/errors"
	"github.com/hail2skins/armory/internal/logger"
)

// errorMetricsCollector is a simple collector for error metrics
type errorMetricsCollector struct{}

// errorMetricsCollectorInstance is the global instance of the error metrics collector
var errorMetricsCollectorInstance = &errorMetricsCollector{}

// Record records an error metric
func (e *errorMetricsCollector) Record(errorType string, status int, duration float64, path string) {
	// Create fields for logging
	fields := map[string]interface{}{
		"error_type": errorType,
		"status":     status,
		"duration":   duration,
		"path":       path,
	}

	// Log the error metric
	logger.Info("Error metrics", fields)
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

// recordErrorMetrics records error metrics
func recordErrorMetrics(c *gin.Context, err error, duration float64) {
	// Default error type if we can't determine it
	errorType := "internal_error"

	// Try to determine the error type if the error has a type method
	if typedErr, ok := err.(interface{ Type() string }); ok {
		errorType = typedErr.Type()
	} else if typedErr, ok := err.(interface{ ErrorType() string }); ok {
		// Alternative method name for error type
		errorType = typedErr.ErrorType()
	} else {
		// Use the error message as the type
		errorType = err.Error()
	}

	// Get the client IP address
	clientIP := c.ClientIP()
	if clientIP == "" {
		clientIP = "unknown"
	}

	// Record metrics in our advanced metrics system
	if errorMetricsInstance != nil {
		errorMetricsInstance.Record(
			errorType,
			c.Writer.Status(),
			duration,
			c.Request.URL.Path,
			clientIP,
		)
	}

	// Also record using our simple collector for logging
	errorMetricsCollectorInstance.Record(
		errorType,
		c.Writer.Status(),
		duration,
		c.Request.URL.Path,
	)
}
