package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/errors"
)

// SetupErrorHandling configures all error handling middleware for a Gin router
func SetupErrorHandling(router *gin.Engine) {
	// Set up 404 handler for undefined routes
	router.NoRoute(errors.NoRouteHandler())

	// Set up 405 handler for method not allowed
	router.NoMethod(errors.NoMethodHandler())

	// Set up panic recovery middleware
	router.Use(errors.RecoveryHandler())

	// Set up error handling middleware
	router.Use(ErrorHandler())
}
