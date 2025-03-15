package models

import (
	"fmt"

	"gorm.io/gorm"
)

// Payment represents a payment made by a user
type Payment struct {
	gorm.Model
	UserID      uint
	User        User  `gorm:"foreignKey:UserID"`
	Amount      int64 // Amount in cents
	Currency    string
	PaymentType string // "subscription", "one-time", etc.
	Status      string // "succeeded", "failed", "pending", etc.
	Description string
	StripeID    string // Stripe payment intent ID
}

// FormatAmount formats the amount as a string with the currency symbol
func (p *Payment) FormatAmount() string {
	// Convert cents to dollars
	dollars := float64(p.Amount) / 100.0

	// Format based on currency
	switch p.Currency {
	case "usd":
		return "$" + formatDollars(dollars)
	case "eur":
		return "€" + formatDollars(dollars)
	case "gbp":
		return "£" + formatDollars(dollars)
	default:
		return formatDollars(dollars) + " " + p.Currency
	}
}

// formatDollars formats a float as a string with 2 decimal places
func formatDollars(amount float64) string {
	return fmt.Sprintf("%.2f", amount)
}

// GetPaymentsByUserID retrieves all payments for a user
func GetPaymentsByUserID(db *gorm.DB, userID uint) ([]Payment, error) {
	var payments []Payment
	if err := db.Where("user_id = ?", userID).Order("created_at desc").Find(&payments).Error; err != nil {
		return nil, err
	}
	return payments, nil
}

// CreatePayment creates a new payment record
func CreatePayment(db *gorm.DB, payment *Payment) error {
	return db.Create(payment).Error
}

// sprintf is a helper function to format strings
func sprintf(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

// FindPaymentByID retrieves a payment by its ID
func FindPaymentByID(db *gorm.DB, id uint) (*Payment, error) {
	var payment Payment
	if err := db.First(&payment, id).Error; err != nil {
		return nil, err
	}
	return &payment, nil
}

// UpdatePayment updates an existing payment in the database
func UpdatePayment(db *gorm.DB, payment *Payment) error {
	return db.Save(payment).Error
}
