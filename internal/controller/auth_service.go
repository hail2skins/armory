package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/shaj13/go-guardian/v2/auth"
)

// AuthService defines the interface for authentication services
// This interface is implemented by AuthController and MockAuthController
type AuthService interface {
	// GetCurrentUser returns the current user and authentication status
	GetCurrentUser(c *gin.Context) (auth.Info, bool)

	// IsAuthenticated returns true if the user is authenticated
	IsAuthenticated(c *gin.Context) bool

	// LoginHandler handles login requests
	LoginHandler(c *gin.Context)

	// LogoutHandler handles logout requests
	LogoutHandler(c *gin.Context)

	// RegisterHandler handles registration requests
	RegisterHandler(c *gin.Context)

	// VerifyEmailHandler handles email verification
	VerifyEmailHandler(c *gin.Context)

	// ForgotPasswordHandler handles forgot password requests
	ForgotPasswordHandler(c *gin.Context)

	// ResetPasswordHandler handles password reset requests
	ResetPasswordHandler(c *gin.Context)
}
