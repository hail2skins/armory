package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/shaj13/go-guardian/v2/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CustomAuthInfo implements auth.Info for testing
type CustomAuthInfo struct {
	username   string
	id         string
	groups     []string
	extensions auth.Extensions
}

func (i *CustomAuthInfo) GetUserName() string {
	return i.username
}

func (i *CustomAuthInfo) GetID() string {
	return i.id
}

func (i *CustomAuthInfo) GetGroups() []string {
	return i.groups
}

func (i *CustomAuthInfo) GetExtensions() auth.Extensions {
	return i.extensions
}

func (i *CustomAuthInfo) SetExtensions(ext auth.Extensions) {
	i.extensions = ext
}

func (i *CustomAuthInfo) SetGroups(groups []string) {
	i.groups = groups
}

func (i *CustomAuthInfo) SetID(id string) {
	i.id = id
}

func (i *CustomAuthInfo) SetUserName(username string) {
	i.username = username
}

// setupCasbinAuthTest creates a router with a Casbin middleware for testing
func setupCasbinAuthTest(t *testing.T) (*gin.Engine, *CasbinAuth) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Create a casbin auth middleware
	casbinAuth, err := NewCasbinAuth("testdata/rbac_model.conf", "testdata/rbac_policy.csv")
	require.NoError(t, err)

	return r, casbinAuth
}

func TestCasbinAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a casbin auth middleware
	casbinAuth, err := NewCasbinAuth("testdata/rbac_model.conf", "testdata/rbac_policy.csv")
	require.NoError(t, err)

	// Test cases
	tests := []struct {
		name           string
		authInfo       auth.Info
		wantStatus     int
		wantRedirect   bool
		wantRedirectTo string
		wantFlash      bool
	}{
		{
			name: "Admin user can access admin routes",
			authInfo: &CustomAuthInfo{
				username: "admin@example.com",
				id:       "1",
			},
			wantStatus:   http.StatusOK,
			wantRedirect: false,
		},
		{
			name: "Regular user cannot access admin routes",
			authInfo: &CustomAuthInfo{
				username: "user@example.com",
				id:       "2",
			},
			wantStatus:     http.StatusSeeOther,
			wantRedirect:   true,
			wantRedirectTo: "/",
			wantFlash:      true,
		},
		{
			name:           "Unauthenticated user cannot access admin routes",
			authInfo:       nil,
			wantStatus:     http.StatusSeeOther,
			wantRedirect:   true,
			wantRedirectTo: "/",
			wantFlash:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new router for each test
			router := gin.New()

			// Create a test handler
			testHandler := func(c *gin.Context) {
				c.String(http.StatusOK, "Admin access granted")
			}

			// Set up a recorder to track if flash message was set
			flashMessageSet := false
			flashMessage := ""

			// Setup flash message middleware
			router.Use(func(c *gin.Context) {
				c.Set("setFlash", func(message string) {
					flashMessageSet = true
					flashMessage = message
					c.Set("flash_message", message)
				})
				c.Next()
			})

			// Add the authentication middleware (simulate auth_info being set)
			authMiddleware := func(c *gin.Context) {
				if tt.authInfo != nil {
					c.Set("auth_info", tt.authInfo)
					c.Next()
				} else {
					// Skip this and let the Casbin middleware handle it
					c.Next()
				}
			}

			// Set up admin route with auth middleware and Casbin
			router.GET("/admin/test", authMiddleware, casbinAuth.Authorize("admin"), testHandler)

			// Create a new test recorder and request
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/admin/test", nil)

			// Process the request
			router.ServeHTTP(w, req)

			// Check the status code
			assert.Equal(t, tt.wantStatus, w.Code, "Status code should match for %s", tt.name)

			// Check for redirects
			if tt.wantRedirect {
				location := w.Header().Get("Location")
				assert.Equal(t, tt.wantRedirectTo, location, "Should redirect to the correct location")

				// Check if a flash message was set
				if tt.wantFlash {
					assert.True(t, flashMessageSet, "Flash message should have been set")
					assert.NotEmpty(t, flashMessage, "Flash message should not be empty")
				}
			} else {
				// If no redirect, check the content
				assert.Contains(t, w.Body.String(), "Admin access granted", "Should contain expected content")
			}
		})
	}
}

func TestCasbinAuthCustomRules(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a casbin auth middleware
	casbinAuth, err := NewCasbinAuth("testdata/rbac_model.conf", "testdata/rbac_policy.csv")
	require.NoError(t, err)

	// Define handlers
	readHandler := func(c *gin.Context) {
		c.String(http.StatusOK, "Read manufacturers")
	}
	writeHandler := func(c *gin.Context) {
		c.String(http.StatusOK, "Create manufacturer")
	}
	updateHandler := func(c *gin.Context) {
		c.String(http.StatusOK, "Update manufacturer")
	}
	deleteHandler := func(c *gin.Context) {
		c.String(http.StatusOK, "Delete manufacturer")
	}

	// Test cases
	tests := []struct {
		name           string
		url            string
		method         string
		userEmail      string
		wantStatus     int
		wantRedirect   bool
		wantRedirectTo string
	}{
		{
			name:       "Admin can read manufacturers",
			url:        "/admin/manufacturers",
			method:     "GET",
			userEmail:  "admin@example.com",
			wantStatus: http.StatusOK,
		},
		{
			name:       "Editor can read manufacturers",
			url:        "/admin/manufacturers",
			method:     "GET",
			userEmail:  "editor@example.com",
			wantStatus: http.StatusOK,
		},
		{
			name:       "Editor can write manufacturers",
			url:        "/admin/manufacturers",
			method:     "POST",
			userEmail:  "editor@example.com",
			wantStatus: http.StatusOK,
		},
		{
			name:           "Editor cannot delete manufacturers",
			url:            "/admin/manufacturers/1/delete",
			method:         "POST",
			userEmail:      "editor@example.com",
			wantStatus:     http.StatusSeeOther,
			wantRedirect:   true,
			wantRedirectTo: "/",
		},
		{
			name:           "Viewer can only read",
			url:            "/admin/manufacturers/1/edit",
			method:         "GET",
			userEmail:      "viewer@example.com",
			wantStatus:     http.StatusSeeOther,
			wantRedirect:   true,
			wantRedirectTo: "/",
		},
	}

	// Test cases for TestCasbinAuthCustomRules
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new router for each test
			router := gin.New()

			// Set up a recorder to track if flash message was set
			flashMessageSet := false
			flashMessage := ""

			// Setup flash message middleware
			router.Use(func(c *gin.Context) {
				c.Set("setFlash", func(message string) {
					flashMessageSet = true
					flashMessage = message
					c.Set("flash_message", message)
				})
				c.Next()
			})

			// Create auth middleware
			authMiddleware := func(c *gin.Context) {
				c.Set("auth_info", &CustomAuthInfo{
					username: tt.userEmail,
					id:       "test-id",
				})
				c.Next()
			}

			// Register routes with appropriate middleware
			manufacturerGroup := router.Group("/admin/manufacturers")
			{
				manufacturerGroup.GET("", authMiddleware, casbinAuth.Authorize("manufacturers", "read"), readHandler)
				manufacturerGroup.POST("", authMiddleware, casbinAuth.Authorize("manufacturers", "write"), writeHandler)
				manufacturerGroup.GET("/:id/edit", authMiddleware, casbinAuth.Authorize("manufacturers", "update"), updateHandler)
				manufacturerGroup.POST("/:id/delete", authMiddleware, casbinAuth.Authorize("manufacturers", "delete"), deleteHandler)
			}

			// Create a new test recorder and request
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(tt.method, tt.url, nil)

			// Process the request
			router.ServeHTTP(w, req)

			// Check status code
			assert.Equal(t, tt.wantStatus, w.Code, "Status code should match for %s", tt.name)

			// Check redirect if expected
			if tt.wantRedirect {
				location := w.Header().Get("Location")
				assert.Equal(t, tt.wantRedirectTo, location, "Should redirect to the correct location")

				// Check if flash message was set
				assert.True(t, flashMessageSet, "Flash message should have been set")
				assert.NotEmpty(t, flashMessage, "Flash message should not be empty")
			}
		})
	}
}
