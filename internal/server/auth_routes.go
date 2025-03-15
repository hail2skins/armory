package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/auth"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/controller"
)

// RegisterAuthRoutes registers all authentication related routes
func (s *Server) RegisterAuthRoutes(r *gin.Engine, authController *controller.AuthController) {
	// Set up render functions for auth views
	setupAuthRenderFunctions(authController)

	// Auth routes
	r.GET("/login", authController.LoginHandler)
	r.POST("/login", authController.LoginHandler)
	r.GET("/register", authController.RegisterHandler)
	r.POST("/register", authController.RegisterHandler)
	r.GET("/logout", authController.LogoutHandler)
	r.GET("/verification-sent", func(c *gin.Context) {
		authData := data.NewAuthData().WithTitle("Verification Email Sent")
		// Set authentication state
		_, authenticated := authController.GetCurrentUser(c)
		authData.Authenticated = authenticated
		// Render the verification sent page
		if authController.RenderVerificationSent != nil {
			authController.RenderVerificationSent(c, authData)
		} else {
			// Fallback if the render function is not set
			c.String(http.StatusOK, "Verification email sent. Please check your inbox to verify your email address.")
		}
	})
	r.POST("/resend-verification", authController.ResendVerificationHandler)
	r.GET("/verify-email", authController.VerifyEmailHandler)
	r.GET("/forgot-password", authController.ForgotPasswordHandler)
	r.POST("/forgot-password", authController.ForgotPasswordHandler)
	r.GET("/reset-password", authController.ResetPasswordHandler)
	r.POST("/reset-password", authController.ResetPasswordHandler)
}

// setupAuthRenderFunctions configures the render functions for auth views
func setupAuthRenderFunctions(authController *controller.AuthController) {
	// Login render function
	authController.RenderLogin = func(c *gin.Context, d interface{}) {
		authData := d.(data.AuthData)
		// Set authentication state
		_, authenticated := authController.GetCurrentUser(c)
		authData.Authenticated = authenticated
		// Set default title if not set
		if authData.Title == "" {
			authData.Title = "Login"
		}
		auth.Login(authData).Render(c.Request.Context(), c.Writer)
	}

	// Register render function
	authController.RenderRegister = func(c *gin.Context, d interface{}) {
		authData := d.(data.AuthData)
		// Set authentication state
		_, authenticated := authController.GetCurrentUser(c)
		authData.Authenticated = authenticated
		// Set default title if not set
		if authData.Title == "" {
			authData.Title = "Register"
		}
		auth.Register(authData).Render(c.Request.Context(), c.Writer)
	}

	// Logout render function
	authController.RenderLogout = func(c *gin.Context, d interface{}) {
		authData := d.(data.AuthData)
		// Set authentication state - should be false after logout
		authData.Authenticated = false
		// Set default title if not set
		if authData.Title == "" {
			authData.Title = "Logout"
		}
		auth.Logout(authData).Render(c.Request.Context(), c.Writer)
	}

	// Verification sent render function
	authController.RenderVerificationSent = func(c *gin.Context, d interface{}) {
		authData := d.(data.AuthData)
		// Set authentication state
		_, authenticated := authController.GetCurrentUser(c)
		authData.Authenticated = authenticated
		// Set default title if not set
		if authData.Title == "" {
			authData.Title = "Verification Email Sent"
		}
		auth.VerificationSent(authData).Render(c.Request.Context(), c.Writer)
	}
}
