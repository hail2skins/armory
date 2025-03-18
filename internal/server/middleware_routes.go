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

	// Set up flash message middleware
	r.Use(func(c *gin.Context) {
		// Set up a function to set flash messages
		c.Set("setFlash", func(message string) {
			// Set the flash message in a cookie
			c.SetCookie("flash", message, 3600, "/", "", false, false)
		})
		c.Next()
	})

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

		// Set both auth keys for compatibility - the new pattern uses "auth"
		// while some existing code might still use "authController"
		c.Set("auth", authController)
		c.Set("authController", authController)

		c.Next()
	})
}
