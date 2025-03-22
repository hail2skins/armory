package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/logger"
)

// SetupErrorHandlers configures the router with error handlers
func SetupErrorHandlers(router *gin.Engine) {
	// Create error controller for direct template rendering
	errorController := controller.NewErrorController()

	// Custom 404 handler that renders the error page directly
	router.NoRoute(func(c *gin.Context) {
		logger.Debug("404 Not Found - Rendering error page", map[string]interface{}{
			"path":   c.Request.URL.Path,
			"method": c.Request.Method,
		})

		// Check if this is an API or JSON request
		if c.GetHeader("Accept") == "application/json" {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    http.StatusNotFound,
				"message": "Page not found",
			})
			return
		}

		// Render the error page directly
		errorController.RenderNotFound(c, "The page you're looking for doesn't exist.")
	})

	// Custom 405 handler that renders the error page directly
	router.NoMethod(func(c *gin.Context) {
		logger.Debug("405 Method Not Allowed - Rendering error page", map[string]interface{}{
			"path":   c.Request.URL.Path,
			"method": c.Request.Method,
		})

		// Check if this is an API or JSON request
		if c.GetHeader("Accept") == "application/json" {
			c.JSON(http.StatusMethodNotAllowed, gin.H{
				"code":    http.StatusMethodNotAllowed,
				"message": "Method not allowed",
			})
			return
		}

		// Render the error page directly
		errorController.RenderError(c, "This method is not allowed for this resource.", http.StatusMethodNotAllowed)
	})

	logger.Info("Error handlers initialized", nil)
}
