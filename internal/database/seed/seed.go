package seed

import (
	"log"

	"gorm.io/gorm"
)

// RunSeeds executes all seed functions
func RunSeeds(db *gorm.DB) {
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
