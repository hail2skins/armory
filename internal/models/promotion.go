package models

import (
	"time"

	"gorm.io/gorm"
)

// Promotion represents a marketing promotion in the system
type Promotion struct {
	gorm.Model
	Name          string    // Name of the promotion
	Type          string    // "free_trial", "discount", etc.
	Active        bool      // Whether the promotion is currently active
	StartDate     time.Time // When the promotion starts
	EndDate       time.Time // When the promotion ends
	BenefitDays   int       // Duration of benefit (e.g., 30 days)
	DisplayOnHome bool      // Whether to display on the home page
	Description   string    // Marketing copy
	Banner        string    // Optional banner image path
}
