package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/admin/munition"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/database"
)

// AdminMunitionsController handles ammunition routes for admin
type AdminMunitionsController struct {
	db database.Service
}

// NewAdminMunitionsController creates a new admin munitions controller
func NewAdminMunitionsController(db database.Service) *AdminMunitionsController {
	return &AdminMunitionsController{
		db: db,
	}
}

// getAdminMunitionsDataFromContext gets admin data from context
func getAdminMunitionsDataFromContext(ctx *gin.Context, title string, currentPath string) *data.AdminData {
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

// Index shows all ammunition
func (c *AdminMunitionsController) Index(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminMunitionsDataFromContext(ctx, "Ammunition Management", "/admin/munitions")

	// Get all ammo
	ammo, err := c.db.FindAllAmmo()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get ammunition"})
		return
	}

	// Get all users
	users, err := c.db.FindAllUsers()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get users"})
		return
	}

	// Create maps for brand, caliber, and other lookups
	brandIDs := make([]uint, 0)
	caliberIDs := make([]uint, 0)
	bulletStyleIDs := make([]uint, 0)
	grainIDs := make([]uint, 0)
	casingIDs := make([]uint, 0)

	// Collect unique IDs
	for _, a := range ammo {
		brandIDs = append(brandIDs, a.BrandID)
		caliberIDs = append(caliberIDs, a.CaliberID)
		if a.BulletStyleID > 0 {
			bulletStyleIDs = append(bulletStyleIDs, a.BulletStyleID)
		}
		if a.GrainID > 0 {
			grainIDs = append(grainIDs, a.GrainID)
		}
		if a.CasingID > 0 {
			casingIDs = append(casingIDs, a.CasingID)
		}
	}

	// Get brands
	brands, err := c.db.FindAllBrands()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get brands"})
		return
	}

	// Get calibers by IDs
	calibers, err := c.db.FindAllCalibersByIDs(caliberIDs)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get calibers"})
		return
	}

	// Get bullet styles
	bulletStyles, err := c.db.FindAllBulletStyles()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get bullet styles"})
		return
	}

	// Get grains
	grains, err := c.db.FindAllGrains()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get grains"})
		return
	}

	// Get casings
	casings, err := c.db.FindAllCasings()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get casings"})
		return
	}

	// Create lookup maps
	brandMap := make(map[uint]string)
	for _, b := range brands {
		brandMap[b.ID] = b.Name
	}

	caliberMap := make(map[uint]string)
	for _, c := range calibers {
		caliberMap[c.ID] = c.Caliber
	}

	bulletStyleMap := make(map[uint]string)
	for _, bs := range bulletStyles {
		bulletStyleMap[bs.ID] = bs.Type
	}

	grainMap := make(map[uint]string)
	for _, g := range grains {
		grainMap[g.ID] = strconv.Itoa(g.Weight) + " gr"
	}

	casingMap := make(map[uint]string)
	for _, c := range casings {
		casingMap[c.ID] = c.Type
	}

	// Create user map with ammunition counts
	userMap := make(map[uint]struct {
		Email     string
		AmmoCount int64
	})

	for _, u := range users {
		count, _ := c.db.CountAmmoByUser(u.ID)
		userMap[u.ID] = struct {
			Email     string
			AmmoCount int64
		}{
			Email:     u.Email,
			AmmoCount: count,
		}
	}

	// Calculate total rounds
	totalRounds := int64(0)
	for _, a := range ammo {
		totalRounds += int64(a.Count)
	}

	// Create ammunition data
	munitionsData := munition.MunitionsIndexData{
		AdminData:      adminData,
		Ammo:           ammo,
		UserMap:        userMap,
		BrandMap:       brandMap,
		CaliberMap:     caliberMap,
		BulletStyleMap: bulletStyleMap,
		GrainMap:       grainMap,
		CasingMap:      casingMap,
		TotalRounds:    totalRounds,
	}

	// Render the ammunition index page
	munition.MunitionsIndex(&munitionsData).Render(ctx.Request.Context(), ctx.Writer)
}

// Show displays details for a specific ammunition
func (c *AdminMunitionsController) Show(ctx *gin.Context) {
	// Get the ID from the URL
	id := ctx.Param("id")
	ammoID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ammunition ID"})
		return
	}

	// Get admin data from context
	adminData := getAdminMunitionsDataFromContext(ctx, "Ammunition Details", "/admin/munitions")

	// Get the ammunition by ID
	ammo, err := c.db.FindAmmoByID(uint(ammoID))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Ammunition not found"})
		return
	}

	// Get user information
	user, err := c.db.GetUserByID(ammo.OwnerID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user information"})
		return
	}

	// Create the show data
	showData := munition.MunitionShowData{
		AdminData: adminData,
		Ammo:      ammo,
		User:      user,
	}

	// Render the ammunition show page
	munition.MunitionShow(&showData).Render(ctx.Request.Context(), ctx.Writer)
}
