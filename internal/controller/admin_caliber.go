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

// Index lists all calibers
func (c *AdminCaliberController) Index(ctx *gin.Context) {
	// Create admin data
	adminData := data.NewAdminData().
		WithTitle("Calibers")

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
	// Create admin data
	adminData := data.NewAdminData().
		WithTitle("New Caliber")

	caliber.New(adminData).Render(ctx.Request.Context(), ctx.Writer)
}

// Create creates a new caliber
func (c *AdminCaliberController) Create(ctx *gin.Context) {
	// Create admin data
	adminData := data.NewAdminData().
		WithTitle("New Caliber")

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
	// Create admin data
	adminData := data.NewAdminData().
		WithTitle("Caliber Details")

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
	// Create admin data
	adminData := data.NewAdminData().
		WithTitle("Edit Caliber")

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
	// Create admin data
	adminData := data.NewAdminData().
		WithTitle("Edit Caliber")

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
	// Get the caliber ID from the URL
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		// Redirect to the index page with an error message
		ctx.Redirect(http.StatusSeeOther, "/admin/calibers?error=Invalid caliber ID")
		return
	}

	// Delete the caliber
	if err := c.db.DeleteCaliber(uint(id)); err != nil {
		// Redirect to the index page with an error message
		ctx.Redirect(http.StatusSeeOther, "/admin/calibers?error=Failed to delete caliber")
		return
	}

	// Redirect to the index page with a success message
	ctx.Redirect(http.StatusSeeOther, "/admin/calibers?success=Caliber deleted successfully")
}
