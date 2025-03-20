package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/admin/promotion"
	"github.com/hail2skins/armory/internal/database"
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
