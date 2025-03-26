package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/admin/payment"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/database"
)

// AdminPaymentController handles admin payment routes
type AdminPaymentController struct {
	db database.Service
}

// NewAdminPaymentController creates a new admin payment controller
func NewAdminPaymentController(db database.Service) *AdminPaymentController {
	return &AdminPaymentController{
		db: db,
	}
}

// getAdminPaymentDataFromContext gets admin data from context
func getAdminPaymentDataFromContext(ctx *gin.Context, title string, currentPath string) *data.AdminData {
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

// ShowPaymentsHistory shows all payments in the system
func (a *AdminPaymentController) ShowPaymentsHistory(c *gin.Context) {
	// Get admin data from context
	adminData := getAdminPaymentDataFromContext(c, "Payment History", "/admin/payments-history")

	// Get all payments from the database
	payments, err := a.db.GetAllPayments()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get payments"})
		return
	}

	// Create payments data
	paymentsData := payment.PaymentsHistoryData{
		AdminData: adminData,
		Payments:  payments,
	}

	// Render the payments history page
	payment.PaymentsHistory(&paymentsData).Render(c.Request.Context(), c.Writer)
}
