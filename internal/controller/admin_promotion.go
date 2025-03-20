package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/admin/promotion"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
)

// AdminPromotionController handles administrative actions for promotions
type AdminPromotionController struct {
	DB database.Service
}

// NewAdminPromotionController creates a new AdminPromotionController
func NewAdminPromotionController(db database.Service) *AdminPromotionController {
	return &AdminPromotionController{
		DB: db,
	}
}

// New renders the form to create a new promotion
func (c *AdminPromotionController) New(ctx *gin.Context) {
	// Get auth data from context
	adminData := getAdminDataFromContext(ctx, "New Promotion", ctx.Request.URL.Path)

	// Render the template
	promotion.New(adminData).Render(ctx.Request.Context(), ctx.Writer)
}

// Create handles the form submission to create a new promotion
func (c *AdminPromotionController) Create(ctx *gin.Context) {
	// Get auth data from context
	adminData := getAdminDataFromContext(ctx, "New Promotion", ctx.Request.URL.Path)

	// Parse form data
	if err := ctx.Request.ParseForm(); err != nil {
		adminData.WithError("Failed to parse form data")
		promotion.New(adminData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Extract form values
	name := ctx.PostForm("name")
	promotionType := ctx.PostForm("type")
	description := ctx.PostForm("description")
	banner := ctx.PostForm("banner")

	// Parse active and displayOnHome checkboxes
	active := ctx.PostForm("active") == "true"
	displayOnHome := ctx.PostForm("displayOnHome") == "true"

	// Parse numeric values
	benefitDays := 0
	if days := ctx.PostForm("benefitDays"); days != "" {
		parsedDays, err := strconv.Atoi(days)
		if err != nil {
			adminData.WithError("Invalid benefit days value")
			promotion.New(adminData).Render(ctx.Request.Context(), ctx.Writer)
			return
		}
		benefitDays = parsedDays
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", ctx.PostForm("startDate"))
	if err != nil {
		adminData.WithError("Invalid start date")
		promotion.New(adminData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	endDate, err := time.Parse("2006-01-02", ctx.PostForm("endDate"))
	if err != nil {
		adminData.WithError("Invalid end date")
		promotion.New(adminData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Basic validation
	if name == "" {
		adminData.WithError("Name is required")
		promotion.New(adminData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	if promotionType == "" {
		adminData.WithError("Type is required")
		promotion.New(adminData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	if description == "" {
		adminData.WithError("Description is required")
		promotion.New(adminData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	if endDate.Before(startDate) {
		adminData.WithError("End date cannot be before start date")
		promotion.New(adminData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Create new promotion
	newPromotion := &models.Promotion{
		Name:          name,
		Type:          promotionType,
		Active:        active,
		StartDate:     startDate,
		EndDate:       endDate,
		BenefitDays:   benefitDays,
		DisplayOnHome: displayOnHome,
		Description:   description,
		Banner:        banner,
	}

	// Save to database
	if err := c.DB.CreatePromotion(newPromotion); err != nil {
		adminData.WithError("Failed to create promotion: " + err.Error())
		promotion.New(adminData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Redirect with success message
	ctx.Redirect(http.StatusSeeOther, "/admin/promotions?success=Promotion created successfully")
}
