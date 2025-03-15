package server

import (
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
)

// RegisterAdminRoutes registers all admin routes
func (s *Server) RegisterAdminRoutes(r *gin.Engine, authController *controller.AuthController) {
	// Create admin controllers
	adminManufacturerController := controller.NewAdminManufacturerController(s.db)
	adminCaliberController := controller.NewAdminCaliberController(s.db)
	adminWeaponTypeController := controller.NewAdminWeaponTypeController(s.db)

	// Create admin route group without authentication middleware for now
	adminGroup := r.Group("/admin")
	{
		// Manufacturer routes
		manufacturerGroup := adminGroup.Group("/manufacturers")
		{
			manufacturerGroup.GET("", adminManufacturerController.Index)
			manufacturerGroup.GET("/new", adminManufacturerController.New)
			manufacturerGroup.POST("", adminManufacturerController.Create)
			manufacturerGroup.GET("/:id", adminManufacturerController.Show)
			manufacturerGroup.GET("/:id/edit", adminManufacturerController.Edit)
			manufacturerGroup.POST("/:id", adminManufacturerController.Update)
			manufacturerGroup.POST("/:id/delete", adminManufacturerController.Delete)
		}

		// Caliber routes
		caliberGroup := adminGroup.Group("/calibers")
		{
			caliberGroup.GET("", adminCaliberController.Index)
			caliberGroup.GET("/new", adminCaliberController.New)
			caliberGroup.POST("", adminCaliberController.Create)
			caliberGroup.GET("/:id", adminCaliberController.Show)
			caliberGroup.GET("/:id/edit", adminCaliberController.Edit)
			caliberGroup.POST("/:id", adminCaliberController.Update)
			caliberGroup.POST("/:id/delete", adminCaliberController.Delete)
		}

		// Weapon type routes
		weaponTypeGroup := adminGroup.Group("/weapon_types")
		{
			weaponTypeGroup.GET("", adminWeaponTypeController.Index)
			weaponTypeGroup.GET("/new", adminWeaponTypeController.New)
			weaponTypeGroup.POST("", adminWeaponTypeController.Create)
			weaponTypeGroup.GET("/:id", adminWeaponTypeController.Show)
			weaponTypeGroup.GET("/:id/edit", adminWeaponTypeController.Edit)
			weaponTypeGroup.POST("/:id", adminWeaponTypeController.Update)
			weaponTypeGroup.POST("/:id/delete", adminWeaponTypeController.Delete)
		}
	}
}
