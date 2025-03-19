package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/admin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
)

// AdminDashboardController handles admin dashboard requests
type AdminDashboardController struct {
	DB database.Service
}

// NewAdminDashboardController creates a new AdminDashboardController
func NewAdminDashboardController(db database.Service) *AdminDashboardController {
	return &AdminDashboardController{
		DB: db,
	}
}

// getAuthData extracts authentication data from the Gin context
func getAuthData(ctx *gin.Context) data.AuthData {
	authDataInterface, exists := ctx.Get("authData")
	if !exists {
		return data.AuthData{
			Authenticated: false,
		}
	}

	// Check if it's already a pointer or if it's a value that needs to be converted
	if authData, ok := authDataInterface.(*data.AuthData); ok {
		return *authData
	} else if authData, ok := authDataInterface.(data.AuthData); ok {
		return authData
	}

	return data.AuthData{
		Authenticated: false,
	}
}

// Dashboard renders the admin dashboard
func (c *AdminDashboardController) Dashboard(ctx *gin.Context) {
	// Get auth data from context
	authData := getAuthData(ctx)
	authData = authData.WithTitle("Admin Dashboard").WithCurrentPath(ctx.Request.URL.Path)

	// Get pagination parameters
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(ctx.DefaultQuery("perPage", "10"))
	if perPage < 1 || perPage > 100 {
		perPage = 10
	}

	// Get sorting parameters
	sortBy := ctx.DefaultQuery("sortBy", "created_at")
	sortOrder := ctx.DefaultQuery("sortOrder", "desc")

	// Get user statistics
	totalUsers, err := c.DB.CountUsers()
	if err != nil {
		ctx.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": fmt.Sprintf("Error getting user statistics: %v", err),
		})
		return
	}

	// Get recent users with pagination and sorting
	offset := (page - 1) * perPage
	dbUsers, err := c.DB.FindRecentUsers(offset, perPage, sortBy, sortOrder)
	if err != nil {
		ctx.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": fmt.Sprintf("Error getting recent users: %v", err),
		})
		return
	}

	// Convert database users to models.User for the template
	// We'll use a wrapper that implements the models.User interface
	recentUsers := make([]models.User, len(dbUsers))
	for i, user := range dbUsers {
		recentUsers[i] = UserWrapper{User: user}
	}

	// Calculate total pages
	totalPages := int(totalUsers) / perPage
	if int(totalUsers)%perPage > 0 {
		totalPages++
	}

	// Mock data for dashboard statistics (replace with real data later)
	subscribedUsers := int64(float64(totalUsers) * 0.7)
	newRegistrations := int64(float64(totalUsers) * 0.1)
	newSubscriptions := int64(float64(subscribedUsers) * 0.15)

	// Create admin data with all the information
	adminData := &data.AdminData{
		AuthData:                   authData,
		TotalUsers:                 totalUsers,
		CurrentPage:                page,
		PerPage:                    perPage,
		TotalPages:                 totalPages,
		SortBy:                     sortBy,
		SortOrder:                  sortOrder,
		RecentUsers:                recentUsers,
		SubscribedUsers:            subscribedUsers,
		NewRegistrations:           newRegistrations,
		NewSubscriptions:           newSubscriptions,
		UserGrowthRate:             5.2,
		SubscribedGrowthRate:       8.7,
		NewRegistrationsGrowthRate: 12.3,
		NewSubscriptionsGrowthRate: 7.1,
		MonthlySubscribers:         int64(float64(subscribedUsers) * 0.45),
		YearlySubscribers:          int64(float64(subscribedUsers) * 0.30),
		LifetimeSubscribers:        int64(float64(subscribedUsers) * 0.15),
		PremiumSubscribers:         int64(float64(subscribedUsers) * 0.10),
		MonthlyGrowthRate:          6.8,
		YearlyGrowthRate:           4.2,
		LifetimeGrowthRate:         1.5,
		PremiumGrowthRate:          9.3,
	}

	// Render the dashboard
	admin.Dashboard(adminData).Render(ctx.Request.Context(), ctx.Writer)
}

// UserWrapper wraps a database.User to implement the models.User interface
type UserWrapper struct {
	User database.User
}

// GetID implements the User interface
func (u UserWrapper) GetID() uint {
	return u.User.ID
}

// GetUserName implements the User interface
func (u UserWrapper) GetUserName() string {
	return u.User.Email
}

// GetCreatedAt implements the User interface
func (u UserWrapper) GetCreatedAt() time.Time {
	return u.User.CreatedAt
}

// GetLastLogin implements the User interface
func (u UserWrapper) GetLastLogin() time.Time {
	return u.User.LastLoginAttempt
}

// GetSubscriptionTier implements the User interface
func (u UserWrapper) GetSubscriptionTier() string {
	return u.User.SubscriptionTier
}

// IsDeleted implements the User interface
func (u UserWrapper) IsDeleted() bool {
	// Check if DeletedAt is not nil and not zero
	return !u.User.DeletedAt.Time.IsZero()
}

// DetailedHealth renders the detailed health page
func (c *AdminDashboardController) DetailedHealth(ctx *gin.Context) {
	// Get auth data from context
	authData := getAuthData(ctx)
	authData = authData.WithTitle("Detailed Health").WithCurrentPath(ctx.Request.URL.Path)

	// Get database health status
	dbStatus := "OK"
	if dbErr := c.DB.Health(); dbErr != nil {
		dbStatus = fmt.Sprintf("Error: %v", dbErr)
	}

	// Create health data
	health := map[string]string{
		"database":        dbStatus,
		"cache":           "OK",
		"storage":         "OK",
		"email_service":   "OK",
		"payment_gateway": "OK",
		"api_gateway":     "OK",
	}

	// Create admin data with auth data
	adminData := &data.AdminData{
		AuthData: authData,
	}

	// Render the detailed health page
	admin.DetailedHealth(adminData, health).Render(ctx.Request.Context(), ctx.Writer)
}

// ErrorMetrics displays error metrics for the application
func (c *AdminDashboardController) ErrorMetrics(ctx *gin.Context) {
	// Get auth data from context
	authData := getAuthData(ctx)
	authData = authData.WithTitle("Error Metrics").WithCurrentPath(ctx.Request.URL.Path)

	// Create admin data with auth data
	adminData := &data.AdminData{
		AuthData: authData,
	}

	// Render the error metrics page
	admin.ErrorMetrics(adminData).Render(ctx.Request.Context(), ctx.Writer)
}
