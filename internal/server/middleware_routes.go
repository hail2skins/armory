package server

import (
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/logger"
	"github.com/hail2skins/armory/internal/middleware"
)

// RegisterMiddleware sets up all middleware for the application
func (s *Server) RegisterMiddleware(r *gin.Engine, authController *controller.AuthController) {
	// Set up error handling middleware
	middleware.SetupErrorHandling(r)

	// Set up flash message middleware - moved before rate limiting
	r.Use(func(c *gin.Context) {
		// Set up a function to set flash messages
		c.Set("setFlash", func(message string) {
			// Set the flash message in a cookie
			c.SetCookie("flash", message, 3600, "/", "", false, false)
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
		AllowOrigins:     []string{"http://localhost:5173"}, // Add your frontend URL
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

	// Set up auth data middleware
	r.Use(func(c *gin.Context) {
		// Make casbinAuth available in the context
		c.Set("casbinAuth", casbinAuth)

		// Get the current user's authentication status and email
		userInfo, authenticated := authController.GetCurrentUser(c)

		// Create AuthData with authentication status and email
		authData := data.NewAuthData()
		authData.Authenticated = authenticated

		// Set email if authenticated
		if authenticated {
			authData.Email = userInfo.GetUserName()

			// Get user roles from Casbin if available
			if s.casbinAuth != nil {
				roles := s.casbinAuth.GetUserRoles(userInfo.GetUserName())
				authData = authData.WithRoles(roles)
			}
		}

		// Add authData to context
		c.Set("authData", authData)

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

	// Default admin user if not specified in environment
	return []string{"support@hamcois.com"}
}
