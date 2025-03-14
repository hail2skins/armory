package controller

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/database"
	"github.com/shaj13/go-guardian/v2/auth"
	"github.com/shaj13/go-guardian/v2/auth/strategies/basic"
	"github.com/shaj13/libcache"
	_ "github.com/shaj13/libcache/lru"
)

// RenderFunc is a function that renders a template
type RenderFunc func(c *gin.Context, data gin.H)

// AuthController handles authentication-related routes
type AuthController struct {
	db             database.Service
	strategy       auth.Strategy
	cache          libcache.Cache
	RenderLogin    RenderFunc
	RenderRegister RenderFunc
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

	// Create default render functions that do nothing
	defaultRender := func(c *gin.Context, data gin.H) {
		c.Header("Content-Type", "text/html")
		c.Writer.WriteHeader(http.StatusOK)
	}

	return &AuthController{
		db:             db,
		strategy:       strategy,
		cache:          cache,
		RenderLogin:    defaultRender,
		RenderRegister: defaultRender,
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
		a.RenderLogin(c, gin.H{})
		return
	}

	// For POST requests, process the login form
	var req LoginRequest
	if err := c.ShouldBind(&req); err != nil {
		a.RenderLogin(c, gin.H{
			"Error": "Invalid form data",
		})
		return
	}

	// Authenticate the user
	user, err := a.db.AuthenticateUser(c.Request.Context(), req.Email, req.Password)
	if err != nil || user == nil {
		a.RenderLogin(c, gin.H{
			"Error": "Invalid email or password",
			"Email": req.Email,
		})
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

	// Redirect to home page
	c.Redirect(http.StatusSeeOther, "/")
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
		a.RenderRegister(c, gin.H{})
		return
	}

	// For POST requests, process the registration form
	var req RegisterRequest
	if err := c.ShouldBind(&req); err != nil {
		a.RenderRegister(c, gin.H{
			"Error": "Invalid form data",
		})
		return
	}

	// Check if the user already exists
	existingUser, err := a.db.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		a.RenderRegister(c, gin.H{
			"Error": "An error occurred",
			"Email": req.Email,
		})
		return
	}

	if existingUser != nil {
		a.RenderRegister(c, gin.H{
			"Error": "Email already registered",
			"Email": req.Email,
		})
		return
	}

	// Create the user
	user, err := a.db.CreateUser(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		a.RenderRegister(c, gin.H{
			"Error": "Failed to create user",
			"Email": req.Email,
		})
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

	// Redirect to home page
	c.Redirect(http.StatusSeeOther, "/")
}

// LogoutHandler handles user logout
func (a *AuthController) LogoutHandler(c *gin.Context) {
	// Get the session cookie
	cookie, err := c.Request.Cookie("auth-session")
	if err == nil {
		// Remove the user info from the cache
		a.cache.Delete(cookie.Value)
	}

	// Clear the cookie
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "auth-session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	// Redirect to home page
	c.Redirect(http.StatusSeeOther, "/")
}

// AuthMiddleware is a middleware that checks if the user is authenticated
func (a *AuthController) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if the user is authenticated
		if !a.isAuthenticated(c) {
			// User is not authenticated, redirect to login page
			c.Redirect(http.StatusSeeOther, "/login")
			c.Abort()
			return
		}

		// User is authenticated, continue
		c.Next()
	}
}

// isAuthenticated checks if the user is authenticated
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

// GetCurrentUser gets the current authenticated user
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
