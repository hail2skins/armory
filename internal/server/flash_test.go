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

func TestSetFlashMessage(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a new router with the session middleware
	r := gin.New()

	// Initialize the cookie store for sessions
	store := cookie.NewStore([]byte("secret"))
	r.Use(sessions.Sessions("armory-session", store))

	// Set up the flash middleware
	r.Use(func(c *gin.Context) {
		// Set up the setFlash function in the context
		c.Set("setFlash", func(message string) {
			session := sessions.Default(c)
			session.AddFlash(message)
			session.Save()
		})
		c.Next()
	})

	// Create a test route that sets a flash message
	r.GET("/test-flash", func(c *gin.Context) {
		setFlash, exists := c.Get("setFlash")
		assert.True(t, exists)

		if flashFunc, ok := setFlash.(func(string)); ok {
			flashFunc("Test flash message")
		}

		c.Status(http.StatusOK)
	})

	// Create a test route that retrieves and clears the flash message
	r.GET("/get-flash", func(c *gin.Context) {
		session := sessions.Default(c)
		flashes := session.Flashes()
		session.Save()

		assert.Equal(t, 1, len(flashes))
		if len(flashes) > 0 {
			assert.Equal(t, "Test flash message", flashes[0])
		}

		c.Status(http.StatusOK)
	})

	// Test setting a flash message
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/test-flash", nil)
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Extract the session cookie
	cookies := w1.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "armory-session" {
			sessionCookie = cookie
			break
		}
	}
	assert.NotNil(t, sessionCookie)

	// Test retrieving the flash message
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/get-flash", nil)
	req2.AddCookie(sessionCookie)
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestHandleFlashMessage(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a new router with the session middleware
	r := gin.New()

	// Initialize the cookie store for sessions
	store := cookie.NewStore([]byte("secret"))
	r.Use(sessions.Sessions("armory-session", store))

	// Set up the flash middleware
	r.Use(func(c *gin.Context) {
		// Set up the setFlash function in the context
		c.Set("setFlash", func(message string) {
			session := sessions.Default(c)
			session.AddFlash(message)
			session.Save()
		})
		c.Next()
	})

	// Create a test route that sets a flash message
	r.GET("/set-flash", func(c *gin.Context) {
		setFlash, exists := c.Get("setFlash")
		assert.True(t, exists)

		if flashFunc, ok := setFlash.(func(string)); ok {
			flashFunc("Test flash message")
		}

		c.Status(http.StatusOK)
	})

	// Create a test route that processes flash messages into data objects
	r.GET("/process-flash", func(c *gin.Context) {
		// Create a basic data object
		ownerData := data.NewOwnerData()

		// Process flash messages
		session := sessions.Default(c)
		flashes := session.Flashes()
		session.Save()

		// Add all flash messages to the data object
		for _, flash := range flashes {
			if flashMsg, ok := flash.(string); ok {
				ownerData = ownerData.WithSuccess(flashMsg)
			}
		}

		// Verify flash was processed correctly
		assert.Equal(t, "Test flash message", ownerData.Auth.Success)

		c.Status(http.StatusOK)
	})

	// Test setting a flash message
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/set-flash", nil)
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Extract the session cookie
	cookies := w1.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "armory-session" {
			sessionCookie = cookie
			break
		}
	}
	assert.NotNil(t, sessionCookie)

	// Test processing the flash message
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/process-flash", nil)
	req2.AddCookie(sessionCookie)
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}
