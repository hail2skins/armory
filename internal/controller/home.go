package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/cmd/web/views/home"
	"github.com/hail2skins/armory/internal/database"
)

// HomeController handles home page related routes
type HomeController struct {
	db database.Service
}

// NewHomeController creates a new home controller
func NewHomeController(db database.Service) *HomeController {
	return &HomeController{
		db: db,
	}
}

// HomeHandler handles the home page route
func (h *HomeController) HomeHandler(c *gin.Context) {
	// Get the current user's authentication status and email
	userInfo, authenticated := c.MustGet("authController").(*AuthController).GetCurrentUser(c)

	// Create HomeData with proper AuthData
	homeData := home.HomeData{
		AuthData: data.AuthData{
			Authenticated: authenticated,
		},
	}

	// Set email if authenticated
	if authenticated {
		homeData.Email = userInfo.GetUserName()
	}

	// Render the home page with the data
	home.Home(homeData).Render(c.Request.Context(), c.Writer)
}
