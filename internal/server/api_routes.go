package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/middleware"
)

// RegisterAPIRoutes registers all API related routes
func (s *Server) RegisterAPIRoutes(r *gin.Engine) {
	// Create an API group
	api := r.Group("/api")

	// Health check endpoint
	api.GET("/health", s.healthHandler)

	// Add admin-only routes with appropriate auth checks
	admin := api.Group("/admin")
	admin.Use(func(c *gin.Context) {
		// Get the auth service from context
		auth, exists := c.Get("auth")
		if !exists {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// Check if user is authenticated and is an admin
		if authService, ok := auth.(interface {
			IsAuthenticated(c *gin.Context) bool
			IsAdmin(c *gin.Context) bool
		}); ok {
			if !authService.IsAuthenticated(c) || !authService.IsAdmin(c) {
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
		} else {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.Next()
	})

	// Add rate limiting statistics endpoint
	admin.GET("/rate-limits", func(c *gin.Context) {
		stats := middleware.GetRateLimitStats()

		// Format timestamps for better readability
		formattedBlocks := make([]map[string]interface{}, 0, len(stats.RecentBlocks))
		for _, block := range stats.RecentBlocks {
			formattedBlocks = append(formattedBlocks, map[string]interface{}{
				"ip":         block.IP,
				"path":       block.Path,
				"time":       block.Timestamp.Format(time.RFC3339),
				"user_agent": block.UserAgent,
			})
		}

		// Format endpoint stats
		formattedEndpoints := make(map[string]map[string]interface{})
		for endpoint, info := range stats.EndpointStats {
			formattedEndpoints[endpoint] = map[string]interface{}{
				"total_attempts":   info.TotalAttempts,
				"blocked_attempts": info.BlockedAttempts,
				"last_block":       info.LastBlock.Format(time.RFC3339),
			}
		}

		// Format persistent abusers
		formattedAbusers := make(map[string]string)
		for ip, timestamp := range stats.PersistentAbusers {
			formattedAbusers[ip] = timestamp.Format(time.RFC3339)
		}

		c.JSON(http.StatusOK, gin.H{
			"total_attempts":     stats.TotalAttempts,
			"blocked_attempts":   stats.BlockedAttempts,
			"blocked_percentage": calculatePercentage(stats.BlockedAttempts, stats.TotalAttempts),
			"offenders":          stats.OffenderAttempts,
			"persistent_abusers": formattedAbusers,
			"recent_blocks":      formattedBlocks,
			"endpoints":          formattedEndpoints,
		})
	})

	// Add rate limit reset endpoint (admin only)
	admin.POST("/rate-limits/reset", func(c *gin.Context) {
		middleware.ResetRateLimitStats()
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "Rate limit statistics have been reset",
		})
	})

	// Add webhook health check endpoint
	admin.GET("/webhook/health", middleware.WebhookHealthCheck())

	// Add webhook stats reset endpoint (admin only)
	admin.POST("/webhook/reset", func(c *gin.Context) {
		middleware.ResetWebhookStats()
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "Webhook statistics have been reset",
		})
	})

	// Future API endpoints can be added here
	// api.GET("/users", userController.ListUsersHandler)
	// api.GET("/users/:id", userController.GetUserHandler)
	// etc.
}

// healthHandler returns the health status of the application
func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, s.db.Health())
}

// calculatePercentage calculates the percentage of part vs total
func calculatePercentage(part, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) / float64(total) * 100
}
