package stripe

import (
	"testing"
	"time"

	"github.com/hail2skins/armory/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stripe/stripe-go/v72"
)

func TestSetSubscriptionEndDate(t *testing.T) {
	// Test cases
	tests := []struct {
		name           string
		user           *database.User
		endTimestamp   int64
		newTier        string
		eventType      string
		expectedResult time.Time
	}{
		{
			name: "New subscription (no previous end date)",
			user: &database.User{
				SubscriptionEndDate: time.Time{}, // Zero time
			},
			endTimestamp:   time.Now().Add(30 * 24 * time.Hour).Unix(), // 30 days from now
			newTier:        "monthly",
			eventType:      "checkout.session.completed",
			expectedResult: time.Unix(time.Now().Add(30*24*time.Hour).Unix(), 0),
		},
		{
			name: "Upgrade from monthly to yearly",
			user: &database.User{
				SubscriptionEndDate: time.Now().Add(15 * 24 * time.Hour), // 15 days left on monthly plan
			},
			endTimestamp:   time.Now().Add(365 * 24 * time.Hour).Unix(), // Would be 365 days from now
			newTier:        "yearly",
			eventType:      "checkout.session.completed",
			expectedResult: time.Now().Add(15 * 24 * time.Hour).Add(365 * 24 * time.Hour), // Should be existing end date + 365 days
		},
		{
			name: "Upgrade from yearly to lifetime",
			user: &database.User{
				SubscriptionEndDate: time.Now().Add(180 * 24 * time.Hour), // 180 days left on yearly plan
			},
			endTimestamp:   time.Now().Add(20 * 365 * 24 * time.Hour).Unix(), // Would be 20 years from now
			newTier:        "lifetime",
			eventType:      "checkout.session.completed",
			expectedResult: time.Now().Add(180 * 24 * time.Hour).Add(20 * 365 * 24 * time.Hour), // Should be existing end date + 20 years
		},
		{
			name: "Wrong event type (should not update)",
			user: &database.User{
				SubscriptionEndDate: time.Now().Add(15 * 24 * time.Hour), // 15 days left
			},
			endTimestamp:   time.Now().Add(30 * 24 * time.Hour).Unix(), // 30 days from now
			newTier:        "monthly",
			eventType:      "invoice.payment_succeeded",         // Not checkout.session.completed
			expectedResult: time.Now().Add(15 * 24 * time.Hour), // Should remain unchanged
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Copy the user to avoid modifying the test case
			user := *tt.user

			// Call the function
			setSubscriptionEndDate(&user, tt.endTimestamp, tt.newTier, tt.eventType)

			// Check if the result is within 2 seconds of expected (to account for slight timing differences)
			diff := user.SubscriptionEndDate.Sub(tt.expectedResult)
			assert.LessOrEqual(t, diff.Abs(), 2*time.Second, "End date should be within 2 seconds of expected")
		})
	}
}

// MockStripeService is a mock implementation of the Service interface for testing
type MockStripeService struct {
	subscriptions map[string]*stripe.Subscription
}

// NewMockStripeService creates a new mock Stripe service for testing
func NewMockStripeService() *MockStripeService {
	return &MockStripeService{
		subscriptions: make(map[string]*stripe.Subscription),
	}
}

// CreateCheckoutSession is a mock implementation for testing
func (m *MockStripeService) CreateCheckoutSession(user *database.User, tier string) (*stripe.CheckoutSession, error) {
	return &stripe.CheckoutSession{
		ID: "mock_session_id",
	}, nil
}

// HandleWebhook is a mock implementation for testing
func (m *MockStripeService) HandleWebhook(payload []byte, signature string) error {
	return nil
}

// GetSubscriptionDetails is a mock implementation for testing
func (m *MockStripeService) GetSubscriptionDetails(subscriptionID string) (*stripe.Subscription, error) {
	if sub, ok := m.subscriptions[subscriptionID]; ok {
		return sub, nil
	}

	// Return a default subscription if not found
	return &stripe.Subscription{
		ID:                subscriptionID,
		Status:            stripe.SubscriptionStatusActive,
		CancelAtPeriodEnd: false,
		CurrentPeriodEnd:  1717027200, // Some future timestamp
	}, nil
}

// CancelSubscription is a mock implementation for testing
func (m *MockStripeService) CancelSubscription(subscriptionID string) error {
	// Get the current subscription or create a new one
	sub, _ := m.GetSubscriptionDetails(subscriptionID)

	// Update the subscription to be canceled at period end
	sub.CancelAtPeriodEnd = true

	// Store the updated subscription
	m.subscriptions[subscriptionID] = sub

	return nil
}

// CancelSubscriptionImmediately is a mock implementation for testing
func (m *MockStripeService) CancelSubscriptionImmediately(subscriptionID string) error {
	// Get the current subscription or create a new one
	sub, _ := m.GetSubscriptionDetails(subscriptionID)

	// Update the subscription to be immediately canceled
	sub.Status = stripe.SubscriptionStatusCanceled
	sub.CancelAtPeriodEnd = false

	// Store the updated subscription
	m.subscriptions[subscriptionID] = sub

	return nil
}

// TestCancelSubscription tests the cancellation of a subscription
func TestCancelSubscription(t *testing.T) {
	// Create a mock Stripe service
	mockService := NewMockStripeService()

	// Test subscription ID
	subscriptionID := "sub_test123"

	// Cancel the subscription at period end
	err := mockService.CancelSubscription(subscriptionID)
	assert.NoError(t, err)

	// Verify the subscription is marked for cancellation at period end
	sub, err := mockService.GetSubscriptionDetails(subscriptionID)
	assert.NoError(t, err)
	assert.Equal(t, stripe.SubscriptionStatusActive, sub.Status)
	assert.True(t, sub.CancelAtPeriodEnd)
}

// TestCancelSubscriptionImmediately tests the immediate cancellation of a subscription
func TestCancelSubscriptionImmediately(t *testing.T) {
	// Create a mock Stripe service
	mockService := NewMockStripeService()

	// Test subscription ID
	subscriptionID := "sub_test456"

	// Cancel the subscription immediately
	err := mockService.CancelSubscriptionImmediately(subscriptionID)
	assert.NoError(t, err)

	// Verify the subscription is canceled immediately
	sub, err := mockService.GetSubscriptionDetails(subscriptionID)
	assert.NoError(t, err)
	assert.Equal(t, stripe.SubscriptionStatusCanceled, sub.Status)
	assert.False(t, sub.CancelAtPeriodEnd)
}
