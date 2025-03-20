package controller

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"log"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/cmd/web/views/owner"
	gunView "github.com/hail2skins/armory/cmd/web/views/owner/gun"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/services/email"
	"github.com/shaj13/go-guardian/v2/auth"
)

// AuthControllerInterface defines the interface for the auth controller
type AuthControllerInterface interface {
	GetCurrentUser(c *gin.Context) (auth.Info, bool)
}

// OwnerController handles owner-related routes
type OwnerController struct {
	db database.Service
}

// NewOwnerController creates a new owner controller
func NewOwnerController(db database.Service) *OwnerController {
	return &OwnerController{
		db: db,
	}
}

// LandingPage handles the owner landing page route
func (o *OwnerController) LandingPage(c *gin.Context) {
	// Get the current user's authentication status and email
	var userInfo auth.Info
	var authenticated bool

	authController, ok := c.MustGet("authController").(AuthControllerInterface)
	if !ok {
		// Try to cast to the concrete type as a fallback
		concreteAuthController, ok := c.MustGet("authController").(*AuthController)
		if !ok {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}
		userInfo, authenticated = concreteAuthController.GetCurrentUser(c)
	} else {
		userInfo, authenticated = authController.GetCurrentUser(c)
	}

	if !authenticated {
		// Set flash message
		if setFlash, exists := c.Get("setFlash"); exists {
			setFlash.(func(string))("You must be logged in to access this page")
		}
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get the user from the database
	ctx := context.Background()
	dbUser, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(c.DefaultQuery("perPage", "10"))
	if perPage < 1 {
		perPage = 10
	}

	// Parse sorting parameters
	sortBy := c.DefaultQuery("sortBy", "created_at")
	sortOrder := c.DefaultQuery("sortOrder", "desc")

	// Validate sort parameters
	validSortFields := map[string]bool{
		"name":         true,
		"created_at":   true,
		"acquired":     true,
		"manufacturer": true,
		"caliber":      true,
		"weapon_type":  true,
	}
	if !validSortFields[sortBy] {
		sortBy = "created_at"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	// Get search term if available
	searchTerm := c.Query("search")

	// Get the user's guns with pagination
	var guns []models.Gun
	var totalGuns int64

	// Use the database service's GetDB method to get the underlying gorm.DB
	db := o.db.GetDB()
	query := db.Model(&models.Gun{}).Where("owner_id = ?", dbUser.ID)

	// Add search functionality if search term is provided
	if searchTerm != "" {
		query = query.Where("name LIKE ?", "%"+searchTerm+"%")
	}

	// Count total entries for pagination
	query.Count(&totalGuns)

	// Execute paginated query with sorting
	gunQuery := db.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").
		Where("owner_id = ?", dbUser.ID)

	// Add search if provided
	if searchTerm != "" {
		gunQuery = gunQuery.Where("name LIKE ?", "%"+searchTerm+"%")
	}

	// Add sorting logic
	if sortBy == "manufacturer" {
		gunQuery = gunQuery.Joins("JOIN manufacturers ON guns.manufacturer_id = manufacturers.id").
			Order("manufacturers.name " + sortOrder)
	} else if sortBy == "caliber" {
		gunQuery = gunQuery.Joins("JOIN calibers ON guns.caliber_id = calibers.id").
			Order("calibers.caliber " + sortOrder)
	} else if sortBy == "weapon_type" {
		gunQuery = gunQuery.Joins("JOIN weapon_types ON guns.weapon_type_id = weapon_types.id").
			Order("weapon_types.type " + sortOrder)
	} else {
		gunQuery = gunQuery.Order(sortBy + " " + sortOrder)
	}

	// Apply pagination
	offset := (page - 1) * perPage
	gunQuery = gunQuery.Offset(offset).Limit(perPage)

	if err := gunQuery.Find(&guns).Error; err != nil {
		guns = []models.Gun{}
	}

	// Calculate total pages
	totalPages := int((totalGuns + int64(perPage) - 1) / int64(perPage))

	// Check if free tier limit applies (only for display, not actual limit)
	var showingFreeLimit bool
	var totalUserGuns int
	if dbUser.SubscriptionTier == "free" && totalGuns > 2 {
		showingFreeLimit = true
		totalUserGuns = int(totalGuns)
		// We don't actually limit the guns here anymore, we'll add a message
	}

	// Format subscription end date if available
	var subscriptionEndsAt string
	if !dbUser.SubscriptionEndDate.IsZero() {
		subscriptionEndsAt = dbUser.SubscriptionEndDate.Format("January 2, 2006")
	}

	// Calculate the total paid amount for all guns
	totalPaid := calculateTotalPaid(guns)

	// Create owner data
	ownerData := data.NewOwnerData().
		WithTitle("Owner Dashboard").
		WithAuthenticated(authenticated).
		WithUser(dbUser).
		WithGuns(guns).
		WithSubscriptionInfo(
			dbUser.HasActiveSubscription(),
			dbUser.SubscriptionTier,
			subscriptionEndsAt,
		).
		WithPagination(page, totalPages, perPage, int(totalGuns)).
		WithSorting(sortBy, sortOrder).
		WithSearchTerm(searchTerm).
		WithFiltersApplied(sortBy, sortOrder, perPage, searchTerm).
		WithTotalPaid(totalPaid)

	// Get authData from context to preserve roles
	if authDataInterface, exists := c.Get("authData"); exists {
		if authData, ok := authDataInterface.(data.AuthData); ok {
			// Use the auth data that already has roles, maintaining our title and other changes
			ownerData.Auth = authData.WithTitle("Owner Dashboard")
		}
	}

	// If the user has more guns than shown, add a message
	if showingFreeLimit {
		ownerData.WithError(fmt.Sprintf("Free Tier shows all %d guns in your collection, but only allows you to add up to 2 guns. Please subscribe to add more firearms.", totalUserGuns))
	}

	// Check for flash message from cookie
	if flashCookie, err := c.Cookie("flash"); err == nil && flashCookie != "" {
		// Add flash message to success messages
		ownerData.WithSuccess(flashCookie)
		// Clear the flash cookie
		c.SetCookie("flash", "", -1, "/", "", false, false)
	}

	// Render the owner landing page with the data
	owner.Owner(ownerData).Render(c.Request.Context(), c.Writer)
}

// Index handles the gun index route
func (o *OwnerController) Index(c *gin.Context) {
	// This will be implemented later
	c.JSON(200, gin.H{"message": "Gun index page"})
}

// New handles the new gun route
func (o *OwnerController) New(c *gin.Context) {
	// Get the current user's authentication status and email
	authController, ok := c.MustGet("authController").(AuthControllerInterface)
	if !ok {
		// Try to cast to the concrete type as a fallback
		concreteAuthController, ok := c.MustGet("authController").(*AuthController)
		if !ok {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}
		userInfo, authenticated := concreteAuthController.GetCurrentUser(c)
		if !authenticated {
			// Set flash message
			if setFlash, exists := c.Get("setFlash"); exists {
				setFlash.(func(string))("You must be logged in to access this page")
			}
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		// Get the user from the database
		ctx := context.Background()
		dbUser, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
		if err != nil {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		// Get all weapon types, calibers, and manufacturers for the form
		weaponTypes, err := o.db.FindAllWeaponTypes()
		if err != nil {
			weaponTypes = []models.WeaponType{}
		}

		calibers, err := o.db.FindAllCalibers()
		if err != nil {
			calibers = []models.Caliber{}
		}

		manufacturers, err := o.db.FindAllManufacturers()
		if err != nil {
			manufacturers = []models.Manufacturer{}
		}

		// Create owner data
		ownerData := data.NewOwnerData().
			WithTitle("Add New Firearm").
			WithAuthenticated(authenticated).
			WithUser(dbUser).
			WithWeaponTypes(weaponTypes).
			WithCalibers(calibers).
			WithManufacturers(manufacturers)

		// Get authData from context to preserve roles
		if authDataInterface, exists := c.Get("authData"); exists {
			if authData, ok := authDataInterface.(data.AuthData); ok {
				// Use the auth data that already has roles, maintaining our title and other changes
				ownerData.Auth = authData.WithTitle("Add New Firearm")
			}
		}

		// Check for flash message from cookie
		if flashCookie, err := c.Cookie("flash"); err == nil && flashCookie != "" {
			// Add flash message to success messages
			ownerData.WithSuccess(flashCookie)
			// Clear the flash cookie
			c.SetCookie("flash", "", -1, "/", "", false, false)
		}

		// Render the new gun form with the data
		gunView.New(ownerData).Render(c.Request.Context(), c.Writer)
		return
	}

	userInfo, authenticated := authController.GetCurrentUser(c)
	if !authenticated {
		// Set flash message
		if setFlash, exists := c.Get("setFlash"); exists {
			setFlash.(func(string))("You must be logged in to access this page")
		}
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get the user from the database
	ctx := context.Background()
	dbUser, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get all weapon types, calibers, and manufacturers for the form
	weaponTypes, err := o.db.FindAllWeaponTypes()
	if err != nil {
		weaponTypes = []models.WeaponType{}
	}

	calibers, err := o.db.FindAllCalibers()
	if err != nil {
		calibers = []models.Caliber{}
	}

	manufacturers, err := o.db.FindAllManufacturers()
	if err != nil {
		manufacturers = []models.Manufacturer{}
	}

	// Create owner data
	ownerData := data.NewOwnerData().
		WithTitle("Add New Firearm").
		WithAuthenticated(authenticated).
		WithUser(dbUser).
		WithWeaponTypes(weaponTypes).
		WithCalibers(calibers).
		WithManufacturers(manufacturers)

	// Get authData from context to preserve roles
	if authDataInterface, exists := c.Get("authData"); exists {
		if authData, ok := authDataInterface.(data.AuthData); ok {
			// Use the auth data that already has roles, maintaining our title and other changes
			ownerData.Auth = authData.WithTitle("Add New Firearm")
		}
	}

	// Check for flash message from cookie
	if flashCookie, err := c.Cookie("flash"); err == nil && flashCookie != "" {
		// Add flash message to success messages
		ownerData.WithSuccess(flashCookie)
		// Clear the flash cookie
		c.SetCookie("flash", "", -1, "/", "", false, false)
	}

	// Render the new gun form with the data
	gunView.New(ownerData).Render(c.Request.Context(), c.Writer)
}

// Create handles the create gun route
func (o *OwnerController) Create(c *gin.Context) {
	// Get the current user's authentication status and email
	authController, ok := c.MustGet("authController").(AuthControllerInterface)
	if !ok {
		// Try to cast to the concrete type as a fallback
		concreteAuthController, ok := c.MustGet("authController").(*AuthController)
		if !ok {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}
		userInfo, authenticated := concreteAuthController.GetCurrentUser(c)
		if !authenticated {
			// Set flash message
			if setFlash, exists := c.Get("setFlash"); exists {
				setFlash.(func(string))("You must be logged in to access this page")
			}
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		// Get the user from the database
		ctx := context.Background()
		dbUser, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
		if err != nil {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		// Check if the user is on the free tier and already has 2 guns
		if dbUser.SubscriptionTier == "free" {
			var count int64
			db := o.db.GetDB()
			db.Model(&models.Gun{}).Where("owner_id = ?", dbUser.ID).Count(&count)

			// If the user already has 2 guns, redirect to the pricing page
			if count >= 2 {
				if setFlash, exists := c.Get("setFlash"); exists {
					setFlash.(func(string))("You must be subscribed to add more to your arsenal")
				}
				c.Redirect(http.StatusSeeOther, "/pricing")
				return
			}
		}

		// Create a new gun
		name := c.PostForm("name")
		serialNumber := c.PostForm("serial_number")
		acquiredDateStr := c.PostForm("acquired")
		weaponTypeIDStr := c.PostForm("weapon_type_id")
		caliberIDStr := c.PostForm("caliber_id")
		manufacturerIDStr := c.PostForm("manufacturer_id")
		paidStr := c.PostForm("paid")

		// Validate fields
		errors := make(map[string]string)
		if name == "" {
			errors["name"] = "Name is required"
		}

		if serialNumber == "" {
			errors["serial_number"] = "Serial number is required"
		}

		// Parse acquired date if provided
		var acquiredDate *time.Time
		if acquiredDateStr != "" {
			parsedDate, err := time.Parse("2006-01-02", acquiredDateStr)
			if err != nil {
				errors["acquired"] = "Invalid date format, use YYYY-MM-DD"
			} else {
				acquiredDate = &parsedDate
			}
		}

		// Parse weapon type ID
		weaponTypeID, err := strconv.Atoi(weaponTypeIDStr)
		if err != nil || weaponTypeID <= 0 {
			errors["weapon_type_id"] = "Invalid weapon type"
		}

		// Parse caliber ID
		caliberID, err := strconv.Atoi(caliberIDStr)
		if err != nil || caliberID <= 0 {
			errors["caliber_id"] = "Invalid caliber"
		}

		// Parse manufacturer ID
		manufacturerID, err := strconv.Atoi(manufacturerIDStr)
		if err != nil || manufacturerID <= 0 {
			errors["manufacturer_id"] = "Invalid manufacturer"
		}

		// Parse paid amount if provided
		var paidAmount *float64
		if paidStr != "" {
			paid, err := strconv.ParseFloat(paidStr, 64)
			if err != nil {
				errors["paid"] = "Invalid amount format, use numbers only (e.g. 1500.50)"
			} else {
				paidAmount = &paid
			}
		}

		// If there are any errors, re-render the form with error messages
		if len(errors) > 0 {
			// Get all weapon types, calibers, and manufacturers for the form
			weaponTypes, err := o.db.FindAllWeaponTypes()
			if err != nil {
				weaponTypes = []models.WeaponType{}
			}

			calibers, err := o.db.FindAllCalibers()
			if err != nil {
				calibers = []models.Caliber{}
			}

			manufacturers, err := o.db.FindAllManufacturers()
			if err != nil {
				manufacturers = []models.Manufacturer{}
			}

			// Create owner data with errors
			ownerData := data.NewOwnerData().
				WithTitle("Add New Firearm").
				WithAuthenticated(authenticated).
				WithUser(dbUser).
				WithWeaponTypes(weaponTypes).
				WithCalibers(calibers).
				WithManufacturers(manufacturers).
				WithFormErrors(errors)

			// Render the new gun form with the data and errors
			gunView.New(ownerData).Render(c.Request.Context(), c.Writer)
			return
		}

		// Create the gun
		newGun := &models.Gun{
			Name:           name,
			SerialNumber:   serialNumber,
			Acquired:       acquiredDate,
			WeaponTypeID:   uint(weaponTypeID),
			CaliberID:      uint(caliberID),
			ManufacturerID: uint(manufacturerID),
			OwnerID:        dbUser.ID,
			Paid:           paidAmount,
		}

		// Save the gun to the database
		db := o.db.GetDB()
		if err := models.CreateGun(db, newGun); err != nil {
			// If there's an error, re-render the form with an error message
			// Get all weapon types, calibers, and manufacturers for the form
			weaponTypes, err := o.db.FindAllWeaponTypes()
			if err != nil {
				weaponTypes = []models.WeaponType{}
			}

			calibers, err := o.db.FindAllCalibers()
			if err != nil {
				calibers = []models.Caliber{}
			}

			manufacturers, err := o.db.FindAllManufacturers()
			if err != nil {
				manufacturers = []models.Manufacturer{}
			}

			// Create owner data with errors
			ownerData := data.NewOwnerData().
				WithTitle("Add New Firearm").
				WithAuthenticated(authenticated).
				WithUser(dbUser).
				WithWeaponTypes(weaponTypes).
				WithCalibers(calibers).
				WithManufacturers(manufacturers).
				WithError("Failed to create gun: " + err.Error())

			// Render the new gun form with the data and errors
			gunView.New(ownerData).Render(c.Request.Context(), c.Writer)
			return
		}

		// Set flash message
		if setFlash, exists := c.Get("setFlash"); exists {
			setFlash.(func(string))("Weapon added to your arsenal")
		}

		// Redirect to the owner dashboard
		c.Redirect(http.StatusSeeOther, "/owner")
		return
	}

	userInfo, authenticated := authController.GetCurrentUser(c)
	if !authenticated {
		// Set flash message
		if setFlash, exists := c.Get("setFlash"); exists {
			setFlash.(func(string))("You must be logged in to access this page")
		}
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get the user from the database
	ctx := context.Background()
	dbUser, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get all weapon types, calibers, and manufacturers for the form
	weaponTypes, err := o.db.FindAllWeaponTypes()
	if err != nil {
		weaponTypes = []models.WeaponType{}
	}

	calibers, err := o.db.FindAllCalibers()
	if err != nil {
		calibers = []models.Caliber{}
	}

	manufacturers, err := o.db.FindAllManufacturers()
	if err != nil {
		manufacturers = []models.Manufacturer{}
	}

	// Create owner data
	ownerData := data.NewOwnerData().
		WithTitle("Add New Firearm").
		WithAuthenticated(authenticated).
		WithUser(dbUser).
		WithWeaponTypes(weaponTypes).
		WithCalibers(calibers).
		WithManufacturers(manufacturers)

	// Get authData from context to preserve roles
	if authDataInterface, exists := c.Get("authData"); exists {
		if authData, ok := authDataInterface.(data.AuthData); ok {
			// Use the auth data that already has roles, maintaining our title and other changes
			ownerData.Auth = authData.WithTitle("Add New Firearm")
		}
	}

	// Check for flash message from cookie
	if flashCookie, err := c.Cookie("flash"); err == nil && flashCookie != "" {
		// Add flash message to success messages
		ownerData.WithSuccess(flashCookie)
		// Clear the flash cookie
		c.SetCookie("flash", "", -1, "/", "", false, false)
	}

	// Render the new gun form with the data
	gunView.New(ownerData).Render(c.Request.Context(), c.Writer)
}

// Show handles the show gun route
func (o *OwnerController) Show(c *gin.Context) {
	// Get the current user's authentication status and email
	authController, ok := c.MustGet("authController").(AuthControllerInterface)
	if !ok {
		// Try to cast to the concrete type as a fallback
		concreteAuthController, ok := c.MustGet("authController").(*AuthController)
		if !ok {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}
		userInfo, authenticated := concreteAuthController.GetCurrentUser(c)
		if !authenticated {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		// Get the user from the database
		ctx := context.Background()
		dbUser, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
		if err != nil {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		// Get the gun ID from the URL
		gunID := c.Param("id")

		// Get the gun from the database
		var gun models.Gun
		db := o.db.GetDB()
		if err := db.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").Where("id = ? AND owner_id = ?", gunID, dbUser.ID).First(&gun).Error; err != nil {
			c.Redirect(http.StatusSeeOther, "/owner")
			return
		}

		// Format subscription end date if available
		var subscriptionEndsAt string
		if !dbUser.SubscriptionEndDate.IsZero() {
			subscriptionEndsAt = dbUser.SubscriptionEndDate.Format("January 2, 2006")
		}

		// Create owner data
		ownerData := data.NewOwnerData().
			WithTitle(fmt.Sprintf("Gun: %s", gun.Name)).
			WithAuthenticated(authenticated).
			WithUser(dbUser).
			WithGun(&gun).
			WithSubscriptionInfo(
				dbUser.HasActiveSubscription(),
				dbUser.SubscriptionTier,
				subscriptionEndsAt,
			)

		// Get authData from context to preserve roles
		if authDataInterface, exists := c.Get("authData"); exists {
			if authData, ok := authDataInterface.(data.AuthData); ok {
				// Use the auth data that already has roles, maintaining our title and other changes
				ownerData.Auth = authData.WithTitle(fmt.Sprintf("Gun: %s", gun.Name))
			}
		}

		// Check for flash message from cookie
		if flashCookie, err := c.Cookie("flash"); err == nil && flashCookie != "" {
			// Clear the flash cookie
			c.SetCookie("flash", "", -1, "/", "", false, true)
			ownerData.WithSuccess(flashCookie)
		}

		// Render the gun show page
		gunView.Show(ownerData).Render(c.Request.Context(), c.Writer)
		return
	}

	userInfo, authenticated := authController.GetCurrentUser(c)
	if !authenticated {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get the user from the database
	ctx := context.Background()
	dbUser, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get the gun ID from the URL
	gunID := c.Param("id")

	// Get the gun from the database
	var gun models.Gun
	db := o.db.GetDB()
	if err := db.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").Where("id = ? AND owner_id = ?", gunID, dbUser.ID).First(&gun).Error; err != nil {
		c.Redirect(http.StatusSeeOther, "/owner")
		return
	}

	// Format subscription end date if available
	var subscriptionEndsAt string
	if !dbUser.SubscriptionEndDate.IsZero() {
		subscriptionEndsAt = dbUser.SubscriptionEndDate.Format("January 2, 2006")
	}

	// Create owner data
	ownerData := data.NewOwnerData().
		WithTitle(fmt.Sprintf("Gun: %s", gun.Name)).
		WithAuthenticated(authenticated).
		WithUser(dbUser).
		WithGun(&gun).
		WithSubscriptionInfo(
			dbUser.HasActiveSubscription(),
			dbUser.SubscriptionTier,
			subscriptionEndsAt,
		)

	// Get authData from context to preserve roles
	if authDataInterface, exists := c.Get("authData"); exists {
		if authData, ok := authDataInterface.(data.AuthData); ok {
			// Use the auth data that already has roles, maintaining our title and other changes
			ownerData.Auth = authData.WithTitle(fmt.Sprintf("Gun: %s", gun.Name))
		}
	}

	// Check for flash message from cookie
	if flashCookie, err := c.Cookie("flash"); err == nil && flashCookie != "" {
		// Clear the flash cookie
		c.SetCookie("flash", "", -1, "/", "", false, true)
		ownerData.WithSuccess(flashCookie)
	}

	// Render the gun show page
	gunView.Show(ownerData).Render(c.Request.Context(), c.Writer)
}

// Edit handles the gun edit route
func (o *OwnerController) Edit(c *gin.Context) {
	// Get the current user's authentication status and email
	authController, ok := c.MustGet("authController").(AuthControllerInterface)
	if !ok {
		// Try to cast to the concrete type as a fallback
		concreteAuthController, ok := c.MustGet("authController").(*AuthController)
		if !ok {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}
		userInfo, authenticated := concreteAuthController.GetCurrentUser(c)
		if !authenticated {
			// Set flash message
			if setFlash, exists := c.Get("setFlash"); exists {
				setFlash.(func(string))("You must be logged in to access this page")
			}
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}
		// Get the user from the database
		ctx := context.Background()
		user, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to get user"})
			return
		}

		// Get the gun ID from the URL
		gunID := c.Param("id")

		// Get the gun from the database
		var gun models.Gun
		db := o.db.GetDB()
		if err := db.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").Where("id = ? AND owner_id = ?", gunID, user.ID).First(&gun).Error; err != nil {
			c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Gun not found"})
			return
		}

		// Get all weapon types, calibers, and manufacturers
		var weaponTypes []models.WeaponType
		if err := db.Order("popularity DESC").Find(&weaponTypes).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to get weapon types"})
			return
		}

		var calibers []models.Caliber
		if err := db.Order("popularity DESC").Find(&calibers).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to get calibers"})
			return
		}

		var manufacturers []models.Manufacturer
		if err := db.Order("popularity DESC").Find(&manufacturers).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to get manufacturers"})
			return
		}

		// Create the data for the view
		viewData := data.NewOwnerData().
			WithTitle(fmt.Sprintf("Edit Gun: %s", gun.Name)).
			WithAuthenticated(authenticated).
			WithUser(user).
			WithGun(&gun).
			WithWeaponTypes(weaponTypes).
			WithCalibers(calibers).
			WithManufacturers(manufacturers)

		// Format subscription end date if available
		var subscriptionEndsAt string
		if !user.SubscriptionEndDate.IsZero() {
			subscriptionEndsAt = user.SubscriptionEndDate.Format("January 2, 2006")
		}

		// Add subscription info
		viewData = viewData.WithSubscriptionInfo(
			user.HasActiveSubscription(),
			user.SubscriptionTier,
			subscriptionEndsAt,
		)

		// Get authData from context to preserve roles
		if authDataInterface, exists := c.Get("authData"); exists {
			if authData, ok := authDataInterface.(data.AuthData); ok {
				// Use the auth data that already has roles, maintaining our title and other changes
				viewData.Auth = authData.WithTitle(fmt.Sprintf("Edit Gun: %s", gun.Name))
			}
		}

		// Render the edit form
		c.Status(http.StatusOK)
		gunView.Edit(viewData).Render(context.Background(), c.Writer)
		return
	}

	userInfo, authenticated := authController.GetCurrentUser(c)
	if !authenticated {
		// Set flash message
		if setFlash, exists := c.Get("setFlash"); exists {
			setFlash.(func(string))("You must be logged in to access this page")
		}
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get the user from the database
	ctx := context.Background()
	user, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to get user"})
		return
	}

	// Get the gun ID from the URL
	gunID := c.Param("id")

	// Get the gun from the database
	var gun models.Gun
	db := o.db.GetDB()
	if err := db.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").Where("id = ? AND owner_id = ?", gunID, user.ID).First(&gun).Error; err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Gun not found"})
		return
	}

	// Get all weapon types, calibers, and manufacturers
	var weaponTypes []models.WeaponType
	if err := db.Order("popularity DESC").Find(&weaponTypes).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to get weapon types"})
		return
	}

	var calibers []models.Caliber
	if err := db.Order("popularity DESC").Find(&calibers).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to get calibers"})
		return
	}

	var manufacturers []models.Manufacturer
	if err := db.Order("popularity DESC").Find(&manufacturers).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to get manufacturers"})
		return
	}

	// Create the data for the view
	viewData := data.NewOwnerData().
		WithTitle(fmt.Sprintf("Edit Gun: %s", gun.Name)).
		WithAuthenticated(authenticated).
		WithUser(user).
		WithGun(&gun).
		WithWeaponTypes(weaponTypes).
		WithCalibers(calibers).
		WithManufacturers(manufacturers)

	// Format subscription end date if available
	var subscriptionEndsAt string
	if !user.SubscriptionEndDate.IsZero() {
		subscriptionEndsAt = user.SubscriptionEndDate.Format("January 2, 2006")
	}

	// Add subscription info
	viewData = viewData.WithSubscriptionInfo(
		user.HasActiveSubscription(),
		user.SubscriptionTier,
		subscriptionEndsAt,
	)

	// Get authData from context to preserve roles
	if authDataInterface, exists := c.Get("authData"); exists {
		if authData, ok := authDataInterface.(data.AuthData); ok {
			// Use the auth data that already has roles, maintaining our title and other changes
			viewData.Auth = authData.WithTitle(fmt.Sprintf("Edit Gun: %s", gun.Name))
		}
	}

	// Render the edit form
	c.Status(http.StatusOK)
	gunView.Edit(viewData).Render(context.Background(), c.Writer)
}

// Update handles the update gun form submission
func (o *OwnerController) Update(c *gin.Context) {
	// Get the current user's authentication status and email
	authController, ok := c.MustGet("authController").(AuthControllerInterface)
	if !ok {
		// Try to cast to the concrete type as a fallback
		concreteAuthController, ok := c.MustGet("authController").(*AuthController)
		if !ok {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}
		userInfo, authenticated := concreteAuthController.GetCurrentUser(c)
		if !authenticated {
			// Set flash message
			if setFlash, exists := c.Get("setFlash"); exists {
				setFlash.(func(string))("You must be logged in to access this page")
			}
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}
		// Get the user from the database
		ctx := context.Background()
		user, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to get user"})
			return
		}

		// Get the gun ID from the URL
		gunID := c.Param("id")

		// Get the gun from the database
		var gun models.Gun
		db := o.db.GetDB()
		if err := db.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").Where("id = ? AND owner_id = ?", gunID, user.ID).First(&gun).Error; err != nil {
			c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Gun not found"})
			return
		}

		// Parse the form
		if err := c.Request.ParseForm(); err != nil {
			c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid form data"})
			return
		}

		// Get the form values
		name := c.PostForm("name")
		serialNumber := c.PostForm("serial_number")
		acquiredDateStr := c.PostForm("acquired_date")
		weaponTypeIDStr := c.PostForm("weapon_type_id")
		caliberIDStr := c.PostForm("caliber_id")
		manufacturerIDStr := c.PostForm("manufacturer_id")
		paidStr := c.PostForm("paid")

		// Validate the form
		formErrors := make(map[string]string)

		if name == "" {
			formErrors["name"] = "Name is required"
		}

		if weaponTypeIDStr == "" {
			formErrors["weapon_type_id"] = "Weapon type is required"
		}

		if caliberIDStr == "" {
			formErrors["caliber_id"] = "Caliber is required"
		}

		if manufacturerIDStr == "" {
			formErrors["manufacturer_id"] = "Manufacturer is required"
		}

		// If there are validation errors, re-render the form with errors
		if len(formErrors) > 0 {
			// Get all weapon types, calibers, and manufacturers
			var weaponTypes []models.WeaponType
			if err := db.Order("popularity DESC").Find(&weaponTypes).Error; err != nil {
				c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to get weapon types"})
				return
			}

			var calibers []models.Caliber
			if err := db.Order("popularity DESC").Find(&calibers).Error; err != nil {
				c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to get calibers"})
				return
			}

			var manufacturers []models.Manufacturer
			if err := db.Order("popularity DESC").Find(&manufacturers).Error; err != nil {
				c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to get manufacturers"})
				return
			}

			// Create the data for the view
			viewData := data.NewOwnerData().
				WithTitle("Edit Firearm").
				WithAuthenticated(true).
				WithUser(user).
				WithGun(&gun).
				WithWeaponTypes(weaponTypes).
				WithCalibers(calibers).
				WithManufacturers(manufacturers).
				WithFormErrors(formErrors)

			// Render the edit form
			c.Status(http.StatusOK)
			gunView.Edit(viewData).Render(context.Background(), c.Writer)
			return
		}

		// Parse the IDs
		weaponTypeID, err := strconv.ParseUint(weaponTypeIDStr, 10, 64)
		if err != nil {
			c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid weapon type ID"})
			return
		}

		caliberID, err := strconv.ParseUint(caliberIDStr, 10, 64)
		if err != nil {
			c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid caliber ID"})
			return
		}

		manufacturerID, err := strconv.ParseUint(manufacturerIDStr, 10, 64)
		if err != nil {
			c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid manufacturer ID"})
			return
		}

		// Parse acquired date if provided
		var acquiredDate *time.Time
		if acquiredDateStr != "" {
			parsedDate, err := time.Parse("2006-01-02", acquiredDateStr)
			if err != nil {
				c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid acquired date"})
				return
			}
			acquiredDate = &parsedDate
		} else {
			acquiredDate = nil
		}

		// Parse paid amount if provided
		var paidAmount *float64
		if paidStr != "" {
			paid, err := strconv.ParseFloat(paidStr, 64)
			if err != nil {
				c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid paid amount"})
				return
			}
			paidAmount = &paid
		} else {
			paidAmount = nil
		}

		// Update the gun
		updatedGun := &models.Gun{
			Name:           name,
			SerialNumber:   serialNumber,
			Acquired:       acquiredDate,
			WeaponTypeID:   uint(weaponTypeID),
			CaliberID:      uint(caliberID),
			ManufacturerID: uint(manufacturerID),
			OwnerID:        user.ID,
			Paid:           paidAmount,
		}
		updatedGun.ID = gun.ID

		// Save the gun
		if err := models.UpdateGun(db, updatedGun); err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to update gun"})
			return
		}

		// Reload the gun with its relationships to ensure they're updated
		if err := db.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").First(&updatedGun, updatedGun.ID).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to reload gun"})
			return
		}

		// Set flash message
		if setFlash, exists := c.Get("setFlash"); exists {
			setFlash.(func(string))("Your gun has been updated.")
		}

		// Redirect to the owner page
		c.Redirect(http.StatusSeeOther, "/owner")
		return
	}

	userInfo, authenticated := authController.GetCurrentUser(c)
	if !authenticated {
		// Set flash message
		if setFlash, exists := c.Get("setFlash"); exists {
			setFlash.(func(string))("You must be logged in to access this page")
		}
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get the user from the database
	ctx := context.Background()
	user, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to get user"})
		return
	}

	// Get the gun ID from the URL
	gunID := c.Param("id")

	// Get the gun from the database
	var gun models.Gun
	db := o.db.GetDB()
	if err := db.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").Where("id = ? AND owner_id = ?", gunID, user.ID).First(&gun).Error; err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Gun not found"})
		return
	}

	// Parse the form
	if err := c.Request.ParseForm(); err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid form data"})
		return
	}

	// Get the form values
	name := c.PostForm("name")
	serialNumber := c.PostForm("serial_number")
	acquiredDateStr := c.PostForm("acquired_date")
	weaponTypeIDStr := c.PostForm("weapon_type_id")
	caliberIDStr := c.PostForm("caliber_id")
	manufacturerIDStr := c.PostForm("manufacturer_id")
	paidStr := c.PostForm("paid")

	// Validate the form
	formErrors := make(map[string]string)

	if name == "" {
		formErrors["name"] = "Name is required"
	}

	if weaponTypeIDStr == "" {
		formErrors["weapon_type_id"] = "Weapon type is required"
	}

	if caliberIDStr == "" {
		formErrors["caliber_id"] = "Caliber is required"
	}

	if manufacturerIDStr == "" {
		formErrors["manufacturer_id"] = "Manufacturer is required"
	}

	// If there are validation errors, re-render the form with errors
	if len(formErrors) > 0 {
		// Get all weapon types, calibers, and manufacturers
		var weaponTypes []models.WeaponType
		if err := db.Order("popularity DESC").Find(&weaponTypes).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to get weapon types"})
			return
		}

		var calibers []models.Caliber
		if err := db.Order("popularity DESC").Find(&calibers).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to get calibers"})
			return
		}

		var manufacturers []models.Manufacturer
		if err := db.Order("popularity DESC").Find(&manufacturers).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to get manufacturers"})
			return
		}

		// Create the data for the view
		viewData := data.NewOwnerData().
			WithTitle("Edit Firearm").
			WithAuthenticated(true).
			WithUser(user).
			WithGun(&gun).
			WithWeaponTypes(weaponTypes).
			WithCalibers(calibers).
			WithManufacturers(manufacturers).
			WithFormErrors(formErrors)

		// Render the edit form
		c.Status(http.StatusOK)
		gunView.Edit(viewData).Render(context.Background(), c.Writer)
		return
	}

	// Parse the IDs
	weaponTypeID, err := strconv.ParseUint(weaponTypeIDStr, 10, 64)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid weapon type ID"})
		return
	}

	caliberID, err := strconv.ParseUint(caliberIDStr, 10, 64)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid caliber ID"})
		return
	}

	manufacturerID, err := strconv.ParseUint(manufacturerIDStr, 10, 64)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid manufacturer ID"})
		return
	}

	// Parse acquired date if provided
	var acquiredDate *time.Time
	if acquiredDateStr != "" {
		parsedDate, err := time.Parse("2006-01-02", acquiredDateStr)
		if err != nil {
			c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid acquired date"})
			return
		}
		acquiredDate = &parsedDate
	} else {
		acquiredDate = nil
	}

	// Parse paid amount if provided
	var paidAmount *float64
	if paidStr != "" {
		paid, err := strconv.ParseFloat(paidStr, 64)
		if err != nil {
			c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid paid amount"})
			return
		}
		paidAmount = &paid
	} else {
		paidAmount = nil
	}

	// Update the gun
	updatedGun := &models.Gun{
		Name:           name,
		SerialNumber:   serialNumber,
		Acquired:       acquiredDate,
		WeaponTypeID:   uint(weaponTypeID),
		CaliberID:      uint(caliberID),
		ManufacturerID: uint(manufacturerID),
		OwnerID:        user.ID,
		Paid:           paidAmount,
	}
	updatedGun.ID = gun.ID

	// Save the gun
	if err := models.UpdateGun(db, updatedGun); err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to update gun"})
		return
	}

	// Reload the gun with its relationships to ensure they're updated
	if err := db.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").First(&updatedGun, updatedGun.ID).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to reload gun"})
		return
	}

	// Set flash message
	if setFlash, exists := c.Get("setFlash"); exists {
		setFlash.(func(string))("Your gun has been updated.")
	}

	// Redirect to the owner page
	c.Redirect(http.StatusSeeOther, "/owner")
}

// Delete handles the delete gun route
func (o *OwnerController) Delete(c *gin.Context) {
	// Get the current user's authentication status and email
	authController, ok := c.MustGet("authController").(AuthControllerInterface)
	if !ok {
		// Try to cast to the concrete type as a fallback
		concreteAuthController, ok := c.MustGet("authController").(*AuthController)
		if !ok {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}
		userInfo, authenticated := concreteAuthController.GetCurrentUser(c)
		if !authenticated {
			// Set flash message
			if setFlash, exists := c.Get("setFlash"); exists {
				setFlash.(func(string))("You must be logged in to access this page")
			}
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		// Get the user from the database
		ctx := context.Background()
		dbUser, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
		if err != nil {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		// Get the gun ID from the URL parameter
		gunID, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			// Set flash message
			if setFlash, exists := c.Get("setFlash"); exists {
				setFlash.(func(string))("Invalid gun ID")
			}
			c.Redirect(http.StatusSeeOther, "/owner")
			return
		}

		// Delete the gun
		db := o.db.GetDB()
		err = o.db.DeleteGun(db, uint(gunID), dbUser.ID)
		if err != nil {
			// Set flash message
			if setFlash, exists := c.Get("setFlash"); exists {
				setFlash.(func(string))("Error deleting gun: " + err.Error())
			}
			c.Redirect(http.StatusSeeOther, "/owner")
			return
		}

		// Set flash message
		if setFlash, exists := c.Get("setFlash"); exists {
			setFlash.(func(string))("Your gun has been deleted.")
		}

		// Redirect to the owner page
		c.Redirect(http.StatusSeeOther, "/owner")
		return
	}

	userInfo, authenticated := authController.GetCurrentUser(c)
	if !authenticated {
		// Set flash message
		if setFlash, exists := c.Get("setFlash"); exists {
			setFlash.(func(string))("You must be logged in to access this page")
		}
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get the user from the database
	ctx := context.Background()
	dbUser, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get the gun ID from the URL parameter
	gunID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		// Set flash message
		if setFlash, exists := c.Get("setFlash"); exists {
			setFlash.(func(string))("Invalid gun ID")
		}
		c.Redirect(http.StatusSeeOther, "/owner")
		return
	}

	// Delete the gun
	db := o.db.GetDB()
	err = o.db.DeleteGun(db, uint(gunID), dbUser.ID)
	if err != nil {
		// Set flash message
		if setFlash, exists := c.Get("setFlash"); exists {
			setFlash.(func(string))("Error deleting gun: " + err.Error())
		}
		c.Redirect(http.StatusSeeOther, "/owner")
		return
	}

	// Set flash message
	if setFlash, exists := c.Get("setFlash"); exists {
		setFlash.(func(string))("Your gun has been deleted.")
	}

	// Redirect to the owner page
	c.Redirect(http.StatusSeeOther, "/owner")
}

// SearchCalibers handles the caliber search API
func (o *OwnerController) SearchCalibers(c *gin.Context) {
	// This will be implemented later
	c.JSON(200, gin.H{"message": "Search calibers"})
}

// Arsenal handles the arsenal view route
func (o *OwnerController) Arsenal(c *gin.Context) {
	// Get the current user's authentication status and email
	authController, ok := c.MustGet("authController").(AuthControllerInterface)
	if !ok {
		// Try to cast to the concrete type as a fallback
		concreteAuthController, ok := c.MustGet("authController").(*AuthController)
		if !ok {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}
		userInfo, authenticated := concreteAuthController.GetCurrentUser(c)
		if !authenticated {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		// Get the user from the database
		ctx := context.Background()
		dbUser, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
		if err != nil {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		// Parse pagination parameters
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		if page < 1 {
			page = 1
		}
		perPage, _ := strconv.Atoi(c.DefaultQuery("perPage", "50"))
		if perPage < 1 {
			perPage = 50
		}

		// Parse sorting parameters
		sortBy := c.DefaultQuery("sortBy", "name")
		sortOrder := c.DefaultQuery("sortOrder", "asc")

		// Validate sort parameters
		validSortFields := map[string]bool{
			"name":         true,
			"created_at":   true,
			"acquired":     true,
			"manufacturer": true,
			"caliber":      true,
			"weapon_type":  true,
		}
		if !validSortFields[sortBy] {
			sortBy = "name"
		}
		if sortOrder != "asc" && sortOrder != "desc" {
			sortOrder = "asc"
		}

		// Get search term if available
		searchTerm := c.Query("search")

		// Get the total count of guns for this owner
		var totalGuns int64
		db := o.db.GetDB()
		query := db.Model(&models.Gun{}).Where("owner_id = ?", dbUser.ID)

		// Add search if provided
		if searchTerm != "" {
			query = query.Where("name LIKE ?", "%"+searchTerm+"%")
		}

		// Count total guns
		query.Count(&totalGuns)

		// Apply pagination and get guns
		var guns []models.Gun
		gunQuery := db.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").
			Where("owner_id = ?", dbUser.ID)

		// Add search if provided
		if searchTerm != "" {
			gunQuery = gunQuery.Where("name LIKE ?", "%"+searchTerm+"%")
		}

		// Add sorting logic
		if sortBy == "manufacturer" {
			gunQuery = gunQuery.Joins("JOIN manufacturers ON guns.manufacturer_id = manufacturers.id").
				Order("manufacturers.name " + sortOrder)
		} else if sortBy == "caliber" {
			gunQuery = gunQuery.Joins("JOIN calibers ON guns.caliber_id = calibers.id").
				Order("calibers.caliber " + sortOrder)
		} else if sortBy == "weapon_type" {
			gunQuery = gunQuery.Joins("JOIN weapon_types ON guns.weapon_type_id = weapon_types.id").
				Order("weapon_types.type " + sortOrder)
		} else {
			gunQuery = gunQuery.Order(sortBy + " " + sortOrder)
		}

		// Apply pagination
		offset := (page - 1) * perPage
		gunQuery = gunQuery.Offset(offset).Limit(perPage)

		if err := gunQuery.Find(&guns).Error; err != nil {
			guns = []models.Gun{}
		}

		// Calculate total pages
		totalPages := int((totalGuns + int64(perPage) - 1) / int64(perPage))

		// Apply free tier limit if needed - this now applies to the display only
		var showFreeLimit bool
		var totalUserGuns int
		if dbUser.SubscriptionTier == "free" && totalGuns > 2 {
			showFreeLimit = true
			totalUserGuns = int(totalGuns)
			// We don't actually limit the guns anymore, just show a message
		}

		// Format subscription end date if available
		var subscriptionEndsAt string
		if !dbUser.SubscriptionEndDate.IsZero() {
			subscriptionEndsAt = dbUser.SubscriptionEndDate.Format("January 2, 2006")
		}

		// Calculate the total paid amount for all guns
		totalPaid := calculateTotalPaid(guns)

		// Create owner data
		ownerData := data.NewOwnerData().
			WithTitle("Your Arsenal").
			WithAuthenticated(authenticated).
			WithUser(dbUser).
			WithGuns(guns).
			WithSubscriptionInfo(
				dbUser.HasActiveSubscription(),
				dbUser.SubscriptionTier,
				subscriptionEndsAt,
			).
			WithPagination(page, totalPages, perPage, int(totalGuns)).
			WithSorting(sortBy, sortOrder).
			WithSearchTerm(searchTerm).
			WithFiltersApplied(sortBy, sortOrder, perPage, searchTerm).
			WithTotalPaid(totalPaid)

		// Get authData from context to preserve roles
		if authDataInterface, exists := c.Get("authData"); exists {
			if authData, ok := authDataInterface.(data.AuthData); ok {
				// Use the auth data that already has roles, maintaining our title and other changes
				ownerData.Auth = authData.WithTitle("Your Arsenal")
			}
		}

		// If the user is on free tier, add a message
		if showFreeLimit {
			ownerData.WithError(fmt.Sprintf("Free Tier shows all %d guns in your collection, but only allows you to add up to 2 guns. Please subscribe to add more firearms.", totalUserGuns))
		}

		// Check for flash message from cookie
		if flashCookie, err := c.Cookie("flash"); err == nil && flashCookie != "" {
			// Clear the flash cookie
			c.SetCookie("flash", "", -1, "/", "", false, true)
			ownerData.WithSuccess(flashCookie)
		}

		// Render the arsenal page
		gunView.Arsenal(ownerData).Render(c.Request.Context(), c.Writer)
		return
	}

	// Get the second half of the method too
	userInfo, authenticated := authController.GetCurrentUser(c)
	if !authenticated {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get the user from the database
	ctx := context.Background()
	dbUser, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(c.DefaultQuery("perPage", "50"))
	if perPage < 1 {
		perPage = 50
	}

	// Parse sorting parameters
	sortBy := c.DefaultQuery("sortBy", "name")
	sortOrder := c.DefaultQuery("sortOrder", "asc")

	// Validate sort parameters
	validSortFields := map[string]bool{
		"name":         true,
		"created_at":   true,
		"acquired":     true,
		"manufacturer": true,
		"caliber":      true,
		"weapon_type":  true,
	}
	if !validSortFields[sortBy] {
		sortBy = "name"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "asc"
	}

	// Get search term if available
	searchTerm := c.Query("search")

	// Get the total count of guns for this owner
	var totalGuns int64
	db := o.db.GetDB()
	query := db.Model(&models.Gun{}).Where("owner_id = ?", dbUser.ID)

	// Add search if provided
	if searchTerm != "" {
		query = query.Where("name LIKE ?", "%"+searchTerm+"%")
	}

	// Count total guns
	query.Count(&totalGuns)

	// Apply pagination and get guns
	var guns []models.Gun
	gunQuery := db.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").
		Where("owner_id = ?", dbUser.ID)

	// Add search if provided
	if searchTerm != "" {
		gunQuery = gunQuery.Where("name LIKE ?", "%"+searchTerm+"%")
	}

	// Add sorting logic
	if sortBy == "manufacturer" {
		gunQuery = gunQuery.Joins("JOIN manufacturers ON guns.manufacturer_id = manufacturers.id").
			Order("manufacturers.name " + sortOrder)
	} else if sortBy == "caliber" {
		gunQuery = gunQuery.Joins("JOIN calibers ON guns.caliber_id = calibers.id").
			Order("calibers.caliber " + sortOrder)
	} else if sortBy == "weapon_type" {
		gunQuery = gunQuery.Joins("JOIN weapon_types ON guns.weapon_type_id = weapon_types.id").
			Order("weapon_types.type " + sortOrder)
	} else {
		gunQuery = gunQuery.Order(sortBy + " " + sortOrder)
	}

	// Apply pagination
	offset := (page - 1) * perPage
	gunQuery = gunQuery.Offset(offset).Limit(perPage)

	if err := gunQuery.Find(&guns).Error; err != nil {
		guns = []models.Gun{}
	}

	// Calculate total pages
	totalPages := int((totalGuns + int64(perPage) - 1) / int64(perPage))

	// Apply free tier limit if needed - this now applies to the display only
	var showFreeLimit bool
	var totalUserGuns int
	if dbUser.SubscriptionTier == "free" && totalGuns > 2 {
		showFreeLimit = true
		totalUserGuns = int(totalGuns)
		// We don't actually limit the guns anymore, just show a message
	}

	// Format subscription end date if available
	var subscriptionEndsAt string
	if !dbUser.SubscriptionEndDate.IsZero() {
		subscriptionEndsAt = dbUser.SubscriptionEndDate.Format("January 2, 2006")
	}

	// Create owner data
	ownerData := data.NewOwnerData().
		WithTitle("Your Arsenal").
		WithAuthenticated(authenticated).
		WithUser(dbUser).
		WithGuns(guns).
		WithSubscriptionInfo(
			dbUser.HasActiveSubscription(),
			dbUser.SubscriptionTier,
			subscriptionEndsAt,
		).
		WithPagination(page, totalPages, perPage, int(totalGuns)).
		WithSorting(sortBy, sortOrder).
		WithSearchTerm(searchTerm).
		WithFiltersApplied(sortBy, sortOrder, perPage, searchTerm)

	// Get authData from context to preserve roles
	if authDataInterface, exists := c.Get("authData"); exists {
		if authData, ok := authDataInterface.(data.AuthData); ok {
			// Use the auth data that already has roles, maintaining our title and other changes
			ownerData.Auth = authData.WithTitle("Your Arsenal")
		}
	}

	// If the user is on free tier, add a message
	if showFreeLimit {
		ownerData.WithError(fmt.Sprintf("Free Tier shows all %d guns in your collection, but only allows you to add up to 2 guns. Please subscribe to add more firearms.", totalUserGuns))
	}

	// Check for flash message from cookie
	if flashCookie, err := c.Cookie("flash"); err == nil && flashCookie != "" {
		// Clear the flash cookie
		c.SetCookie("flash", "", -1, "/", "", false, true)
		ownerData.WithSuccess(flashCookie)
	}

	// Render the arsenal page
	gunView.Arsenal(ownerData).Render(c.Request.Context(), c.Writer)
}

// Profile handles the owner profile page route
func (o *OwnerController) Profile(c *gin.Context) {
	// Get the current user's authentication status and email
	authController, ok := c.MustGet("authController").(AuthControllerInterface)
	if !ok {
		// Try to cast to the concrete type as a fallback
		concreteAuthController, ok := c.MustGet("authController").(*AuthController)
		if !ok {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}
		userInfo, authenticated := concreteAuthController.GetCurrentUser(c)
		if !authenticated {
			// Set flash message
			if setFlash, exists := c.Get("setFlash"); exists {
				setFlash.(func(string))("You must be logged in to access this page")
			}
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		// Get the user from the database
		ctx := context.Background()
		dbUser, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
		if err != nil {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		// Format subscription end date if available
		var subscriptionEndsAt string
		if !dbUser.SubscriptionEndDate.IsZero() {
			subscriptionEndsAt = dbUser.SubscriptionEndDate.Format("January 2, 2006")
		}

		// Create owner data
		ownerData := data.NewOwnerData().
			WithTitle("Profile").
			WithAuthenticated(authenticated).
			WithUser(dbUser).
			WithSubscriptionInfo(
				dbUser.HasActiveSubscription(),
				dbUser.SubscriptionTier,
				subscriptionEndsAt,
			)

		// Get authData from context to preserve roles
		if authDataInterface, exists := c.Get("authData"); exists {
			if authData, ok := authDataInterface.(data.AuthData); ok {
				// Use the auth data that already has roles, maintaining our title and other changes
				ownerData.Auth = authData.WithTitle("Profile")
			}
		}

		// Check for flash message from cookie
		if flashCookie, err := c.Cookie("flash"); err == nil && flashCookie != "" {
			// Add flash message to success messages
			ownerData.WithSuccess(flashCookie)
			// Clear the flash cookie
			c.SetCookie("flash", "", -1, "/", "", false, false)
		}

		// Render the profile page with the data
		owner.Profile(ownerData).Render(c.Request.Context(), c.Writer)
		return
	}

	userInfo, authenticated := authController.GetCurrentUser(c)
	if !authenticated {
		// Set flash message
		if setFlash, exists := c.Get("setFlash"); exists {
			setFlash.(func(string))("You must be logged in to access this page")
		}
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get the user from the database
	ctx := context.Background()
	dbUser, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Format subscription end date if available
	var subscriptionEndsAt string
	if !dbUser.SubscriptionEndDate.IsZero() {
		subscriptionEndsAt = dbUser.SubscriptionEndDate.Format("January 2, 2006")
	}

	// Create owner data
	ownerData := data.NewOwnerData().
		WithTitle("Profile").
		WithAuthenticated(authenticated).
		WithUser(dbUser).
		WithSubscriptionInfo(
			dbUser.HasActiveSubscription(),
			dbUser.SubscriptionTier,
			subscriptionEndsAt,
		)

	// Get authData from context to preserve roles
	if authDataInterface, exists := c.Get("authData"); exists {
		if authData, ok := authDataInterface.(data.AuthData); ok {
			// Use the auth data that already has roles, maintaining our title and other changes
			ownerData.Auth = authData.WithTitle("Profile")
		}
	}

	// Check for flash message from cookie
	if flashCookie, err := c.Cookie("flash"); err == nil && flashCookie != "" {
		// Add flash message to success messages
		ownerData.WithSuccess(flashCookie)
		// Clear the flash cookie
		c.SetCookie("flash", "", -1, "/", "", false, false)
	}

	// Render the profile page with the data
	owner.Profile(ownerData).Render(c.Request.Context(), c.Writer)
}

// EditProfile renders the profile edit page
func (o *OwnerController) EditProfile(c *gin.Context) {
	// Get the current user's authentication status and email
	authController, ok := c.MustGet("authController").(AuthControllerInterface)
	if !ok {
		// Try to cast to the concrete type as a fallback
		concreteAuthController, ok := c.MustGet("authController").(*AuthController)
		if !ok {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}
		userInfo, authenticated := concreteAuthController.GetCurrentUser(c)
		if !authenticated {
			// Set flash message
			if setFlash, exists := c.Get("setFlash"); exists {
				setFlash.(func(string))("You must be logged in to access this page")
			}
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		// Get the user from the database
		ctx := context.Background()
		dbUser, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
		if err != nil {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		// Format subscription end date if available
		var subscriptionEndsAt string
		if !dbUser.SubscriptionEndDate.IsZero() {
			subscriptionEndsAt = dbUser.SubscriptionEndDate.Format("January 2, 2006")
		}

		// Create owner data
		ownerData := data.NewOwnerData().
			WithTitle("Edit Profile").
			WithAuthenticated(authenticated).
			WithUser(dbUser).
			WithSubscriptionInfo(
				dbUser.HasActiveSubscription(),
				dbUser.SubscriptionTier,
				subscriptionEndsAt,
			)

		// Get authData from context to preserve roles
		if authDataInterface, exists := c.Get("authData"); exists {
			if authData, ok := authDataInterface.(data.AuthData); ok {
				// Use the auth data that already has roles, maintaining our title and other changes
				ownerData.Auth = authData.WithTitle("Edit Profile")
			}
		}

		// Check for flash message from cookie
		if flashCookie, err := c.Cookie("flash"); err == nil && flashCookie != "" {
			// Add flash message to success messages
			ownerData.WithSuccess(flashCookie)
			// Clear the flash cookie
			c.SetCookie("flash", "", -1, "/", "", false, false)
		}

		// Render the profile edit page
		owner.Edit(ownerData).Render(c.Request.Context(), c.Writer)
		return
	}

	userInfo, authenticated := authController.GetCurrentUser(c)
	if !authenticated {
		// Set flash message
		if setFlash, exists := c.Get("setFlash"); exists {
			setFlash.(func(string))("You must be logged in to access this page")
		}
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get the user from the database
	ctx := context.Background()
	dbUser, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Format subscription end date if available
	var subscriptionEndsAt string
	if !dbUser.SubscriptionEndDate.IsZero() {
		subscriptionEndsAt = dbUser.SubscriptionEndDate.Format("January 2, 2006")
	}

	// Create owner data
	ownerData := data.NewOwnerData().
		WithTitle("Edit Profile").
		WithAuthenticated(authenticated).
		WithUser(dbUser).
		WithSubscriptionInfo(
			dbUser.HasActiveSubscription(),
			dbUser.SubscriptionTier,
			subscriptionEndsAt,
		)

	// Get authData from context to preserve roles
	if authDataInterface, exists := c.Get("authData"); exists {
		if authData, ok := authDataInterface.(data.AuthData); ok {
			// Use the auth data that already has roles, maintaining our title and other changes
			ownerData.Auth = authData.WithTitle("Edit Profile")
		}
	}

	// Check for flash message from cookie
	if flashCookie, err := c.Cookie("flash"); err == nil && flashCookie != "" {
		// Add flash message to success messages
		ownerData.WithSuccess(flashCookie)
		// Clear the flash cookie
		c.SetCookie("flash", "", -1, "/", "", false, false)
	}

	// Render the profile edit page
	owner.Edit(ownerData).Render(c.Request.Context(), c.Writer)
}

// UpdateProfile handles the update profile form submission
func (o *OwnerController) UpdateProfile(c *gin.Context) {
	// Get the current user's authentication status and email
	authController, ok := c.MustGet("authController").(AuthControllerInterface)
	if !ok {
		// Try to cast to the concrete type as a fallback
		concreteAuthController, ok := c.MustGet("authController").(*AuthController)
		if !ok {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}
		userInfo, authenticated := concreteAuthController.GetCurrentUser(c)
		if !authenticated {
			// Set flash message
			if setFlash, exists := c.Get("setFlash"); exists {
				setFlash.(func(string))("You must be logged in to access this page")
			}
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		// Get the user from the database
		ctx := context.Background()
		dbUser, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
		if err != nil {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		// Parse form data
		if err := c.Request.ParseForm(); err != nil {
			// Create owner data with error
			ownerData := data.NewOwnerData().
				WithTitle("Edit Profile").
				WithAuthenticated(authenticated).
				WithUser(dbUser).
				WithError("Failed to parse form data")

			// Render the profile edit page with error
			owner.Edit(ownerData).Render(c.Request.Context(), c.Writer)
			return
		}

		// Get email from form
		newEmail := c.PostForm("email")

		// Validate email format
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(newEmail) {
			// Create owner data with error
			ownerData := data.NewOwnerData().
				WithTitle("Edit Profile").
				WithAuthenticated(authenticated).
				WithUser(dbUser).
				WithError("Invalid email format")

			// Render the profile edit page with error
			c.Status(http.StatusOK)
			owner.Edit(ownerData).Render(c.Request.Context(), c.Writer)
			return
		}

		// Check if email is being changed
		emailChanged := newEmail != dbUser.Email

		// If email is changed, update the user's email and unset verified status
		if emailChanged {
			// Store the new email in the pending_email field instead of changing the actual email
			dbUser.PendingEmail = newEmail
			// The user remains verified for their current email
			// Only set verified to false after verification with the new email
			dbUser.VerificationToken = generateToken()
			dbUser.VerificationTokenExpiry = time.Now().Add(24 * time.Hour)

			// Update the user in the database
			if err := o.db.UpdateUser(ctx, dbUser); err != nil {
				// Create owner data with error
				ownerData := data.NewOwnerData().
					WithTitle("Edit Profile").
					WithAuthenticated(authenticated).
					WithUser(dbUser).
					WithError("Failed to update user: " + err.Error())

				// Render the profile edit page with error
				c.Status(http.StatusOK)
				owner.Edit(ownerData).Render(c.Request.Context(), c.Writer)
				return
			}

			// Send verification email
			emailService, exists := c.Get("emailService")
			if exists {
				if emailSvc, ok := emailService.(email.EmailService); ok {
					// Get the scheme and host from the request
					scheme := "http"
					if c.Request.TLS != nil || c.Request.Header.Get("X-Forwarded-Proto") == "https" {
						scheme = "https"
					}
					baseURL := fmt.Sprintf("%s://%s", scheme, c.Request.Host)

					if err := emailSvc.SendEmailChangeVerification(newEmail, dbUser.VerificationToken, baseURL); err != nil {
						// Log the error but don't fail the update
						log.Printf("Failed to send verification email: %v", err)
					}
				}
			}

			// Set a cookie with the user's email for the verification sent page
			c.SetCookie("verification_email", newEmail, 3600, "/", "", false, false)

			// Log the user out by clearing the auth cookie
			http.SetCookie(c.Writer, &http.Cookie{
				Name:     "auth-session",
				Value:    "",
				Path:     "/",
				HttpOnly: true,
				MaxAge:   -1,
			})

			// Redirect to verification sent page instead of profile
			c.Redirect(http.StatusSeeOther, "/verification-sent")
			return
		} else {
			// No changes, set a success message
			if setFlash, exists := c.Get("setFlash"); exists {
				setFlash.(func(string))("Your profile has been updated.")
			} else {
				// Fallback to cookie if context function not available
				c.SetCookie("flash", "Your profile has been updated.", 10, "/", "", false, true)
			}

			// Redirect to profile page for non-email changes
			c.Redirect(http.StatusSeeOther, "/owner/profile")
		}
		return
	}

	userInfo, authenticated := authController.GetCurrentUser(c)
	if !authenticated {
		// Set flash message
		if setFlash, exists := c.Get("setFlash"); exists {
			setFlash.(func(string))("You must be logged in to access this page")
		}
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get the user from the database
	ctx := context.Background()
	dbUser, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Parse form data
	if err := c.Request.ParseForm(); err != nil {
		// Create owner data with error
		ownerData := data.NewOwnerData().
			WithTitle("Edit Profile").
			WithAuthenticated(authenticated).
			WithUser(dbUser).
			WithError("Failed to parse form data")

		// Render the profile edit page with error
		owner.Edit(ownerData).Render(c.Request.Context(), c.Writer)
		return
	}

	// Get email from form
	newEmail := c.PostForm("email")

	// Validate email format
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(newEmail) {
		// Create owner data with error
		ownerData := data.NewOwnerData().
			WithTitle("Edit Profile").
			WithAuthenticated(authenticated).
			WithUser(dbUser).
			WithError("Invalid email format")

		// Render the profile edit page with error
		c.Status(http.StatusOK)
		owner.Edit(ownerData).Render(c.Request.Context(), c.Writer)
		return
	}

	// Check if email is being changed
	emailChanged := newEmail != dbUser.Email

	// If email is changed, update the user's email and unset verified status
	if emailChanged {
		// Store the new email in the pending_email field instead of changing the actual email
		dbUser.PendingEmail = newEmail
		// The user remains verified for their current email
		// Only set verified to false after verification with the new email
		dbUser.VerificationToken = generateToken()
		dbUser.VerificationTokenExpiry = time.Now().Add(24 * time.Hour)

		// Update the user in the database
		if err := o.db.UpdateUser(ctx, dbUser); err != nil {
			// Create owner data with error
			ownerData := data.NewOwnerData().
				WithTitle("Edit Profile").
				WithAuthenticated(authenticated).
				WithUser(dbUser).
				WithError("Failed to update user: " + err.Error())

			// Render the profile edit page with error
			c.Status(http.StatusOK)
			owner.Edit(ownerData).Render(c.Request.Context(), c.Writer)
			return
		}

		// Send verification email
		emailService, exists := c.Get("emailService")
		if exists {
			if emailSvc, ok := emailService.(email.EmailService); ok {
				// Get the scheme and host from the request
				scheme := "http"
				if c.Request.TLS != nil || c.Request.Header.Get("X-Forwarded-Proto") == "https" {
					scheme = "https"
				}
				baseURL := fmt.Sprintf("%s://%s", scheme, c.Request.Host)

				if err := emailSvc.SendEmailChangeVerification(newEmail, dbUser.VerificationToken, baseURL); err != nil {
					// Log the error but don't fail the update
					log.Printf("Failed to send verification email: %v", err)
				}
			}
		}

		// Set a cookie with the user's email for the verification sent page
		c.SetCookie("verification_email", newEmail, 3600, "/", "", false, false)

		// Log the user out by clearing the auth cookie
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     "auth-session",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			MaxAge:   -1,
		})

		// Redirect to verification sent page instead of profile
		c.Redirect(http.StatusSeeOther, "/verification-sent")
		return
	} else {
		// No changes, set a success message
		if setFlash, exists := c.Get("setFlash"); exists {
			setFlash.(func(string))("Your profile has been updated.")
		} else {
			// Fallback to cookie if context function not available
			c.SetCookie("flash", "Your profile has been updated.", 10, "/", "", false, true)
		}

		// Redirect to profile page for non-email changes
		c.Redirect(http.StatusSeeOther, "/owner/profile")
	}
}

// Helper function to generate a random token
func generateToken() string {
	token := make([]byte, 32)
	_, err := rand.Read(token)
	if err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(token)
}

// Subscription handles the subscription management page route
func (o *OwnerController) Subscription(c *gin.Context) {
	// Get the current user's authentication status and email
	authController, ok := c.MustGet("authController").(AuthControllerInterface)
	if !ok {
		// Try to cast to the concrete type as a fallback
		concreteAuthController, ok := c.MustGet("authController").(*AuthController)
		if !ok {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}
		userInfo, authenticated := concreteAuthController.GetCurrentUser(c)
		if !authenticated {
			// Set flash message
			if setFlash, exists := c.Get("setFlash"); exists {
				setFlash.(func(string))("You must be logged in to access this page")
			}
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		// Get the user from the database
		ctx := context.Background()
		dbUser, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
		if err != nil {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		// Get the user's payment history
		payments, err := o.db.GetPaymentsByUserID(dbUser.ID)
		if err != nil {
			payments = []models.Payment{} // Empty slice if error
		}

		// Format subscription end date if available
		var subscriptionEndsAt string
		if !dbUser.SubscriptionEndDate.IsZero() {
			subscriptionEndsAt = dbUser.SubscriptionEndDate.Format("January 2, 2006")
		}

		// Create owner data
		ownerData := data.NewOwnerData().
			WithTitle("Subscription Management").
			WithAuthenticated(authenticated).
			WithUser(dbUser).
			WithSubscriptionInfo(
				dbUser.HasActiveSubscription(),
				dbUser.SubscriptionTier,
				subscriptionEndsAt,
			).
			WithPayments(payments)

		// Get authData from context to preserve roles
		if authDataInterface, exists := c.Get("authData"); exists {
			if authData, ok := authDataInterface.(data.AuthData); ok {
				// Use the auth data that already has roles, maintaining our title and other changes
				ownerData.Auth = authData.WithTitle("Subscription Management")
			}
		}

		// Check for flash message from cookie
		if flashCookie, err := c.Cookie("flash"); err == nil && flashCookie != "" {
			// Add flash message to success messages
			ownerData.WithSuccess(flashCookie)
			// Clear the flash cookie
			c.SetCookie("flash", "", -1, "/", "", false, false)
		}

		// Render the subscription management page with the data
		owner.Subscription(ownerData).Render(c.Request.Context(), c.Writer)
		return
	}

	userInfo, authenticated := authController.GetCurrentUser(c)
	if !authenticated {
		// Set flash message
		if setFlash, exists := c.Get("setFlash"); exists {
			setFlash.(func(string))("You must be logged in to access this page")
		}
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get the user from the database
	ctx := context.Background()
	dbUser, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get the user's payment history
	payments, err := o.db.GetPaymentsByUserID(dbUser.ID)
	if err != nil {
		payments = []models.Payment{} // Empty slice if error
	}

	// Format subscription end date if available
	var subscriptionEndsAt string
	if !dbUser.SubscriptionEndDate.IsZero() {
		subscriptionEndsAt = dbUser.SubscriptionEndDate.Format("January 2, 2006")
	}

	// Create owner data
	ownerData := data.NewOwnerData().
		WithTitle("Subscription Management").
		WithAuthenticated(authenticated).
		WithUser(dbUser).
		WithSubscriptionInfo(
			dbUser.HasActiveSubscription(),
			dbUser.SubscriptionTier,
			subscriptionEndsAt,
		).
		WithPayments(payments)

	// Get authData from context to preserve roles
	if authDataInterface, exists := c.Get("authData"); exists {
		if authData, ok := authDataInterface.(data.AuthData); ok {
			// Use the auth data that already has roles, maintaining our title and other changes
			ownerData.Auth = authData.WithTitle("Subscription Management")
		}
	}

	// Check for flash message from cookie
	if flashCookie, err := c.Cookie("flash"); err == nil && flashCookie != "" {
		// Add flash message to success messages
		ownerData.WithSuccess(flashCookie)
		// Clear the flash cookie
		c.SetCookie("flash", "", -1, "/", "", false, false)
	}

	// Render the subscription management page with the data
	owner.Subscription(ownerData).Render(c.Request.Context(), c.Writer)
}

// DeleteAccountConfirm renders the confirmation page for deleting an account
func (o *OwnerController) DeleteAccountConfirm(c *gin.Context) {
	// Get the current user's authentication status and email
	authController, ok := c.MustGet("authController").(AuthControllerInterface)
	if !ok {
		// Try to cast to the concrete type as a fallback
		concreteAuthController, ok := c.MustGet("authController").(*AuthController)
		if !ok {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}
		userInfo, authenticated := concreteAuthController.GetCurrentUser(c)
		if !authenticated {
			// Set flash message
			if setFlash, exists := c.Get("setFlash"); exists {
				setFlash.(func(string))("You must be logged in to access this page")
			}
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		// Get the user from the database
		ctx := context.Background()
		dbUser, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
		if err != nil {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		// Format subscription end date if available
		var subscriptionEndsAt string
		if !dbUser.SubscriptionEndDate.IsZero() {
			subscriptionEndsAt = dbUser.SubscriptionEndDate.Format("January 2, 2006")
		}

		// Create owner data
		ownerData := data.NewOwnerData().
			WithTitle("Delete Account").
			WithAuthenticated(authenticated).
			WithUser(dbUser).
			WithSubscriptionInfo(
				dbUser.HasActiveSubscription(),
				dbUser.SubscriptionTier,
				subscriptionEndsAt,
			)

		// Render the delete confirmation page
		owner.DeleteConfirm(ownerData).Render(c.Request.Context(), c.Writer)
		return
	}

	userInfo, authenticated := authController.GetCurrentUser(c)
	if !authenticated {
		// Set flash message
		if setFlash, exists := c.Get("setFlash"); exists {
			setFlash.(func(string))("You must be logged in to access this page")
		}
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get the user from the database
	ctx := context.Background()
	dbUser, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Format subscription end date if available
	var subscriptionEndsAt string
	if !dbUser.SubscriptionEndDate.IsZero() {
		subscriptionEndsAt = dbUser.SubscriptionEndDate.Format("January 2, 2006")
	}

	// Create owner data
	ownerData := data.NewOwnerData().
		WithTitle("Delete Account").
		WithAuthenticated(authenticated).
		WithUser(dbUser).
		WithSubscriptionInfo(
			dbUser.HasActiveSubscription(),
			dbUser.SubscriptionTier,
			subscriptionEndsAt,
		)

	// Render the delete confirmation page
	owner.DeleteConfirm(ownerData).Render(c.Request.Context(), c.Writer)
}

// DeleteAccountHandler handles the POST request to delete an account
func (o *OwnerController) DeleteAccountHandler(c *gin.Context) {
	// Get the current user's authentication status and email
	authController, ok := c.MustGet("authController").(AuthControllerInterface)
	if !ok {
		// Try to cast to the concrete type as a fallback
		concreteAuthController, ok := c.MustGet("authController").(*AuthController)
		if !ok {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}
		userInfo, authenticated := concreteAuthController.GetCurrentUser(c)
		if !authenticated {
			// Set flash message
			if setFlash, exists := c.Get("setFlash"); exists {
				setFlash.(func(string))("You must be logged in to access this page")
			}
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		// Get the user from the database
		ctx := context.Background()
		dbUser, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
		if err != nil {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		// Check if the user confirmed the deletion
		confirm := c.PostForm("confirm")
		if confirm != "true" {
			// User did not confirm, redirect back to profile
			c.Redirect(http.StatusSeeOther, "/owner/profile")
			return
		}

		// Soft-delete the user (GORM will use DeletedAt)
		if err := o.db.GetDB().Delete(&dbUser).Error; err != nil {
			// Set error flash message
			if setFlash, exists := c.Get("setFlash"); exists {
				setFlash.(func(string))("Failed to delete account: " + err.Error())
			}
			c.Redirect(http.StatusSeeOther, "/owner/profile")
			return
		}

		// Invalidate the user's session - use the correct cookie name "auth-session"
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     "auth-session",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			MaxAge:   -1, // Delete the cookie
		})

		// Set flash message
		c.SetCookie("flash", "Your account has been deleted. Please come back any time!", 10, "/", "", false, false)

		// Redirect to home page
		c.Redirect(http.StatusSeeOther, "/")
		return
	}

	userInfo, authenticated := authController.GetCurrentUser(c)
	if !authenticated {
		// Set flash message
		if setFlash, exists := c.Get("setFlash"); exists {
			setFlash.(func(string))("You must be logged in to access this page")
		}
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get the user from the database
	ctx := context.Background()
	dbUser, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Check if the user confirmed the deletion
	confirm := c.PostForm("confirm")
	if confirm != "true" {
		// User did not confirm, redirect back to profile
		c.Redirect(http.StatusSeeOther, "/owner/profile")
		return
	}

	// Soft-delete the user (GORM will use DeletedAt)
	if err := o.db.GetDB().Delete(&dbUser).Error; err != nil {
		// Set error flash message
		if setFlash, exists := c.Get("setFlash"); exists {
			setFlash.(func(string))("Failed to delete account: " + err.Error())
		}
		c.Redirect(http.StatusSeeOther, "/owner/profile")
		return
	}

	// Invalidate the user's session - use the correct cookie name "auth-session"
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "auth-session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1, // Delete the cookie
	})

	// Set flash message
	c.SetCookie("flash", "Your account has been deleted. Please come back any time!", 10, "/", "", false, false)

	// Redirect to home page
	c.Redirect(http.StatusSeeOther, "/")
}
