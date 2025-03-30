package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/admin/bulletstyle" // Import the bullet style view package
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
)

// AdminBulletStyleController handles admin bullet style routes
type AdminBulletStyleController struct {
	db database.Service
}

// NewAdminBulletStyleController creates a new admin bullet style controller
func NewAdminBulletStyleController(db database.Service) *AdminBulletStyleController {
	return &AdminBulletStyleController{
		db: db,
	}
}

// getAdminBulletStyleDataFromContext gets admin data from context, specifically for BulletStyle views
func getAdminBulletStyleDataFromContext(ctx *gin.Context, title string, currentPath string) *data.AdminData {
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

// Index handles the GET /admin/bullet_styles route
func (c *AdminBulletStyleController) Index(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminBulletStyleDataFromContext(ctx, "Bullet Styles", ctx.Request.URL.Path)

	// Get success message from query params
	success := ctx.Query("success")
	if success != "" {
		adminData = adminData.WithSuccess(success)
	}

	// Get all bullet styles from the database
	bulletStyles, err := c.db.FindAllBulletStyles()
	if err != nil {
		// Render the template with an error message
		adminData = adminData.WithError("Failed to load bullet styles")
		component := bulletstyle.Index(adminData)
		ctx.Writer.WriteHeader(http.StatusInternalServerError) // Set appropriate status code
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Add bullet styles to admin data
	adminData = adminData.WithBulletStyles(bulletStyles)

	// Render the template with the bullet styles
	component := bulletstyle.Index(adminData)
	component.Render(ctx.Request.Context(), ctx.Writer)
}

// New handles the GET /admin/bullet_styles/new route
func (c *AdminBulletStyleController) New(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminBulletStyleDataFromContext(ctx, "New Bullet Style", ctx.Request.URL.Path)

	// Render the new bullet style form template
	component := bulletstyle.New(adminData)
	component.Render(ctx.Request.Context(), ctx.Writer)
}

// Create handles the POST /admin/bullet_styles route
func (c *AdminBulletStyleController) Create(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminBulletStyleDataFromContext(ctx, "Create Bullet Style", ctx.Request.URL.Path)

	// Get form values
	bulletStyleType := ctx.PostForm("type")
	nickname := ctx.PostForm("nickname")
	popularityStr := ctx.PostForm("popularity")

	// Debug log
	fmt.Printf("DEBUG: Create bullet style - Type: %s, Nickname: %s, Popularity: %s\n", bulletStyleType, nickname, popularityStr)

	// Validate required fields
	if bulletStyleType == "" {
		adminData = adminData.WithError("Bullet style type is required")
		component := bulletstyle.New(adminData)
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

	// Create the bullet style model
	bs := models.BulletStyle{
		Type:       bulletStyleType,
		Nickname:   nickname,
		Popularity: popularity,
	}

	// Debug log before saving
	fmt.Printf("DEBUG: About to create bullet style in DB: %+v\n", bs)

	// Attempt to create the bullet style in the database
	if err := c.db.CreateBulletStyle(&bs); err != nil {
		// Debug log for error
		fmt.Printf("DEBUG: Error creating bullet style: %v\n", err)

		adminData = adminData.WithError("Failed to create bullet style: " + err.Error())
		component := bulletstyle.New(adminData)
		ctx.Writer.WriteHeader(http.StatusInternalServerError)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Debug log after saving
	fmt.Printf("DEBUG: Bullet style created successfully with ID: %d\n", bs.ID)

	// Redirect to the index page with a success message
	ctx.Redirect(http.StatusSeeOther, "/admin/bullet_styles?success=Bullet style created successfully")
}

// Show handles the GET /admin/bullet_styles/:id route
func (c *AdminBulletStyleController) Show(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminBulletStyleDataFromContext(ctx, "Show Bullet Style", ctx.Request.URL.Path)

	// Get bullet style ID from the URL parameter
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		// Handle invalid ID format
		adminData = adminData.WithError("Invalid bullet style ID format")
		component := bulletstyle.Show(adminData)
		ctx.Writer.WriteHeader(http.StatusBadRequest)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Fetch the bullet style from the database
	bs, err := c.db.FindBulletStyleByID(uint(id))
	if err != nil {
		// Handle case where bullet style is not found or other DB error
		adminData = adminData.WithError("Bullet style not found or database error")
		component := bulletstyle.Show(adminData)
		ctx.Writer.WriteHeader(http.StatusNotFound)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Add the found bullet style to the admin data
	adminData = adminData.WithBulletStyle(bs)

	// Render the Show view template
	component := bulletstyle.Show(adminData)
	component.Render(ctx.Request.Context(), ctx.Writer)
}

// Edit handles the GET /admin/bullet_styles/:id/edit route
func (c *AdminBulletStyleController) Edit(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminBulletStyleDataFromContext(ctx, "Edit Bullet Style", ctx.Request.URL.Path)

	// Get bullet style ID from the URL parameter
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		// Handle invalid ID format
		adminData = adminData.WithError("Invalid bullet style ID format")
		component := bulletstyle.Edit(adminData)
		ctx.Writer.WriteHeader(http.StatusBadRequest)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Fetch the bullet style from the database
	bs, err := c.db.FindBulletStyleByID(uint(id))
	if err != nil {
		// Handle case where bullet style is not found
		adminData = adminData.WithError("Bullet style not found or database error")
		component := bulletstyle.Edit(adminData)
		ctx.Writer.WriteHeader(http.StatusNotFound)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Add the found bullet style to the admin data
	adminData = adminData.WithBulletStyle(bs)

	// Render the Edit view template
	component := bulletstyle.Edit(adminData)
	component.Render(ctx.Request.Context(), ctx.Writer)
}

// Update handles the POST /admin/bullet_styles/:id route
func (c *AdminBulletStyleController) Update(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminBulletStyleDataFromContext(ctx, "Update Bullet Style", ctx.Request.URL.Path)

	// Get bullet style ID from the URL parameter
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		// Handle invalid ID format
		adminData = adminData.WithError("Invalid bullet style ID format")
		component := bulletstyle.Edit(adminData)
		ctx.Writer.WriteHeader(http.StatusBadRequest)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Get the existing bullet style from the database
	existingBulletStyle, err := c.db.FindBulletStyleByID(uint(id))
	if err != nil {
		// Handle case where bullet style is not found
		adminData = adminData.WithError("Bullet style not found or database error")
		component := bulletstyle.Edit(adminData)
		ctx.Writer.WriteHeader(http.StatusNotFound)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Get form values
	bulletStyleType := ctx.PostForm("type")
	nickname := ctx.PostForm("nickname")
	popularityStr := ctx.PostForm("popularity")

	// Validate required fields
	if bulletStyleType == "" {
		adminData = adminData.WithError("Bullet style type is required")
		adminData = adminData.WithBulletStyle(existingBulletStyle) // Retain the existing bullet style in the form
		component := bulletstyle.Edit(adminData)
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

	// Update the bullet style properties
	existingBulletStyle.Type = bulletStyleType
	existingBulletStyle.Nickname = nickname
	existingBulletStyle.Popularity = popularity

	// Attempt to update the bullet style in the database
	if err := c.db.UpdateBulletStyle(existingBulletStyle); err != nil {
		// Handle update failure
		adminData = adminData.WithError("Failed to update bullet style: " + err.Error())
		adminData = adminData.WithBulletStyle(existingBulletStyle)
		component := bulletstyle.Edit(adminData)
		ctx.Writer.WriteHeader(http.StatusInternalServerError)
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Redirect to the index page with a success message
	ctx.Redirect(http.StatusFound, "/admin/bullet_styles?success=Bullet style updated successfully")
}

// Delete handles the POST /admin/bullet_styles/:id/delete route
func (c *AdminBulletStyleController) Delete(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminBulletStyleDataFromContext(ctx, "Delete Bullet Style", ctx.Request.URL.Path)

	// Get bullet style ID from the URL parameter
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		// Handle invalid ID format
		adminData = adminData.WithError("Invalid bullet style ID format")
		// Redirect to index with error
		ctx.Redirect(http.StatusFound, "/admin/bullet_styles?error=Invalid bullet style ID format")
		return
	}

	// Attempt to delete the bullet style from the database
	if err := c.db.DeleteBulletStyle(uint(id)); err != nil {
		// Handle delete failure
		adminData = adminData.WithError("Failed to delete bullet style: " + err.Error())
		// Redirect to index with error
		ctx.Redirect(http.StatusFound, "/admin/bullet_styles?error=Failed to delete bullet style")
		return
	}

	// Redirect to the index page with a success message
	ctx.Redirect(http.StatusFound, "/admin/bullet_styles?success=Bullet style deleted successfully")
}
