package server

import (
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
)

// RegisterErrorRoutes sets up routes for custom error pages
func (s *Server) RegisterErrorRoutes(r *gin.Engine) {
	// Create the error controller
	errorController := controller.NewErrorController()

	// Configure HTML Renderer for templ templates
	r.HTMLRender = errorController.CreateTemplRenderer()

	// Add a separate route group for error pages
	errGroup := r.Group("/error")
	{
		// Route for 404 errors
		errGroup.GET("/404", func(c *gin.Context) {
			errorController.RenderNotFound(c, "The page you're looking for doesn't exist.")
		})

		// Route for generic errors
		errGroup.GET("/500", func(c *gin.Context) {
			errorController.RenderInternalServerError(c, "An internal server error occurred", "")
		})

		// Route for forbidden errors
		errGroup.GET("/403", func(c *gin.Context) {
			errorController.RenderForbidden(c, "You don't have permission to access this resource.")
		})

		// Route for unauthorized errors
		errGroup.GET("/401", func(c *gin.Context) {
			errorController.RenderUnauthorized(c, "Authentication is required to access this resource.")
		})
	}

	// Add a catch-all route that will redirect to the 404 page
	r.NoRoute(func(c *gin.Context) {
		c.Redirect(302, "/error/404")
	})
}
