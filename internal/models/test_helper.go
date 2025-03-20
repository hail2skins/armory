package models

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	testDB     *gorm.DB
	testDBOnce sync.Once
)

// GetTestDB returns a singleton test database instance
// This avoids creating multiple database connections in tests
func GetTestDB() *gorm.DB {
	testDBOnce.Do(func() {
		// Create a temporary directory for the SQLite database
		tempDir, err := os.MkdirTemp("", "armory-models-test-*")
		if err != nil {
			log.Fatalf("Failed to create temp dir: %v", err)
		}

		// Create a SQLite database in the temporary directory
		dbPath := filepath.Join(tempDir, "models-test.db")

		// Configure GORM logger
		gormLogger := logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
			logger.Config{
				SlowThreshold:             time.Second, // Slow SQL threshold
				LogLevel:                  logger.Info, // Log level
				IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
				Colorful:                  true,        // Enable color
			},
		)

		// Open connection to database
		db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
			Logger: gormLogger,
		})
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}

		// Auto migrate the schema for all model types
		if err := db.AutoMigrate(
			&WeaponType{},
			&Caliber{},
			&Manufacturer{},
			&Gun{},
			&Promotion{},
		); err != nil {
			log.Fatalf("Error auto migrating schema: %v", err)
		}

		// Seed test data
		SeedTestData(db)

		testDB = db
	})

	return testDB
}

// SeedTestData populates the database with test data
func SeedTestData(db *gorm.DB) {
	// Create test weapon types
	types := []WeaponType{
		{Type: "Test Rifle", Nickname: "Test Rifle", Popularity: 90},
		{Type: "Test Pistol", Nickname: "Test Pistol", Popularity: 100},
		{Type: "Test Shotgun", Nickname: "Test Shotgun", Popularity: 80},
	}

	for i := range types {
		db.Where(WeaponType{Type: types[i].Type}).FirstOrCreate(&types[i])
	}

	// Create test calibers
	calibers := []Caliber{
		{Caliber: "Test 5.56", Nickname: "Test 5.56", Popularity: 75},
		{Caliber: "Test 9mm", Nickname: "Test 9mm", Popularity: 80},
		{Caliber: "Test 12 Gauge", Nickname: "Test 12ga", Popularity: 70},
	}

	for i := range calibers {
		db.Where(Caliber{Caliber: calibers[i].Caliber}).FirstOrCreate(&calibers[i])
	}

	// Create test manufacturers
	manufacturers := []Manufacturer{
		{Name: "Test Glock", Country: "Austria", Popularity: 85},
		{Name: "Test Smith & Wesson", Country: "USA", Popularity: 80},
		{Name: "Test Remington", Country: "USA", Popularity: 75},
	}

	for i := range manufacturers {
		db.Where(Manufacturer{Name: manufacturers[i].Name}).FirstOrCreate(&manufacturers[i])
	}
}
