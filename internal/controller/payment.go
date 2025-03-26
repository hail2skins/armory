package controller

import (
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/cmd/web/views/payment"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/logger"
	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/services/stripe"
)

// AuthProvider defines an interface for authentication providers
type AuthProvider interface {
	GetCurrentUser(c *gin.Context) (models.User, bool)
}

// PaymentController handles payment-related routes
type PaymentController struct {
	db            database.Service
	stripeService stripe.Service
}

// NewPaymentController creates a new payment controller
func NewPaymentController(db database.Service) *PaymentController {
	return &PaymentController{
		db:            db,
		stripeService: stripe.NewService(db),
	}
}

// canSubscribeToTier checks if a user can subscribe to a specific tier based on their current subscription
func (p *PaymentController) canSubscribeToTier(currentTier string, targetTier string) bool {
	// Users can always upgrade to a higher tier
	switch currentTier {
	case "free":
		return true // Free users can subscribe to any tier
	case "monthly":
		// Monthly users can upgrade to yearly, lifetime, or premium_lifetime
		return targetTier != "monthly"
	case "yearly":
		// Yearly users can upgrade to lifetime or premium_lifetime
		return targetTier != "monthly" && targetTier != "yearly"
	case "lifetime":
		// Lifetime users can only upgrade to premium_lifetime
		return targetTier == "premium_lifetime"
	case "premium_lifetime":
		// Premium lifetime users cannot subscribe to any other tier
		return false
	default:
		return true
	}
}

// PricingHandler handles the GET request to /pricing
func (p *PaymentController) PricingHandler(c *gin.Context) {
	// Get the current user's authentication status and email
	var userInfo interface{ GetUserName() string }
	var authenticated bool

	// Check if authController is of type *AuthController
	authController, ok := c.MustGet("authController").(*AuthController)
	if ok {
		userInfo, authenticated = authController.GetCurrentUser(c)
	} else {
		// Handle the case where authController is not of type *AuthController
		// This could be a mock in tests
		authControllerValue := c.MustGet("authController")
		if mockAuth, ok := authControllerValue.(interface {
			GetCurrentUser(*gin.Context) (interface{ GetUserName() string }, bool)
		}); ok {
			userInfo, authenticated = mockAuth.GetCurrentUser(c)
		} else {
			// Default to unauthenticated if we can't get the auth status
			authenticated = false
		}
	}

	// Create AuthData
	authData := data.NewAuthData()
	authData.Authenticated = authenticated
	authData.Title = "Pricing"

	// Set SEO metadata
	authData = authData.WithMetaDescription("Choose from our flexible pricing plans for The Virtual Armory. From free basic access to premium lifetime memberships, find the plan that fits your collection management needs.")
	authData = authData.WithOgImage("/assets/pricing-back.jpg")
	authData = authData.WithCanonicalURL("https://" + c.Request.Host + "/pricing")
	authData = authData.WithMetaKeywords("virtual armory pricing, gun collection subscription, firearm tracking plans, arsenal management cost")
	authData = authData.WithOgType("website")

	// Add JSON-LD structured data for the pricing page
	authData = authData.WithStructuredData(map[string]interface{}{
		"@context":    "https://schema.org",
		"@type":       "Product",
		"name":        "The Virtual Armory Subscription",
		"description": "Digital firearm collection management service for responsible gun owners",
		"offers": map[string]interface{}{
			"@type":         "AggregateOffer",
			"priceCurrency": "USD",
			"lowPrice":      "0",
			"highPrice":     "100",
			"offerCount":    "4",
		},
	})

	// Set email if authenticated
	if authenticated {
		authData.Email = userInfo.GetUserName()

		// Get Casbin from context
		if casbinAuth, exists := c.Get("casbinAuth"); exists && casbinAuth != nil {
			if ca, ok := casbinAuth.(interface{ GetUserRoles(string) []string }); ok {
				roles := ca.GetUserRoles(userInfo.GetUserName())
				authData = authData.WithRoles(roles)

				// Log roles for debugging
				logger.Info("Pricing page - Casbin roles for user", map[string]interface{}{
					"email":   userInfo.GetUserName(),
					"roles":   roles,
					"isAdmin": authData.IsCasbinAdmin,
				})
			}
		}
	}

	// Check for authData from context to preserve roles
	if authDataInterface, exists := c.Get("authData"); exists {
		if contextAuthData, ok := authDataInterface.(data.AuthData); ok {
			// Use the auth data that already has roles, maintaining our title
			authData = contextAuthData.WithTitle("Pricing")

			// Set SEO metadata
			authData = authData.WithMetaDescription("Choose from our flexible pricing plans for The Virtual Armory. From free basic access to premium lifetime memberships, find the plan that fits your collection management needs.")
			authData = authData.WithOgImage("/assets/pricing-back.jpg")
			authData = authData.WithCanonicalURL("https://" + c.Request.Host + "/pricing")
			authData = authData.WithMetaKeywords("virtual armory pricing, gun collection subscription, firearm tracking plans, arsenal management cost")
			authData = authData.WithOgType("website")

			// Add JSON-LD structured data for the pricing page
			authData = authData.WithStructuredData(map[string]interface{}{
				"@context":    "https://schema.org",
				"@type":       "Product",
				"name":        "The Virtual Armory Subscription",
				"description": "Digital firearm collection management service for responsible gun owners",
				"offers": map[string]interface{}{
					"@type":         "AggregateOffer",
					"priceCurrency": "USD",
					"lowPrice":      "0",
					"highPrice":     "100",
					"offerCount":    "4",
				},
			})

			// Re-fetch roles from Casbin to ensure they're fresh
			if authenticated && authData.Email != "" {
				// Get Casbin from context
				if casbinAuth, exists := c.Get("casbinAuth"); exists && casbinAuth != nil {
					if ca, ok := casbinAuth.(interface{ GetUserRoles(string) []string }); ok {
						roles := ca.GetUserRoles(authData.Email)
						authData = authData.WithRoles(roles)

						// Log roles for debugging
						logger.Info("Pricing page - Casbin roles from context", map[string]interface{}{
							"email":   authData.Email,
							"roles":   roles,
							"isAdmin": authData.IsCasbinAdmin,
						})
					}
				}
			}
		}
	}

	// Create PricingData with the AuthData
	pricingData := payment.PricingData{
		AuthData: authData,
		// For now, we'll just use a default value for CurrentPlan
		// In the future, this will come from the user's subscription data
		CurrentPlan: "free",
	}

	// Get the CSRF token from the context and set it in PricingData
	if csrfToken, exists := c.Get("csrf_token"); exists {
		if tokenStr, ok := csrfToken.(string); ok {
			pricingData.CSRFToken = tokenStr
		}
	}

	// Process flash messages using session directly
	session := sessions.Default(c)
	if flashes := session.Flashes(); len(flashes) > 0 {
		session.Save()
		for _, flash := range flashes {
			if flashMsg, ok := flash.(string); ok {
				pricingData.Success = flashMsg
			}
		}
	}

	// If authenticated, get the user's subscription tier
	if authenticated {
		// Get the user from the database to get subscription info
		dbUser, err := p.db.GetUserByEmail(c.Request.Context(), userInfo.GetUserName())
		if err == nil && dbUser != nil {
			pricingData.CurrentPlan = dbUser.SubscriptionTier
		}
	}

	// Render the pricing page with the data
	payment.Pricing(pricingData).Render(c.Request.Context(), c.Writer)
}

// CreateCheckoutSession creates a Stripe checkout session for a subscription
func (p *PaymentController) CreateCheckoutSession(c *gin.Context) {
	// Get the current user's authentication status and email
	userInfo, authenticated := c.MustGet("authController").(*AuthController).GetCurrentUser(c)

	// Check if user is authenticated
	if !authenticated {
		// Use the setFlash function from middleware to set the flash message
		if setFlash, exists := c.Get("setFlash"); exists {
			setFlash.(func(string))("You must be logged in to subscribe")
		}

		// Redirect to login
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Verify CSRF token here if needed
	// The CSRF middleware should already be handling this, but we can add a check if necessary

	// Get the subscription tier from the form
	tier := c.PostForm("tier")
	if tier == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Subscription tier is required"})
		return
	}

	// Get the user from the database
	dbUser, err := p.db.GetUserByEmail(c.Request.Context(), userInfo.GetUserName())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	// Check if the user can subscribe to the tier
	if !p.canSubscribeToTier(dbUser.SubscriptionTier, tier) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You cannot subscribe to this tier"})
		return
	}

	// Create a checkout session
	session, err := p.stripeService.CreateCheckoutSession(dbUser, tier)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create checkout session"})
		return
	}

	// Redirect to the checkout session
	if os.Getenv("APP_ENV") == "test" {
		c.JSON(http.StatusOK, gin.H{"url": session.URL})
	} else {
		c.Redirect(http.StatusSeeOther, session.URL)
	}
}

// HandlePaymentSuccess handles the success callback from Stripe
func (p *PaymentController) HandlePaymentSuccess(c *gin.Context) {
	// Get the session ID from the query parameters
	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.String(http.StatusBadRequest, "Session ID is required")
		return
	}

	// Get the current user's authentication status and email
	userInfo, authenticated := c.MustGet("authController").(*AuthController).GetCurrentUser(c)

	// Create AuthData
	authData := data.NewAuthData()
	authData.Authenticated = authenticated
	authData.Title = "Payment Success"

	// Check for authData from context to preserve roles
	if authDataInterface, exists := c.Get("authData"); exists {
		if contextAuthData, ok := authDataInterface.(data.AuthData); ok {
			// Use the auth data that already has roles, maintaining our title
			authData = contextAuthData.WithTitle("Payment Success")
		}
	}

	// Create SuccessData with the AuthData
	successData := payment.SuccessData{
		AuthData:  authData,
		SessionID: sessionID,
	}

	// Set email if authenticated
	if authenticated {
		successData.Email = userInfo.GetUserName()
	}

	// Render the success page with the data
	payment.Success(successData).Render(c.Request.Context(), c.Writer)

	// Set a delayed redirect to /owner
	// We'll use a JS variable that the response template can use to redirect
	c.Header("HX-Redirect", "/owner")
}

// HandlePaymentCancellation handles the cancellation callback from Stripe
func (p *PaymentController) HandlePaymentCancellation(c *gin.Context) {
	// Set a flash message using the session
	session := sessions.Default(c)
	session.AddFlash("Payment cancelled")
	session.Save()

	// Redirect to the pricing page
	c.Redirect(http.StatusSeeOther, "/pricing")
}

// HandleWebhook handles Stripe webhook events
func (p *PaymentController) HandleWebhook(c *gin.Context) {
	// Read the request body
	payload, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Get the Stripe signature from the headers
	signature := c.GetHeader("Stripe-Signature")
	if signature == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Stripe signature is required"})
		return
	}

	// Handle the webhook event
	err = p.stripeService.HandleWebhook(payload, signature)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Return a 200 OK response
	c.Status(http.StatusOK)
}

// ShowPaymentHistory shows the user's payment history
func (p *PaymentController) ShowPaymentHistory(c *gin.Context) {
	// Get the current user's authentication status and email
	userInfo, authenticated := c.MustGet("authController").(*AuthController).GetCurrentUser(c)

	// Check if user is authenticated
	if !authenticated {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get the user from the database
	dbUser, err := p.db.GetUserByEmail(c.Request.Context(), userInfo.GetUserName())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	// Get the user's payments from the database
	payments, err := p.db.GetPaymentsByUserID(dbUser.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get payment history"})
		return
	}

	// Create AuthData
	authData := data.NewAuthData()
	authData.Authenticated = authenticated
	authData.Title = "Payment History"

	// Check for authData from context to preserve roles
	if authDataInterface, exists := c.Get("authData"); exists {
		if contextAuthData, ok := authDataInterface.(data.AuthData); ok {
			// Use the auth data that already has roles, maintaining our title
			authData = contextAuthData.WithTitle("Payment History")

			// Re-fetch roles from Casbin to ensure they're up to date
			if casbinAuth, exists := c.Get("casbinAuth"); exists && casbinAuth != nil {
				if ca, ok := casbinAuth.(interface{ GetUserRoles(string) []string }); ok {
					roles := ca.GetUserRoles(userInfo.GetUserName())
					logger.Info("Casbin roles for user in payment history page", map[string]interface{}{
						"email": userInfo.GetUserName(),
						"roles": roles,
					})
					authData = authData.WithRoles(roles)
				}
			}
		}
	}

	// Create PaymentHistoryData with the AuthData
	paymentHistoryData := payment.PaymentHistoryData{
		AuthData: authData,
		Payments: payments,
	}

	// Set email if authenticated
	if authenticated {
		paymentHistoryData.Email = userInfo.GetUserName()
	}

	// Render the payment history page with the data
	payment.PaymentHistory(paymentHistoryData).Render(c.Request.Context(), c.Writer)
}

// ShowCancelConfirmation shows the subscription cancellation confirmation page
func (p *PaymentController) ShowCancelConfirmation(c *gin.Context) {
	// Get the current user's authentication status and email
	userInfo, authenticated := c.MustGet("authController").(*AuthController).GetCurrentUser(c)

	// Check if user is authenticated
	if !authenticated {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get the user from the database
	dbUser, err := p.db.GetUserByEmail(c.Request.Context(), userInfo.GetUserName())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	// Check if the user has an active subscription
	if !dbUser.HasActiveSubscription() {
		c.Redirect(http.StatusSeeOther, "/pricing")
		return
	}

	// Create AuthData
	authData := data.NewAuthData()
	authData.Authenticated = authenticated
	authData.Title = "Cancel Subscription"

	// Check for authData from context to preserve roles
	if authDataInterface, exists := c.Get("authData"); exists {
		if contextAuthData, ok := authDataInterface.(data.AuthData); ok {
			// Use the auth data that already has roles, maintaining our title
			authData = contextAuthData.WithTitle("Cancel Subscription")

			// Re-fetch roles from Casbin to ensure they're up to date
			if casbinAuth, exists := c.Get("casbinAuth"); exists && casbinAuth != nil {
				if ca, ok := casbinAuth.(interface{ GetUserRoles(string) []string }); ok {
					roles := ca.GetUserRoles(userInfo.GetUserName())
					logger.Info("Casbin roles for user in cancel subscription page", map[string]interface{}{
						"email": userInfo.GetUserName(),
						"roles": roles,
					})
					authData = authData.WithRoles(roles)
				}
			}
		}
	}

	// Create CancelConfirmationData with the AuthData
	cancelConfirmationData := payment.CancelConfirmationData{
		AuthData:         authData,
		SubscriptionTier: dbUser.SubscriptionTier,
	}

	// Set email if authenticated
	if authenticated {
		cancelConfirmationData.Email = userInfo.GetUserName()
	}

	// Render the cancel confirmation page with the data
	payment.CancelConfirmation(cancelConfirmationData).Render(c.Request.Context(), c.Writer)
}

// CancelSubscription cancels the user's subscription
func (p *PaymentController) CancelSubscription(c *gin.Context) {
	// Get the current user's authentication status and email
	userInfo, authenticated := c.MustGet("authController").(*AuthController).GetCurrentUser(c)

	// Check if user is authenticated
	if !authenticated {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get the user from the database
	dbUser, err := p.db.GetUserByEmail(c.Request.Context(), userInfo.GetUserName())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	// Check if the user has an active subscription
	if !dbUser.HasActiveSubscription() {
		c.Redirect(http.StatusSeeOther, "/pricing")
		return
	}

	// Cancel the subscription at the end of the current period
	err = p.stripeService.CancelSubscription(dbUser.StripeSubscriptionID)
	if err != nil {
		// Check if the error is because the subscription is already canceled
		if strings.Contains(err.Error(), "invalid-canceled-subscription-fields") ||
			strings.Contains(err.Error(), "A canceled subscription can only update") {
			// Subscription is already canceled in Stripe, just update our database
			logger.Info("Subscription already canceled in Stripe, updating local status", nil)

			// Force the subscription status to pending_cancellation since Stripe already has it as canceled
			dbUser.SubscriptionStatus = "pending_cancellation"
			err = p.db.UpdateUser(c.Request.Context(), dbUser)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
				return
			}

			// Format subscription end date for the flash message
			var expiresMessage string
			if !dbUser.SubscriptionEndDate.IsZero() {
				expiresDate := dbUser.SubscriptionEndDate.Format("January 2, 2006")
				expiresMessage = "Your subscription has been cancelled but will remain active until " + expiresDate + "."
			} else {
				expiresMessage = "Your subscription has been cancelled."
			}

			// Set a flash message using the session
			session := sessions.Default(c)
			session.AddFlash(expiresMessage)
			session.Save()

			// Redirect to the owner dashboard
			c.Redirect(http.StatusSeeOther, "/owner")
			return
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel subscription"})
			return
		}
	}

	// Get updated subscription info to check cancellation status
	subscription, err := p.stripeService.GetSubscriptionDetails(dbUser.StripeSubscriptionID)
	if err != nil {
		logger.Error("Failed to get subscription details after cancellation", err, nil)
		// Continue even if this fails
	}

	// Update the user's subscription information
	// If the subscription is still active but marked for cancellation, we use "pending_cancellation"
	// This distinguishes between immediate cancellation and end-of-period cancellation
	if subscription != nil && subscription.Status == "active" && subscription.CancelAtPeriodEnd {
		dbUser.SubscriptionStatus = "pending_cancellation"
	} else {
		dbUser.SubscriptionStatus = "canceled"
	}

	err = p.db.UpdateUser(c.Request.Context(), dbUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	// Format subscription end date for the flash message
	var expiresMessage string
	if !dbUser.SubscriptionEndDate.IsZero() {
		expiresDate := dbUser.SubscriptionEndDate.Format("January 2, 2006")
		expiresMessage = "Your subscription has been cancelled but will remain active until " + expiresDate + "."
	} else if subscription != nil && subscription.CurrentPeriodEnd > 0 {
		// If we have the period end from Stripe, use that
		endDate := time.Unix(subscription.CurrentPeriodEnd, 0).Format("January 2, 2006")
		expiresMessage = "Your subscription has been cancelled but will remain active until " + endDate + "."
	} else {
		expiresMessage = "Your subscription has been cancelled."
	}

	// Set a flash message using the session
	session := sessions.Default(c)
	session.AddFlash(expiresMessage)
	session.Save()

	// Redirect to the owner dashboard
	c.Redirect(http.StatusSeeOther, "/owner")
}
