package server

import (
	"fmt"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/logger"
	"github.com/hail2skins/armory/internal/middleware"
	"github.com/hail2skins/armory/internal/services/email"
)

// RegisterOwnerRoutes registers all owner-related routes
func RegisterOwnerRoutes(router *gin.Engine, db database.Service, authController *controller.AuthController) {
	// Create the owner controller
	ownerController := controller.NewOwnerController(db)

	// Initialize email service
	emailService, _ := email.NewMailjetService()

	// API routes
	apiGroup := router.Group("/api")
	{
		// Caliber search API
		apiGroup.GET("/calibers/search", ownerController.SearchCalibers)
	}

	// Owner routes (require authentication)
	ownerGroup := router.Group("/owner")
	// Set email service in the context for all owner routes
	ownerGroup.Use(func(c *gin.Context) {
		// Add email service to context
		c.Set("emailService", emailService)
		// Add auth controller to context
		c.Set("authController", authController)
		c.Next()
	})
	// Check if user is authenticated
	ownerGroup.Use(func(c *gin.Context) {
		// Check if user is authenticated
		_, authenticated := authController.GetCurrentUser(c)
		if !authenticated {
			// Set flash message
			if setFlash, exists := c.Get("setFlash"); exists {
				setFlash.(func(string))("You must be logged in to access this page")
			}
			c.Redirect(302, "/login")
			c.Abort()
			return
		}
		c.Next()
	})
	{
		// Owner landing page
		ownerGroup.GET("", ownerController.LandingPage)

		// Owner profile page
		ownerGroup.GET("/profile", ownerController.Profile)

		// Owner profile edit and update
		ownerGroup.GET("/profile/edit", ownerController.EditProfile)
		ownerGroup.POST("/profile/update", ownerController.UpdateProfile)

		// Owner account deletion
		ownerGroup.GET("/profile/delete", ownerController.DeleteAccountConfirm)
		ownerGroup.POST("/profile/delete", ownerController.DeleteAccountHandler)

		// Owner subscription management
		ownerGroup.GET("/profile/subscription", ownerController.Subscription)

		// Gun routes nested under owner
		gunGroup := ownerGroup.Group("/guns")
		{
			// List all guns for the owner
			gunGroup.GET("", ownerController.Index)

			// Arsenal view - shows all guns with sorting and searching
			gunGroup.GET("/arsenal", ownerController.Arsenal)

			// Create a new gun
			gunGroup.GET("/new", ownerController.New)
			gunGroup.POST("", ownerController.Create)

			// Show a specific gun
			gunGroup.GET("/:id", ownerController.Show)

			// Edit a gun
			gunGroup.GET("/:id/edit", ownerController.Edit)
			gunGroup.POST("/:id", ownerController.Update)

			// Delete a gun
			gunGroup.POST("/:id/delete", ownerController.Delete)
		}

		// Ammunition routes - protected by permission middleware
		// This will ensure only users with appropriate permissions can access the ammo features
		ammoGroup := ownerGroup.Group("/munitions")
		// Add middleware to check for permissions
		ammoGroup.Use(func(c *gin.Context) {
			// Get the auth controller from the context
			authControllerInterface, exists := c.Get("authController")
			if !exists {
				logger.Error("Auth controller not in context", nil, map[string]interface{}{
					"path": c.Request.URL.Path,
				})
				c.Redirect(http.StatusSeeOther, "/")
				c.Abort()
				return
			}

			authController, ok := authControllerInterface.(*controller.AuthController)
			if !ok {
				logger.Error("Auth controller not correct type", nil, map[string]interface{}{
					"path": c.Request.URL.Path,
					"type": fmt.Sprintf("%T", authControllerInterface),
				})
				c.Redirect(http.StatusSeeOther, "/")
				c.Abort()
				return
			}

			// Get current user
			userInfo, authenticated := authController.GetCurrentUser(c)
			if !authenticated {
				logger.Error("User not authenticated", nil, map[string]interface{}{
					"path": c.Request.URL.Path,
				})
				c.Redirect(http.StatusSeeOther, "/login")
				c.Abort()
				return
			}

			// Get the casbin auth from the context (set up in server.go)
			casbinAuth, exists := c.Get("casbinAuth")
			if !exists {
				logger.Error("CasbinAuth not in context", nil, map[string]interface{}{
					"path": c.Request.URL.Path,
				})
				c.Redirect(http.StatusSeeOther, "/")
				c.Abort()
				return
			}

			// Check permission using casbin
			if casbinAuthInstance, ok := casbinAuth.(*middleware.CasbinAuth); ok {
				// Reload policy from database to ensure we have the latest permissions
				if err := casbinAuthInstance.ReloadPolicy(); err != nil {
					logger.Error("Failed to reload policy", err, map[string]interface{}{
						"path": c.Request.URL.Path,
					})
					c.Redirect(http.StatusSeeOther, "/")
					c.Abort()
					return
				}

				// Get username to check permissions
				username := userInfo.GetUserName()
				logger.Info("Checking ammunition permissions", map[string]interface{}{
					"username": username,
					"path":     c.Request.URL.Path,
				})

				// Check user roles
				roles := casbinAuthInstance.GetUserRoles(username)
				logger.Info("User roles for ammunition access", map[string]interface{}{
					"username": username,
					"roles":    roles,
				})

				// Check for admin role first
				isAdmin := false
				for _, role := range roles {
					if role == "admin" {
						isAdmin = true
						break
					}
				}

				if isAdmin {
					logger.Info("Access granted (admin)", map[string]interface{}{
						"username": username,
					})
					c.Next()
					return
				}

				// Check for any role - this is what's actually working in production
				hasAnyRole := len(roles) > 0

				logger.Info("Role check results", map[string]interface{}{
					"username":   username,
					"hasAnyRole": hasAnyRole,
					"roles":      roles,
				})

				if !hasAnyRole {
					logger.Info("Access denied (no roles)", map[string]interface{}{
						"username": username,
					})
					session := sessions.Default(c)
					session.AddFlash("You don't have permission to access ammunition management")
					session.Save()

					c.Redirect(http.StatusSeeOther, "/")
					c.Abort()
					return
				}

				// User has a role, continue to next middleware
				logger.Info("Access granted (role found)", map[string]interface{}{
					"username": username,
					"roles":    roles,
				})
				c.Next()
			} else {
				logger.Error("CasbinAuth not correct type", nil, map[string]interface{}{
					"path": c.Request.URL.Path,
					"type": fmt.Sprintf("%T", casbinAuth),
				})
				c.Redirect(http.StatusSeeOther, "/")
				c.Abort()
				return
			}
		})
		{
			// Index and Create ammunition
			ammoGroup.GET("", ownerController.AmmoIndex)
			ammoGroup.GET("/new", ownerController.AmmoNew)
			ammoGroup.POST("", ownerController.AmmoCreate)

			// Show ammunition details
			ammoGroup.GET("/:id", ownerController.AmmoShow)

			// Edit and Update ammunition
			ammoGroup.GET("/:id/edit", ownerController.AmmoEdit)
			ammoGroup.POST("/:id", ownerController.AmmoUpdate)

			// Delete will be implemented later
			// ammoGroup.POST("/:id/delete", ownerController.AmmoDelete)

			// Search routes for HTMX dropdown filters - Removed in favor of client-side filtering with Choices.js
			// ammoGroup.GET("/search/brands", ownerController.SearchBrands)
			// ammoGroup.GET("/search/calibers", ownerController.SearchCalibers)
			// ammoGroup.GET("/search/bullet-styles", ownerController.SearchBulletStyles)
			// ammoGroup.GET("/search/grains", ownerController.SearchGrains)
			// ammoGroup.GET("/search/casings", ownerController.SearchCasings)
		}
	}
}
