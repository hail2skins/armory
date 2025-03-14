package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDB is a mock implementation of the database.Service interface
type MockDB struct {
	mock.Mock
}

func (m *MockDB) Health() map[string]string {
	args := m.Called()
	return args.Get(0).(map[string]string)
}

func (m *MockDB) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDB) CreateUser(ctx context.Context, email, password string) (*database.User, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

func (m *MockDB) GetUserByEmail(ctx context.Context, email string) (*database.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

func (m *MockDB) AuthenticateUser(ctx context.Context, email, password string) (*database.User, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

func TestAuthenticationFlow(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	t.Run("Authentication flow changes navigation bar", func(t *testing.T) {
		mockDB := new(MockDB)

		// Setup mock responses
		testUser := &database.User{
			Email: "test@example.com",
		}
		database.SetUserID(testUser, 1)

		// GetUserByEmail returns nil for a new user
		mockDB.On("GetUserByEmail", mock.Anything, "test@example.com").Return(nil, nil)

		// CreateUser returns a new user
		mockDB.On("CreateUser", mock.Anything, "test@example.com", "password123").Return(testUser, nil)

		// AuthenticateUser returns the user when credentials are correct
		mockDB.On("AuthenticateUser", mock.Anything, "test@example.com", "password123").Return(testUser, nil)

		// Create auth controller
		authController := NewAuthController(mockDB)

		// Create a test HTTP server
		router := gin.New()

		// Setup render functions to capture HTML output
		var loginHTML, registerHTML string

		authController.RenderLogin = func(c *gin.Context, data interface{}) {
			loginHTML = "Login Form"
			c.String(http.StatusOK, loginHTML)
		}

		authController.RenderRegister = func(c *gin.Context, data interface{}) {
			registerHTML = "Register Form"
			c.String(http.StatusOK, registerHTML)
		}

		// Setup routes
		router.GET("/login", authController.LoginHandler)
		router.POST("/login", authController.LoginHandler)
		router.GET("/register", authController.RegisterHandler)
		router.POST("/register", authController.RegisterHandler)
		router.GET("/logout", authController.LogoutHandler)

		// Add auth-links endpoint
		router.GET("/auth-links", func(c *gin.Context) {
			_, authenticated := authController.GetCurrentUser(c)
			c.Header("Content-Type", "text/html")

			if authenticated {
				c.String(http.StatusOK, `<a href="/logout">Logout</a>`)
			} else {
				c.String(http.StatusOK, `<a href="/login">Login</a><a href="/register">Register</a>`)
			}
		})

		// Add home page
		router.GET("/", func(c *gin.Context) {
			_, authenticated := authController.GetCurrentUser(c)

			// Simplified home page with navigation
			html := `
			<nav>
				<a href="/">Home</a>
				<span id="auth-links">
			`

			if authenticated {
				html += `<a href="/logout">Logout</a>`
			} else {
				html += `<a href="/login">Login</a><a href="/register">Register</a>`
			}

			html += `
				</span>
			</nav>
			<main>Welcome to Armory</main>
			`

			c.Header("Content-Type", "text/html")
			c.String(http.StatusOK, html)
		})

		// Step 1: Check unauthenticated state
		t.Run("Unauthenticated user sees login and register links", func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/auth-links", nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusOK, resp.Code)
			assert.Contains(t, resp.Body.String(), `<a href="/login">Login</a>`)
			assert.Contains(t, resp.Body.String(), `<a href="/register">Register</a>`)
		})

		// Step 2: Register a new user
		t.Run("User can register", func(t *testing.T) {
			form := url.Values{}
			form.Add("email", "test@example.com")
			form.Add("password", "password123")
			form.Add("confirm_password", "password123")

			req, _ := http.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			// Should redirect to home page
			assert.Equal(t, http.StatusSeeOther, resp.Code)
			assert.Equal(t, "/", resp.Header().Get("Location"))

			// Check for auth cookie
			cookies := resp.Result().Cookies()
			var authCookie *http.Cookie
			for _, cookie := range cookies {
				if cookie.Name == "auth-session" {
					authCookie = cookie
					break
				}
			}
			assert.NotNil(t, authCookie, "Auth cookie should be set")
		})

		// Step 3: Check authenticated state
		t.Run("Authenticated user sees logout link", func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/auth-links", nil)
			// Add auth cookie from previous step
			req.AddCookie(&http.Cookie{
				Name:  "auth-session",
				Value: "1", // User ID from the test user
			})
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusOK, resp.Code)
			assert.Contains(t, resp.Body.String(), `<a href="/logout">Logout</a>`)
			assert.NotContains(t, resp.Body.String(), `<a href="/login">Login</a>`)
		})

		// Step 4: Logout
		t.Run("User can logout", func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/logout", nil)
			// Add auth cookie
			req.AddCookie(&http.Cookie{
				Name:  "auth-session",
				Value: "1", // User ID from the test user
			})
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			// Should redirect to home page
			assert.Equal(t, http.StatusSeeOther, resp.Code)
			assert.Equal(t, "/", resp.Header().Get("Location"))

			// Check that auth cookie is cleared
			cookies := resp.Result().Cookies()
			var authCookie *http.Cookie
			for _, cookie := range cookies {
				if cookie.Name == "auth-session" {
					authCookie = cookie
					break
				}
			}
			assert.NotNil(t, authCookie, "Auth cookie should be present")
			assert.Equal(t, "", authCookie.Value, "Auth cookie should be cleared")
			assert.Less(t, authCookie.MaxAge, 0, "Auth cookie should be expired")
		})

		// Step 5: Check unauthenticated state again
		t.Run("After logout, user sees login and register links", func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/auth-links", nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusOK, resp.Code)
			assert.Contains(t, resp.Body.String(), `<a href="/login">Login</a>`)
			assert.Contains(t, resp.Body.String(), `<a href="/register">Register</a>`)
		})
	})
}
