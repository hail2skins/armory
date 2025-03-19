package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/metrics"
)

// errorMetricsInstance is the global error metrics instance
var errorMetricsInstance *metrics.ErrorMetrics

// Initialize the error metrics
func init() {
	errorMetricsInstance = metrics.NewErrorMetrics()

	// Start a goroutine to periodically clean up old error entries
	go func() {
		for {
			time.Sleep(1 * time.Hour)
			errorMetricsInstance.Cleanup(24 * time.Hour * 7) // Keep errors for 7 days
		}
	}()
}

// GetErrorMetrics returns the global error metrics instance
func GetErrorMetrics() *metrics.ErrorMetrics {
	return errorMetricsInstance
}

// ErrorMetricsMiddleware returns a middleware that records error metrics
func ErrorMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Record the start time
		startTime := time.Now()

		// Process the request
		c.Next()

		// Calculate the request duration
		duration := time.Since(startTime).Seconds()

		// If there were errors, record them
		if len(c.Errors) > 0 {
			// Get the last error
			err := c.Errors.Last()

			// Get the error type
			errorType := "internal_error" // Default type

			// Try to determine the error type
			switch e := err.Err.(type) {
			case interface{ ErrorType() string }:
				// If the error has an ErrorType method, use that
				errorType = e.ErrorType()
			default:
				// Otherwise, use the error message
				errorType = e.Error()
			}

			// Record the error metrics
			errorMetricsInstance.Record(
				errorType,
				c.Writer.Status(),
				duration,
				c.Request.URL.Path,
			)
		}
	}
}
