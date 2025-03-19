package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Custom error type that implements ErrorType() method
type testErrorWithType struct {
	message   string
	errorType string
}

func (e *testErrorWithType) Error() string {
	return e.message
}

func (e *testErrorWithType) ErrorType() string {
	return e.errorType
}

func TestErrorMetricsMiddleware(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create mock error metrics
	mockMetrics := metrics.NewErrorMetrics()

	// Save original and restore after test
	originalErrorMetrics := errorMetricsInstance
	defer func() { errorMetricsInstance = originalErrorMetrics }()

	// Replace with mock
	errorMetricsInstance = mockMetrics

	// Create a new Gin router
	r := gin.New()

	// Add the error metrics middleware
	r.Use(ErrorMetricsMiddleware())

	// Add a test route that returns an error
	r.GET("/test-error", func(c *gin.Context) {
		c.Error(errors.New("test error"))
		c.Status(http.StatusInternalServerError)
	})

	// Add a custom error type route
	r.GET("/custom-error", func(c *gin.Context) {
		c.Error(&testErrorWithType{
			message:   "custom error",
			errorType: "custom_error_type",
		})
		c.Status(http.StatusBadRequest)
	})

	// Add a success route
	r.GET("/success", func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})

	t.Run("Records error metrics on error", func(t *testing.T) {
		// Create a test request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test-error", nil)

		// Serve the request
		r.ServeHTTP(w, req)

		// Get stats to verify recording
		stats := mockMetrics.GetStats()
		errorCounts, ok := stats["error_counts"].(map[string]*metrics.ErrorEntry)
		assert.True(t, ok)

		// Check that our error was recorded
		assert.Contains(t, errorCounts, "test error")
		assert.Equal(t, int64(1), errorCounts["test error"].Count)
		assert.Equal(t, "/test-error", errorCounts["test error"].Path)
	})

	t.Run("Records custom error type", func(t *testing.T) {
		// Create a test request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/custom-error", nil)

		// Serve the request
		r.ServeHTTP(w, req)

		// Get stats to verify recording
		stats := mockMetrics.GetStats()
		errorCounts, ok := stats["error_counts"].(map[string]*metrics.ErrorEntry)
		assert.True(t, ok)

		// Check that our error was recorded
		assert.Contains(t, errorCounts, "custom_error_type")
		assert.Equal(t, int64(1), errorCounts["custom_error_type"].Count)
		assert.Equal(t, "/custom-error", errorCounts["custom_error_type"].Path)
	})

	t.Run("Doesn't record metrics for successful requests", func(t *testing.T) {
		// Create a test request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/success", nil)

		// Count errors before
		statsBefore := mockMetrics.GetStats()
		errorCountsBefore := statsBefore["error_counts"].(map[string]*metrics.ErrorEntry)
		totalCountBefore := len(errorCountsBefore)

		// Serve the request
		r.ServeHTTP(w, req)

		// Count errors after
		statsAfter := mockMetrics.GetStats()
		errorCountsAfter := statsAfter["error_counts"].(map[string]*metrics.ErrorEntry)
		totalCountAfter := len(errorCountsAfter)

		// No new error types should be added
		assert.Equal(t, totalCountBefore, totalCountAfter)
	})
}

func TestGetErrorMetrics(t *testing.T) {
	// Get the metrics instance
	result := GetErrorMetrics()

	// Verify it's not nil
	require.NotNil(t, result)
}
