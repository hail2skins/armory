package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
)

// DebugController handles debug routes
type DebugController struct{}

// NewDebugController creates a new debug controller
func NewDebugController() *DebugController {
	return &DebugController{}
}

// RegisterRoutes registers debug routes on a router group
func (d *DebugController) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/debug/roles/details", d.RolesDebugHandler)
}

// RolesDebugHandler displays current user role information
func (d *DebugController) RolesDebugHandler(c *gin.Context) {
	// Get the auth data from the context
	c.Header("Content-Type", "text/html")

	c.Writer.Write([]byte("<html><body><h1>Authentication Debug</h1>"))

	// Try to get authData from context
	if authDataInterface, exists := c.Get("authData"); exists {
		authData, ok := authDataInterface.(data.AuthData)
		if ok {
			c.Writer.Write([]byte(fmt.Sprintf("<p>AuthData found in context</p>")))
			c.Writer.Write([]byte(fmt.Sprintf("<p>Email: %s</p>", authData.Email)))
			c.Writer.Write([]byte(fmt.Sprintf("<p>Authenticated: %v</p>", authData.Authenticated)))
			c.Writer.Write([]byte(fmt.Sprintf("<p>IsCasbinAdmin: %v</p>", authData.IsCasbinAdmin)))
			c.Writer.Write([]byte(fmt.Sprintf("<p>AlwaysTrue: %v</p>", authData.AlwaysTrue)))
			c.Writer.Write([]byte(fmt.Sprintf("<p>Roles: %v</p>", authData.Roles)))
			c.Writer.Write([]byte(fmt.Sprintf("<p>HasRole('admin'): %v</p>", authData.HasRole("admin"))))
		} else {
			c.Writer.Write([]byte("<p>AuthData found in context but couldn't be cast to data.AuthData</p>"))
		}
	} else {
		c.Writer.Write([]byte("<p>No AuthData found in context</p>"))
	}

	// Try to get auth from middleware
	if authInterface, exists := c.Get("auth"); exists {
		c.Writer.Write([]byte("<h2>Auth Data from Middleware</h2>"))
		c.Writer.Write([]byte(fmt.Sprintf("<pre>%+v</pre>", authInterface)))

		// Try to get user info
		if auth, ok := authInterface.(AuthService); ok {
			c.Writer.Write([]byte("<p>Auth middleware found!</p>"))
			userInfo, authenticated := auth.GetCurrentUser(c)
			c.Writer.Write([]byte(fmt.Sprintf("<p>Authenticated: %v</p>", authenticated)))
			if authenticated {
				c.Writer.Write([]byte(fmt.Sprintf("<p>Email: %s</p>", userInfo.GetUserName())))
			}
		} else {
			c.Writer.Write([]byte("<p>Auth middleware not of expected type</p>"))
		}
	} else {
		c.Writer.Write([]byte("<p>No auth middleware found in context</p>"))
	}

	c.Writer.Write([]byte("</body></html>"))
}
