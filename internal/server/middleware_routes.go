package server

import (
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/logger"
	"github.com/hail2skins/armory/internal/middleware"
)

// Add security headers middleware
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Protection against clickjacking
		c.Header("X-Frame-Options", "DENY")

		// Protection against MIME-type confusion attacks
		c.Header("X-Content-Type-Options", "nosniff")

		// Protection against XSS attacks (for older browsers)
		c.Header("X-XSS-Protection", "1; mode=block")

		// Enforce HTTPS (only in production)
		if os.Getenv("GO_ENV") == "production" {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Content Security Policy
		// Updated to allow Tailwind CDN and other necessary resources
		csp := []string{
			"default-src 'self'",
			"script-src 'self' https://cdn.tailwindcss.com https://cdn.jsdelivr.net 'unsafe-inline'",
			"style-src 'self' https://cdn.tailwindcss.com https://fonts.googleapis.com 'unsafe-inline'",
			"img-src 'self' data: https:",
			"font-src 'self' https://fonts.gstatic.com",
			"connect-src 'self'",
			"frame-src 'self' https://buy.stripe.com",
		}

		c.Header("Content-Security-Policy", strings.Join(csp, "; "))

		c.Next()
	}
}

// getCorsOrigins returns the allowed origins for CORS based on environment
func getCorsOrigins() []string {
	// Check for a specific environment variable for CORS origins
	corsOrigins := os.Getenv("CORS_ORIGINS")
	if corsOrigins != "" {
		// Split the origins by comma
		origins := strings.Split(corsOrigins, ",")
		// Trim whitespace
		for i, origin := range origins {
			origins[i] = strings.TrimSpace(origin)
		}
		return origins
	}

	// Environment-specific defaults
	env := os.Getenv("APP_ENV")
	if env == "production" {
		// Default for production - should be overridden with CORS_ORIGINS in actual production
		logger.Warn("CORS_ORIGINS not set in production environment. Using default configuration.", nil)
		return []string{"https://armory.example.com"}
	} else if env == "test" {
		// For tests, allow any origin
		return []string{"*"}
	}

	// Default for development
	return []string{"http://localhost:5173", "http://localhost:3000", "http://localhost:8080"}
}

// RegisterMiddleware sets up all middleware for the application
func (s *Server) RegisterMiddleware(r *gin.Engine, authController *controller.AuthController) {
	// Set up error handling middleware
	middleware.SetupErrorHandling(r)

	// Add error metrics to the context for admin routes
	r.Use(func(c *gin.Context) {
		// Get the error metrics instance
		errorMetrics := middleware.GetErrorMetrics()

		// Add to context
		c.Set("errorMetrics", errorMetrics)

		c.Next()
	})

	// Apply security headers
	r.Use(securityHeaders())

	// Set up flash message middleware - moved before rate limiting
	r.Use(func(c *gin.Context) {
		// Set up a function to set flash messages
		c.Set("setFlash", func(message string) {
			// Set the flash message in a cookie
			c.SetCookie("flash", message, 10, "/", "", false, false)
		})
		c.Next()
	})

	// Set up rate limiting middleware - moved after flash middleware
	middleware.SetupRateLimiting(r)

	// Apply webhook monitoring to webhook endpoints
	r.Use(func(c *gin.Context) {
		// Apply webhook monitoring only to the webhook endpoint
		if strings.HasPrefix(c.Request.URL.Path, "/webhook") {
			middleware.WebhookMonitor()(c)
		} else {
			c.Next()
		}
	})

	// Set up CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     getCorsOrigins(), // Get origins from environment
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true, // Enable cookies/auth
	}))

	// Initialize Casbin (store in the server instance for admin routes to use)
	casbinAuth, err := middleware.SetupCasbin()
	if err != nil {
		// Log the error but continue (admin routes will check for nil)
		logger.Warn("Casbin setup failed, RBAC will be disabled", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Configure initial admin users
	adminEmails := getAdminEmails()
	if casbinAuth != nil && len(adminEmails) > 0 {
		if err := middleware.ConfigureInitialRoles(casbinAuth, adminEmails); err != nil {
			logger.Warn("Failed to configure initial admin roles", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	// Store the casbin auth in the server for admin routes to use
	s.casbinAuth = casbinAuth

	// Initialize promotion service
	logger.Info("Initializing promotion service", nil)
	promotionService := s.createPromotionService()

	// Set promotion service on auth controller
	logger.Info("Setting promotion service on auth controller", nil)
	authController.SetPromotionService(promotionService)

	// Initialize auth data middleware first (without active promotion yet)
	authMiddleware := middleware.AuthMiddleware(authController)

	// Add the promotion banner middleware
	logger.Info("Adding promotion banner middleware", nil)
	r.Use(middleware.PromotionBanner(promotionService))

	// Now apply the auth middleware after the promotion middleware has run
	r.Use(authMiddleware)

	// Set up auth compatibility middleware (controller in context)
	r.Use(func(c *gin.Context) {
		// Make casbinAuth available in the context
		c.Set("casbinAuth", casbinAuth)

		// Set both auth keys for compatibility - the new pattern uses "auth"
		// while some existing code might still use "authController"
		c.Set("auth", authController)
		c.Set("authController", authController)

		c.Next()
	})
}

// getAdminEmails returns the list of admin emails from environment variables or configuration
func getAdminEmails() []string {
	// Get admin emails from environment variable
	adminEmail := os.Getenv("CASBIN_ADMIN")
	if adminEmail != "" {
		// Split by comma if multiple emails are provided
		if strings.Contains(adminEmail, ",") {
			return strings.Split(adminEmail, ",")
		}
		return []string{adminEmail}
	}

	// Check if we're in test mode
	if os.Getenv("GO_ENV") == "test" {
		// In test mode, use a default test admin
		return []string{"test@example.com"}
	}

	// Log a warning that no admin email is configured (only in non-test environments)
	logger.Warn("No CASBIN_ADMIN environment variable set. Admin functionality will be unavailable until configured.", nil)

	// Return an empty slice - no default admin is set
	return []string{}
}
