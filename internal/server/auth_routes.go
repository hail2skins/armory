package server

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/auth"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/middleware"
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
	r.POST("/logout", authController.LogoutHandler)
	r.GET("/verification-sent", func(c *gin.Context) {
		authData := data.NewAuthData().WithTitle("Verification Email Sent")
		// Set authentication state
		_, authenticated := authController.GetCurrentUser(c)
		authData.Authenticated = authenticated

		// Add CSRF token for form protection
		csrfToken := middleware.GetCSRFToken(c)
		authData = authData.WithCSRFToken(csrfToken)

		// Get email from cookie if it exists
		if cookie, err := c.Cookie("verification_email"); err == nil && cookie != "" {
			// URL decode the cookie value
			decodedEmail, err := url.QueryUnescape(cookie)
			if err == nil {
				authData.Email = decodedEmail
			} else {
				authData.Email = cookie // Fallback to raw value if decoding fails
			}
			// Clear the cookie
			c.SetCookie("verification_email", "", -1, "/", "", false, false)
		}

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
	r.GET("/reset-password", authController.ResetPasswordHandler)
	r.POST("/reset-password", authController.ResetPasswordHandler)
	r.GET("/reset-password/new", authController.ForgotPasswordHandler)
	r.POST("/reset-password/new", authController.ForgotPasswordHandler)
}

// handleAuthFlashMessage checks for flash messages in the session and adds them to the AuthData
func handleAuthFlashMessage(c *gin.Context, authData data.AuthData) data.AuthData {
	// Use the new session-based flash helper
	return handleSessionFlash(c, authData)
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

		// Add CSRF token for form protection
		csrfToken := middleware.GetCSRFToken(c)
		authData = authData.WithCSRFToken(csrfToken)

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

		// Add CSRF token for form protection
		csrfToken := middleware.GetCSRFToken(c)
		authData = authData.WithCSRFToken(csrfToken)

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

		// Add CSRF token for form protection
		csrfToken := middleware.GetCSRFToken(c)
		authData = authData.WithCSRFToken(csrfToken)

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

		// Add CSRF token for form protection
		csrfToken := middleware.GetCSRFToken(c)
		authData = authData.WithCSRFToken(csrfToken)

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
			authData.Title = "Reset Password"
		}

		// Add CSRF token for form protection
		csrfToken := middleware.GetCSRFToken(c)
		authData = authData.WithCSRFToken(csrfToken)

		// Handle flash messages
		authData = handleAuthFlashMessage(c, authData)

		auth.ResetPasswordRequest(authData).Render(c.Request.Context(), c.Writer)
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

		// Add CSRF token for form protection
		csrfToken := middleware.GetCSRFToken(c)
		authData = authData.WithCSRFToken(csrfToken)

		// Handle flash messages
		authData = handleAuthFlashMessage(c, authData)

		auth.ResetPassword(authData).Render(c.Request.Context(), c.Writer)
	}
}
