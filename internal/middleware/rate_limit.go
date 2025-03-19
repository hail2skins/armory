package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/errors"
	"github.com/hail2skins/armory/internal/logger"
)

// RateLimitStats tracks statistics about rate limiting
type RateLimitStats struct {
	TotalAttempts     int64                       // Total attempts tracked
	BlockedAttempts   int64                       // Total blocked attempts
	OffenderAttempts  map[string]int              // Track attempts by IP
	PersistentAbusers map[string]time.Time        // Track IPs that consistently abuse limits
	RecentBlocks      []RateLimitBlock            // Recent blocks for monitoring
	EndpointStats     map[string]EndpointRateInfo // Stats per endpoint
	mu                sync.Mutex
}

// RateLimitBlock represents a single rate limit block event
type RateLimitBlock struct {
	IP        string
	Path      string
	Timestamp time.Time
	UserAgent string
}

// EndpointRateInfo tracks rate limiting info per endpoint
type EndpointRateInfo struct {
	TotalAttempts   int64
	BlockedAttempts int64
	LastBlock       time.Time
}

// Global stats tracker
var rateLimitStats = RateLimitStats{
	OffenderAttempts:  make(map[string]int),
	PersistentAbusers: make(map[string]time.Time),
	RecentBlocks:      make([]RateLimitBlock, 0, 100), // Keep last 100 blocks
	EndpointStats:     make(map[string]EndpointRateInfo),
}

// GetRateLimitStats returns a copy of the current rate limit statistics
func GetRateLimitStats() RateLimitStats {
	rateLimitStats.mu.Lock()
	defer rateLimitStats.mu.Unlock()

	// Return a copy to prevent concurrent modification
	stats := RateLimitStats{
		TotalAttempts:     rateLimitStats.TotalAttempts,
		BlockedAttempts:   rateLimitStats.BlockedAttempts,
		OffenderAttempts:  make(map[string]int, len(rateLimitStats.OffenderAttempts)),
		PersistentAbusers: make(map[string]time.Time, len(rateLimitStats.PersistentAbusers)),
		RecentBlocks:      make([]RateLimitBlock, len(rateLimitStats.RecentBlocks)),
		EndpointStats:     make(map[string]EndpointRateInfo, len(rateLimitStats.EndpointStats)),
	}

	// Copy maps and slices to prevent concurrent modification
	for k, v := range rateLimitStats.OffenderAttempts {
		stats.OffenderAttempts[k] = v
	}

	for k, v := range rateLimitStats.PersistentAbusers {
		stats.PersistentAbusers[k] = v
	}

	copy(stats.RecentBlocks, rateLimitStats.RecentBlocks)

	for k, v := range rateLimitStats.EndpointStats {
		stats.EndpointStats[k] = v
	}

	return stats
}

// ResetRateLimitStats resets all rate limiting statistics
func ResetRateLimitStats() {
	rateLimitStats.mu.Lock()
	defer rateLimitStats.mu.Unlock()

	rateLimitStats.TotalAttempts = 0
	rateLimitStats.BlockedAttempts = 0
	rateLimitStats.OffenderAttempts = make(map[string]int)
	rateLimitStats.PersistentAbusers = make(map[string]time.Time)
	rateLimitStats.RecentBlocks = make([]RateLimitBlock, 0, 100)
	rateLimitStats.EndpointStats = make(map[string]EndpointRateInfo)
}

// track records an attempt in the statistics
func track(clientIP, path string, blocked bool, userAgent string) {
	rateLimitStats.mu.Lock()
	defer rateLimitStats.mu.Unlock()

	// Track total attempts
	rateLimitStats.TotalAttempts++

	// Update endpoint stats
	info, exists := rateLimitStats.EndpointStats[path]
	if !exists {
		info = EndpointRateInfo{}
	}
	info.TotalAttempts++

	// If blocked, record more detailed information
	if blocked {
		// Track blocked attempts
		rateLimitStats.BlockedAttempts++
		info.BlockedAttempts++
		info.LastBlock = time.Now()

		// Track offender
		rateLimitStats.OffenderAttempts[clientIP]++

		// If this IP has been blocked more than 10 times, consider it a persistent abuser
		if rateLimitStats.OffenderAttempts[clientIP] > 10 {
			rateLimitStats.PersistentAbusers[clientIP] = time.Now()
		}

		// Record this block in recent blocks
		block := RateLimitBlock{
			IP:        clientIP,
			Path:      path,
			Timestamp: time.Now(),
			UserAgent: userAgent,
		}

		// Keep last 100 blocks
		if len(rateLimitStats.RecentBlocks) >= 100 {
			rateLimitStats.RecentBlocks = append(rateLimitStats.RecentBlocks[1:], block)
		} else {
			rateLimitStats.RecentBlocks = append(rateLimitStats.RecentBlocks, block)
		}
	}

	// Update endpoint stats
	rateLimitStats.EndpointStats[path] = info
}

type RateLimiter struct {
	requests map[string][]time.Time
	mu       sync.Mutex
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
	}
}

// isStripeWebhook checks if the request is from Stripe
func isStripeWebhook(c *gin.Context) bool {
	userAgent := c.GetHeader("User-Agent")
	return strings.HasPrefix(userAgent, "Stripe/")
}

// isUserFacingRoute checks if this is a route that would be accessed directly by users in a browser
func isUserFacingRoute(path string) bool {
	userFacingPaths := []string{
		"/login",
		"/register",
		"/reset-password",
	}

	for _, p := range userFacingPaths {
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}

	return false
}

// getFlashFunction attempts to get the flash message function from the context
func getFlashFunction(c *gin.Context) (func(string), bool) {
	if fn, exists := c.Get("setFlash"); exists {
		if setFlash, ok := fn.(func(string)); ok {
			return setFlash, true
		}
	}
	return nil, false
}

// RateLimit creates middleware that limits requests per client
// limit: number of requests allowed
// duration: time window for the limit (e.g., 1 minute)
func (rl *RateLimiter) RateLimit(limit int, duration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the request path (NOT using c.FullPath() as it may be empty or different)
		path := c.Request.URL.Path

		// Skip rate limiting for Stripe webhooks
		if path == "/webhook" && isStripeWebhook(c) {
			c.Next()
			return
		}

		// Use IP address as client identifier
		clientIP := c.ClientIP()

		// Add path to identifier for separate limits per endpoint
		identifier := fmt.Sprintf("%s:%s", clientIP, path)

		rl.mu.Lock()
		defer rl.mu.Unlock()

		// Clean old requests
		now := time.Now()
		windowStart := now.Add(-duration)

		// Get existing requests for this client
		times := rl.requests[identifier]
		valid := make([]time.Time, 0)

		// Keep only requests within our time window
		for _, t := range times {
			if t.After(windowStart) {
				valid = append(valid, t)
			}
		}

		// Update requests for this client
		rl.requests[identifier] = valid

		// Track this attempt in stats
		track(clientIP, path, len(valid) >= limit, c.GetHeader("User-Agent"))

		// Check if limit exceeded
		if len(valid) >= limit {
			errorMessage := fmt.Sprintf("Rate limit exceeded. Try again in %v", duration)
			err := errors.NewRateLimitError(errorMessage)

			// Log the rate limit error with detailed information
			logger.Error("Rate limit exceeded", err, map[string]interface{}{
				"ip":         clientIP,
				"path":       path,
				"limit":      limit,
				"duration":   duration.String(),
				"user_agent": c.GetHeader("User-Agent"),
				"attempts":   len(valid) + 1,
			})

			// First check if this is a user-facing route
			isUserRoute := isUserFacingRoute(path)

			// For user-facing routes, use a flash message and redirect
			if isUserRoute {
				// Try to set a flash message if the function is available
				if setFlash, exists := getFlashFunction(c); exists {
					setFlash(errorMessage)
					c.Redirect(http.StatusSeeOther, "/")
					c.Abort()
					c.Error(err) // Record the error
					return
				}

				// If no flash function, use a plain HTML message
				c.Header("Content-Type", "text/html; charset=utf-8")
				c.String(http.StatusTooManyRequests, "<html><body><h1>Rate Limit Exceeded</h1><p>%s</p><p><a href=\"/\">Return to home page</a></p></body></html>", errorMessage)
				c.Abort()
				c.Error(err)
				return
			}

			// For API routes, return a JSON response
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": errorMessage,
			})
			c.Error(err)
			return
		}

		// Add current request
		rl.requests[identifier] = append(rl.requests[identifier], now)

		c.Next()
	}
}

// LoginRateLimit creates middleware specifically for login attempts
func (rl *RateLimiter) LoginRateLimit() gin.HandlerFunc {
	return rl.RateLimit(5, time.Minute)
}

// RegisterRateLimit creates middleware specifically for registration attempts
func (rl *RateLimiter) RegisterRateLimit() gin.HandlerFunc {
	return rl.RateLimit(5, time.Minute)
}

// PasswordResetRateLimit creates middleware specifically for password reset attempts
func (rl *RateLimiter) PasswordResetRateLimit() gin.HandlerFunc {
	return rl.RateLimit(3, time.Hour)
}

// WebhookRateLimit creates middleware specifically for webhook endpoints
func (rl *RateLimiter) WebhookRateLimit() gin.HandlerFunc {
	return rl.RateLimit(10, time.Minute)
}
