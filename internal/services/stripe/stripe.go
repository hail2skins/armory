package stripe

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/checkout/session"
	"github.com/stripe/stripe-go/v72/customer"
	"github.com/stripe/stripe-go/v72/price"
	"github.com/stripe/stripe-go/v72/sub"
	"github.com/stripe/stripe-go/v72/webhook"
)

// Service defines the interface for Stripe operations
type Service interface {
	// CreateCheckoutSession creates a Stripe checkout session for a subscription
	CreateCheckoutSession(user *database.User, tier string) (*stripe.CheckoutSession, error)

	// HandleWebhook handles Stripe webhook events
	HandleWebhook(payload []byte, signature string) error

	// GetSubscriptionDetails gets details about a subscription
	GetSubscriptionDetails(subscriptionID string) (*stripe.Subscription, error)

	// CancelSubscription cancels a subscription
	CancelSubscription(subscriptionID string) error
}

// service implements the Service interface
type service struct {
	db database.Service
}

// NewService creates a new Stripe service
func NewService(db database.Service) Service {
	// Set Stripe API key
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	return &service{
		db: db,
	}
}

// CreateCheckoutSession creates a Stripe checkout session for a subscription
func (s *service) CreateCheckoutSession(user *database.User, tier string) (*stripe.CheckoutSession, error) {
	// Get the product ID for the tier
	productID, err := getProductIDForTier(tier)
	if err != nil {
		return nil, err
	}

	// Create or get a Stripe customer
	customerID, err := s.getOrCreateCustomer(user)
	if err != nil {
		return nil, err
	}

	// Get the base URL for success and cancel URLs
	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}

	// Create a price for the product
	priceID, err := s.createPrice(productID, tier)
	if err != nil {
		return nil, err
	}

	// Create checkout session parameters
	params := &stripe.CheckoutSessionParams{
		Customer:   stripe.String(customerID),
		SuccessURL: stripe.String(fmt.Sprintf("%s/payment/success?session_id={CHECKOUT_SESSION_ID}", baseURL)),
		CancelURL:  stripe.String(fmt.Sprintf("%s/payment/cancel", baseURL)),
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		ClientReferenceID: stripe.String(strconv.FormatUint(uint64(user.ID), 10)),
		// Enable automatic tax calculation
		AutomaticTax: &stripe.CheckoutSessionAutomaticTaxParams{
			Enabled: stripe.Bool(true),
		},
		// Allow Stripe to collect and update customer address
		CustomerUpdate: &stripe.CheckoutSessionCustomerUpdateParams{
			Address:  stripe.String("auto"),
			Shipping: stripe.String("auto"),
			Name:     stripe.String("auto"),
		},
		// Set tax ID collection
		TaxIDCollection: &stripe.CheckoutSessionTaxIDCollectionParams{
			Enabled: stripe.Bool(true),
		},
	}

	// For one-time payments (lifetime subscriptions)
	if tier == "lifetime" || tier == "premium_lifetime" {
		params.Mode = stripe.String(string(stripe.CheckoutSessionModePayment))
	}

	// Create the checkout session
	session, err := session.New(params)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// HandleWebhook handles Stripe webhook events
func (s *service) HandleWebhook(payload []byte, signature string) error {
	// Get the webhook secret
	webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	if webhookSecret == "" {
		return errors.New("stripe webhook secret is not set")
	}

	// Verify the webhook signature
	event, err := webhook.ConstructEvent(payload, signature, webhookSecret)
	if err != nil {
		return err
	}

	// Handle different event types
	switch event.Type {
	case "checkout.session.completed":
		// Handle successful checkout
		var session stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			return err
		}

		// Get the user ID from the client reference ID
		userID, err := strconv.ParseUint(session.ClientReferenceID, 10, 64)
		if err != nil {
			return err
		}

		// Log the event
		fmt.Printf("User %d completed checkout for subscription %s\n", userID, session.ID)

		// Get the user from the database
		user, err := s.db.GetUserByID(uint(userID))
		if err != nil {
			return err
		}

		// For one-time payments, we'll create a payment record here
		if session.Mode == "payment" {
			// Create a payment record
			payment := &models.Payment{
				UserID:      uint(userID),
				Amount:      session.AmountTotal,
				Currency:    string(session.Currency),
				PaymentType: "one-time",
				Status:      "succeeded",
				Description: "Lifetime subscription",
				StripeID:    session.ID,
			}

			// Save the payment to the database
			if err := s.db.CreatePayment(payment); err != nil {
				return err
			}

			// Determine the subscription tier from the amount
			var tier string
			if session.AmountTotal == 10000 {
				tier = "lifetime"
			} else if session.AmountTotal == 100000 {
				tier = "premium_lifetime"
			}

			user.SubscriptionTier = tier
			user.SubscriptionStatus = "active"

			// Update the user in the database
			if err := s.db.UpdateUser(nil, user); err != nil {
				return err
			}
		} else if session.Mode == "subscription" {
			// For subscription mode, we'll set the subscription tier based on the line items
			// The actual payment record will be created when we receive the invoice.payment_succeeded event

			// Get the subscription ID from the session
			if session.Subscription != nil {
				// Get subscription details
				subscription, err := s.GetSubscriptionDetails(session.Subscription.ID)
				if err != nil {
					return err
				}

				// Determine the subscription tier
				var tier string
				if subscription.Items != nil && len(subscription.Items.Data) > 0 {
					amount := subscription.Items.Data[0].Price.UnitAmount
					if amount == 500 {
						tier = "monthly"
					} else if amount == 3000 {
						tier = "yearly"
					}
				}

				// If we couldn't determine the tier, check the metadata
				if tier == "" && subscription.Metadata != nil {
					if t, ok := subscription.Metadata["tier"]; ok {
						tier = t
					}
				}

				// If we still couldn't determine the tier, default to monthly
				if tier == "" {
					tier = "monthly"
				}

				// Update the user's subscription information
				user.SubscriptionTier = tier
				user.SubscriptionStatus = "active"
				user.StripeSubscriptionID = subscription.ID

				// Set subscription end date if available
				if subscription.CurrentPeriodEnd > 0 {
					endDate := time.Unix(subscription.CurrentPeriodEnd, 0)
					user.SubscriptionEndDate = endDate
				}

				// Update the user in the database
				if err := s.db.UpdateUser(nil, user); err != nil {
					return err
				}
			}
		}

	case "invoice.payment_succeeded":
		// Handle successful subscription payments
		var invoice stripe.Invoice
		err := json.Unmarshal(event.Data.Raw, &invoice)
		if err != nil {
			return err
		}

		// Only process subscription invoices
		if invoice.Subscription == nil {
			return nil
		}

		// Get the customer ID
		customerID := invoice.Customer.ID

		// Find the user by Stripe customer ID
		user, err := s.db.GetUserByStripeCustomerID(customerID)
		if err != nil {
			return err
		}

		if user == nil {
			return fmt.Errorf("user not found for Stripe customer ID: %s", customerID)
		}

		// Create a payment record
		payment := &models.Payment{
			UserID:      user.ID,
			Amount:      invoice.AmountPaid,
			Currency:    string(invoice.Currency),
			PaymentType: "subscription",
			Status:      "succeeded",
			Description: "Subscription payment",
			StripeID:    invoice.ID,
		}

		// Save the payment to the database
		if err := s.db.CreatePayment(payment); err != nil {
			return err
		}

		// Update the user's subscription information
		subscription, err := s.GetSubscriptionDetails(invoice.Subscription.ID)
		if err != nil {
			return err
		}

		// Determine the subscription tier from the product
		var tier string
		if subscription.Items != nil && len(subscription.Items.Data) > 0 {
			priceID := subscription.Items.Data[0].Price.ID
			// Check for price ID patterns or use metadata
			if strings.Contains(priceID, "monthly") {
				tier = "monthly"
			} else if strings.Contains(priceID, "yearly") {
				tier = "yearly"
			} else {
				// Try to get tier from metadata
				if subscription.Metadata != nil {
					if t, ok := subscription.Metadata["tier"]; ok {
						tier = t
					}
				}

				// If still not set, check the amount
				if tier == "" {
					amount := subscription.Items.Data[0].Price.UnitAmount
					if amount == 500 {
						tier = "monthly"
					} else if amount == 3000 {
						tier = "yearly"
					}
				}
			}
		}

		// If we still couldn't determine the tier, default to monthly
		if tier == "" {
			tier = "monthly"
		}

		user.SubscriptionTier = tier
		user.SubscriptionStatus = "active"
		user.StripeSubscriptionID = subscription.ID

		// Set subscription end date if available
		if subscription.CurrentPeriodEnd > 0 {
			endDate := time.Unix(subscription.CurrentPeriodEnd, 0)
			user.SubscriptionEndDate = endDate
		}

		// Update the user in the database
		if err := s.db.UpdateUser(nil, user); err != nil {
			return err
		}

	case "customer.subscription.updated":
		// Handle subscription updates
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			return err
		}

		// Update the user's subscription information
		fmt.Printf("Subscription %s updated\n", subscription.ID)

		// Find the user by Stripe customer ID
		user, err := s.db.GetUserByStripeCustomerID(subscription.Customer.ID)
		if err != nil {
			return err
		}

		if user == nil {
			return fmt.Errorf("user not found for Stripe customer ID: %s", subscription.Customer.ID)
		}

		// Update the user's subscription information
		user.StripeSubscriptionID = subscription.ID
		user.SubscriptionStatus = string(subscription.Status)

		// Set subscription end date if available
		if subscription.CurrentPeriodEnd > 0 {
			endDate := time.Unix(subscription.CurrentPeriodEnd, 0)
			user.SubscriptionEndDate = endDate
		}

		// Update the user in the database
		if err := s.db.UpdateUser(nil, user); err != nil {
			return err
		}

	case "customer.subscription.deleted":
		// Handle subscription cancellations
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			return err
		}

		// Find the user by Stripe customer ID
		user, err := s.db.GetUserByStripeCustomerID(subscription.Customer.ID)
		if err != nil {
			return err
		}

		if user == nil {
			return fmt.Errorf("user not found for Stripe customer ID: %s", subscription.Customer.ID)
		}

		// Update the user's subscription information
		user.SubscriptionStatus = "canceled"

		// Update the user in the database
		if err := s.db.UpdateUser(nil, user); err != nil {
			return err
		}

		fmt.Printf("Subscription %s cancelled\n", subscription.ID)
	}

	return nil
}

// GetSubscriptionDetails gets details about a subscription
func (s *service) GetSubscriptionDetails(subscriptionID string) (*stripe.Subscription, error) {
	subscription, err := sub.Get(subscriptionID, nil)
	if err != nil {
		return nil, err
	}

	return subscription, nil
}

// CancelSubscription cancels a subscription
func (s *service) CancelSubscription(subscriptionID string) error {
	_, err := sub.Cancel(subscriptionID, nil)
	return err
}

// getOrCreateCustomer creates a new Stripe customer or returns an existing one
func (s *service) getOrCreateCustomer(user *database.User) (string, error) {
	// If the user already has a Stripe customer ID, return it
	if user.StripeCustomerID != "" {
		return user.StripeCustomerID, nil
	}

	// Create a new Stripe customer
	params := &stripe.CustomerParams{
		Email: stripe.String(user.Email),
	}

	// Add metadata
	params.AddMetadata("user_id", strconv.FormatUint(uint64(user.ID), 10))

	customer, err := customer.New(params)
	if err != nil {
		return "", err
	}

	// Update the user with the new Stripe customer ID
	user.StripeCustomerID = customer.ID
	if err := s.db.UpdateUser(nil, user); err != nil {
		return "", err
	}

	return customer.ID, nil
}

// createPrice creates a new price for a product
func (s *service) createPrice(productID, tier string) (string, error) {
	var amount int64
	var interval string
	var intervalCount int64

	// Set the price based on the tier
	switch tier {
	case "monthly":
		amount = 500 // $5.00
		interval = string(stripe.PriceRecurringIntervalMonth)
		intervalCount = 1
	case "yearly":
		amount = 3000 // $30.00
		interval = string(stripe.PriceRecurringIntervalYear)
		intervalCount = 1
	case "lifetime":
		amount = 10000 // $100.00
		interval = ""
		intervalCount = 0
	case "premium_lifetime":
		amount = 100000 // $1,000.00
		interval = ""
		intervalCount = 0
	default:
		return "", fmt.Errorf("invalid subscription tier: %s", tier)
	}

	// Create price parameters
	params := &stripe.PriceParams{
		Product:    stripe.String(productID),
		UnitAmount: stripe.Int64(amount),
		Currency:   stripe.String("usd"),
	}

	// Add metadata about the tier
	params.AddMetadata("tier", tier)

	// Add recurring parameters for subscription tiers
	if tier == "monthly" || tier == "yearly" {
		params.Recurring = &stripe.PriceRecurringParams{
			Interval:      stripe.String(interval),
			IntervalCount: stripe.Int64(intervalCount),
		}
	}

	// Create the price
	p, err := price.New(params)
	if err != nil {
		return "", err
	}

	return p.ID, nil
}

// getProductIDForTier returns the Stripe product ID for a subscription tier
func getProductIDForTier(tier string) (string, error) {
	var envVar string

	switch tier {
	case "monthly":
		envVar = "STRIPE_PRICE_MONTHLY"
	case "yearly":
		envVar = "STRIPE_PRICE_YEARLY"
	case "lifetime":
		envVar = "STRIPE_PRICE_LIFETIME"
	case "premium_lifetime":
		envVar = "STRIPE_PRICE_PREMIUM_LIFETIME"
	default:
		return "", fmt.Errorf("invalid subscription tier: %s", tier)
	}

	productID := os.Getenv(envVar)
	if productID == "" {
		return "", fmt.Errorf("product ID for tier %s is not set", tier)
	}

	return productID, nil
}
