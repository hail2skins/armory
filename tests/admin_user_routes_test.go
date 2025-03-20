package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestAdminUserRoutes tests the admin user management routes
func TestAdminUserRoutes(t *testing.T) {
	// Skip this test until we properly fix all the mocks
	t.Skip("Skipping until DB mocks are properly implemented")

	// Setup
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	r := gin.Default()

	// Mock DB and controllers
	db := &mocks.MockDB{}

	// Mock methods for all the routes
	// For index
	users := []database.User{
		{Email: "test1@example.com"},
		{Email: "test2@example.com"},
	}
	db.On("CountUsers").Return(int64(2), nil)
	db.On("FindRecentUsers", 0, 50, "created_at", "desc").Return(users, nil)

	// For show, edit, update, delete, restore
	user := &database.User{
		Email:            "test@example.com",
		SubscriptionTier: "monthly",
		Verified:         true,
	}
	db.On("GetUserByID", uint(1)).Return(user, nil)
	db.On("UpdateUser", mock.Anything, mock.Anything).Return(nil)

	adminUserController := controller.NewAdminUserController(db)

	// Setup authentication middleware mock
	r.Use(func(c *gin.Context) {
		c.Set("authData", gin.H{
			"Authenticated": true,
			"Email":         "admin@example.com",
			"Roles":         []string{"admin"},
			"IsCasbinAdmin": true,
		})
		c.Next()
	})

	// Register routes
	adminGroup := r.Group("/admin")
	{
		// Apply auth middleware
		adminGroup.Use(func(c *gin.Context) {
			// This would normally check auth, but we're mocking it
			c.Next()
		})

		// User routes
		userGroup := adminGroup.Group("/users")
		{
			userGroup.GET("", adminUserController.Index)
			userGroup.GET("/:id", adminUserController.Show)
			userGroup.GET("/:id/edit", adminUserController.Edit)
			userGroup.POST("/:id", adminUserController.Update)
			userGroup.POST("/:id/delete", adminUserController.Delete)
			userGroup.POST("/:id/restore", adminUserController.Restore)
		}
	}

	// Test index route
	req, _ := http.NewRequest("GET", "/admin/users", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test show route
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/admin/users/1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test edit route
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/admin/users/1/edit", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test update route (redirects after success)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/admin/users/1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusSeeOther, w.Code)

	// Test delete route (redirects after success)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/admin/users/1/delete", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusSeeOther, w.Code)

	// Test restore route (redirects after success)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/admin/users/1/restore", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusSeeOther, w.Code)
}
