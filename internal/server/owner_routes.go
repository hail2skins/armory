package server

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/middleware"
	"github.com/hail2skins/armory/internal/services/email"
	"github.com/shaj13/go-guardian/v2/auth"
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
			// Get the casbin auth from the context (set up in server.go)
			casbinAuth, exists := c.Get("casbinAuth")
			if !exists {
				c.Redirect(http.StatusSeeOther, "/")
				c.Abort()
				return
			}

			// Check permission using casbin
			if casbinAuthInstance, ok := casbinAuth.(*middleware.CasbinAuth); ok {
				// Get auth info
				authInfo, authExists := c.Get("auth_info")
				if !authExists {
					c.Redirect(http.StatusSeeOther, "/")
					c.Abort()
					return
				}

				// Get username to check permissions
				userInfo, ok := authInfo.(auth.Info)
				if !ok {
					c.Redirect(http.StatusSeeOther, "/")
					c.Abort()
					return
				}

				username := userInfo.GetUserName()

				// Check user roles
				roles := casbinAuthInstance.GetUserRoles(username)

				// First check if user has admin role (admins can access everything)
				isAdmin := false
				for _, role := range roles {
					if role == "admin" {
						isAdmin = true
						break
					}
				}

				if isAdmin {
					c.Next()
					return
				}

				// Check for ammo_feature_testers role
				hasAmmoRole := false
				for _, role := range roles {
					if role == "ammo_feature_testers" {
						hasAmmoRole = true
						break
					}
				}

				if !hasAmmoRole {
					session := sessions.Default(c)
					session.AddFlash("You don't have permission to access ammunition management")
					session.Save()

					c.Redirect(http.StatusSeeOther, "/")
					c.Abort()
					return
				}

				// User has the role, continue to next middleware
				c.Next()
			} else {
				c.Redirect(http.StatusSeeOther, "/")
				c.Abort()
				return
			}
		})
		{
			// These are placeholders - create the controller methods as needed
			// ammoGroup.GET("", ownerController.AmmoIndex)
			ammoGroup.GET("/new", ownerController.AmmoNew)
			// ammoGroup.POST("", ownerController.AmmoCreate)
			// ammoGroup.GET("/:id", ownerController.AmmoShow)
			// ammoGroup.GET("/:id/edit", ownerController.AmmoEdit)
			// ammoGroup.POST("/:id", ownerController.AmmoUpdate)
			// ammoGroup.POST("/:id/delete", ownerController.AmmoDelete)
		}
	}
}
