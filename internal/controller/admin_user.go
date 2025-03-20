package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

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
