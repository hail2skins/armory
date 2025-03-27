package controller_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAdminFeatureFlagsController(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	t.Run("Index lists all feature flags", func(t *testing.T) {
		// Create a mock DB
		mockDB := new(mocks.MockDB)

		// Setup expectations
		mockFlags := []models.FeatureFlag{
			{ID: 1, Name: "test_feature", Enabled: true, Description: "Test feature"},
			{ID: 2, Name: "new_feature", Enabled: false, Description: "New feature"},
		}
		mockDB.On("FindAllFeatureFlags").Return(mockFlags, nil)

		// Create router with session middleware
		router := gin.New()
		store := cookie.NewStore([]byte("secret"))
		router.Use(sessions.Sessions("armory_session", store))

		// Create controller
		controller := controller.NewAdminFeatureFlagsController(mockDB)

		// Add route
		router.GET("/admin/permissions/feature-flags", func(c *gin.Context) {
			// Create auth data for the request
			authData := data.NewAuthData()
			authData.Authenticated = true
			authData.Email = "test@example.com"
			authData.CSRFToken = "test-csrf-token"
			c.Set("authData", authData)

			// Call the controller
			controller.Index(c)
		})

		// Create a request
		req, _ := http.NewRequest("GET", "/admin/permissions/feature-flags", nil)
		resp := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(resp, req)

		// Assert
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), "test_feature")
		assert.Contains(t, resp.Body.String(), "new_feature")
		mockDB.AssertExpectations(t)
	})

	t.Run("Create displays new feature flag form", func(t *testing.T) {
		// Create a mock DB
		mockDB := new(mocks.MockDB)

		// Setup expectations for roles
		roles := []string{"admin", "editor", "viewer"}

		// Don't expect the GetDB call since we're providing roles through context
		// mockDB.On("GetDB").Return(nil)

		// Create router with session middleware
		router := gin.New()
		store := cookie.NewStore([]byte("secret"))
		router.Use(sessions.Sessions("armory_session", store))

		// Create controller
		controller := controller.NewAdminFeatureFlagsController(mockDB)

		// Add route
		router.GET("/admin/permissions/feature-flags/create", func(c *gin.Context) {
			// Create auth data for the request
			authData := data.NewAuthData()
			authData.Authenticated = true
			authData.Email = "test@example.com"
			authData.CSRFToken = "test-csrf-token"
			c.Set("authData", authData)

			// Add roles to context
			c.Set("available_roles", roles)

			// Call the controller
			controller.Create(c)
		})

		// Create a request
		req, _ := http.NewRequest("GET", "/admin/permissions/feature-flags/create", nil)
		resp := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(resp, req)

		// Assert
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), "Create Feature Flag")
		assert.Contains(t, resp.Body.String(), "test-csrf-token")
		mockDB.AssertExpectations(t)
	})

	t.Run("Store creates a new feature flag", func(t *testing.T) {
		// Create a mock DB
		mockDB := new(mocks.MockDB)

		// Setup expectations
		mockDB.On("CreateFeatureFlag", mock.AnythingOfType("*models.FeatureFlag")).Return(nil)

		// Create router with session middleware
		router := gin.New()
		store := cookie.NewStore([]byte("secret"))
		router.Use(sessions.Sessions("armory_session", store))

		// Create controller
		controller := controller.NewAdminFeatureFlagsController(mockDB)

		// Add route
		router.POST("/admin/permissions/feature-flags/create", func(c *gin.Context) {
			// Create auth data for the request
			authData := data.NewAuthData()
			authData.Authenticated = true
			authData.Email = "test@example.com"
			authData.CSRFToken = "test-csrf-token"
			c.Set("authData", authData)

			// Call the controller
			controller.Store(c)
		})

		// Create a request with form data
		form := url.Values{}
		form.Add("name", "new_feature")
		form.Add("description", "A new feature")
		form.Add("enabled", "true")
		form.Add("csrf_token", "test-csrf-token")
		req, _ := http.NewRequest("POST", "/admin/permissions/feature-flags/create", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(resp, req)

		// Assert
		assert.Equal(t, http.StatusFound, resp.Code)
		assert.Equal(t, "/admin/permissions/feature-flags", resp.Header().Get("Location"))
		mockDB.AssertExpectations(t)
	})

	t.Run("Edit displays feature flag edit form", func(t *testing.T) {
		// Create a mock DB
		mockDB := new(mocks.MockDB)

		// Setup expectations
		mockFlag := &models.FeatureFlag{
			ID:          1,
			Name:        "test_feature",
			Enabled:     true,
			Description: "Test feature",
			Roles:       []models.FeatureFlagRole{{ID: 1, FeatureFlagID: 1, Role: "admin"}},
		}
		mockDB.On("FindFeatureFlagByID", uint(1)).Return(mockFlag, nil)

		// Don't expect the GetDB call since we're providing roles through context
		// mockDB.On("GetDB").Return(nil)

		// Create router with session middleware
		router := gin.New()
		store := cookie.NewStore([]byte("secret"))
		router.Use(sessions.Sessions("armory_session", store))

		// Create controller
		controller := controller.NewAdminFeatureFlagsController(mockDB)

		// Add route
		router.GET("/admin/permissions/feature-flags/edit/:id", func(c *gin.Context) {
			// Create auth data for the request
			authData := data.NewAuthData()
			authData.Authenticated = true
			authData.Email = "test@example.com"
			authData.CSRFToken = "test-csrf-token"
			c.Set("authData", authData)

			// Add roles to context
			c.Set("available_roles", []string{"admin", "editor", "viewer"})

			// Call the controller
			controller.Edit(c)
		})

		// Create a request
		req, _ := http.NewRequest("GET", "/admin/permissions/feature-flags/edit/1", nil)
		resp := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(resp, req)

		// Assert
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), "Edit Feature Flag")
		assert.Contains(t, resp.Body.String(), "test_feature")
		mockDB.AssertExpectations(t)
	})

	t.Run("Update updates a feature flag", func(t *testing.T) {
		// Create a mock DB
		mockDB := new(mocks.MockDB)

		// Setup expectations
		mockFlag := &models.FeatureFlag{
			ID:          1,
			Name:        "test_feature",
			Enabled:     true,
			Description: "Test feature",
		}
		mockDB.On("FindFeatureFlagByID", uint(1)).Return(mockFlag, nil)
		mockDB.On("UpdateFeatureFlag", mock.AnythingOfType("*models.FeatureFlag")).Return(nil)

		// Create router with session middleware
		router := gin.New()
		store := cookie.NewStore([]byte("secret"))
		router.Use(sessions.Sessions("armory_session", store))

		// Create controller
		controller := controller.NewAdminFeatureFlagsController(mockDB)

		// Add route
		router.POST("/admin/permissions/feature-flags/edit/:id", func(c *gin.Context) {
			// Create auth data for the request
			authData := data.NewAuthData()
			authData.Authenticated = true
			authData.Email = "test@example.com"
			authData.CSRFToken = "test-csrf-token"
			c.Set("authData", authData)

			// Call the controller
			controller.Update(c)
		})

		// Create a request with form data
		form := url.Values{}
		form.Add("name", "updated_feature")
		form.Add("description", "Updated feature")
		form.Add("enabled", "false")
		form.Add("csrf_token", "test-csrf-token")
		req, _ := http.NewRequest("POST", "/admin/permissions/feature-flags/edit/1", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(resp, req)

		// Assert
		assert.Equal(t, http.StatusFound, resp.Code)
		assert.Equal(t, "/admin/permissions/feature-flags", resp.Header().Get("Location"))
		mockDB.AssertExpectations(t)
	})

	t.Run("Delete removes a feature flag", func(t *testing.T) {
		// Create a mock DB
		mockDB := new(mocks.MockDB)

		// Setup expectations
		mockDB.On("DeleteFeatureFlag", uint(1)).Return(nil)

		// Create router with session middleware
		router := gin.New()
		store := cookie.NewStore([]byte("secret"))
		router.Use(sessions.Sessions("armory_session", store))

		// Create controller
		controller := controller.NewAdminFeatureFlagsController(mockDB)

		// Add route
		router.POST("/admin/permissions/feature-flags/delete/:id", func(c *gin.Context) {
			// Create auth data for the request
			authData := data.NewAuthData()
			authData.Authenticated = true
			authData.Email = "test@example.com"
			authData.CSRFToken = "test-csrf-token"
			c.Set("authData", authData)

			// Call the controller
			controller.Delete(c)
		})

		// Create a request with form data
		form := url.Values{}
		form.Add("csrf_token", "test-csrf-token")
		req, _ := http.NewRequest("POST", "/admin/permissions/feature-flags/delete/1", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(resp, req)

		// Assert
		assert.Equal(t, http.StatusFound, resp.Code)
		assert.Equal(t, "/admin/permissions/feature-flags", resp.Header().Get("Location"))
		mockDB.AssertExpectations(t)
	})

	t.Run("AddRole adds a role to a feature flag", func(t *testing.T) {
		// Create a mock DB
		mockDB := new(mocks.MockDB)

		// Setup expectations
		mockDB.On("AddRoleToFeatureFlag", uint(1), "editor").Return(nil)

		// Create router with session middleware
		router := gin.New()
		store := cookie.NewStore([]byte("secret"))
		router.Use(sessions.Sessions("armory_session", store))

		// Create controller
		controller := controller.NewAdminFeatureFlagsController(mockDB)

		// Add route
		router.POST("/admin/permissions/feature-flags/:id/roles", func(c *gin.Context) {
			// Create auth data for the request
			authData := data.NewAuthData()
			authData.Authenticated = true
			authData.Email = "test@example.com"
			authData.CSRFToken = "test-csrf-token"
			c.Set("authData", authData)

			// Call the controller
			controller.AddRole(c)
		})

		// Create a request with form data
		form := url.Values{}
		form.Add("role", "editor")
		form.Add("csrf_token", "test-csrf-token")
		req, _ := http.NewRequest("POST", "/admin/permissions/feature-flags/1/roles", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(resp, req)

		// Assert
		assert.Equal(t, http.StatusFound, resp.Code)
		assert.Equal(t, "/admin/permissions/feature-flags/edit/1", resp.Header().Get("Location"))
		mockDB.AssertExpectations(t)
	})

	t.Run("RemoveRole removes a role from a feature flag", func(t *testing.T) {
		// Create a mock DB
		mockDB := new(mocks.MockDB)

		// Setup expectations
		mockDB.On("RemoveRoleFromFeatureFlag", uint(1), "editor").Return(nil)

		// Create router with session middleware
		router := gin.New()
		store := cookie.NewStore([]byte("secret"))
		router.Use(sessions.Sessions("armory_session", store))

		// Create controller
		controller := controller.NewAdminFeatureFlagsController(mockDB)

		// Add route
		router.POST("/admin/permissions/feature-flags/:id/roles/remove", func(c *gin.Context) {
			// Create auth data for the request
			authData := data.NewAuthData()
			authData.Authenticated = true
			authData.Email = "test@example.com"
			authData.CSRFToken = "test-csrf-token"
			c.Set("authData", authData)

			// Call the controller
			controller.RemoveRole(c)
		})

		// Create a request with form data
		form := url.Values{}
		form.Add("role", "editor")
		form.Add("csrf_token", "test-csrf-token")
		req, _ := http.NewRequest("POST", "/admin/permissions/feature-flags/1/roles/remove", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(resp, req)

		// Assert
		assert.Equal(t, http.StatusFound, resp.Code)
		assert.Equal(t, "/admin/permissions/feature-flags/edit/1", resp.Header().Get("Location"))
		mockDB.AssertExpectations(t)
	})
}
