package seed

import (
	"log"

	"github.com/hail2skins/armory/internal/models"
	"gorm.io/gorm"
)

// RunSeeds executes seed functions for core data tables *only* if the respective table is empty.
func RunSeeds(db *gorm.DB) {
	log.Println("Checking if seeding is needed for individual tables...")

	// Seed Manufacturers if the table is empty
	var manufacturerCount int64
	if err := db.Model(&models.Manufacturer{}).Count(&manufacturerCount).Error; err != nil {
		log.Printf("Error checking manufacturers count: %v", err)
	} else if manufacturerCount == 0 {
		log.Println("Seeding manufacturers...")
		SeedManufacturers(db)
	} else {
		log.Printf("Manufacturers table already seeded (count: %d), skipping.", manufacturerCount)
	}

	// Seed Calibers if the table is empty
	var caliberCount int64
	if err := db.Model(&models.Caliber{}).Count(&caliberCount).Error; err != nil {
		log.Printf("Error checking calibers count: %v", err)
	} else if caliberCount == 0 {
		log.Println("Seeding calibers...")
		SeedCalibers(db)
	} else {
		log.Printf("Calibers table already seeded (count: %d), skipping.", caliberCount)
	}

	// Seed Weapon Types if the table is empty
	var weaponTypeCount int64
	if err := db.Model(&models.WeaponType{}).Count(&weaponTypeCount).Error; err != nil {
		log.Printf("Error checking weapon types count: %v", err)
	} else if weaponTypeCount == 0 {
		log.Println("Seeding weapon types...")
		SeedWeaponTypes(db)
	} else {
		log.Printf("Weapon Types table already seeded (count: %d), skipping.", weaponTypeCount)
	}

	// Seed Casings if the table is empty
	var casingCount int64
	if err := db.Model(&models.Casing{}).Count(&casingCount).Error; err != nil {
		log.Printf("Error checking casings count: %v", err)
	} else if casingCount == 0 {
		log.Println("Seeding casings...")
		SeedCasings(db)
	} else {
		log.Printf("Casings table already seeded (count: %d), skipping.", casingCount)
	}

	// Add more seed functions here following the same pattern

	log.Println("Individual table seeding checks completed.")
}

// NeedsSeeding function is likely redundant with the new RunSeeds logic, but kept for now
// in case it's used elsewhere or for potential future refactoring.
func NeedsSeeding(db *gorm.DB) bool {
	// Check weapon types
	var weaponTypeCount int64
	if err := db.Model(&models.WeaponType{}).Count(&weaponTypeCount).Error; err != nil {
		log.Printf("Error checking weapon types: %v", err)
		// return true // If there's an error, don't assume seeding is needed if other tables might have data
	} else if weaponTypeCount > 0 {
		return false // Data found, no need to seed
	}

	// Check calibers
	var caliberCount int64
	if err := db.Model(&models.Caliber{}).Count(&caliberCount).Error; err != nil {
		log.Printf("Error checking calibers: %v", err)
	} else if caliberCount > 0 {
		return false // Data found, no need to seed
	}

	// Check manufacturers
	var manufacturerCount int64
	if err := db.Model(&models.Manufacturer{}).Count(&manufacturerCount).Error; err != nil {
		log.Printf("Error checking manufacturers: %v", err)
	} else if manufacturerCount > 0 {
		return false // Data found, no need to seed
	}

	// Check casings <--- NEW CHECK
	var casingCount int64
	if err := db.Model(&models.Casing{}).Count(&casingCount).Error; err != nil {
		log.Printf("Error checking casings: %v", err)
	} else if casingCount > 0 {
		return false // Data found, no need to seed
	}

	// If we reach here, NONE of the checked tables contained data, so we need to seed.
	log.Println("No data found in Manufacturers, Calibers, WeaponTypes, or Casings. Proceeding with seed.")
	return true
}
