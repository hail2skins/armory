package seed

import (
	"log"

	"github.com/hail2skins/armory/internal/models"
	"gorm.io/gorm"
)

// RunSeeds executes all seed functions
func RunSeeds(db *gorm.DB) {
	// Check if database already contains data
	if !NeedsSeeding(db) {
		log.Println("Database already contains data, skipping seed")
		return
	}

	log.Println("Starting database seeding...")

	// Run manufacturer seeds
	log.Println("Seeding manufacturers...")
	SeedManufacturers(db)

	// Run caliber seeds
	log.Println("Seeding calibers...")
	SeedCalibers(db)

	// Run weapon type seeds
	log.Println("Seeding weapon types...")
	SeedWeaponTypes(db)

	// Add more seed functions here as needed

	log.Println("Database seeding completed")
}

// NeedsSeeding checks if the database needs to be seeded by checking
// if any essential reference data already exists
func NeedsSeeding(db *gorm.DB) bool {
	// Check weapon types (if any exist, we'll assume the database has been seeded)
	var weaponTypeCount int64
	if err := db.Model(&models.WeaponType{}).Count(&weaponTypeCount).Error; err != nil {
		log.Printf("Error checking weapon types: %v", err)
		return true // If there's an error, assume we need to seed
	}
	if weaponTypeCount > 0 {
		return false
	}

	// Check calibers
	var caliberCount int64
	if err := db.Model(&models.Caliber{}).Count(&caliberCount).Error; err != nil {
		log.Printf("Error checking calibers: %v", err)
		return true
	}
	if caliberCount > 0 {
		return false
	}

	// Check manufacturers
	var manufacturerCount int64
	if err := db.Model(&models.Manufacturer{}).Count(&manufacturerCount).Error; err != nil {
		log.Printf("Error checking manufacturers: %v", err)
		return true
	}
	if manufacturerCount > 0 {
		return false
	}

	// If we reach here, no data exists, so we need to seed
	return true
}
