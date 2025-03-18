package mocks

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/services/email"
)

// TestAuthController is a simplified version of the auth controller for testing
type TestAuthController struct {
	DB                     database.Service
	EmailService           email.EmailService
	RenderLogin            func(c *gin.Context, data interface{})
	RenderRegister         func(c *gin.Context, data interface{})
	RenderLogout           func(c *gin.Context, data interface{})
	RenderForgotPassword   func(c *gin.Context, data interface{})
	RenderResetPassword    func(c *gin.Context, data interface{})
	RenderVerificationSent func(c *gin.Context, data interface{})
}

// NewTestAuthController creates a new test auth controller
func NewTestAuthController(db database.Service) *TestAuthController {
	return &TestAuthController{
		DB: db,
	}
}

// SetEmailService sets the email service
func (a *TestAuthController) SetEmailService(emailService email.EmailService) {
	a.EmailService = emailService
}

// ForgotPasswordHandler handles forgot password requests
func (a *TestAuthController) ForgotPasswordHandler(c *gin.Context) {
	if c.Request.Method == http.MethodGet {
		// GET request - render the form
		authData := data.NewAuthData().WithTitle("Reset Password")
		if a.RenderForgotPassword != nil {
			a.RenderForgotPassword(c, authData)
		} else {
			c.String(http.StatusOK, "Reset Password Form")
		}
		return
	}

	// POST request - process the form
	var req struct {
		Email string `form:"email" binding:"required,email"`
	}

	if err := c.ShouldBind(&req); err != nil {
		authData := data.NewAuthData().WithTitle("Reset Password").WithError("Invalid email format")
		if a.RenderForgotPassword != nil {
			a.RenderForgotPassword(c, authData)
		} else {
			c.String(http.StatusOK, "Error: Invalid email format")
		}
		return
	}

	// Check if user exists
	user, err := a.DB.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil || user == nil {
		// For security reasons, don't reveal if the user exists or not
		authData := data.NewAuthData().WithTitle("Reset Password").WithSuccess("If your email is registered, you will receive a password reset link.")
		if a.RenderForgotPassword != nil {
			a.RenderForgotPassword(c, authData)
		} else {
			c.String(http.StatusOK, "If your email is registered, you will receive a password reset link.")
		}
		return
	}

	// Generate recovery token
	user, err = a.DB.RequestPasswordReset(c.Request.Context(), req.Email)
	if err != nil {
		authData := data.NewAuthData().WithTitle("Reset Password").WithError("Failed to process your request")
		if a.RenderForgotPassword != nil {
			a.RenderForgotPassword(c, authData)
		} else {
			c.String(http.StatusOK, "Failed to process your request")
		}
		return
	}

	// Send password reset email
	if a.EmailService != nil {
		err = a.EmailService.SendPasswordResetEmail(user.Email, user.RecoveryToken)
		if err != nil {
			authData := data.NewAuthData().WithTitle("Reset Password").WithError("Failed to send password reset email")
			if a.RenderForgotPassword != nil {
				a.RenderForgotPassword(c, authData)
			} else {
				c.String(http.StatusOK, "Failed to send password reset email")
			}
			return
		}
	}

	// Send success response
	authData := data.NewAuthData().WithTitle("Reset Password").WithSuccess("Password reset email sent. Please check your inbox.")
	if a.RenderForgotPassword != nil {
		a.RenderForgotPassword(c, authData)
	} else {
		c.String(http.StatusOK, "Password reset email sent. Please check your inbox.")
	}
}

// ResetPasswordHandler handles password reset
func (a *TestAuthController) ResetPasswordHandler(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		token = c.PostForm("token")
	}

	if token == "" {
		fmt.Printf("ERROR: No token provided in request\n")
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Verify token exists
	user, err := a.DB.GetUserByRecoveryToken(c.Request.Context(), token)
	if err != nil || user == nil {
		if err != nil {
			fmt.Printf("ERROR: Failed to get user by recovery token: %s\n", err.Error())
		} else {
			fmt.Printf("ERROR: No user found with recovery token: %s\n", token)
		}
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Check if token is expired
	isExpired, err := a.DB.IsRecoveryExpired(c.Request.Context(), token)
	if err != nil || isExpired {
		if err != nil {
			fmt.Printf("ERROR: Failed to check if token is expired: %s\n", err.Error())
		} else if isExpired {
			fmt.Printf("ERROR: Recovery token is expired\n")
		}
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	if c.Request.Method == http.MethodGet {
		// GET request - render the form
		authData := data.NewAuthData().WithTitle("Reset Password")
		authData.Token = token

		if a.RenderResetPassword != nil {
			a.RenderResetPassword(c, authData)
		} else {
			c.String(http.StatusOK, "Reset Password Form")
		}
		return
	}

	// POST request - process the form
	password := c.PostForm("password")
	confirmPassword := c.PostForm("confirm_password")

	if password == "" || confirmPassword == "" {
		fmt.Printf("ERROR: Empty password fields. Password: '%s', Confirm: '%s'\n", password, confirmPassword)
		authData := data.NewAuthData().WithTitle("Reset Password").WithError("Password cannot be empty")
		authData.Token = token

		if a.RenderResetPassword != nil {
			a.RenderResetPassword(c, authData)
		} else {
			c.String(http.StatusOK, "Error: Password cannot be empty")
		}
		return
	}

	if password != confirmPassword {
		fmt.Printf("ERROR: Passwords do not match. Password: '%s', Confirm: '%s'\n", password, confirmPassword)
		authData := data.NewAuthData().WithTitle("Reset Password").WithError("Passwords do not match")
		authData.Token = token

		if a.RenderResetPassword != nil {
			a.RenderResetPassword(c, authData)
		} else {
			c.String(http.StatusOK, "Error: Passwords do not match")
		}
		return
	}

	// Reset password
	fmt.Printf("DEBUG: Resetting password for user %s with token %s\n", user.Email, token)
	err = a.DB.ResetPassword(c.Request.Context(), token, password)
	if err != nil {
		fmt.Printf("ERROR: Failed to reset password: %s\n", err.Error())
		authData := data.NewAuthData().WithTitle("Reset Password").WithError("Failed to reset password: " + err.Error())
		authData.Token = token

		if a.RenderResetPassword != nil {
			a.RenderResetPassword(c, authData)
		} else {
			c.String(http.StatusOK, "Failed to reset password: "+err.Error())
		}
		return
	}

	fmt.Printf("SUCCESS: Password reset successful for user %s\n", user.Email)
	// Redirect to login
	c.Redirect(http.StatusSeeOther, "/login")
}
