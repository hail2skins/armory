package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/admin/caliber"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
)

// AdminCaliberController handles admin caliber routes
type AdminCaliberController struct {
	db database.Service
}

// NewAdminCaliberController creates a new admin caliber controller
func NewAdminCaliberController(db database.Service) *AdminCaliberController {
	return &AdminCaliberController{
		db: db,
	}
}

// getAdminCaliberDataFromContext gets admin data from context
func getAdminCaliberDataFromContext(ctx *gin.Context, title string, currentPath string) *data.AdminData {
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

// Index lists all calibers
func (c *AdminCaliberController) Index(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminCaliberDataFromContext(ctx, "Calibers", ctx.Request.URL.Path)

	// Get all calibers
	calibers, err := c.db.FindAllCalibers()
	if err != nil {
		// Handle error
		adminData = adminData.WithError("Failed to retrieve calibers")
		// Render error page
		ctx.Writer.WriteHeader(http.StatusInternalServerError)
		caliber.Index(adminData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Render the template with the calibers
	caliber.Index(adminData.WithCalibers(calibers)).Render(ctx.Request.Context(), ctx.Writer)
}

// New shows the form to create a new caliber
func (c *AdminCaliberController) New(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminCaliberDataFromContext(ctx, "New Caliber", ctx.Request.URL.Path)

	caliber.New(adminData).Render(ctx.Request.Context(), ctx.Writer)
}

// Create creates a new caliber
func (c *AdminCaliberController) Create(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminCaliberDataFromContext(ctx, "Create Caliber", ctx.Request.URL.Path)

	// Get form values
	caliberName := ctx.PostForm("caliber")
	nickname := ctx.PostForm("nickname")

	// Validate required fields
	if caliberName == "" {
		component := caliber.New(adminData.WithError("Caliber name is required"))
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Create the caliber
	cal := models.Caliber{
		Caliber:  caliberName,
		Nickname: nickname,
	}

	if err := c.db.CreateCaliber(&cal); err != nil {
		component := caliber.New(adminData.WithError("Failed to create caliber"))
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Redirect to the index page with a success message
	ctx.Redirect(http.StatusSeeOther, "/admin/calibers?success=Caliber created successfully")
}

// Show shows a caliber
func (c *AdminCaliberController) Show(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminCaliberDataFromContext(ctx, "Caliber Details", ctx.Request.URL.Path)

	// Get the caliber ID from the URL
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.HTML(http.StatusBadRequest, "error.templ", adminData.
			WithError("Invalid caliber ID"))
		return
	}

	// Get the caliber
	cal, err := c.db.FindCaliberByID(uint(id))
	if err != nil {
		ctx.HTML(http.StatusNotFound, "error.templ", adminData.
			WithError("Caliber not found"))
		return
	}

	// Render the template with the caliber
	component := caliber.Show(adminData.WithCaliber(cal))
	component.Render(ctx.Request.Context(), ctx.Writer)
}

// Edit shows the form to edit a caliber
func (c *AdminCaliberController) Edit(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminCaliberDataFromContext(ctx, "Edit Caliber", ctx.Request.URL.Path)

	// Get the caliber ID from the URL
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.HTML(http.StatusBadRequest, "error.templ", adminData.
			WithError("Invalid caliber ID"))
		return
	}

	// Get the caliber
	cal, err := c.db.FindCaliberByID(uint(id))
	if err != nil {
		ctx.HTML(http.StatusNotFound, "error.templ", adminData.
			WithError("Caliber not found"))
		return
	}

	// Render the template with the caliber
	component := caliber.Edit(adminData.WithCaliber(cal))
	component.Render(ctx.Request.Context(), ctx.Writer)
}

// Update updates a caliber
func (c *AdminCaliberController) Update(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminCaliberDataFromContext(ctx, "Update Caliber", ctx.Request.URL.Path)

	// Get the caliber ID from the URL
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.HTML(http.StatusBadRequest, "error.templ", adminData.
			WithError("Invalid caliber ID"))
		return
	}

	// Get the caliber
	cal, err := c.db.FindCaliberByID(uint(id))
	if err != nil {
		ctx.HTML(http.StatusNotFound, "error.templ", adminData.
			WithError("Caliber not found"))
		return
	}

	// Get form values
	caliberName := ctx.PostForm("caliber")
	nickname := ctx.PostForm("nickname")

	// Validate required fields
	if caliberName == "" {
		component := caliber.Edit(adminData.
			WithCaliber(cal).
			WithError("Caliber name is required"))
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Update the caliber
	cal.Caliber = caliberName
	cal.Nickname = nickname

	if err := c.db.UpdateCaliber(cal); err != nil {
		component := caliber.Edit(adminData.
			WithCaliber(cal).
			WithError("Failed to update caliber"))
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Redirect to the show page with a success message
	ctx.Redirect(http.StatusSeeOther, "/admin/calibers/"+strconv.FormatUint(id, 10)+"?success=Caliber updated successfully")
}

// Delete deletes a caliber
func (c *AdminCaliberController) Delete(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminCaliberDataFromContext(ctx, "Delete Caliber", ctx.Request.URL.Path)

	// Get the caliber ID from the URL
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.HTML(http.StatusBadRequest, "error.templ", adminData.WithError("Invalid caliber ID"))
		return
	}

	// Delete the caliber
	err = c.db.DeleteCaliber(uint(id))
	if err != nil {
		ctx.HTML(http.StatusInternalServerError, "error.templ", adminData.WithError("Failed to delete caliber"))
		return
	}

	// Redirect to the calibers page with success message
	ctx.Redirect(http.StatusSeeOther, "/admin/calibers?success=Caliber deleted successfully")
}
