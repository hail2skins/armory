package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookMonitor(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Reset stats before test
	ResetWebhookStats()

	// Create a router with the webhook monitor middleware
	r := gin.New()
	r.Use(WebhookMonitor())

	// Add test routes for successful and failing requests
	r.GET("/webhook/success", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	r.GET("/webhook/error", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "test error"})
	})

	// Test a successful request
	t.Run("Successful request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/webhook/success", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Check stats were updated
		stats := GetWebhookStats()
		assert.Equal(t, int64(1), stats.TotalRequests)
		assert.Equal(t, int64(1), stats.SuccessfulRequests)
		assert.Equal(t, int64(0), stats.FailedRequests)
		assert.NotZero(t, stats.LastRequestTime)
		assert.True(t, stats.LastErrorTime.IsZero()) // Should still be zero
		assert.Empty(t, stats.LastError)
	})

	// Test an error request
	t.Run("Error request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/webhook/error", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		// Check stats were updated
		stats := GetWebhookStats()
		assert.Equal(t, int64(2), stats.TotalRequests)
		assert.Equal(t, int64(1), stats.SuccessfulRequests)
		assert.Equal(t, int64(1), stats.FailedRequests)
		assert.NotZero(t, stats.LastRequestTime)
		assert.NotZero(t, stats.LastErrorTime)
		assert.NotEmpty(t, stats.LastError)
		assert.Contains(t, stats.LastError, "test error")
	})
}

func TestWebhookHealthCheck(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a router with the health check endpoint
	r := gin.New()
	r.GET("/health", WebhookHealthCheck())

	tests := []struct {
		name               string
		setupStats         func()
		expectedStatus     string
		expectedSuccessful int64
		expectedFailed     int64
		expectedTotal      int64
	}{
		{
			name: "Healthy with traffic",
			setupStats: func() {
				ResetWebhookStats()
				webhookStats.mu.Lock()
				webhookStats.TotalRequests = 100
				webhookStats.SuccessfulRequests = 95
				webhookStats.FailedRequests = 5
				webhookStats.LastRequestTime = time.Now()
				webhookStats.mu.Unlock()
			},
			expectedStatus:     "healthy",
			expectedSuccessful: 95,
			expectedFailed:     5,
			expectedTotal:      100,
		},
		{
			name: "Unhealthy with high failure rate",
			setupStats: func() {
				ResetWebhookStats()
				webhookStats.mu.Lock()
				webhookStats.TotalRequests = 100
				webhookStats.SuccessfulRequests = 70
				webhookStats.FailedRequests = 30
				webhookStats.LastRequestTime = time.Now()
				webhookStats.mu.Unlock()
			},
			expectedStatus:     "unhealthy",
			expectedSuccessful: 70,
			expectedFailed:     30,
			expectedTotal:      100,
		},
		{
			name: "Degraded with no recent requests",
			setupStats: func() {
				ResetWebhookStats()
				webhookStats.mu.Lock()
				webhookStats.TotalRequests = 100
				webhookStats.SuccessfulRequests = 95
				webhookStats.FailedRequests = 5
				webhookStats.LastRequestTime = time.Now().Add(-25 * time.Hour) // More than 24 hours ago
				webhookStats.mu.Unlock()
			},
			expectedStatus:     "degraded",
			expectedSuccessful: 95,
			expectedFailed:     5,
			expectedTotal:      100,
		},
		{
			name: "No traffic yet",
			setupStats: func() {
				ResetWebhookStats()
			},
			expectedStatus:     "healthy",
			expectedSuccessful: 0,
			expectedFailed:     0,
			expectedTotal:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup the test stats
			tt.setupStats()

			// Get the health check
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/health", nil)
			r.ServeHTTP(w, req)

			// Verify response
			assert.Equal(t, http.StatusOK, w.Code)

			// Parse the response
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Check the status
			assert.Equal(t, tt.expectedStatus, response["status"])
			assert.Equal(t, float64(tt.expectedTotal), response["total_requests"])
			assert.Equal(t, float64(tt.expectedSuccessful), response["successful"])
			assert.Equal(t, float64(tt.expectedFailed), response["failed"])

			// Check success rate
			if tt.expectedTotal > 0 {
				expectedRate := float64(tt.expectedSuccessful) / float64(tt.expectedTotal) * 100
				assert.InDelta(t, expectedRate, response["success_rate"], 0.1)
			} else {
				assert.Equal(t, float64(0), response["success_rate"])
			}
		})
	}
}

func TestResetWebhookStats(t *testing.T) {
	// Set some values in the stats
	webhookStats.mu.Lock()
	webhookStats.TotalRequests = 100
	webhookStats.SuccessfulRequests = 90
	webhookStats.FailedRequests = 10
	webhookStats.LastError = "test error"
	webhookStats.LastRequestTime = time.Now()
	webhookStats.LastErrorTime = time.Now()
	webhookStats.mu.Unlock()

	// Reset stats
	ResetWebhookStats()

	// Verify stats were reset
	stats := GetWebhookStats()
	assert.Equal(t, int64(0), stats.TotalRequests)
	assert.Equal(t, int64(0), stats.SuccessfulRequests)
	assert.Equal(t, int64(0), stats.FailedRequests)
	assert.Empty(t, stats.LastError)

	// Time fields should not be reset to zero
	assert.False(t, stats.LastRequestTime.IsZero())
	assert.False(t, stats.LastErrorTime.IsZero())
}

func TestBodyLogWriter(t *testing.T) {
	// Create a test response writer using Gin's test context
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	// Create a bodyLogWriter wrapping the gin response writer
	writer := &bodyLogWriter{
		ResponseWriter: c.Writer,
		body:           []byte{},
	}

	// Test writing to the body
	testContent := []byte("Hello, webhook!")
	n, err := writer.Write(testContent)

	// Verify write was successful
	assert.NoError(t, err)
	assert.Equal(t, len(testContent), n)

	// Verify content was captured in the body
	assert.Equal(t, testContent, writer.body)
}

func TestGetWebhookStats(t *testing.T) {
	// Reset stats before test
	ResetWebhookStats()

	// Set some initial values
	initialTime := time.Now().Add(-time.Hour)
	webhookStats.mu.Lock()
	webhookStats.TotalRequests = 50
	webhookStats.SuccessfulRequests = 40
	webhookStats.FailedRequests = 10
	webhookStats.LastRequestTime = initialTime
	webhookStats.LastErrorTime = initialTime
	webhookStats.LastError = "initial error"
	webhookStats.mu.Unlock()

	// Get a copy of the stats
	stats := GetWebhookStats()

	// Verify the copy has the correct values
	assert.Equal(t, int64(50), stats.TotalRequests)
	assert.Equal(t, int64(40), stats.SuccessfulRequests)
	assert.Equal(t, int64(10), stats.FailedRequests)
	assert.Equal(t, initialTime, stats.LastRequestTime)
	assert.Equal(t, initialTime, stats.LastErrorTime)
	assert.Equal(t, "initial error", stats.LastError)

	// Modify the global stats
	webhookStats.mu.Lock()
	webhookStats.TotalRequests = 100
	webhookStats.mu.Unlock()

	// Verify the copy was not affected
	assert.Equal(t, int64(50), stats.TotalRequests)

	// Get a new copy and verify it has the updated value
	newStats := GetWebhookStats()
	assert.Equal(t, int64(100), newStats.TotalRequests)
}
