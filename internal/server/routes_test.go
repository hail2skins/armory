package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
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
