package data

import (
	"github.com/gin-gonic/gin"
)

// ViewData is a generic container for view data
type ViewData struct {
	AuthData
	Data     interface{}
	ErrorMsg string
	Errors   []string
}

// NewViewData creates a new ViewData with the given title and auth context
func NewViewData(title string, ctx *gin.Context) ViewData {
	// Get auth data from context if available
	var authData AuthData
	if authDataInterface, exists := ctx.Get("authData"); exists {
		if ad, ok := authDataInterface.(AuthData); ok {
			authData = ad
		} else {
			authData = NewAuthData()
		}
	} else {
		authData = NewAuthData()
	}

	// Set the title
	authData = authData.WithTitle(title)

	// Get the current path
	authData = authData.WithCurrentPath(ctx.Request.URL.Path)

	// Create the view data
	return ViewData{
		AuthData: authData,
		Data:     nil, // Will be set by the caller
	}
}
