package seed

import (
	"log"

	"github.com/hail2skins/armory/internal/models"
	"gorm.io/gorm"
)

// SeedCalibers seeds the database with common calibers
func SeedCalibers(db *gorm.DB) {
	// Define common calibers
	calibers := []models.Caliber{
		// Catch-all option
		{Caliber: "Other", Nickname: "Other", Popularity: 999},

		// Most popular calibers with high popularity values
		{Caliber: "9mm Parabellum", Nickname: "9", Popularity: 100},
		{Caliber: "45 ACP", Nickname: "45", Popularity: 90},
		{Caliber: "22 Long Rifle", Nickname: "22 LR", Popularity: 85},
		{Caliber: "12 Gauge", Nickname: "12", Popularity: 80},
		{Caliber: "5.56×45mm NATO", Nickname: "5.56", Popularity: 75},
		{Caliber: "308 Winchester", Nickname: "308", Popularity: 70},
		{Caliber: "38 Special", Nickname: "38", Popularity: 65},
		{Caliber: "357 Magnum", Nickname: "357", Popularity: 60},
		{Caliber: "40 S&W", Nickname: "40", Popularity: 55},
		{Caliber: "380 ACP", Nickname: "380", Popularity: 50},

		// Less common calibers with lower popularity values
		{Caliber: "22 Magnum", Nickname: "22 Mag", Popularity: 30},
		{Caliber: "25 ACP", Nickname: "25 ACP", Popularity: 20},
		{Caliber: "32 ACP", Nickname: "32 ACP", Popularity: 20},
		{Caliber: "32 S&W", Nickname: "32 S&W", Popularity: 15},
		{Caliber: "9×19mm", Nickname: "9", Popularity: 40},
		{Caliber: "44 Special", Nickname: "44", Popularity: 25},
		{Caliber: "44 Magnum", Nickname: "44 Mag", Popularity: 35},
		{Caliber: "50 AE", Nickname: "50 AE", Popularity: 15},

		// Common rifle calibers with medium popularity
		{Caliber: "223 Remington", Nickname: "223", Popularity: 45},
		{Caliber: "22-250 Remington", Nickname: "22-250", Popularity: 20},
		{Caliber: "243 Winchester", Nickname: "243", Popularity: 30},
		{Caliber: "270 Winchester", Nickname: "270", Popularity: 35},
		{Caliber: "30-06 Springfield", Nickname: "30-06", Popularity: 40},
		{Caliber: "300 Winchester Magnum", Nickname: "300 WM", Popularity: 25},
		{Caliber: "6.5 Creedmoor", Nickname: "6.5", Popularity: 45},

		// Intermediate and less common rifle rounds with lower popularity
		{Caliber: "7.62×39mm", Nickname: "7.62", Popularity: 40},
		{Caliber: "7.62×51mm NATO", Nickname: "7.62 NATO", Popularity: 35},
		{Caliber: "7.62×54mm R", Nickname: "7.62 R", Popularity: 15},
		{Caliber: "300 AAC Blackout", Nickname: "300 BLK", Popularity: 30},
		{Caliber: "6.8 SPC", Nickname: "6.8 SPC", Popularity: 15},
		{Caliber: "6mm Creedmoor", Nickname: "6 Creedmoor", Popularity: 15},

		// Big bore and magnum calibers with lower popularity
		{Caliber: "338 Lapua Magnum", Nickname: "338 Lapua", Popularity: 15},
		{Caliber: "375 H&H Magnum", Nickname: "375 H&H", Popularity: 10},
		{Caliber: "458 Winchester Magnum", Nickname: "458 WM", Popularity: 10},
		{Caliber: "416 Rigby", Nickname: "416 Rigby", Popularity: 10},
		{Caliber: "500 S&W Magnum", Nickname: "500 S&W", Popularity: 15},
		{Caliber: "338 Federal", Nickname: "338 Fed", Popularity: 10},

		// Shotgun gauges with varying popularity
		{Caliber: "20 Gauge", Nickname: "20", Popularity: 40},
		{Caliber: "28 Gauge", Nickname: "28", Popularity: 15},
		{Caliber: "410 Bore", Nickname: "410", Popularity: 25},
		{Caliber: "10 Gauge", Nickname: "10", Popularity: 15},
		{Caliber: "16 Gauge", Nickname: "16", Popularity: 15},
	}

	// Insert calibers into the database
	for _, caliber := range calibers {
		var count int64
		if err := db.Model(&models.Caliber{}).Where("caliber = ?", caliber.Caliber).Count(&count).Error; err != nil {
			log.Printf("Error checking caliber %s: %v", caliber.Caliber, err)
			continue
		}

		if count == 0 {
			if err := db.Create(&caliber).Error; err != nil {
				log.Printf("Error seeding caliber %s: %v", caliber.Caliber, err)
			} else {
				log.Printf("Seeded caliber: %s", caliber.Caliber)
			}
		} else {
			// Update the popularity for existing calibers
			if err := db.Model(&models.Caliber{}).Where("caliber = ?", caliber.Caliber).Update("popularity", caliber.Popularity).Error; err != nil {
				log.Printf("Error updating popularity for caliber %s: %v", caliber.Caliber, err)
			} else {
				log.Printf("Updated popularity for caliber: %s", caliber.Caliber)
			}
		}
	}
}
