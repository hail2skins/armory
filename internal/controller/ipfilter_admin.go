package controller

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/services/stripe"
)

// IPFilterAdminController provides methods for administering the Stripe IP filter
type IPFilterAdminController struct {
	ipFilterService stripe.IPFilterService
}

// NewIPFilterAdminController creates a new IP filter admin controller
func NewIPFilterAdminController(ipFilterService stripe.IPFilterService) *IPFilterAdminController {
	return &IPFilterAdminController{
		ipFilterService: ipFilterService,
	}
}

// RefreshIPRanges forces a refresh of the Stripe IP ranges
func (c *IPFilterAdminController) RefreshIPRanges(ctx *gin.Context) {
	// Force a refresh of the IP ranges
	err := c.ipFilterService.FetchIPRanges()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to refresh IP ranges: " + err.Error(),
		})
		return
	}

	// Return success
	ctx.JSON(http.StatusOK, gin.H{
		"message": "IP ranges refreshed successfully",
	})
}

// GetIPRangeStatus returns the current status of the IP ranges
func (c *IPFilterAdminController) GetIPRangeStatus(ctx *gin.Context) {
	// Get the status
	status := c.ipFilterService.GetLastUpdateStatus()

	// Format the time for display
	var lastUpdateTime string
	if status.LastUpdate.IsZero() {
		lastUpdateTime = "Never"
	} else {
		lastUpdateTime = status.LastUpdate.Format(time.RFC3339)
	}

	// Return the status
	ctx.JSON(http.StatusOK, gin.H{
		"last_update":  lastUpdateTime,
		"num_ranges":   status.NumRanges,
		"last_failed":  status.Failed,
		"is_enabled":   os.Getenv("STRIPE_IP_FILTER_ENABLED") == "true",
		"override_set": os.Getenv("STRIPE_OVERRIDE_SECRET") != "",
	})
}

// ToggleIPFilter enables or disables the IP filter
func (c *IPFilterAdminController) ToggleIPFilter(ctx *gin.Context) {
	// Get the current status
	currentStatus := os.Getenv("STRIPE_IP_FILTER_ENABLED") == "true"

	// Toggle the status
	newStatus := !currentStatus

	// Set the environment variable
	if newStatus {
		os.Setenv("STRIPE_IP_FILTER_ENABLED", "true")
	} else {
		os.Setenv("STRIPE_IP_FILTER_ENABLED", "false")
	}

	// Return the new status
	ctx.JSON(http.StatusOK, gin.H{
		"message":   "IP filter toggled successfully",
		"enabled":   newStatus,
		"mechanism": "Environment variable (temporary until restart)",
	})
}

// IsIPAllowed checks if a given IP is in the allowed Stripe IP ranges
func (c *IPFilterAdminController) IsIPAllowed(ctx *gin.Context) {
	// Get the IP from the query parameter
	ip := ctx.Query("ip")
	if ip == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "IP parameter is required",
		})
		return
	}

	// Check if the IP is allowed
	allowed := c.ipFilterService.IsStripeIP(ip)

	// Return the result
	ctx.JSON(http.StatusOK, gin.H{
		"ip":       ip,
		"allowed":  allowed,
		"num_ips":  c.ipFilterService.GetLastUpdateStatus().NumRanges,
		"is_valid": true, // If we got this far, the IP was valid format
	})
}
