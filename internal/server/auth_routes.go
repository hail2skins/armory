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

		// Handle flash messages
		authData = handleAuthFlashMessage(c, authData)

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

// handleAuthFlashMessage checks for a flash message cookie and adds it to the AuthData
func handleAuthFlashMessage(c *gin.Context, authData data.AuthData) data.AuthData {
	// Check for flash message
	if flashCookie, err := c.Cookie("flash"); err == nil && flashCookie != "" {
		// Add flash message to success messages
		authData.Success = flashCookie
		// Clear the flash cookie
		c.SetCookie("flash", "", -1, "/", "", false, false)
	}
	return authData
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

		// Handle flash messages
		authData = handleAuthFlashMessage(c, authData)

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

		// Handle flash messages
		authData = handleAuthFlashMessage(c, authData)

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

		// Handle flash messages
		authData = handleAuthFlashMessage(c, authData)

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

		// Handle flash messages
		authData = handleAuthFlashMessage(c, authData)

		auth.VerificationSent(authData).Render(c.Request.Context(), c.Writer)
	}

	// ForgotPassword render function
	authController.RenderForgotPassword = func(c *gin.Context, d interface{}) {
		authData := d.(data.AuthData)
		// Set authentication state
		_, authenticated := authController.GetCurrentUser(c)
		authData.Authenticated = authenticated
		// Set default title if not set
		if authData.Title == "" {
			authData.Title = "Forgot Password"
		}

		// Handle flash messages
		authData = handleAuthFlashMessage(c, authData)

		auth.ForgotPassword(authData).Render(c.Request.Context(), c.Writer)
	}

	// ResetPassword render function
	authController.RenderResetPassword = func(c *gin.Context, d interface{}) {
		authData := d.(data.AuthData)
		// Set authentication state
		_, authenticated := authController.GetCurrentUser(c)
		authData.Authenticated = authenticated
		// Set default title if not set
		if authData.Title == "" {
			authData.Title = "Reset Password"
		}

		// Handle flash messages
		authData = handleAuthFlashMessage(c, authData)

		auth.ResetPassword(authData).Render(c.Request.Context(), c.Writer)
	}
}
