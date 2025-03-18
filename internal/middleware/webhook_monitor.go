package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/logger"
)

// WebhookStats tracks statistics about webhook calls
type WebhookStats struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	LastRequestTime    time.Time
	LastErrorTime      time.Time
	LastError          string
	mu                 sync.Mutex
}

var webhookStats = WebhookStats{}

// GetWebhookStats returns the current webhook statistics
func GetWebhookStats() WebhookStats {
	webhookStats.mu.Lock()
	defer webhookStats.mu.Unlock()

	// Return a copy to prevent concurrent modification
	stats := WebhookStats{
		TotalRequests:      webhookStats.TotalRequests,
		SuccessfulRequests: webhookStats.SuccessfulRequests,
		FailedRequests:     webhookStats.FailedRequests,
		LastRequestTime:    webhookStats.LastRequestTime,
		LastErrorTime:      webhookStats.LastErrorTime,
		LastError:          webhookStats.LastError,
	}

	return stats
}

// ResetWebhookStats resets all webhook statistics to zero
func ResetWebhookStats() {
	webhookStats.mu.Lock()
	defer webhookStats.mu.Unlock()

	webhookStats.TotalRequests = 0
	webhookStats.SuccessfulRequests = 0
	webhookStats.FailedRequests = 0
	webhookStats.LastError = ""
	// Don't reset time fields to zero as they would be invalid
}

// WebhookMonitor middleware tracks webhook health and metrics
func WebhookMonitor() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Record start time
		startTime := time.Now()

		// Create a response writer that captures the status code
		blw := &bodyLogWriter{body: []byte{}, ResponseWriter: c.Writer}
		c.Writer = blw

		// Process request
		c.Next()

		// Update stats after request is processed
		webhookStats.mu.Lock()
		defer webhookStats.mu.Unlock()

		webhookStats.TotalRequests++
		webhookStats.LastRequestTime = startTime

		// Check if request was successful
		if blw.Status() >= 200 && blw.Status() < 300 {
			webhookStats.SuccessfulRequests++
		} else {
			webhookStats.FailedRequests++
			webhookStats.LastErrorTime = startTime
			webhookStats.LastError = string(blw.body)

			// Log webhook errors
			logger.Error("Webhook error", nil, map[string]interface{}{
				"status": blw.Status(),
				"error":  string(blw.body),
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
			})
		}

		// Log request duration for monitoring
		duration := time.Since(startTime)
		logger.Info("Webhook request", map[string]interface{}{
			"path":     c.Request.URL.Path,
			"method":   c.Request.Method,
			"status":   blw.Status(),
			"duration": duration.String(),
		})
	}
}

// bodyLogWriter captures the response body and status code
type bodyLogWriter struct {
	gin.ResponseWriter
	body []byte
}

// Write captures the response body
func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return w.ResponseWriter.Write(b)
}

// WebhookHealthCheck returns a handler that checks webhook health
func WebhookHealthCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		stats := GetWebhookStats()

		// Calculate success rate
		var successRate float64 = 0
		if stats.TotalRequests > 0 {
			successRate = float64(stats.SuccessfulRequests) / float64(stats.TotalRequests) * 100
		}

		// Check if webhook is healthy
		isHealthy := true
		healthStatus := "healthy"

		// If we've had requests and the success rate is below 80%, consider unhealthy
		if stats.TotalRequests > 0 && successRate < 80 {
			isHealthy = false
			healthStatus = "unhealthy"
		}

		// If it's been more than 24 hours since the last request, consider degraded
		// But only if we've had at least one request before
		if stats.TotalRequests > 0 && (stats.LastRequestTime.IsZero() || time.Since(stats.LastRequestTime) > 24*time.Hour) {
			if isHealthy {
				healthStatus = "degraded"
			}
		}

		// Return health status
		c.JSON(http.StatusOK, gin.H{
			"status":            healthStatus,
			"total_requests":    stats.TotalRequests,
			"successful":        stats.SuccessfulRequests,
			"failed":            stats.FailedRequests,
			"success_rate":      successRate,
			"last_request":      stats.LastRequestTime,
			"last_error":        stats.LastErrorTime,
			"last_error_detail": stats.LastError,
		})
	}
}
