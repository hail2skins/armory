package util

import (
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
)

// GetAdminDataFromContext gets admin data from context with CSRF token
// getCSRFToken is a function that retrieves the CSRF token from context
func GetAdminDataFromContext(ctx *gin.Context, title string, currentPath string, getCSRFToken func(*gin.Context) string) *data.AdminData {
	// Get admin data from context
	adminDataInterface, exists := ctx.Get("admin_data")
	if exists && adminDataInterface != nil {
		if adminData, ok := adminDataInterface.(*data.AdminData); ok {
			// Update the title and current path
			adminData.AuthData = adminData.AuthData.WithTitle(title).WithCurrentPath(currentPath)

			// Ensure CSRF token is set
			if adminData.AuthData.CSRFToken == "" {
				csrfToken := getCSRFToken(ctx)
				adminData.AuthData = adminData.AuthData.WithCSRFToken(csrfToken)
			}

			return adminData
		}
	}

	// Get auth data from context
	authDataInterface, exists := ctx.Get("authData")
	if exists && authDataInterface != nil {
		if authData, ok := authDataInterface.(data.AuthData); ok {
			// Set the title and current path
			authData = authData.WithTitle(title).WithCurrentPath(currentPath)

			// Ensure CSRF token is set
			if authData.CSRFToken == "" {
				csrfToken := getCSRFToken(ctx)
				authData = authData.WithCSRFToken(csrfToken)
			}

			// Create admin data with auth data
			adminData := data.NewAdminData()
			adminData.AuthData = authData
			return adminData
		}
	}

	// If we couldn't get auth data from context, create a new one
	adminData := data.NewAdminData()
	adminData.AuthData = adminData.AuthData.WithTitle(title).WithCurrentPath(currentPath)

	// Set CSRF token
	csrfToken := getCSRFToken(ctx)
	adminData.AuthData = adminData.AuthData.WithCSRFToken(csrfToken)

	return adminData
}
