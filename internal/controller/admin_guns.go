package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/admin/gun"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/database"
)

// AdminGunsController handles gun routes for admin
type AdminGunsController struct {
	db database.Service
}

// NewAdminGunsController creates a new admin guns controller
func NewAdminGunsController(db database.Service) *AdminGunsController {
	return &AdminGunsController{
		db: db,
	}
}

// getAdminGunsDataFromContext gets admin data from context
func getAdminGunsDataFromContext(ctx *gin.Context, title string, currentPath string) *data.AdminData {
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

// Index shows all guns
func (c *AdminGunsController) Index(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminGunsDataFromContext(ctx, "Guns Management", "/admin/guns")

	// Get all guns
	guns, err := c.db.FindAllGuns()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get guns"})
		return
	}

	// Get all users
	users, err := c.db.FindAllUsers()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get users"})
		return
	}

	// Create maps for manufacturer, caliber, and weapon type lookups
	manufacturerIDs := make([]uint, 0)
	caliberIDs := make([]uint, 0)
	weaponTypeIDs := make([]uint, 0)

	// Collect unique IDs
	for _, g := range guns {
		manufacturerIDs = append(manufacturerIDs, g.ManufacturerID)
		caliberIDs = append(caliberIDs, g.CaliberID)
		weaponTypeIDs = append(weaponTypeIDs, g.WeaponTypeID)
	}

	// Get manufacturers
	manufacturers, err := c.db.FindAllManufacturers()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get manufacturers"})
		return
	}

	// Get calibers by IDs
	calibers, err := c.db.FindAllCalibersByIDs(caliberIDs)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get calibers"})
		return
	}

	// Get weapon types by IDs
	weaponTypes, err := c.db.FindAllWeaponTypesByIDs(weaponTypeIDs)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get weapon types"})
		return
	}

	// Create lookup maps
	manufacturerMap := make(map[uint]string)
	for _, m := range manufacturers {
		manufacturerMap[m.ID] = m.Name
	}

	caliberMap := make(map[uint]string)
	for _, c := range calibers {
		caliberMap[c.ID] = c.Caliber
	}

	weaponTypeMap := make(map[uint]string)
	for _, wt := range weaponTypes {
		weaponTypeMap[wt.ID] = wt.Type
	}

	// Create user map with gun counts
	userMap := make(map[uint]struct {
		Name     string
		Email    string
		GunCount int64
	})

	for _, u := range users {
		count, _ := c.db.CountGunsByUser(u.ID)
		userMap[u.ID] = struct {
			Name     string
			Email    string
			GunCount int64
		}{
			Name:     u.Email,
			Email:    u.Email,
			GunCount: count,
		}
	}

	// Total guns count
	totalGuns := int64(len(guns))

	// Create guns data
	gunsData := gun.GunsIndexData{
		AdminData:       adminData,
		Guns:            guns,
		UserMap:         userMap,
		ManufacturerMap: manufacturerMap,
		CaliberMap:      caliberMap,
		WeaponTypeMap:   weaponTypeMap,
		TotalGuns:       totalGuns,
	}

	// Render the guns index page
	gun.GunsIndex(&gunsData).Render(ctx.Request.Context(), ctx.Writer)
}
