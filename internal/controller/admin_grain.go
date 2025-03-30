package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	grainView "github.com/hail2skins/armory/cmd/web/views/admin/grain"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
)

// AdminGrainController handles CRUD operations for grains
type AdminGrainController struct {
	db database.Service
}

// NewAdminGrainController creates a new instance of the AdminGrainController
func NewAdminGrainController(db database.Service) *AdminGrainController {
	return &AdminGrainController{
		db: db,
	}
}

// getAdminGrainDataFromContext gets admin data from context, specifically for Grain views
func getAdminGrainDataFromContext(ctx *gin.Context, title string, currentPath string) *data.AdminData {
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

// Index shows all grains
func (c *AdminGrainController) Index(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminGrainDataFromContext(ctx, "Grains", ctx.Request.URL.Path)

	// Get success message from query params
	success := ctx.Query("success")
	if success != "" {
		adminData = adminData.WithSuccess(success)
	}

	// Get all grains
	grains, err := c.db.FindAllGrains()
	if err != nil {
		// Render the template with an error message
		adminData = adminData.WithError("Failed to load grains")
		component := grainView.Index(adminData)
		ctx.Writer.WriteHeader(http.StatusInternalServerError) // Set appropriate status code
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Add grains to admin data
	adminData = adminData.WithGrains(grains)

	// Render the template with the grains
	component := grainView.Index(adminData)
	component.Render(ctx.Request.Context(), ctx.Writer)
}

// New shows a form to create a new grain
func (c *AdminGrainController) New(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminGrainDataFromContext(ctx, "New Grain", ctx.Request.URL.Path)

	// Render the new grain form template
	component := grainView.New(adminData)
	component.Render(ctx.Request.Context(), ctx.Writer)
}

// Create creates a new grain
func (c *AdminGrainController) Create(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminGrainDataFromContext(ctx, "Create Grain", ctx.Request.URL.Path)

	// Get form values
	weightStr := ctx.PostForm("weight")
	popularityStr := ctx.PostForm("popularity")

	// Debug log
	fmt.Printf("DEBUG: Create grain - Weight: %s, Popularity: %s\n", weightStr, popularityStr)

	// Validate required fields
	if weightStr == "" {
		adminData = adminData.WithError("Grain weight is required")
		component := grainView.New(adminData)
		ctx.Writer.WriteHeader(http.StatusBadRequest)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Convert weight to int
	weight, err := strconv.Atoi(weightStr)
	if err != nil {
		adminData = adminData.WithError("Invalid weight value: " + err.Error())
		component := grainView.New(adminData)
		ctx.Writer.WriteHeader(http.StatusBadRequest)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Convert popularity to int
	popularity, err := strconv.Atoi(popularityStr)
	if err != nil {
		adminData = adminData.WithError("Invalid popularity value: " + err.Error())
		component := grainView.New(adminData)
		ctx.Writer.WriteHeader(http.StatusBadRequest)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Create new grain
	grainObj := &models.Grain{
		Weight:     weight,
		Popularity: popularity,
	}

	// Debug log before saving
	fmt.Printf("DEBUG: About to create grain in DB: %+v\n", grainObj)

	// Save to database
	err = c.db.CreateGrain(grainObj)
	if err != nil {
		// Debug log for error
		fmt.Printf("DEBUG: Error creating grain: %v\n", err)

		adminData = adminData.WithError("Failed to create grain: " + err.Error())
		component := grainView.New(adminData)
		ctx.Writer.WriteHeader(http.StatusInternalServerError)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Debug log after saving
	fmt.Printf("DEBUG: Grain created successfully with ID: %d\n", grainObj.ID)

	// Redirect to index
	ctx.Redirect(http.StatusSeeOther, "/admin/grains?success=Grain created successfully")
}

// Show displays a specific grain
func (c *AdminGrainController) Show(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminGrainDataFromContext(ctx, "Show Grain", ctx.Request.URL.Path)

	// Get grain ID from URL
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		// Handle invalid ID format
		adminData = adminData.WithError("Invalid grain ID format")
		component := grainView.Show(adminData)
		ctx.Writer.WriteHeader(http.StatusBadRequest)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Find the grain
	grainObj, err := c.db.FindGrainByID(uint(id))
	if err != nil {
		// Handle case where grain is not found or other DB error
		adminData = adminData.WithError("Grain not found or database error")
		component := grainView.Show(adminData)
		ctx.Writer.WriteHeader(http.StatusNotFound)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Add the found grain to the admin data
	adminData = adminData.WithGrain(grainObj)

	// Render the Show view template
	component := grainView.Show(adminData)
	component.Render(ctx.Request.Context(), ctx.Writer)
}

// Edit shows a form to edit an existing grain
func (c *AdminGrainController) Edit(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminGrainDataFromContext(ctx, "Edit Grain", ctx.Request.URL.Path)

	// Get grain ID from URL
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		// Handle invalid ID format
		adminData = adminData.WithError("Invalid grain ID format")
		component := grainView.Edit(adminData)
		ctx.Writer.WriteHeader(http.StatusBadRequest)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Find the grain
	grainObj, err := c.db.FindGrainByID(uint(id))
	if err != nil {
		// Handle case where grain is not found
		adminData = adminData.WithError("Grain not found or database error")
		component := grainView.Edit(adminData)
		ctx.Writer.WriteHeader(http.StatusNotFound)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Add the found grain to the admin data
	adminData = adminData.WithGrain(grainObj)

	// Render the Edit view template
	component := grainView.Edit(adminData)
	component.Render(ctx.Request.Context(), ctx.Writer)
}

// Update updates an existing grain
func (c *AdminGrainController) Update(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminGrainDataFromContext(ctx, "Update Grain", ctx.Request.URL.Path)

	// Get grain ID from URL
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		// Handle invalid ID format
		adminData = adminData.WithError("Invalid grain ID format")
		component := grainView.Edit(adminData)
		ctx.Writer.WriteHeader(http.StatusBadRequest)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Get the existing grain from the database
	existingGrain, err := c.db.FindGrainByID(uint(id))
	if err != nil {
		// Handle case where grain is not found
		adminData = adminData.WithError("Grain not found or database error")
		component := grainView.Edit(adminData)
		ctx.Writer.WriteHeader(http.StatusNotFound)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Get form values
	weightStr := ctx.PostForm("weight")
	popularityStr := ctx.PostForm("popularity")

	// Validate required fields
	if weightStr == "" {
		adminData = adminData.WithError("Grain weight is required")
		adminData = adminData.WithGrain(existingGrain) // Retain the existing grain in the form
		component := grainView.Edit(adminData)
		ctx.Writer.WriteHeader(http.StatusBadRequest)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Convert weight to int
	weight, err := strconv.Atoi(weightStr)
	if err != nil {
		adminData = adminData.WithError("Invalid weight value: " + err.Error())
		adminData = adminData.WithGrain(existingGrain)
		component := grainView.Edit(adminData)
		ctx.Writer.WriteHeader(http.StatusBadRequest)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Convert popularity to int
	popularity, err := strconv.Atoi(popularityStr)
	if err != nil {
		adminData = adminData.WithError("Invalid popularity value: " + err.Error())
		adminData = adminData.WithGrain(existingGrain)
		component := grainView.Edit(adminData)
		ctx.Writer.WriteHeader(http.StatusBadRequest)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Update grain fields
	existingGrain.Weight = weight
	existingGrain.Popularity = popularity

	// Save to database
	err = c.db.UpdateGrain(existingGrain)
	if err != nil {
		// Handle update failure
		adminData = adminData.WithError("Failed to update grain: " + err.Error())
		adminData = adminData.WithGrain(existingGrain)
		component := grainView.Edit(adminData)
		ctx.Writer.WriteHeader(http.StatusInternalServerError)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Redirect to index with success message
	ctx.Redirect(http.StatusFound, "/admin/grains?success=Grain updated successfully")
}

// Delete removes a grain
func (c *AdminGrainController) Delete(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminGrainDataFromContext(ctx, "Delete Grain", ctx.Request.URL.Path)

	// Get grain ID from URL
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		// Handle invalid ID format
		adminData = adminData.WithError("Invalid grain ID format")
		// Redirect to index with error
		ctx.Redirect(http.StatusFound, "/admin/grains?error=Invalid grain ID format")
		return
	}

	// Delete from database
	err = c.db.DeleteGrain(uint(id))
	if err != nil {
		// Handle delete failure
		adminData = adminData.WithError("Failed to delete grain: " + err.Error())
		// Redirect to index with error
		ctx.Redirect(http.StatusFound, "/admin/grains?error=Failed to delete grain")
		return
	}

	// Redirect to index with success message
	ctx.Redirect(http.StatusFound, "/admin/grains?success=Grain deleted successfully")
}
