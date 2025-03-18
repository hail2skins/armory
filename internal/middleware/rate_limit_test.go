package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_RateLimit(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a rate limiter with a limit of 2 requests per second
	// (not used directly - we create a new one in each test case)
	_ = NewRateLimiter()

	tests := []struct {
		name           string
		path           string
		userAgent      string
		requests       int
		expectedStatus []int
	}{
		{
			name:           "Normal requests under limit",
			path:           "/test",
			userAgent:      "Test/Agent",
			requests:       2,
			expectedStatus: []int{http.StatusOK, http.StatusOK},
		},
		{
			name:           "Requests exceeding limit",
			path:           "/test",
			userAgent:      "Test/Agent",
			requests:       3,
			expectedStatus: []int{http.StatusOK, http.StatusOK, http.StatusTooManyRequests},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new rate limiter for each test to prevent shared state
			rl := NewRateLimiter()
			middleware := rl.RateLimit(2, time.Second)

			// Create a router for each test
			router := gin.New()
			router.Use(middleware)
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "success")
			})

			// Make multiple requests
			for i := 0; i < tt.requests; i++ {
				req, _ := http.NewRequest("GET", tt.path, nil)
				req.Header.Set("User-Agent", tt.userAgent)
				req.RemoteAddr = "192.168.1.100:12345" // Set the IP address

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				// Check the status code
				assert.Equal(t, tt.expectedStatus[i], w.Code, "Unexpected status code on request %d", i+1)
			}
		})
	}
}

func TestRateLimiter_StripeWebhook(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a rate limiter with a low limit
	rl := NewRateLimiter()
	middleware := rl.RateLimit(1, time.Second)

	// Create router
	router := gin.New()
	router.Use(middleware)
	router.POST("/webhook", func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})

	// Test Stripe webhook requests - should not be rate limited
	t.Run("Stripe webhook not rate limited", func(t *testing.T) {
		// Make multiple requests with Stripe user agent
		for i := 0; i < 3; i++ {
			req, _ := http.NewRequest("POST", "/webhook", nil)
			req.Header.Set("User-Agent", "Stripe/1.0")
			req.RemoteAddr = "192.168.1.1:12345"

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// All requests should succeed regardless of rate limit
			assert.Equal(t, http.StatusOK, w.Code, "Stripe webhook request should not be rate limited")
		}
	})

	// Test non-Stripe requests to webhook endpoint - should be rate limited
	t.Run("Non-Stripe webhook rate limited", func(t *testing.T) {
		expectedStatus := []int{http.StatusOK, http.StatusTooManyRequests}

		// Make multiple requests with non-Stripe user agent
		for i := 0; i < 2; i++ {
			req, _ := http.NewRequest("POST", "/webhook", nil)
			req.Header.Set("User-Agent", "Mozilla/5.0")
			req.RemoteAddr = "192.168.1.2:12345"

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Check that rate limiting is applied
			assert.Equal(t, expectedStatus[i], w.Code, "Non-Stripe webhook request should be rate limited after limit exceeded")
		}
	})
}

func TestLoginRateLimit(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a rate limiter
	rl := NewRateLimiter()

	// Check if the path is detected as a user-facing route
	if !isUserFacingRoute("/login") {
		t.Fatal("Error: /login should be detected as a user-facing route")
	} else {
		t.Log("Successfully detected /login as a user-facing route")
	}

	middleware := rl.LoginRateLimit()

	// Create router
	router := gin.New()

	// Set up flash function
	router.Use(func(c *gin.Context) {
		// Create a mock flash message function that stores the flash in context
		c.Set("setFlash", func(message string) {
			t.Logf("Flash message set: %s", message)
			c.Set("flash_message", message)
		})
		c.Next()
	})

	router.Use(middleware)
	router.POST("/login", func(c *gin.Context) {
		t.Logf("Handling login request, path: %s", c.Request.URL.Path)
		c.String(http.StatusOK, "success")
	})

	// Test login rate limit (should allow 5 requests per minute)
	t.Run("Login rate limit", func(t *testing.T) {
		expectedStatus := []int{
			http.StatusOK, http.StatusOK, http.StatusOK, http.StatusOK, http.StatusOK,
			http.StatusSeeOther, // Changed from 429 to 303 for redirect
		}

		// Make multiple requests
		for i := 0; i < 6; i++ {
			req, _ := http.NewRequest("POST", "/login", nil)
			req.RemoteAddr = "192.168.1.3:12345"

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			t.Logf("Request %d - Status: %d, Path: %s", i+1, w.Code, req.URL.Path)
			if i == 5 {
				t.Logf("Headers: %v", w.Header())
			}

			// Check status code
			assert.Equal(t, expectedStatus[i], w.Code, "Unexpected status on login request %d", i+1)

			// Check for flash message on rate limit exceeded
			if i == 5 {
				// For user-facing routes we now redirect with a flash message
				assert.Equal(t, "/", w.Header().Get("Location"), "Should redirect to home page")
				// We can't easily check the flash message value in this test setup
			}
		}
	})
}

func TestPasswordResetRateLimit(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a rate limiter
	rl := NewRateLimiter()
	middleware := rl.PasswordResetRateLimit()

	// Create router
	router := gin.New()

	// Set up flash function
	router.Use(func(c *gin.Context) {
		// Create a mock flash message function that stores the flash in context
		c.Set("setFlash", func(message string) {
			c.Set("flash_message", message)
		})
		c.Next()
	})

	router.Use(middleware)
	router.GET("/reset-password", func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})

	// Test password reset rate limit (should allow 3 requests per hour)
	t.Run("Password reset rate limit", func(t *testing.T) {
		expectedStatus := []int{http.StatusOK, http.StatusOK, http.StatusOK, http.StatusSeeOther}

		// Make multiple requests
		for i := 0; i < 4; i++ {
			req, _ := http.NewRequest("GET", "/reset-password", nil)
			req.RemoteAddr = "192.168.1.4:12345"

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Check status code
			assert.Equal(t, expectedStatus[i], w.Code, "Unexpected status on password reset request %d", i+1)

			// Check for flash message on rate limit exceeded
			if i == 3 {
				// For user-facing routes we now redirect with a flash message
				assert.Equal(t, "/", w.Header().Get("Location"), "Should redirect to home page")
				// We can't easily check the flash message value in this test setup
			}
		}
	})
}

func TestWebhookRateLimit(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a rate limiter
	rl := NewRateLimiter()
	middleware := rl.WebhookRateLimit()

	// Create router
	router := gin.New()
	router.Use(middleware)
	router.POST("/api/webhook", func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})

	// Test webhook rate limit (should allow 10 requests per minute)
	t.Run("Webhook rate limit", func(t *testing.T) {
		// Make 10 successful requests
		for i := 0; i < 10; i++ {
			req, _ := http.NewRequest("POST", "/api/webhook", nil)
			req.RemoteAddr = "192.168.1.5:12345"
			req.Header.Set("User-Agent", "TestAgent/1.0") // Not Stripe

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i+1)
		}

		// 11th request should be rate limited
		req, _ := http.NewRequest("POST", "/api/webhook", nil)
		req.RemoteAddr = "192.168.1.5:12345"
		req.Header.Set("User-Agent", "TestAgent/1.0")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTooManyRequests, w.Code, "Request should be rate limited")
	})
}
