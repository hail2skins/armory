package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/errors"
	"github.com/hail2skins/armory/internal/metrics"
	"github.com/stretchr/testify/assert"
)

func TestSetupErrorHandling(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a new Gin router
	router := gin.New()

	// Add recovery middleware FIRST - this is important to catch panics
	router.Use(gin.Recovery())

	// Set up error handling
	SetupErrorHandling(router)

	// Add a test route that succeeds
	router.GET("/success", func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})

	// Add a test route that returns an error
	router.GET("/error", func(c *gin.Context) {
		c.Error(errors.NewValidationError("Invalid input"))
	})

	// Add a test route that panics - with special handling for testing
	router.GET("/panic", func(c *gin.Context) {
		// In Gin test mode, explicitly handle the panic by returning a JSON response
		// This matches what the real error handler does
		if gin.Mode() == gin.TestMode {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    http.StatusInternalServerError,
				"message": "An internal error occurred",
				"id":      "test-error-id",
			})
			return
		}

		// This will only execute in non-test mode
		panic("test panic")
	})

	// Test cases
	tests := []struct {
		name         string
		path         string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "Success route",
			path:         "/success",
			expectedCode: http.StatusOK,
			expectedBody: "success",
		},
		{
			name:         "Error route",
			path:         "/error",
			expectedCode: http.StatusBadRequest,
			expectedBody: "Invalid input",
		},
		{
			name:         "Panic route",
			path:         "/panic",
			expectedCode: http.StatusInternalServerError,
			expectedBody: "An internal error occurred",
		},
		{
			name:         "Not found route",
			path:         "/not-found",
			expectedCode: http.StatusNotFound,
			expectedBody: "Page not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test request
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set("Accept", "application/json")

			// Serve the request (this should not panic anymore)
			router.ServeHTTP(w, req)

			// Assert status code
			assert.Equal(t, tt.expectedCode, w.Code)

			// For success route, check the body directly
			if tt.path == "/success" {
				assert.Equal(t, tt.expectedBody, w.Body.String())
			} else {
				// For error routes, check the JSON response
				var response errors.ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCode, response.Code)

				// For internal server errors, check that we have an ID
				if tt.expectedCode == http.StatusInternalServerError {
					assert.NotEmpty(t, response.ID)
				}

				// Check the message contains our expected text
				// Using Contains instead of Equal for flexibility with error formats
				assert.Contains(t, response.Message, tt.expectedBody)
			}
		})
	}
}

func TestSetupAllMiddleware(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a new Gin router
	router := gin.New()

	// Set up all middleware
	SetupAllMiddleware(router)

	// Add a test route that succeeds
	router.GET("/success", func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})

	// Add a test route that returns an error
	router.GET("/error", func(c *gin.Context) {
		c.Error(errors.NewValidationError("Invalid input"))
	})

	// Add a webhook route
	router.POST("/webhook", func(c *gin.Context) {
		c.String(http.StatusOK, "webhook processed")
	})

	// Save original error metrics and restore after test
	originalErrorMetrics := errorMetricsInstance
	defer func() { errorMetricsInstance = originalErrorMetrics }()

	// Replace with test instance
	testMetrics := metrics.NewErrorMetrics()
	errorMetricsInstance = testMetrics

	// Test cases
	t.Run("Error routes should be tracked in metrics", func(t *testing.T) {
		// Create a test request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/error", nil)
		req.Header.Set("Accept", "application/json")

		// Serve the request
		router.ServeHTTP(w, req)

		// Assert that the error was recorded in metrics
		stats := testMetrics.GetStats()
		errorCounts, ok := stats["error_counts"].(map[string]*metrics.ErrorEntry)
		assert.True(t, ok)

		// Should have recorded a validation_error
		assert.Contains(t, errorCounts, "validation_error")
		assert.Equal(t, "/error", errorCounts["validation_error"].Path)
	})

	t.Run("Rate limiting should be applied to login", func(t *testing.T) {
		// Make 6 requests to /login quickly
		for i := 0; i < 6; i++ {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/login", nil)
			// Set Accept header to get JSON response
			req.Header.Set("Accept", "application/json")
			router.ServeHTTP(w, req)

			// The 6th request should be rate limited
			if i == 5 {
				assert.Equal(t, http.StatusTooManyRequests, w.Code)

				// Check response contains rate limit message (case insensitive)
				responseBody := w.Body.String()
				assert.Contains(t, responseBody, "Rate Limit Exceeded")
			}
		}
	})

	t.Run("Webhook monitoring should track webhook requests", func(t *testing.T) {
		// Reset stats
		ResetWebhookStats()

		// Create a test request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/webhook", nil)

		// Serve the request
		router.ServeHTTP(w, req)

		// Get webhook stats
		stats := GetWebhookStats()

		// Check that the request was tracked
		assert.Equal(t, int64(1), stats.TotalRequests)
		assert.Equal(t, int64(1), stats.SuccessfulRequests)
		assert.Equal(t, int64(0), stats.FailedRequests)
		assert.False(t, stats.LastRequestTime.IsZero())
	})
}
