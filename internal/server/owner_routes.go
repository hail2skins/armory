package server

import (
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/database"
)

// RegisterOwnerRoutes registers all owner-related routes
func RegisterOwnerRoutes(router *gin.Engine, db database.Service, authController *controller.AuthController) {
	// Create the owner controller
	ownerController := controller.NewOwnerController(db)

	// API routes
	apiGroup := router.Group("/api")
	{
		// Caliber search API
		apiGroup.GET("/calibers/search", ownerController.SearchCalibers)
	}

	// Owner routes (require authentication)
	ownerGroup := router.Group("/owner")
	// Set the auth controller in the context for all owner routes
	ownerGroup.Use(func(c *gin.Context) {
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
	}
}
