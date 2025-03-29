package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/admin/casing" // Import the casing view package
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
	// "github.com/hail2skins/armory/internal/models" // No longer directly needed here
)

// AdminCasingController handles admin casing routes
type AdminCasingController struct {
	db database.Service
}

// NewAdminCasingController creates a new admin casing controller
func NewAdminCasingController(db database.Service) *AdminCasingController {
	return &AdminCasingController{
		db: db,
	}
}

// getAdminCasingDataFromContext gets admin data from context, specifically for Casing views
func getAdminCasingDataFromContext(ctx *gin.Context, title string, currentPath string) *data.AdminData {
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

// Index handles the GET /admin/casings route
func (c *AdminCasingController) Index(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminCasingDataFromContext(ctx, "Casings", ctx.Request.URL.Path)

	// Get success message from query params
	success := ctx.Query("success")
	if success != "" {
		adminData = adminData.WithSuccess(success)
	}

	// Get all casings from the database
	casings, err := c.db.FindAllCasings()
	if err != nil {
		// Render the template with an error message
		adminData = adminData.WithError("Failed to load casings")
		component := casing.Index(adminData)
		ctx.Writer.WriteHeader(http.StatusInternalServerError) // Set appropriate status code
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Add casings to admin data
	adminData = adminData.WithCasings(casings)

	// Render the template with the casings
	component := casing.Index(adminData)
	component.Render(ctx.Request.Context(), ctx.Writer)
}

// New handles the GET /admin/casings/new route
func (c *AdminCasingController) New(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminCasingDataFromContext(ctx, "New Casing", ctx.Request.URL.Path)

	// Render the new casing form template
	component := casing.New(adminData)
	component.Render(ctx.Request.Context(), ctx.Writer)
}

// Create handles the POST /admin/casings route
func (c *AdminCasingController) Create(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminCasingDataFromContext(ctx, "Create Casing", ctx.Request.URL.Path)

	// Get form values
	casingType := ctx.PostForm("type")
	popularityStr := ctx.PostForm("popularity")

	// Debug log
	fmt.Printf("DEBUG: Create casing - Type: %s, Popularity: %s\n", casingType, popularityStr)

	// Validate required fields
	if casingType == "" {
		adminData = adminData.WithError("Casing type is required")
		component := casing.New(adminData)
		ctx.Writer.WriteHeader(http.StatusBadRequest)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Convert popularity to int
	popularity, err := strconv.Atoi(popularityStr)
	if err != nil {
		// Handle error - maybe default to 0 or show error
		popularity = 0 // Defaulting to 0 for simplicity
	}

	// Create the casing model
	cas := models.Casing{
		Type:       casingType,
		Popularity: popularity,
	}

	// Debug log before saving
	fmt.Printf("DEBUG: About to create casing in DB: %+v\n", cas)

	// Attempt to create the casing in the database
	if err := c.db.CreateCasing(&cas); err != nil {
		// Debug log for error
		fmt.Printf("DEBUG: Error creating casing: %v\n", err)

		adminData = adminData.WithError("Failed to create casing: " + err.Error())
		component := casing.New(adminData)
		ctx.Writer.WriteHeader(http.StatusInternalServerError)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Debug log after saving
	fmt.Printf("DEBUG: Casing created successfully with ID: %d\n", cas.ID)

	// Redirect to the index page with a success message
	ctx.Redirect(http.StatusSeeOther, "/admin/casings?success=Casing created successfully")
}

// Show handles the GET /admin/casings/:id route
func (c *AdminCasingController) Show(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminCasingDataFromContext(ctx, "Show Casing", ctx.Request.URL.Path)

	// Get casing ID from the URL parameter
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		// Handle invalid ID format
		adminData = adminData.WithError("Invalid casing ID format")
		component := casing.Show(adminData) // Need a Show view component
		ctx.Writer.WriteHeader(http.StatusBadRequest)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Fetch the casing from the database
	cas, err := c.db.FindCasingByID(uint(id))
	if err != nil {
		// Handle case where casing is not found or other DB error
		adminData = adminData.WithError("Casing not found or database error")
		component := casing.Show(adminData) // Need a Show view component
		ctx.Writer.WriteHeader(http.StatusNotFound)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Add the found casing to the admin data
	adminData = adminData.WithCasing(cas) // Need to add WithCasing to AdminData

	// Render the Show view template
	component := casing.Show(adminData) // Need a Show view component
	component.Render(ctx.Request.Context(), ctx.Writer)
}

// Edit handles the GET /admin/casings/:id/edit route
func (c *AdminCasingController) Edit(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminCasingDataFromContext(ctx, "Edit Casing", ctx.Request.URL.Path)

	// Get casing ID from the URL parameter
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		// Handle invalid ID format
		adminData = adminData.WithError("Invalid casing ID format")
		component := casing.Edit(adminData) // Need an Edit view component
		ctx.Writer.WriteHeader(http.StatusBadRequest)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Fetch the casing from the database
	cas, err := c.db.FindCasingByID(uint(id))
	if err != nil {
		// Handle case where casing is not found
		adminData = adminData.WithError("Casing not found or database error")
		component := casing.Edit(adminData) // Need an Edit view component
		ctx.Writer.WriteHeader(http.StatusNotFound)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Add the found casing to the admin data
	adminData = adminData.WithCasing(cas)

	// Render the Edit view template
	component := casing.Edit(adminData) // Need an Edit view component
	component.Render(ctx.Request.Context(), ctx.Writer)
}

// Update handles the POST /admin/casings/:id route
func (c *AdminCasingController) Update(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminCasingDataFromContext(ctx, "Update Casing", ctx.Request.URL.Path)

	// Get casing ID from the URL parameter
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		// Handle invalid ID format
		adminData = adminData.WithError("Invalid casing ID format")
		component := casing.Edit(adminData)
		ctx.Writer.WriteHeader(http.StatusBadRequest)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Get the existing casing from the database
	existingCasing, err := c.db.FindCasingByID(uint(id))
	if err != nil {
		// Handle case where casing is not found
		adminData = adminData.WithError("Casing not found or database error")
		component := casing.Edit(adminData)
		ctx.Writer.WriteHeader(http.StatusNotFound)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Get form values
	casingType := ctx.PostForm("type")
	popularityStr := ctx.PostForm("popularity")

	// Validate required fields
	if casingType == "" {
		adminData = adminData.WithError("Casing type is required")
		adminData = adminData.WithCasing(existingCasing) // Retain the existing casing in the form
		component := casing.Edit(adminData)
		ctx.Writer.WriteHeader(http.StatusBadRequest)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Convert popularity to int
	popularity, err := strconv.Atoi(popularityStr)
	if err != nil {
		// Handle error - maybe default to 0 or show error
		popularity = 0 // Defaulting to 0 for simplicity
	}

	// Update the casing properties
	existingCasing.Type = casingType
	existingCasing.Popularity = popularity

	// Attempt to update the casing in the database
	if err := c.db.UpdateCasing(existingCasing); err != nil {
		// Handle update failure
		adminData = adminData.WithError("Failed to update casing: " + err.Error())
		adminData = adminData.WithCasing(existingCasing)
		component := casing.Edit(adminData)
		ctx.Writer.WriteHeader(http.StatusInternalServerError)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Redirect to the index page with a success message
	ctx.Redirect(http.StatusFound, "/admin/casings?success=Casing updated successfully")
}

// Delete handles the POST /admin/casings/:id/delete route
func (c *AdminCasingController) Delete(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminCasingDataFromContext(ctx, "Delete Casing", ctx.Request.URL.Path)

	// Get casing ID from the URL parameter
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		// Handle invalid ID format
		adminData = adminData.WithError("Invalid casing ID format")
		// Redirect to index with error
		ctx.Redirect(http.StatusFound, "/admin/casings?error=Invalid casing ID format")
		return
	}

	// Attempt to delete the casing from the database
	if err := c.db.DeleteCasing(uint(id)); err != nil {
		// Handle delete failure
		adminData = adminData.WithError("Failed to delete casing: " + err.Error())
		// Redirect to index with error
		ctx.Redirect(http.StatusFound, "/admin/casings?error=Failed to delete casing")
		return
	}

	// Redirect to the index page with a success message
	ctx.Redirect(http.StatusFound, "/admin/casings?success=Casing deleted successfully")
}

// Add other CRUD methods (Delete) here...
