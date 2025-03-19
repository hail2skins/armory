package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/admin/weapon_type"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
)

// AdminWeaponTypeController handles admin weapon type routes
type AdminWeaponTypeController struct {
	db database.Service
}

// NewAdminWeaponTypeController creates a new admin weapon type controller
func NewAdminWeaponTypeController(db database.Service) *AdminWeaponTypeController {
	return &AdminWeaponTypeController{
		db: db,
	}
}

// getAdminWeaponTypeDataFromContext gets admin data from context
func getAdminWeaponTypeDataFromContext(ctx *gin.Context, title string, currentPath string) *data.AdminData {
	// Get admin data from context
	adminDataInterface, exists := ctx.Get("admin_data")
	if exists && adminDataInterface != nil {
		if adminData, ok := adminDataInterface.(*data.AdminData); ok {
			// Update the title and current path
			adminData.AuthData = adminData.AuthData.WithTitle(title).WithCurrentPath(currentPath)
			return adminData
		}
	}

	// Get auth data from context
	authDataInterface, exists := ctx.Get("authData")
	if exists && authDataInterface != nil {
		if authData, ok := authDataInterface.(data.AuthData); ok {
			// Set the title and current path
			authData = authData.WithTitle(title).WithCurrentPath(currentPath)

			// Create admin data with auth data
			adminData := data.NewAdminData()
			adminData.AuthData = authData
			return adminData
		}
	}

	// If we couldn't get auth data from context, create a new one
	adminData := data.NewAdminData()
	adminData.AuthData = adminData.AuthData.WithTitle(title).WithCurrentPath(currentPath)
	return adminData
}

// Index lists all weapon types
func (c *AdminWeaponTypeController) Index(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminWeaponTypeDataFromContext(ctx, "Weapon Types", ctx.Request.URL.Path)

	// Get all weapon types
	weaponTypes, err := c.db.FindAllWeaponTypes()
	if err != nil {
		// Handle error
		adminData = adminData.WithError("Failed to retrieve weapon types")
		// Render error page
		ctx.Writer.WriteHeader(http.StatusInternalServerError)
		weapon_type.Index(adminData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Render the template with the weapon types
	weapon_type.Index(adminData.WithWeaponTypes(weaponTypes)).Render(ctx.Request.Context(), ctx.Writer)
}

// New shows the form to create a new weapon type
func (c *AdminWeaponTypeController) New(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminWeaponTypeDataFromContext(ctx, "New Weapon Type", ctx.Request.URL.Path)

	weapon_type.New(adminData).Render(ctx.Request.Context(), ctx.Writer)
}

// Create creates a new weapon type
func (c *AdminWeaponTypeController) Create(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminWeaponTypeDataFromContext(ctx, "Create Weapon Type", ctx.Request.URL.Path)

	// Get form values
	typeName := ctx.PostForm("type")
	nickname := ctx.PostForm("nickname")

	// Validate required fields
	if typeName == "" {
		component := weapon_type.New(adminData.WithError("Type name is required"))
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Create the weapon type
	wt := models.WeaponType{
		Type:     typeName,
		Nickname: nickname,
	}

	if err := c.db.CreateWeaponType(&wt); err != nil {
		component := weapon_type.New(adminData.WithError("Failed to create weapon type"))
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Redirect to the index page with a success message
	ctx.Redirect(http.StatusSeeOther, "/admin/weapon_types?success=Weapon type created successfully")
}

// Show shows a weapon type
func (c *AdminWeaponTypeController) Show(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminWeaponTypeDataFromContext(ctx, "Weapon Type Details", ctx.Request.URL.Path)

	// Get the weapon type ID from the URL
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.HTML(http.StatusBadRequest, "error.templ", adminData.
			WithError("Invalid weapon type ID"))
		return
	}

	// Get the weapon type
	wt, err := c.db.FindWeaponTypeByID(uint(id))
	if err != nil {
		ctx.HTML(http.StatusNotFound, "error.templ", adminData.
			WithError("Weapon type not found"))
		return
	}

	// Render the template with the weapon type
	component := weapon_type.Show(adminData.WithWeaponType(wt))
	component.Render(ctx.Request.Context(), ctx.Writer)
}

// Edit shows the form to edit a weapon type
func (c *AdminWeaponTypeController) Edit(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminWeaponTypeDataFromContext(ctx, "Edit Weapon Type", ctx.Request.URL.Path)

	// Get the weapon type ID from the URL
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.HTML(http.StatusBadRequest, "error.templ", adminData.
			WithError("Invalid weapon type ID"))
		return
	}

	// Get the weapon type
	wt, err := c.db.FindWeaponTypeByID(uint(id))
	if err != nil {
		ctx.HTML(http.StatusNotFound, "error.templ", adminData.
			WithError("Weapon type not found"))
		return
	}

	// Render the template with the weapon type
	component := weapon_type.Edit(adminData.WithWeaponType(wt))
	component.Render(ctx.Request.Context(), ctx.Writer)
}

// Update updates a weapon type
func (c *AdminWeaponTypeController) Update(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminWeaponTypeDataFromContext(ctx, "Update Weapon Type", ctx.Request.URL.Path)

	// Get the weapon type ID from the URL
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.HTML(http.StatusBadRequest, "error.templ", adminData.
			WithError("Invalid weapon type ID"))
		return
	}

	// Get the weapon type
	wt, err := c.db.FindWeaponTypeByID(uint(id))
	if err != nil {
		ctx.HTML(http.StatusNotFound, "error.templ", adminData.
			WithError("Weapon type not found"))
		return
	}

	// Get form values
	typeName := ctx.PostForm("type")
	nickname := ctx.PostForm("nickname")

	// Validate required fields
	if typeName == "" {
		component := weapon_type.Edit(adminData.
			WithWeaponType(wt).
			WithError("Type name is required"))
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Update the weapon type
	wt.Type = typeName
	wt.Nickname = nickname

	if err := c.db.UpdateWeaponType(wt); err != nil {
		component := weapon_type.Edit(adminData.
			WithWeaponType(wt).
			WithError("Failed to update weapon type"))
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Redirect to the show page with a success message
	ctx.Redirect(http.StatusSeeOther, "/admin/weapon_types/"+strconv.FormatUint(id, 10)+"?success=Weapon type updated successfully")
}

// Delete deletes a weapon type
func (c *AdminWeaponTypeController) Delete(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminWeaponTypeDataFromContext(ctx, "Delete Weapon Type", ctx.Request.URL.Path)

	// Get the weapon type ID from the URL
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.HTML(http.StatusBadRequest, "error.templ", adminData.WithError("Invalid weapon type ID"))
		return
	}

	// Delete the weapon type
	err = c.db.DeleteWeaponType(uint(id))
	if err != nil {
		ctx.HTML(http.StatusInternalServerError, "error.templ", adminData.WithError("Failed to delete weapon type"))
		return
	}

	// Redirect to the weapon types page with success message
	ctx.Redirect(http.StatusSeeOther, "/admin/weapon_types?success=Weapon type deleted successfully")
}
