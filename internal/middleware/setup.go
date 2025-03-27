package middleware

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/logger"
	"gorm.io/gorm"
)

// SetupErrorHandling configures all error handling middleware for a Gin router
func SetupErrorHandling(router *gin.Engine) {
	// Set up the custom error handlers with templates
	SetupErrorHandlers(router)

	// Set up error handling middleware
	router.Use(ErrorHandler())

	// Set up error metrics middleware
	router.Use(ErrorMetricsMiddleware())
}

// NewRateLimiterMiddleware creates and returns a new rate limiter instance
func NewRateLimiterMiddleware() *RateLimiter {
	return NewRateLimiter()
}

// SetupRateLimiting configures rate limiting for critical endpoints
func SetupRateLimiting(router *gin.Engine) {
	rateLimiter := NewRateLimiter()

	// Create a single middleware for all routes
	router.Use(func(c *gin.Context) {
		// Get the actual request path
		path := c.Request.URL.Path

		// Apply the appropriate rate limiter based on the path
		switch {
		case path == "/login":
			// Login rate limit (5 requests per minute)
			rateLimiter.RateLimit(5, time.Minute)(c)
		case path == "/register":
			// Register rate limit (5 requests per minute)
			rateLimiter.RateLimit(5, time.Minute)(c)
		case path == "/reset-password" || path == "/reset-password/new":
			// Password reset rate limit (3 requests per hour)
			rateLimiter.RateLimit(3, time.Hour)(c)
		case path == "/webhook":
			// Webhook rate limit (10 requests per minute)
			rateLimiter.RateLimit(10, time.Minute)(c)
		default:
			// No rate limiting for other routes
			c.Next()
		}
	})
}

// SetupCasbin initializes Casbin RBAC middleware
func SetupCasbin() (*CasbinAuth, error) {
	// Get config directory
	configDir := filepath.Join("configs", "casbin")

	// Ensure the directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create casbin config directory: %w", err)
	}

	modelPath := filepath.Join(configDir, "rbac_model.conf")
	policyPath := filepath.Join(configDir, "rbac_policy.csv")

	// Create default RBAC model file if it doesn't exist
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		defaultModel := `[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && (r.obj == p.obj || p.obj == "*") && (r.act == p.act || p.act == "*") || g(r.sub, "admin")`

		if err := os.WriteFile(modelPath, []byte(defaultModel), 0644); err != nil {
			return nil, fmt.Errorf("failed to create default RBAC model file: %w", err)
		}
	}

	// Create empty policy file if it doesn't exist
	if _, err := os.Stat(policyPath); os.IsNotExist(err) {
		defaultPolicy := `p, admin, *, *
p, editor, manufacturers, read
p, editor, manufacturers, write
p, editor, manufacturers, update
p, editor, calibers, read
p, editor, calibers, write
p, editor, calibers, update
p, editor, weapon_types, read
p, editor, weapon_types, write
p, editor, weapon_types, update
p, viewer, manufacturers, read
p, viewer, calibers, read
p, viewer, weapon_types, read`

		if err := os.WriteFile(policyPath, []byte(defaultPolicy), 0644); err != nil {
			return nil, fmt.Errorf("failed to create default RBAC policy file: %w", err)
		}
	}

	// Initialize Casbin with models and policies
	casbinAuth, err := NewCasbinAuth(modelPath, policyPath)
	if err != nil {
		logger.Error("Failed to initialize Casbin", err, nil)
		return nil, err
	}

	return casbinAuth, nil
}

// SetupCasbinWithDB initializes Casbin RBAC middleware with database support
func SetupCasbinWithDB(db *gorm.DB) (*CasbinAuth, error) {
	// Get config directory
	configDir := filepath.Join("configs", "casbin")

	// Ensure the directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create casbin config directory: %w", err)
	}

	modelPath := filepath.Join(configDir, "rbac_model.conf")

	// Create default RBAC model file if it doesn't exist
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		defaultModel := `[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && (r.obj == p.obj || p.obj == "*") && (r.act == p.act || p.act == "*") || g(r.sub, "admin")`

		if err := os.WriteFile(modelPath, []byte(defaultModel), 0644); err != nil {
			return nil, fmt.Errorf("failed to create default RBAC model file: %w", err)
		}
	}

	// Initialize Casbin with model from file and policies from database
	casbinAuth, err := NewCasbinAuthWithDB(modelPath, db)
	if err != nil {
		logger.Error("Failed to initialize Casbin with database", err, nil)
		return nil, err
	}

	return casbinAuth, nil
}

// SetupAllMiddleware configures all middleware for a Gin router
func SetupAllMiddleware(router *gin.Engine) (*CasbinAuth, error) {
	// Set up error handling
	SetupErrorHandling(router)

	// Set up rate limiting
	SetupRateLimiting(router)

	// Set up webhook monitoring for /webhook endpoints
	router.Use(func(c *gin.Context) {
		if c.Request.URL.Path == "/webhook" {
			WebhookMonitor()(c)
		} else {
			c.Next()
		}
	})

	// Initialize Casbin (but don't apply it globally - it's applied per route)
	casbinAuth, err := SetupCasbin()
	if err != nil {
		logger.Warn("Casbin setup failed, RBAC will be disabled", map[string]interface{}{
			"error": err.Error(),
		})
	}

	return casbinAuth, err
}

// SetupAllMiddlewareWithDB configures all middleware for a Gin router with database support for Casbin
func SetupAllMiddlewareWithDB(router *gin.Engine, db *gorm.DB) (*CasbinAuth, error) {
	// Set up error handling
	SetupErrorHandling(router)

	// Set up rate limiting
	SetupRateLimiting(router)

	// Set up webhook monitoring for /webhook endpoints
	router.Use(func(c *gin.Context) {
		if c.Request.URL.Path == "/webhook" {
			WebhookMonitor()(c)
		} else {
			c.Next()
		}
	})

	// Initialize Casbin with database (but don't apply it globally - it's applied per route)
	casbinAuth, err := SetupCasbinWithDB(db)
	if err != nil {
		logger.Warn("Casbin setup with database failed, RBAC will be disabled", map[string]interface{}{
			"error": err.Error(),
		})
	}

	return casbinAuth, err
}

// ConfigureInitialRoles sets up initial roles for admin users when the app starts
func ConfigureInitialRoles(casbinAuth *CasbinAuth, adminEmails []string) error {
	if casbinAuth == nil {
		return fmt.Errorf("casbin middleware not initialized")
	}

	// Add each admin user from the configuration
	for _, email := range adminEmails {
		// Check if the user already has the admin role to avoid duplicates
		if !casbinAuth.HasRole(email, "admin") {
			logger.Info("Adding admin role to user", map[string]interface{}{
				"email": email,
			})

			// Add the admin role
			casbinAuth.AddUserRole(email, "admin")
		}
	}

	// Save the policy to persist changes
	if err := casbinAuth.SavePolicy(); err != nil {
		return fmt.Errorf("failed to save policy: %w", err)
	}

	return nil
}
