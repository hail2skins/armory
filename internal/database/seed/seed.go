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

	// Seed Bullet Styles if the table is empty
	var bulletStyleCount int64
	if err := db.Model(&models.BulletStyle{}).Count(&bulletStyleCount).Error; err != nil {
		log.Printf("Error checking bullet styles count: %v", err)
	} else if bulletStyleCount == 0 {
		log.Println("Seeding bullet styles...")
		SeedBulletStyles(db)
	} else {
		log.Printf("Bullet Styles table already seeded (count: %d), skipping.", bulletStyleCount)
	}

	// Seed Grains if the table is empty
	var grainCount int64
	if err := db.Model(&models.Grain{}).Count(&grainCount).Error; err != nil {
		log.Printf("Error checking grain weights count: %v", err)
	} else if grainCount == 0 {
		log.Println("Seeding grain weights...")
		SeedGrains(db)
	} else {
		log.Printf("Grain weights table already seeded (count: %d), skipping.", grainCount)
	}

	// Seed Brands if the table is empty
	var brandCount int64
	if err := db.Model(&models.Brand{}).Count(&brandCount).Error; err != nil {
		log.Printf("Error checking brands count: %v", err)
	} else if brandCount == 0 {
		log.Println("Seeding brands...")
		SeedBrands(db)
	} else {
		log.Printf("Brands table already seeded (count: %d), skipping.", brandCount)
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

	// Check casings
	var casingCount int64
	if err := db.Model(&models.Casing{}).Count(&casingCount).Error; err != nil {
		log.Printf("Error checking casings: %v", err)
	} else if casingCount > 0 {
		return false // Data found, no need to seed
	}

	// Check bullet styles
	var bulletStyleCount int64
	if err := db.Model(&models.BulletStyle{}).Count(&bulletStyleCount).Error; err != nil {
		log.Printf("Error checking bullet styles: %v", err)
	} else if bulletStyleCount > 0 {
		return false // Data found, no need to seed
	}

	// Check grains
	var grainCount int64
	if err := db.Model(&models.Grain{}).Count(&grainCount).Error; err != nil {
		log.Printf("Error checking grain weights: %v", err)
	} else if grainCount > 0 {
		return false // Data found, no need to seed
	}

	// Check brands
	var brandCount int64
	if err := db.Model(&models.Brand{}).Count(&brandCount).Error; err != nil {
		log.Printf("Error checking brands: %v", err)
	} else if brandCount > 0 {
		return false // Data found, no need to seed
	}

	return true // No data found in any table, seeding is needed
}
