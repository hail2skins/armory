package server

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
)

// handleSessionFlash handles flash messages from the session for auth data
func handleSessionFlash(c *gin.Context, authData data.AuthData) data.AuthData {
	session := sessions.Default(c)
	flashes := session.Flashes()
	if len(flashes) > 0 {
		session.Save()
		for _, flash := range flashes {
			if flashMsg, ok := flash.(string); ok {
				authData = authData.WithSuccess(flashMsg)
			}
		}
	}
	return authData
}

// handleOwnerSessionFlash handles flash messages from the session for owner data
func handleOwnerSessionFlash(c *gin.Context, ownerData *data.OwnerData) *data.OwnerData {
	session := sessions.Default(c)
	flashes := session.Flashes()
	if len(flashes) > 0 {
		session.Save()
		for _, flash := range flashes {
			if flashMsg, ok := flash.(string); ok {
				ownerData = ownerData.WithSuccess(flashMsg)
			}
		}
	}
	return ownerData
}

// setSessionFlash sets a flash message in the session
func setSessionFlash(c *gin.Context, message string) {
	session := sessions.Default(c)
	session.AddFlash(message)
	session.Save()
}

// FlashMiddleware sets up a middleware to add a setFlash function to the context
func FlashMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set up a function to set flash messages
		c.Set("setFlash", func(message string) {
			setSessionFlash(c, message)
		})
		c.Next()
	}
}
