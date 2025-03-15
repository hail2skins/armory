package seed

import (
	"log"

	"github.com/hail2skins/armory/internal/models"
	"gorm.io/gorm"
)

// SeedWeaponTypes seeds the database with common weapon types
func SeedWeaponTypes(db *gorm.DB) {
	// Define common weapon types
	weaponTypes := []models.WeaponType{
		// Catch-all option
		{Type: "Other", Nickname: "Other", Popularity: 999},

		// Most popular weapon types with high popularity values
		{Type: "Handgun", Nickname: "Pistol", Popularity: 100},
		{Type: "Semi-Automatic Rifle", Nickname: "AR", Popularity: 90},
		{Type: "Shotgun", Nickname: "Shotgun", Popularity: 85},
		{Type: "Revolver", Nickname: "Revolver", Popularity: 80},
		{Type: "Rifle", Nickname: "Rifle", Popularity: 75},

		// Medium popularity weapon types
		{Type: "Carbine", Nickname: "Carbine", Popularity: 60},
		{Type: "Bolt-Action Rifle", Nickname: "Bolt Rifle", Popularity: 55},
		{Type: "Semi-Automatic Shotgun", Nickname: "Semi-Auto Shotgun", Popularity: 50},
		{Type: "Pump-Action Shotgun", Nickname: "Pump Shotgun", Popularity: 45},
		{Type: "Lever-Action Rifle", Nickname: "Lever Rifle", Popularity: 40},

		// Less common weapon types with lower popularity
		{Type: "Sniper Rifle", Nickname: "Sniper", Popularity: 35},
		{Type: "Designated Marksman Rifle", Nickname: "DMR", Popularity: 30},
		{Type: "Submachine Gun", Nickname: "SMG", Popularity: 25},
		{Type: "Personal Defense Weapon", Nickname: "PDW", Popularity: 20},
		{Type: "Machine Gun", Nickname: "MG", Popularity: 15},
		{Type: "Anti-Materiel Rifle", Nickname: "AMR", Popularity: 10},
		{Type: "Battle Rifle", Nickname: "Battle Rifle", Popularity: 25},
		{Type: "Precision Rifle", Nickname: "Precision Rifle", Popularity: 30},
	}

	// Loop through each weapon type
	for _, wt := range weaponTypes {
		var count int64
		// Check if the record exists (by Type)
		if err := db.Model(&models.WeaponType{}).Where("type = ?", wt.Type).Count(&count).Error; err != nil {
			log.Printf("Error checking weapon type %s: %v", wt.Type, err)
			continue
		}
		if count == 0 {
			if err := db.Create(&wt).Error; err != nil {
				log.Printf("Error seeding weapon type %s: %v", wt.Type, err)
			} else {
				log.Printf("Seeded weapon type: %s", wt.Type)
			}
		} else {
			// Update the popularity for existing weapon types
			if err := db.Model(&models.WeaponType{}).Where("type = ?", wt.Type).Update("popularity", wt.Popularity).Error; err != nil {
				log.Printf("Error updating popularity for weapon type %s: %v", wt.Type, err)
			} else {
				log.Printf("Updated popularity for weapon type: %s", wt.Type)
			}
		}
	}
}
