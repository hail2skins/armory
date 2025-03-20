package server

import (
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/logger"
	"github.com/hail2skins/armory/internal/middleware"
	"github.com/shaj13/go-guardian/v2/auth"
)

// RegisterAdminRoutes registers all admin routes
func (s *Server) RegisterAdminRoutes(r *gin.Engine, authController *controller.AuthController) {
	// Create admin controllers
	adminManufacturerController := controller.NewAdminManufacturerController(s.db)
	adminCaliberController := controller.NewAdminCaliberController(s.db)
	adminWeaponTypeController := controller.NewAdminWeaponTypeController(s.db)
	adminDashboardController := controller.NewAdminDashboardController(s.db)
	adminPromotionController := controller.NewAdminPromotionController(s.db)
	adminUserController := controller.NewAdminUserController(s.db)

	// Use the shared Casbin auth instance from the server
	casbinAuth := s.casbinAuth
	if casbinAuth == nil {
		// If the shared instance is not available, create a new one
		configDir := filepath.Join("configs", "casbin")
		var err error
		casbinAuth, err = middleware.NewCasbinAuth(
			filepath.Join(configDir, "rbac_model.conf"),
			filepath.Join(configDir, "rbac_policy.csv"),
		)
		if err != nil {
			// Log and continue without Casbin protection - will use existing auth
			logger.Error("Failed to initialize Casbin", err, nil)
		}
	}

	// Create admin route group with authentication middleware
	adminGroup := r.Group("/admin")
	{
		// Apply the authentication middleware to ensure user is logged in
		adminGroup.Use(authController.AuthMiddleware())

		// Add middleware to ensure roles are set in the context
		adminGroup.Use(func(c *gin.Context) {
			// Get authData from context
			authDataInterface, exists := c.Get("authData")
			if !exists {
				c.Next()
				return
			}

			// Get the auth info for current user
			userInfo, exists := c.Get("auth_info")
			if !exists {
				c.Next()
				return
			}

			info, ok := userInfo.(auth.Info)
			if !ok {
				c.Next()
				return
			}

			// Update authData with roles
			authData, ok := authDataInterface.(data.AuthData)
			if ok && casbinAuth != nil {
				roles := casbinAuth.GetUserRoles(info.GetUserName())
				isAdmin := false
				for _, role := range roles {
					if role == "admin" {
						isAdmin = true
						break
					}
				}

				// Update authData
				authData.Roles = roles
				authData.IsCasbinAdmin = isAdmin

				// Put back in context
				c.Set("authData", authData)
			}

			c.Next()
		})

		// If Casbin auth is available, also apply role-based access control for admin
		if casbinAuth != nil {
			adminGroup.Use(casbinAuth.Authorize("admin"))
		}

		// Dashboard routes
		if casbinAuth != nil {
			adminGroup.GET("/dashboard", casbinAuth.Authorize("dashboard", "read"), adminDashboardController.Dashboard)
			adminGroup.GET("/detailed-health", casbinAuth.Authorize("dashboard", "read"), adminDashboardController.DetailedHealth)
			adminGroup.GET("/error-metrics", casbinAuth.Authorize("dashboard", "read"), adminDashboardController.ErrorMetrics)
		} else {
			adminGroup.GET("/dashboard", adminDashboardController.Dashboard)
			adminGroup.GET("/detailed-health", adminDashboardController.DetailedHealth)
			adminGroup.GET("/error-metrics", adminDashboardController.ErrorMetrics)
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

		// ===== Promotion Routes =====
		promotionGroup := adminGroup.Group("/promotions")
		{
			if casbinAuth != nil {
				// Define routes with fine-grained Casbin authorization
				promotionGroup.GET("", casbinAuth.Authorize("admin", "read"), adminPromotionController.Index)
				promotionGroup.GET("/index", casbinAuth.Authorize("admin", "read"), adminPromotionController.Index)
				promotionGroup.GET("/new", casbinAuth.Authorize("admin", "write"), adminPromotionController.New)
				promotionGroup.POST("", casbinAuth.Authorize("admin", "write"), adminPromotionController.Create)
				promotionGroup.GET("/:id", casbinAuth.Authorize("admin", "read"), adminPromotionController.Show)
				promotionGroup.GET("/:id/edit", casbinAuth.Authorize("admin", "update"), adminPromotionController.Edit)
				promotionGroup.POST("/:id", casbinAuth.Authorize("admin", "update"), adminPromotionController.Update)
				promotionGroup.POST("/:id/delete", casbinAuth.Authorize("admin", "delete"), adminPromotionController.Delete)
			} else {
				// Without Casbin, register routes with just authentication middleware
				promotionGroup.GET("", adminPromotionController.Index)
				promotionGroup.GET("/index", adminPromotionController.Index)
				promotionGroup.GET("/new", adminPromotionController.New)
				promotionGroup.POST("", adminPromotionController.Create)
				promotionGroup.GET("/:id", adminPromotionController.Show)
				promotionGroup.GET("/:id/edit", adminPromotionController.Edit)
				promotionGroup.POST("/:id", adminPromotionController.Update)
				promotionGroup.POST("/:id/delete", adminPromotionController.Delete)
			}
		}

		// ===== User Management Routes =====
		userGroup := adminGroup.Group("/users")
		{
			if casbinAuth != nil {
				// Define routes with fine-grained Casbin authorization
				userGroup.GET("", casbinAuth.Authorize("admin", "read"), adminUserController.Index)
				userGroup.GET("/:id", casbinAuth.Authorize("admin", "read"), adminUserController.Show)
				userGroup.GET("/:id/edit", casbinAuth.Authorize("admin", "update"), adminUserController.Edit)
				userGroup.POST("/:id", casbinAuth.Authorize("admin", "update"), adminUserController.Update)
				userGroup.POST("/:id/delete", casbinAuth.Authorize("admin", "delete"), adminUserController.Delete)
				userGroup.POST("/:id/restore", casbinAuth.Authorize("admin", "update"), adminUserController.Restore)
				userGroup.GET("/:id/grant-subscription", casbinAuth.Authorize("admin", "update"), adminUserController.ShowGrantSubscription)
				userGroup.POST("/:id/grant-subscription", casbinAuth.Authorize("admin", "update"), adminUserController.GrantSubscription)
			} else {
				// Without Casbin, register routes with just authentication middleware
				userGroup.GET("", adminUserController.Index)
				userGroup.GET("/:id", adminUserController.Show)
				userGroup.GET("/:id/edit", adminUserController.Edit)
				userGroup.POST("/:id", adminUserController.Update)
				userGroup.POST("/:id/delete", adminUserController.Delete)
				userGroup.POST("/:id/restore", adminUserController.Restore)
				userGroup.GET("/:id/grant-subscription", adminUserController.ShowGrantSubscription)
				userGroup.POST("/:id/grant-subscription", adminUserController.GrantSubscription)
			}
		}

		// ===== Dashboard Routes =====
		adminGroup.GET("", adminDashboardController.Dashboard)
	}
}
