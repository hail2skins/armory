package data

import (
	"testing"

	"github.com/hail2skins/armory/internal/models"
	"github.com/stretchr/testify/assert"
)

// TestOwnerDataWithTotalPaid tests that the owner data struct can track total paid amounts
func TestOwnerDataWithTotalPaid(t *testing.T) {
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

	// Create owner data
	ownerData := NewOwnerData().
		WithTitle("Test").
		WithGuns(guns)

	// Test that WithTotalPaid method correctly sets the total paid amount
	totalPaid := 1500.50 // $1000.00 + $500.50
	ownerData = ownerData.WithTotalPaid(totalPaid)

	// Verify the total paid is correct
	assert.Equal(t, 1500.50, ownerData.TotalPaid)
}
