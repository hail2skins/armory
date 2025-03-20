package models

import "time"

// User represents a user in the system
type User interface {
	// GetUserName returns the user's email or username
	GetUserName() string

	// GetID returns the user's ID
	GetID() uint

	// GetCreatedAt returns when the user was created
	GetCreatedAt() time.Time

	// GetLastLogin returns when the user last logged in
	GetLastLogin() time.Time

	// GetSubscriptionTier returns the user's subscription tier
	GetSubscriptionTier() string

	// IsDeleted returns whether the user is deleted
	IsDeleted() bool

	// IsVerified returns whether the user's email is verified
	IsVerified() bool

	// GetSubscriptionStatus returns the user's subscription status
	GetSubscriptionStatus() string

	// GetSubscriptionEndDate returns when the user's subscription ends
	GetSubscriptionEndDate() time.Time
}
