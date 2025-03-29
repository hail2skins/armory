package seed

import (
	"log"

	"github.com/hail2skins/armory/internal/models" // Adjust import path if needed
	"gorm.io/gorm"
)

// SeedCasings seeds the database with common ammunition casing types
func SeedCasings(db *gorm.DB) {
	// Define common casing types
	casings := []models.Casing{
		// Catch-all option
		{Type: "Other", Popularity: 999}, // Highest popularity for default/catch-all

		// Most common casing types
		{Type: "Brass", Popularity: 100},              // Very common, reloadable
		{Type: "Steel", Popularity: 80},               // Common, especially certain calibers/imports, generally not reloadable
		{Type: "Nickel-Plated Brass", Popularity: 70}, // Common in defensive/premium ammo, looks silver, reloadable
		{Type: "Aluminum", Popularity: 50},            // Less common, not reloadable (e.g., CCI Blazer)

		// Less common/niche types
		{Type: "Polymer", Popularity: 20}, // Relatively new/niche
		// {Type: "Plastic", Popularity: 10}, // Primarily for shotgun shells, might be handled differently
	}

	// Loop through each casing type
	for _, cs := range casings {
		var count int64
		// Check if the record exists (by Type)
		if err := db.Model(&models.Casing{}).Where("type = ?", cs.Type).Count(&count).Error; err != nil {
			log.Printf("Error checking casing type %s: %v", cs.Type, err)
			continue // Skip to next casing on error
		}

		if count == 0 {
			// Casing type does not exist, create it
			if err := db.Create(&cs).Error; err != nil {
				log.Printf("Error seeding casing type %s: %v", cs.Type, err)
			} else {
				log.Printf("Seeded casing type: %s", cs.Type)
			}
		} else {
			// Casing type exists, update its popularity
			// Ensure we only update popularity, not other fields unintentionally
			if err := db.Model(&models.Casing{}).Where("type = ?", cs.Type).Update("popularity", cs.Popularity).Error; err != nil {
				log.Printf("Error updating popularity for casing type %s: %v", cs.Type, err)
			} else {
				log.Printf("Updated popularity for casing type: %s", cs.Type)
			}
		}
	}
}
