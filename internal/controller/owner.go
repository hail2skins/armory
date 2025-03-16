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
		if !dbUser.HasActiveSubscription() && len(guns) > 2 {
			if len(guns) > 0 {
				guns[0].HasMoreGuns = true
				guns[0].TotalGuns = len(guns)
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
	if !dbUser.HasActiveSubscription() && len(guns) > 2 {
		if len(guns) > 0 {
			guns[0].HasMoreGuns = true
			guns[0].TotalGuns = len(guns)
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

// Edit handles the edit gun route
func (o *OwnerController) Edit(c *gin.Context) {
	// This will be implemented later
	c.JSON(200, gin.H{"message": "Edit gun"})
}

// Update handles the update gun route
func (o *OwnerController) Update(c *gin.Context) {
	// This will be implemented later
	c.JSON(200, gin.H{"message": "Update gun"})
}

// Delete handles the delete gun route
func (o *OwnerController) Delete(c *gin.Context) {
	// This will be implemented later
	c.JSON(200, gin.H{"message": "Delete gun"})
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

	// Check for flash message from cookie
	if flashCookie, err := c.Cookie("flash"); err == nil && flashCookie != "" {
		// Clear the flash cookie
		c.SetCookie("flash", "", -1, "/", "", false, true)
		ownerData.WithSuccess(flashCookie)
	}

	// Render the arsenal page
	gunView.Arsenal(ownerData).Render(c.Request.Context(), c.Writer)
}
