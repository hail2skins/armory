package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/logger"
	"github.com/hail2skins/armory/internal/services"
)

// PromotionBanner is a middleware that adds active promotions to the context
// so they can be displayed in the views. This middleware implements multiple
// layers of security to ensure promotions are only shown in appropriate contexts:
//
// 1. Path-based security: Only specific whitelisted public pages show promotions
// 2. Multiple redundant checks to prevent promotions on admin/owner pages
// 3. Logging of all promotion display decisions for audit and debugging
//
// The promotion is added to the Gin context under the key "active_promotion"
// and templates can check for its existence and type-assert to access it.
func PromotionBanner(promotionService *services.PromotionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip for non-GET requests to avoid showing promotions during form submissions
		// and other non-content viewing operations
		if c.Request.Method != "GET" {
			c.Next()
			return
		}

		path := c.Request.URL.Path

		// STRICT WHITELIST: Only show banner on these specific public pages
		// This is the primary security mechanism - a whitelist approach is
		// more secure than a blacklist approach
		// NEVER show on /owner/*, /admin/*, or any other pages
		isAllowedPath := path == "/" || path == "/about" || path == "/contact" || path == "/pricing"

		// Log the path and whether the banner will be shown
		// This helps with debugging and security auditing
		logger.Info("Checking promotion banner visibility", map[string]interface{}{
			"path":       path,
			"showBanner": isAllowedPath,
		})

		if !isAllowedPath {
			// For all non-whitelisted paths, don't add promotion to context
			// This means the banner won't appear on these pages
			c.Next()
			return
		}

		// DEFENSE IN DEPTH: Additional safety check explicitly blocking admin/owner paths
		// This is redundant with the whitelist but adds an extra layer of protection
		// in case the whitelist is accidentally modified in the future
		if strings.HasPrefix(path, "/owner") || strings.HasPrefix(path, "/admin") {
			logger.Warn("Blocked promotion banner on restricted path", map[string]interface{}{
				"path": path,
			})
			c.Next()
			return
		}

		// Skip for login and register pages to avoid confusion
		// (redundant with the whitelist but keeping for clarity and future-proofing)
		if strings.HasPrefix(path, "/login") || strings.HasPrefix(path, "/register") {
			c.Next()
			return
		}

		// NOTE: The base.templ template has an additional check to prevent
		// authenticated users from seeing the promotion banner, even on
		// allowed paths. This creates a multi-layered approach to banner visibility.

		// Get the best active promotion from the service
		// The service handles the logic of determining which promotion to show
		// if multiple promotions are active simultaneously
		promotion, err := promotionService.GetBestActivePromotion()
		if err != nil {
			// Log the error but don't show it to the user
			// This prevents exposing internal errors to end users
			c.Set("promotion_error", err.Error())
		} else if promotion != nil {
			// Add promotion to context and log it for audit purposes
			c.Set("active_promotion", promotion)
			logger.Info("Adding promotion to context", map[string]interface{}{
				"path":           path,
				"promotion_name": promotion.Name,
				"benefit_days":   promotion.BenefitDays,
			})
		} else {
			// No active promotion found - add debug flag to context
			// This helps with debugging when promotions are expected but not showing
			c.Set("debug_no_promotion", "true")
		}

		// Continue processing the request
		c.Next()
	}
}
