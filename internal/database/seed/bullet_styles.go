package seed

import (
	"log"

	"github.com/hail2skins/armory/internal/models" // Adjust import path if needed
	"gorm.io/gorm"
)

// SeedBulletStyles seeds the database with common bullet styles
func SeedBulletStyles(db *gorm.DB) {
	// Define common bullet styles with nicknames and popularity
	bulletStyles := []models.BulletStyle{
		// Catch-all option
		{Type: "Other", Nickname: "Other", Popularity: 999}, // Highest popularity

		// Most common types
		{Type: "Full Metal Jacket", Nickname: "FMJ", Popularity: 100},
		{Type: "Jacketed Hollow Point", Nickname: "JHP", Popularity: 95},
		{Type: "Soft Point", Nickname: "SP", Popularity: 85},
		{Type: "Ballistic Tip", Nickname: "BT", Popularity: 80}, // Often polymer tipped

		// Target/Specialty types
		{Type: "Wadcutter", Nickname: "WC", Popularity: 70},
		{Type: "Semi-Wadcutter", Nickname: "SWC", Popularity: 65},
		{Type: "Hollow Point Boat Tail", Nickname: "HPBT", Popularity: 60},       // Common for match/long range
		{Type: "Boat Tail Hollow Point", Nickname: "BTHP", Popularity: 60},       // Same as above, different naming convention
		{Type: "Full Metal Jacket Boat Tail", Nickname: "FMJBT", Popularity: 55}, // Common for rifle/long range FMJ
		{Type: "Flat Nose", Nickname: "FN", Popularity: 50},
		{Type: "Round Nose", Nickname: "RN", Popularity: 45},
		{Type: "Lead Round Nose", Nickname: "LRN", Popularity: 40}, // Common for practice/reloading

		// Less common / More specialized
		{Type: "Frangible", Nickname: "Frangible", Popularity: 35},
		{Type: "Tracer", Nickname: "Tracer", Popularity: 30},
		{Type: "Armor Piercing", Nickname: "AP", Popularity: 25},
		{Type: "Incendiary", Nickname: "Incendiary", Popularity: 20},
		{Type: "Solid Copper", Nickname: "Solid", Popularity: 15}, // Often used for hunting in lead-restricted areas
		{Type: "Plated", Nickname: "Plated", Popularity: 10},      // Lead core with thin copper plating
		{Type: "Slug", Nickname: "Slug", Popularity: 5},           // For shotguns
		// Add more as needed...
	}

	// Loop through each bullet style
	for _, bs := range bulletStyles {
		var count int64
		// Check if the record exists (by Type)
		if err := db.Model(&models.BulletStyle{}).Where("type = ?", bs.Type).Count(&count).Error; err != nil {
			log.Printf("Error checking bullet style %s: %v", bs.Type, err)
			continue // Skip to next style on error
		}

		if count == 0 {
			// Bullet style does not exist, create it
			if err := db.Create(&bs).Error; err != nil {
				log.Printf("Error seeding bullet style %s: %v", bs.Type, err)
			} else {
				log.Printf("Seeded bullet style: %s", bs.Type)
			}
		} else {
			// Bullet style exists, update its Nickname and Popularity
			// Using a map to update specific fields
			updates := map[string]interface{}{
				"nickname":   bs.Nickname,
				"popularity": bs.Popularity,
			}
			if err := db.Model(&models.BulletStyle{}).Where("type = ?", bs.Type).Updates(updates).Error; err != nil {
				log.Printf("Error updating bullet style %s: %v", bs.Type, err)
			} else {
				log.Printf("Updated bullet style: %s", bs.Type)
			}
		}
	}
}
