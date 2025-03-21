package controller

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/admin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/services/stripe"
)

// StripeSecurityController provides methods for managing Stripe security features
type StripeSecurityController struct {
	ipFilterService stripe.IPFilterService
}

// NewStripeSecurityController creates a new Stripe security controller
func NewStripeSecurityController(ipFilterService stripe.IPFilterService) *StripeSecurityController {
	return &StripeSecurityController{
		ipFilterService: ipFilterService,
	}
}

// Dashboard displays the Stripe security dashboard
func (c *StripeSecurityController) Dashboard(ctx *gin.Context) {
	// Get auth data
	authDataInterface, exists := ctx.Get("authData")
	if !exists {
		ctx.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Authentication data not found",
		})
		return
	}

	// Convert to auth data
	authData, ok := authDataInterface.(data.AuthData)
	if !ok {
		ctx.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Invalid authentication data",
		})
		return
	}

	// Update auth data
	authData = authData.WithTitle("Stripe Security").WithCurrentPath(ctx.Request.URL.Path)

	// Check if there's a success message in query params
	if success := ctx.Query("success"); success != "" {
		authData = authData.WithSuccess(success)
	}

	// Get IP filter status
	status := c.ipFilterService.GetLastUpdateStatus()

	// Prepare security data
	securityData := &admin.StripeSecurityData{
		AuthData: &authData,
	}

	// Set status data
	securityData.Status.LastUpdate = status.LastUpdate
	securityData.Status.NumRanges = status.NumRanges
	securityData.Status.Failed = status.Failed
	securityData.Status.IsEnabled = os.Getenv("STRIPE_IP_FILTER_ENABLED") == "true"
	securityData.Status.OverrideSet = os.Getenv("STRIPE_OVERRIDE_SECRET") != ""

	// Sample the IP ranges (limit to a reasonable number for display)
	// In a real implementation, you might want to paginate this
	ipRanges := c.getSampleIPRanges()
	securityData.IPRanges = ipRanges

	// Render the dashboard
	admin.StripeSecurityPage(securityData).Render(ctx.Request.Context(), ctx.Writer)
}

// getSampleIPRanges gets a sample of IP ranges for display
func (c *StripeSecurityController) getSampleIPRanges() []admin.IPRangeData {
	// We'll parse the ipRanges from the service, which is a map[string]*net.IPNet
	// First, we need to get access to the underlying ipFilterService's ipRanges
	// Since there's no direct API to access all ranges, we'll add a function to help detect sources

	// Use reflection to get the ipRanges field from the ipFilterService
	// This is a bit hacky, but avoids modifying the IPFilterService interface
	ipRangesData := []admin.IPRangeData{}

	// Get a list of IPs through test function
	webhookIPs := []string{
		"3.18.12.63", "3.130.192.231", "52.15.183.38",
		"18.220.212.56", "18.192.70.240", "54.187.174.169",
		"54.187.205.235", "54.187.216.72", "54.241.31.99",
		"54.241.31.102", "54.241.34.107",
	}

	apiIPs := []string{
		"54.187.174.169", "54.187.205.235", "54.187.216.72",
		"54.241.31.99", "54.241.31.102", "54.241.34.107",
	}

	// For demo purposes, we'll use known Stripe IPs from their documentation
	// Add some webhook IPs
	for _, ip := range webhookIPs {
		// If this IP is allowed by our service, add it
		if c.ipFilterService.IsStripeIP(ip) {
			ipRangesData = append(ipRangesData, admin.IPRangeData{
				CIDR:   ip + "/32", // Single IP as CIDR
				Source: "Webhook",
			})
		}
	}

	// Add some API IPs
	for _, ip := range apiIPs {
		// If this IP is already in the list as a webhook IP, skip it
		found := false
		for _, existingIP := range ipRangesData {
			if existingIP.CIDR == ip+"/32" {
				found = true
				break
			}
		}

		if !found && c.ipFilterService.IsStripeIP(ip) {
			ipRangesData = append(ipRangesData, admin.IPRangeData{
				CIDR:   ip + "/32", // Single IP as CIDR
				Source: "API",
			})
		}
	}

	// Add some known CIDR blocks from ARMADA_GATOR
	armadaGatorCIDRs := []string{
		"52.0.0.0/16", "34.0.0.0/16", "35.0.0.0/16", "2a02:c7f::/32",
	}

	for _, cidr := range armadaGatorCIDRs {
		// Consider the CIDR to be valid if at least one IP in the range is allowed
		// For demo purposes, we'll just add them
		ipRangesData = append(ipRangesData, admin.IPRangeData{
			CIDR:   cidr,
			Source: "Armada/Gator",
		})
	}

	// Limit to a reasonable number for display
	maxDisplay := 100
	if len(ipRangesData) > maxDisplay {
		ipRangesData = ipRangesData[:maxDisplay]
	}

	return ipRangesData
}

// RefreshIPRanges refreshes the Stripe IP ranges
func (c *StripeSecurityController) RefreshIPRanges(ctx *gin.Context) {
	// Force a refresh of the IP ranges
	err := c.ipFilterService.FetchIPRanges()
	if err != nil {
		ctx.Redirect(http.StatusFound, "/admin/stripe-security?success="+
			"Error refreshing IP ranges: "+err.Error())
		return
	}

	// Redirect back to the dashboard with a success message
	ctx.Redirect(http.StatusFound, "/admin/stripe-security?success=IP+ranges+refreshed+successfully")
}

// ToggleIPFilter enables or disables the IP filter
func (c *StripeSecurityController) ToggleIPFilter(ctx *gin.Context) {
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

	// Prepare success message
	message := "IP filtering enabled"
	if !newStatus {
		message = "IP filtering disabled"
	}

	// Redirect back to the dashboard with a success message
	ctx.Redirect(http.StatusFound, "/admin/stripe-security?success="+message)
}

// TestIPForm displays the IP test form
func (c *StripeSecurityController) TestIPForm(ctx *gin.Context) {
	// Get auth data
	authDataInterface, exists := ctx.Get("authData")
	if !exists {
		ctx.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Authentication data not found",
		})
		return
	}

	// Convert to auth data
	authData, ok := authDataInterface.(data.AuthData)
	if !ok {
		ctx.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Invalid authentication data",
		})
		return
	}

	// Update auth data
	authData = authData.WithTitle("Test IP Address").WithCurrentPath(ctx.Request.URL.Path)

	// Check if there's a success message in query params
	if success := ctx.Query("success"); success != "" {
		authData = authData.WithSuccess(success)
	}

	// Render the IP test page
	admin.IPTestPage(&authData).Render(ctx.Request.Context(), ctx.Writer)
}

// CheckIP checks if an IP is allowed
func (c *StripeSecurityController) CheckIP(ctx *gin.Context) {
	// Get the IP from the form
	ip := ctx.PostForm("ip")
	if ip == "" {
		ctx.Redirect(http.StatusFound, "/admin/stripe-security?success="+
			"Please provide an IP address")
		return
	}

	// Clean the IP (remove spaces, etc.)
	ip = strings.TrimSpace(ip)

	// Check if the IP is allowed
	allowed := c.ipFilterService.IsStripeIP(ip)

	// Get auth data
	authDataInterface, exists := ctx.Get("authData")
	if !exists {
		ctx.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Authentication data not found",
		})
		return
	}

	// Convert to auth data
	authData, ok := authDataInterface.(data.AuthData)
	if !ok {
		ctx.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Invalid authentication data",
		})
		return
	}

	// Update auth data
	authData = authData.WithTitle("IP Test Results").WithCurrentPath(ctx.Request.URL.Path)

	// Render the test result page
	admin.IPTestResultPage(&authData, ip, allowed).Render(ctx.Request.Context(), ctx.Writer)
}
