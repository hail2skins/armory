package testhelper

import (
	"context"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/database"
	"github.com/shaj13/go-guardian/v2/auth"
)

// AuthInfo represents authentication information for a user
type AuthInfo interface {
	GetUserName() string
	GetID() string
	GetGroups() []string
	GetExtensions() auth.Extensions
}

// AuthService defines the interface for authentication-related operations
type AuthService interface {
	// Authenticate a user with the given email and password
	AuthenticateUser(ctx context.Context, email, password string) (*database.User, error)

	// Get the currently authenticated user from the context
	GetCurrentUser(c *gin.Context) (auth.Info, bool)

	// Check if the current request is authenticated
	IsAuthenticated(c *gin.Context) bool

	// Set up authentication for testing (middleware setup)
	SetupAuthMiddleware(router *gin.Engine) *gin.Engine

	// Set up an authenticated router for testing
	SetupAuthenticatedRouter(userID uint, email string) *gin.Engine
}

// MockAuthInfo implements the AuthInfo interface for testing
type MockAuthInfo struct {
	Username   string
	ID         string
	Groups     []string
	Extensions auth.Extensions
}

func (m *MockAuthInfo) GetUserName() string {
	return m.Username
}

func (m *MockAuthInfo) SetUserName(username string) {
	m.Username = username
}

func (m *MockAuthInfo) GetID() string {
	if m.ID == "" {
		return "1"
	}
	return m.ID
}

func (m *MockAuthInfo) SetID(id string) {
	m.ID = id
}

func (m *MockAuthInfo) GetGroups() []string {
	return m.Groups
}

func (m *MockAuthInfo) SetGroups(groups []string) {
	m.Groups = groups
}

func (m *MockAuthInfo) GetExtensions() auth.Extensions {
	if m.Extensions == nil {
		return auth.Extensions{}
	}
	return m.Extensions
}

func (m *MockAuthInfo) SetExtensions(exts auth.Extensions) {
	m.Extensions = exts
}

// MockAuthService implements AuthService for testing
type MockAuthService struct {
	DB database.Service
}

// NewMockAuthService creates a new mock authentication service
func NewMockAuthService(db database.Service) *MockAuthService {
	return &MockAuthService{
		DB: db,
	}
}

// AuthenticateUser mocks authenticating a user
func (m *MockAuthService) AuthenticateUser(ctx context.Context, email, password string) (*database.User, error) {
	// For testing, we'll simplify and accept preset credentials
	if email == "test@example.com" && password == "Password123!" {
		user := &database.User{}
		user.Model.ID = 1
		user.Email = email
		user.Verified = true
		return user, nil
	}
	return nil, nil
}

// GetCurrentUser mocks getting the current user from context
func (m *MockAuthService) GetCurrentUser(c *gin.Context) (auth.Info, bool) {
	// Check if the user is set in the context (from middleware)
	if user, exists := c.Get("user"); exists {
		info := &MockAuthInfo{
			Username: user.(gin.H)["email"].(string),
			ID:       "1",
		}
		return info, true
	}
	return nil, false
}

// IsAuthenticated checks if the current request is authenticated
func (m *MockAuthService) IsAuthenticated(c *gin.Context) bool {
	_, exists := c.Get("authenticated")
	return exists
}

// SetupAuthMiddleware sets up authentication middleware for testing
func (m *MockAuthService) SetupAuthMiddleware(router *gin.Engine) *gin.Engine {
	router.Use(func(c *gin.Context) {
		// Set flash middleware function
		c.Set("setFlash", func(msg string) {
			c.Set("flash_message", msg)
		})
		c.Next()
	})
	return router
}

// SetupAuthenticatedRouter sets up an authenticated router for testing
func (m *MockAuthService) SetupAuthenticatedRouter(userID uint, email string) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	// Add sessions middleware
	store := cookie.NewStore([]byte("secret"))
	router.Use(sessions.Sessions("armory_session", store))

	// Add flash middleware
	router.Use(func(c *gin.Context) {
		c.Set("setFlash", func(msg string) {
			c.Set("flash_message", msg)
		})
		c.Next()
	})

	// Add authentication middleware
	router.Use(func(c *gin.Context) {
		// Create user data
		c.Set("user", gin.H{"id": userID, "email": email})
		c.Set("authenticated", true)

		// Set the auth controller for controller methods that check it
		c.Set("authController", m)

		// Set a csrf token for forms
		c.Set("csrf_token", "test_csrf_token")

		// Set auth data for views
		c.Set("authData", gin.H{
			"IsAuthenticated": true,
			"UserEmail":       email,
			"UserID":          userID,
			"CSRFToken":       "test_csrf_token",
		})

		c.Next()
	})

	return router
}
