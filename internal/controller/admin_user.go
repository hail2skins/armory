package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/admin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
	"gorm.io/gorm"
)

// AdminUserController handles admin user management requests
type AdminUserController struct {
	DB database.Service
}

// NewAdminUserController creates a new AdminUserController
func NewAdminUserController(db database.Service) *AdminUserController {
	return &AdminUserController{
		DB: db,
	}
}

// Index renders the user management page with a list of all users
func (c *AdminUserController) Index(ctx *gin.Context) {
	// Get auth data from context
	authData := getAuthData(ctx)
	authData = authData.WithTitle("User Management").WithCurrentPath(ctx.Request.URL.Path)

	// Check for success message in query params
	if success := ctx.Query("success"); success != "" {
		authData = authData.WithSuccess(success)
	}

	// Check for error message in query params
	if errorMsg := ctx.Query("error"); errorMsg != "" {
		authData = authData.WithError(errorMsg)
	}

	// Get pagination parameters
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(ctx.DefaultQuery("perPage", "50"))
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}

	// Get sorting parameters
	sortBy := ctx.DefaultQuery("sortBy", "created_at")
	sortOrder := ctx.DefaultQuery("sortOrder", "desc")

	// Get search query
	searchQuery := ctx.DefaultQuery("q", "")
	searchQuery = strings.TrimSpace(searchQuery)

	// Prepare DB query
	query := c.DB.GetDB().Model(&database.User{})

	// Apply search filter if provided
	if searchQuery != "" {
		searchTerm := "%" + searchQuery + "%"
		query = query.Where("email LIKE ?", searchTerm)
	}

	// Count total users matching the search
	var totalUsers int64
	query.Count(&totalUsers)

	// Get users with pagination and sorting
	offset := (page - 1) * perPage
	var dbUsers []database.User

	err := query.
		Order(sortBy + " " + sortOrder).
		Offset(offset).
		Limit(perPage).
		Find(&dbUsers).Error

	if err != nil {
		ctx.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": fmt.Sprintf("Error getting users: %v", err),
		})
		return
	}

	// Convert database users to models.User for the template
	users := make([]models.User, len(dbUsers))
	for i, user := range dbUsers {
		users[i] = UserWrapper{User: user}
	}

	// Calculate total pages
	totalPages := int(totalUsers) / perPage
	if int(totalUsers)%perPage > 0 {
		totalPages++
	}

	// Create data for the template
	userData := &data.UserListData{
		AuthData:    authData,
		Users:       users,
		TotalUsers:  totalUsers,
		CurrentPage: page,
		PerPage:     perPage,
		TotalPages:  totalPages,
		SortBy:      sortBy,
		SortOrder:   sortOrder,
		SearchQuery: searchQuery,
	}

	// Render the user management page
	admin.UserList(userData).Render(ctx.Request.Context(), ctx.Writer)
}

// Show renders the details of a specific user
func (c *AdminUserController) Show(ctx *gin.Context) {
	// Get auth data from context
	authData := getAuthData(ctx)
	authData = authData.WithTitle("User Details").WithCurrentPath(ctx.Request.URL.Path)

	// Get user ID from URL
	userID, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.Redirect(http.StatusSeeOther, "/admin/users?error=Invalid+user+ID")
		return
	}

	// Get user from database
	user, err := c.DB.GetUserByID(uint(userID))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.Redirect(http.StatusSeeOther, "/admin/users?error=User+not+found")
		} else {
			ctx.Redirect(http.StatusSeeOther, "/admin/users?error="+fmt.Sprintf("Error loading user: %v", err))
		}
		return
	}

	// Create data for the template
	userData := &data.UserDetailData{
		AuthData: authData,
		User:     UserWrapper{User: *user},
	}

	// Render the user detail page
	admin.UserDetail(userData).Render(ctx.Request.Context(), ctx.Writer)
}

// Edit renders the form to edit a user
func (c *AdminUserController) Edit(ctx *gin.Context) {
	// Get auth data from context
	authData := getAuthData(ctx)
	authData = authData.WithTitle("Edit User").WithCurrentPath(ctx.Request.URL.Path)

	// Get user ID from URL
	userID, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.Redirect(http.StatusSeeOther, "/admin/users?error=Invalid+user+ID")
		return
	}

	// Get user from database
	user, err := c.DB.GetUserByID(uint(userID))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.Redirect(http.StatusSeeOther, "/admin/users?error=User+not+found")
		} else {
			ctx.Redirect(http.StatusSeeOther, "/admin/users?error="+fmt.Sprintf("Error loading user: %v", err))
		}
		return
	}

	// Create data for the template
	userData := &data.UserEditData{
		AuthData: authData,
		User:     UserWrapper{User: *user},
	}

	// Render the user edit page
	admin.UserEdit(userData).Render(ctx.Request.Context(), ctx.Writer)
}

// Update processes the form submission to update a user
func (c *AdminUserController) Update(ctx *gin.Context) {
	// Get user ID from URL
	userID, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.Redirect(http.StatusSeeOther, "/admin/users?error=Invalid+user+ID")
		return
	}

	// Get user from database
	user, err := c.DB.GetUserByID(uint(userID))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.Redirect(http.StatusSeeOther, "/admin/users?error=User+not+found")
		} else {
			ctx.Redirect(http.StatusSeeOther, "/admin/users?error="+fmt.Sprintf("Error loading user: %v", err))
		}
		return
	}

	// Get form values
	email := ctx.PostForm("email")
	subscriptionTier := ctx.PostForm("subscription_tier")
	verified := ctx.PostForm("verified") == "on"

	// Update user fields
	user.Email = email
	user.SubscriptionTier = subscriptionTier
	user.Verified = verified

	// Save user to database
	err = c.DB.UpdateUser(ctx.Request.Context(), user)
	if err != nil {
		ctx.Redirect(http.StatusSeeOther, fmt.Sprintf("/admin/users/%d/edit?error=%s", userID, fmt.Sprintf("Error updating user: %v", err)))
		return
	}

	// Redirect to user detail page with success message
	ctx.Redirect(http.StatusSeeOther, fmt.Sprintf("/admin/users/%d?success=User+updated+successfully", userID))
}

// Delete soft-deletes a user
func (c *AdminUserController) Delete(ctx *gin.Context) {
	// Get user ID from URL
	userID, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.Redirect(http.StatusSeeOther, "/admin/users?error=Invalid+user+ID")
		return
	}

	// Get user from database
	user, err := c.DB.GetUserByID(uint(userID))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.Redirect(http.StatusSeeOther, "/admin/users?error=User+not+found")
		} else {
			ctx.Redirect(http.StatusSeeOther, "/admin/users?error="+fmt.Sprintf("Error loading user: %v", err))
		}
		return
	}

	// Soft delete the user
	err = c.DB.GetDB().Delete(user).Error
	if err != nil {
		ctx.Redirect(http.StatusSeeOther, "/admin/users?error="+fmt.Sprintf("Error deleting user: %v", err))
		return
	}

	// Redirect to user list with success message
	ctx.Redirect(http.StatusSeeOther, "/admin/users?success=User+deleted+successfully")
}

// Restore restores a soft-deleted user
func (c *AdminUserController) Restore(ctx *gin.Context) {
	// Get user ID from URL
	userID, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.Redirect(http.StatusSeeOther, "/admin/users?error=Invalid+user+ID")
		return
	}

	// Restore the user
	err = c.DB.GetDB().Unscoped().Model(&database.User{}).Where("id = ?", userID).Update("deleted_at", nil).Error
	if err != nil {
		ctx.Redirect(http.StatusSeeOther, "/admin/users?error="+fmt.Sprintf("Error restoring user: %v", err))
		return
	}

	// Redirect to user list with success message
	ctx.Redirect(http.StatusSeeOther, "/admin/users?success=User+restored+successfully")
}

// ShowGrantSubscription renders the form to grant a subscription to a user
func (c *AdminUserController) ShowGrantSubscription(ctx *gin.Context) {
	// Get auth data from context
	authData := getAuthData(ctx)
	authData = authData.WithTitle("Grant Subscription").WithCurrentPath(ctx.Request.URL.Path)

	// Get user ID from URL
	userID, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.Redirect(http.StatusSeeOther, "/admin/users?error=Invalid+user+ID")
		return
	}

	// Get user from database
	user, err := c.DB.GetUserByID(uint(userID))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.Redirect(http.StatusSeeOther, "/admin/users?error=User+not+found")
		} else {
			ctx.Redirect(http.StatusSeeOther, "/admin/users?error="+fmt.Sprintf("Error loading user: %v", err))
		}
		return
	}

	// Create data for the template
	userData := &data.UserGrantSubscriptionData{
		AuthData: authData,
		User:     UserWrapper{User: *user},
	}

	// Render the grant subscription page
	admin.UserGrantSubscription(userData).Render(ctx.Request.Context(), ctx.Writer)
}

// GrantSubscription processes the form submission to grant a subscription to a user
func (c *AdminUserController) GrantSubscription(ctx *gin.Context) {
	// Get user ID from URL
	userID, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.Redirect(http.StatusSeeOther, "/admin/users?error=Invalid+user+ID")
		return
	}

	// Get user from database
	user, err := c.DB.GetUserByID(uint(userID))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.Redirect(http.StatusSeeOther, "/admin/users?error=User+not+found")
		} else {
			ctx.Redirect(http.StatusSeeOther, "/admin/users?error="+fmt.Sprintf("Error loading user: %v", err))
		}
		return
	}

	// Get form values
	subscriptionType := ctx.PostForm("subscription_type")
	grantReason := ctx.PostForm("grant_reason")
	durationDays := ctx.PostForm("duration_days")
	isLifetime := ctx.PostForm("is_lifetime") == "on"

	// Validate form values
	if subscriptionType == "" {
		// Get auth data for error page
		authData := getAuthData(ctx)
		authData = authData.WithTitle("Grant Subscription").WithCurrentPath(ctx.Request.URL.Path)
		authData = authData.WithError("Subscription type is required")

		// Create data for the template with error
		userData := &data.UserGrantSubscriptionData{
			AuthData: authData,
			User:     UserWrapper{User: *user},
		}

		// Render the grant subscription page with error
		ctx.Status(http.StatusBadRequest)
		admin.UserGrantSubscription(userData).Render(ctx.Request.Context(), ctx.Writer)
		return
	}

	// Update user subscription based on subscription type
	user.SubscriptionStatus = "active"
	user.IsAdminGranted = true
	user.GrantReason = grantReason

	// Get admin user info from context - use default admin ID 1 if not available
	adminID := uint(1)
	// Get a reference to an admin user if needed in the future by email
	authDataInterface, exists := ctx.Get("authData")
	if exists {
		if authData, ok := authDataInterface.(data.AuthData); ok && authData.Email != "" {
			// In a real implementation, we would look up the admin user by email
			// For now, we'll just use ID 1
			// adminUser, err := c.DB.GetUserByEmail(ctx.Request.Context(), authData.Email)
			// if err == nil && adminUser != nil {
			//     adminID = adminUser.ID
			// }
		}
	}
	user.GrantedByID = adminID

	if subscriptionType == "admin_grant" {
		// For admin grants, set the tier to admin_grant
		user.SubscriptionTier = "admin_grant"

		if isLifetime {
			// For lifetime subscriptions, set IsLifetime true and don't set an end date
			user.IsLifetime = true
			user.SubscriptionEndDate = time.Time{} // Zero time for lifetime
		} else {
			// For non-lifetime, calculate end date based on duration days
			days, err := strconv.Atoi(durationDays)
			if err != nil || days <= 0 {
				// Get auth data for error page
				authData := getAuthData(ctx)
				authData = authData.WithTitle("Grant Subscription").WithCurrentPath(ctx.Request.URL.Path)
				authData = authData.WithError("Please enter a valid number of days")

				// Create data for the template with error
				userData := &data.UserGrantSubscriptionData{
					AuthData: authData,
					User:     UserWrapper{User: *user},
				}

				// Render the grant subscription page with error
				ctx.Status(http.StatusBadRequest)
				admin.UserGrantSubscription(userData).Render(ctx.Request.Context(), ctx.Writer)
				return
			}

			// Set end date based on duration
			user.SubscriptionEndDate = time.Now().AddDate(0, 0, days)
		}
	} else {
		// For existing subscription types, set the tier accordingly
		user.SubscriptionTier = subscriptionType

		// Set end date based on tier (using a standard duration)
		switch subscriptionType {
		case "monthly":
			user.SubscriptionEndDate = time.Now().AddDate(0, 1, 0) // 1 month
		case "yearly":
			user.SubscriptionEndDate = time.Now().AddDate(1, 0, 0) // 1 year
		case "lifetime", "premium_lifetime":
			user.IsLifetime = true
			user.SubscriptionEndDate = time.Time{} // Zero time for lifetime
		}
	}

	// Save user to database
	err = c.DB.UpdateUser(ctx.Request.Context(), user)
	if err != nil {
		ctx.Redirect(http.StatusSeeOther, fmt.Sprintf("/admin/users/%d/grant-subscription?error=%s", userID, fmt.Sprintf("Error updating subscription: %v", err)))
		return
	}

	// Redirect to user detail page with success message
	ctx.Redirect(http.StatusSeeOther, fmt.Sprintf("/admin/users/%d?success=Subscription+granted+successfully", userID))
}
