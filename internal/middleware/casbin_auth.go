package middleware

import (
	"fmt"
	"net/http"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/logger"
	"github.com/shaj13/go-guardian/v2/auth"
)

// CasbinAuth is the middleware for Casbin-based authorization
type CasbinAuth struct {
	enforcer *casbin.Enforcer
}

// NewCasbinAuth creates a new Casbin authorization middleware
func NewCasbinAuth(modelPath, policyPath string) (*CasbinAuth, error) {
	enforcer, err := casbin.NewEnforcer(modelPath, policyPath)
	if err != nil {
		return nil, err
	}

	return &CasbinAuth{
		enforcer: enforcer,
	}, nil
}

// Authorize returns a middleware that authorizes a user based on Casbin policies
func (ca *CasbinAuth) Authorize(obj string, act ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the current user from the context (set by authentication middleware)
		authInfo, exists := c.Get("auth_info")
		if !exists {
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
			})
			setFlashMessage(c, "An error occurred. Please try again later.")
			c.Redirect(http.StatusSeeOther, "/")
			c.Abort()
			return
		}

		sub := userInfo.GetUserName()

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
			})
			setFlashMessage(c, "You do not have authorization for that resource")
			c.Redirect(http.StatusSeeOther, "/")
			c.Abort()
			return
		}

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
		// Fallback to cookie if setFlash function is not available
		c.SetCookie("flash", message, 3600, "/", "", false, false)
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
	roles, _ := ca.enforcer.GetRolesForUser(user)
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
