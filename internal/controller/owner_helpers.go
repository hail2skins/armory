package controller

import (
	"github.com/hail2skins/armory/internal/models"
)

// calculateTotalPaid calculates the total amount paid for all guns
func calculateTotalPaid(guns []models.Gun) float64 {
	var total float64
	for _, gun := range guns {
		if gun.Paid != nil {
			total += *gun.Paid
		}
	}
	return total
}
