package testutils

import (
	"context"
	"log"

	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
)

// SeedData provides a central location for test data setup
type SeedData struct {
	DB database.Service
}

// NewSeedData creates a new seed data utility
func NewSeedData(db database.Service) *SeedData {
	return &SeedData{DB: db}
}

// CreateTestUser creates a test user with default credentials
func (s *SeedData) CreateTestUser(ctx context.Context) *database.User {
	user, err := CreateTestUser(ctx, s.DB, "test@example.com", "password123")
	if err != nil {
		log.Printf("Error creating test user: %v", err)
		return nil
	}
	return user
}

// CreateVerifiedTestUser creates a test user that is already verified
func (s *SeedData) CreateVerifiedTestUser(ctx context.Context) *database.User {
	user, err := s.DB.CreateUser(ctx, "verified@example.com", "password123")
	if err != nil {
		log.Printf("Error creating verified test user: %v", err)
		return nil
	}

	user.Verified = true
	err = s.DB.UpdateUser(ctx, user)
	if err != nil {
		log.Printf("Error verifying test user: %v", err)
		return nil
	}

	return user
}

// CreateTestWeaponTypes creates standard weapons types
func (s *SeedData) CreateTestWeaponTypes() []models.WeaponType {
	db := s.DB.GetDB()

	types := []models.WeaponType{
		{Type: "Test Pistol", Nickname: "Pistol", Popularity: 100},
		{Type: "Test Rifle", Nickname: "Rifle", Popularity: 90},
		{Type: "Test Shotgun", Nickname: "Shotgun", Popularity: 80},
	}

	for i := range types {
		db.Where(models.WeaponType{Type: types[i].Type}).
			FirstOrCreate(&types[i])
	}

	return types
}

// CreateTestCalibers creates standard calibers
func (s *SeedData) CreateTestCalibers() []models.Caliber {
	db := s.DB.GetDB()

	calibers := []models.Caliber{
		{Caliber: "Test 9mm", Nickname: "9mm", Popularity: 80},
		{Caliber: "Test 5.56", Nickname: "5.56", Popularity: 75},
		{Caliber: "Test 12 Gauge", Nickname: "12ga", Popularity: 70},
	}

	for i := range calibers {
		db.Where(models.Caliber{Caliber: calibers[i].Caliber}).
			FirstOrCreate(&calibers[i])
	}

	return calibers
}

// CreateTestManufacturers creates standard manufacturers
func (s *SeedData) CreateTestManufacturers() []models.Manufacturer {
	db := s.DB.GetDB()

	manufacturers := []models.Manufacturer{
		{Name: "Test Glock", Country: "Austria", Popularity: 85},
		{Name: "Test Smith & Wesson", Country: "USA", Popularity: 80},
		{Name: "Test Remington", Country: "USA", Popularity: 75},
	}

	for i := range manufacturers {
		db.Where(models.Manufacturer{Name: manufacturers[i].Name}).
			FirstOrCreate(&manufacturers[i])
	}

	return manufacturers
}

// CreateTestGun creates a test gun with the provided user and fixture data
func (s *SeedData) CreateTestGun(user *database.User) (*models.Gun, error) {
	// Get fixture data
	weaponTypes := s.CreateTestWeaponTypes()
	calibers := s.CreateTestCalibers()
	manufacturers := s.CreateTestManufacturers()

	// Create the gun
	gun := &models.Gun{
		Name:           "Test Gun",
		SerialNumber:   "TEST123",
		WeaponTypeID:   weaponTypes[0].ID,
		CaliberID:      calibers[0].ID,
		ManufacturerID: manufacturers[0].ID,
		OwnerID:        user.Model.ID,
	}

	// Save to database
	err := models.CreateGun(s.DB.GetDB(), gun)
	if err != nil {
		return nil, err
	}

	return gun, nil
}

// CleanupTestData removes all testing data from the database
func (s *SeedData) CleanupTestData() {
	db := s.DB.GetDB()

	// Use transaction to ensure all-or-nothing cleanup
	tx := db.Begin()

	// Delete in reverse order of dependencies
	tx.Where("name LIKE ?", "Test%").Delete(&models.Gun{})
	tx.Where("email LIKE ?", "%@example.com").Delete(&database.User{})
	tx.Where("name LIKE ?", "Test%").Delete(&models.Manufacturer{})
	tx.Where("caliber LIKE ?", "Test%").Delete(&models.Caliber{})
	tx.Where("type LIKE ?", "Test%").Delete(&models.WeaponType{})

	tx.Commit()
}
