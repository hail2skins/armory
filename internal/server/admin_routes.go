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
	adminPaymentController := controller.NewAdminPaymentController(s.db)
	adminGunsController := controller.NewAdminGunsController(s.db)
	adminPermissionsController := controller.NewAdminPermissionsController(s.db)
	adminFeatureFlagsController := controller.NewAdminFeatureFlagsController(s.db)
	adminCasingController := controller.NewAdminCasingController(s.db)

	// Create Stripe security controller
	stripeSecurityController := controller.NewStripeSecurityController(s.ipFilterService)

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

	// Create a middleware function to set webhook stats in context
	webhookStatsMiddleware := func(c *gin.Context) {
		// Get webhook stats from middleware
		stats := middleware.GetWebhookStats()

		// Convert to the type expected by the controller
		controllerStats := controller.WebhookStats{
			TotalRequests:      stats.TotalRequests,
			SuccessfulRequests: stats.SuccessfulRequests,
			FailedRequests:     stats.FailedRequests,
			LastRequestTime:    stats.LastRequestTime,
			LastErrorTime:      stats.LastErrorTime,
			LastError:          stats.LastError,
		}

		// Add to context
		c.Set("webhookStats", controllerStats)
		c.Next()
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

		// TODO: Delete this commented code block in a future cleanup
		// The FlexibleAuthorize approach used on individual routes makes this global admin check unnecessary
		// If Casbin auth is available, also apply role-based access control for admin
		// COMMENTING OUT GLOBAL ADMIN CHECK: This was preventing users with specific resource permissions
		// from accessing admin routes unless they had the admin role
		// if casbinAuth != nil {
		//     adminGroup.Use(casbinAuth.Authorize("admin"))
		// }

		// ==================================================================================
		// SECURITY WARNING: ALL admin routes MUST use FlexibleAuthorize for proper security
		// When adding new routes, follow this pattern:
		//
		// if casbinAuth != nil {
		//     adminGroup.GET("/route", casbinAuth.FlexibleAuthorize("resource", "action"), controller.Action)
		// } else {
		//     adminGroup.GET("/route", controller.Action)
		// }
		//
		// This ensures:
		// 1. Proper security checks for all admin routes
		// 2. Consistent permission model throughout the application
		// 3. Support for both role-based and permission-based access
		// ==================================================================================

		// Admin dashboard routes
		if casbinAuth != nil {
			adminGroup.GET("/dashboard", casbinAuth.FlexibleAuthorize("dashboard", "read"), adminDashboardController.Dashboard)
			adminGroup.GET("/detailed-health", casbinAuth.FlexibleAuthorize("dashboard", "read"), webhookStatsMiddleware, adminDashboardController.DetailedHealth)
			adminGroup.GET("/error-metrics", casbinAuth.FlexibleAuthorize("dashboard", "read"), adminDashboardController.ErrorMetrics)
		} else {
			adminGroup.GET("/dashboard", adminDashboardController.Dashboard)
			adminGroup.GET("/detailed-health", webhookStatsMiddleware, adminDashboardController.DetailedHealth)
			adminGroup.GET("/error-metrics", adminDashboardController.ErrorMetrics)
		}

		// Stripe security routes
		if casbinAuth != nil {
			adminGroup.GET("/stripe-security", casbinAuth.FlexibleAuthorize("stripe", "read"), stripeSecurityController.Dashboard)
			adminGroup.POST("/stripe-security/refresh", casbinAuth.FlexibleAuthorize("stripe", "write"), stripeSecurityController.RefreshIPRanges)
			adminGroup.POST("/stripe-security/toggle-filtering", casbinAuth.FlexibleAuthorize("stripe", "write"), stripeSecurityController.ToggleIPFilter)
			adminGroup.GET("/stripe-security/test-ip", casbinAuth.FlexibleAuthorize("stripe", "read"), stripeSecurityController.TestIPForm)
			adminGroup.POST("/stripe-security/check-ip", casbinAuth.FlexibleAuthorize("stripe", "read"), stripeSecurityController.CheckIP)
		} else {
			adminGroup.GET("/stripe-security", stripeSecurityController.Dashboard)
			adminGroup.POST("/stripe-security/refresh", stripeSecurityController.RefreshIPRanges)
			adminGroup.POST("/stripe-security/toggle-filtering", stripeSecurityController.ToggleIPFilter)
			adminGroup.GET("/stripe-security/test-ip", stripeSecurityController.TestIPForm)
			adminGroup.POST("/stripe-security/check-ip", stripeSecurityController.CheckIP)
		}

		// Manufacturer routes
		manufacturerGroup := adminGroup.Group("/manufacturers")
		{
			if casbinAuth != nil {
				// Define routes with flexible Casbin authorization that checks for specific permissions OR admin role
				manufacturerGroup.GET("", casbinAuth.FlexibleAuthorize("manufacturers", "read"), adminManufacturerController.Index)
				manufacturerGroup.GET("/new", casbinAuth.FlexibleAuthorize("manufacturers", "write"), adminManufacturerController.New)
				manufacturerGroup.POST("", casbinAuth.FlexibleAuthorize("manufacturers", "write"), adminManufacturerController.Create)
				manufacturerGroup.GET("/:id", casbinAuth.FlexibleAuthorize("manufacturers", "read"), adminManufacturerController.Show)
				manufacturerGroup.GET("/:id/edit", casbinAuth.FlexibleAuthorize("manufacturers", "update"), adminManufacturerController.Edit)
				manufacturerGroup.POST("/:id", casbinAuth.FlexibleAuthorize("manufacturers", "update"), adminManufacturerController.Update)
				manufacturerGroup.POST("/:id/delete", casbinAuth.FlexibleAuthorize("manufacturers", "delete"), adminManufacturerController.Delete)
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
				// Define routes with flexible Casbin authorization
				caliberGroup.GET("", casbinAuth.FlexibleAuthorize("calibers", "read"), adminCaliberController.Index)
				caliberGroup.GET("/new", casbinAuth.FlexibleAuthorize("calibers", "write"), adminCaliberController.New)
				caliberGroup.POST("", casbinAuth.FlexibleAuthorize("calibers", "write"), adminCaliberController.Create)
				caliberGroup.GET("/:id", casbinAuth.FlexibleAuthorize("calibers", "read"), adminCaliberController.Show)
				caliberGroup.GET("/:id/edit", casbinAuth.FlexibleAuthorize("calibers", "update"), adminCaliberController.Edit)
				caliberGroup.POST("/:id", casbinAuth.FlexibleAuthorize("calibers", "update"), adminCaliberController.Update)
				caliberGroup.POST("/:id/delete", casbinAuth.FlexibleAuthorize("calibers", "delete"), adminCaliberController.Delete)
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
				// Define routes with flexible Casbin authorization
				weaponTypeGroup.GET("", casbinAuth.FlexibleAuthorize("weapon_types", "read"), adminWeaponTypeController.Index)
				weaponTypeGroup.GET("/new", casbinAuth.FlexibleAuthorize("weapon_types", "write"), adminWeaponTypeController.New)
				weaponTypeGroup.POST("", casbinAuth.FlexibleAuthorize("weapon_types", "write"), adminWeaponTypeController.Create)
				weaponTypeGroup.GET("/:id", casbinAuth.FlexibleAuthorize("weapon_types", "read"), adminWeaponTypeController.Show)
				weaponTypeGroup.GET("/:id/edit", casbinAuth.FlexibleAuthorize("weapon_types", "update"), adminWeaponTypeController.Edit)
				weaponTypeGroup.POST("/:id", casbinAuth.FlexibleAuthorize("weapon_types", "update"), adminWeaponTypeController.Update)
				weaponTypeGroup.POST("/:id/delete", casbinAuth.FlexibleAuthorize("weapon_types", "delete"), adminWeaponTypeController.Delete)
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

		// Casing routes
		casingGroup := adminGroup.Group("/casings")
		{
			if casbinAuth != nil {
				// Define routes with flexible Casbin authorization
				casingGroup.GET("", casbinAuth.FlexibleAuthorize("casings", "read"), adminCasingController.Index)
				casingGroup.GET("/new", casbinAuth.FlexibleAuthorize("casings", "write"), adminCasingController.New)
				casingGroup.POST("", casbinAuth.FlexibleAuthorize("casings", "write"), adminCasingController.Create)
				casingGroup.GET("/:id", casbinAuth.FlexibleAuthorize("casings", "read"), adminCasingController.Show)
				casingGroup.GET("/:id/edit", casbinAuth.FlexibleAuthorize("casings", "write"), adminCasingController.Edit)
				casingGroup.POST("/:id", casbinAuth.FlexibleAuthorize("casings", "write"), adminCasingController.Update)
				casingGroup.POST("/:id/delete", casbinAuth.FlexibleAuthorize("casings", "write"), adminCasingController.Delete)
			} else {
				// Without Casbin, register routes with just authentication middleware
				casingGroup.GET("", adminCasingController.Index)
				casingGroup.GET("/new", adminCasingController.New)
				casingGroup.POST("", adminCasingController.Create)
				casingGroup.GET("/:id", adminCasingController.Show)
				casingGroup.GET("/:id/edit", adminCasingController.Edit)
				casingGroup.POST("/:id", adminCasingController.Update)
				casingGroup.POST("/:id/delete", adminCasingController.Delete)
			}
		}

		// ===== Promotion Routes =====
		promotionGroup := adminGroup.Group("/promotions")
		{
			if casbinAuth != nil {
				// Define routes with flexible Casbin authorization
				promotionGroup.GET("", casbinAuth.FlexibleAuthorize("promotions", "read"), adminPromotionController.Index)
				promotionGroup.GET("/index", casbinAuth.FlexibleAuthorize("promotions", "read"), adminPromotionController.Index)
				promotionGroup.GET("/new", casbinAuth.FlexibleAuthorize("promotions", "write"), adminPromotionController.New)
				promotionGroup.POST("", casbinAuth.FlexibleAuthorize("promotions", "write"), adminPromotionController.Create)
				promotionGroup.GET("/:id", casbinAuth.FlexibleAuthorize("promotions", "read"), adminPromotionController.Show)
				promotionGroup.GET("/:id/edit", casbinAuth.FlexibleAuthorize("promotions", "update"), adminPromotionController.Edit)
				promotionGroup.POST("/:id", casbinAuth.FlexibleAuthorize("promotions", "update"), adminPromotionController.Update)
				promotionGroup.POST("/:id/delete", casbinAuth.FlexibleAuthorize("promotions", "delete"), adminPromotionController.Delete)
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
				// Define routes with flexible Casbin authorization
				userGroup.GET("", casbinAuth.FlexibleAuthorize("users", "read"), adminUserController.Index)
				userGroup.GET("/:id", casbinAuth.FlexibleAuthorize("users", "read"), adminUserController.Show)
				userGroup.GET("/:id/edit", casbinAuth.FlexibleAuthorize("users", "update"), adminUserController.Edit)
				userGroup.POST("/:id", casbinAuth.FlexibleAuthorize("users", "update"), adminUserController.Update)
				userGroup.POST("/:id/delete", casbinAuth.FlexibleAuthorize("users", "delete"), adminUserController.Delete)
				userGroup.POST("/:id/restore", casbinAuth.FlexibleAuthorize("users", "update"), adminUserController.Restore)
				userGroup.GET("/:id/grant-subscription", casbinAuth.FlexibleAuthorize("users", "update"), adminUserController.ShowGrantSubscription)
				userGroup.POST("/:id/grant-subscription", casbinAuth.FlexibleAuthorize("users", "update"), adminUserController.GrantSubscription)
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

		// ===== Permission Management Routes =====
		permissionsGroup := adminGroup.Group("/permissions")
		{
			if casbinAuth != nil {
				// Define routes with flexible Casbin authorization
				permissionsGroup.GET("", casbinAuth.FlexibleAuthorize("permissions", "read"), adminPermissionsController.Index)

				// Role management
				permissionsGroup.GET("/roles/create", casbinAuth.FlexibleAuthorize("permissions", "write"), adminPermissionsController.CreateRole)
				permissionsGroup.POST("/roles/create", casbinAuth.FlexibleAuthorize("permissions", "write"), adminPermissionsController.StoreRole)
				permissionsGroup.GET("/roles/edit/:role", casbinAuth.FlexibleAuthorize("permissions", "write"), adminPermissionsController.EditRole)
				permissionsGroup.POST("/roles/update", casbinAuth.FlexibleAuthorize("permissions", "write"), adminPermissionsController.UpdateRole)
				permissionsGroup.GET("/roles/delete/:role", casbinAuth.FlexibleAuthorize("permissions", "write"), adminPermissionsController.DeleteRole)

				// User role assignment
				permissionsGroup.GET("/assign-role", casbinAuth.FlexibleAuthorize("permissions", "write"), adminPermissionsController.AssignRole)
				permissionsGroup.POST("/assign-role", casbinAuth.FlexibleAuthorize("permissions", "write"), adminPermissionsController.StoreAssignRole)
				permissionsGroup.POST("/remove-user-role", casbinAuth.FlexibleAuthorize("permissions", "write"), adminPermissionsController.RemoveUserRole)

				// Import default policies
				permissionsGroup.POST("/import-default-policies", casbinAuth.FlexibleAuthorize("permissions", "write"), adminPermissionsController.ImportDefaultPolicies)

				// Feature flags management
				permissionsGroup.GET("/feature-flags", casbinAuth.FlexibleAuthorize("feature_flags", "read"), adminFeatureFlagsController.Index)
				permissionsGroup.GET("/feature-flags/create", casbinAuth.FlexibleAuthorize("feature_flags", "write"), adminFeatureFlagsController.Create)
				permissionsGroup.POST("/feature-flags/create", casbinAuth.FlexibleAuthorize("feature_flags", "write"), adminFeatureFlagsController.Store)
				permissionsGroup.GET("/feature-flags/edit/:id", casbinAuth.FlexibleAuthorize("feature_flags", "update"), adminFeatureFlagsController.Edit)
				permissionsGroup.POST("/feature-flags/edit/:id", casbinAuth.FlexibleAuthorize("feature_flags", "update"), adminFeatureFlagsController.Update)
				permissionsGroup.POST("/feature-flags/delete/:id", casbinAuth.FlexibleAuthorize("feature_flags", "delete"), adminFeatureFlagsController.Delete)
				permissionsGroup.POST("/feature-flags/:id/roles", casbinAuth.FlexibleAuthorize("feature_flags", "update"), adminFeatureFlagsController.AddRole)
				permissionsGroup.POST("/feature-flags/:id/roles/remove", casbinAuth.FlexibleAuthorize("feature_flags", "update"), adminFeatureFlagsController.RemoveRole)
			} else {
				// Without Casbin, register routes with just authentication middleware
				permissionsGroup.GET("", adminPermissionsController.Index)

				// Role management
				permissionsGroup.GET("/roles/create", adminPermissionsController.CreateRole)
				permissionsGroup.POST("/roles/create", adminPermissionsController.StoreRole)
				permissionsGroup.GET("/roles/edit/:role", adminPermissionsController.EditRole)
				permissionsGroup.POST("/roles/update", adminPermissionsController.UpdateRole)
				permissionsGroup.GET("/roles/delete/:role", adminPermissionsController.DeleteRole)

				// User role assignment
				permissionsGroup.GET("/assign-role", adminPermissionsController.AssignRole)
				permissionsGroup.POST("/assign-role", adminPermissionsController.StoreAssignRole)
				permissionsGroup.POST("/remove-user-role", adminPermissionsController.RemoveUserRole)

				// Import default policies
				permissionsGroup.POST("/import-default-policies", adminPermissionsController.ImportDefaultPolicies)

				// Feature flags management
				permissionsGroup.GET("/feature-flags", adminFeatureFlagsController.Index)
				permissionsGroup.GET("/feature-flags/create", adminFeatureFlagsController.Create)
				permissionsGroup.POST("/feature-flags/create", adminFeatureFlagsController.Store)
				permissionsGroup.GET("/feature-flags/edit/:id", adminFeatureFlagsController.Edit)
				permissionsGroup.POST("/feature-flags/edit/:id", adminFeatureFlagsController.Update)
				permissionsGroup.POST("/feature-flags/delete/:id", adminFeatureFlagsController.Delete)
				permissionsGroup.POST("/feature-flags/:id/roles", adminFeatureFlagsController.AddRole)
				permissionsGroup.POST("/feature-flags/:id/roles/remove", adminFeatureFlagsController.RemoveRole)
			}
		}

		// ===== Payment Management Routes =====
		// Payments history
		if casbinAuth != nil {
			adminGroup.GET("/payments-history", casbinAuth.FlexibleAuthorize("payments", "read"), adminPaymentController.ShowPaymentsHistory)
		} else {
			adminGroup.GET("/payments-history", adminPaymentController.ShowPaymentsHistory)
		}

		// ===== Guns Management Routes =====
		if casbinAuth != nil {
			adminGroup.GET("/guns", casbinAuth.FlexibleAuthorize("guns", "read"), adminGunsController.Index)
		} else {
			adminGroup.GET("/guns", adminGunsController.Index)
		}

		// ===== Dashboard Routes =====
		if casbinAuth != nil {
			adminGroup.GET("", casbinAuth.FlexibleAuthorize("dashboard", "read"), adminDashboardController.Dashboard)
		} else {
			adminGroup.GET("", adminDashboardController.Dashboard)
		}
	}
}
