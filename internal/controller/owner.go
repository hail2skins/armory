package controller

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/cmd/web/views/owner"
	gunView "github.com/hail2skins/armory/cmd/web/views/owner/gun"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
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

		// Get the user's guns
		var guns []models.Gun
		// Use the database service's GetDB method to get the underlying gorm.DB
		db := o.db.GetDB()
		if err := db.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").Where("owner_id = ?", dbUser.ID).Find(&guns).Error; err != nil {
			guns = []models.Gun{}
		}

		// Apply free tier limit if needed
		var totalGuns int
		if dbUser.SubscriptionTier == "free" && len(guns) > 2 {
			totalGuns = len(guns)
			if len(guns) > 0 {
				guns[0].HasMoreGuns = true
				guns[0].TotalGuns = totalGuns
			}
			guns = guns[:2]
		}

		// Format subscription end date if available
		var subscriptionEndsAt string
		if !dbUser.SubscriptionEndDate.IsZero() {
			subscriptionEndsAt = dbUser.SubscriptionEndDate.Format("January 2, 2006")
		}

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
			)

		// If the user has more guns than shown, add a message
		if totalGuns > 2 && dbUser.SubscriptionTier == "free" {
			ownerData.WithError("Please subscribe to see the rest of your collection")
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

	// Get the user's guns
	var guns []models.Gun
	// Use the database service's GetDB method to get the underlying gorm.DB
	db := o.db.GetDB()
	if err := db.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").Where("owner_id = ?", dbUser.ID).Find(&guns).Error; err != nil {
		guns = []models.Gun{}
	}

	// Apply free tier limit if needed
	var totalGuns int
	if dbUser.SubscriptionTier == "free" && len(guns) > 2 {
		totalGuns = len(guns)
		if len(guns) > 0 {
			guns[0].HasMoreGuns = true
			guns[0].TotalGuns = totalGuns
		}
		guns = guns[:2]
	}

	// Format subscription end date if available
	var subscriptionEndsAt string
	if !dbUser.SubscriptionEndDate.IsZero() {
		subscriptionEndsAt = dbUser.SubscriptionEndDate.Format("January 2, 2006")
	}

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
		)

	// If the user has more guns than shown, add a message
	if totalGuns > 2 && dbUser.SubscriptionTier == "free" {
		ownerData.WithError("Please subscribe to see the rest of your collection")
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

		// Parse form data
		name := c.PostForm("name")
		serialNumber := c.PostForm("serial_number")
		weaponTypeIDStr := c.PostForm("weapon_type_id")
		caliberIDStr := c.PostForm("caliber_id")
		manufacturerIDStr := c.PostForm("manufacturer_id")
		acquiredDateStr := c.PostForm("acquired_date")

		// Validate required fields
		errors := make(map[string]string)
		if name == "" {
			errors["name"] = "Name is required"
		}
		if weaponTypeIDStr == "" {
			errors["weapon_type_id"] = "Weapon type is required"
		}
		if caliberIDStr == "" {
			errors["caliber_id"] = "Caliber is required"
		}
		if manufacturerIDStr == "" {
			errors["manufacturer_id"] = "Manufacturer is required"
		}

		// If there are validation errors, re-render the form with errors
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

		// Convert IDs to uint
		weaponTypeID, _ := strconv.ParseUint(weaponTypeIDStr, 10, 64)
		caliberID, _ := strconv.ParseUint(caliberIDStr, 10, 64)
		manufacturerID, _ := strconv.ParseUint(manufacturerIDStr, 10, 64)

		// Parse acquired date if provided
		var acquiredDate *time.Time
		if acquiredDateStr != "" {
			parsedDate, err := time.Parse("2006-01-02", acquiredDateStr)
			if err == nil {
				acquiredDate = &parsedDate
			}
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

	// Parse form data
	name := c.PostForm("name")
	serialNumber := c.PostForm("serial_number")
	weaponTypeIDStr := c.PostForm("weapon_type_id")
	caliberIDStr := c.PostForm("caliber_id")
	manufacturerIDStr := c.PostForm("manufacturer_id")
	acquiredDateStr := c.PostForm("acquired_date")

	// Validate required fields
	errors := make(map[string]string)
	if name == "" {
		errors["name"] = "Name is required"
	}
	if weaponTypeIDStr == "" {
		errors["weapon_type_id"] = "Weapon type is required"
	}
	if caliberIDStr == "" {
		errors["caliber_id"] = "Caliber is required"
	}
	if manufacturerIDStr == "" {
		errors["manufacturer_id"] = "Manufacturer is required"
	}

	// If there are validation errors, re-render the form with errors
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

	// Convert IDs to uint
	weaponTypeID, _ := strconv.ParseUint(weaponTypeIDStr, 10, 64)
	caliberID, _ := strconv.ParseUint(caliberIDStr, 10, 64)
	manufacturerID, _ := strconv.ParseUint(manufacturerIDStr, 10, 64)

	// Parse acquired date if provided
	var acquiredDate *time.Time
	if acquiredDateStr != "" {
		parsedDate, err := time.Parse("2006-01-02", acquiredDateStr)
		if err == nil {
			acquiredDate = &parsedDate
		}
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
			WithTitle("Firearm Details").
			WithAuthenticated(authenticated).
			WithUser(dbUser).
			WithGun(&gun).
			WithSubscriptionInfo(
				dbUser.HasActiveSubscription(),
				dbUser.SubscriptionTier,
				subscriptionEndsAt,
			)

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
		WithTitle("Firearm Details").
		WithAuthenticated(authenticated).
		WithUser(dbUser).
		WithGun(&gun).
		WithSubscriptionInfo(
			dbUser.HasActiveSubscription(),
			dbUser.SubscriptionTier,
			subscriptionEndsAt,
		)

	// Check for flash message from cookie
	if flashCookie, err := c.Cookie("flash"); err == nil && flashCookie != "" {
		// Clear the flash cookie
		c.SetCookie("flash", "", -1, "/", "", false, true)
		ownerData.WithSuccess(flashCookie)
	}

	// Render the gun show page
	gunView.Show(ownerData).Render(c.Request.Context(), c.Writer)
}

// Edit handles the edit gun page
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
			WithTitle("Edit Firearm").
			WithAuthenticated(true).
			WithUser(user).
			WithGun(&gun).
			WithWeaponTypes(weaponTypes).
			WithCalibers(calibers).
			WithManufacturers(manufacturers)

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
		WithTitle("Edit Firearm").
		WithAuthenticated(true).
		WithUser(user).
		WithGun(&gun).
		WithWeaponTypes(weaponTypes).
		WithCalibers(calibers).
		WithManufacturers(manufacturers)

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

		// Update the gun
		gun.Name = name
		gun.SerialNumber = serialNumber
		gun.WeaponTypeID = uint(weaponTypeID)
		gun.CaliberID = uint(caliberID)
		gun.ManufacturerID = uint(manufacturerID)

		// Parse the acquired date if provided
		if acquiredDateStr != "" {
			acquiredDate, err := time.Parse("2006-01-02", acquiredDateStr)
			if err != nil {
				c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid acquired date"})
				return
			}
			gun.Acquired = &acquiredDate
		} else {
			gun.Acquired = nil
		}

		// Save the gun
		if err := models.UpdateGun(db, &gun); err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to update gun"})
			return
		}

		// Reload the gun with its relationships to ensure they're updated
		if err := db.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").First(&gun, gun.ID).Error; err != nil {
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

	// Update the gun
	gun.Name = name
	gun.SerialNumber = serialNumber
	gun.WeaponTypeID = uint(weaponTypeID)
	gun.CaliberID = uint(caliberID)
	gun.ManufacturerID = uint(manufacturerID)

	// Parse the acquired date if provided
	if acquiredDateStr != "" {
		acquiredDate, err := time.Parse("2006-01-02", acquiredDateStr)
		if err != nil {
			c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid acquired date"})
			return
		}
		gun.Acquired = &acquiredDate
	} else {
		gun.Acquired = nil
	}

	// Save the gun
	if err := models.UpdateGun(db, &gun); err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to update gun"})
		return
	}

	// Reload the gun with its relationships to ensure they're updated
	if err := db.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").First(&gun, gun.ID).Error; err != nil {
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

		// Get all guns for this owner
		var guns []models.Gun
		db := o.db.GetDB()
		if err := db.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").Where("owner_id = ?", dbUser.ID).Order("name ASC").Find(&guns).Error; err != nil {
			guns = []models.Gun{}
		}

		// Apply free tier limit if needed
		var totalGuns int
		if dbUser.SubscriptionTier == "free" && len(guns) > 2 {
			totalGuns = len(guns)
			if len(guns) > 0 {
				guns[0].HasMoreGuns = true
				guns[0].TotalGuns = totalGuns
			}
			guns = guns[:2]
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
			)

		// If the user has more guns than shown, add a message
		if totalGuns > 2 && dbUser.SubscriptionTier == "free" {
			ownerData.WithError("Please subscribe to see the rest of your collection")
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

	// Get all guns for this owner
	var guns []models.Gun
	db := o.db.GetDB()
	if err := db.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").Where("owner_id = ?", dbUser.ID).Order("name ASC").Find(&guns).Error; err != nil {
		guns = []models.Gun{}
	}

	// Apply free tier limit if needed
	var totalGuns int
	if dbUser.SubscriptionTier == "free" && len(guns) > 2 {
		totalGuns = len(guns)
		if len(guns) > 0 {
			guns[0].HasMoreGuns = true
			guns[0].TotalGuns = totalGuns
		}
		guns = guns[:2]
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
		)

	// If the user has more guns than shown, add a message
	if totalGuns > 2 && dbUser.SubscriptionTier == "free" {
		ownerData.WithError("Please subscribe to see the rest of your collection")
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
			WithTitle("Your Profile").
			WithAuthenticated(authenticated).
			WithUser(dbUser).
			WithSubscriptionInfo(
				dbUser.HasActiveSubscription(),
				dbUser.SubscriptionTier,
				subscriptionEndsAt,
			)

		// Check for flash message from cookie
		if flashCookie, err := c.Cookie("flash"); err == nil && flashCookie != "" {
			// Add flash message to success messages
			ownerData.WithSuccess(flashCookie)
			// Clear the flash cookie
			c.SetCookie("flash", "", -1, "/", "", false, false)
		}

		// Render the owner profile page with the data
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
		WithTitle("Your Profile").
		WithAuthenticated(authenticated).
		WithUser(dbUser).
		WithSubscriptionInfo(
			dbUser.HasActiveSubscription(),
			dbUser.SubscriptionTier,
			subscriptionEndsAt,
		)

	// Check for flash message from cookie
	if flashCookie, err := c.Cookie("flash"); err == nil && flashCookie != "" {
		// Add flash message to success messages
		ownerData.WithSuccess(flashCookie)
		// Clear the flash cookie
		c.SetCookie("flash", "", -1, "/", "", false, false)
	}

	// Render the owner profile page with the data
	owner.Profile(ownerData).Render(c.Request.Context(), c.Writer)
}
