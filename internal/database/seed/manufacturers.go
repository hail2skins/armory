package seed

import (
	"log"

	"github.com/hail2skins/armory/internal/models"
	"gorm.io/gorm"
)

// SeedManufacturers is our seed function that checks and inserts manufacturers.
func SeedManufacturers(db *gorm.DB) {
	// List of manufacturers to seed.
	manufacturers := []models.Manufacturer{
		// Catch-all option
		{Name: "Other", Country: "Various", Nickname: "Other", Popularity: 999},

		// Most popular manufacturers with high popularity values
		{Name: "Glock", Country: "Austria", Nickname: "Glock", Popularity: 100},
		{Name: "Smith & Wesson", Country: "USA", Nickname: "S&W", Popularity: 95},
		{Name: "Sig Sauer", Country: "Germany/USA", Nickname: "Sig", Popularity: 90},
		{Name: "Colt's Manufacturing Company", Country: "USA", Nickname: "Colt", Popularity: 85},
		{Name: "Remington Arms", Country: "USA", Nickname: "Remington", Popularity: 80},
		{Name: "Winchester Repeating Arms", Country: "USA", Nickname: "Winchester", Popularity: 75},
		{Name: "Sturm, Ruger & Co.", Country: "USA", Nickname: "Ruger", Popularity: 70},
		{Name: "Beretta", Country: "Italy", Nickname: "Beretta", Popularity: 65},
		{Name: "Browning", Country: "USA", Nickname: "Browning", Popularity: 60},
		{Name: "Heckler & Koch", Country: "Germany", Nickname: "H&K", Popularity: 55},

		// Less popular but still common manufacturers
		{Name: "Taurus", Country: "Brazil/USA", Nickname: "Taurus", Popularity: 50},
		{Name: "Kimber Manufacturing", Country: "USA", Nickname: "Kimber", Popularity: 45},
		{Name: "Springfield Armory", Country: "USA", Nickname: "Springfield", Popularity: 45},
		{Name: "Barrett Firearms Manufacturing", Country: "USA", Nickname: "Barrett", Popularity: 40},
		{Name: "Bushmaster Firearms International", Country: "USA", Nickname: "Bushmaster", Popularity: 35},
		{Name: "Franklin Armory", Country: "USA", Nickname: "Franklin", Popularity: 30},
		{Name: "Accuracy International", Country: "UK", Nickname: "AI", Popularity: 30},

		// International Manufacturers with medium popularity
		{Name: "Česká zbrojovka (CZ)", Country: "Czech Republic", Nickname: "CZ", Popularity: 45},
		{Name: "FN Herstal", Country: "Belgium", Nickname: "FN", Popularity: 40},
		{Name: "Steyr Mannlicher", Country: "Austria", Nickname: "Steyr", Popularity: 35},
		{Name: "Walther", Country: "Germany", Nickname: "Walther", Popularity: 40},
		{Name: "IWI (Israel Weapon Industries)", Country: "Israel", Nickname: "IWI", Popularity: 35},

		// Smaller or Specialty Manufacturers with lower popularity
		{Name: "Kel-Tec", Country: "USA", Nickname: "Kel-Tec", Popularity: 30},
		{Name: "Rossi", Country: "USA/Brazil", Nickname: "Rossi", Popularity: 25},
		{Name: "Charter Arms", Country: "USA", Nickname: "Charter", Popularity: 20},
		{Name: "Uberti", Country: "Italy/USA", Nickname: "Uberti", Popularity: 20},
		{Name: "ArmaLite", Country: "USA", Nickname: "ArmaLite", Popularity: 30},
		{Name: "Magnum Research", Country: "USA", Nickname: "Magnum", Popularity: 25},

		// Classic/Historical Brands with lower popularity
		{Name: "Mauser", Country: "Germany", Nickname: "Mauser", Popularity: 30},
		{Name: "Luger", Country: "Germany", Nickname: "Luger", Popularity: 20},
		{Name: "Webley", Country: "UK", Nickname: "Webley", Popularity: 15},
		{Name: "Enfield", Country: "UK", Nickname: "Enfield", Popularity: 20},

		// Custom / High-End Specialty Manufacturers with lower popularity
		{Name: "Wilson Combat", Country: "USA", Nickname: "Wilson", Popularity: 25},
		{Name: "Les Baer", Country: "USA", Nickname: "Baer", Popularity: 20},
		{Name: "Nighthawk Custom", Country: "USA", Nickname: "Nighthawk", Popularity: 20},
		{Name: "Taran Tactical Innovations", Country: "USA", Nickname: "Taran", Popularity: 15},
		{Name: "Ed Brown Products", Country: "USA", Nickname: "Ed Brown", Popularity: 15},
		{Name: "CCI (Cascade Cartridge Inc.)", Country: "USA", Nickname: "CCI", Popularity: 15},
	}

	// Loop through each manufacturer
	for _, m := range manufacturers {
		var count int64
		// Check if the record exists (by Name)
		if err := db.Model(&models.Manufacturer{}).Where("name = ?", m.Name).Count(&count).Error; err != nil {
			log.Printf("Error checking manufacturer %s: %v", m.Name, err)
			continue
		}
		if count == 0 {
			if err := db.Create(&m).Error; err != nil {
				log.Printf("Error seeding manufacturer %s: %v", m.Name, err)
			} else {
				log.Printf("Seeded manufacturer: %s", m.Name)
			}
		} else {
			// Update the popularity for existing manufacturers
			if err := db.Model(&models.Manufacturer{}).Where("name = ?", m.Name).Update("popularity", m.Popularity).Error; err != nil {
				log.Printf("Error updating popularity for manufacturer %s: %v", m.Name, err)
			} else {
				log.Printf("Updated popularity for manufacturer: %s", m.Name)
			}
		}
	}
}
