package seed

import (
	"log"

	"github.com/hail2skins/armory/internal/models" // Adjust import path if needed
	"gorm.io/gorm"
)

// SeedGrains seeds the database with common ammunition grain weights
func SeedGrains(db *gorm.DB) {
	// Define common grain weights with popularity
	// Popularity is subjective and based on commonality across various calibers
	grains := []models.Grain{
		// Catch-all option - Using 0 or -1 might be better if 'Other' isn't a weight
		// Let's use 0 for 'Other' or 'Not Specified'
		{Weight: 0, Popularity: 999}, // Highest popularity for default/unspecified

		// Common Pistol Grains (9mm, .40 S&W, .45 ACP)
		{Weight: 115, Popularity: 100}, // Very common 9mm
		{Weight: 124, Popularity: 95},  // Common 9mm
		{Weight: 147, Popularity: 90},  // Common 9mm (subsonic/defensive)
		{Weight: 180, Popularity: 85},  // Common .40 S&W
		{Weight: 165, Popularity: 80},  // Common .40 S&W
		{Weight: 230, Popularity: 90},  // Very common .45 ACP
		{Weight: 185, Popularity: 75},  // Common .45 ACP

		// Common Rifle Grains (5.56 NATO / .223 Rem, 7.62x39, .308 Win)
		{Weight: 55, Popularity: 100}, // Very common 5.56/.223
		{Weight: 62, Popularity: 95},  // Common 5.56/.223 (M855/SS109)
		{Weight: 77, Popularity: 80},  // Common 5.56/.223 (Mk262)
		{Weight: 123, Popularity: 90}, // Common 7.62x39
		{Weight: 150, Popularity: 95}, // Common .308 Win / 7.62 NATO
		{Weight: 168, Popularity: 85}, // Common .308 Win (match)
		{Weight: 175, Popularity: 80}, // Common .308 Win (match/long range)

		// Common Rimfire Grains (.22 LR)
		{Weight: 40, Popularity: 90}, // Very common .22 LR
		{Weight: 36, Popularity: 85}, // Common .22 LR

		// Other examples
		{Weight: 158, Popularity: 70}, // Common .38 Special / .357 Magnum
		{Weight: 240, Popularity: 65}, // Common .44 Magnum
		{Weight: 75, Popularity: 60},  // Heavier 5.56/.223
		{Weight: 90, Popularity: 55},  // Common 6.8 SPC
		{Weight: 140, Popularity: 50}, // Common 6.5 Creedmoor
		{Weight: 300, Popularity: 45}, // Common .300 Blackout (subsonic)
		{Weight: 110, Popularity: 40}, // Common .300 Blackout (supersonic)

		// Add more common weights as needed...
	}

	// Loop through each grain weight
	for _, g := range grains {
		var count int64
		// Check if the record exists (by Weight)
		if err := db.Model(&models.Grain{}).Where("weight = ?", g.Weight).Count(&count).Error; err != nil {
			log.Printf("Error checking grain weight %d: %v", g.Weight, err)
			continue // Skip to next grain on error
		}

		if count == 0 {
			// Grain weight does not exist, create it
			if err := db.Create(&g).Error; err != nil {
				log.Printf("Error seeding grain weight %d: %v", g.Weight, err)
			} else {
				log.Printf("Seeded grain weight: %d", g.Weight)
			}
		} else {
			// Grain weight exists, update its popularity
			if err := db.Model(&models.Grain{}).Where("weight = ?", g.Weight).Update("popularity", g.Popularity).Error; err != nil {
				log.Printf("Error updating popularity for grain weight %d: %v", g.Weight, err)
			} else {
				log.Printf("Updated popularity for grain weight: %d", g.Weight)
			}
		}
	}
}
