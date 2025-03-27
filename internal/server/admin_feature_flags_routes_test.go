package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/middleware"
	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestFeatureFlagsAuthController implements the auth controller interface for tests
type TestFeatureFlagsAuthController struct {
	authenticated bool
	email         string
	roles         []string
	isAdmin       bool
}

// IsAuthenticated returns whether the user is authenticated
func (m *TestFeatureFlagsAuthController) IsAuthenticated(c *gin.Context) bool {
	return m.authenticated
}

// GetCurrentUser returns the current user
func (m *TestFeatureFlagsAuthController) GetCurrentUser(c *gin.Context) (interface{}, bool) {
	if !m.authenticated {
		return nil, false
	}
	return m, true
}

// GetUserName returns the email
func (m *TestFeatureFlagsAuthController) GetUserName() string {
	return m.email
}

// GetGroups returns the roles
func (m *TestFeatureFlagsAuthController) GetGroups() []string {
	return m.roles
}

// IsAdmin returns admin status
func (m *TestFeatureFlagsAuthController) IsAdmin(c *gin.Context) bool {
	return m.isAdmin
}

// TestCasbinAuth provides test implementation of Casbin authorize
type TestCasbinAuth struct {
	roles []string
}

// FlexibleAuthorize checks permissions based on role and requested resource/action
func (m *TestCasbinAuth) FlexibleAuthorize(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the roles
		roles := m.roles

		// Default to forbidden
		hasPermission := false

		// Check if the user has permission based on role
		for _, role := range roles {
			// Admin role has all permissions
			if role == "admin" {
				hasPermission = true
				break
			}

			// No other roles have permissions to admin features
		}

		if !hasPermission {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		c.Next()
	}
}

// TestFeatureFlagRoutesWithDifferentUsers tests the feature flag routes with different user types
func TestFeatureFlagRoutesWithDifferentUsers(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Setup user types
	userTypes := []struct {
		name          string
		auth          TestFeatureFlagsAuthController
		expectedCodes map[string]int
	}{
		{
			name: "guest",
			auth: TestFeatureFlagsAuthController{
				authenticated: false,
				email:         "",
				roles:         []string{},
				isAdmin:       false,
			},
			expectedCodes: map[string]int{
				"index":       http.StatusFound, // Redirect to login
				"create":      http.StatusFound, // Redirect to login
				"store":       http.StatusFound, // Redirect to login
				"edit":        http.StatusFound, // Redirect to login
				"update":      http.StatusFound, // Redirect to login
				"delete":      http.StatusFound, // Redirect to login
				"add_role":    http.StatusFound, // Redirect to login
				"remove_role": http.StatusFound, // Redirect to login
			},
		},
		{
			name: "normal_user",
			auth: TestFeatureFlagsAuthController{
				authenticated: true,
				email:         "user@example.com",
				roles:         []string{"viewer"},
				isAdmin:       false,
			},
			expectedCodes: map[string]int{
				"index":       http.StatusForbidden, // No permission
				"create":      http.StatusForbidden, // No permission
				"store":       http.StatusForbidden, // No permission
				"edit":        http.StatusForbidden, // No permission
				"update":      http.StatusForbidden, // No permission
				"delete":      http.StatusForbidden, // No permission
				"add_role":    http.StatusForbidden, // No permission
				"remove_role": http.StatusForbidden, // No permission
			},
		},
		{
			name: "admin_user",
			auth: TestFeatureFlagsAuthController{
				authenticated: true,
				email:         "admin@example.com",
				roles:         []string{"admin"},
				isAdmin:       true,
			},
			expectedCodes: map[string]int{
				"index":       http.StatusOK,    // Has all permissions
				"create":      http.StatusOK,    // Has all permissions
				"store":       http.StatusFound, // Redirects after form submission
				"edit":        http.StatusOK,    // Has all permissions
				"update":      http.StatusFound, // Redirects after form submission
				"delete":      http.StatusFound, // Redirects after form submission
				"add_role":    http.StatusFound, // Redirects after form submission
				"remove_role": http.StatusFound, // Redirects after form submission
			},
		},
	}

	// Run tests for each user type
	for _, userType := range userTypes {
		t.Run(userType.name, func(t *testing.T) {
			// Create a new router for each test
			router := gin.New()

			// Create mock DB
			mockDB := new(mocks.MockDB)

			// Setup mock methods
			mockDB.On("FindAllFeatureFlags").Return([]models.FeatureFlag{}, nil)
			mockDB.On("FindFeatureFlagByID", uint(1)).Return(&models.FeatureFlag{ID: 1, Name: "test_flag"}, nil)
			mockDB.On("CreateFeatureFlag", mock.Anything).Return(nil)
			mockDB.On("UpdateFeatureFlag", mock.Anything).Return(nil)
			mockDB.On("DeleteFeatureFlag", uint(1)).Return(nil)
			mockDB.On("AddRoleToFeatureFlag", uint(1), "admin").Return(nil)
			mockDB.On("RemoveRoleFromFeatureFlag", uint(1), "admin").Return(nil)
			mockDB.On("FindAllRoles").Return([]string{"admin", "editor"}, nil)
			mockDB.On("GetDB").Return(nil)

			// Create controller
			adminFeatureFlagsController := controller.NewAdminFeatureFlagsController(mockDB)

			// Setup Casbin auth
			casbinAuth := &TestCasbinAuth{
				roles: userType.auth.roles,
			}

			// Enable test mode for CSRF
			middleware.EnableTestMode()

			// Setup session
			store := cookie.NewStore([]byte("test-secret"))
			router.Use(sessions.Sessions("armory-session", store))

			// Setup authentication middleware
			router.Use(func(c *gin.Context) {
				// Set test CSRF token
				c.Set("csrf_token", "test-csrf-token")

				// Set auth in context
				c.Set("auth", &userType.auth)
				c.Set("authController", &userType.auth)

				c.Next()
			})

			// Setup admin group with auth check
			adminGroup := router.Group("/admin")
			adminGroup.Use(func(c *gin.Context) {
				if !userType.auth.authenticated {
					c.Redirect(http.StatusFound, "/login")
					c.Abort()
					return
				}
				c.Next()
			})

			// Register routes
			permissionsGroup := adminGroup.Group("/permissions")

			// Setup routes based on user type
			featureFlagsGroup := permissionsGroup.Group("/feature-flags")

			if userType.auth.isAdmin {
				// Admin access
				featureFlagsGroup.GET("", adminFeatureFlagsController.Index)
				featureFlagsGroup.GET("/create", adminFeatureFlagsController.Create)
				featureFlagsGroup.POST("/create", adminFeatureFlagsController.Store)
				featureFlagsGroup.GET("/edit/:id", adminFeatureFlagsController.Edit)
				featureFlagsGroup.POST("/edit/:id", adminFeatureFlagsController.Update)
				featureFlagsGroup.POST("/delete/:id", adminFeatureFlagsController.Delete)
				featureFlagsGroup.POST("/:id/roles", adminFeatureFlagsController.AddRole)
				featureFlagsGroup.POST("/:id/roles/remove", adminFeatureFlagsController.RemoveRole)
			} else {
				// Non-admin permissions with Casbin
				featureFlagsGroup.GET("", casbinAuth.FlexibleAuthorize("feature_flags", "read"), adminFeatureFlagsController.Index)
				featureFlagsGroup.GET("/create", casbinAuth.FlexibleAuthorize("feature_flags", "write"), adminFeatureFlagsController.Create)
				featureFlagsGroup.POST("/create", casbinAuth.FlexibleAuthorize("feature_flags", "write"), adminFeatureFlagsController.Store)
				featureFlagsGroup.GET("/edit/:id", casbinAuth.FlexibleAuthorize("feature_flags", "update"), adminFeatureFlagsController.Edit)
				featureFlagsGroup.POST("/edit/:id", casbinAuth.FlexibleAuthorize("feature_flags", "update"), adminFeatureFlagsController.Update)
				featureFlagsGroup.POST("/delete/:id", casbinAuth.FlexibleAuthorize("feature_flags", "delete"), adminFeatureFlagsController.Delete)
				featureFlagsGroup.POST("/:id/roles", casbinAuth.FlexibleAuthorize("feature_flags", "update"), adminFeatureFlagsController.AddRole)
				featureFlagsGroup.POST("/:id/roles/remove", casbinAuth.FlexibleAuthorize("feature_flags", "update"), adminFeatureFlagsController.RemoveRole)
			}

			// Setup test routes
			testRoutes := map[string]struct {
				method string
				path   string
				form   url.Values
			}{
				"index": {
					method: http.MethodGet,
					path:   "/admin/permissions/feature-flags",
				},
				"create": {
					method: http.MethodGet,
					path:   "/admin/permissions/feature-flags/create",
				},
				"store": {
					method: http.MethodPost,
					path:   "/admin/permissions/feature-flags/create",
					form: url.Values{
						"csrf_token":  {"test-csrf-token"},
						"name":        {"test_flag"},
						"description": {"Test flag"},
						"enabled":     {"true"},
					},
				},
				"edit": {
					method: http.MethodGet,
					path:   "/admin/permissions/feature-flags/edit/1",
				},
				"update": {
					method: http.MethodPost,
					path:   "/admin/permissions/feature-flags/edit/1",
					form: url.Values{
						"csrf_token":  {"test-csrf-token"},
						"name":        {"test_flag"},
						"description": {"Updated test flag"},
						"enabled":     {"true"},
					},
				},
				"delete": {
					method: http.MethodPost,
					path:   "/admin/permissions/feature-flags/delete/1",
					form: url.Values{
						"csrf_token": {"test-csrf-token"},
					},
				},
				"add_role": {
					method: http.MethodPost,
					path:   "/admin/permissions/feature-flags/1/roles",
					form: url.Values{
						"csrf_token": {"test-csrf-token"},
						"role":       {"admin"},
					},
				},
				"remove_role": {
					method: http.MethodPost,
					path:   "/admin/permissions/feature-flags/1/roles/remove",
					form: url.Values{
						"csrf_token": {"test-csrf-token"},
						"role":       {"admin"},
					},
				},
			}

			// Test each route
			for name, route := range testRoutes {
				t.Run(name, func(t *testing.T) {
					var req *http.Request

					if route.method == http.MethodGet {
						req, _ = http.NewRequest(route.method, route.path, nil)
					} else {
						// For POST requests, handle form properly
						formData := strings.NewReader(route.form.Encode())
						req, _ = http.NewRequest(route.method, route.path, formData)
						req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
					}

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					expectedStatus := userType.expectedCodes[name]
					assert.Equal(t, expectedStatus, w.Code,
						"Expected status %d for %s user on route %s, got %d",
						expectedStatus, userType.name, route.path, w.Code)
				})
			}

			// Disable test mode after tests
			middleware.DisableTestMode()
		})
	}
}
