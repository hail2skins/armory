package database

import (
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

// CreatePayment creates a new payment record
func (s *service) CreatePayment(payment *Payment) error {
	return s.db.Create(payment).Error
}

// GetPaymentsByUserID retrieves all payments for a user
func (s *service) GetPaymentsByUserID(userID uint) ([]Payment, error) {
	var payments []Payment
	if err := s.db.Where("user_id = ?", userID).Order("created_at desc").Find(&payments).Error; err != nil {
		return nil, err
	}
	return payments, nil
}

// FindPaymentByID retrieves a payment by its ID
func (s *service) FindPaymentByID(id uint) (*Payment, error) {
	var payment Payment
	if err := s.db.First(&payment, id).Error; err != nil {
		return nil, err
	}
	return &payment, nil
}

// UpdatePayment updates an existing payment in the database
func (s *service) UpdatePayment(payment *Payment) error {
	return s.db.Save(payment).Error
}
