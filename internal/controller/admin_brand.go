package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/admin/brand" // Import the brand view package
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
)

// AdminBrandController handles admin brand routes
type AdminBrandController struct {
	db database.Service
}

// NewAdminBrandController creates a new admin brand controller
func NewAdminBrandController(db database.Service) *AdminBrandController {
	return &AdminBrandController{
		db: db,
	}
}

// getAdminBrandDataFromContext gets admin data from context, specifically for Brand views
func getAdminBrandDataFromContext(ctx *gin.Context, title string, currentPath string) *data.AdminData {
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

// Index handles the GET /admin/brands route
func (c *AdminBrandController) Index(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminBrandDataFromContext(ctx, "Brands", ctx.Request.URL.Path)

	// Get success message from query params
	success := ctx.Query("success")
	if success != "" {
		adminData = adminData.WithSuccess(success)
	}

	// Get all brands from the database
	brands, err := c.db.FindAllBrands()
	if err != nil {
		// Render the template with an error message
		adminData = adminData.WithError("Failed to load brands")
		component := brand.Index(adminData)
		ctx.Writer.WriteHeader(http.StatusInternalServerError) // Set appropriate status code
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Add brands to admin data
	adminData = adminData.WithBrands(brands)

	// Render the template with the brands
	component := brand.Index(adminData)
	component.Render(ctx.Request.Context(), ctx.Writer)
}

// New handles the GET /admin/brands/new route
func (c *AdminBrandController) New(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminBrandDataFromContext(ctx, "New Brand", ctx.Request.URL.Path)

	// Render the new brand form template
	component := brand.New(adminData)
	component.Render(ctx.Request.Context(), ctx.Writer)
}

// Create handles the POST /admin/brands route
func (c *AdminBrandController) Create(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminBrandDataFromContext(ctx, "Create Brand", ctx.Request.URL.Path)

	// Get form values
	name := ctx.PostForm("name")
	nickname := ctx.PostForm("nickname")
	popularityStr := ctx.PostForm("popularity")

	// Debug log
	fmt.Printf("DEBUG: Create brand - Name: %s, Nickname: %s, Popularity: %s\n", name, nickname, popularityStr)

	// Validate required fields
	if name == "" {
		adminData = adminData.WithError("Brand name is required")
		component := brand.New(adminData)
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

	// Create the brand model
	br := models.Brand{
		Name:       name,
		Nickname:   nickname,
		Popularity: popularity,
	}

	// Debug log before saving
	fmt.Printf("DEBUG: About to create brand in DB: %+v\n", br)

	// Attempt to create the brand in the database
	if err := c.db.CreateBrand(&br); err != nil {
		// Debug log for error
		fmt.Printf("DEBUG: Error creating brand: %v\n", err)

		adminData = adminData.WithError("Failed to create brand: " + err.Error())
		component := brand.New(adminData)
		ctx.Writer.WriteHeader(http.StatusInternalServerError)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Debug log after saving
	fmt.Printf("DEBUG: Brand created successfully with ID: %d\n", br.ID)

	// Redirect to the index page with a success message
	ctx.Redirect(http.StatusSeeOther, "/admin/brands?success=Brand created successfully")
}

// Show handles the GET /admin/brands/:id route
func (c *AdminBrandController) Show(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminBrandDataFromContext(ctx, "Show Brand", ctx.Request.URL.Path)

	// Get brand ID from the URL parameter
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		// Handle invalid ID format
		adminData = adminData.WithError("Invalid brand ID format")
		component := brand.Show(adminData)
		ctx.Writer.WriteHeader(http.StatusBadRequest)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Fetch the brand from the database
	br, err := c.db.FindBrandByID(uint(id))
	if err != nil {
		// Handle case where brand is not found or other DB error
		adminData = adminData.WithError("Brand not found or database error")
		component := brand.Show(adminData)
		ctx.Writer.WriteHeader(http.StatusNotFound)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Add the found brand to the admin data
	adminData = adminData.WithBrand(br)

	// Render the Show view template
	component := brand.Show(adminData)
	component.Render(ctx.Request.Context(), ctx.Writer)
}

// Edit handles the GET /admin/brands/:id/edit route
func (c *AdminBrandController) Edit(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminBrandDataFromContext(ctx, "Edit Brand", ctx.Request.URL.Path)

	// Get brand ID from the URL parameter
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		// Handle invalid ID format
		adminData = adminData.WithError("Invalid brand ID format")
		component := brand.Edit(adminData)
		ctx.Writer.WriteHeader(http.StatusBadRequest)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Fetch the brand from the database
	br, err := c.db.FindBrandByID(uint(id))
	if err != nil {
		// Handle case where brand is not found
		adminData = adminData.WithError("Brand not found or database error")
		component := brand.Edit(adminData)
		ctx.Writer.WriteHeader(http.StatusNotFound)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Add the found brand to the admin data
	adminData = adminData.WithBrand(br)

	// Render the Edit view template
	component := brand.Edit(adminData)
	component.Render(ctx.Request.Context(), ctx.Writer)
}

// Update handles the POST /admin/brands/:id route
func (c *AdminBrandController) Update(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminBrandDataFromContext(ctx, "Update Brand", ctx.Request.URL.Path)

	// Get brand ID from the URL parameter
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		// Handle invalid ID format
		adminData = adminData.WithError("Invalid brand ID format")
		component := brand.Edit(adminData)
		ctx.Writer.WriteHeader(http.StatusBadRequest)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Get the existing brand from the database
	existingBrand, err := c.db.FindBrandByID(uint(id))
	if err != nil {
		// Handle case where brand is not found
		adminData = adminData.WithError("Brand not found or database error")
		component := brand.Edit(adminData)
		ctx.Writer.WriteHeader(http.StatusNotFound)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Get form values
	name := ctx.PostForm("name")
	nickname := ctx.PostForm("nickname")
	popularityStr := ctx.PostForm("popularity")

	// Validate required fields
	if name == "" {
		adminData = adminData.WithError("Brand name is required")
		adminData = adminData.WithBrand(existingBrand) // Retain the existing brand in the form
		component := brand.Edit(adminData)
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

	// Update the brand properties
	existingBrand.Name = name
	existingBrand.Nickname = nickname
	existingBrand.Popularity = popularity

	// Attempt to update the brand in the database
	if err := c.db.UpdateBrand(existingBrand); err != nil {
		// Handle update failure
		adminData = adminData.WithError("Failed to update brand: " + err.Error())
		adminData = adminData.WithBrand(existingBrand)
		component := brand.Edit(adminData)
		ctx.Writer.WriteHeader(http.StatusInternalServerError)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Redirect to the index page with a success message
	ctx.Redirect(http.StatusFound, "/admin/brands?success=Brand updated successfully")
}

// Delete handles the POST /admin/brands/:id/delete route
func (c *AdminBrandController) Delete(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminBrandDataFromContext(ctx, "Delete Brand", ctx.Request.URL.Path)

	// Get brand ID from the URL parameter
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		// Handle invalid ID format
		adminData = adminData.WithError("Invalid brand ID format")
		// Redirect to index with error
		ctx.Redirect(http.StatusFound, "/admin/brands?error=Invalid brand ID format")
		return
	}

	// Attempt to delete the brand from the database
	if err := c.db.DeleteBrand(uint(id)); err != nil {
		// Handle delete failure
		adminData = adminData.WithError("Failed to delete brand: " + err.Error())
		// Redirect to index with error
		ctx.Redirect(http.StatusFound, "/admin/brands?error=Failed to delete brand")
		return
	}

	// Redirect to the index page with a success message
	ctx.Redirect(http.StatusFound, "/admin/brands?success=Brand deleted successfully")
}
