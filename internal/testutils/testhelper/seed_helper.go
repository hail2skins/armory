package testhelper

import (
	"log"

	"gorm.io/gorm"
)

// SeedTestData seeds the database with test data for models tests
func SeedTestData(db *gorm.DB) {
	// Seed weapon types
	seedWeaponTypes(db)

	// Seed calibers
	seedCalibers(db)

	// Seed manufacturers
	seedManufacturers(db)
}

// seedWeaponTypes seeds the database with test weapon types
func seedWeaponTypes(db *gorm.DB) {
	// Create a test weapon type
	weaponType := map[string]interface{}{
		"Type":       "Test Rifle",
		"Nickname":   "Test Rifle",
		"Popularity": 10,
	}

	// Check if it already exists
	var count int64
	db.Table("weapon_types").Where("type = ?", weaponType["Type"]).Count(&count)
	if count == 0 {
		if err := db.Table("weapon_types").Create(weaponType).Error; err != nil {
			log.Printf("Error seeding weapon type: %v", err)
		}
	}
}

// seedCalibers seeds the database with test calibers
func seedCalibers(db *gorm.DB) {
	// Create a test caliber
	caliber := map[string]interface{}{
		"Caliber":    "Test .223",
		"Nickname":   "Test .223",
		"Popularity": 10,
	}

	// Check if it already exists
	var count int64
	db.Table("calibers").Where("caliber = ?", caliber["Caliber"]).Count(&count)
	if count == 0 {
		if err := db.Table("calibers").Create(caliber).Error; err != nil {
			log.Printf("Error seeding caliber: %v", err)
		}
	}
}

// seedManufacturers seeds the database with test manufacturers
func seedManufacturers(db *gorm.DB) {
	// Create a test manufacturer
	manufacturer := map[string]interface{}{
		"Name":       "Test Manufacturer",
		"Nickname":   "Test Mfg",
		"Country":    "Test Country",
		"Popularity": 10,
	}

	// Check if it already exists
	var count int64
	db.Table("manufacturers").Where("name = ?", manufacturer["Name"]).Count(&count)
	if count == 0 {
		if err := db.Table("manufacturers").Create(manufacturer).Error; err != nil {
			log.Printf("Error seeding manufacturer: %v", err)
		}
	}
}
