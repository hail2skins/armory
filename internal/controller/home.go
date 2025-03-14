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
	authData := data.NewAuthData()
	authData.Authenticated = authenticated
	authData.Title = "Home"

	homeData := home.HomeData{
		AuthData: authData,
	}

	// Set email if authenticated
	if authenticated {
		homeData.Email = userInfo.GetUserName()
	}

	// Render the home page with the data
	home.Home(homeData).Render(c.Request.Context(), c.Writer)
}

// AboutHandler handles the about page route
func (h *HomeController) AboutHandler(c *gin.Context) {
	// Get the current user's authentication status and email
	userInfo, authenticated := c.MustGet("authController").(*AuthController).GetCurrentUser(c)

	// Create AuthData
	authData := data.NewAuthData()
	authData.Authenticated = authenticated
	authData.Title = "About"

	// Create AboutData with the AuthData
	aboutData := home.AboutData{
		AuthData: authData,
	}

	// Set email if authenticated
	if authenticated {
		aboutData.Email = userInfo.GetUserName()
	}

	// Render the about page with the data
	home.About(aboutData).Render(c.Request.Context(), c.Writer)
}
