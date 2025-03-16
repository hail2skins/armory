package controller

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/cmd/web/views/owner"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
)

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
	userInfo, authenticated := c.MustGet("authController").(*AuthController).GetCurrentUser(c)
	if !authenticated {
		// Set flash message
		if setFlash, exists := c.Get("setFlash"); exists {
			setFlash.(func(string))("You must be logged in to access this page")
		}
		c.Redirect(302, "/login")
		return
	}

	// Get the user from the database
	ctx := context.Background()
	dbUser, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
	if err != nil {
		c.Redirect(302, "/login")
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
	// This will be implemented later
	c.JSON(200, gin.H{"message": "New gun page"})
}

// Create handles the create gun route
func (o *OwnerController) Create(c *gin.Context) {
	// This will be implemented later
	c.JSON(200, gin.H{"message": "Create gun"})
}

// Show handles the show gun route
func (o *OwnerController) Show(c *gin.Context) {
	// Get the current user's authentication status and email
	userInfo, authenticated := c.MustGet("authController").(*AuthController).GetCurrentUser(c)
	if !authenticated {
		c.Redirect(302, "/login")
		return
	}

	// Get the user from the database
	ctx := context.Background()
	dbUser, err := o.db.GetUserByEmail(ctx, userInfo.GetUserName())
	if err != nil {
		c.Redirect(302, "/login")
		return
	}

	// Get the gun ID from the URL
	gunID := c.Param("id")

	// Get the gun from the database
	var gun models.Gun
	db := o.db.GetDB()
	if err := db.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").Where("id = ? AND owner_id = ?", gunID, dbUser.ID).First(&gun).Error; err != nil {
		c.Redirect(302, "/owner/guns")
		return
	}

	// Render the gun show page with the data
	c.JSON(200, gin.H{
		"gun":  gun,
		"user": dbUser,
	})
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
