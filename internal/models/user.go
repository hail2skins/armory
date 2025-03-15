package models

// User represents a user in the system
type User interface {
	// GetUserName returns the user's email or username
	GetUserName() string

	// GetID returns the user's ID
	GetID() uint
}
