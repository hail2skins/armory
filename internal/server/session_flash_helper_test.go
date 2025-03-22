package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/stretchr/testify/assert"
)

func TestHandleSessionFlash(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a new router with the session middleware
	r := gin.New()

	// Initialize the cookie store for sessions
	store := cookie.NewStore([]byte("secret"))
	r.Use(sessions.Sessions("armory-session", store))

	// Create a test route that sets a flash message and then processes it
	r.GET("/test-session-flash", func(c *gin.Context) {
		// Set a flash message in the session
		session := sessions.Default(c)
		session.AddFlash("Test flash message")
		session.Save()

		// Create a basic AuthData and process the flash
		authData := data.NewAuthData()
		authData = handleSessionFlash(c, authData)

		// Verify flash was processed correctly
		assert.Equal(t, "Test flash message", authData.Success)

		// Flashes should be cleared after processing
		flashes := session.Flashes()
		session.Save()
		assert.Equal(t, 0, len(flashes))

		c.Status(http.StatusOK)
	})

	// Test the route
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test-session-flash", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleOwnerSessionFlash(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a new router with the session middleware
	r := gin.New()

	// Initialize the cookie store for sessions
	store := cookie.NewStore([]byte("secret"))
	r.Use(sessions.Sessions("armory-session", store))

	// Create a test route that sets a flash message and then processes it
	r.GET("/test-owner-session-flash", func(c *gin.Context) {
		// Set a flash message in the session
		session := sessions.Default(c)
		session.AddFlash("Test owner flash message")
		session.Save()

		// Create a basic OwnerData and process the flash
		ownerData := data.NewOwnerData()
		ownerData = handleOwnerSessionFlash(c, ownerData)

		// Verify flash was processed correctly
		assert.Equal(t, "Test owner flash message", ownerData.Auth.Success)

		// Flashes should be cleared after processing
		flashes := session.Flashes()
		session.Save()
		assert.Equal(t, 0, len(flashes))

		c.Status(http.StatusOK)
	})

	// Test the route
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test-owner-session-flash", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSetSessionFlash(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a new router with the session middleware
	r := gin.New()

	// Initialize the cookie store for sessions
	store := cookie.NewStore([]byte("secret"))
	r.Use(sessions.Sessions("armory-session", store))

	// Create a test route that sets a flash message
	r.GET("/test-set-session-flash", func(c *gin.Context) {
		// Set a flash message
		setSessionFlash(c, "Test set flash message")

		// Verify flash was set correctly
		session := sessions.Default(c)
		flashes := session.Flashes()
		session.Save()

		assert.Equal(t, 1, len(flashes))
		if len(flashes) > 0 {
			assert.Equal(t, "Test set flash message", flashes[0])
		}

		c.Status(http.StatusOK)
	})

	// Test the route
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test-set-session-flash", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFlashMiddleware(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a new router with the session middleware
	r := gin.New()

	// Initialize the cookie store for sessions
	store := cookie.NewStore([]byte("secret"))
	r.Use(sessions.Sessions("armory-session", store))

	// Add the flash middleware
	r.Use(FlashMiddleware())

	// Create a test route that uses the setFlash function
	r.GET("/test-flash-middleware", func(c *gin.Context) {
		// Get the setFlash function
		setFlash, exists := c.Get("setFlash")
		assert.True(t, exists)

		// Use the function to set a flash
		if flashFunc, ok := setFlash.(func(string)); ok {
			flashFunc("Middleware flash message")
		}

		// Verify the flash was set
		session := sessions.Default(c)
		flashes := session.Flashes()
		session.Save()

		assert.Equal(t, 1, len(flashes))
		if len(flashes) > 0 {
			assert.Equal(t, "Middleware flash message", flashes[0])
		}

		c.Status(http.StatusOK)
	})

	// Test the route
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test-flash-middleware", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
