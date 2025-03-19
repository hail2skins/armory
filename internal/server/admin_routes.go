package server

import (
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/logger"
	"github.com/hail2skins/armory/internal/middleware"
)

// RegisterAdminRoutes registers all admin routes
func (s *Server) RegisterAdminRoutes(r *gin.Engine, authController *controller.AuthController) {
	// Create admin controllers
	adminManufacturerController := controller.NewAdminManufacturerController(s.db)
	adminCaliberController := controller.NewAdminCaliberController(s.db)
	adminWeaponTypeController := controller.NewAdminWeaponTypeController(s.db)

	// Get config directory for Casbin
	configDir := filepath.Join("configs", "casbin")

	// Create Casbin auth middleware
	casbinAuth, err := middleware.NewCasbinAuth(
		filepath.Join(configDir, "rbac_model.conf"),
		filepath.Join(configDir, "rbac_policy.csv"),
	)
	if err != nil {
		// Log and continue without Casbin protection - will use existing auth
		logger.Error("Failed to initialize Casbin", err, nil)
	}

	// Create admin route group with authentication middleware
	adminGroup := r.Group("/admin")
	{
		// Apply the authentication middleware to ensure user is logged in
		adminGroup.Use(authController.AuthMiddleware())

		// If Casbin auth is available, also apply role-based access control for admin
		if casbinAuth != nil {
			adminGroup.Use(casbinAuth.Authorize("admin"))
		}

		// Manufacturer routes
		manufacturerGroup := adminGroup.Group("/manufacturers")
		{
			if casbinAuth != nil {
				// Define routes with fine-grained Casbin authorization
				manufacturerGroup.GET("", casbinAuth.Authorize("manufacturers", "read"), adminManufacturerController.Index)
				manufacturerGroup.GET("/new", casbinAuth.Authorize("manufacturers", "write"), adminManufacturerController.New)
				manufacturerGroup.POST("", casbinAuth.Authorize("manufacturers", "write"), adminManufacturerController.Create)
				manufacturerGroup.GET("/:id", casbinAuth.Authorize("manufacturers", "read"), adminManufacturerController.Show)
				manufacturerGroup.GET("/:id/edit", casbinAuth.Authorize("manufacturers", "update"), adminManufacturerController.Edit)
				manufacturerGroup.POST("/:id", casbinAuth.Authorize("manufacturers", "update"), adminManufacturerController.Update)
				manufacturerGroup.POST("/:id/delete", casbinAuth.Authorize("manufacturers", "delete"), adminManufacturerController.Delete)
			} else {
				// Without Casbin, register routes with just authentication middleware
				manufacturerGroup.GET("", adminManufacturerController.Index)
				manufacturerGroup.GET("/new", adminManufacturerController.New)
				manufacturerGroup.POST("", adminManufacturerController.Create)
				manufacturerGroup.GET("/:id", adminManufacturerController.Show)
				manufacturerGroup.GET("/:id/edit", adminManufacturerController.Edit)
				manufacturerGroup.POST("/:id", adminManufacturerController.Update)
				manufacturerGroup.POST("/:id/delete", adminManufacturerController.Delete)
			}
		}

		// Caliber routes
		caliberGroup := adminGroup.Group("/calibers")
		{
			if casbinAuth != nil {
				// Define routes with fine-grained Casbin authorization
				caliberGroup.GET("", casbinAuth.Authorize("calibers", "read"), adminCaliberController.Index)
				caliberGroup.GET("/new", casbinAuth.Authorize("calibers", "write"), adminCaliberController.New)
				caliberGroup.POST("", casbinAuth.Authorize("calibers", "write"), adminCaliberController.Create)
				caliberGroup.GET("/:id", casbinAuth.Authorize("calibers", "read"), adminCaliberController.Show)
				caliberGroup.GET("/:id/edit", casbinAuth.Authorize("calibers", "update"), adminCaliberController.Edit)
				caliberGroup.POST("/:id", casbinAuth.Authorize("calibers", "update"), adminCaliberController.Update)
				caliberGroup.POST("/:id/delete", casbinAuth.Authorize("calibers", "delete"), adminCaliberController.Delete)
			} else {
				caliberGroup.GET("", adminCaliberController.Index)
				caliberGroup.GET("/new", adminCaliberController.New)
				caliberGroup.POST("", adminCaliberController.Create)
				caliberGroup.GET("/:id", adminCaliberController.Show)
				caliberGroup.GET("/:id/edit", adminCaliberController.Edit)
				caliberGroup.POST("/:id", adminCaliberController.Update)
				caliberGroup.POST("/:id/delete", adminCaliberController.Delete)
			}
		}

		// Weapon type routes
		weaponTypeGroup := adminGroup.Group("/weapon_types")
		{
			if casbinAuth != nil {
				// Define routes with fine-grained Casbin authorization
				weaponTypeGroup.GET("", casbinAuth.Authorize("weapon_types", "read"), adminWeaponTypeController.Index)
				weaponTypeGroup.GET("/new", casbinAuth.Authorize("weapon_types", "write"), adminWeaponTypeController.New)
				weaponTypeGroup.POST("", casbinAuth.Authorize("weapon_types", "write"), adminWeaponTypeController.Create)
				weaponTypeGroup.GET("/:id", casbinAuth.Authorize("weapon_types", "read"), adminWeaponTypeController.Show)
				weaponTypeGroup.GET("/:id/edit", casbinAuth.Authorize("weapon_types", "update"), adminWeaponTypeController.Edit)
				weaponTypeGroup.POST("/:id", casbinAuth.Authorize("weapon_types", "update"), adminWeaponTypeController.Update)
				weaponTypeGroup.POST("/:id/delete", casbinAuth.Authorize("weapon_types", "delete"), adminWeaponTypeController.Delete)
			} else {
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
}
