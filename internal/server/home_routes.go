package server

import (
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
)

// RegisterHomeRoutes registers all home page related routes
func (s *Server) RegisterHomeRoutes(r *gin.Engine, homeController *controller.HomeController) {
	// Home page routes
	r.GET("/", homeController.HomeHandler)
	r.GET("/about", homeController.AboutHandler)
	r.GET("/contact", homeController.ContactHandler)
	r.POST("/contact", homeController.ContactHandler)

	// Future routes can be added here:
	// r.GET("/pricing", homeController.PricingHandler)
}
