package testhelper

import (
	"gorm.io/gorm"
)

// Basic type definitions that match models but don't import them
// This avoids import cycles in model tests

// WeaponType represents a weapon type for testing
type WeaponType struct {
	gorm.Model
	Type       string `gorm:"uniqueIndex;not null"`
	Nickname   string
	Popularity int
}

// Caliber represents a caliber for testing
type Caliber struct {
	gorm.Model
	Caliber    string `gorm:"uniqueIndex;not null"`
	Nickname   string
	Popularity int
}

// Manufacturer represents a manufacturer for testing
type Manufacturer struct {
	gorm.Model
	Name       string `gorm:"uniqueIndex;not null"`
	Country    string
	Popularity int
}

// SeedTestData is a simplified function for older tests that just need basic setup
// This helps avoid import cycles with models package
func SeedTestData(db *gorm.DB) {
	// Create test weapon types
	types := []WeaponType{
		{Type: "Test Rifle", Nickname: "Test Rifle", Popularity: 90},
		{Type: "Test Pistol", Nickname: "Test Pistol", Popularity: 100},
		{Type: "Test Shotgun", Nickname: "Test Shotgun", Popularity: 80},
	}

	for i := range types {
		db.Where(WeaponType{Type: types[i].Type}).Table("weapon_types").
			FirstOrCreate(&types[i])
	}

	// Create test calibers
	calibers := []Caliber{
		{Caliber: "Test 9mm", Nickname: "Test 9mm", Popularity: 80},
		{Caliber: "Test 5.56", Nickname: "Test 5.56", Popularity: 75},
		{Caliber: "Test 12 Gauge", Nickname: "Test 12ga", Popularity: 70},
	}

	for i := range calibers {
		db.Where(Caliber{Caliber: calibers[i].Caliber}).Table("calibers").
			FirstOrCreate(&calibers[i])
	}

	// Create test manufacturers
	manufacturers := []Manufacturer{
		{Name: "Test Glock", Country: "Austria", Popularity: 85},
		{Name: "Test Smith & Wesson", Country: "USA", Popularity: 80},
		{Name: "Test Remington", Country: "USA", Popularity: 75},
	}

	for i := range manufacturers {
		db.Where(Manufacturer{Name: manufacturers[i].Name}).Table("manufacturers").
			FirstOrCreate(&manufacturers[i])
	}
}
