package server

import (
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
)

// handleOwnerFlashMessage checks for flash messages in the session and adds them to the OwnerData
func handleOwnerFlashMessage(c *gin.Context, ownerData *data.OwnerData) *data.OwnerData {
	// Use the new session-based flash helper
	return handleOwnerSessionFlash(c, ownerData)
}
