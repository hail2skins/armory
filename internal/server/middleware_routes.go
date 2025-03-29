package server

import (
	"log"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/logger"
	"github.com/hail2skins/armory/internal/middleware"
	"github.com/newrelic/go-agent/v3/integrations/nrgin"
)

// securityHeaders applies security headers to all responses
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Content Security Policy
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' https://unpkg.com; style-src 'self' 'unsafe-inline' https://unpkg.com; img-src 'self' data:;")

		// X-Content-Type-Options
		c.Header("X-Content-Type-Options", "nosniff")

		// X-Frame-Options
		c.Header("X-Frame-Options", "DENY")

		// X-XSS-Protection
		c.Header("X-XSS-Protection", "1; mode=block")

		// Referrer-Policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions-Policy
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=(), interest-cohort=()")

		c.Next()
	}
}

// getCorsOrigins returns the list of allowed origins for CORS
func getCorsOrigins() []string {
	corsOrigins := os.Getenv("CORS_ORIGINS")
	if corsOrigins != "" {
		return strings.Split(corsOrigins, ",")
	}

	// Default local development origins
	return []string{"http://localhost:5173", "http://localhost:3000", "http://localhost:8080"}
}

// RegisterMiddleware sets up all middleware for the application
func (s *Server) RegisterMiddleware(r *gin.Engine, authController *controller.AuthController) {
	// Set up error handling middleware
	middleware.SetupErrorHandling(r)

	// Add New Relic middleware first to ensure transaction is available
	if s.newRelicApp != nil {
		r.Use(nrgin.Middleware(s.newRelicApp))
	}

	// Add logging middleware to set up transaction-aware logger
	r.Use(func(c *gin.Context) {
		// Get a transaction-aware logger
		txnLogger := logger.GetContextLogger(c)
		// Store it in the context
		c.Set("logger", txnLogger)
		c.Next()
	})

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

	// Set up sessions middleware with a cookie store
	store := cookie.NewStore([]byte(os.Getenv("SESSION_SECRET")))
	if os.Getenv("SESSION_SECRET") == "" {
		log.Println("Warning: Using default session secret. This is not secure for production.")
		store = cookie.NewStore([]byte("armory-session-secret"))
	}
	r.Use(sessions.Sessions("armory-session", store))

	// Set up flash message middleware - moved before rate limiting
	r.Use(FlashMiddleware())

	// Set up CSRF protection middleware with exclusions for webhooks
	r.Use(func(c *gin.Context) {
		// Exclude webhook endpoints from CSRF protection
		if strings.HasPrefix(c.Request.URL.Path, "/webhook") {
			c.Next()
			return
		}

		// Apply CSRF middleware to all other routes
		middleware.CSRFMiddleware()(c)
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

	// Initialize Casbin with the database adapter (stored in the server instance for admin routes to use)
	casbinAuth, err = middleware.SetupCasbinWithDB(s.db.GetDB())
	if err != nil {
		// Log the error but continue (admin routes will check for nil)
		logger.Warn("Casbin setup with database failed, RBAC will be disabled", map[string]interface{}{
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
