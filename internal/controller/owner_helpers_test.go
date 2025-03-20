package controller

import (
	"testing"

	"github.com/hail2skins/armory/internal/models"
	"github.com/stretchr/testify/assert"
)

// TestCalculateTotalPaid tests the calculateTotalPaid function
func TestCalculateTotalPaid(t *testing.T) {
	// Create test guns with paid values
	paidAmount1 := 1000.00
	gun1 := models.Gun{
		Name: "Test Gun 1",
		Paid: &paidAmount1,
	}

	paidAmount2 := 500.50
	gun2 := models.Gun{
		Name: "Test Gun 2",
		Paid: &paidAmount2,
	}

	// Create a gun with no paid value
	gun3 := models.Gun{
		Name: "Test Gun 3",
		Paid: nil,
	}

	// Create guns array
	guns := []models.Gun{gun1, gun2, gun3}

	// Call the function
	totalPaid := calculateTotalPaid(guns)

	// Verify the total is correct (1000.00 + 500.50 = 1500.50)
	assert.Equal(t, 1500.50, totalPaid)

	// Test with empty slice
	emptyGuns := []models.Gun{}
	emptyTotal := calculateTotalPaid(emptyGuns)
	assert.Equal(t, 0.0, emptyTotal)

	// Test with all nil values
	nilGun1 := models.Gun{Name: "Nil Gun 1", Paid: nil}
	nilGun2 := models.Gun{Name: "Nil Gun 2", Paid: nil}
	nilGuns := []models.Gun{nilGun1, nilGun2}
	nilTotal := calculateTotalPaid(nilGuns)
	assert.Equal(t, 0.0, nilTotal)
}
