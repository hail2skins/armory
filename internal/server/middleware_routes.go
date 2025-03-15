package server

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/middleware"
)

// RegisterMiddleware sets up all middleware for the application
func (s *Server) RegisterMiddleware(r *gin.Engine, authController *controller.AuthController) {
	// Set up error handling middleware
	middleware.SetupErrorHandling(r)

	// Set up CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"}, // Add your frontend URL
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true, // Enable cookies/auth
	}))

	// Set up auth data middleware
	r.Use(func(c *gin.Context) {
		// Get the current user's authentication status and email
		userInfo, authenticated := authController.GetCurrentUser(c)

		// Create AuthData with authentication status and email
		authData := data.NewAuthData()
		authData.Authenticated = authenticated

		// Set email if authenticated
		if authenticated {
			authData.Email = userInfo.GetUserName()
		}

		// Add authData to context
		c.Set("authData", authData)
		c.Set("authController", authController)

		c.Next()
	})
}
