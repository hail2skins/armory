package controller

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
)

// HandleSessionFlash processes flash messages from the session for AuthData
func HandleSessionFlash(c *gin.Context, authData data.AuthData) data.AuthData {
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

// HandleSessionFlashForOwner processes flash messages from the session for OwnerData
func HandleSessionFlashForOwner(c *gin.Context, ownerData *data.OwnerData) *data.OwnerData {
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

// SetSessionFlash adds a flash message to the session
func SetSessionFlash(c *gin.Context, message string) {
	session := sessions.Default(c)
	session.AddFlash(message)
	session.Save()
}
