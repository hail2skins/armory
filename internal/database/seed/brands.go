package seed

import (
	"log"

	"github.com/hail2skins/armory/internal/models" // Adjust import path if needed
	"gorm.io/gorm"
)

// SeedBrands seeds the database with common ammunition brands
func SeedBrands(db *gorm.DB) {
	// Define common ammunition brands
	// Popularity is subjective based on general recognition and availability
	brands := []models.Brand{
		// Catch-all option
		{Name: "Other/Unknown", Nickname: "Other", Popularity: 999}, // Highest popularity

		// Major US Brands (often produce across categories)
		{Name: "Federal Premium Ammunition", Nickname: "Federal", Popularity: 100},
		{Name: "Remington Arms Company", Nickname: "Remington", Popularity: 98},
		{Name: "Winchester Ammunition", Nickname: "Winchester", Popularity: 97},
		{Name: "CCI (Cascade Cartridge, Inc.)", Nickname: "CCI", Popularity: 95},              // Known for rimfire & primers
		{Name: "Speer", Nickname: "Speer", Popularity: 90},                                    // Known for Gold Dot, Lawman
		{Name: "Hornady Manufacturing", Nickname: "Hornady", Popularity: 96},                  // Known for precision & hunting
		{Name: "PMC Ammunition (Precision Made Cartridges)", Nickname: "PMC", Popularity: 85}, // Korean, widely available
		{Name: "Fiocchi Ammunition", Nickname: "Fiocchi", Popularity: 80},                     // Italian, US presence
		{Name: "Blazer (by CCI/Vista Outdoor)", Nickname: "Blazer", Popularity: 88},           // Often aluminum or brass case
		{Name: "American Eagle (by Federal/Vista Outdoor)", Nickname: "American Eagle", Popularity: 92},
		{Name: "Sierra Bullets", Nickname: "Sierra", Popularity: 75}, // Primarily bullet maker, but sells loaded ammo
		{Name: "Nosler", Nickname: "Nosler", Popularity: 70},         // Known for hunting bullets, sells loaded ammo
		{Name: "Barnes Bullets", Nickname: "Barnes", Popularity: 65}, // Known for solid copper bullets, sells loaded ammo

		// Major European Brands
		{Name: "Sellier & Bellot", Nickname: "S&B", Popularity: 78},  // Czech
		{Name: "Prvi Partizan", Nickname: "PPU", Popularity: 72},     // Serbian
		{Name: "Norma Precision", Nickname: "Norma", Popularity: 68}, // Swedish
		{Name: "Lapua", Nickname: "Lapua", Popularity: 66},           // Finnish (High-end match)
		{Name: "GECO", Nickname: "GECO", Popularity: 64},             // German/Swiss
		{Name: "RWS", Nickname: "RWS", Popularity: 62},               // German (High-end hunting/match)

		// Common Import/Budget Brands (often Russian origin, availability varies)
		{Name: "TulaAmmo", Nickname: "Tula", Popularity: 50},
		{Name: "Wolf Performance Ammunition", Nickname: "Wolf", Popularity: 48},
		{Name: "Barnaul Ammunition", Nickname: "Barnaul", Popularity: 45},

		// Other Notable Brands
		{Name: "Aguila Ammunition", Nickname: "Aguila", Popularity: 55},             // Mexican
		{Name: "Magtech Ammunition", Nickname: "Magtech", Popularity: 60},           // Brazilian (Part of CBC group with S&B)
		{Name: "Underwood Ammo", Nickname: "Underwood", Popularity: 40},             // Known for powerful loads
		{Name: "Buffalo Bore Ammunition", Nickname: "Buffalo Bore", Popularity: 38}, // Known for powerful loads
		{Name: "Cor-Bon", Nickname: "Cor-Bon", Popularity: 35},
		{Name: "DoubleTap Ammunition", Nickname: "DoubleTap", Popularity: 32},
		{Name: "HSM Ammunition", Nickname: "HSM", Popularity: 30},
		{Name: "Black Hills Ammunition", Nickname: "Black Hills", Popularity: 42}, // Known for match grade
		{Name: "SIG Sauer Ammunition", Nickname: "SIG Ammo", Popularity: 76},      // Firearm manufacturer's ammo line

		// Primarily Bullet Brands (sometimes sell loaded ammo)
		{Name: "Berger Bullets", Nickname: "Berger", Popularity: 28},
		{Name: "Swift Bullet Company", Nickname: "Swift", Popularity: 25},

		// Primarily Rimfire/Specialty
		{Name: "Eley", Nickname: "Eley", Popularity: 36},        // High-end .22 LR match ammo
		{Name: "SK Ammunition", Nickname: "SK", Popularity: 34}, // Rimfire, associated with Lapua

		// Shotgun Focused Brands (though many above make shotshells too)
		{Name: "Kent Cartridge", Nickname: "Kent", Popularity: 22},
		{Name: "Rio Ammunition", Nickname: "Rio", Popularity: 20},
		{Name: "Estate Cartridge (by Federal/Vista Outdoor)", Nickname: "Estate", Popularity: 18},
		{Name: "Fiocchi Shotshells", Nickname: "Fiocchi", Popularity: 79},       // Reiterate Fiocchi for shotshell focus
		{Name: "Remington Shotshells", Nickname: "Remington", Popularity: 97},   // Reiterate Remington
		{Name: "Winchester Shotshells", Nickname: "Winchester", Popularity: 96}, // Reiterate Winchester
		{Name: "Federal Shotshells", Nickname: "Federal", Popularity: 99},       // Reiterate Federal
	}

	// Loop through each brand
	for _, b := range brands {
		var count int64
		// Check if the record exists (by Name)
		if err := db.Model(&models.Brand{}).Where("name = ?", b.Name).Count(&count).Error; err != nil {
			log.Printf("Error checking brand %s: %v", b.Name, err)
			continue // Skip to next brand on error
		}

		if count == 0 {
			// Brand does not exist, create it
			if err := db.Create(&b).Error; err != nil {
				log.Printf("Error seeding brand %s: %v", b.Name, err)
			} else {
				log.Printf("Seeded brand: %s", b.Name)
			}
		} else {
			// Brand exists, update its Nickname and Popularity
			updates := map[string]interface{}{
				"nickname":   b.Nickname,
				"popularity": b.Popularity,
			}
			if err := db.Model(&models.Brand{}).Where("name = ?", b.Name).Updates(updates).Error; err != nil {
				log.Printf("Error updating brand %s: %v", b.Name, err)
			} else {
				log.Printf("Updated brand: %s", b.Name)
			}
		}
	}
}
