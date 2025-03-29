package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/middleware"
	"github.com/hail2skins/armory/internal/testutils"
	"github.com/stretchr/testify/assert"
)

// mockServer creates a test server with the necessary middleware
func mockServer(t *testing.T) (*gin.Engine, database.Service) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create test database service
	testDB := testutils.NewTestDB()
	db := testutils.NewTestService(testDB.DB)

	// Create a new gin router
	r := gin.New()

	// Create controllers
	authController := controller.NewAuthController(db)

	// Set up middleware
	middleware.SetupErrorHandling(r)
	middleware.SetupRateLimiting(r)

	// Set up auth context middleware
	r.Use(func(c *gin.Context) {
		c.Set("auth", authController)
		c.Set("authController", authController)

		// Add flash message function for rate limiting
		c.Set("setFlash", func(message string) {
			c.Set("flash_message", message)
		})

		c.Next()
	})

	// Set up test routes
	r.POST("/login", func(c *gin.Context) {
		c.String(http.StatusOK, "login success")
	})
	r.POST("/register", func(c *gin.Context) {
		c.String(http.StatusOK, "register success")
	})
	r.GET("/reset-password", func(c *gin.Context) {
		c.String(http.StatusOK, "reset password success")
	})
	r.POST("/webhook", func(c *gin.Context) {
		c.String(http.StatusOK, "webhook success")
	})

	return r, db
}

func TestRateLimitingIntegration(t *testing.T) {
	r, _ := mockServer(t)

	// Test rate limiting for login endpoint
	t.Run("Login rate limiting", func(t *testing.T) {
		// Make 5 successful requests
		for i := 0; i < 5; i++ {
			req, _ := http.NewRequest(http.MethodPost, "/login", nil)
			req.RemoteAddr = "192.168.1.10:12345"
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i+1)
		}

		// 6th request should be rate limited
		req, _ := http.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = "192.168.1.10:12345"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code, "Request should be rate limited")
	})

	// Test rate limiting for register endpoint
	t.Run("Register rate limiting", func(t *testing.T) {
		// Make 5 successful requests
		for i := 0; i < 5; i++ {
			req, _ := http.NewRequest(http.MethodPost, "/register", nil)
			req.RemoteAddr = "192.168.1.11:12345"
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i+1)
		}

		// 6th request should be rate limited
		req, _ := http.NewRequest(http.MethodPost, "/register", nil)
		req.RemoteAddr = "192.168.1.11:12345"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code, "Request should be rate limited")
	})

	// Test rate limiting for password reset endpoint
	t.Run("Password reset rate limiting", func(t *testing.T) {
		// Make 3 successful requests
		for i := 0; i < 3; i++ {
			req, _ := http.NewRequest(http.MethodGet, "/reset-password", nil)
			req.RemoteAddr = "192.168.1.12:12345"
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i+1)
		}

		// 4th request should be rate limited
		req, _ := http.NewRequest(http.MethodGet, "/reset-password", nil)
		req.RemoteAddr = "192.168.1.12:12345"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code, "Request should be rate limited")
	})

	// Test Stripe webhook exemption
	t.Run("Stripe webhook exemption", func(t *testing.T) {
		// Make multiple requests with Stripe user agent
		for i := 0; i < 15; i++ {
			req, _ := http.NewRequest(http.MethodPost, "/webhook", nil)
			req.Header.Set("User-Agent", "Stripe/1.0")
			req.RemoteAddr = "192.168.1.13:12345"
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code, "Stripe webhook request %d should not be rate limited", i+1)
		}
	})

	// Test non-Stripe webhook rate limiting
	t.Run("Non-Stripe webhook rate limiting", func(t *testing.T) {
		// Make 10 successful requests
		for i := 0; i < 10; i++ {
			req, _ := http.NewRequest(http.MethodPost, "/webhook", nil)
			req.Header.Set("User-Agent", "Mozilla/5.0")
			req.RemoteAddr = "192.168.1.14:12345"
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i+1)
		}

		// 11th request should be rate limited
		req, _ := http.NewRequest(http.MethodPost, "/webhook", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")
		req.RemoteAddr = "192.168.1.14:12345"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code, "Request should be rate limited")
	})
}
