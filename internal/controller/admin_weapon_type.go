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

// Index lists all weapon types
func (c *AdminWeaponTypeController) Index(ctx *gin.Context) {
	// Create admin data
	adminData := data.NewAdminData().
		WithTitle("Weapon Types")

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
	// Create admin data
	adminData := data.NewAdminData().
		WithTitle("New Weapon Type")

	weapon_type.New(adminData).Render(ctx.Request.Context(), ctx.Writer)
}

// Create creates a new weapon type
func (c *AdminWeaponTypeController) Create(ctx *gin.Context) {
	// Create admin data
	adminData := data.NewAdminData().
		WithTitle("New Weapon Type")

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
	// Create admin data
	adminData := data.NewAdminData().
		WithTitle("Weapon Type Details")

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
	// Create admin data
	adminData := data.NewAdminData().
		WithTitle("Edit Weapon Type")

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
	// Create admin data
	adminData := data.NewAdminData().
		WithTitle("Edit Weapon Type")

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
	// Get the weapon type ID from the URL
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		// Redirect to the index page with an error message
		ctx.Redirect(http.StatusSeeOther, "/admin/weapon_types?error=Invalid weapon type ID")
		return
	}

	// Delete the weapon type
	if err := c.db.DeleteWeaponType(uint(id)); err != nil {
		// Redirect to the index page with an error message
		ctx.Redirect(http.StatusSeeOther, "/admin/weapon_types?error=Failed to delete weapon type")
		return
	}

	// Redirect to the index page with a success message
	ctx.Redirect(http.StatusSeeOther, "/admin/weapon_types?success=Weapon type deleted successfully")
}
