package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/admin/manufacturer"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
)

// AdminManufacturerController handles admin manufacturer routes
type AdminManufacturerController struct {
	db database.Service
}

// NewAdminManufacturerController creates a new admin manufacturer controller
func NewAdminManufacturerController(db database.Service) *AdminManufacturerController {
	return &AdminManufacturerController{
		db: db,
	}
}

// getAdminDataFromContext gets admin data from context
func getAdminDataFromContext(ctx *gin.Context, title string, currentPath string) *data.AdminData {
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

// Index shows all manufacturers
func (c *AdminManufacturerController) Index(ctx *gin.Context) {
	// Get auth data from context
	adminData := getAdminDataFromContext(ctx, "Manufacturers", ctx.Request.URL.Path)

	// Get success message from query params
	success := ctx.Query("success")
	if success != "" {
		adminData = adminData.WithSuccess(success)
	}

	// Get all manufacturers
	manufacturers, err := c.db.FindAllManufacturers()
	if err != nil {
		// Render the template with an error
		component := manufacturer.Index(adminData.WithError("Failed to load manufacturers"))
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Render the template with manufacturers
	component := manufacturer.Index(adminData.WithManufacturers(manufacturers))
	component.Render(ctx.Request.Context(), ctx.Writer)
}

// New shows the form to create a new manufacturer
func (c *AdminManufacturerController) New(ctx *gin.Context) {
	// Get auth data from context
	adminData := getAdminDataFromContext(ctx, "New Manufacturer", ctx.Request.URL.Path)

	// Render the template
	manufacturer.New(adminData).Render(ctx.Request.Context(), ctx.Writer)
}

// Create creates a new manufacturer
func (c *AdminManufacturerController) Create(ctx *gin.Context) {
	// Get auth data from context
	adminData := getAdminDataFromContext(ctx, "New Manufacturer", ctx.Request.URL.Path)

	// Get form values
	name := ctx.PostForm("name")
	nickname := ctx.PostForm("nickname")
	country := ctx.PostForm("country")

	// Validate required fields
	if name == "" || country == "" {
		component := manufacturer.New(adminData.WithError("Name and country are required"))
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Create the manufacturer
	mfr := models.Manufacturer{
		Name:     name,
		Nickname: nickname,
		Country:  country,
	}

	if err := c.db.CreateManufacturer(&mfr); err != nil {
		component := manufacturer.New(adminData.WithError("Failed to create manufacturer"))
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Redirect to the index page with a success message
	ctx.Redirect(http.StatusSeeOther, "/admin/manufacturers?success=Manufacturer created successfully")
}

// Show shows a manufacturer
func (c *AdminManufacturerController) Show(ctx *gin.Context) {
	// Get auth data from context
	adminData := getAdminDataFromContext(ctx, "Manufacturer Details", ctx.Request.URL.Path)

	// Get the manufacturer ID from the URL
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.HTML(http.StatusBadRequest, "error.templ", adminData.
			WithError("Invalid manufacturer ID"))
		return
	}

	// Get the manufacturer
	mfr, err := c.db.FindManufacturerByID(uint(id))
	if err != nil {
		ctx.HTML(http.StatusNotFound, "error.templ", adminData.
			WithError("Manufacturer not found"))
		return
	}

	// Render the template with the manufacturer
	component := manufacturer.Show(adminData.WithManufacturer(mfr))
	component.Render(ctx.Request.Context(), ctx.Writer)
}

// Edit shows the form to edit a manufacturer
func (c *AdminManufacturerController) Edit(ctx *gin.Context) {
	// Get auth data from context
	adminData := getAdminDataFromContext(ctx, "Edit Manufacturer", ctx.Request.URL.Path)

	// Get the manufacturer ID from the URL
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.HTML(http.StatusBadRequest, "error.templ", adminData.
			WithError("Invalid manufacturer ID"))
		return
	}

	// Get the manufacturer
	mfr, err := c.db.FindManufacturerByID(uint(id))
	if err != nil {
		ctx.HTML(http.StatusNotFound, "error.templ", adminData.
			WithError("Manufacturer not found"))
		return
	}

	// Render the template with the manufacturer
	component := manufacturer.Edit(adminData.WithManufacturer(mfr))
	component.Render(ctx.Request.Context(), ctx.Writer)
}

// Update updates a manufacturer
func (c *AdminManufacturerController) Update(ctx *gin.Context) {
	// Get auth data from context
	adminData := getAdminDataFromContext(ctx, "Edit Manufacturer", ctx.Request.URL.Path)

	// Get the manufacturer ID from the URL
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.HTML(http.StatusBadRequest, "error.templ", adminData.
			WithError("Invalid manufacturer ID"))
		return
	}

	// Get the manufacturer
	mfr, err := c.db.FindManufacturerByID(uint(id))
	if err != nil {
		ctx.HTML(http.StatusNotFound, "error.templ", adminData.
			WithError("Manufacturer not found"))
		return
	}

	// Get form values
	name := ctx.PostForm("name")
	nickname := ctx.PostForm("nickname")
	country := ctx.PostForm("country")

	// Validate required fields
	if name == "" || country == "" {
		component := manufacturer.Edit(adminData.
			WithManufacturer(mfr).
			WithError("Name and country are required"))
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Update the manufacturer
	mfr.Name = name
	mfr.Nickname = nickname
	mfr.Country = country

	if err := c.db.UpdateManufacturer(mfr); err != nil {
		component := manufacturer.Edit(adminData.
			WithManufacturer(mfr).
			WithError("Failed to update manufacturer"))
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Redirect to the index page with a success message
	ctx.Redirect(http.StatusSeeOther, "/admin/manufacturers/"+strconv.FormatUint(id, 10)+"?success=Manufacturer updated successfully")
}

// Delete deletes a manufacturer
func (c *AdminManufacturerController) Delete(ctx *gin.Context) {
	// Get auth data just in case we need to display an error page
	adminData := getAdminDataFromContext(ctx, "Delete Manufacturer", ctx.Request.URL.Path)

	// Get the manufacturer ID from the URL
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		// Render error page instead of redirecting
		ctx.HTML(http.StatusBadRequest, "error.templ", adminData.
			WithError("Invalid manufacturer ID"))
		return
	}

	// Delete the manufacturer
	if err := c.db.DeleteManufacturer(uint(id)); err != nil {
		// Render error page instead of redirecting
		ctx.HTML(http.StatusInternalServerError, "error.templ", adminData.
			WithError("Failed to delete manufacturer"))
		return
	}

	// Redirect to the index page with a success message
	ctx.Redirect(http.StatusSeeOther, "/admin/manufacturers?success=Manufacturer deleted successfully")
}
