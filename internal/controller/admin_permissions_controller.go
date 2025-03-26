package controller

import (
	"net/http"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	permsViews "github.com/hail2skins/armory/cmd/web/views/admin/permissions"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
)

// AdminPermissionsController handles admin permission management
type AdminPermissionsController struct {
	db database.Service
}

// NewAdminPermissionsController creates a new admin permissions controller
func NewAdminPermissionsController(db database.Service) *AdminPermissionsController {
	return &AdminPermissionsController{
		db: db,
	}
}

// Helper function to set flash message
func setFlashMessage(ctx *gin.Context, message string) {
	session := sessions.Default(ctx)
	session.AddFlash(message)
	session.Save()
}

// Index handles the admin permissions index page
func (c *AdminPermissionsController) Index(ctx *gin.Context) {
	gormDB := c.db.GetDB()

	// Create view data
	viewData := data.NewViewData("Permissions Management", ctx)
	permissionsData := &data.PermissionsViewData{
		RolePermissions: make(map[string][]data.Permission),
		RoleUsers:       make(map[string][]string),
	}

	// Get adapter for the RBAC service
	adapter := models.NewCasbinDBAdapter(gormDB)
	enforcer, err := models.GetEnforcer(adapter)
	if err != nil {
		viewData.ErrorMsg = "Error initializing RBAC service"
		// Use templ renderer instead of HTML
		permsViews.Index(viewData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Get all roles from the enforcer
	roles := models.GetAllRoles(enforcer)
	permissionsData.Roles = roles

	// Get permissions for each role
	for _, role := range roles {
		permsForRole := models.GetPermissionsForRole(enforcer, role)

		perms := make([]data.Permission, 0)
		for _, p := range permsForRole {
			perm := data.Permission{
				Role:     role,
				Resource: p.Resource,
				Action:   p.Action,
			}
			perms = append(perms, perm)
			permissionsData.Permissions = append(permissionsData.Permissions, perm)
		}
		permissionsData.RolePermissions[role] = perms
	}

	// Get user assignments for each role
	for _, role := range roles {
		users := models.GetUsersForRole(enforcer, role)
		permissionsData.RoleUsers[role] = users

		for _, user := range users {
			permissionsData.UserRoles = append(permissionsData.UserRoles, data.UserRole{
				User: user,
				Role: role,
			})
		}
	}

	viewData.Data = permissionsData
	// Use templ renderer instead of HTML
	permsViews.Index(viewData).Render(ctx.Request.Context(), ctx.Writer)
}

// NewRole handles displaying the form to create a new role
func (c *AdminPermissionsController) NewRole(ctx *gin.Context) {
	// Get resources that can be assigned permissions
	resources := []string{
		"manufacturers",
		"calibers",
		"weapon_types",
		"promotions",
		"owners",
		"stripe_security",
		"dashboard",
		"users",
		"payments",
		"guns",
		"permissions",
		"*", // Wildcard for all resources
	}

	// Get actions that can be assigned
	actions := []string{
		"read",
		"write",
		"update",
		"delete",
		"*", // Wildcard for all actions
	}

	// Create the page data
	pageData := data.AdminNewRoleData{
		Resources: resources,
		Actions:   actions,
	}

	// Create view data
	viewData := data.NewViewData("New Role", ctx)
	viewData.Data = pageData

	// Update to use proper templ rendering
	permsViews.CreateRole(viewData).Render(ctx.Request.Context(), ctx.Writer)
}

// CreateRole displays the create role form
func (c *AdminPermissionsController) CreateRole(ctx *gin.Context) {
	// Create view data
	viewData := data.NewViewData("Create Role", ctx)

	// Create form data
	formData := &data.CreateRoleViewData{
		Resources: []string{
			"manufacturers", "calibers", "weapon_types", "promotions", "permissions",
		},
		Actions: []string{
			"read", "create", "update", "delete", "manage",
		},
	}

	viewData.Data = formData
	// Use templ renderer
	permsViews.CreateRole(viewData).Render(ctx.Request.Context(), ctx.Writer)
}

// StoreRole handles the role creation form submission
func (c *AdminPermissionsController) StoreRole(ctx *gin.Context) {
	gormDB := c.db.GetDB()

	// Create view data for potential error redisplay
	viewData := data.NewViewData("Create Role", ctx)

	// Get form data
	roleName := ctx.PostForm("role")
	permissionsList := ctx.PostFormArray("permissions")

	if roleName == "" {
		viewData.ErrorMsg = "Role name is required"
		formData := &data.CreateRoleViewData{
			Resources: []string{
				"manufacturers", "calibers", "weapon_types", "promotions", "permissions",
			},
			Actions: []string{
				"read", "create", "update", "delete", "manage",
			},
		}
		viewData.Data = formData
		// Use templ renderer
		permsViews.CreateRole(viewData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Get adapter for the RBAC service
	adapter := models.NewCasbinDBAdapter(gormDB)
	enforcer, err := models.GetEnforcer(adapter)
	if err != nil {
		viewData.ErrorMsg = "Error initializing RBAC service"
		// Use templ renderer
		permsViews.CreateRole(viewData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Check if role already exists
	exists := models.RoleExists(enforcer, roleName)
	if exists {
		viewData.ErrorMsg = "Role already exists"
		formData := &data.CreateRoleViewData{
			Resources: []string{
				"manufacturers", "calibers", "weapon_types", "promotions", "permissions",
			},
			Actions: []string{
				"read", "create", "update", "delete", "manage",
			},
		}
		viewData.Data = formData
		// Use templ renderer
		permsViews.CreateRole(viewData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Create role with permissions
	for _, perm := range permissionsList {
		parts := strings.Split(perm, ":")
		if len(parts) != 2 {
			continue
		}
		resource := parts[0]
		action := parts[1]

		err := models.AddPermissionForRole(enforcer, roleName, resource, action)
		if err != nil {
			viewData.ErrorMsg = "Error adding permission: " + err.Error()
			// Use templ renderer
			permsViews.CreateRole(viewData).Render(ctx.Request.Context(), ctx.Writer)
			return
		}
	}

	// Set success flash message
	ctx.Set("flash", "Role created successfully")

	// Redirect to permissions index
	ctx.Redirect(http.StatusSeeOther, "/admin/permissions")
}

// EditRole handles displaying the form to edit a role
func (c *AdminPermissionsController) EditRole(ctx *gin.Context) {
	gormDB := c.db.GetDB()

	// Get role name from URL parameter
	roleName := ctx.Param("role")

	// Create view data
	viewData := data.NewViewData("Edit Role: "+roleName, ctx)

	// Get adapter for the RBAC service
	adapter := models.NewCasbinDBAdapter(gormDB)
	enforcer, err := models.GetEnforcer(adapter)
	if err != nil {
		viewData.ErrorMsg = "Error initializing RBAC service"
		// Use templ renderer
		ctx.HTML(http.StatusInternalServerError, "error.templ", viewData)
		return
	}

	// Check if role exists
	exists := models.RoleExists(enforcer, roleName)
	if !exists {
		viewData.ErrorMsg = "Role does not exist"
		ctx.HTML(http.StatusNotFound, "error.templ", viewData)
		return
	}

	// Get permissions for the role
	perms := models.GetPermissionsForRole(enforcer, roleName)

	// Create form data
	formData := &data.EditRoleViewData{
		Role: roleName,
		Resources: []string{
			"manufacturers", "calibers", "weapon_types", "promotions", "permissions",
		},
		Actions: []string{
			"read", "create", "update", "delete", "manage",
		},
		SelectedPermissions: make(map[string]bool),
	}

	// Mark selected permissions
	for _, perm := range perms {
		key := perm.Resource + ":" + perm.Action
		formData.SelectedPermissions[key] = true
	}

	viewData.Data = formData
	// Use templ renderer
	permsViews.EditRole(viewData).Render(ctx.Request.Context(), ctx.Writer)
}

// UpdateRole handles updating a role
func (c *AdminPermissionsController) UpdateRole(ctx *gin.Context) {
	gormDB := c.db.GetDB()

	// Get form data
	roleName := ctx.PostForm("role")
	permsList := ctx.PostFormArray("permissions")

	// Create view data for potential error redisplay
	viewData := data.NewViewData("Edit Role: "+roleName, ctx)

	if roleName == "" {
		viewData.ErrorMsg = "Role name is required"
		ctx.HTML(http.StatusBadRequest, "error.templ", viewData)
		return
	}

	// Get adapter for the RBAC service
	adapter := models.NewCasbinDBAdapter(gormDB)
	enforcer, err := models.GetEnforcer(adapter)
	if err != nil {
		viewData.ErrorMsg = "Error initializing RBAC service"
		ctx.HTML(http.StatusInternalServerError, "error.templ", viewData)
		return
	}

	// Check if role exists
	exists := models.RoleExists(enforcer, roleName)
	if !exists {
		viewData.ErrorMsg = "Role does not exist"
		ctx.HTML(http.StatusNotFound, "error.templ", viewData)
		return
	}

	// Get existing permissions for the role
	existingPermissions := models.GetPermissionsForRole(enforcer, roleName)

	// Remove all existing permissions
	for _, perm := range existingPermissions {
		err := models.RemovePermissionForRole(enforcer, roleName, perm.Resource, perm.Action)
		if err != nil {
			viewData.ErrorMsg = "Error removing permission: " + err.Error()
			ctx.HTML(http.StatusInternalServerError, "error.templ", viewData)
			return
		}
	}

	// Add new permissions
	for _, perm := range permsList {
		parts := strings.Split(perm, ":")
		if len(parts) != 2 {
			continue
		}
		resource := parts[0]
		action := parts[1]

		err := models.AddPermissionForRole(enforcer, roleName, resource, action)
		if err != nil {
			viewData.ErrorMsg = "Error adding permission: " + err.Error()
			ctx.HTML(http.StatusInternalServerError, "error.templ", viewData)
			return
		}
	}

	// Set success flash message
	ctx.Set("flash", "Role updated successfully")

	// Redirect to permissions index
	ctx.Redirect(http.StatusSeeOther, "/admin/permissions")
}

// DeleteRole handles deleting a role
func (c *AdminPermissionsController) DeleteRole(ctx *gin.Context) {
	gormDB := c.db.GetDB()

	// Get role name from URL parameter
	roleName := ctx.Param("role")

	// Prevent deletion of built-in roles
	if roleName == "admin" || roleName == "editor" || roleName == "viewer" {
		// Set error flash message
		ctx.Set("flash", "Cannot delete built-in role: "+roleName)

		// Redirect to permissions index
		ctx.Redirect(http.StatusSeeOther, "/admin/permissions")
		return
	}

	// Get adapter for the RBAC service
	adapter := models.NewCasbinDBAdapter(gormDB)
	enforcer, err := models.GetEnforcer(adapter)
	if err != nil {
		// Set error flash message
		ctx.Set("flash", "Error initializing RBAC service")

		// Redirect to permissions index
		ctx.Redirect(http.StatusSeeOther, "/admin/permissions")
		return
	}

	// Delete the role and its permissions
	err = models.DeleteRoleWithEnforcer(enforcer, roleName)
	if err != nil {
		// Set error flash message
		ctx.Set("flash", "Error deleting role: "+err.Error())

		// Redirect to permissions index
		ctx.Redirect(http.StatusSeeOther, "/admin/permissions")
		return
	}

	// Set success flash message
	ctx.Set("flash", "Role deleted successfully")

	// Redirect to permissions index
	ctx.Redirect(http.StatusSeeOther, "/admin/permissions")
}

// AssignRole handles assigning a role to a user
func (c *AdminPermissionsController) AssignRole(ctx *gin.Context) {
	gormDB := c.db.GetDB()

	// Create view data
	viewData := data.NewViewData("Assign Role", ctx)

	// Get adapter for the RBAC service
	adapter := models.NewCasbinDBAdapter(gormDB)
	enforcer, err := models.GetEnforcer(adapter)
	if err != nil {
		viewData.ErrorMsg = "Error initializing RBAC service"
		ctx.HTML(http.StatusInternalServerError, "error.templ", viewData)
		return
	}

	// Get all roles
	roles := models.GetAllRoles(enforcer)

	// Get all users for the dropdown
	var users []database.User
	result := gormDB.Find(&users)
	if result.Error != nil {
		viewData.ErrorMsg = "Error retrieving users"
		ctx.HTML(http.StatusInternalServerError, "error.templ", viewData)
		return
	}

	// Create form data
	formData := &data.AssignRoleViewData{
		Roles: roles,
		Users: users,
	}

	viewData.Data = formData
	// Use templ renderer
	permsViews.AssignRole(viewData).Render(ctx.Request.Context(), ctx.Writer)
}

// StoreAssignRole handles the role assignment form submission
func (c *AdminPermissionsController) StoreAssignRole(ctx *gin.Context) {
	gormDB := c.db.GetDB()

	// Get form data
	userID := ctx.PostForm("user_id")
	role := ctx.PostForm("role")

	// Create view data for potential error redisplay
	viewData := data.NewViewData("Assign Role", ctx)

	if userID == "" || role == "" {
		viewData.ErrorMsg = "User and role are required"

		// Get adapter for the RBAC service
		adapter := models.NewCasbinDBAdapter(gormDB)
		enforcer, err := models.GetEnforcer(adapter)
		if err == nil {
			roles := models.GetAllRoles(enforcer)
			if err == nil {
				formData := &data.AssignRoleViewData{
					Roles: roles,
				}
				viewData.Data = formData
			}
		}

		// Use templ renderer
		permsViews.AssignRole(viewData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Need to query the user email from the ID for Casbin
	var user database.User
	if err := gormDB.First(&user, userID).Error; err != nil {
		viewData.ErrorMsg = "User not found"
		permsViews.AssignRole(viewData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Get adapter for the RBAC service
	adapter := models.NewCasbinDBAdapter(gormDB)
	enforcer, err := models.GetEnforcer(adapter)
	if err != nil {
		viewData.ErrorMsg = "Error initializing RBAC service"
		// Use templ renderer
		permsViews.AssignRole(viewData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Check if role exists
	exists := models.RoleExists(enforcer, role)
	if !exists {
		viewData.ErrorMsg = "Role does not exist"
		// Use templ renderer
		permsViews.AssignRole(viewData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Check if user already has this role
	hasRole := models.HasRole(enforcer, user.Email, role)
	if hasRole {
		viewData.ErrorMsg = "User already has this role"
		// Use templ renderer
		permsViews.AssignRole(viewData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Assign role to user
	err = models.AssignRoleToUser(enforcer, user.Email, role)
	if err != nil {
		viewData.ErrorMsg = "Error assigning role: " + err.Error()
		// Use templ renderer
		permsViews.AssignRole(viewData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Set success flash message
	ctx.Set("flash", "Role assigned successfully")

	// Redirect to permissions index
	ctx.Redirect(http.StatusSeeOther, "/admin/permissions")
}

// RemoveUserRole handles removing a role from a user
func (c *AdminPermissionsController) RemoveUserRole(ctx *gin.Context) {
	gormDB := c.db.GetDB()

	// Get form data
	user := ctx.PostForm("user")
	role := ctx.PostForm("role")

	if user == "" || role == "" {
		// Set error flash message
		ctx.Set("flash", "User and role are required")

		// Redirect to permissions index
		ctx.Redirect(http.StatusSeeOther, "/admin/permissions")
		return
	}

	// Prevent removing admin role from the last admin user
	if role == "admin" {
		// Get adapter for the RBAC service
		adapter := models.NewCasbinDBAdapter(gormDB)
		enforcer, err := models.GetEnforcer(adapter)
		if err == nil {
			// Check if this is the last admin
			adminUsers := models.GetUsersForRole(enforcer, "admin")
			if err == nil && len(adminUsers) <= 1 && adminUsers[0] == user {
				// Set error flash message
				ctx.Set("flash", "Cannot remove the last admin user")

				// Redirect to permissions index
				ctx.Redirect(http.StatusSeeOther, "/admin/permissions")
				return
			}
		}
	}

	// Get adapter for the RBAC service
	adapter := models.NewCasbinDBAdapter(gormDB)
	enforcer, err := models.GetEnforcer(adapter)
	if err != nil {
		// Set error flash message
		ctx.Set("flash", "Error initializing RBAC service")

		// Redirect to permissions index
		ctx.Redirect(http.StatusSeeOther, "/admin/permissions")
		return
	}

	// Remove role from user
	err = models.RemoveRoleFromUser(enforcer, user, role)
	if err != nil {
		// Set error flash message
		ctx.Set("flash", "Error removing role: "+err.Error())

		// Redirect to permissions index
		ctx.Redirect(http.StatusSeeOther, "/admin/permissions")
		return
	}

	// Set success flash message
	ctx.Set("flash", "Role removed successfully")

	// Redirect to permissions index
	ctx.Redirect(http.StatusSeeOther, "/admin/permissions")
}

// ImportDefaultPolicies imports the default policies
func (c *AdminPermissionsController) ImportDefaultPolicies(ctx *gin.Context) {
	gormDB := c.db.GetDB()

	// Get adapter for the RBAC service
	adapter := models.NewCasbinDBAdapter(gormDB)
	enforcer, err := models.GetEnforcer(adapter)
	if err != nil {
		// Set error flash message
		ctx.Set("flash", "Error initializing RBAC service")

		// Redirect to permissions index
		ctx.Redirect(http.StatusSeeOther, "/admin/permissions")
		return
	}

	// Clear all existing policies
	err = models.ClearPolicies(enforcer)
	if err != nil {
		// Set error flash message
		ctx.Set("flash", "Error clearing policies: "+err.Error())

		// Redirect to permissions index
		ctx.Redirect(http.StatusSeeOther, "/admin/permissions")
		return
	}

	// Import default policies
	err = models.ImportDefaultPolicies(enforcer)
	if err != nil {
		// Set error flash message
		ctx.Set("flash", "Error importing policies: "+err.Error())

		// Redirect to permissions index
		ctx.Redirect(http.StatusSeeOther, "/admin/permissions")
		return
	}

	// Set success flash message
	ctx.Set("flash", "Default policies imported successfully")

	// Redirect to permissions index
	ctx.Redirect(http.StatusSeeOther, "/admin/permissions")
}
