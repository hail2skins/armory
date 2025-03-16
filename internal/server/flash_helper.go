package server

import (
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
)

// handleOwnerFlashMessage checks for a flash message cookie and adds it to the OwnerData
func handleOwnerFlashMessage(c *gin.Context, ownerData *data.OwnerData) *data.OwnerData {
	// Check for flash message
	if flashCookie, err := c.Cookie("flash"); err == nil && flashCookie != "" {
		// Add flash message to success messages
		ownerData.WithSuccess(flashCookie)
		// Clear the flash cookie
		c.SetCookie("flash", "", -1, "/", "", false, false)
	}
	return ownerData
}
