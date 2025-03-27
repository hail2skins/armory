package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	featureFlagViews "github.com/hail2skins/armory/cmd/web/views/admin/permissions/feature_flags"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
)

// AdminFeatureFlagsController handles feature flag management
type AdminFeatureFlagsController struct {
	db database.Service
}

// NewAdminFeatureFlagsController creates a new admin feature flags controller
func NewAdminFeatureFlagsController(db database.Service) *AdminFeatureFlagsController {
	return &AdminFeatureFlagsController{
		db: db,
	}
}

// Helper to set a flash message
func setFlash(ctx *gin.Context, message string) {
	session := sessions.Default(ctx)
	session.AddFlash(message)
	session.Save()
}

// Helper to get available roles
func (c *AdminFeatureFlagsController) getAvailableRoles() ([]string, error) {
	gormDB := c.db.GetDB()
	if gormDB == nil {
		return []string{}, nil // Return empty slice if db is nil (for testing)
	}

	adapter := models.NewCasbinDBAdapter(gormDB)
	enforcer, err := models.GetEnforcer(adapter)
	if err != nil {
		return nil, err
	}

	return models.GetAllRoles(enforcer), nil
}

// Index handles GET /admin/permissions/feature-flags
func (c *AdminFeatureFlagsController) Index(ctx *gin.Context) {
	// Get all feature flags
	flags, err := c.db.FindAllFeatureFlags()
	if err != nil {
		viewData := data.NewViewData("Feature Flags", ctx)
		viewData.ErrorMsg = "Error fetching feature flags: " + err.Error()
		featureFlagViews.Index(viewData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Create view data
	viewData := data.NewViewData("Feature Flags", ctx)
	flagsData := &data.FeatureFlagsViewData{
		FeatureFlags: flags,
		FlagRoles:    make(map[uint][]string),
	}

	// Get roles for each flag
	for _, flag := range flags {
		var roles []string
		for _, role := range flag.Roles {
			roles = append(roles, role.Role)
		}
		flagsData.FlagRoles[flag.ID] = roles
	}

	viewData.Data = flagsData
	featureFlagViews.Index(viewData).Render(ctx.Request.Context(), ctx.Writer)
}

// Create handles GET /admin/permissions/feature-flags/create
func (c *AdminFeatureFlagsController) Create(ctx *gin.Context) {
	// Get available roles from context (set by middleware) or fetch from DB
	var availableRoles []string
	rolesInterface, exists := ctx.Get("available_roles")
	if exists {
		if roles, ok := rolesInterface.([]string); ok {
			availableRoles = roles
		}
	} else {
		// Fetch roles from DB
		roles, err := c.getAvailableRoles()
		if err != nil {
			viewData := data.NewViewData("Create Feature Flag", ctx)
			viewData.ErrorMsg = "Error fetching roles: " + err.Error()
			featureFlagViews.Create(viewData).Render(ctx.Request.Context(), ctx.Writer)
			return
		}
		availableRoles = roles
	}

	// Create form data
	formData := &data.FeatureFlagFormData{
		FeatureFlag:    nil, // New flag
		AvailableRoles: availableRoles,
		AssignedRoles:  []string{},
	}

	// Create view data
	viewData := data.NewViewData("Create Feature Flag", ctx)
	viewData.Data = formData
	featureFlagViews.Create(viewData).Render(ctx.Request.Context(), ctx.Writer)
}

// Store handles POST /admin/permissions/feature-flags/create
func (c *AdminFeatureFlagsController) Store(ctx *gin.Context) {
	// Get form data
	name := ctx.PostForm("name")
	description := ctx.PostForm("description")
	enabledStr := ctx.PostForm("enabled")
	roles := ctx.PostFormArray("roles")

	// Create the feature flag
	flag := &models.FeatureFlag{
		Name:        name,
		Description: description,
		Enabled:     enabledStr == "true" || enabledStr == "on" || enabledStr == "1",
	}

	// Save to database
	if err := c.db.CreateFeatureFlag(flag); err != nil {
		// Get available roles
		availableRoles, _ := c.getAvailableRoles()

		// Create form data for re-display
		formData := &data.FeatureFlagFormData{
			FeatureFlag:    flag,
			AvailableRoles: availableRoles,
			AssignedRoles:  roles,
		}

		// Create view data with error message
		viewData := data.NewViewData("Create Feature Flag", ctx)
		viewData.ErrorMsg = "Error creating feature flag: " + err.Error()
		viewData.Data = formData
		featureFlagViews.Create(viewData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Add roles if any selected
	for _, role := range roles {
		if err := c.db.AddRoleToFeatureFlag(flag.ID, role); err != nil {
			// Just log the error but continue
			setFlash(ctx, fmt.Sprintf("Warning: Could not add role %s: %s", role, err.Error()))
		}
	}

	// Set success flash and redirect
	setFlash(ctx, "Feature flag created successfully")
	ctx.Redirect(http.StatusFound, "/admin/permissions/feature-flags")
}

// Edit handles GET /admin/permissions/feature-flags/edit/:id
func (c *AdminFeatureFlagsController) Edit(ctx *gin.Context) {
	// Get flag ID from URL
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		viewData := data.NewViewData("Edit Feature Flag", ctx)
		viewData.ErrorMsg = "Invalid feature flag ID"
		featureFlagViews.Edit(viewData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Get feature flag from database
	flag, err := c.db.FindFeatureFlagByID(uint(id))
	if err != nil {
		viewData := data.NewViewData("Edit Feature Flag", ctx)
		viewData.ErrorMsg = "Error fetching feature flag: " + err.Error()
		featureFlagViews.Edit(viewData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Get available roles
	var availableRoles []string
	rolesInterface, exists := ctx.Get("available_roles")
	if exists {
		if roles, ok := rolesInterface.([]string); ok {
			availableRoles = roles
		}
	} else {
		// Fetch roles from DB
		roles, err := c.getAvailableRoles()
		if err != nil {
			viewData := data.NewViewData("Edit Feature Flag", ctx)
			viewData.ErrorMsg = "Error fetching roles: " + err.Error()
			featureFlagViews.Edit(viewData).Render(ctx.Request.Context(), ctx.Writer)
			return
		}
		availableRoles = roles
	}

	// Get assigned roles
	assignedRoles := make([]string, 0, len(flag.Roles))
	for _, role := range flag.Roles {
		assignedRoles = append(assignedRoles, role.Role)
	}

	// Create form data
	formData := &data.FeatureFlagFormData{
		FeatureFlag:    flag,
		AvailableRoles: availableRoles,
		AssignedRoles:  assignedRoles,
	}

	// Create view data
	viewData := data.NewViewData("Edit Feature Flag", ctx)
	viewData.Data = formData
	featureFlagViews.Edit(viewData).Render(ctx.Request.Context(), ctx.Writer)
}

// Update handles POST /admin/permissions/feature-flags/edit/:id
func (c *AdminFeatureFlagsController) Update(ctx *gin.Context) {
	// Get flag ID from URL
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		viewData := data.NewViewData("Edit Feature Flag", ctx)
		viewData.ErrorMsg = "Invalid feature flag ID"
		featureFlagViews.Edit(viewData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Get feature flag from database
	flag, err := c.db.FindFeatureFlagByID(uint(id))
	if err != nil {
		viewData := data.NewViewData("Edit Feature Flag", ctx)
		viewData.ErrorMsg = "Error fetching feature flag: " + err.Error()
		featureFlagViews.Edit(viewData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Update flag properties
	flag.Name = ctx.PostForm("name")
	flag.Description = ctx.PostForm("description")
	enabledStr := ctx.PostForm("enabled")
	flag.Enabled = enabledStr == "true" || enabledStr == "on" || enabledStr == "1"

	// Save to database
	if err := c.db.UpdateFeatureFlag(flag); err != nil {
		viewData := data.NewViewData("Edit Feature Flag", ctx)
		viewData.ErrorMsg = "Error updating feature flag: " + err.Error()

		// Get roles and recreate form data
		availableRoles, _ := c.getAvailableRoles()
		assignedRoles := make([]string, 0, len(flag.Roles))
		for _, role := range flag.Roles {
			assignedRoles = append(assignedRoles, role.Role)
		}

		formData := &data.FeatureFlagFormData{
			FeatureFlag:    flag,
			AvailableRoles: availableRoles,
			AssignedRoles:  assignedRoles,
		}

		viewData.Data = formData
		featureFlagViews.Edit(viewData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Set success flash and redirect
	setFlash(ctx, "Feature flag updated successfully")
	ctx.Redirect(http.StatusFound, "/admin/permissions/feature-flags")
}

// Delete handles POST /admin/permissions/feature-flags/delete/:id
func (c *AdminFeatureFlagsController) Delete(ctx *gin.Context) {
	// Get flag ID from URL
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		setFlash(ctx, "Invalid feature flag ID")
		ctx.Redirect(http.StatusFound, "/admin/permissions/feature-flags")
		return
	}

	// Delete from database
	if err := c.db.DeleteFeatureFlag(uint(id)); err != nil {
		setFlash(ctx, "Error deleting feature flag: "+err.Error())
		ctx.Redirect(http.StatusFound, "/admin/permissions/feature-flags")
		return
	}

	// Set success flash and redirect
	setFlash(ctx, "Feature flag deleted successfully")
	ctx.Redirect(http.StatusFound, "/admin/permissions/feature-flags")
}

// AddRole handles POST /admin/permissions/feature-flags/:id/roles
func (c *AdminFeatureFlagsController) AddRole(ctx *gin.Context) {
	// Get flag ID from URL
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		setFlash(ctx, "Invalid feature flag ID")
		ctx.Redirect(http.StatusFound, "/admin/permissions/feature-flags")
		return
	}

	// Get role from form
	role := ctx.PostForm("role")
	if role == "" {
		setFlash(ctx, "Role is required")
		ctx.Redirect(http.StatusFound, fmt.Sprintf("/admin/permissions/feature-flags/edit/%s", idStr))
		return
	}

	// Add role to feature flag
	if err := c.db.AddRoleToFeatureFlag(uint(id), role); err != nil {
		setFlash(ctx, "Error adding role: "+err.Error())
		ctx.Redirect(http.StatusFound, fmt.Sprintf("/admin/permissions/feature-flags/edit/%s", idStr))
		return
	}

	// Set success flash and redirect
	setFlash(ctx, fmt.Sprintf("Role '%s' added successfully", role))
	ctx.Redirect(http.StatusFound, fmt.Sprintf("/admin/permissions/feature-flags/edit/%s", idStr))
}

// RemoveRole handles POST /admin/permissions/feature-flags/:id/roles/remove
func (c *AdminFeatureFlagsController) RemoveRole(ctx *gin.Context) {
	// Get flag ID from URL
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		setFlash(ctx, "Invalid feature flag ID")
		ctx.Redirect(http.StatusFound, "/admin/permissions/feature-flags")
		return
	}

	// Get role from form
	role := ctx.PostForm("role")
	if role == "" {
		setFlash(ctx, "Role is required")
		ctx.Redirect(http.StatusFound, fmt.Sprintf("/admin/permissions/feature-flags/edit/%s", idStr))
		return
	}

	// Remove role from feature flag
	if err := c.db.RemoveRoleFromFeatureFlag(uint(id), role); err != nil {
		setFlash(ctx, "Error removing role: "+err.Error())
		ctx.Redirect(http.StatusFound, fmt.Sprintf("/admin/permissions/feature-flags/edit/%s", idStr))
		return
	}

	// Set success flash and redirect
	setFlash(ctx, fmt.Sprintf("Role '%s' removed successfully", role))
	ctx.Redirect(http.StatusFound, fmt.Sprintf("/admin/permissions/feature-flags/edit/%s", idStr))
}
