package server

import (
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/database"
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

		// Ammunition routes - no longer protected by permission middleware
		// Making ammunition features available to all authenticated users
		ammoGroup := ownerGroup.Group("/munitions")
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

			// Delete ammunition
			ammoGroup.POST("/:id/delete", ownerController.AmmoDelete)

			// Search routes for HTMX dropdown filters - Removed in favor of client-side filtering with Choices.js
			// ammoGroup.GET("/search/brands", ownerController.SearchBrands)
			// ammoGroup.GET("/search/calibers", ownerController.SearchCalibers)
			// ammoGroup.GET("/search/bullet-styles", ownerController.SearchBulletStyles)
			// ammoGroup.GET("/search/grains", ownerController.SearchGrains)
			// ammoGroup.GET("/search/casings", ownerController.SearchCasings)
		}
	}
}
