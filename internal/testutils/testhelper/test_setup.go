package testhelper

import (
	"context"
	"log"

	"github.com/hail2skins/armory/internal/database"
	"gorm.io/gorm"
)

// TestData holds all test fixtures
type TestData struct {
	DB            *gorm.DB
	WeaponTypes   []WeaponType
	Calibers      []Caliber
	Manufacturers []Manufacturer
	Users         []database.User
	Guns          []Gun
	Service       database.Service // Database service for operations that need it
}

// Gun represents a simplified gun for testing
type Gun struct {
	gorm.Model
	Name           string
	SerialNumber   string
	WeaponTypeID   uint
	CaliberID      uint
	ManufacturerID uint
	OwnerID        uint
}

// SetupTestData creates a single source of test data
func SetupTestData(db *gorm.DB) *TestData {
	td := &TestData{DB: db}

	// Create standard test data
	td.WeaponTypes = createTestWeaponTypes(db)
	td.Calibers = createTestCalibers(db)
	td.Manufacturers = createTestManufacturers(db)

	return td
}

// NewTestData creates a new TestData instance with database service
func NewTestData(db *gorm.DB, service database.Service) *TestData {
	td := &TestData{
		DB:      db,
		Service: service,
	}

	// Create standard test data
	td.WeaponTypes = createTestWeaponTypes(db)
	td.Calibers = createTestCalibers(db)
	td.Manufacturers = createTestManufacturers(db)

	return td
}

// createTestWeaponTypes creates standard weapon types for testing
func createTestWeaponTypes(db *gorm.DB) []WeaponType {
	types := []WeaponType{
		{Type: "Test Pistol", Nickname: "Test Pistol", Popularity: 100},
		{Type: "Test Rifle", Nickname: "Test Rifle", Popularity: 90},
		{Type: "Test Shotgun", Nickname: "Test Shotgun", Popularity: 80},
	}

	for i := range types {
		db.Where(WeaponType{Type: types[i].Type}).Table("weapon_types").
			FirstOrCreate(&types[i])
	}

	return types
}

// createTestCalibers creates standard calibers for testing
func createTestCalibers(db *gorm.DB) []Caliber {
	calibers := []Caliber{
		{Caliber: "Test 9mm", Nickname: "Test 9mm", Popularity: 80},
		{Caliber: "Test 5.56", Nickname: "Test 5.56", Popularity: 75},
		{Caliber: "Test 12 Gauge", Nickname: "Test 12ga", Popularity: 70},
	}

	for i := range calibers {
		db.Where(Caliber{Caliber: calibers[i].Caliber}).Table("calibers").
			FirstOrCreate(&calibers[i])
	}

	return calibers
}

// createTestManufacturers creates standard manufacturers for testing
func createTestManufacturers(db *gorm.DB) []Manufacturer {
	manufacturers := []Manufacturer{
		{Name: "Test Glock", Country: "Austria", Popularity: 85},
		{Name: "Test Smith & Wesson", Country: "USA", Popularity: 80},
		{Name: "Test Remington", Country: "USA", Popularity: 75},
	}

	for i := range manufacturers {
		db.Where(Manufacturer{Name: manufacturers[i].Name}).Table("manufacturers").
			FirstOrCreate(&manufacturers[i])
	}

	return manufacturers
}

// CreateTestUser is a helper for creating a test user with standard data
func (td *TestData) CreateTestUser(ctx context.Context) *database.User {
	if td.Service != nil {
		// Use the database service if provided
		user, err := td.Service.CreateUser(ctx, "test@example.com", "Password123!")
		if err != nil {
			log.Printf("Error creating test user: %v", err)
			return nil
		}

		// Mark user as verified for testing convenience
		user.Verified = true
		err = td.Service.UpdateUser(ctx, user)
		if err != nil {
			log.Printf("Error verifying test user: %v", err)
			return nil
		}

		return user
	}

	// Fallback to direct database operations
	var user database.User
	user.Email = "test@example.com"
	user.Password = "Password123!"
	user.Verified = true

	if err := td.DB.Create(&user).Error; err != nil {
		log.Printf("Error creating user directly: %v", err)
		return nil
	}

	return &user
}

// CreateTestGun is a helper for creating a test gun with standard relationships
func (td *TestData) CreateTestGun(user *database.User) *Gun {
	if len(td.WeaponTypes) == 0 || len(td.Calibers) == 0 || len(td.Manufacturers) == 0 {
		log.Printf("Warning: Test fixtures not initialized properly")
		return nil
	}

	// Create a test gun
	gun := &Gun{
		Name:           "Test Gun",
		SerialNumber:   "TEST123",
		WeaponTypeID:   td.WeaponTypes[0].Model.ID,
		CaliberID:      td.Calibers[0].Model.ID,
		ManufacturerID: td.Manufacturers[0].Model.ID,
	}

	// Set owner ID if user is provided
	if user != nil {
		gun.OwnerID = user.Model.ID
	}

	// Save it to the database
	if err := td.DB.Table("guns").Create(gun).Error; err != nil {
		log.Printf("Error creating test gun: %v", err)
		return nil
	}

	return gun
}

// CleanupTestData removes all test data from the database
func (td *TestData) CleanupTestData() {
	// Use transaction to ensure all-or-nothing cleanup
	tx := td.DB.Begin()

	// Delete in reverse order of dependencies
	// Only delete test data with identifiable prefixes/patterns to avoid affecting real data
	tx.Where("name LIKE ?", "Test%").Table("guns").Delete(&Gun{})
	tx.Where("email LIKE ?", "%@example.com").Delete(&database.User{})
	tx.Where("name LIKE ?", "Test%").Table("manufacturers").Delete(&Manufacturer{})
	tx.Where("caliber LIKE ?", "Test%").Table("calibers").Delete(&Caliber{})
	tx.Where("type LIKE ?", "Test%").Table("weapon_types").Delete(&WeaponType{})

	tx.Commit()
}
