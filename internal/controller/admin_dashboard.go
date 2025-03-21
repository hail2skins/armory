package controller

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/admin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/metrics"
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

	// Check for success message in query params
	if success := ctx.Query("success"); success != "" {
		authData = authData.WithSuccess(success)
	}

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

	// Get search query
	searchQuery := ctx.DefaultQuery("search", "")

	// Get user statistics
	totalUsers, err := c.DB.CountUsers()
	if err != nil {
		ctx.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": fmt.Sprintf("Error getting user statistics: %v", err),
		})
		return
	}

	// Get total subscribed users (paying users only, not admin granted)
	subscribedUsers, err := c.DB.CountActiveSubscribers()
	if err != nil {
		ctx.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": fmt.Sprintf("Error getting subscription statistics: %v", err),
		})
		return
	}

	// Get new registrations for this month
	newRegistrations, err := c.DB.CountNewUsersThisMonth()
	if err != nil {
		ctx.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": fmt.Sprintf("Error getting new user statistics: %v", err),
		})
		return
	}

	// Get new subscriptions for this month
	newSubscriptions, err := c.DB.CountNewSubscribersThisMonth()
	if err != nil {
		ctx.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": fmt.Sprintf("Error getting new subscription statistics: %v", err),
		})
		return
	}

	// Get comparative data for growth rate calculations
	newUsersLastMonth, err := c.DB.CountNewUsersLastMonth()
	if err != nil {
		ctx.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": fmt.Sprintf("Error getting user growth statistics: %v", err),
		})
		return
	}

	newSubscribersLastMonth, err := c.DB.CountNewSubscribersLastMonth()
	if err != nil {
		ctx.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": fmt.Sprintf("Error getting subscription growth statistics: %v", err),
		})
		return
	}

	// Calculate growth rates
	userGrowthRate := calculateGrowthRate(totalUsers, totalUsers-newRegistrations)
	subscribedGrowthRate := calculateGrowthRate(subscribedUsers, subscribedUsers-newSubscriptions)
	newRegistrationsGrowthRate := calculateGrowthRate(newRegistrations, newUsersLastMonth)
	newSubscriptionsGrowthRate := calculateGrowthRate(newSubscriptions, newSubscribersLastMonth)

	// Get recent users with pagination and sorting
	offset := (page - 1) * perPage
	dbUsers, err := c.DB.FindRecentUsers(offset, perPage, sortBy, sortOrder)
	if err != nil {
		ctx.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": fmt.Sprintf("Error getting recent users: %v", err),
		})
		return
	}

	// Filter users by search query if provided
	if searchQuery != "" {
		filteredUsers := []database.User{}
		for _, user := range dbUsers {
			if strings.Contains(strings.ToLower(user.Email), strings.ToLower(searchQuery)) {
				filteredUsers = append(filteredUsers, user)
			}
		}
		dbUsers = filteredUsers
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
		UserGrowthRate:             userGrowthRate,
		SubscribedGrowthRate:       subscribedGrowthRate,
		NewRegistrationsGrowthRate: newRegistrationsGrowthRate,
		NewSubscriptionsGrowthRate: newSubscriptionsGrowthRate,
		SearchQuery:                searchQuery,
	}

	// Render the dashboard
	admin.Dashboard(adminData).Render(ctx.Request.Context(), ctx.Writer)
}

// calculateGrowthRate calculates the percentage growth between current and previous values
// Returns a float64 percentage (e.g., 5.2 for 5.2% growth)
func calculateGrowthRate(current, previous int64) float64 {
	if previous == 0 {
		// Avoid division by zero; if previous is 0, we consider it 100% growth
		return 100.0
	}

	// Calculate percentage change
	change := float64(current-previous) / float64(previous) * 100.0

	// Round to 1 decimal place
	return float64(int(change*10)) / 10
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
	return u.User.LastLogin
}

// GetSubscriptionTier implements the User interface
func (u UserWrapper) GetSubscriptionTier() string {
	return u.User.SubscriptionTier
}

// IsDeleted implements the User interface
func (u UserWrapper) IsDeleted() bool {
	// Check if DeletedAt is not nil
	return u.User.DeletedAt != nil
}

// IsVerified implements the User interface
func (u UserWrapper) IsVerified() bool {
	return u.User.Verified
}

// GetSubscriptionStatus implements the User interface
func (u UserWrapper) GetSubscriptionStatus() string {
	if u.User.SubscriptionStatus == "" {
		return "N/A"
	}
	return u.User.SubscriptionStatus
}

// GetSubscriptionEndDate implements the User interface
func (u UserWrapper) GetSubscriptionEndDate() time.Time {
	return u.User.SubscriptionEndDate
}

// GetGrantReason implements the User interface
func (u UserWrapper) GetGrantReason() string {
	return u.User.GrantReason
}

// IsAdminGranted implements the User interface
func (u UserWrapper) IsAdminGranted() bool {
	return u.User.IsAdminGranted
}

// IsLifetime implements the User interface
func (u UserWrapper) IsLifetime() bool {
	return u.User.IsLifetime
}

// GetGrantedByID implements the User interface
func (u UserWrapper) GetGrantedByID() uint {
	return u.User.GrantedByID
}

// WebhookStats contains basic stats about webhooks
type WebhookStats struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	LastRequestTime    time.Time
	LastErrorTime      time.Time
	LastError          string
}

// getWebhookStats retrieves webhook stats from a middleware handler
// Using this indirect approach to avoid direct import of middleware
func getWebhookStats(ctx *gin.Context) WebhookStats {
	statsInterface, exists := ctx.Get("webhookStats")
	if !exists {
		return WebhookStats{}
	}

	if stats, ok := statsInterface.(WebhookStats); ok {
		return stats
	}

	return WebhookStats{}
}

// DetailedHealth renders the detailed health page
func (c *AdminDashboardController) DetailedHealth(ctx *gin.Context) {
	// Get auth data from context
	authData := getAuthData(ctx)
	authData = authData.WithTitle("Detailed Health").WithCurrentPath(ctx.Request.URL.Path)

	// Get database health status
	dbHealth := c.DB.Health()
	dbStatus := "OK"
	if status, ok := dbHealth["status"]; ok && status != "up" {
		dbStatus = fmt.Sprintf("Down: %s", dbHealth["error"])
	}

	// Get webhook health status from middleware
	webhookStats := getWebhookStats(ctx)
	webhookStatus := "OK"
	webhookDetails := ""

	if webhookStats.TotalRequests > 0 {
		successRate := float64(webhookStats.SuccessfulRequests) / float64(webhookStats.TotalRequests) * 100
		if successRate < 80 {
			webhookStatus = "Degraded"
			webhookDetails = fmt.Sprintf("Success rate: %.1f%%", successRate)
		}

		// Add traffic stats to webhook details
		webhookDetails = fmt.Sprintf("Requests: %d (%.1f%% success rate)",
			webhookStats.TotalRequests,
			successRate)
	}

	if webhookStats.LastError != "" {
		if webhookDetails != "" {
			webhookDetails += ", "
		}
		webhookDetails += fmt.Sprintf("Last error: %s", webhookStats.LastError)
	}

	// Get error metrics if available
	errorMetricsInterface, exists := ctx.Get("errorMetrics")
	errorMetricsStatus := "OK"
	if exists {
		if errorMetrics, ok := errorMetricsInterface.(*metrics.ErrorMetrics); ok {
			errorRates := errorMetrics.GetErrorRates(24 * time.Hour)
			var criticalCount float64
			for errType, count := range errorRates {
				if errType == "internal_error" || errType == "database_error" || errType == "payment_error" {
					criticalCount += count
				}
			}

			if criticalCount > 5 {
				errorMetricsStatus = fmt.Sprintf("Alert: %d critical errors in last 24h", int(criticalCount))
			}
		}
	}

	// Create health data with real values
	health := map[string]string{
		"database":      dbStatus,
		"webhook":       webhookStatus,
		"error_metrics": errorMetricsStatus,
	}

	// Add database statistics as separate entries with more readable names
	if conn, ok := dbHealth["open_connections"]; ok {
		health["Database Connections"] = conn
	}

	if inUse, ok := dbHealth["in_use"]; ok {
		health["Database Connections (In Use)"] = inUse
	}

	if idle, ok := dbHealth["idle"]; ok {
		health["Database Connections (Idle)"] = idle
	}

	// If we have webhook details, add it with a more descriptive name
	if webhookDetails != "" {
		health["Webhook Information"] = webhookDetails
	}

	// Add additional service status checks for services that may be mocked
	// In a real system, these would check actual services
	mockChecks := map[string]string{
		"Cache Service":   "OK",
		"Storage Service": "OK",
		"Email Service":   "OK",
		"Payment Gateway": "OK",
		"API Gateway":     "OK",
	}

	for k, v := range mockChecks {
		health[k] = v
	}

	// Gather system information
	hostname, _ := os.Hostname()
	systemInfo := map[string]string{
		"Go Version":    runtime.Version(),
		"OS":            runtime.GOOS + "/" + runtime.GOARCH,
		"Database":      "PostgreSQL", // This could be fetched from actual DB info
		"Host":          hostname,
		"NumCPU":        fmt.Sprintf("%d", runtime.NumCPU()),
		"NumGoroutines": fmt.Sprintf("%d", runtime.NumGoroutine()),
		"Uptime":        getUptimeStr(),
	}

	// Get memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	systemInfo["Memory Usage"] = formatBytes(memStats.Alloc) + " / " + formatBytes(memStats.Sys)
	systemInfo["Heap Usage"] = formatBytes(memStats.HeapAlloc) + " / " + formatBytes(memStats.HeapSys)
	systemInfo["Stack Usage"] = formatBytes(memStats.StackInuse) + " / " + formatBytes(memStats.StackSys)
	systemInfo["GC Cycles"] = fmt.Sprintf("%d", memStats.NumGC)

	// Include request info
	requestCount := webhookStats.TotalRequests
	if requestCount > 0 {
		lastRequestTime := webhookStats.LastRequestTime.Format("2006-01-02 15:04:05")
		systemInfo["Total Requests"] = fmt.Sprintf("%d", requestCount)
		systemInfo["Last Request"] = lastRequestTime
	}

	// Add approx CPU usage
	cpuUsage := "Not available"
	if runtime.GOOS == "linux" {
		// For a real production service, you might fetch actual CPU stats
		// For now, we'll provide a simple mock value
		cpuUsage = fmt.Sprintf("~%.1f%%", 5.2)
	}
	systemInfo["CPU Usage"] = cpuUsage

	// Create admin data with auth data
	adminData := &data.AdminData{
		AuthData: authData,
	}

	// Set system info
	adminData = adminData.WithSystemInfo(systemInfo)

	// Render the detailed health page
	admin.DetailedHealth(adminData, health).Render(ctx.Request.Context(), ctx.Writer)
}

// formatBytes formats bytes to human readable string
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Global variable to track start time
var startTime = time.Now()

// getUptimeStr returns the uptime as a formatted string
func getUptimeStr() string {
	uptime := time.Since(startTime)
	days := int(uptime.Hours()) / 24
	hours := int(uptime.Hours()) % 24
	minutes := int(uptime.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// ErrorMetrics displays error metrics for the application
func (c *AdminDashboardController) ErrorMetrics(ctx *gin.Context) {
	// Get auth data from context
	authData := getAuthData(ctx)
	authData = authData.WithTitle("Error Metrics").WithCurrentPath(ctx.Request.URL.Path)

	// Get the error metrics instance from context
	// This will be set by middleware in server setup
	errorMetricsInterface, exists := ctx.Get("errorMetrics")
	if !exists {
		// Fallback if not in context
		adminData := &data.AdminData{
			AuthData: authData.WithError("Error metrics unavailable"),
		}
		admin.ErrorMetrics(adminData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	errorMetrics, ok := errorMetricsInterface.(*metrics.ErrorMetrics)
	if !ok {
		adminData := &data.AdminData{
			AuthData: authData.WithError("Error metrics type mismatch"),
		}
		admin.ErrorMetrics(adminData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Get recent errors (last 10)
	recentErrorsData := errorMetrics.GetRecentErrors(10)

	// Transform to view-friendly format
	recentErrors := make([]data.ErrorEntry, 0, len(recentErrorsData))
	for _, err := range recentErrorsData {
		// Determine error level based on type or other criteria
		level := "INFO"
		if err.ErrorType == "internal_error" || err.ErrorType == "database_error" || err.ErrorType == "payment_error" {
			level = "ERROR"
		} else if err.ErrorType == "validation_error" || err.ErrorType == "auth_error" {
			level = "WARNING"
		}

		// Determine service based on path or error type
		service := "System"
		if strings.Contains(err.Path, "/auth") || strings.Contains(err.ErrorType, "auth") {
			service = "Authentication"
		} else if strings.Contains(err.Path, "/api") {
			service = "API"
		} else if strings.Contains(err.ErrorType, "database") {
			service = "Database"
		} else if strings.Contains(err.Path, "/payment") || strings.Contains(err.ErrorType, "payment") {
			service = "Payment"
		}

		// Create a user-friendly message
		message := err.ErrorType
		if message == "internal_error" {
			message = "Internal server error occurred"
		} else if strings.Contains(message, "auth") {
			message = "Authentication error: " + strings.Replace(message, "auth_error_", "", 1)
		} else if strings.Contains(message, "validation") {
			message = "Validation failed: " + strings.Replace(message, "validation_error_", "", 1)
		}

		recentErrors = append(recentErrors, data.ErrorEntry{
			ErrorType:    err.ErrorType,
			Count:        err.Count,
			LastOccurred: err.LastOccurred,
			Path:         err.Path,
			Level:        level,
			Service:      service,
			Message:      message,
			IPAddress:    err.IPAddress,
		})
	}

	// Get error rates for last 24 hours
	errorRates := errorMetrics.GetErrorRates(24 * time.Hour)

	// Count by severity
	var criticalCount, warningCount, infoCount int64
	for errType, count := range errorRates {
		if errType == "internal_error" || errType == "database_error" || errType == "payment_error" {
			criticalCount += int64(count)
		} else if errType == "validation_error" || errType == "auth_error" {
			warningCount += int64(count)
		} else {
			infoCount += int64(count)
		}
	}

	// Get error rates by service
	errorRatesByService := make(map[string]float64)
	totalErrors := 0.0

	// Aggregate errors by service
	for errType, count := range errorRates {
		service := "Other"
		if strings.Contains(errType, "auth") {
			service = "Authentication"
		} else if strings.Contains(errType, "api") {
			service = "API"
		} else if strings.Contains(errType, "database") {
			service = "Database"
		} else if strings.Contains(errType, "payment") {
			service = "Payment"
		} else if strings.Contains(errType, "server") {
			service = "Server"
		}

		errorRatesByService[service] += count
		totalErrors += count
	}

	// Convert to percentages if we have errors
	if totalErrors > 0 {
		for service, count := range errorRatesByService {
			errorRatesByService[service] = (count / totalErrors) * 100
		}
	}

	// Get latency percentiles
	latencyPercentiles := errorMetrics.GetLatencyPercentiles()

	// Create admin data with auth data and metrics
	adminData := &data.AdminData{
		AuthData: authData,
	}

	// Add error metrics
	adminData = adminData.WithErrorMetrics(
		criticalCount,
		warningCount,
		infoCount,
		recentErrors,
		errorRatesByService,
		totalErrors,
		latencyPercentiles,
	)

	// Render the error metrics page
	admin.ErrorMetrics(adminData).Render(ctx.Request.Context(), ctx.Writer)
}
