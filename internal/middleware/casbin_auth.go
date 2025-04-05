package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/logger"
	"github.com/hail2skins/armory/internal/models"
	"github.com/shaj13/go-guardian/v2/auth"
	"gorm.io/gorm"
)

// CasbinAuth is the middleware for Casbin-based authorization
type CasbinAuth struct {
	enforcer *casbin.Enforcer
}

// NewCasbinAuth creates a new Casbin authorization middleware
func NewCasbinAuth(modelPath, policyPath string) (*CasbinAuth, error) {
	// This is using the file adapter, which is not loading role assignments from the database
	enforcer, err := casbin.NewEnforcer(modelPath, policyPath)
	if err != nil {
		return nil, err
	}

	return &CasbinAuth{
		enforcer: enforcer,
	}, nil
}

// NewCasbinAuthWithDB creates a new Casbin authorization middleware using the database adapter
func NewCasbinAuthWithDB(modelPath string, db *gorm.DB) (*CasbinAuth, error) {
	// Create a new Casbin DB adapter
	adapter := models.NewCasbinDBAdapter(db)

	// Load the model from the file but policies from the database
	enforcer, err := casbin.NewEnforcer(modelPath, adapter)
	if err != nil {
		return nil, err
	}

	// Load policies from the database
	if err := enforcer.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("failed to load policies from database: %w", err)
	}

	logger.Info("Successfully initialized Casbin with database adapter", nil)

	return &CasbinAuth{
		enforcer: enforcer,
	}, nil
}

// Authorize returns a middleware function that authorizes a request
func (ca *CasbinAuth) Authorize(obj string, act ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Reload policy from database to ensure we have the latest permissions
		err := ca.ReloadPolicy()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load policies"})
			c.Abort()
			return
		}

		// Get the current user from the context (set by authentication middleware)
		authInfo, exists := c.Get("auth_info")
		if !exists {
			// Log the missing auth info
			logger.Warn("Missing auth_info in context during Casbin authorization", map[string]interface{}{
				"path": c.Request.URL.Path,
				"keys": fmt.Sprintf("%v", c.Keys),
			})

			// User is not authenticated, redirect to home page
			setFlashMessage(c, "You must log in to access that resource")
			c.Redirect(http.StatusSeeOther, "/")
			c.Abort()
			return
		}

		// Get the user's email address as the subject
		userInfo, ok := authInfo.(auth.Info)
		if !ok {
			logger.Error("Invalid auth info in context", nil, map[string]interface{}{
				"auth_info_type": fmt.Sprintf("%T", authInfo),
				"path":           c.Request.URL.Path,
			})
			setFlashMessage(c, "An error occurred. Please try again later.")
			c.Redirect(http.StatusSeeOther, "/")
			c.Abort()
			return
		}

		sub := userInfo.GetUserName()
		logger.Info("Checking authorization", map[string]interface{}{
			"user":   sub,
			"path":   c.Request.URL.Path,
			"object": obj,
		})

		// Default action is wildcard if none provided
		action := "*"
		if len(act) > 0 {
			action = act[0]
		}

		// Check if the user is authorized
		allowed, err := ca.enforcer.Enforce(sub, obj, action)
		if err != nil {
			logger.Error("Casbin enforcement error", err, map[string]interface{}{
				"subject": sub,
				"object":  obj,
				"action":  action,
				"path":    c.Request.URL.Path,
			})
			setFlashMessage(c, "An error occurred. Please try again later.")
			c.Redirect(http.StatusSeeOther, "/")
			c.Abort()
			return
		}

		if !allowed {
			// Log the authorization failure
			logger.Info("Authorization denied", map[string]interface{}{
				"subject": sub,
				"object":  obj,
				"action":  action,
				"path":    c.Request.URL.Path,
			})
			setFlashMessage(c, "You do not have authorization for that resource")
			c.Redirect(http.StatusSeeOther, "/")
			c.Abort()
			return
		}

		// User is authorized, continue to the next handler
		logger.Info("Authorization granted", map[string]interface{}{
			"subject": sub,
			"object":  obj,
			"action":  action,
			"path":    c.Request.URL.Path,
		})
		c.Next()
	}
}

// setFlashMessage sets a flash message if the setFlash function is available in the context
func setFlashMessage(c *gin.Context, message string) {
	// Check if the setFlash function is available
	if setFlash, exists := c.Get("setFlash"); exists {
		// Try to call the setFlash function
		if flashFunc, ok := setFlash.(func(string)); ok {
			flashFunc(message)
		}
	} else {
		// Fallback to using the session directly
		session := sessions.Default(c)
		session.AddFlash(message)
		session.Save()
	}
}

// AddUserRole adds a user to a role in the Casbin model
func (ca *CasbinAuth) AddUserRole(user, role string) bool {
	success, _ := ca.enforcer.AddGroupingPolicy(user, role)
	return success
}

// RemoveUserRole removes a user from a role in the Casbin model
func (ca *CasbinAuth) RemoveUserRole(user, role string) bool {
	success, _ := ca.enforcer.RemoveGroupingPolicy(user, role)
	return success
}

// HasRole checks if a user has a specific role
func (ca *CasbinAuth) HasRole(user, role string) bool {
	roles, _ := ca.enforcer.GetRolesForUser(user)
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// GetUserRoles returns all roles for a user
func (ca *CasbinAuth) GetUserRoles(user string) []string {
	// Reload policy to ensure we have the latest data from the database
	if err := ca.enforcer.LoadPolicy(); err != nil {
		logger.Error("Failed to reload Casbin policy", err, map[string]interface{}{
			"user": user,
		})
	}

	roles, _ := ca.enforcer.GetRolesForUser(user)
	logger.Debug("GetUserRoles called", map[string]interface{}{
		"user":  user,
		"roles": roles,
	})
	return roles
}

// AddPolicy adds a new policy rule (role, resource, permission)
func (ca *CasbinAuth) AddPolicy(role, resource, permission string) bool {
	success, _ := ca.enforcer.AddPolicy(role, resource, permission)
	return success
}

// RemovePolicy removes a policy rule
func (ca *CasbinAuth) RemovePolicy(role, resource, permission string) bool {
	success, _ := ca.enforcer.RemovePolicy(role, resource, permission)
	return success
}

// SavePolicy saves the current policy to the policy file
func (ca *CasbinAuth) SavePolicy() error {
	return ca.enforcer.SavePolicy()
}

// ReloadPolicy reloads the policy from the policy file
func (ca *CasbinAuth) ReloadPolicy() error {
	return ca.enforcer.LoadPolicy()
}

// FlexibleAuthorize checks multiple permission paths:
// 1. First checks if feature has public access (bypass all permission checks)
// 2. If not, checks if user has specific resource:action permission
// 3. If not, checks if user has admin role (which grants all permissions)
// 4. Finally, checks if user has a role with the same name as the object
func (ca *CasbinAuth) FlexibleAuthorize(obj string, act ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// First check if this resource/feature has public access enabled
		if ca.isFeaturePublic(obj) {
			// Feature has public access, allow without checking permissions
			logger.Info("Authorization granted via public access feature flag", map[string]interface{}{
				"object": obj,
				"path":   c.Request.URL.Path,
			})
			c.Next()
			return
		}

		// Get the current user from the context (set by authentication middleware)
		authInfo, exists := c.Get("auth_info")
		if !exists {
			// Log the missing auth info
			logger.Warn("Missing auth_info in context during Casbin authorization", map[string]interface{}{
				"path": c.Request.URL.Path,
				"keys": fmt.Sprintf("%v", c.Keys),
			})

			// User is not authenticated, redirect to home page
			setFlashMessage(c, "You must log in to access that resource")
			c.Redirect(http.StatusSeeOther, "/")
			c.Abort()
			return
		}

		// Get the user's email address as the subject
		userInfo, ok := authInfo.(auth.Info)
		if !ok {
			logger.Error("Invalid auth info in context", nil, map[string]interface{}{
				"auth_info_type": fmt.Sprintf("%T", authInfo),
				"path":           c.Request.URL.Path,
			})
			setFlashMessage(c, "An error occurred. Please try again later.")
			c.Redirect(http.StatusSeeOther, "/")
			c.Abort()
			return
		}

		sub := userInfo.GetUserName()
		logger.Info("Flexible authorization check", map[string]interface{}{
			"user":   sub,
			"object": obj,
		})

		// Default action is wildcard if none provided
		action := "*"
		if len(act) > 0 {
			action = act[0]
		}

		// First check: specific resource permission
		specificAllowed, err := ca.enforcer.Enforce(sub, obj, action)
		if err == nil && specificAllowed {
			// User has specific permission for this resource
			logger.Info("Authorization granted via specific permission", map[string]interface{}{
				"subject": sub,
				"object":  obj,
				"action":  action,
				"path":    c.Request.URL.Path,
			})
			c.Next()
			return
		}

		// Second check: admin role permission (admin can access anything)
		adminAllowed, err := ca.enforcer.Enforce(sub, "admin", "*")
		if err == nil && adminAllowed {
			// User has admin permission
			logger.Info("Authorization granted via admin role", map[string]interface{}{
				"subject": sub,
				"object":  obj,
				"action":  action,
				"path":    c.Request.URL.Path,
			})
			c.Next()
			return
		}

		// Third check: directly check if the user has the role that matches the object
		// This is a fallback that checks if the user has a role like "test_read_manufacturers"
		roles := ca.GetUserRoles(sub)
		logger.Info("Checking user roles", map[string]interface{}{
			"subject": sub,
			"roles":   roles,
		})

		for _, role := range roles {
			// Check if this role has direct access to the requested object
			if strings.Contains(role, obj) {
				// If requesting read (or wildcard) action, any role with the object name is usually sufficient
				if action == "read" || action == "*" {
					logger.Info("Authorization granted via role membership", map[string]interface{}{
						"subject": sub,
						"object":  obj,
						"action":  action,
						"role":    role,
						"path":    c.Request.URL.Path,
					})
					c.Next()
					return
				}

				// For write/update/delete, require a more specific match
				if strings.Contains(role, action) {
					logger.Info("Authorization granted via specific role membership", map[string]interface{}{
						"subject": sub,
						"object":  obj,
						"action":  action,
						"role":    role,
						"path":    c.Request.URL.Path,
					})
					c.Next()
					return
				}
			}
		}

		// If we got here, check if there was an error in any of the enforcements
		if err != nil {
			logger.Error("Casbin enforcement error", err, map[string]interface{}{
				"subject": sub,
				"object":  obj,
				"action":  action,
				"path":    c.Request.URL.Path,
			})
			setFlashMessage(c, "An error occurred. Please try again later.")
			c.Redirect(http.StatusSeeOther, "/")
			c.Abort()
			return
		}

		// Log the authorization failure
		logger.Info("Authorization denied", map[string]interface{}{
			"subject": sub,
			"object":  obj,
			"action":  action,
			"path":    c.Request.URL.Path,
		})
		setFlashMessage(c, "You do not have authorization for that resource")
		c.Redirect(http.StatusSeeOther, "/")
		c.Abort()
		return
	}
}

// GetEnforcer returns the internal casbin enforcer for direct access (debugging purposes only)
func (ca *CasbinAuth) GetEnforcer() *casbin.Enforcer {
	return ca.enforcer
}

// isFeaturePublic checks if a feature/resource has public access enabled
func (ca *CasbinAuth) isFeaturePublic(resource string) bool {
	// If the resource name represents a potential feature, check if it has public access
	db := ca.enforcer.GetAdapter().(*models.CasbinDBAdapter).GetDB()
	if db == nil {
		return false
	}

	// Check if there's a feature flag with this name and public access
	var flag models.FeatureFlag
	err := db.Where("name = ? AND enabled = ? AND public_access = ?", resource, true, true).First(&flag).Error
	if err != nil {
		return false
	}

	return true
}
