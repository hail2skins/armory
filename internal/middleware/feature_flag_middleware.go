package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/database"
)

// Allow mocking the database.New function for testing
var databaseNewFunc = database.New

// RequireFeature creates a middleware that checks if a user can access a specific feature
// based on feature flags and associated roles
func RequireFeature(featureName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get auth controller from context
		authControllerInterface, exists := c.Get("authController")
		if !exists {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// Check if the auth controller implements IsAuthenticated
		// This is a type assertion check to make sure we have the right interface
		auth, ok := authControllerInterface.(interface {
			IsAuthenticated(c *gin.Context) bool
			IsAdmin(c *gin.Context) bool
		})
		if !ok {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		// If not authenticated, redirect to login
		if !auth.IsAuthenticated(c) {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// If admin, allow access automatically
		if auth.IsAdmin(c) {
			c.Next()
			return
		}

		// Get user info for checking roles
		userInterface, exists := c.Get("auth")
		if !exists {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// Try to get user methods
		user, ok := userInterface.(interface {
			GetUserName() string
		})
		if !ok {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		// Get username
		username := user.GetUserName()

		// Check if user can access feature based on roles
		db := databaseNewFunc()
		canAccess, err := db.CanUserAccessFeature(username, featureName)

		if err != nil || !canAccess {
			// Set flash message if session exists
			setFlashIfAvailable(c, "You don't have access to this feature")

			// Redirect to dashboard or suitable page
			c.Redirect(http.StatusFound, "/dashboard")
			c.Abort()
			return
		}

		// Store feature flag access in context for templates
		c.Set("has_"+featureName+"_access", true)

		c.Next()
	}
}

// Helper to set flash message if session is available
func setFlashIfAvailable(c *gin.Context, message string) {
	flashSetter, exists := c.Get("setFlash")
	if exists {
		if setter, ok := flashSetter.(func(string)); ok {
			setter(message)
		}
	}
}

// CheckFeatureAccessForTemplates middleware adds feature access flags to the context
// for all registered feature flags, to be used in templates
func CheckFeatureAccessForTemplates() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get auth controller from context
		authControllerInterface, exists := c.Get("authController")
		if !exists {
			c.Next()
			return
		}

		// Check if the auth controller implements IsAuthenticated and IsAdmin
		auth, ok := authControllerInterface.(interface {
			IsAuthenticated(c *gin.Context) bool
			IsAdmin(c *gin.Context) bool
		})
		if !ok {
			c.Next()
			return
		}

		// If not authenticated, no need to check features
		if !auth.IsAuthenticated(c) {
			c.Next()
			return
		}

		// Get user info for checking roles
		userInterface, exists := c.Get("auth")
		if !exists {
			c.Next()
			return
		}

		// Try to get user methods
		user, ok := userInterface.(interface {
			GetUserName() string
		})
		if !ok {
			c.Next()
			return
		}

		// Get username
		username := user.GetUserName()

		// If admin, set access to all features
		if auth.IsAdmin(c) {
			// Create a map of feature accesses for templates
			featureAccess := make(map[string]bool)

			// Get all feature flags
			db := databaseNewFunc()
			flags, err := db.FindAllFeatureFlags()
			if err == nil {
				for _, flag := range flags {
					featureAccess[flag.Name] = true
				}
			}

			c.Set("feature_access", featureAccess)
			c.Next()
			return
		}

		// For regular users, check each feature
		db := databaseNewFunc()
		flags, err := db.FindAllFeatureFlags()
		if err != nil {
			c.Next()
			return
		}

		// Create a map of feature accesses for templates
		featureAccess := make(map[string]bool)

		for _, flag := range flags {
			canAccess, err := db.CanUserAccessFeature(username, flag.Name)
			if err == nil && canAccess {
				featureAccess[flag.Name] = true
			} else {
				featureAccess[flag.Name] = false
			}
		}

		c.Set("feature_access", featureAccess)
		c.Next()
	}
}

// Helper function to check feature access in templates
func HasFeatureAccess(c *gin.Context, featureName string) bool {
	// Check if feature_access map exists
	featureAccessInterface, exists := c.Get("feature_access")
	if !exists {
		return false
	}

	// Try to cast to map
	featureAccess, ok := featureAccessInterface.(map[string]bool)
	if !ok {
		return false
	}

	// Check if user has access to the feature
	access, exists := featureAccess[featureName]
	return exists && access
}
