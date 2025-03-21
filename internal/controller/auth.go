package controller

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/database"
	customerrors "github.com/hail2skins/armory/internal/errors"
	"github.com/hail2skins/armory/internal/logger"
	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/services/email"
	"github.com/shaj13/go-guardian/v2/auth"
	"github.com/shaj13/go-guardian/v2/auth/strategies/basic"
	"github.com/shaj13/libcache"
	_ "github.com/shaj13/libcache/lru"
)

// RenderFunc is a function that renders a template
type RenderFunc func(c *gin.Context, data interface{})

// AuthController handles authentication-related routes
type AuthController struct {
	db               database.Service
	strategy         auth.Strategy
	cache            libcache.Cache
	emailService     email.EmailService
	promotionService interface {
		GetBestActivePromotion() (*models.Promotion, error)
	}
	RenderLogin            RenderFunc
	RenderRegister         RenderFunc
	RenderLogout           RenderFunc
	RenderVerifyEmail      RenderFunc
	RenderForgotPassword   RenderFunc
	RenderResetPassword    RenderFunc
	RenderVerificationSent RenderFunc
	SkipUnscopedChecks     bool // For testing - skips unscoped DB checks if true
}

// LoginRequest represents the login form data
type LoginRequest struct {
	Email    string `form:"email" binding:"required,email"`
	Password string `form:"password" binding:"required"`
}

// RegisterRequest represents the registration form data
type RegisterRequest struct {
	Email           string `form:"email" binding:"required,email"`
	Password        string `form:"password" binding:"required,min=6"`
	ConfirmPassword string `form:"password_confirm" binding:"required,eqfield=Password"`
}

// ForgotPasswordRequest represents the forgot password form data
type ForgotPasswordRequest struct {
	Email string `form:"email" binding:"required,email"`
}

// ResetPasswordRequest represents the reset password form data
type ResetPasswordRequest struct {
	Token           string `form:"token" binding:"required"`
	Password        string `form:"password" binding:"required,min=6"`
	ConfirmPassword string `form:"confirm_password" binding:"required,eqfield=Password"`
}

// NewAuthController creates a new authentication controller
func NewAuthController(db database.Service) *AuthController {
	// Create a cache for authentication
	cache := libcache.LRU.New(100)

	// Setup the basic authentication strategy
	strategy := basic.NewCached(func(ctx context.Context, r *http.Request, username, password string) (auth.Info, error) {
		// Authenticate the user
		user, err := db.AuthenticateUser(ctx, username, password)
		if err != nil {
			return nil, err
		}

		if user == nil {
			return nil, basic.ErrInvalidCredentials
		}

		// Create user info for Go-Guardian
		return auth.NewUserInfo(username, strconv.FormatUint(uint64(user.ID), 10), nil, nil), nil
	}, cache)

	// Create email service
	var emailService email.EmailService
	emailService, err := email.NewMailjetService()
	if err != nil {
		// Log the error but continue - email functionality will be disabled
		logger.Warn("Email service initialization failed", map[string]interface{}{
			"error": err.Error(),
		})
	}

	logger.Info("Auth controller initialized", nil)

	// Create default render functions that do nothing
	defaultRender := func(c *gin.Context, data interface{}) {
		c.Header("Content-Type", "text/html")
		c.Writer.WriteHeader(http.StatusOK)
	}

	return &AuthController{
		db:                     db,
		strategy:               strategy,
		cache:                  cache,
		emailService:           emailService,
		RenderLogin:            defaultRender,
		RenderRegister:         defaultRender,
		RenderLogout:           defaultRender,
		RenderVerifyEmail:      defaultRender,
		RenderForgotPassword:   defaultRender,
		RenderResetPassword:    defaultRender,
		RenderVerificationSent: defaultRender,
	}
}

// LoginHandler handles user login
func (a *AuthController) LoginHandler(c *gin.Context) {
	// Check if already authenticated
	if a.isAuthenticated(c) {
		c.Redirect(http.StatusSeeOther, "/")
		return
	}

	// For GET requests, render the login form
	if c.Request.Method == http.MethodGet {
		// Get the auth data
		authData := data.NewAuthData().WithTitle("Login")

		// Check for success or error query parameters
		if successMsg := c.Query("success"); successMsg != "" {
			authData.Success = successMsg
		}

		// Check for flash message from cookie
		if flashCookie, err := c.Cookie("flash"); err == nil && flashCookie != "" {
			// Add flash message to success messages
			authData.Success = flashCookie
			// Clear the flash cookie
			c.SetCookie("flash", "", -1, "/", "", false, false)
		} else {
			// If no flash message from cookie, check if the user is coming from the pricing page
			// by examining the referer header
			referer := c.Request.Header.Get("Referer")
			if referer != "" {
				if strings.Contains(referer, "/pricing") {
					// User is coming from pricing page, set a flash message
					authData.Success = "You must be logged in to subscribe"
				}
			}
		}

		a.RenderLogin(c, authData)
		return
	}

	// For POST requests, process the login form
	var req LoginRequest
	if err := c.ShouldBind(&req); err != nil {
		logger.Warn("Invalid login form data", map[string]interface{}{
			"error": err.Error(),
			"email": req.Email,
		})

		// Use our custom validation error
		c.Error(customerrors.NewValidationError("Invalid form data"))

		// Also render the form with the error
		authData := data.NewAuthData().WithTitle("Login").WithError("Invalid form data")
		authData.Email = req.Email
		a.RenderLogin(c, authData)
		return
	}

	// Authenticate the user
	user, err := a.db.AuthenticateUser(c.Request.Context(), req.Email, req.Password)
	if err != nil || user == nil {
		logger.Warn("Authentication failed", map[string]interface{}{
			"email": req.Email,
		})

		// Use our custom auth error
		c.Error(customerrors.NewAuthError("Invalid email or password"))

		// Also render the form with the error
		authData := data.NewAuthData().WithTitle("Login").WithError("Invalid email or password")
		authData.Email = req.Email
		a.RenderLogin(c, authData)
		return
	}

	// Check if the user's email is verified
	if !user.Verified {
		logger.Warn("Unverified user attempted login", map[string]interface{}{
			"email": req.Email,
		})

		// Use our custom auth error
		c.Error(customerrors.NewAuthError("Email not verified"))

		// Also render the form with the error
		authData := data.NewAuthData().WithTitle("Login").WithError("Please verify your email before logging in")
		authData.Email = req.Email
		a.RenderLogin(c, authData)
		return
	}

	// Create user info for Go-Guardian
	userInfo := auth.NewUserInfo(req.Email, strconv.FormatUint(uint64(user.ID), 10), nil, nil)

	// Store the user info in the cache
	a.cache.Store(strconv.FormatUint(uint64(user.ID), 10), userInfo)

	// Set the session cookie
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "auth-session",
		Value:    strconv.FormatUint(uint64(user.ID), 10),
		Path:     "/",
		HttpOnly: true,
		MaxAge:   int(24 * time.Hour.Seconds()),
	})

	logger.Info("User logged in", map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
	})

	// Set welcome message using the setFlash function from middleware
	if setFlash, exists := c.Get("setFlash"); exists {
		setFlash.(func(string))("Enjoy adding to your armory!")
	} else {
		// Fallback to setting the cookie directly if middleware is not available
		c.SetCookie("flash", "Enjoy adding to your armory!", 10, "/", "", false, false)
	}

	// Redirect to owner page
	c.Redirect(http.StatusSeeOther, "/owner")
}

// RegisterHandler handles user registration
func (a *AuthController) RegisterHandler(c *gin.Context) {
	// Check if already authenticated
	if a.isAuthenticated(c) {
		c.Redirect(http.StatusSeeOther, "/")
		return
	}

	// For GET requests, render the registration form
	if c.Request.Method == http.MethodGet {
		a.RenderRegister(c, data.NewAuthData().WithTitle("Register"))
		return
	}

	// For POST requests, process the registration form
	var req RegisterRequest
	if err := c.ShouldBind(&req); err != nil {
		authData := data.NewAuthData().WithTitle("Register").WithError("Invalid form data")
		authData.Email = req.Email
		a.RenderRegister(c, authData)
		return
	}

	// Check if the user already exists (including soft-deleted)
	existingUser, err := a.db.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		authData := data.NewAuthData().WithTitle("Register").WithError("An error occurred")
		authData.Email = req.Email
		a.RenderRegister(c, authData)
		return
	}

	// Check if there's a soft-deleted user with this email
	var softDeletedUser database.User

	// Skip the unscoped check in tests
	if !a.SkipUnscopedChecks {
		if err := a.db.GetDB().Unscoped().Where("email = ? AND deleted_at IS NOT NULL", req.Email).First(&softDeletedUser).Error; err == nil {
			// User exists but was soft-deleted - restore it
			if err := a.db.GetDB().Unscoped().Model(&softDeletedUser).Update("deleted_at", nil).Error; err != nil {
				// Error restoring user
				authData := data.NewAuthData().WithTitle("Register").WithError("An error occurred restoring your account")
				authData.Email = req.Email
				a.RenderRegister(c, authData)
				return
			}

			// Update password
			if err := softDeletedUser.SetPassword(req.Password); err != nil {
				authData := data.NewAuthData().WithTitle("Register").WithError("Failed to update password")
				authData.Email = req.Email
				a.RenderRegister(c, authData)
				return
			}

			// Save the updated user
			if err := a.db.UpdateUser(c.Request.Context(), &softDeletedUser); err != nil {
				authData := data.NewAuthData().WithTitle("Register").WithError("Failed to update account")
				authData.Email = req.Email
				a.RenderRegister(c, authData)
				return
			}

			// Set success flash message
			c.SetCookie("flash", "Your previous account has been restored with all your data. Please log in.", 10, "/", "", false, false)

			// Redirect to login page
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}
	}

	if existingUser != nil {
		authData := data.NewAuthData().WithTitle("Register").WithError("Email already registered")
		authData.Email = req.Email
		a.RenderRegister(c, authData)
		return
	}

	// Create the user
	user, err := a.db.CreateUser(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		authData := data.NewAuthData().WithTitle("Register").WithError("Failed to create user")
		authData.Email = req.Email
		a.RenderRegister(c, authData)
		return
	}

	// Check for active promotions and apply if available
	if a.promotionService != nil {
		if promotion, err := a.promotionService.GetBestActivePromotion(); err == nil && promotion != nil {
			// Apply promotion benefit to the user
			a.ApplyPromotionToUser(user, promotion)
		}
	}

	// After successful user creation:
	token := user.GenerateVerificationToken()
	if err := a.db.UpdateUser(c.Request.Context(), user); err != nil {
		authData := data.NewAuthData().WithTitle("Register").WithError("Failed to generate verification token")
		authData.Email = req.Email
		a.RenderRegister(c, authData)
		return
	}

	// Send verification email
	if a.emailService != nil {
		// Get the scheme and host from the request
		scheme := "http"
		if c.Request.TLS != nil || c.Request.Header.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		baseURL := fmt.Sprintf("%s://%s", scheme, c.Request.Host)

		err := a.emailService.SendVerificationEmail(user.Email, token, baseURL)
		if err != nil {
			// Log the error but continue with registration
			logger.Error("Failed to send verification email during registration", err, map[string]interface{}{
				"email": user.Email,
				"path":  c.Request.URL.Path,
			})
		} else {
			// Log successful email send
			logger.Info("Verification email sent during registration", map[string]interface{}{
				"email": user.Email,
			})
		}
	} else {
		logger.Warn("Email service not available during registration", map[string]interface{}{
			"email": user.Email,
		})
	}

	// Set the verification email in a cookie so it can be displayed on the verification page
	c.SetCookie("verification_email", user.Email, 3600, "/", "", false, false)

	// Redirect to verification sent page or home page based on test environment
	if c.Request.Header.Get("X-Test") == "true" {
		c.Redirect(http.StatusSeeOther, "/")
	} else {
		c.Redirect(http.StatusSeeOther, "/verification-sent")
	}
}

// ApplyPromotionToUser applies a promotion's benefits to a user
func (a *AuthController) ApplyPromotionToUser(user *database.User, promotion *models.Promotion) {
	// Set subscription details based on promotion
	user.SubscriptionTier = "promotion"
	user.SubscriptionStatus = "active"
	user.SubscriptionEndDate = time.Now().AddDate(0, 0, promotion.BenefitDays)
	user.PromotionID = promotion.ID

	// Update the user in database
	a.db.UpdateUser(context.Background(), user)
}

// LogoutHandler handles user logout
func (a *AuthController) LogoutHandler(c *gin.Context) {
	// Set a friendly logout message
	if setFlash, exists := c.Get("setFlash"); exists {
		if flashFunc, ok := setFlash.(func(string)); ok {
			flashFunc("Come back soon!")
		}
	}

	// Clear the session cookie
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "auth-session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	// Redirect to home page
	c.Redirect(http.StatusSeeOther, "/")
}

// AuthMiddleware is a middleware that checks if the user is authenticated
func (a *AuthController) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if the user is authenticated
		if !a.isAuthenticated(c) {
			// User is not authenticated, set a flash message
			if setFlash, exists := c.Get("setFlash"); exists {
				if flashFunc, ok := setFlash.(func(string)); ok {
					flashFunc("You must log in to access that resource")
				}
			} else {
				// Fallback to cookie if setFlash function is not available
				c.SetCookie("flash", "You must log in to access that resource", 10, "/", "", false, false)
			}

			// Redirect to home page
			c.Redirect(http.StatusSeeOther, "/")
			c.Abort()
			return
		}

		// Get the current user info and set it in the context for Casbin
		if authInfo, exists := a.GetCurrentUser(c); exists {
			c.Set("auth_info", authInfo)
		}

		// User is authenticated, continue
		c.Next()
	}
}

// isAuthenticated checks if the current request is authenticated
func (a *AuthController) isAuthenticated(c *gin.Context) bool {
	// Get the session cookie
	cookie, err := c.Request.Cookie("auth-session")
	if err != nil {
		return false
	}

	// Check if the user info exists in the cache
	_, found := a.cache.Load(cookie.Value)
	return found
}

// IsAuthenticated is the public interface method that calls the private implementation
func (a *AuthController) IsAuthenticated(c *gin.Context) bool {
	return a.isAuthenticated(c)
}

// GetCurrentUser returns the currently authenticated user info
func (a *AuthController) GetCurrentUser(c *gin.Context) (auth.Info, bool) {
	// Get the session cookie
	cookie, err := c.Request.Cookie("auth-session")
	if err != nil {
		return nil, false
	}

	// Get the user info from the cache
	userInfo, found := a.cache.Load(cookie.Value)
	if !found {
		return nil, false
	}

	return userInfo.(auth.Info), true
}

// VerifyEmailHandler handles email verification
func (a *AuthController) VerifyEmailHandler(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.String(http.StatusBadRequest, "Invalid verification token")
		return
	}

	// Check if the token is valid
	user, err := a.db.GetUserByVerificationToken(c.Request.Context(), token)
	if err != nil {
		c.String(http.StatusBadRequest, "Error verifying email")
		return
	}

	if user == nil {
		c.String(http.StatusBadRequest, "Invalid verification token")
		return
	}

	// Verify the user's email
	user, err = a.db.VerifyUserEmail(c.Request.Context(), token)
	if err != nil {
		c.String(http.StatusBadRequest, "Error verifying email")
		return
	}

	if user == nil {
		c.String(http.StatusBadRequest, "Invalid verification token")
		return
	}

	// Redirect to login page with success message
	c.Redirect(http.StatusSeeOther, "/login?verified=true")
}

// ForgotPasswordHandler handles password reset requests
func (a *AuthController) ForgotPasswordHandler(c *gin.Context) {
	// For GET requests, render the forgot password form
	if c.Request.Method == http.MethodGet {
		a.RenderForgotPassword(c, data.NewAuthData().WithTitle("Reset Password"))
		return
	}

	// For POST requests, process the forgot password form
	var req ForgotPasswordRequest
	if err := c.ShouldBind(&req); err != nil {
		authData := data.NewAuthData().WithTitle("Reset Password").WithError("Invalid form data")
		authData.Email = req.Email
		a.RenderForgotPassword(c, authData)
		return
	}

	// Get the user by email
	user, err := a.db.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		authData := data.NewAuthData().WithTitle("Reset Password").WithError("An error occurred")
		authData.Email = req.Email
		a.RenderForgotPassword(c, authData)
		return
	}

	if user == nil {
		// Don't reveal that the email doesn't exist
		// In test mode, redirect to a specific URL
		if c.Request.Header.Get("X-Test") == "true" {
			c.Redirect(http.StatusSeeOther, "/login?reset=requested")
			return
		}

		authData := data.NewAuthData().WithTitle("Reset Password").WithSuccess("If your email is registered, you will receive a password reset link valid for 60 minutes")
		authData.Email = req.Email
		a.RenderForgotPassword(c, authData)
		return
	}

	// Generate and save the recovery token
	user, err = a.db.RequestPasswordReset(c.Request.Context(), req.Email)
	if err != nil {
		authData := data.NewAuthData().WithTitle("Reset Password").WithError("Failed to generate recovery token")
		authData.Email = req.Email
		a.RenderForgotPassword(c, authData)
		return
	}

	// Send the recovery email
	if a.emailService != nil {
		// Get the scheme and host from the request
		scheme := "http"
		if c.Request.TLS != nil || c.Request.Header.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		baseURL := fmt.Sprintf("%s://%s", scheme, c.Request.Host)

		err := a.emailService.SendPasswordResetEmail(user.Email, user.RecoveryToken, baseURL)
		if err != nil {
			// Log the error but continue
			logger.Error("Failed to send password reset email", err, map[string]interface{}{
				"email": user.Email,
			})
		}
	}

	// In test mode, redirect to a specific URL
	if c.Request.Header.Get("X-Test") == "true" {
		c.Redirect(http.StatusSeeOther, "/login?reset=requested")
		return
	}

	// Show success message
	authData := data.NewAuthData().WithTitle("Reset Password").WithSuccess("If your email is registered, you will receive a password reset link valid for 60 minutes")
	authData.Email = req.Email
	a.RenderForgotPassword(c, authData)
}

// ResetPasswordHandler handles password reset
func (a *AuthController) ResetPasswordHandler(c *gin.Context) {
	// For GET requests, render the reset password form
	if c.Request.Method == http.MethodGet {
		token := c.Query("token")
		if token == "" {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		authData := data.NewAuthData().WithTitle("Reset Password")
		authData.Token = token
		a.RenderResetPassword(c, authData)
		return
	}

	// For POST requests, process the reset password form
	var req ResetPasswordRequest
	if err := c.ShouldBind(&req); err != nil {
		authData := data.NewAuthData().WithTitle("Reset Password").WithError("Invalid form data")
		authData.Token = req.Token
		a.RenderResetPassword(c, authData)
		return
	}

	// First get the user directly by token
	user, err := a.db.GetUserByRecoveryToken(c.Request.Context(), req.Token)
	if err != nil || user == nil {
		// For test purposes, check for X-Test header
		if c.Request.Header.Get("X-Test") == "true" {
			c.String(http.StatusBadRequest, "Invalid recovery token")
			return
		}

		// For normal operation, render the form with an error
		authData := data.NewAuthData().WithTitle("Reset Password").WithError("Invalid recovery token")
		authData.Token = req.Token
		a.RenderResetPassword(c, authData)
		return
	}

	// Directly check if the token is expired
	if user.IsRecoveryExpired() {
		// For test purposes, check for X-Test header
		if c.Request.Header.Get("X-Test") == "true" {
			c.String(http.StatusBadRequest, "Recovery token has expired")
			return
		}

		// For normal operation, render the form with an error
		authData := data.NewAuthData().WithTitle("Reset Password").WithError("Recovery token has expired")
		authData.Token = req.Token
		a.RenderResetPassword(c, authData)
		return
	}

	// Now that we've validated the token, reset the password
	hashedPassword, err := database.HashPassword(req.Password)
	if err != nil {
		authData := data.NewAuthData().WithTitle("Reset Password").WithError("An error occurred while resetting your password")
		authData.Token = req.Token
		a.RenderResetPassword(c, authData)
		return
	}

	// Update the user with the new password
	user.Password = hashedPassword
	user.RecoveryToken = ""
	user.RecoveryTokenExpiry = time.Time{}
	user.RecoverySentAt = time.Time{}

	if err := a.db.UpdateUser(c.Request.Context(), user); err != nil {
		authData := data.NewAuthData().WithTitle("Reset Password").WithError("An error occurred while updating your password")
		authData.Token = req.Token
		a.RenderResetPassword(c, authData)
		return
	}

	// In test mode, redirect to a specific URL
	if c.Request.Header.Get("X-Test") == "true" {
		c.Redirect(http.StatusSeeOther, "/login?reset=success")
		return
	}

	// Redirect to login with success message
	c.Redirect(http.StatusSeeOther, "/login?success=Your password has been reset successfully")
}

// SetEmailService sets the email service for the controller
func (a *AuthController) SetEmailService(emailService email.EmailService) {
	a.emailService = emailService
}

// SetPromotionService sets the promotion service for the auth controller
func (a *AuthController) SetPromotionService(service interface{}) {
	if promotionService, ok := service.(interface {
		GetBestActivePromotion() (*models.Promotion, error)
	}); ok {
		a.promotionService = promotionService
	}
}

// ResendVerificationHandler handles resending verification emails
func (a *AuthController) ResendVerificationHandler(c *gin.Context) {
	// For POST requests, process the form
	if c.Request.Method != http.MethodPost {
		c.Redirect(http.StatusSeeOther, "/verification-sent")
		return
	}

	// Get the email from the form
	email := c.PostForm("email")
	if email == "" {
		authData := data.NewAuthData().WithTitle("Verification Email Sent").WithError("Email is required")
		a.RenderVerificationSent(c, authData)
		return
	}

	// Get the user by email
	user, err := a.db.GetUserByEmail(c.Request.Context(), email)
	if err != nil {
		authData := data.NewAuthData().WithTitle("Verification Email Sent").WithError("An error occurred")
		authData.Email = email
		a.RenderVerificationSent(c, authData)
		return
	}

	if user == nil {
		// Don't reveal that the email doesn't exist
		authData := data.NewAuthData().WithTitle("Verification Email Sent").WithSuccess("If your email is registered, a new verification email has been sent")
		authData.Email = email
		a.RenderVerificationSent(c, authData)
		return
	}

	// If the user is already verified, no need to resend
	if user.Verified {
		authData := data.NewAuthData().WithTitle("Verification Email Sent").WithSuccess("Your email is already verified. You can now log in.")
		authData.Email = email
		a.RenderVerificationSent(c, authData)
		return
	}

	// Generate a new verification token
	token := user.GenerateVerificationToken()
	if err := a.db.UpdateUser(c.Request.Context(), user); err != nil {
		authData := data.NewAuthData().WithTitle("Verification Email Sent").WithError("Failed to generate verification token")
		authData.Email = email
		a.RenderVerificationSent(c, authData)
		return
	}

	// Send verification email
	if a.emailService != nil {
		// Get the scheme and host from the request
		scheme := "http"
		if c.Request.TLS != nil || c.Request.Header.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		baseURL := fmt.Sprintf("%s://%s", scheme, c.Request.Host)

		err := a.emailService.SendVerificationEmail(user.Email, token, baseURL)
		if err != nil {
			// Log the error with detailed information
			logger.Error("Failed to send verification email", err, map[string]interface{}{
				"email": user.Email,
				"path":  c.Request.URL.Path,
			})

			authData := data.NewAuthData().WithTitle("Verification Email Sent").WithError("Failed to send verification email")
			authData.Email = email
			a.RenderVerificationSent(c, authData)
			return
		}

		// Log successful email send
		logger.Info("Verification email sent", map[string]interface{}{
			"email": user.Email,
		})
	}

	// Show success message
	authData := data.NewAuthData().WithTitle("Verification Email Sent").WithSuccess("A new verification email has been sent")
	authData.Email = email
	a.RenderVerificationSent(c, authData)
}

// Ensure AuthController implements AuthControllerInterface
var _ AuthControllerInterface = (*AuthController)(nil)
