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

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/cmd/web/views/owner"
	gunView "github.com/hail2skins/armory/cmd/web/views/owner/gun"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/logger"
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

			// Re-fetch roles from Casbin to ensure they're up to date
			if casbinAuth, exists := c.Get("casbinAuth"); exists && casbinAuth != nil {
				if ca, ok := casbinAuth.(interface{ GetUserRoles(string) []string }); ok {
					roles := ca.GetUserRoles(userInfo.GetUserName())
					logger.Info("Casbin roles for user in owner page", map[string]interface{}{
						"email": userInfo.GetUserName(),
						"roles": roles,
					})
					ownerData.Auth = ownerData.Auth.WithRoles(roles)
				}
			}
		}
	}

	// If the user has more guns than shown, add a message
	if showingFreeLimit {
		ownerData.WithError(fmt.Sprintf("Free Tier shows all %d guns in your collection, but only allows you to add up to 2 guns. Please subscribe to add more firearms.", totalUserGuns))
	}

	// Check for flash messages from session
	session := sessions.Default(c)
	flashes := session.Flashes()
	if len(flashes) > 0 {
		session.Save()
		for _, flash := range flashes {
			if flashMsg, ok := flash.(string); ok {
				ownerData.WithSuccess(flashMsg)
			}
		}
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

		// Check for flash messages from session
		session := sessions.Default(c)
		flashes := session.Flashes()
		if len(flashes) > 0 {
			session.Save()
			for _, flash := range flashes {
				if flashMsg, ok := flash.(string); ok {
					ownerData.WithSuccess(flashMsg)
				}
			}
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

			// Re-fetch roles from Casbin to ensure they're up to date
			if casbinAuth, exists := c.Get("casbinAuth"); exists && casbinAuth != nil {
				if ca, ok := casbinAuth.(interface{ GetUserRoles(string) []string }); ok {
					roles := ca.GetUserRoles(userInfo.GetUserName())
					logger.Info("Casbin roles for user in new gun page", map[string]interface{}{
						"email": userInfo.GetUserName(),
						"roles": roles,
					})
					ownerData.Auth = ownerData.Auth.WithRoles(roles)
				}
			}
		}
	}

	// Check for flash messages from session
	session := sessions.Default(c)
	flashes := session.Flashes()
	if len(flashes) > 0 {
		session.Save()
		for _, flash := range flashes {
			if flashMsg, ok := flash.(string); ok {
				ownerData.WithSuccess(flashMsg)
			}
		}
	}

	// Render the new gun form with the data
	gunView.New(ownerData).Render(c.Request.Context(), c.Writer)
}

// Create handles the create gun route
func (o *OwnerController) Create(c *gin.Context) {
	logger.Info("Create method called - IMPROVED VERSION", nil)

	// Get all form values
	name := c.PostForm("name")
	serialNumber := c.PostForm("serial_number")
	acquiredDateStr := c.PostForm("acquired_date")
	weaponTypeIDStr := c.PostForm("weapon_type_id")
	caliberIDStr := c.PostForm("caliber_id")
	manufacturerIDStr := c.PostForm("manufacturer_id")
	paidStr := c.PostForm("paid")
	purpose := c.PostForm("purpose")

	logger.Info("Form data received", map[string]interface{}{
		"name":            name,
		"serial_number":   serialNumber,
		"acquired":        acquiredDateStr,
		"weapon_type_id":  weaponTypeIDStr,
		"caliber_id":      caliberIDStr,
		"manufacturer_id": manufacturerIDStr,
		"paid":            paidStr,
		"purpose":         purpose,
	})

	// Get current user
	authController, ok := c.MustGet("authController").(*AuthController)
	if !ok {
		logger.Error("Invalid auth controller type", nil, nil)
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	userInfo, authenticated := authController.GetCurrentUser(c)
	if !authenticated {
		logger.Error("User not authenticated", nil, nil)
		if setFlash, exists := c.Get("setFlash"); exists {
			setFlash.(func(string))("You must be logged in to access this page")
		}
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get user from database
	ctx := context.Background()
	dbUser, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
	if err != nil {
		logger.Error("Failed to get user", err, nil)
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Check if user is on free tier and already has 2 guns
	if dbUser.SubscriptionTier == "free" {
		var count int64
		db := o.db.GetDB()
		db.Model(&models.Gun{}).Where("owner_id = ?", dbUser.ID).Count(&count)

		if count >= 2 {
			if setFlash, exists := c.Get("setFlash"); exists {
				setFlash.(func(string))("You must be subscribed to add more to your arsenal")
			}
			c.Redirect(http.StatusSeeOther, "/pricing")
			return
		}
	}

	// Initialize error map for form validation
	errors := make(map[string]string)

	// Parse weapon type ID
	weaponTypeID, err := strconv.Atoi(weaponTypeIDStr)
	if err != nil {
		errors["weapon_type_id"] = "Valid weapon type is required"
	}

	// Parse caliber ID
	caliberID, err := strconv.Atoi(caliberIDStr)
	if err != nil {
		errors["caliber_id"] = "Valid caliber is required"
	}

	// Parse manufacturer ID
	manufacturerID, err := strconv.Atoi(manufacturerIDStr)
	if err != nil {
		errors["manufacturer_id"] = "Valid manufacturer is required"
	}

	// Parse acquired date if provided
	var acquiredDate *time.Time
	if acquiredDateStr != "" {
		parsedDate, err := time.Parse("2006-01-02", acquiredDateStr)
		if err != nil {
			errors["acquired_date"] = "Invalid date format, use MM-DD-YYYY"
		} else {
			// Check if date is in the future
			if parsedDate.After(time.Now()) {
				errors["acquired_date"] = "Acquisition date cannot be in the future"
			} else {
				acquiredDate = &parsedDate
			}
		}
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

	// If there are parsing errors, re-render the form with errors
	if len(errors) > 0 {
		logger.Info("Parsing errors found", map[string]interface{}{"errors": errors})

		// Get reference data for the form
		weaponTypes, _ := o.db.FindAllWeaponTypes()
		calibers, _ := o.db.FindAllCalibers()
		manufacturers, _ := o.db.FindAllManufacturers()

		// Create owner data with errors
		ownerData := data.NewOwnerData().
			WithTitle("Add New Firearm").
			WithAuthenticated(true).
			WithUser(dbUser).
			WithWeaponTypes(weaponTypes).
			WithCalibers(calibers).
			WithManufacturers(manufacturers).
			WithFormErrors(errors)

		// Add auth data if available
		if authDataInterface, exists := c.Get("authData"); exists {
			if authData, ok := authDataInterface.(data.AuthData); ok {
				ownerData.Auth = authData.WithTitle("Add New Firearm")
			}
		}

		// Add CSRF token to view data if not already provided by authData
		if ownerData.Auth.CSRFToken == "" {
			if csrfTokenInterface, exists := c.Get("csrf_token"); exists {
				if csrfToken, ok := csrfTokenInterface.(string); ok {
					ownerData.Auth = ownerData.Auth.WithCSRFToken(csrfToken)
				}
			}
		}

		// Render the form with errors
		gunView.New(ownerData).Render(c.Request.Context(), c.Writer)
		return
	}

	// Create a new gun
	newGun := &models.Gun{
		Name:           name,
		SerialNumber:   serialNumber,
		Purpose:        purpose,
		Acquired:       acquiredDate,
		WeaponTypeID:   uint(weaponTypeID),
		CaliberID:      uint(caliberID),
		ManufacturerID: uint(manufacturerID),
		OwnerID:        dbUser.ID,
		Paid:           paidAmount,
	}

	logger.Info("About to create gun with validation", map[string]interface{}{
		"gun": map[string]interface{}{
			"name":            newGun.Name,
			"owner_id":        newGun.OwnerID,
			"weapon_type_id":  newGun.WeaponTypeID,
			"caliber_id":      newGun.CaliberID,
			"manufacturer_id": newGun.ManufacturerID,
		},
	})

	// Use the new validation-enabled create function
	db := o.db.GetDB()
	err = models.CreateGunWithValidation(db, newGun)
	if err != nil {
		logger.Error("Gun validation or creation failed", err, map[string]interface{}{
			"error_details": err.Error(),
		})

		// Map model validation errors to form errors
		switch err {
		case models.ErrGunNameTooLong:
			errors["name"] = "Name cannot exceed 100 characters"
		case models.ErrNegativePrice:
			errors["paid"] = "Price cannot be negative"
		case models.ErrFutureDate:
			errors["acquired_date"] = "Acquisition date cannot be in the future"
		case models.ErrInvalidWeaponType:
			errors["weapon_type_id"] = "Selected weapon type doesn't exist"
		case models.ErrInvalidCaliber:
			errors["caliber_id"] = "Selected caliber doesn't exist"
		case models.ErrInvalidManufacturer:
			errors["manufacturer_id"] = "Selected manufacturer doesn't exist"
		default:
			// Generic database error
			errors["general"] = "Failed to create gun: " + err.Error()
		}

		// Get reference data for the form
		weaponTypes, _ := o.db.FindAllWeaponTypes()
		calibers, _ := o.db.FindAllCalibers()
		manufacturers, _ := o.db.FindAllManufacturers()

		// Create owner data with errors
		ownerData := data.NewOwnerData().
			WithTitle("Add New Firearm").
			WithAuthenticated(true).
			WithUser(dbUser).
			WithWeaponTypes(weaponTypes).
			WithCalibers(calibers).
			WithManufacturers(manufacturers).
			WithFormErrors(errors)

		// Add auth data if available
		if authDataInterface, exists := c.Get("authData"); exists {
			if authData, ok := authDataInterface.(data.AuthData); ok {
				ownerData.Auth = authData.WithTitle("Add New Firearm")
			}
		}

		// Add CSRF token to view data if not already provided by authData
		if ownerData.Auth.CSRFToken == "" {
			if csrfTokenInterface, exists := c.Get("csrf_token"); exists {
				if csrfToken, ok := csrfTokenInterface.(string); ok {
					ownerData.Auth = ownerData.Auth.WithCSRFToken(csrfToken)
				}
			}
		}

		// Render the form with the error
		gunView.New(ownerData).Render(c.Request.Context(), c.Writer)
		return
	}

	// Success path
	logger.Info("Gun created successfully", map[string]interface{}{
		"gun_id":   newGun.ID,
		"gun_name": newGun.Name,
	})

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
			// Use session flash message instead of HTML rendering
			session := sessions.Default(c)
			session.AddFlash("That's not your gun!")
			session.Save()

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

				// Re-fetch roles from Casbin to ensure they're up to date
				if casbinAuth, exists := c.Get("casbinAuth"); exists && casbinAuth != nil {
					if ca, ok := casbinAuth.(interface{ GetUserRoles(string) []string }); ok {
						roles := ca.GetUserRoles(userInfo.GetUserName())
						logger.Info("Casbin roles for user in gun show page", map[string]interface{}{
							"email": userInfo.GetUserName(),
							"roles": roles,
						})
						ownerData.Auth = ownerData.Auth.WithRoles(roles)
					}
				}
			}
		}

		// Check for flash messages from session
		session := sessions.Default(c)
		flashes := session.Flashes()
		if len(flashes) > 0 {
			session.Save()
			for _, flash := range flashes {
				if flashMsg, ok := flash.(string); ok {
					ownerData.WithSuccess(flashMsg)
				}
			}
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
		// Use session flash message instead of HTML rendering
		session := sessions.Default(c)
		session.AddFlash("That's not your gun!")
		session.Save()

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

			// Re-fetch roles from Casbin to ensure they're up to date
			if casbinAuth, exists := c.Get("casbinAuth"); exists && casbinAuth != nil {
				if ca, ok := casbinAuth.(interface{ GetUserRoles(string) []string }); ok {
					roles := ca.GetUserRoles(userInfo.GetUserName())
					logger.Info("Casbin roles for user in gun show page", map[string]interface{}{
						"email": userInfo.GetUserName(),
						"roles": roles,
					})
					ownerData.Auth = ownerData.Auth.WithRoles(roles)
				}
			}
		}
	}

	// Check for flash messages from session
	session := sessions.Default(c)
	flashes := session.Flashes()
	if len(flashes) > 0 {
		session.Save()
		for _, flash := range flashes {
			if flashMsg, ok := flash.(string); ok {
				ownerData.WithSuccess(flashMsg)
			}
		}
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
			// Use session flash message instead of HTML rendering
			session := sessions.Default(c)
			session.AddFlash("An error occurred. Please try again.")
			session.Save()

			c.Redirect(http.StatusSeeOther, "/owner")
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
			// Use session flash message instead of HTML rendering
			session := sessions.Default(c)
			session.AddFlash("An error occurred loading weapon types. Please try again.")
			session.Save()

			c.Redirect(http.StatusSeeOther, "/owner")
			return
		}

		var calibers []models.Caliber
		if err := db.Order("popularity DESC").Find(&calibers).Error; err != nil {
			// Use session flash message instead of HTML rendering
			session := sessions.Default(c)
			session.AddFlash("An error occurred loading calibers. Please try again.")
			session.Save()

			c.Redirect(http.StatusSeeOther, "/owner")
			return
		}

		var manufacturers []models.Manufacturer
		if err := db.Order("popularity DESC").Find(&manufacturers).Error; err != nil {
			// Use session flash message instead of HTML rendering
			session := sessions.Default(c)
			session.AddFlash("An error occurred loading manufacturers. Please try again.")
			session.Save()

			c.Redirect(http.StatusSeeOther, "/owner")
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
		// Use session flash message instead of HTML rendering
		session := sessions.Default(c)
		session.AddFlash("An error occurred. Please try again.")
		session.Save()

		c.Redirect(http.StatusSeeOther, "/owner")
		return
	}

	// Get the gun ID from the URL
	gunID := c.Param("id")

	// Get the gun from the database
	var gun models.Gun
	db := o.db.GetDB()
	if err := db.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").Where("id = ? AND owner_id = ?", gunID, user.ID).First(&gun).Error; err != nil {
		// Use session flash message instead of HTML rendering
		session := sessions.Default(c)
		session.AddFlash("That's not your gun!")
		session.Save()

		c.Redirect(http.StatusSeeOther, "/owner")
		return
	}

	// Get all weapon types, calibers, and manufacturers
	var weaponTypes []models.WeaponType
	if err := db.Order("popularity DESC").Find(&weaponTypes).Error; err != nil {
		// Use session flash message instead of HTML rendering
		session := sessions.Default(c)
		session.AddFlash("An error occurred loading weapon types. Please try again.")
		session.Save()

		c.Redirect(http.StatusSeeOther, "/owner")
		return
	}

	var calibers []models.Caliber
	if err := db.Order("popularity DESC").Find(&calibers).Error; err != nil {
		// Use session flash message instead of HTML rendering
		session := sessions.Default(c)
		session.AddFlash("An error occurred loading calibers. Please try again.")
		session.Save()

		c.Redirect(http.StatusSeeOther, "/owner")
		return
	}

	var manufacturers []models.Manufacturer
	if err := db.Order("popularity DESC").Find(&manufacturers).Error; err != nil {
		// Use session flash message instead of HTML rendering
		session := sessions.Default(c)
		session.AddFlash("An error occurred loading manufacturers. Please try again.")
		session.Save()

		c.Redirect(http.StatusSeeOther, "/owner")
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

			// Re-fetch roles from Casbin to ensure they're up to date
			if casbinAuth, exists := c.Get("casbinAuth"); exists && casbinAuth != nil {
				if ca, ok := casbinAuth.(interface{ GetUserRoles(string) []string }); ok {
					roles := ca.GetUserRoles(userInfo.GetUserName())
					logger.Info("Casbin roles for user in edit gun page", map[string]interface{}{
						"email": userInfo.GetUserName(),
						"roles": roles,
					})

					viewData.Auth = viewData.Auth.WithRoles(roles)
				}
			}
		}
	}

	// Render the edit form
	c.Status(http.StatusOK)
	gunView.Edit(viewData).Render(context.Background(), c.Writer)
}

// Update handles the update gun route
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
			// Use session flash message instead of HTML error
			session := sessions.Default(c)
			session.AddFlash("Failed to get user information")
			session.Save()
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		// Get the gun ID from the URL
		gunID := c.Param("id")

		// Get the gun from the database
		var gun models.Gun
		db := o.db.GetDB()
		if err := db.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").Where("id = ? AND owner_id = ?", gunID, user.ID).First(&gun).Error; err != nil {
			// Use session flash message instead of HTML error
			session := sessions.Default(c)
			session.AddFlash("That's not your gun!")
			session.Save()
			c.Redirect(http.StatusSeeOther, "/owner")
			return
		}

		// Parse the form
		if err := c.Request.ParseForm(); err != nil {
			// Use session flash message instead of HTML error
			session := sessions.Default(c)
			session.AddFlash("Invalid form data")
			session.Save()
			c.Redirect(http.StatusSeeOther, "/owner")
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
		purpose := c.PostForm("purpose")

		// Initialize form errors map
		formErrors := make(map[string]string)

		// Parse IDs with basic format validation
		weaponTypeID, err := strconv.ParseUint(weaponTypeIDStr, 10, 64)
		if err != nil {
			formErrors["weapon_type_id"] = "Invalid weapon type format"
		}

		caliberID, err := strconv.ParseUint(caliberIDStr, 10, 64)
		if err != nil {
			formErrors["caliber_id"] = "Invalid caliber format"
		}

		manufacturerID, err := strconv.ParseUint(manufacturerIDStr, 10, 64)
		if err != nil {
			formErrors["manufacturer_id"] = "Invalid manufacturer format"
		}

		// Parse acquired date if provided
		var acquiredDate *time.Time
		if acquiredDateStr != "" {
			parsedDate, err := time.Parse("2006-01-02", acquiredDateStr)
			if err != nil {
				formErrors["acquired_date"] = "Invalid date format, use MM-DD-YYYY"
			} else {
				// Check if date is in the future
				if parsedDate.After(time.Now()) {
					formErrors["acquired_date"] = "Acquisition date cannot be in the future"
				} else {
					acquiredDate = &parsedDate
				}
			}
		}

		// Parse paid amount if provided
		var paidAmount *float64
		if paidStr != "" {
			paid, err := strconv.ParseFloat(paidStr, 64)
			if err != nil {
				formErrors["paid"] = "Invalid amount format"
			} else {
				paidAmount = &paid
			}
		}

		// If there are basic parsing errors, return to the form
		if len(formErrors) > 0 {
			// Get all weapon types, calibers, and manufacturers
			var weaponTypes []models.WeaponType
			if err := db.Order("popularity DESC").Find(&weaponTypes).Error; err != nil {
				session := sessions.Default(c)
				session.AddFlash("Failed to get weapon types")
				session.Save()
				c.Redirect(http.StatusSeeOther, "/owner")
				return
			}

			var calibers []models.Caliber
			if err := db.Order("popularity DESC").Find(&calibers).Error; err != nil {
				session := sessions.Default(c)
				session.AddFlash("Failed to get calibers")
				session.Save()
				c.Redirect(http.StatusSeeOther, "/owner")
				return
			}

			var manufacturers []models.Manufacturer
			if err := db.Order("popularity DESC").Find(&manufacturers).Error; err != nil {
				session := sessions.Default(c)
				session.AddFlash("Failed to get manufacturers")
				session.Save()
				c.Redirect(http.StatusSeeOther, "/owner")
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

			// Add CSRF token to view data
			if csrfTokenInterface, exists := c.Get("csrf_token"); exists {
				if csrfToken, ok := csrfTokenInterface.(string); ok {
					viewData.Auth = viewData.Auth.WithCSRFToken(csrfToken)
				}
			}

			// Render the edit form
			c.Status(http.StatusOK)
			gunView.Edit(viewData).Render(context.Background(), c.Writer)
			return
		}

		// Create updated gun object
		updatedGun := &models.Gun{
			Name:           name,
			SerialNumber:   serialNumber,
			Purpose:        purpose,
			Acquired:       acquiredDate,
			WeaponTypeID:   uint(weaponTypeID),
			CaliberID:      uint(caliberID),
			ManufacturerID: uint(manufacturerID),
			OwnerID:        user.ID,
			Paid:           paidAmount,
		}
		updatedGun.ID = gun.ID

		// Use the validation-enabled update function
		err = models.UpdateGunWithValidation(db, updatedGun)
		if err != nil {
			// Map validation errors to form errors
			switch err {
			case models.ErrGunNameTooLong:
				formErrors["name"] = "Name cannot exceed 100 characters"
			case models.ErrNegativePrice:
				formErrors["paid"] = "Price cannot be negative"
			case models.ErrFutureDate:
				formErrors["acquired_date"] = "Acquisition date cannot be in the future"
			case models.ErrInvalidWeaponType:
				formErrors["weapon_type_id"] = "Selected weapon type doesn't exist"
			case models.ErrInvalidCaliber:
				formErrors["caliber_id"] = "Selected caliber doesn't exist"
			case models.ErrInvalidManufacturer:
				formErrors["manufacturer_id"] = "Selected manufacturer doesn't exist"
			default:
				// Set generic flash message
				session := sessions.Default(c)
				session.AddFlash("Failed to update gun: " + err.Error())
				session.Save()
				c.Redirect(http.StatusSeeOther, fmt.Sprintf("/owner/guns/%s/edit", gunID))
				return
			}

			// If we got specific validation errors, show them in the form
			if len(formErrors) > 0 {
				// Get reference data for the form
				var weaponTypes []models.WeaponType
				if err := db.Order("popularity DESC").Find(&weaponTypes).Error; err != nil {
					session := sessions.Default(c)
					session.AddFlash("Failed to get weapon types")
					session.Save()
					c.Redirect(http.StatusSeeOther, "/owner")
					return
				}

				var calibers []models.Caliber
				if err := db.Order("popularity DESC").Find(&calibers).Error; err != nil {
					session := sessions.Default(c)
					session.AddFlash("Failed to get calibers")
					session.Save()
					c.Redirect(http.StatusSeeOther, "/owner")
					return
				}

				var manufacturers []models.Manufacturer
				if err := db.Order("popularity DESC").Find(&manufacturers).Error; err != nil {
					session := sessions.Default(c)
					session.AddFlash("Failed to get manufacturers")
					session.Save()
					c.Redirect(http.StatusSeeOther, "/owner")
					return
				}

				// Create the data for the view with validation errors
				viewData := data.NewOwnerData().
					WithTitle("Edit Firearm").
					WithAuthenticated(true).
					WithUser(user).
					WithGun(&gun).
					WithWeaponTypes(weaponTypes).
					WithCalibers(calibers).
					WithManufacturers(manufacturers).
					WithFormErrors(formErrors)

				// Add CSRF token to view data
				if csrfTokenInterface, exists := c.Get("csrf_token"); exists {
					if csrfToken, ok := csrfTokenInterface.(string); ok {
						viewData.Auth = viewData.Auth.WithCSRFToken(csrfToken)
					}
				}

				// Render the edit form with validation errors
				c.Status(http.StatusOK)
				gunView.Edit(viewData).Render(context.Background(), c.Writer)
				return
			}
		}

		// Set flash message for success
		session := sessions.Default(c)
		session.AddFlash("Your gun has been updated successfully")
		session.Save()

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
		// Use session flash message instead of HTML error
		session := sessions.Default(c)
		session.AddFlash("Failed to get user information")
		session.Save()
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get the gun ID from the URL
	gunID := c.Param("id")

	// Get the gun from the database
	var gun models.Gun
	db := o.db.GetDB()
	if err := db.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").Where("id = ? AND owner_id = ?", gunID, user.ID).First(&gun).Error; err != nil {
		// Use session flash message instead of HTML error
		session := sessions.Default(c)
		session.AddFlash("That's not your gun!")
		session.Save()

		c.Redirect(http.StatusSeeOther, "/owner")
		return
	}

	// Parse the form
	if err := c.Request.ParseForm(); err != nil {
		// Use session flash message instead of HTML error
		session := sessions.Default(c)
		session.AddFlash("Invalid form data")
		session.Save()
		c.Redirect(http.StatusSeeOther, "/owner")
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
	purpose := c.PostForm("purpose")

	// Initialize form errors map
	formErrors := make(map[string]string)

	// Parse IDs with basic format validation
	weaponTypeID, err := strconv.ParseUint(weaponTypeIDStr, 10, 64)
	if err != nil {
		formErrors["weapon_type_id"] = "Invalid weapon type format"
	}

	caliberID, err := strconv.ParseUint(caliberIDStr, 10, 64)
	if err != nil {
		formErrors["caliber_id"] = "Invalid caliber format"
	}

	manufacturerID, err := strconv.ParseUint(manufacturerIDStr, 10, 64)
	if err != nil {
		formErrors["manufacturer_id"] = "Invalid manufacturer format"
	}

	// Parse acquired date if provided
	var acquiredDate *time.Time
	if acquiredDateStr != "" {
		parsedDate, err := time.Parse("2006-01-02", acquiredDateStr)
		if err != nil {
			formErrors["acquired_date"] = "Invalid date format, use MM-DD-YYYY"
		} else {
			// Check if date is in the future
			if parsedDate.After(time.Now()) {
				formErrors["acquired_date"] = "Acquisition date cannot be in the future"
			} else {
				acquiredDate = &parsedDate
			}
		}
	}

	// Parse paid amount if provided
	var paidAmount *float64
	if paidStr != "" {
		paid, err := strconv.ParseFloat(paidStr, 64)
		if err != nil {
			formErrors["paid"] = "Invalid amount format"
		} else {
			paidAmount = &paid
		}
	}

	// If there are basic parsing errors, return to the form
	if len(formErrors) > 0 {
		// Get all weapon types, calibers, and manufacturers
		var weaponTypes []models.WeaponType
		if err := db.Order("popularity DESC").Find(&weaponTypes).Error; err != nil {
			session := sessions.Default(c)
			session.AddFlash("Failed to get weapon types")
			session.Save()
			c.Redirect(http.StatusSeeOther, "/owner")
			return
		}

		var calibers []models.Caliber
		if err := db.Order("popularity DESC").Find(&calibers).Error; err != nil {
			session := sessions.Default(c)
			session.AddFlash("Failed to get calibers")
			session.Save()
			c.Redirect(http.StatusSeeOther, "/owner")
			return
		}

		var manufacturers []models.Manufacturer
		if err := db.Order("popularity DESC").Find(&manufacturers).Error; err != nil {
			session := sessions.Default(c)
			session.AddFlash("Failed to get manufacturers")
			session.Save()
			c.Redirect(http.StatusSeeOther, "/owner")
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

		// Add CSRF token to view data
		if csrfTokenInterface, exists := c.Get("csrf_token"); exists {
			if csrfToken, ok := csrfTokenInterface.(string); ok {
				viewData.Auth = viewData.Auth.WithCSRFToken(csrfToken)
			}
		}

		// Render the edit form
		c.Status(http.StatusOK)
		gunView.Edit(viewData).Render(context.Background(), c.Writer)
		return
	}

	// Create updated gun object
	updatedGun := &models.Gun{
		Name:           name,
		SerialNumber:   serialNumber,
		Purpose:        purpose,
		Acquired:       acquiredDate,
		WeaponTypeID:   uint(weaponTypeID),
		CaliberID:      uint(caliberID),
		ManufacturerID: uint(manufacturerID),
		OwnerID:        user.ID,
		Paid:           paidAmount,
	}
	updatedGun.ID = gun.ID

	// Use the validation-enabled update function
	err = models.UpdateGunWithValidation(db, updatedGun)
	if err != nil {
		// Map validation errors to form errors
		switch err {
		case models.ErrGunNameTooLong:
			formErrors["name"] = "Name cannot exceed 100 characters"
		case models.ErrNegativePrice:
			formErrors["paid"] = "Price cannot be negative"
		case models.ErrFutureDate:
			formErrors["acquired_date"] = "Acquisition date cannot be in the future"
		case models.ErrInvalidWeaponType:
			formErrors["weapon_type_id"] = "Selected weapon type doesn't exist"
		case models.ErrInvalidCaliber:
			formErrors["caliber_id"] = "Selected caliber doesn't exist"
		case models.ErrInvalidManufacturer:
			formErrors["manufacturer_id"] = "Selected manufacturer doesn't exist"
		default:
			// Set generic flash message
			session := sessions.Default(c)
			session.AddFlash("Failed to update gun: " + err.Error())
			session.Save()
			c.Redirect(http.StatusSeeOther, fmt.Sprintf("/owner/guns/%s/edit", gunID))
			return
		}

		// If we got specific validation errors, show them in the form
		if len(formErrors) > 0 {
			// Get reference data for the form
			var weaponTypes []models.WeaponType
			if err := db.Order("popularity DESC").Find(&weaponTypes).Error; err != nil {
				session := sessions.Default(c)
				session.AddFlash("Failed to get weapon types")
				session.Save()
				c.Redirect(http.StatusSeeOther, "/owner")
				return
			}

			var calibers []models.Caliber
			if err := db.Order("popularity DESC").Find(&calibers).Error; err != nil {
				session := sessions.Default(c)
				session.AddFlash("Failed to get calibers")
				session.Save()
				c.Redirect(http.StatusSeeOther, "/owner")
				return
			}

			var manufacturers []models.Manufacturer
			if err := db.Order("popularity DESC").Find(&manufacturers).Error; err != nil {
				session := sessions.Default(c)
				session.AddFlash("Failed to get manufacturers")
				session.Save()
				c.Redirect(http.StatusSeeOther, "/owner")
				return
			}

			// Create the data for the view with validation errors
			viewData := data.NewOwnerData().
				WithTitle("Edit Firearm").
				WithAuthenticated(true).
				WithUser(user).
				WithGun(&gun).
				WithWeaponTypes(weaponTypes).
				WithCalibers(calibers).
				WithManufacturers(manufacturers).
				WithFormErrors(formErrors)

			// Add CSRF token to view data
			if csrfTokenInterface, exists := c.Get("csrf_token"); exists {
				if csrfToken, ok := csrfTokenInterface.(string); ok {
					viewData.Auth = viewData.Auth.WithCSRFToken(csrfToken)
				}
			}

			// Render the edit form with validation errors
			c.Status(http.StatusOK)
			gunView.Edit(viewData).Render(context.Background(), c.Writer)
			return
		}
	}

	// Set flash message for success
	session := sessions.Default(c)
	session.AddFlash("Your gun has been updated successfully")
	session.Save()

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

		// Get the gun ID from the URL
		gunID := c.Param("id")

		// Delete the gun
		db := o.db.GetDB()
		gunIDUint, err := strconv.ParseUint(gunID, 10, 64)
		if err != nil {
			// Set flash message
			if setFlash, exists := c.Get("setFlash"); exists {
				setFlash.(func(string))("Invalid gun ID")
			}
			c.Redirect(http.StatusSeeOther, "/owner")
			return
		}
		err = o.db.DeleteGun(db, uint(gunIDUint), dbUser.ID)
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

	// Get the gun ID from the URL
	gunID := c.Param("id")

	// Delete the gun
	db := o.db.GetDB()
	gunIDUint, err := strconv.ParseUint(gunID, 10, 64)
	if err != nil {
		// Set flash message
		if setFlash, exists := c.Get("setFlash"); exists {
			setFlash.(func(string))("Invalid gun ID")
		}
		c.Redirect(http.StatusSeeOther, "/owner")
		return
	}
	err = o.db.DeleteGun(db, uint(gunIDUint), dbUser.ID)
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

				// Re-fetch roles from Casbin to ensure they're up to date
				if casbinAuth, exists := c.Get("casbinAuth"); exists && casbinAuth != nil {
					if ca, ok := casbinAuth.(interface{ GetUserRoles(string) []string }); ok {
						roles := ca.GetUserRoles(userInfo.GetUserName())
						logger.Info("Casbin roles for user in arsenal page", map[string]interface{}{
							"email": userInfo.GetUserName(),
							"roles": roles,
						})
						ownerData.Auth = ownerData.Auth.WithRoles(roles)
					}
				}
			}
		}

		// If the user is on free tier, add a message
		if showFreeLimit {
			ownerData.WithError(fmt.Sprintf("Free Tier shows all %d guns in your collection, but only allows you to add up to 2 guns. Please subscribe to add more firearms.", totalUserGuns))
		}

		// Check for flash messages from session
		session := sessions.Default(c)
		flashes := session.Flashes()
		if len(flashes) > 0 {

			session.Save()
			for _, flash := range flashes {
				if flashMsg, ok := flash.(string); ok {
					ownerData.WithSuccess(flashMsg)
				}
			}
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

			// Re-fetch roles from Casbin to ensure they're up to date
			if casbinAuth, exists := c.Get("casbinAuth"); exists && casbinAuth != nil {
				if ca, ok := casbinAuth.(interface{ GetUserRoles(string) []string }); ok {
					roles := ca.GetUserRoles(userInfo.GetUserName())
					logger.Info("Casbin roles for user in arsenal page", map[string]interface{}{
						"email": userInfo.GetUserName(),
						"roles": roles,
					})

					ownerData.Auth = ownerData.Auth.WithRoles(roles)
				}
			}
		}
	}

	// If the user is on free tier, add a message
	if showFreeLimit {
		ownerData.WithError(fmt.Sprintf("Free Tier shows all %d guns in your collection, but only allows you to add up to 2 guns. Please subscribe to add more firearms.", totalUserGuns))
	}

	// Check for flash messages from session
	session := sessions.Default(c)
	flashes := session.Flashes()
	if len(flashes) > 0 {

		session.Save()
		for _, flash := range flashes {
			if flashMsg, ok := flash.(string); ok {
				ownerData.WithSuccess(flashMsg)
			}
		}
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

				// Re-fetch roles from Casbin to ensure they're up to date
				if casbinAuth, exists := c.Get("casbinAuth"); exists && casbinAuth != nil {
					if ca, ok := casbinAuth.(interface{ GetUserRoles(string) []string }); ok {
						roles := ca.GetUserRoles(userInfo.GetUserName())
						logger.Info("Casbin roles for user in profile page", map[string]interface{}{
							"email": userInfo.GetUserName(),
							"roles": roles,
						})
						ownerData.Auth = ownerData.Auth.WithRoles(roles)
					}
				}
			}
		}

		// Check for flash messages from session
		session := sessions.Default(c)
		flashes := session.Flashes()
		if len(flashes) > 0 {

			session.Save()
			for _, flash := range flashes {
				if flashMsg, ok := flash.(string); ok {
					ownerData.WithSuccess(flashMsg)
				}
			}
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

			// Re-fetch roles from Casbin to ensure they're up to date
			if casbinAuth, exists := c.Get("casbinAuth"); exists && casbinAuth != nil {
				if ca, ok := casbinAuth.(interface{ GetUserRoles(string) []string }); ok {
					roles := ca.GetUserRoles(userInfo.GetUserName())
					logger.Info("Casbin roles for user in profile page", map[string]interface{}{
						"email": userInfo.GetUserName(),
						"roles": roles,
					})
					ownerData.Auth = ownerData.Auth.WithRoles(roles)
				}
			}
		}
	}

	// Check for flash messages from session
	session := sessions.Default(c)
	flashes := session.Flashes()
	if len(flashes) > 0 {

		session.Save()
		for _, flash := range flashes {
			if flashMsg, ok := flash.(string); ok {
				ownerData.WithSuccess(flashMsg)
			}
		}
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

		// Check for flash messages from session
		session := sessions.Default(c)
		flashes := session.Flashes()
		if len(flashes) > 0 {

			session.Save()
			for _, flash := range flashes {
				if flashMsg, ok := flash.(string); ok {
					ownerData.WithSuccess(flashMsg)
				}
			}
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

	// Check for flash messages from session
	session := sessions.Default(c)
	flashes := session.Flashes()
	if len(flashes) > 0 {

		session.Save()
		for _, flash := range flashes {
			if flashMsg, ok := flash.(string); ok {
				ownerData.WithSuccess(flashMsg)
			}
		}
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
			dbUser.VerificationTokenExpiry = time.Now().Add(1 * time.Hour)
			dbUser.VerificationSentAt = time.Now()

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
				// Fallback to session directly if context function not available
				session := sessions.Default(c)
				session.AddFlash("Your profile has been updated.")
				session.Save()
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
		dbUser.VerificationTokenExpiry = time.Now().Add(1 * time.Hour)
		dbUser.VerificationSentAt = time.Now()

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
			// Fallback to session directly if context function not available
			session := sessions.Default(c)
			session.AddFlash("Your profile has been updated.")
			session.Save()
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

				// Re-fetch roles from Casbin to ensure they're up to date
				if casbinAuth, exists := c.Get("casbinAuth"); exists && casbinAuth != nil {
					if ca, ok := casbinAuth.(interface{ GetUserRoles(string) []string }); ok {
						roles := ca.GetUserRoles(userInfo.GetUserName())
						logger.Info("Casbin roles for user in subscription page", map[string]interface{}{
							"email": userInfo.GetUserName(),
							"roles": roles,
						})
						ownerData.Auth = ownerData.Auth.WithRoles(roles)
					}
				}
			}
		}

		// Check for flash messages from session
		session := sessions.Default(c)
		flashes := session.Flashes()
		if len(flashes) > 0 {

			session.Save()
			for _, flash := range flashes {
				if flashMsg, ok := flash.(string); ok {
					ownerData.WithSuccess(flashMsg)
				}
			}
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

			// Re-fetch roles from Casbin to ensure they're up to date
			if casbinAuth, exists := c.Get("casbinAuth"); exists && casbinAuth != nil {
				if ca, ok := casbinAuth.(interface{ GetUserRoles(string) []string }); ok {
					roles := ca.GetUserRoles(userInfo.GetUserName())
					logger.Info("Casbin roles for user in subscription page", map[string]interface{}{
						"email": userInfo.GetUserName(),
						"roles": roles,
					})
					ownerData.Auth = ownerData.Auth.WithRoles(roles)
				}
			}
		}
	}

	// Check for flash messages from session
	session := sessions.Default(c)
	flashes := session.Flashes()
	if len(flashes) > 0 {

		session.Save()
		for _, flash := range flashes {
			if flashMsg, ok := flash.(string); ok {
				ownerData.WithSuccess(flashMsg)
			}
		}
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

		// Get CSRF token from context and set it in auth data
		if csrfTokenInterface, exists := c.Get("csrf_token"); exists {
			if csrfToken, ok := csrfTokenInterface.(string); ok {
				ownerData.Auth = ownerData.Auth.WithCSRFToken(csrfToken)
			}
		}

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

	// Get CSRF token from context and set it in auth data
	if csrfTokenInterface, exists := c.Get("csrf_token"); exists {
		if csrfToken, ok := csrfTokenInterface.(string); ok {
			ownerData.Auth = ownerData.Auth.WithCSRFToken(csrfToken)
		}
	}

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

		// Clear the session
		session := sessions.Default(c)
		session.Clear()
		session.AddFlash("Your account has been deleted. Please come back any time!")
		if err := session.Save(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
			return
		}

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

	// Clear the session
	session := sessions.Default(c)
	session.Clear()
	session.AddFlash("Your account has been deleted. Please come back any time!")
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		return
	}

	// Redirect to home page
	c.Redirect(http.StatusSeeOther, "/")
}
