package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/admin/promotion"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
	"gorm.io/gorm"
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

	// Prepare initial form data
	formData := map[string]interface{}{
		"startDateFormatted":   time.Now().Format("2006-01-02"),
		"endDateFormatted":     time.Now().AddDate(0, 1, 0).Format("2006-01-02"),
		"activeChecked":        true,
		"displayOnHomeChecked": false,
		"typeOptions": []map[string]interface{}{
			{"value": "free_trial", "label": "Free Trial", "selected": true},
			{"value": "discount", "label": "Discount", "selected": false},
			{"value": "special_offer", "label": "Special Offer", "selected": false},
		},
	}

	// Add the form data to adminData
	adminData = adminData.WithFormData(formData)

	// Render the template
	promotion.New(adminData).Render(ctx.Request.Context(), ctx.Writer)
}

// Create handles the form submission to create a new promotion
func (c *AdminPromotionController) Create(ctx *gin.Context) {
	// Get auth data from context
	adminData := getAdminDataFromContext(ctx, "New Promotion", ctx.Request.URL.Path)

	// Parse form data
	if err := ctx.Request.ParseForm(); err != nil {
		// Prepare form data for display
		formData := map[string]interface{}{
			"startDateFormatted":   time.Now().Format("2006-01-02"),
			"endDateFormatted":     time.Now().AddDate(0, 1, 0).Format("2006-01-02"),
			"activeChecked":        true,
			"displayOnHomeChecked": false,
			"typeOptions": []map[string]interface{}{
				{"value": "free_trial", "label": "Free Trial", "selected": true},
				{"value": "discount", "label": "Discount", "selected": false},
				{"value": "special_offer", "label": "Special Offer", "selected": false},
			},
		}

		// Render the form again with error
		adminData = adminData.WithError("Failed to parse form data").WithFormData(formData)
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
			// Re-prepare form data for display
			formData := map[string]interface{}{
				"startDateFormatted":   time.Now().Format("2006-01-02"),
				"endDateFormatted":     time.Now().AddDate(0, 1, 0).Format("2006-01-02"),
				"activeChecked":        true,
				"displayOnHomeChecked": false,
				"typeOptions": []map[string]interface{}{
					{"value": "free_trial", "label": "Free Trial", "selected": true},
					{"value": "discount", "label": "Discount", "selected": false},
					{"value": "special_offer", "label": "Special Offer", "selected": false},
				},
			}

			// Render the form again with error
			adminData = adminData.WithError("Invalid benefit days value").WithFormData(formData)
			promotion.New(adminData).Render(ctx.Request.Context(), ctx.Writer)
			return
		}
		benefitDays = parsedDays
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", ctx.PostForm("startDate"))
	if err != nil {
		// Re-prepare form data for display
		formData := map[string]interface{}{
			"startDateFormatted":   time.Now().Format("2006-01-02"),
			"endDateFormatted":     time.Now().AddDate(0, 1, 0).Format("2006-01-02"),
			"activeChecked":        true,
			"displayOnHomeChecked": false,
			"typeOptions": []map[string]interface{}{
				{"value": "free_trial", "label": "Free Trial", "selected": true},
				{"value": "discount", "label": "Discount", "selected": false},
				{"value": "special_offer", "label": "Special Offer", "selected": false},
			},
		}

		// Render the form again with error
		adminData = adminData.WithError("Invalid start date").WithFormData(formData)
		promotion.New(adminData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	endDate, err := time.Parse("2006-01-02", ctx.PostForm("endDate"))
	if err != nil {
		// Re-prepare form data for display
		formData := map[string]interface{}{
			"startDateFormatted":   time.Now().Format("2006-01-02"),
			"endDateFormatted":     time.Now().AddDate(0, 1, 0).Format("2006-01-02"),
			"activeChecked":        true,
			"displayOnHomeChecked": false,
			"typeOptions": []map[string]interface{}{
				{"value": "free_trial", "label": "Free Trial", "selected": true},
				{"value": "discount", "label": "Discount", "selected": false},
				{"value": "special_offer", "label": "Special Offer", "selected": false},
			},
		}

		// Render the form again with error
		adminData = adminData.WithError("Invalid end date").WithFormData(formData)
		promotion.New(adminData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Basic validation
	if name == "" {
		// Re-prepare form data for display
		formData := map[string]interface{}{
			"startDateFormatted":   time.Now().Format("2006-01-02"),
			"endDateFormatted":     time.Now().AddDate(0, 1, 0).Format("2006-01-02"),
			"activeChecked":        true,
			"displayOnHomeChecked": false,
			"typeOptions": []map[string]interface{}{
				{"value": "free_trial", "label": "Free Trial", "selected": true},
				{"value": "discount", "label": "Discount", "selected": false},
				{"value": "special_offer", "label": "Special Offer", "selected": false},
			},
		}

		// Render the form again with error
		adminData = adminData.WithError("Name is required").WithFormData(formData)
		promotion.New(adminData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	if promotionType == "" {
		// Re-prepare form data for display
		formData := map[string]interface{}{
			"startDateFormatted":   time.Now().Format("2006-01-02"),
			"endDateFormatted":     time.Now().AddDate(0, 1, 0).Format("2006-01-02"),
			"activeChecked":        true,
			"displayOnHomeChecked": false,
			"typeOptions": []map[string]interface{}{
				{"value": "free_trial", "label": "Free Trial", "selected": true},
				{"value": "discount", "label": "Discount", "selected": false},
				{"value": "special_offer", "label": "Special Offer", "selected": false},
			},
		}

		// Render the form again with error
		adminData = adminData.WithError("Type is required").WithFormData(formData)
		promotion.New(adminData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	if description == "" {
		// Re-prepare form data for display
		formData := map[string]interface{}{
			"startDateFormatted":   time.Now().Format("2006-01-02"),
			"endDateFormatted":     time.Now().AddDate(0, 1, 0).Format("2006-01-02"),
			"activeChecked":        true,
			"displayOnHomeChecked": false,
			"typeOptions": []map[string]interface{}{
				{"value": "free_trial", "label": "Free Trial", "selected": true},
				{"value": "discount", "label": "Discount", "selected": false},
				{"value": "special_offer", "label": "Special Offer", "selected": false},
			},
		}

		// Render the form again with error
		adminData = adminData.WithError("Description is required").WithFormData(formData)
		promotion.New(adminData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	if endDate.Before(startDate) {
		// Re-prepare form data for display
		formData := map[string]interface{}{
			"startDateFormatted":   time.Now().Format("2006-01-02"),
			"endDateFormatted":     time.Now().AddDate(0, 1, 0).Format("2006-01-02"),
			"activeChecked":        true,
			"displayOnHomeChecked": false,
			"typeOptions": []map[string]interface{}{
				{"value": "free_trial", "label": "Free Trial", "selected": true},
				{"value": "discount", "label": "Discount", "selected": false},
				{"value": "special_offer", "label": "Special Offer", "selected": false},
			},
		}

		// Render the form again with error
		adminData = adminData.WithError("End date cannot be before start date").WithFormData(formData)
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
		// Prepare form data for display again
		formData := map[string]interface{}{
			"startDateFormatted":   startDate.Format("2006-01-02"),
			"endDateFormatted":     endDate.Format("2006-01-02"),
			"activeChecked":        active,
			"displayOnHomeChecked": displayOnHome,
			"typeOptions": []map[string]interface{}{
				{"value": "free_trial", "label": "Free Trial", "selected": promotionType == "free_trial"},
				{"value": "discount", "label": "Discount", "selected": promotionType == "discount"},
				{"value": "special_offer", "label": "Special Offer", "selected": promotionType == "special_offer"},
			},
		}

		// Render the form again with error
		adminData = adminData.WithError("Failed to create promotion: " + err.Error()).WithFormData(formData)
		promotion.New(adminData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Redirect with success message
	ctx.Redirect(http.StatusSeeOther, "/admin/dashboard?success=Promotion+created+successfully")
}

// Index displays a list of all promotions
func (c *AdminPromotionController) Index(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminDataFromContext(ctx, "Promotions", ctx.Request.URL.Path)

	// Check for success message in query params
	if success := ctx.Query("success"); success != "" {
		adminData = adminData.WithSuccess(success)
	}

	// Get all promotions from the database
	promotions, err := c.DB.FindAllPromotions()
	if err != nil {
		// If there's an error, render the index template with an error message
		component := promotion.Index(adminData.WithError("Failed to load promotions"))
		component.Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Render the index template with the promotions
	component := promotion.Index(adminData.WithPromotions(promotions))
	component.Render(ctx.Request.Context(), ctx.Writer)
}

// Show displays a specific promotion
func (c *AdminPromotionController) Show(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminDataFromContext(ctx, "Promotion Details", ctx.Request.URL.Path)

	// Get promotion ID from URL parameter
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		// For tests, use a simpler error response
		ctx.String(http.StatusBadRequest, "Invalid promotion ID")
		return
	}

	// Get the promotion from the database
	promo, err := c.DB.FindPromotionByID(uint(id))
	if err != nil {
		// For tests, use a simpler error response
		ctx.String(http.StatusNotFound, "Promotion not found")
		return
	}

	// Render the show template with the promotion
	component := promotion.Show(adminData.WithPromotion(promo))
	component.Render(ctx.Request.Context(), ctx.Writer)
}

// Edit displays the form to edit an existing promotion
func (c *AdminPromotionController) Edit(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminDataFromContext(ctx, "Edit Promotion", ctx.Request.URL.Path)

	// Get promotion ID from URL parameter
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		// For tests, use a simpler error response
		ctx.String(http.StatusBadRequest, "Invalid promotion ID")
		return
	}

	// Get the promotion from the database
	promo, err := c.DB.FindPromotionByID(uint(id))
	if err != nil {
		// For tests, use a simpler error response
		ctx.String(http.StatusNotFound, "Promotion not found")
		return
	}

	// Format dates for the template
	formData := map[string]interface{}{
		"startDateFormatted":   promo.StartDate.Format("2006-01-02"),
		"endDateFormatted":     promo.EndDate.Format("2006-01-02"),
		"activeChecked":        promo.Active,
		"displayOnHomeChecked": promo.DisplayOnHome,
		"typeOptions": []map[string]interface{}{
			{"value": "free_trial", "label": "Free Trial", "selected": promo.Type == "free_trial"},
			{"value": "discount", "label": "Discount", "selected": promo.Type == "discount"},
			{"value": "special_offer", "label": "Special Offer", "selected": promo.Type == "special_offer"},
		},
	}

	// Set the promotion in adminData
	adminData = adminData.WithPromotion(promo).WithFormData(formData)

	// Render the edit template with the promotion data
	promotion.Edit(adminData).Render(ctx.Request.Context(), ctx.Writer)
}

// Update handles the form submission to update an existing promotion
func (c *AdminPromotionController) Update(ctx *gin.Context) {
	// Get admin data from context
	adminData := getAdminDataFromContext(ctx, "Edit Promotion", ctx.Request.URL.Path)

	// Get promotion ID from URL parameter
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		// For tests, use a simpler error response
		ctx.String(http.StatusBadRequest, "Invalid promotion ID")
		return
	}

	// Get the existing promotion from the database
	existingPromo, err := c.DB.FindPromotionByID(uint(id))
	if err != nil {
		// For tests, use a simpler error response
		ctx.String(http.StatusNotFound, "Promotion not found")
		return
	}

	// Parse form data
	if err := ctx.Request.ParseForm(); err != nil {
		// Re-prepare form data for display
		formData := map[string]interface{}{
			"startDateFormatted":   existingPromo.StartDate.Format("2006-01-02"),
			"endDateFormatted":     existingPromo.EndDate.Format("2006-01-02"),
			"activeChecked":        existingPromo.Active,
			"displayOnHomeChecked": existingPromo.DisplayOnHome,
			"typeOptions": []map[string]interface{}{
				{"value": "free_trial", "label": "Free Trial", "selected": existingPromo.Type == "free_trial"},
				{"value": "discount", "label": "Discount", "selected": existingPromo.Type == "discount"},
				{"value": "special_offer", "label": "Special Offer", "selected": existingPromo.Type == "special_offer"},
			},
		}

		// Render the form again with error
		adminData = adminData.WithError("Failed to parse form data").WithPromotion(existingPromo).WithFormData(formData)
		promotion.Edit(adminData).Render(ctx.Request.Context(), ctx.Writer)
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
			// Re-prepare form data for display
			formData := map[string]interface{}{
				"startDateFormatted":   existingPromo.StartDate.Format("2006-01-02"),
				"endDateFormatted":     existingPromo.EndDate.Format("2006-01-02"),
				"activeChecked":        existingPromo.Active,
				"displayOnHomeChecked": existingPromo.DisplayOnHome,
				"typeOptions": []map[string]interface{}{
					{"value": "free_trial", "label": "Free Trial", "selected": existingPromo.Type == "free_trial"},
					{"value": "discount", "label": "Discount", "selected": existingPromo.Type == "discount"},
					{"value": "special_offer", "label": "Special Offer", "selected": existingPromo.Type == "special_offer"},
				},
			}

			// Render the form again with error
			adminData = adminData.WithError("Invalid benefit days value").WithPromotion(existingPromo).WithFormData(formData)
			promotion.Edit(adminData).Render(ctx.Request.Context(), ctx.Writer)
			return
		}
		benefitDays = parsedDays
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", ctx.PostForm("startDate"))
	if err != nil {
		// Re-prepare form data for display
		formData := map[string]interface{}{
			"startDateFormatted":   existingPromo.StartDate.Format("2006-01-02"),
			"endDateFormatted":     existingPromo.EndDate.Format("2006-01-02"),
			"activeChecked":        existingPromo.Active,
			"displayOnHomeChecked": existingPromo.DisplayOnHome,
			"typeOptions": []map[string]interface{}{
				{"value": "free_trial", "label": "Free Trial", "selected": existingPromo.Type == "free_trial"},
				{"value": "discount", "label": "Discount", "selected": existingPromo.Type == "discount"},
				{"value": "special_offer", "label": "Special Offer", "selected": existingPromo.Type == "special_offer"},
			},
		}

		// Render the form again with error
		adminData = adminData.WithError("Invalid start date").WithPromotion(existingPromo).WithFormData(formData)
		promotion.Edit(adminData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	endDate, err := time.Parse("2006-01-02", ctx.PostForm("endDate"))
	if err != nil {
		// Re-prepare form data for display
		formData := map[string]interface{}{
			"startDateFormatted":   existingPromo.StartDate.Format("2006-01-02"),
			"endDateFormatted":     existingPromo.EndDate.Format("2006-01-02"),
			"activeChecked":        existingPromo.Active,
			"displayOnHomeChecked": existingPromo.DisplayOnHome,
			"typeOptions": []map[string]interface{}{
				{"value": "free_trial", "label": "Free Trial", "selected": existingPromo.Type == "free_trial"},
				{"value": "discount", "label": "Discount", "selected": existingPromo.Type == "discount"},
				{"value": "special_offer", "label": "Special Offer", "selected": existingPromo.Type == "special_offer"},
			},
		}

		// Render the form again with error
		adminData = adminData.WithError("Invalid end date").WithPromotion(existingPromo).WithFormData(formData)
		promotion.Edit(adminData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Basic validation
	if name == "" {
		// Re-prepare form data for display
		formData := map[string]interface{}{
			"startDateFormatted":   existingPromo.StartDate.Format("2006-01-02"),
			"endDateFormatted":     existingPromo.EndDate.Format("2006-01-02"),
			"activeChecked":        existingPromo.Active,
			"displayOnHomeChecked": existingPromo.DisplayOnHome,
			"typeOptions": []map[string]interface{}{
				{"value": "free_trial", "label": "Free Trial", "selected": existingPromo.Type == "free_trial"},
				{"value": "discount", "label": "Discount", "selected": existingPromo.Type == "discount"},
				{"value": "special_offer", "label": "Special Offer", "selected": existingPromo.Type == "special_offer"},
			},
		}

		// Render the form again with error
		adminData = adminData.WithError("Name is required").WithPromotion(existingPromo).WithFormData(formData)
		promotion.Edit(adminData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	if promotionType == "" {
		// Re-prepare form data for display
		formData := map[string]interface{}{
			"startDateFormatted":   existingPromo.StartDate.Format("2006-01-02"),
			"endDateFormatted":     existingPromo.EndDate.Format("2006-01-02"),
			"activeChecked":        existingPromo.Active,
			"displayOnHomeChecked": existingPromo.DisplayOnHome,
			"typeOptions": []map[string]interface{}{
				{"value": "free_trial", "label": "Free Trial", "selected": existingPromo.Type == "free_trial"},
				{"value": "discount", "label": "Discount", "selected": existingPromo.Type == "discount"},
				{"value": "special_offer", "label": "Special Offer", "selected": existingPromo.Type == "special_offer"},
			},
		}

		// Render the form again with error
		adminData = adminData.WithError("Type is required").WithPromotion(existingPromo).WithFormData(formData)
		promotion.Edit(adminData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	if description == "" {
		// Re-prepare form data for display
		formData := map[string]interface{}{
			"startDateFormatted":   existingPromo.StartDate.Format("2006-01-02"),
			"endDateFormatted":     existingPromo.EndDate.Format("2006-01-02"),
			"activeChecked":        existingPromo.Active,
			"displayOnHomeChecked": existingPromo.DisplayOnHome,
			"typeOptions": []map[string]interface{}{
				{"value": "free_trial", "label": "Free Trial", "selected": existingPromo.Type == "free_trial"},
				{"value": "discount", "label": "Discount", "selected": existingPromo.Type == "discount"},
				{"value": "special_offer", "label": "Special Offer", "selected": existingPromo.Type == "special_offer"},
			},
		}

		// Render the form again with error
		adminData = adminData.WithError("Description is required").WithPromotion(existingPromo).WithFormData(formData)
		promotion.Edit(adminData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	if endDate.Before(startDate) {
		// Re-prepare form data for display
		formData := map[string]interface{}{
			"startDateFormatted":   existingPromo.StartDate.Format("2006-01-02"),
			"endDateFormatted":     existingPromo.EndDate.Format("2006-01-02"),
			"activeChecked":        existingPromo.Active,
			"displayOnHomeChecked": existingPromo.DisplayOnHome,
			"typeOptions": []map[string]interface{}{
				{"value": "free_trial", "label": "Free Trial", "selected": existingPromo.Type == "free_trial"},
				{"value": "discount", "label": "Discount", "selected": existingPromo.Type == "discount"},
				{"value": "special_offer", "label": "Special Offer", "selected": existingPromo.Type == "special_offer"},
			},
		}

		// Render the form again with error
		adminData = adminData.WithError("End date cannot be before start date").WithPromotion(existingPromo).WithFormData(formData)
		promotion.Edit(adminData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Update promotion fields
	existingPromo.Name = name
	existingPromo.Type = promotionType
	existingPromo.Active = active
	existingPromo.StartDate = startDate
	existingPromo.EndDate = endDate
	existingPromo.BenefitDays = benefitDays
	existingPromo.DisplayOnHome = displayOnHome
	existingPromo.Description = description
	existingPromo.Banner = banner

	// Save to database
	if err := c.DB.UpdatePromotion(existingPromo); err != nil {
		// Re-prepare form data for display
		formData := map[string]interface{}{
			"startDateFormatted":   existingPromo.StartDate.Format("2006-01-02"),
			"endDateFormatted":     existingPromo.EndDate.Format("2006-01-02"),
			"activeChecked":        existingPromo.Active,
			"displayOnHomeChecked": existingPromo.DisplayOnHome,
			"typeOptions": []map[string]interface{}{
				{"value": "free_trial", "label": "Free Trial", "selected": existingPromo.Type == "free_trial"},
				{"value": "discount", "label": "Discount", "selected": existingPromo.Type == "discount"},
				{"value": "special_offer", "label": "Special Offer", "selected": existingPromo.Type == "special_offer"},
			},
		}

		// Render the form again with error
		adminData = adminData.WithError("Failed to update promotion: " + err.Error()).WithPromotion(existingPromo).WithFormData(formData)
		promotion.Edit(adminData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Redirect with success message
	ctx.Redirect(http.StatusSeeOther, "/admin/dashboard?success=Promotion+has+been+updated+successfully")
}

// Delete handles the deletion of a promotion
func (c *AdminPromotionController) Delete(ctx *gin.Context) {
	// Parse the ID parameter
	idParam := ctx.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		ctx.String(http.StatusBadRequest, "Invalid promotion ID")
		return
	}

	// Find the promotion to make sure it exists
	_, err = c.DB.FindPromotionByID(uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.String(http.StatusNotFound, "Promotion not found")
		} else {
			ctx.String(http.StatusInternalServerError, "Error finding promotion")
		}
		return
	}

	// Delete the promotion (soft delete since it uses gorm.Model)
	err = c.DB.DeletePromotion(uint(id))
	if err != nil {
		ctx.String(http.StatusInternalServerError, "Error deleting promotion")
		return
	}

	// Redirect to the dashboard with a success message
	ctx.Redirect(http.StatusSeeOther, "/admin/dashboard?success=Promotion+has+been+deleted+successfully")
}
