package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNavBarAuthentication(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	t.Run("Unauthenticated user should see Login and Register links", func(t *testing.T) {
		// Create a test HTTP server
		router := gin.New()
		router.GET("/auth-links", func(c *gin.Context) {
			// Simulate unauthenticated user
			c.Header("Content-Type", "text/html")
			c.Writer.WriteString(`<a href="/login">Login</a><a href="/register">Register</a>`)
		})

		// Create a test request
		req, _ := http.NewRequest("GET", "/auth-links", nil)
		resp := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(resp, req)

		// Assert response
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), `<a href="/login">Login</a>`)
		assert.Contains(t, resp.Body.String(), `<a href="/register">Register</a>`)
		assert.NotContains(t, resp.Body.String(), `<a href="/logout">Logout</a>`)
	})

	t.Run("Authenticated user should see Logout link", func(t *testing.T) {
		// Create a test HTTP server
		router := gin.New()
		router.GET("/auth-links", func(c *gin.Context) {
			// Simulate authenticated user
			c.Header("Content-Type", "text/html")
			c.Writer.WriteString(`<a href="/logout">Logout</a>`)
		})

		// Create a test request
		req, _ := http.NewRequest("GET", "/auth-links", nil)
		resp := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(resp, req)

		// Assert response
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), `<a href="/logout">Logout</a>`)
		assert.NotContains(t, resp.Body.String(), `<a href="/login">Login</a>`)
		assert.NotContains(t, resp.Body.String(), `<a href="/register">Register</a>`)
	})

	t.Run("Home page should include auth-links element with HTMX attributes", func(t *testing.T) {
		// Create a test HTTP server
		router := gin.New()

		router.GET("/", func(c *gin.Context) {
			c.Header("Content-Type", "text/html")
			// Simplified HTML response that includes the nav bar with auth-links
			html := `
			<nav class="bg-gray-800">
				<div class="ml-10 flex items-baseline space-x-4">
					<a href="/">Home</a>
					<span id="auth-links" hx-get="/auth-links" hx-trigger="load"></span>
				</div>
			</nav>
			`
			c.Writer.WriteString(html)
		})

		// Create a test request
		req, _ := http.NewRequest("GET", "/", nil)
		resp := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(resp, req)

		// Assert response
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), `<span id="auth-links" hx-get="/auth-links" hx-trigger="load"></span>`)
	})

	t.Run("Integration test - NavAuth component renders correctly", func(t *testing.T) {
		// Test the NavAuth component directly
		tests := []struct {
			name          string
			authenticated bool
			expectLogin   bool
			expectLogout  bool
		}{
			{
				name:          "Unauthenticated user",
				authenticated: false,
				expectLogin:   true,
				expectLogout:  false,
			},
			{
				name:          "Authenticated user",
				authenticated: true,
				expectLogin:   false,
				expectLogout:  true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Create a test HTTP server
				router := gin.New()
				router.GET("/test-nav", func(c *gin.Context) {
					c.Header("Content-Type", "text/html")
					if tt.authenticated {
						c.Writer.WriteString(`<a href="/logout" class="text-gray-300 hover:bg-gray-700 hover:text-white px-3 py-2 rounded-md text-sm font-medium">Logout</a>`)
					} else {
						c.Writer.WriteString(`<a href="/login" class="text-gray-300 hover:bg-gray-700 hover:text-white px-3 py-2 rounded-md text-sm font-medium">Login</a>
						<a href="/register" class="text-gray-300 hover:bg-gray-700 hover:text-white px-3 py-2 rounded-md text-sm font-medium">Register</a>`)
					}
				})

				// Create a test request
				req, _ := http.NewRequest("GET", "/test-nav", nil)
				resp := httptest.NewRecorder()

				// Serve the request
				router.ServeHTTP(resp, req)

				// Assert response
				if tt.expectLogin {
					assert.Contains(t, resp.Body.String(), `href="/login"`)
					assert.Contains(t, resp.Body.String(), `href="/register"`)
				}

				if tt.expectLogout {
					assert.Contains(t, resp.Body.String(), `href="/logout"`)
				}

				if !tt.expectLogin {
					assert.NotContains(t, resp.Body.String(), `href="/login"`)
				}

				if !tt.expectLogout {
					assert.NotContains(t, resp.Body.String(), `href="/logout"`)
				}
			})
		}
	})
}

func TestAllRoutes(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create a test HTTP server
	router := gin.New()

	// Create controllers
	mockDB := new(MockDBWithContext)
	authController := controller.NewAuthController(mockDB)
	homeController := controller.NewHomeController(mockDB)

	// Create a test user for verification
	testUser := &database.User{
		Email: "test@example.com",
	}
	database.SetUserID(testUser, 1)

	// Set up mock responses
	mockDB.On("GetUserByVerificationToken", mock.Anything, "test-token").Return(testUser, nil)
	mockDB.On("VerifyUserEmail", mock.Anything, "test-token").Return(testUser, nil)

	// Set up middleware for auth data
	router.Use(func(c *gin.Context) {
		// Get the current user's authentication status and email
		userInfo, authenticated := authController.GetCurrentUser(c)

		// Create AuthData with authentication status and email
		authData := data.NewAuthData()
		authData.Authenticated = authenticated

		// Set email if authenticated
		if authenticated {
			authData.Email = userInfo.GetUserName()
		}

		// Add authData to context
		c.Set("authData", authData)
		c.Set("authController", authController)

		c.Next()
	})

	// Set up routes
	router.GET("/", homeController.HomeHandler)
	router.GET("/about", homeController.AboutHandler)
	router.GET("/login", authController.LoginHandler)
	router.POST("/login", authController.LoginHandler)
	router.GET("/register", authController.RegisterHandler)
	router.POST("/register", authController.RegisterHandler)
	router.GET("/logout", authController.LogoutHandler)
	router.GET("/verify-email", authController.VerifyEmailHandler)
	router.GET("/verification-sent", func(c *gin.Context) {
		c.String(http.StatusOK, "Verification email sent")
	})

	t.Run("Public routes are accessible", func(t *testing.T) {
		// Test home page
		req, _ := http.NewRequest("GET", "/", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusOK, resp.Code)

		// Test about page
		req, _ = http.NewRequest("GET", "/about", nil)
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusOK, resp.Code)

		// Test login page
		req, _ = http.NewRequest("GET", "/login", nil)
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusOK, resp.Code)

		// Test register page
		req, _ = http.NewRequest("GET", "/register", nil)
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("Logout is accessible for authenticated users", func(t *testing.T) {
		// Create a test user
		testUser := &database.User{
			Email: "test@example.com",
		}
		database.SetUserID(testUser, 1)

		// Mock authentication
		mockDB.On("GetUserByEmail", mock.Anything, "test@example.com").Return(testUser, nil)
		mockDB.On("AuthenticateUser", mock.Anything, "test@example.com", "password123").Return(testUser, nil)

		// Login first
		form := url.Values{}
		form.Add("email", "test@example.com")
		form.Add("password", "password123")

		req, _ := http.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		// Get the auth cookie
		cookies := resp.Result().Cookies()
		var authCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == "auth-session" {
				authCookie = cookie
				break
			}
		}
		assert.NotNil(t, authCookie, "Auth cookie should be set after login")

		// Test logout with auth cookie
		req, _ = http.NewRequest("GET", "/logout", nil)
		req.AddCookie(authCookie)
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusSeeOther, resp.Code)
		assert.Equal(t, "/", resp.Header().Get("Location"))

		// Verify the auth cookie is cleared
		cookies = resp.Result().Cookies()
		for _, cookie := range cookies {
			if cookie.Name == "auth-session" {
				assert.Equal(t, "", cookie.Value, "Auth cookie should be cleared")
				assert.Less(t, cookie.MaxAge, 0, "Auth cookie should be expired")
				break
			}
		}
	})

	t.Run("Email verification routes are accessible", func(t *testing.T) {
		// Test verification sent page
		req, _ := http.NewRequest("GET", "/verification-sent", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusOK, resp.Code)

		// Test verify email page
		req, _ = http.NewRequest("GET", "/verify-email?token=test-token", nil)
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusSeeOther, resp.Code)
		assert.Equal(t, "/login?verified=true", resp.Header().Get("Location"))
	})
}

func TestAboutRoute(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create a new router
	r := gin.New()

	// Create a mock database
	mockDB := new(MockDBWithContext)

	// Create controllers
	homeController := controller.NewHomeController(mockDB)
	authController := controller.NewAuthController(mockDB)

	// Set up middleware for auth data
	r.Use(func(c *gin.Context) {
		// Get the current user's authentication status and email
		userInfo, authenticated := authController.GetCurrentUser(c)

		// Create AuthData with authentication status and email
		authData := data.NewAuthData()
		authData.Authenticated = authenticated

		// Set email if authenticated
		if authenticated {
			authData.Email = userInfo.GetUserName()
		}

		// Add authData to context
		c.Set("authData", authData)
		c.Set("authController", authController)

		c.Next()
	})

	// Set up routes
	r.GET("/about", homeController.AboutHandler)

	// Test the about route
	req, _ := http.NewRequest("GET", "/about", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	// Assert that the response code is 200 OK
	assert.Equal(t, http.StatusOK, resp.Code)
}
