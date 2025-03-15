package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterAPIRoutes registers all API related routes
func (s *Server) RegisterAPIRoutes(r *gin.Engine) {
	// Create an API group
	api := r.Group("/api")

	// Health check endpoint
	api.GET("/health", s.healthHandler)

	// Future API endpoints can be added here
	// api.GET("/users", userController.ListUsersHandler)
	// api.GET("/users/:id", userController.GetUserHandler)
	// etc.
}

// healthHandler returns the health status of the application
func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, s.db.Health())
}
