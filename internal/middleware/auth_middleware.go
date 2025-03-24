package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/logger"
	"github.com/hail2skins/armory/internal/models"
	"github.com/justinas/nosurf"
)

// AuthMiddleware adds authentication data to the Gin context
func AuthMiddleware(authService controller.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Initialize with default values
		authData := data.NewAuthData()
		authData = authData.WithCurrentPath(c.Request.URL.Path)

		// Check if user is authenticated
		if authService.IsAuthenticated(c) {
			// Get user info directly from auth service
			userInfo, authenticated := authService.GetCurrentUser(c)
			if authenticated && userInfo != nil {
				email := userInfo.GetUserName()
				authData = authData.WithEmail(email)

				// Set authenticated flag
				authData.Authenticated = true

				// Set user email in context for other middleware or handlers
				c.Set("userEmail", email)

				// Get user roles if available
				if roles, exists := c.Get("userRoles"); exists {
					if userRoles, ok := roles.([]string); ok {
						authData = authData.WithRoles(userRoles)
					}
				}

				// Always try to get roles from Casbin, regardless of whether roles were already set
				if casbinAuth, exists := c.Get("casbinAuth"); exists && casbinAuth != nil {
					if ca, ok := casbinAuth.(interface{ GetUserRoles(string) []string }); ok {
						roles := ca.GetUserRoles(email)
						logger.Info("Casbin roles for user", map[string]interface{}{
							"email": email,
							"roles": roles,
						})

						// Apply these roles directly
						authData = authData.WithRoles(roles)

						// Also set in context for other components
						c.Set("userRoles", roles)

						// Debug output for admin status
						logger.Info("User admin status", map[string]interface{}{
							"email":   email,
							"isAdmin": authData.IsCasbinAdmin,
							"roles":   roles,
						})
					}
				}
			} else {
				logger.Warn("User authenticated but GetCurrentUser returned false", nil)
			}
		}

		// Add title from context if set by controller
		if title, exists := c.Get("title"); exists {
			if titleStr, ok := title.(string); ok {
				authData = authData.WithTitle(titleStr)
			}
		}

		// Add flash messages if they exist
		if flash, exists := c.Get("error"); exists {
			if flashStr, ok := flash.(string); ok && flashStr != "" {
				authData = authData.WithError(flashStr)
			}
		}

		if flash, exists := c.Get("success"); exists {
			if flashStr, ok := flash.(string); ok && flashStr != "" {
				authData = authData.WithSuccess(flashStr)
			}
		}

		// Get CSRF token directly from nosurf
		token := nosurf.Token(c.Request)
		if token != "" {
			authData = authData.WithCSRFToken(token)
			// Also set it in the context for other middleware
			c.Set("csrf_token", token)
		} else {
			// Try to get it from context as fallback
			if csrfToken, exists := c.Get("csrf_token"); exists {
				if tokenStr, ok := csrfToken.(string); ok && tokenStr != "" {
					authData = authData.WithCSRFToken(tokenStr)
				}
			}
		}

		// Add active promotion if it exists in context
		if promotion, exists := c.Get("active_promotion"); exists {
			if promo, ok := promotion.(*models.Promotion); ok {
				authData = authData.WithActivePromotion(promo)
				// Debug: add details about the promotion
				c.Set("debug_promo_added_to_auth", "true")
			} else {
				// Debug: type assertion failed
				c.Set("debug_promo_type_error", "true")
				c.Set("debug_promo_actual_type", fmt.Sprintf("%T", promotion))
			}
		} else {
			// Debug: no promotion in context
			c.Set("debug_no_promo_in_context", "true")
		}

		// Set auth data in context for views to access
		c.Set("authData", authData)
		c.Next()
	}
}
