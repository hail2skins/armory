package server

import (
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
)

// RegisterErrorRoutes sets up routes for error handling
func (s *Server) RegisterErrorRoutes(r *gin.Engine) {
	// Create error controller for handling error templates
	errorController := controller.NewErrorController()

	// Register templ component renderer for error templates
	r.HTMLRender = errorController.CreateTemplRenderer()

	// Set up NoRoute handler for 404 errors
	r.NoRoute(errorController.NoRouteHandler())

	// Set up NoMethod handler for 405 errors
	r.NoMethod(errorController.NoMethodHandler())

	// Set up recovery handler for panic recovery
	r.Use(errorController.RecoveryHandler())

	// Make error controller available in the context for other controllers to use
	r.Use(func(c *gin.Context) {
		c.Set("errorController", errorController)
		c.Next()
	})
}
