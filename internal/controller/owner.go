package controller

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"log"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/cmd/web/views/owner"
	gunView "github.com/hail2skins/armory/cmd/web/views/owner/gun"
	"github.com/hail2skins/armory/cmd/web/views/owner/munitions"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/logger"
	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/services/email"
	"github.com/shaj13/go-guardian/v2/auth"
	"gorm.io/gorm"
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

	// Check if the user has a promotional subscription that has expired
	// This will automatically update the subscription status if needed
	if updated, err := o.db.CheckExpiredPromotionSubscription(dbUser); err != nil {
		logger.Error("Failed to check expired promotion subscription", err, map[string]interface{}{
			"user_id": dbUser.ID,
			"email":   dbUser.Email,
		})
	} else if updated {
		// Log that we updated the user's subscription status
		logger.Info("Updated expired promotion subscription", map[string]interface{}{
			"user_id":               dbUser.ID,
			"email":                 dbUser.Email,
			"subscription_status":   dbUser.SubscriptionStatus,
			"subscription_end_date": dbUser.SubscriptionEndDate,
		})
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

		// Limit the guns for free tier users to only 2 guns
		if len(guns) > 2 {
			guns = guns[:2]
		}
	}

	// Get the ammunition count for this user
	ammoCount, err := o.db.CountAmmoByUser(dbUser.ID)
	if err != nil {
		logger.Error("Failed to count user's ammunition", err, map[string]interface{}{
			"user_id": dbUser.ID,
			"email":   dbUser.Email,
		})
		ammoCount = 0
	}

	// Get the total ammunition quantity for this user
	totalAmmoQuantity, err := o.db.SumAmmoQuantityByUser(dbUser.ID)
	if err != nil {
		logger.Error("Failed to sum user's ammunition quantity", err, map[string]interface{}{
			"user_id": dbUser.ID,
			"email":   dbUser.Email,
		})
		totalAmmoQuantity = 0
	}

	// Get the total expended ammunition for this user
	totalAmmoExpended, err := o.db.SumAmmoExpendedByUser(dbUser.ID)
	if err != nil {
		logger.Error("Failed to sum user's expended ammunition", err, map[string]interface{}{
			"user_id": dbUser.ID,
			"email":   dbUser.Email,
		})
		totalAmmoExpended = 0
	}

	// Get the user's ammunition
	ammoItems, err := models.FindAmmoByOwner(o.db.GetDB(), dbUser.ID)
	if err != nil {
		logger.Error("Failed to fetch user's ammunition", err, map[string]interface{}{
			"user_id": dbUser.ID,
			"email":   dbUser.Email,
		})
		ammoItems = []models.Ammo{}
	}

	// Calculate total paid for ammunition
	var totalAmmoPaid float64
	for _, ammo := range ammoItems {
		if ammo.Paid != nil {
			totalAmmoPaid += *ammo.Paid
		}
	}

	// Format subscription end date if available
	var subscriptionEndsAt string
	if !dbUser.SubscriptionEndDate.IsZero() {
		subscriptionEndsAt = dbUser.SubscriptionEndDate.Format("January 2, 2006")
	}

	// Calculate total paid for guns
	var totalPaid float64
	for _, gun := range guns {
		if gun.Paid != nil {
			totalPaid += *gun.Paid
		}
	}

	// Create owner data for the view
	ownerData := data.NewOwnerData().
		WithTitle("Owner Dashboard").
		WithAuthenticated(authenticated).
		WithUser(dbUser).
		WithGuns(guns).
		WithAmmo(ammoItems).
		WithSubscriptionInfo(dbUser.HasActiveSubscription(), dbUser.SubscriptionTier, subscriptionEndsAt).
		WithPagination(page, totalPages, perPage, int(totalGuns)).
		WithSorting(sortBy, sortOrder).
		WithSearchTerm(searchTerm).
		WithFiltersApplied(sortBy, sortOrder, perPage, searchTerm).
		WithTotalPaid(totalPaid).
		WithAmmoCount(ammoCount).
		WithTotalAmmoQuantity(totalAmmoQuantity).
		WithTotalAmmoPaid(totalAmmoPaid).
		WithTotalAmmoExpended(totalAmmoExpended)

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
		ownerData.WithError(fmt.Sprintf("Free tier only allows 2 guns. You have %d in your arsenal. Subscribe to see more.", totalUserGuns))
		// Add a note that will display below the table
		ownerData.WithNote("To see your remaining firearms please subscribe.")
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
	finish := c.PostForm("finish")

	logger.Info("Form data received", map[string]interface{}{
		"name":            name,
		"serial_number":   serialNumber,
		"acquired":        acquiredDateStr,
		"weapon_type_id":  weaponTypeIDStr,
		"caliber_id":      caliberIDStr,
		"manufacturer_id": manufacturerIDStr,
		"paid":            paidStr,
		"purpose":         purpose,
		"finish":          finish,
	})

	// Get current user - Using c.Get instead of c.MustGet for safety
	authControllerInterface, exists := c.Get("authController")
	if !exists {
		logger.Error("Auth controller not found", nil, nil)
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Try to cast to the interface type first
	authController, ok := authControllerInterface.(AuthControllerInterface)
	if !ok {
		// Try concrete type as fallback
		concreteAuthController, ok := authControllerInterface.(*AuthController)
		if !ok {
			logger.Error("Invalid auth controller type", nil, nil)
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}
		authController = concreteAuthController
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

		// Set HTTP status code to 422 Unprocessable Entity
		c.Status(http.StatusUnprocessableEntity)

		// Render the form with errors
		gunView.New(ownerData).Render(c.Request.Context(), c.Writer)
		return
	}

	// Create a new gun
	newGun := &models.Gun{
		Name:           name,
		SerialNumber:   serialNumber,
		Purpose:        purpose,
		Finish:         finish,
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
		case models.ErrGunPurposeTooLong:
			errors["purpose"] = "Purpose cannot exceed 100 characters"
		case models.ErrGunFinishTooLong:
			errors["finish"] = "Finish cannot exceed 100 characters"
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

		// Set proper status code for validation errors
		c.Status(http.StatusUnprocessableEntity)

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
		finish := c.PostForm("finish")

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
			if err == nil {
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
			if err == nil && paid >= 0 {
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
			Finish:         finish,
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
			case models.ErrGunPurposeTooLong:
				formErrors["purpose"] = "Purpose cannot exceed 100 characters"
			case models.ErrGunFinishTooLong:
				formErrors["finish"] = "Finish cannot exceed 100 characters"
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
	finish := c.PostForm("finish")

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
		if err == nil {
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
		if err == nil && paid >= 0 {
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
		Finish:         finish,
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
		case models.ErrGunPurposeTooLong:
			formErrors["purpose"] = "Purpose cannot exceed 100 characters"
		case models.ErrGunFinishTooLong:
			formErrors["finish"] = "Finish cannot exceed 100 characters"
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
// Deprecated: No longer used with Choices.js implementation for munitions form, still used by API
func (o *OwnerController) SearchCalibers(c *gin.Context) {
	query := c.Query("q")
	db := o.db.GetDB()

	var calibers []models.Caliber
	if query != "" {
		db.Where("LOWER(caliber) LIKE ? OR LOWER(nickname) LIKE ?",
			"%"+strings.ToLower(query)+"%",
			"%"+strings.ToLower(query)+"%").
			Order("popularity DESC, caliber ASC").
			Find(&calibers)
	} else {
		db.Order("popularity DESC, caliber ASC").Find(&calibers)
	}

	html := ""
	for _, caliber := range calibers {
		displayText := caliber.Caliber
		if caliber.Nickname != "" {
			displayText += fmt.Sprintf(" (%s)", caliber.Nickname)
		}
		html += fmt.Sprintf(`<div class="custom-dropdown-item" data-id="%d">%s</div>`, caliber.ID, displayText)
	}

	c.Data(200, "text/html", []byte(html))
}

// SearchBrands handles the brand search for HTMX dropdown
// Deprecated: No longer used with Choices.js implementation
func (o *OwnerController) SearchBrands(c *gin.Context) {
	query := c.Query("q")
	logger.Info("Brand search request received", map[string]interface{}{
		"query": query,
	})

	db := o.db.GetDB()

	var brands []models.Brand
	if query != "" {
		logger.Info("Searching brands with query", map[string]interface{}{
			"query": query,
		})
		db.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(query)+"%").Order("popularity DESC, name ASC").Find(&brands)
	} else {
		logger.Info("Loading all brands", nil)
		db.Order("popularity DESC, name ASC").Find(&brands)
	}

	html := ""
	for _, brand := range brands {
		html += fmt.Sprintf(`<div class="custom-dropdown-item" data-id="%d">%s</div>`, brand.ID, brand.Name)
	}

	logger.Info("Brand search response", map[string]interface{}{
		"count": len(brands),
	})

	c.Data(200, "text/html", []byte(html))
}

// SearchBulletStyles handles the bullet style search for HTMX dropdown
// Deprecated: No longer used with Choices.js implementation
func (o *OwnerController) SearchBulletStyles(c *gin.Context) {
	query := c.Query("q")
	db := o.db.GetDB()

	var styles []models.BulletStyle
	if query != "" {
		db.Where("LOWER(type) LIKE ?", "%"+strings.ToLower(query)+"%").Order("popularity DESC, type ASC").Find(&styles)
	} else {
		db.Order("popularity DESC, type ASC").Find(&styles)
	}

	html := ""
	for _, style := range styles {
		html += fmt.Sprintf(`<div class="custom-dropdown-item" data-id="%d">%s</div>`, style.ID, style.Type)
	}

	c.Data(200, "text/html", []byte(html))
}

// SearchGrains handles the grain search for HTMX dropdown
// Deprecated: No longer used with Choices.js implementation
func (o *OwnerController) SearchGrains(c *gin.Context) {
	query := c.Query("q")
	db := o.db.GetDB()

	var grains []models.Grain
	if query != "" {
		db.Where("weight LIKE ?", "%"+query+"%").Order("popularity DESC, weight ASC").Find(&grains)
	} else {
		db.Order("popularity DESC, weight ASC").Find(&grains)
	}

	html := ""
	for _, grain := range grains {
		displayText := ""
		if grain.Weight == 0 {
			displayText = "Other"
		} else {
			displayText = fmt.Sprintf("%d gr", grain.Weight)
		}
		html += fmt.Sprintf(`<div class="custom-dropdown-item" data-id="%d">%s</div>`, grain.ID, displayText)
	}

	c.Data(200, "text/html", []byte(html))
}

// SearchCasings handles the casing search for HTMX dropdown
// Deprecated: No longer used with Choices.js implementation
func (o *OwnerController) SearchCasings(c *gin.Context) {
	query := c.Query("q")
	db := o.db.GetDB()

	var casings []models.Casing
	if query != "" {
		db.Where("LOWER(type) LIKE ?", "%"+strings.ToLower(query)+"%").Order("popularity DESC, type ASC").Find(&casings)
	} else {
		db.Order("popularity DESC, type ASC").Find(&casings)
	}

	html := ""
	for _, casing := range casings {
		html += fmt.Sprintf(`<div class="custom-dropdown-item" data-id="%d">%s</div>`, casing.ID, casing.Type)
	}

	c.Data(200, "text/html", []byte(html))
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

			// Limit the guns for free tier users to only 2 guns
			if len(guns) > 2 {
				guns = guns[:2]
			}
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
			ownerData.WithError(fmt.Sprintf("Free tier only allows 2 guns. You have %d in your arsenal. Subscribe to see more.", totalUserGuns))
			// Add a note that will display below the table
			ownerData.WithNote("To see your remaining firearms please subscribe.")
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

		// Limit the guns for free tier users to only 2 guns
		if len(guns) > 2 {
			guns = guns[:2]
		}
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
		ownerData.WithError(fmt.Sprintf("Free tier only allows 2 guns. You have %d in your arsenal. Subscribe to see more.", totalUserGuns))
		// Add a note that will display below the table
		ownerData.WithNote("To see your remaining firearms please subscribe.")
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

// AmmoNew handles the new ammunition route
func (o *OwnerController) AmmoNew(c *gin.Context) {
	// Get the current user's authentication status and email
	authController, exists := c.Get("authController")
	if !exists {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get current user information
	authInterface := authController.(AuthControllerInterface)
	userInfo, authenticated := authInterface.GetCurrentUser(c)
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

	// Get the database connection
	db := o.db.GetDB()

	// Fetch brands ordered by popularity
	var brands []models.Brand
	if err := db.Order("popularity DESC, name ASC").Find(&brands).Error; err != nil {
		logger.Error("Failed to fetch brands", err, nil)
		brands = []models.Brand{}
	}

	// Fetch calibers ordered by popularity
	var calibers []models.Caliber
	if err := db.Order("popularity DESC, caliber ASC").Find(&calibers).Error; err != nil {
		logger.Error("Failed to fetch calibers", err, nil)
		calibers = []models.Caliber{}
	}

	// Fetch bullet styles ordered by popularity
	var bulletStyles []models.BulletStyle
	if err := db.Order("popularity DESC, type ASC").Find(&bulletStyles).Error; err != nil {
		logger.Error("Failed to fetch bullet styles", err, nil)
		bulletStyles = []models.BulletStyle{}
	}

	// Fetch grains ordered by popularity
	var grains []models.Grain
	if err := db.Order("popularity DESC, weight ASC").Find(&grains).Error; err != nil {
		logger.Error("Failed to fetch grains", err, nil)
		grains = []models.Grain{}
	}

	// Fetch casings ordered by popularity
	var casings []models.Casing
	if err := db.Order("popularity DESC, type ASC").Find(&casings).Error; err != nil {
		logger.Error("Failed to fetch casings", err, nil)
		casings = []models.Casing{}
	}

	// Create owner data for the view
	ownerData := data.NewOwnerData().
		WithTitle("New Ammunition").
		WithAuthenticated(true).
		WithUser(dbUser)

	// Add the data for dropdowns
	ownerData.Brands = brands
	ownerData.Calibers = calibers
	ownerData.BulletStyles = bulletStyles
	ownerData.Grains = grains
	ownerData.Casings = casings

	// Set authentication data
	if csrfToken, exists := c.Get("csrf_token"); exists {
		if token, ok := csrfToken.(string); ok {
			ownerData.Auth.CSRFToken = token
		}
	}

	// Get authData from context to preserve roles
	if authDataInterface, exists := c.Get("authData"); exists {
		if authData, ok := authDataInterface.(data.AuthData); ok {
			// Use the auth data that already has roles, maintaining our title and other changes
			ownerData.Auth = authData.WithTitle("New Ammunition")

			// Re-fetch roles from Casbin to ensure they're up to date
			if casbinAuth, exists := c.Get("casbinAuth"); exists && casbinAuth != nil {
				if ca, ok := casbinAuth.(interface{ GetUserRoles(string) []string }); ok {
					roles := ca.GetUserRoles(userInfo.GetUserName())
					logger.Info("Casbin roles for user in ammunition page", map[string]interface{}{
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

	// Render the ammo new view
	munitions.New(ownerData).Render(c.Request.Context(), c.Writer)
}

// AmmoCreate handles the creation of new ammunition
func (o *OwnerController) AmmoCreate(c *gin.Context) {
	// Get the current user's authentication status and email
	authController, exists := c.Get("authController")
	if !exists {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get current user information
	authInterface := authController.(AuthControllerInterface)
	userInfo, authenticated := authInterface.GetCurrentUser(c)
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

	// Check if user is on free tier and already has 4 ammunition items
	if dbUser.SubscriptionTier == "free" {
		var count int64
		db := o.db.GetDB()
		db.Model(&models.Ammo{}).Where("owner_id = ?", dbUser.ID).Count(&count)

		if count >= 4 {
			if setFlash, exists := c.Get("setFlash"); exists {
				setFlash.(func(string))("You must be subscribed to add more to your munitions depot")
			}
			c.Redirect(http.StatusSeeOther, "/pricing")
			return
		}
	}

	// Parse form values
	err = c.Request.ParseForm()
	if err != nil {
		// Handle error
		handleAmmoCreateError(c, dbUser, "Failed to parse form", nil, http.StatusUnprocessableEntity, o.db.GetDB())
		return
	}

	// Extract ammunition data from form
	name := c.Request.PostForm.Get("name")
	countStr := c.Request.PostForm.Get("count")
	brandIDStr := c.Request.PostForm.Get("brand_id")
	bulletStyleIDStr := c.Request.PostForm.Get("bullet_style_id")
	grainIDStr := c.Request.PostForm.Get("grain_id")
	caliberIDStr := c.Request.PostForm.Get("caliber_id")
	casingIDStr := c.Request.PostForm.Get("casing_id")
	acquiredDateStr := c.Request.PostForm.Get("acquired_date")
	paidStr := c.Request.PostForm.Get("paid")
	expendedStr := c.Request.PostForm.Get("expended")

	// Prepare form errors map
	formErrors := make(map[string]string)

	// Validate name (required and max 100 chars)
	if name == "" {
		formErrors["name"] = "Name is required"
	} else if len(name) > 100 {
		formErrors["name"] = "Name is too long (maximum 100 characters)"
	}

	// Parse count (required)
	var count int
	if countStr == "" {
		formErrors["count"] = "Count is required"
	} else {
		count, err = strconv.Atoi(countStr)
		if err != nil || count < 0 {
			formErrors["count"] = "Count must be a number"
		}
	}

	// Parse brand ID (required)
	var brandID uint64
	if brandIDStr == "" {
		formErrors["brand_id"] = "Brand is required"
	} else {
		brandID, err = strconv.ParseUint(brandIDStr, 10, 64)
		if err != nil {
			formErrors["brand_id"] = "Invalid brand"
		}
	}

	// Parse caliber ID (required)
	var caliberID uint64
	if caliberIDStr == "" {
		formErrors["caliber_id"] = "Caliber is required"
	} else {
		caliberID, err = strconv.ParseUint(caliberIDStr, 10, 64)
		if err != nil {
			formErrors["caliber_id"] = "Invalid caliber"
		}
	}

	// If there are validation errors, respond with them
	if len(formErrors) > 0 {
		// Format and return all errors
		handleAmmoCreateError(c, dbUser, "Please fix the errors below", formErrors, http.StatusUnprocessableEntity, o.db.GetDB())
		return
	}

	// Create new ammunition
	ammo := &models.Ammo{
		Name:      name,
		BrandID:   uint(brandID),
		CaliberID: uint(caliberID),
		OwnerID:   dbUser.ID,
		Count:     count,
	}

	db := o.db.GetDB()

	// Parse bullet style ID (optional)
	if bulletStyleIDStr != "" {
		bulletStyleID, err := strconv.ParseUint(bulletStyleIDStr, 10, 64)
		if err == nil {
			ammo.BulletStyleID = uint(bulletStyleID)
		}
	}

	// Parse grain ID (optional)
	if grainIDStr != "" {
		grainID, err := strconv.ParseUint(grainIDStr, 10, 64)
		if err == nil {
			ammo.GrainID = uint(grainID)
		}
	}

	// Parse casing ID (optional)
	if casingIDStr != "" {
		casingID, err := strconv.ParseUint(casingIDStr, 10, 64)
		if err == nil {
			ammo.CasingID = uint(casingID)
		}
	}

	// Parse acquisition date (optional)
	if acquiredDateStr != "" {
		acquiredDate, err := time.Parse("2006-01-02", acquiredDateStr)
		if err == nil {
			ammo.Acquired = &acquiredDate
		}
	}

	// Parse paid amount (optional)
	if paidStr != "" {
		paid, err := strconv.ParseFloat(paidStr, 64)
		if err == nil && paid >= 0 {
			ammo.Paid = &paid
		}
	}

	// Parse expended amount (optional, defaulting to 0)
	if expendedStr != "" {
		expended, err := strconv.Atoi(expendedStr)
		if err == nil && expended >= 0 {
			ammo.Expended = expended
		}
	}

	// Validate and create the ammunition using the DB service
	if err := models.CreateAmmoWithValidation(db, ammo); err != nil {
		// Create detailed error message based on the validation error
		handleAmmoCreateError(c, dbUser, "Failed to create ammunition: "+err.Error(), nil, http.StatusUnprocessableEntity, db)
		return
	}

	// Set success message
	session := sessions.Default(c)
	session.AddFlash("Ammunition added successfully")
	session.Save()

	// Redirect to ammunition index page
	c.Redirect(http.StatusSeeOther, "/owner/munitions")
}

// Helper function to handle ammunition creation errors
func handleAmmoCreateError(c *gin.Context, dbUser *database.User, errMsg string, formErrors map[string]string, statusCode int, db *gorm.DB) {
	// Get the database connection to fetch reference data
	// db := c.MustGet("db").(*gorm.DB)

	// Fetch brands ordered by popularity
	var brands []models.Brand
	if err := db.Order("popularity DESC, name ASC").Find(&brands).Error; err != nil {
		logger.Error("Failed to fetch brands", err, nil)
		brands = []models.Brand{}
	}

	// Fetch calibers ordered by popularity
	var calibers []models.Caliber
	if err := db.Order("popularity DESC, caliber ASC").Find(&calibers).Error; err != nil {
		logger.Error("Failed to fetch calibers", err, nil)
		calibers = []models.Caliber{}
	}

	// Fetch bullet styles ordered by popularity
	var bulletStyles []models.BulletStyle
	if err := db.Order("popularity DESC, type ASC").Find(&bulletStyles).Error; err != nil {
		logger.Error("Failed to fetch bullet styles", err, nil)
		bulletStyles = []models.BulletStyle{}
	}

	// Fetch grains ordered by popularity
	var grains []models.Grain
	if err := db.Order("popularity DESC, weight ASC").Find(&grains).Error; err != nil {
		logger.Error("Failed to fetch grains", err, nil)
		grains = []models.Grain{}
	}

	// Fetch casings ordered by popularity
	var casings []models.Casing
	if err := db.Order("popularity DESC, type ASC").Find(&casings).Error; err != nil {
		logger.Error("Failed to fetch casings", err, nil)
		casings = []models.Casing{}
	}

	// Create owner data for the view
	ownerData := data.NewOwnerData().
		WithTitle("New Ammunition").
		WithAuthenticated(true).
		WithUser(dbUser).
		WithError(errMsg)

	// Add the data for dropdowns
	ownerData.Brands = brands
	ownerData.Calibers = calibers
	ownerData.BulletStyles = bulletStyles
	ownerData.Grains = grains
	ownerData.Casings = casings

	// Initialize form errors if not provided
	if formErrors == nil {
		formErrors = make(map[string]string)
	}

	// Preserve user input data by storing in form errors with a special prefix
	// This allows the template to access the values using the same FormErrors map
	formErrors["value_name"] = c.Request.PostForm.Get("name")
	formErrors["value_count"] = c.Request.PostForm.Get("count")
	formErrors["value_brand_id"] = c.Request.PostForm.Get("brand_id")
	formErrors["value_bullet_style_id"] = c.Request.PostForm.Get("bullet_style_id")
	formErrors["value_grain_id"] = c.Request.PostForm.Get("grain_id")
	formErrors["value_caliber_id"] = c.Request.PostForm.Get("caliber_id")
	formErrors["value_casing_id"] = c.Request.PostForm.Get("casing_id")
	formErrors["value_acquired_date"] = c.Request.PostForm.Get("acquired_date")
	formErrors["value_paid"] = c.Request.PostForm.Get("paid")

	// Add form errors to the ownerData
	ownerData = ownerData.WithFormErrors(formErrors)

	// Set authentication data from context
	if csrfToken, exists := c.Get("csrf_token"); exists {
		if token, ok := csrfToken.(string); ok {
			ownerData.Auth.CSRFToken = token
		}
	}

	// Get authData from context to preserve roles
	if authDataInterface, exists := c.Get("authData"); exists {
		if authData, ok := authDataInterface.(data.AuthData); ok {
			// Use the auth data that already has roles, maintaining our title and other changes
			ownerData.Auth = authData.WithTitle("New Ammunition")
		}
	}

	// Set appropriate HTTP status code
	c.Status(statusCode)

	// Render the ammo new view with error
	munitions.New(ownerData).Render(c.Request.Context(), c.Writer)
}

// AmmoIndex displays all ammunition for the owner
func (o *OwnerController) AmmoIndex(c *gin.Context) {
	// Get the current user's authentication status and email
	authController, exists := c.Get("authController")
	if !exists {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get current user information
	authInterface := authController.(AuthControllerInterface)
	userInfo, authenticated := authInterface.GetCurrentUser(c)
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
		"name":       true,
		"created_at": true,
		"acquired":   true,
		"brand":      true,
		"caliber":    true,
		"count":      true,
	}
	if !validSortFields[sortBy] {
		sortBy = "created_at"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	// Get search term if available
	searchTerm := c.Query("search")

	// Get all ammunition count for this user
	ammoCount, err := o.db.CountAmmoByUser(dbUser.ID)
	if err != nil {
		logger.Error("Failed to count user's ammunition", err, map[string]interface{}{
			"user_id": dbUser.ID,
			"email":   dbUser.Email,
		})
		ammoCount = 0
	}

	// Get the total ammunition quantity for this user
	totalAmmoQuantity, err := o.db.SumAmmoQuantityByUser(dbUser.ID)
	if err != nil {
		logger.Error("Failed to sum user's ammunition quantity", err, map[string]interface{}{
			"user_id": dbUser.ID,
			"email":   dbUser.Email,
		})
		totalAmmoQuantity = 0
	}

	// Get the total expended ammunition for this user
	totalAmmoExpended, err := o.db.SumAmmoExpendedByUser(dbUser.ID)
	if err != nil {
		logger.Error("Failed to sum user's expended ammunition", err, map[string]interface{}{
			"user_id": dbUser.ID,
			"email":   dbUser.Email,
		})
		totalAmmoExpended = 0
	}

	// Get the user's ammunition with pagination and filtering
	db := o.db.GetDB()

	// Base query for counting
	countQuery := db.Model(&models.Ammo{}).Where("owner_id = ?", dbUser.ID)

	// Add search functionality if search term is provided
	if searchTerm != "" {
		countQuery = countQuery.Where("name LIKE ?", "%"+searchTerm+"%")
	}

	// Count total matching entries for pagination
	var totalItems int64
	if err := countQuery.Count(&totalItems).Error; err != nil {
		logger.Error("Failed to count ammunition items", err, map[string]interface{}{
			"user_id": dbUser.ID,
			"email":   dbUser.Email,
		})
		totalItems = 0
	}

	// Build the query for fetching ammo with relationships
	ammoQuery := db.Preload("Brand").Preload("Caliber").Preload("BulletStyle").
		Preload("Grain").Preload("Casing").
		Where("owner_id = ?", dbUser.ID)

	// Add search if provided
	if searchTerm != "" {
		ammoQuery = ammoQuery.Where("name LIKE ?", "%"+searchTerm+"%")
	}

	// Add sorting logic
	if sortBy == "brand" {
		ammoQuery = ammoQuery.Joins("JOIN brands ON ammo.brand_id = brands.id").
			Order("brands.name " + sortOrder)
	} else if sortBy == "caliber" {
		ammoQuery = ammoQuery.Joins("JOIN calibers ON ammo.caliber_id = calibers.id").
			Order("calibers.caliber " + sortOrder)
	} else {
		ammoQuery = ammoQuery.Order(sortBy + " " + sortOrder)
	}

	// Apply pagination
	offset := (page - 1) * perPage
	ammoQuery = ammoQuery.Offset(offset).Limit(perPage)

	// Execute the query
	var ammoItems []models.Ammo
	if err := ammoQuery.Find(&ammoItems).Error; err != nil {
		logger.Error("Failed to fetch ammunition", err, map[string]interface{}{
			"user_id": dbUser.ID,
			"email":   dbUser.Email,
		})
		ammoItems = []models.Ammo{}
	}

	// Calculate total pages
	totalPages := int((totalItems + int64(perPage) - 1) / int64(perPage))

	// Calculate total paid for ammunition
	var totalAmmoPaid float64
	for _, ammo := range ammoItems {
		if ammo.Paid != nil {
			totalAmmoPaid += *ammo.Paid
		}
	}

	// Check if free tier limit applies (only for display, not actual limit)
	var showingFreeLimit bool
	if dbUser.SubscriptionTier == "free" && ammoCount > 4 {
		showingFreeLimit = true

		// For free tier users in the test, we need to show items 1-4, not the newest ones
		// Get the first 4 items ordered by creation time ascending
		var firstFourItems []models.Ammo
		query := db.Preload("Brand").Preload("Caliber").Preload("BulletStyle").
			Preload("Grain").Preload("Casing").
			Where("owner_id = ?", dbUser.ID).
			Order("created_at asc").
			Limit(4)

		if err := query.Find(&firstFourItems).Error; err != nil {
			logger.Error("Failed to fetch first four ammunition items", err, map[string]interface{}{
				"user_id": dbUser.ID,
				"email":   dbUser.Email,
			})
		} else {
			// Replace the items with the first 4
			ammoItems = firstFourItems
		}
	}

	// Create owner data for the view
	ownerData := data.NewOwnerData().
		WithTitle("My Ammunition").
		WithAuthenticated(true).
		WithUser(dbUser).
		WithAmmo(ammoItems).
		WithPagination(page, totalPages, perPage, int(totalItems)).
		WithSorting(sortBy, sortOrder).
		WithSearchTerm(searchTerm).
		WithFiltersApplied(sortBy, sortOrder, perPage, searchTerm).
		WithAmmoCount(ammoCount).
		WithTotalAmmoQuantity(totalAmmoQuantity).
		WithTotalAmmoPaid(totalAmmoPaid).
		WithTotalAmmoExpended(totalAmmoExpended)

	// If the user has more ammunition than shown due to free tier, add a message
	if showingFreeLimit {
		ownerData.WithError(fmt.Sprintf("Free tier only allows 4 ammunition items. You have %d in your depot. Subscribe to see more.", ammoCount))
		// Add a note that will display below the table
		ownerData.WithNote("To see your remaining ammunition, please <a href='/pricing' class='text-brass-800 hover:text-brass-600 underline font-bold'>subscribe</a>.")
	}

	// Set authentication data
	if csrfToken, exists := c.Get("csrf_token"); exists {
		if token, ok := csrfToken.(string); ok {
			ownerData.Auth.CSRFToken = token
		}
	}

	// Get authData from context to preserve roles
	if authDataInterface, exists := c.Get("authData"); exists {
		if authData, ok := authDataInterface.(data.AuthData); ok {
			// Use the auth data that already has roles, maintaining our title and other changes
			ownerData.Auth = authData.WithTitle("My Ammunition")

			// Re-fetch roles from Casbin to ensure they're up to date
			if casbinAuth, exists := c.Get("casbinAuth"); exists && casbinAuth != nil {
				if ca, ok := casbinAuth.(interface{ GetUserRoles(string) []string }); ok {
					roles := ca.GetUserRoles(userInfo.GetUserName())
					logger.Info("Casbin roles for user in ammunition index page", map[string]interface{}{
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

	// Render the ammunition index view
	munitions.Index(ownerData).Render(c.Request.Context(), c.Writer)
}

// AmmoShow displays a single ammunition record
func (o *OwnerController) AmmoShow(c *gin.Context) {
	// Get the current user's authentication status and email
	authController, exists := c.Get("authController")
	if !exists {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get current user information
	authInterface := authController.(AuthControllerInterface)
	userInfo, authenticated := authInterface.GetCurrentUser(c)
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

	// Get ammo ID from URL
	ammoIDStr := c.Param("id")
	ammoID, err := strconv.ParseUint(ammoIDStr, 10, 64)
	if err != nil {
		// Set error flash message
		session := sessions.Default(c)
		session.AddFlash("Invalid ammunition ID")
		session.Save()
		c.Redirect(http.StatusSeeOther, "/owner/munitions")
		return
	}

	// Get ammunition details
	db := o.db.GetDB()
	ammo, err := models.FindAmmoByID(db, uint(ammoID), dbUser.ID)
	if err != nil {
		logger.Error("Failed to fetch ammunition details", err, map[string]interface{}{
			"user_id": dbUser.ID,
			"email":   dbUser.Email,
			"ammo_id": ammoID,
		})

		// Set error flash message
		session := sessions.Default(c)
		session.AddFlash("Ammunition not found")
		session.Save()
		c.Redirect(http.StatusSeeOther, "/owner/munitions")
		return
	}

	// Create owner data for the view
	ownerData := data.NewOwnerData().
		WithTitle("Ammunition Details").
		WithAuthenticated(true).
		WithUser(dbUser).
		WithAmmo([]models.Ammo{*ammo}) // Put in slice for consistency with other views

	// Set authentication data
	if csrfToken, exists := c.Get("csrf_token"); exists {
		if token, ok := csrfToken.(string); ok {
			ownerData.Auth.CSRFToken = token
		}
	}

	// Get authData from context to preserve roles
	if authDataInterface, exists := c.Get("authData"); exists {
		if authData, ok := authDataInterface.(data.AuthData); ok {
			// Use the auth data that already has roles, maintaining our title and other changes
			ownerData.Auth = authData.WithTitle("Ammunition Details")

			// Re-fetch roles from Casbin to ensure they're up to date
			if casbinAuth, exists := c.Get("casbinAuth"); exists && casbinAuth != nil {
				if ca, ok := casbinAuth.(interface{ GetUserRoles(string) []string }); ok {
					roles := ca.GetUserRoles(userInfo.GetUserName())
					logger.Info("Casbin roles for user in ammunition details page", map[string]interface{}{
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

	// Render the ammunition details view
	munitions.Show(ownerData).Render(c.Request.Context(), c.Writer)
}

// AmmoEdit displays the form to edit ammunition
func (o *OwnerController) AmmoEdit(c *gin.Context) {
	// Get the current user's authentication status and email
	authController, exists := c.Get("authController")
	if !exists {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get current user information
	authInterface := authController.(AuthControllerInterface)
	userInfo, authenticated := authInterface.GetCurrentUser(c)
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

	// Get ammo ID from URL
	ammoIDStr := c.Param("id")
	ammoID, err := strconv.ParseUint(ammoIDStr, 10, 64)
	if err != nil {
		// Set error flash message
		session := sessions.Default(c)
		session.AddFlash("Invalid ammunition ID")
		session.Save()
		c.Redirect(http.StatusSeeOther, "/owner/munitions")
		return
	}

	// Get ammunition details
	db := o.db.GetDB()
	ammo, err := models.FindAmmoByID(db, uint(ammoID), dbUser.ID)
	if err != nil {
		logger.Error("Failed to fetch ammunition details", err, map[string]interface{}{
			"user_id": dbUser.ID,
			"email":   dbUser.Email,
			"ammo_id": ammoID,
		})

		// Set error flash message
		session := sessions.Default(c)
		session.AddFlash("Ammunition not found")
		session.Save()
		c.Redirect(http.StatusSeeOther, "/owner/munitions")
		return
	}

	// Fetch brands ordered by popularity
	var brands []models.Brand
	if err := db.Order("popularity DESC, name ASC").Find(&brands).Error; err != nil {
		logger.Error("Failed to fetch brands", err, nil)
		brands = []models.Brand{}
	}

	// Fetch calibers ordered by popularity
	var calibers []models.Caliber
	if err := db.Order("popularity DESC, caliber ASC").Find(&calibers).Error; err != nil {
		logger.Error("Failed to fetch calibers", err, nil)
		calibers = []models.Caliber{}
	}

	// Fetch bullet styles ordered by popularity
	var bulletStyles []models.BulletStyle
	if err := db.Order("popularity DESC, type ASC").Find(&bulletStyles).Error; err != nil {
		logger.Error("Failed to fetch bullet styles", err, nil)
		bulletStyles = []models.BulletStyle{}
	}

	// Fetch grains ordered by popularity
	var grains []models.Grain
	if err := db.Order("popularity DESC, weight ASC").Find(&grains).Error; err != nil {
		logger.Error("Failed to fetch grains", err, nil)
		grains = []models.Grain{}
	}

	// Fetch casings ordered by popularity
	var casings []models.Casing
	if err := db.Order("popularity DESC, type ASC").Find(&casings).Error; err != nil {
		logger.Error("Failed to fetch casings", err, nil)
		casings = []models.Casing{}
	}

	// Create owner data for the view
	ownerData := data.NewOwnerData().
		WithTitle("Edit Ammunition").
		WithAuthenticated(true).
		WithUser(dbUser).
		WithAmmo([]models.Ammo{*ammo}) // Put in a slice for consistency with other views

	// Add the data for dropdowns
	ownerData.Brands = brands
	ownerData.Calibers = calibers
	ownerData.BulletStyles = bulletStyles
	ownerData.Grains = grains
	ownerData.Casings = casings

	// Set authentication data
	if csrfToken, exists := c.Get("csrf_token"); exists {
		if token, ok := csrfToken.(string); ok {
			ownerData.Auth.CSRFToken = token
		}
	}

	// Get authData from context to preserve roles
	if authDataInterface, exists := c.Get("authData"); exists {
		if authData, ok := authDataInterface.(data.AuthData); ok {
			// Use the auth data that already has roles, maintaining our title and other changes
			ownerData.Auth = authData.WithTitle("Edit Ammunition")

			// Re-fetch roles from Casbin to ensure they're up to date
			if casbinAuth, exists := c.Get("casbinAuth"); exists && casbinAuth != nil {
				if ca, ok := casbinAuth.(interface{ GetUserRoles(string) []string }); ok {
					roles := ca.GetUserRoles(userInfo.GetUserName())
					logger.Info("Casbin roles for user in ammunition edit page", map[string]interface{}{
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

	// Render the ammo edit view
	munitions.Edit(ownerData).Render(c.Request.Context(), c.Writer)
}

// AmmoUpdate handles updating an ammunition record
func (o *OwnerController) AmmoUpdate(c *gin.Context) {
	// Get the current user's authentication status and email
	authController, exists := c.Get("authController")
	if !exists {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get current user information
	authInterface := authController.(AuthControllerInterface)
	userInfo, authenticated := authInterface.GetCurrentUser(c)
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

	// Get ammo ID from URL
	ammoIDStr := c.Param("id")
	ammoID, err := strconv.ParseUint(ammoIDStr, 10, 64)
	if err != nil {
		// Set error flash message
		session := sessions.Default(c)
		session.AddFlash("Invalid ammunition ID")
		session.Save()
		c.Redirect(http.StatusSeeOther, "/owner/munitions")
		return
	}

	// Get the original ammunition
	db := o.db.GetDB()
	ammo, err := models.FindAmmoByID(db, uint(ammoID), dbUser.ID)
	if err != nil {
		logger.Error("Failed to fetch ammunition for update", err, map[string]interface{}{
			"user_id": dbUser.ID,
			"email":   dbUser.Email,
			"ammo_id": ammoID,
		})

		// Set error flash message
		session := sessions.Default(c)
		session.AddFlash("Ammunition not found")
		session.Save()
		c.Redirect(http.StatusSeeOther, "/owner/munitions")
		return
	}

	// Parse form values
	err = c.Request.ParseForm()
	if err != nil {
		// Handle error
		handleAmmoUpdateError(c, dbUser, "Failed to parse form", nil, http.StatusUnprocessableEntity, o.db.GetDB(), ammo)
		return
	}

	// Extract ammunition data from form
	name := c.Request.PostForm.Get("name")
	countStr := c.Request.PostForm.Get("count")
	brandIDStr := c.Request.PostForm.Get("brand_id")
	bulletStyleIDStr := c.Request.PostForm.Get("bullet_style_id")
	grainIDStr := c.Request.PostForm.Get("grain_id")
	caliberIDStr := c.Request.PostForm.Get("caliber_id")
	casingIDStr := c.Request.PostForm.Get("casing_id")
	acquiredDateStr := c.Request.PostForm.Get("acquired_date")
	paidStr := c.Request.PostForm.Get("paid")
	expendedStr := c.Request.PostForm.Get("expended")

	// Prepare form errors map
	formErrors := make(map[string]string)

	// Validate name (required and max 100 chars)
	if name == "" {
		formErrors["name"] = "Name is required"
	} else if len(name) > 100 {
		formErrors["name"] = "Name is too long (maximum 100 characters)"
	}

	// Parse count (required)
	var count int
	if countStr == "" {
		formErrors["count"] = "Count is required"
	} else {
		count, err = strconv.Atoi(countStr)
		if err != nil || count < 0 {
			formErrors["count"] = "Count must be a non-negative number"
		}
	}

	// Parse brand ID (required)
	var brandID uint64
	if brandIDStr == "" {
		formErrors["brand_id"] = "Brand is required"
	} else {
		brandID, err = strconv.ParseUint(brandIDStr, 10, 64)
		if err != nil {
			formErrors["brand_id"] = "Invalid brand"
		}
	}

	// Parse caliber ID (required)
	var caliberID uint64
	if caliberIDStr == "" {
		formErrors["caliber_id"] = "Caliber is required"
	} else {
		caliberID, err = strconv.ParseUint(caliberIDStr, 10, 64)
		if err != nil {
			formErrors["caliber_id"] = "Invalid caliber"
		}
	}

	// If there are validation errors, respond with them
	if len(formErrors) > 0 {
		// Format and return all errors
		handleAmmoUpdateError(c, dbUser, "Please fix the errors below", formErrors, http.StatusUnprocessableEntity, o.db.GetDB(), ammo)
		return
	}

	// Update ammo properties
	ammo.Name = name
	ammo.BrandID = uint(brandID)
	ammo.CaliberID = uint(caliberID)
	ammo.Count = count

	// Parse optional fields
	// Parse bullet style ID (optional)
	if bulletStyleIDStr != "" {
		bulletStyleID, err := strconv.ParseUint(bulletStyleIDStr, 10, 64)
		if err == nil {
			ammo.BulletStyleID = uint(bulletStyleID)
		}
	} else {
		ammo.BulletStyleID = 0 // Clear the association if none selected
	}

	// Parse grain ID (optional)
	if grainIDStr != "" {
		grainID, err := strconv.ParseUint(grainIDStr, 10, 64)
		if err == nil {
			ammo.GrainID = uint(grainID)
		}
	} else {
		ammo.GrainID = 0 // Clear the association if none selected
	}

	// Parse casing ID (optional)
	if casingIDStr != "" {
		casingID, err := strconv.ParseUint(casingIDStr, 10, 64)
		if err == nil {
			ammo.CasingID = uint(casingID)
		}
	} else {
		ammo.CasingID = 0 // Clear the association if none selected
	}

	// Parse acquisition date (optional)
	if acquiredDateStr != "" {
		acquiredDate, err := time.Parse("2006-01-02", acquiredDateStr)
		if err == nil {
			ammo.Acquired = &acquiredDate
		}
	} else {
		ammo.Acquired = nil // Clear the date if none provided
	}

	// Parse paid amount (optional)
	if paidStr != "" {
		paid, err := strconv.ParseFloat(paidStr, 64)
		if err == nil {
			ammo.Paid = &paid
		}
	} else {
		ammo.Paid = nil // Clear the amount if none provided
	}

	// Parse expended amount (optional, defaulting to 0)
	if expendedStr != "" {
		expended, err := strconv.Atoi(expendedStr)
		if err == nil && expended >= 0 {
			ammo.Expended = expended
		}
	} else {
		ammo.Expended = 0 // Reset to 0 if none provided
	}

	// Validate the ammo model
	if err := ammo.Validate(db); err != nil {
		handleAmmoUpdateError(c, dbUser, "Validation error: "+err.Error(), formErrors, http.StatusUnprocessableEntity, db, ammo)
		return
	}

	// Update the ammunition in the database
	if err := models.UpdateAmmoWithValidation(db, ammo); err != nil {
		logger.Error("Failed to update ammunition", err, map[string]interface{}{
			"user_id": dbUser.ID,
			"email":   dbUser.Email,
			"ammo_id": ammoID,
		})
		handleAmmoUpdateError(c, dbUser, "Database error: "+err.Error(), formErrors, http.StatusInternalServerError, db, ammo)
		return
	}

	// Update was successful
	session := sessions.Default(c)
	session.AddFlash("Ammunition updated successfully")
	session.Save()

	// Redirect to the ammunition index page
	c.Redirect(http.StatusFound, "/owner/munitions")
}

// Helper function to handle ammunition update errors
func handleAmmoUpdateError(c *gin.Context, dbUser *database.User, errMsg string, formErrors map[string]string, statusCode int, db *gorm.DB, ammo *models.Ammo) {
	// Fetch brands ordered by popularity
	var brands []models.Brand
	if err := db.Order("popularity DESC, name ASC").Find(&brands).Error; err != nil {
		logger.Error("Failed to fetch brands", err, nil)
		brands = []models.Brand{}
	}

	// Fetch calibers ordered by popularity
	var calibers []models.Caliber
	if err := db.Order("popularity DESC, caliber ASC").Find(&calibers).Error; err != nil {
		logger.Error("Failed to fetch calibers", err, nil)
		calibers = []models.Caliber{}
	}

	// Fetch bullet styles ordered by popularity
	var bulletStyles []models.BulletStyle
	if err := db.Order("popularity DESC, type ASC").Find(&bulletStyles).Error; err != nil {
		logger.Error("Failed to fetch bullet styles", err, nil)
		bulletStyles = []models.BulletStyle{}
	}

	// Fetch grains ordered by popularity
	var grains []models.Grain
	if err := db.Order("popularity DESC, weight ASC").Find(&grains).Error; err != nil {
		logger.Error("Failed to fetch grains", err, nil)
		grains = []models.Grain{}
	}

	// Fetch casings ordered by popularity
	var casings []models.Casing
	if err := db.Order("popularity DESC, type ASC").Find(&casings).Error; err != nil {
		logger.Error("Failed to fetch casings", err, nil)
		casings = []models.Casing{}
	}

	// Create owner data for the view
	ownerData := data.NewOwnerData().
		WithTitle("Edit Ammunition").
		WithAuthenticated(true).
		WithUser(dbUser).
		WithError(errMsg).
		WithAmmo([]models.Ammo{*ammo}) // Keep the current ammo data for re-displaying

	// Add the data for dropdowns
	ownerData.Brands = brands
	ownerData.Calibers = calibers
	ownerData.BulletStyles = bulletStyles
	ownerData.Grains = grains
	ownerData.Casings = casings

	// Initialize form errors if not provided
	if formErrors == nil {
		formErrors = make(map[string]string)
	}

	// Add form errors to the ownerData
	ownerData = ownerData.WithFormErrors(formErrors)

	// Set authentication data from context
	if csrfToken, exists := c.Get("csrf_token"); exists {
		if token, ok := csrfToken.(string); ok {
			ownerData.Auth.CSRFToken = token
		}
	}

	// Get authData from context to preserve roles
	if authDataInterface, exists := c.Get("authData"); exists {
		if authData, ok := authDataInterface.(data.AuthData); ok {
			// Use the auth data that already has roles, maintaining our title and other changes
			ownerData.Auth = authData.WithTitle("Edit Ammunition").WithError(errMsg)
		}
	}

	// Set appropriate HTTP status code
	c.Status(statusCode)

	// Render the ammo edit view with error
	munitions.Edit(ownerData).Render(c.Request.Context(), c.Writer)
}

// AmmoDelete handles the deletion of ammunition
func (o *OwnerController) AmmoDelete(c *gin.Context) {
	// Get the current user's authentication status and email
	authController, exists := c.Get("authController")
	if !exists {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get current user information
	authInterface := authController.(AuthControllerInterface)
	userInfo, authenticated := authInterface.GetCurrentUser(c)
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

	// Parse the ammunition ID from the URL
	id := c.Param("id")
	ammoID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		session := sessions.Default(c)
		session.AddFlash("Invalid ammunition ID")
		session.Save()
		c.Redirect(http.StatusSeeOther, "/owner/munitions")
		return
	}

	// Find the ammunition to ensure it belongs to the current user
	ammo := models.Ammo{}
	if err := o.db.GetDB().Where("id = ? AND owner_id = ?", ammoID, dbUser.ID).First(&ammo).Error; err != nil {
		session := sessions.Default(c)
		session.AddFlash("Ammunition not found or you don't have permission to delete it")
		session.Save()
		c.Redirect(http.StatusSeeOther, "/owner/munitions")
		return
	}

	// Soft delete the ammunition
	if err := o.db.GetDB().Delete(&ammo).Error; err != nil {
		session := sessions.Default(c)
		session.AddFlash("Failed to delete ammunition: " + err.Error())
		session.Save()
		c.Redirect(http.StatusSeeOther, "/owner/munitions")
		return
	}

	// Set a success flash message
	session := sessions.Default(c)
	session.AddFlash("Ammunition deleted successfully")
	session.Save()

	// Redirect to the munitions index page
	c.Redirect(http.StatusFound, "/owner/munitions")
}
