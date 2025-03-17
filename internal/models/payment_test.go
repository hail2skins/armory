package models_test

import (
	"testing"

	"github.com/hail2skins/armory/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates a new in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	// Migrate the schema for Payment
	err = db.AutoMigrate(&models.Payment{})
	assert.NoError(t, err)

	return db
}

// TestCreatePayment tests the CreatePayment function
func TestCreatePayment(t *testing.T) {
	// Setup test database
	db := setupTestDB(t)

	// Create a test payment
	payment := &models.Payment{
		UserID:      1,
		Amount:      1999, // $19.99
		Currency:    "usd",
		PaymentType: "subscription",
		Status:      "succeeded",
		Description: "Test subscription payment",
		StripeID:    "pi_test_123456",
	}

	// Call the function being tested
	err := models.CreatePayment(db, payment)
	assert.NoError(t, err)
	assert.NotZero(t, payment.ID)

	// Verify the payment was created
	var foundPayment models.Payment
	err = db.First(&foundPayment, payment.ID).Error
	assert.NoError(t, err)

	// Verify payment details
	assert.Equal(t, uint(1), foundPayment.UserID)
	assert.Equal(t, int64(1999), foundPayment.Amount)
	assert.Equal(t, "usd", foundPayment.Currency)
	assert.Equal(t, "subscription", foundPayment.PaymentType)
	assert.Equal(t, "succeeded", foundPayment.Status)
	assert.Equal(t, "Test subscription payment", foundPayment.Description)
	assert.Equal(t, "pi_test_123456", foundPayment.StripeID)
}

// TestGetPaymentsByUserID tests the GetPaymentsByUserID function
func TestGetPaymentsByUserID(t *testing.T) {
	// Setup test database
	db := setupTestDB(t)

	// Clear existing payments for user 1 (if any)
	db.Where("user_id = ?", 1).Delete(&models.Payment{})

	// Create multiple test payments for user 1
	payment1 := &models.Payment{
		UserID:      1,
		Amount:      1999,
		Currency:    "usd",
		PaymentType: "subscription",
		Status:      "succeeded",
		Description: "First test payment",
		StripeID:    "pi_test_1",
	}
	err := db.Create(payment1).Error
	assert.NoError(t, err)

	payment2 := &models.Payment{
		UserID:      1,
		Amount:      2999,
		Currency:    "usd",
		PaymentType: "one-time",
		Status:      "succeeded",
		Description: "Second test payment",
		StripeID:    "pi_test_2",
	}
	err = db.Create(payment2).Error
	assert.NoError(t, err)

	// Create a payment for a different user
	payment3 := &models.Payment{
		UserID:      2,
		Amount:      999,
		Currency:    "eur",
		PaymentType: "one-time",
		Status:      "succeeded",
		Description: "Other user payment",
		StripeID:    "pi_test_3",
	}
	err = db.Create(payment3).Error
	assert.NoError(t, err)

	// Call the function being tested
	payments, err := models.GetPaymentsByUserID(db, 1)
	assert.NoError(t, err)

	// Check that we got the correct number of payments
	assert.Equal(t, 2, len(payments))

	// Verify all returned payments are for user 1
	for _, p := range payments {
		assert.Equal(t, uint(1), p.UserID)
	}

	// Test for user 2
	payments, err = models.GetPaymentsByUserID(db, 2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(payments))
	assert.Equal(t, uint(2), payments[0].UserID)
	assert.Equal(t, "eur", payments[0].Currency)
}

// TestFindPaymentByID tests the FindPaymentByID function
func TestFindPaymentByID(t *testing.T) {
	// Setup test database
	db := setupTestDB(t)

	// Create a test payment
	payment := &models.Payment{
		UserID:      1,
		Amount:      1999,
		Currency:    "usd",
		PaymentType: "subscription",
		Status:      "succeeded",
		Description: "Test payment for lookup",
		StripeID:    "pi_test_lookup",
	}
	err := db.Create(payment).Error
	assert.NoError(t, err)

	// Call the function being tested
	foundPayment, err := models.FindPaymentByID(db, payment.ID)
	assert.NoError(t, err)
	assert.NotNil(t, foundPayment)

	// Verify payment details
	assert.Equal(t, payment.ID, foundPayment.ID)
	assert.Equal(t, uint(1), foundPayment.UserID)
	assert.Equal(t, int64(1999), foundPayment.Amount)
	assert.Equal(t, "usd", foundPayment.Currency)
	assert.Equal(t, "subscription", foundPayment.PaymentType)
	assert.Equal(t, "succeeded", foundPayment.Status)
	assert.Equal(t, "Test payment for lookup", foundPayment.Description)
	assert.Equal(t, "pi_test_lookup", foundPayment.StripeID)

	// Test with non-existent ID
	foundPayment, err = models.FindPaymentByID(db, 9999)
	assert.Error(t, err)
	assert.Nil(t, foundPayment)
}

// TestUpdatePayment tests the UpdatePayment function
func TestUpdatePayment(t *testing.T) {
	// Setup test database
	db := setupTestDB(t)

	// Create a test payment
	payment := &models.Payment{
		UserID:      1,
		Amount:      1999,
		Currency:    "usd",
		PaymentType: "subscription",
		Status:      "pending",
		Description: "Test payment for update",
		StripeID:    "pi_test_update",
	}
	err := db.Create(payment).Error
	assert.NoError(t, err)

	// Update the payment
	payment.Status = "succeeded"
	payment.Description = "Updated test payment"

	// Call the function being tested
	err = models.UpdatePayment(db, payment)
	assert.NoError(t, err)

	// Retrieve the payment again to verify updates
	var updatedPayment models.Payment
	err = db.First(&updatedPayment, payment.ID).Error
	assert.NoError(t, err)

	// Verify the updated fields
	assert.Equal(t, "succeeded", updatedPayment.Status)
	assert.Equal(t, "Updated test payment", updatedPayment.Description)

	// Verify other fields remain unchanged
	assert.Equal(t, uint(1), updatedPayment.UserID)
	assert.Equal(t, int64(1999), updatedPayment.Amount)
	assert.Equal(t, "usd", updatedPayment.Currency)
	assert.Equal(t, "subscription", updatedPayment.PaymentType)
	assert.Equal(t, "pi_test_update", updatedPayment.StripeID)
}

// TestPaymentFormatAmount tests the FormatAmount method
func TestPaymentFormatAmount(t *testing.T) {
	testCases := []struct {
		payment  models.Payment
		expected string
	}{
		{
			payment: models.Payment{
				Amount:   1999,
				Currency: "usd",
			},
			expected: "$19.99",
		},
		{
			payment: models.Payment{
				Amount:   2050,
				Currency: "eur",
			},
			expected: "€20.50",
		},
		{
			payment: models.Payment{
				Amount:   1000,
				Currency: "gbp",
			},
			expected: "£10.00",
		},
		{
			payment: models.Payment{
				Amount:   5000,
				Currency: "jpy", // Not one of the recognized currencies
			},
			expected: "50.00 jpy",
		},
	}

	for _, tc := range testCases {
		result := tc.payment.FormatAmount()
		assert.Equal(t, tc.expected, result)
	}
}
