package stripe

import (
	"testing"
	"time"

	"github.com/hail2skins/armory/internal/database"
	"github.com/stretchr/testify/assert"
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
