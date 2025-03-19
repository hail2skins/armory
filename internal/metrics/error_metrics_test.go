package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestErrorMetricsRecord(t *testing.T) {
	// Create a new error metrics instance
	em := NewErrorMetrics()

	// Record an error
	em.Record("validation_error", 400, 0.5, "/users")

	// Get stats
	stats := em.GetStats()

	// Check error counts
	errorCounts, ok := stats["error_counts"].(map[string]*ErrorEntry)
	assert.True(t, ok)
	assert.Equal(t, int64(1), errorCounts["validation_error"].Count)
	assert.Equal(t, "/users", errorCounts["validation_error"].Path)
	assert.Equal(t, 1, len(errorCounts["validation_error"].Latencies))
	assert.Equal(t, 0.5, errorCounts["validation_error"].Latencies[0])

	// Check status code counts
	statusCounts, ok := stats["status_counts"].(map[int]*ErrorEntry)
	assert.True(t, ok)
	assert.Equal(t, int64(1), statusCounts[400].Count)

	// Check endpoint counts
	endpointCounts, ok := stats["endpoint_counts"].(map[string]*ErrorEntry)
	assert.True(t, ok)
	assert.Equal(t, int64(1), endpointCounts["/users"].Count)
}

func TestMultipleErrorsAndLatencies(t *testing.T) {
	em := NewErrorMetrics()

	// Record multiple errors of the same type
	em.Record("validation_error", 400, 0.3, "/users")
	em.Record("validation_error", 400, 0.5, "/users")
	em.Record("validation_error", 400, 0.7, "/users")

	// Record a different error type
	em.Record("auth_error", 401, 0.2, "/login")

	// Get stats
	stats := em.GetStats()
	errorCounts := stats["error_counts"].(map[string]*ErrorEntry)

	// Check validation error counts
	assert.Equal(t, int64(3), errorCounts["validation_error"].Count)
	assert.Equal(t, 3, len(errorCounts["validation_error"].Latencies))

	// Check auth error counts
	assert.Equal(t, int64(1), errorCounts["auth_error"].Count)

	// Check average latency
	assert.InDelta(t, 0.5, errorCounts["validation_error"].AvgLatency(), 0.01)
}

func TestGetRecentErrors(t *testing.T) {
	em := NewErrorMetrics()

	// Record errors with different times
	now := time.Now()
	earlier := now.Add(-5 * time.Minute)

	// Manually record with specific timestamps
	em.recordWithTime("validation_error", 400, 0.3, "/users", earlier)
	em.recordWithTime("auth_error", 401, 0.2, "/login", now)

	// Get recent errors (limit to 2)
	recentErrors := em.GetRecentErrors(2)

	// Should be sorted by time (most recent first)
	assert.Equal(t, 2, len(recentErrors))
	assert.Equal(t, "auth_error", recentErrors[0].ErrorType)
	assert.Equal(t, "validation_error", recentErrors[1].ErrorType)
}

func TestCleanup(t *testing.T) {
	em := NewErrorMetrics()

	// Create an old error (1 day ago)
	oldTime := time.Now().Add(-24 * time.Hour)
	em.recordWithTime("old_error", 500, 0.3, "/old", oldTime)

	// Create a new error (just now)
	em.Record("new_error", 400, 0.2, "/new")

	// Cleanup errors older than 1 hour
	em.Cleanup(1 * time.Hour)

	// Get stats
	stats := em.GetStats()
	errorCounts := stats["error_counts"].(map[string]*ErrorEntry)

	// Old error should be cleared
	assert.Equal(t, int64(0), errorCounts["old_error"].Count)
	assert.Nil(t, errorCounts["old_error"].Latencies)
	assert.Nil(t, errorCounts["old_error"].Timestamps)

	// New error should still be there
	assert.Equal(t, int64(1), errorCounts["new_error"].Count)
}

func TestGetErrorRates(t *testing.T) {
	em := NewErrorMetrics()

	// Record errors at different times
	now := time.Now()
	em.recordWithTime("validation_error", 400, 0.3, "/users", now.Add(-5*time.Minute))
	em.recordWithTime("validation_error", 400, 0.3, "/users", now.Add(-15*time.Minute))
	em.recordWithTime("validation_error", 400, 0.3, "/users", now.Add(-25*time.Minute))
	em.recordWithTime("auth_error", 401, 0.2, "/login", now.Add(-2*time.Minute))

	// Get error rates for last 10 minutes
	rates := em.GetErrorRatesWithReference(10*time.Minute, now)

	// Should only include errors in the last 10 minutes
	assert.Equal(t, float64(1), rates["validation_error"])
	assert.Equal(t, float64(1), rates["auth_error"])
}

func TestPercentiles(t *testing.T) {
	em := NewErrorMetrics()

	// Record errors with a wide range of latencies
	for i := 0; i < 100; i++ {
		em.Record("test_error", 500, float64(i)/10.0, "/test")
	}

	// Get percentiles
	percentiles := em.GetLatencyPercentiles()

	// Check values with larger delta to account for implementation differences
	assert.InDelta(t, 4.9, percentiles["p50"], 0.5) // 50th percentile
	assert.InDelta(t, 9.4, percentiles["p95"], 0.5) // 95th percentile
	assert.InDelta(t, 9.8, percentiles["p99"], 0.5) // 99th percentile
}
