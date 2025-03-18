package web

import (
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
)

// setupRoutes configures all application routes
func setupRoutes(router *gin.Engine, authController *controller.AuthController) {
	// Static files
	router.Static("/static", "./static")

	// Authentication routes
	router.GET("/login", authController.LoginHandler)
	router.POST("/login", authController.LoginHandler)
	router.GET("/register", authController.RegisterHandler)
	router.POST("/register", authController.RegisterHandler)
	router.GET("/logout", authController.LogoutHandler)

	// Email verification routes
	router.GET("/verify", authController.VerifyEmailHandler)
	router.GET("/verification-sent", func(c *gin.Context) {
		authController.RenderVerificationSent(c, nil)
	})

	// Password reset routes
	router.GET("/reset-password/new", authController.ForgotPasswordHandler)
	router.POST("/reset-password/new", authController.ForgotPasswordHandler)
	router.GET("/reset-password", authController.ResetPasswordHandler)
	router.POST("/reset-password", authController.ResetPasswordHandler)

	// Protected routes
	protected := router.Group("/")
	protected.Use(authController.AuthMiddleware())
	{
		// Add protected routes here
		router.GET("/", func(c *gin.Context) {
			c.String(200, "Welcome to the Virtual Armory!")
		})
	}
}
