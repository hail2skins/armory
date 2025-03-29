package testutils

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestDB represents a test database
type TestDB struct {
	DB *gorm.DB
}

// NewTestDB creates a new test database
func NewTestDB() *TestDB {
	// Create a temporary directory for the SQLite database
	tempDir, err := os.MkdirTemp("", "armory-test-*")
	if err != nil {
		log.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create a SQLite database in the temporary directory
	dbPath := filepath.Join(tempDir, "test.db")

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

	// Auto migrate the schema
	if err := db.AutoMigrate(
		&database.User{},
		&models.Manufacturer{},
		&models.Caliber{},
		&models.WeaponType{},
		&models.Gun{},
		&models.Payment{},
		&models.Promotion{},
		&models.Casing{},
	); err != nil {
		log.Fatalf("Error auto migrating schema: %v", err)
	}

	// Run seed data - REMOVED TO BREAK IMPORT CYCLE
	// seed.RunSeeds(db)

	return &TestDB{
		DB: db,
	}
}

// Close closes the database connection and removes the temporary directory
func (tdb *TestDB) Close() error {
	// Get the database connection
	sqlDB, err := tdb.DB.DB()
	if err != nil {
		return err
	}

	// Close the database connection
	if err := sqlDB.Close(); err != nil {
		return err
	}

	return nil
}

// NewTestService creates a new test database service using the provided db connection
func NewTestService(db *gorm.DB) database.Service {
	return &TestService{
		db: db,
	}
}

// Global shared database instance for testing
var (
	sharedDBInstance    database.Service
	sharedDBInitialized bool
)

// SharedTestService returns a singleton database service for testing
//
// IMPORTANT: Use this function instead of NewTestService() when possible to avoid
// repeatedly seeding the database in each test file. The shared database is seeded only
// once and reused across tests, which significantly improves test performance.
//
// Example usage:
//
//	db := testutils.SharedTestService()
//	defer db.Close() // Close is a no-op for the shared service, so it's safe to call
func SharedTestService() database.Service {
	if !sharedDBInitialized {
		// Create the DB instance first
		testDB := NewTestDB()
		// Pass the DB connection to NewTestService
		sharedDBInstance = NewTestService(testDB.DB)
		sharedDBInitialized = true
		// Note: We might need a mechanism to close testDB.DB when tests are done, but let's address the primary issue first.
	}
	return sharedDBInstance
}

// TestService is a test implementation of the database.Service interface
type TestService struct {
	db *gorm.DB
	// Mock data
	Users         []database.User
	Manufacturers []models.Manufacturer
	Calibers      []models.Caliber
	WeaponTypes   []models.WeaponType
	Casings       []models.Casing
}

// Health returns a map of health status information
func (s *TestService) Health() map[string]string {
	return map[string]string{
		"status":  "up",
		"message": "Test database is healthy",
	}
}

// Close closes the database connection
func (s *TestService) Close() error {
	// If this is the shared instance, don't actually close it
	if sharedDBInitialized && sharedDBInstance == s {
		return nil
	}

	// Get the database connection
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}

	// Close the database connection
	return sqlDB.Close()
}

// Promotion-related methods implementation

// FindAllPromotions retrieves all promotions
func (s *TestService) FindAllPromotions() ([]models.Promotion, error) {
	var promotions []models.Promotion
	if err := s.db.Find(&promotions).Error; err != nil {
		return nil, err
	}
	return promotions, nil
}

// FindPromotionByID retrieves a promotion by its ID
func (s *TestService) FindPromotionByID(id uint) (*models.Promotion, error) {
	var promotion models.Promotion
	if err := s.db.First(&promotion, id).Error; err != nil {
		return nil, err
	}
	return &promotion, nil
}

// CreatePromotion creates a new promotion
func (s *TestService) CreatePromotion(promotion *models.Promotion) error {
	return s.db.Create(promotion).Error
}

// UpdatePromotion updates an existing promotion
func (s *TestService) UpdatePromotion(promotion *models.Promotion) error {
	return s.db.Save(promotion).Error
}

// DeletePromotion deletes a promotion
func (s *TestService) DeletePromotion(id uint) error {
	return s.db.Delete(&models.Promotion{}, id).Error
}

// FindActivePromotions finds all active promotions for the current time period
func (s *TestService) FindActivePromotions() ([]models.Promotion, error) {
	var promotions []models.Promotion
	now := time.Now()

	// Find promotions that are:
	// 1. Marked as active
	// 2. Current date is between start date and end date
	err := s.db.Where("active = ? AND ? BETWEEN start_date AND end_date", true, now).
		Find(&promotions).Error

	if err != nil {
		return nil, err
	}

	return promotions, nil
}

// CreateUser creates a new user
func (s *TestService) CreateUser(ctx context.Context, email, password string) (*database.User, error) {
	user := &database.User{
		Email: email,
	}

	// Hash the password
	hashedPassword, err := database.HashPassword(password)
	if err != nil {
		return nil, err
	}
	user.Password = hashedPassword

	// Create the user
	if err := s.db.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByEmail gets a user by email
func (s *TestService) GetUserByEmail(ctx context.Context, email string) (*database.User, error) {
	var user database.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// AuthenticateUser authenticates a user
func (s *TestService) AuthenticateUser(ctx context.Context, email, password string) (*database.User, error) {
	user, err := s.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, database.ErrInvalidCredentials
	}

	// Check the password
	matches := database.CheckPassword(password, user.Password)
	if !matches {
		return nil, database.ErrInvalidCredentials
	}

	return user, nil
}

// GetUserByVerificationToken gets a user by verification token
func (s *TestService) GetUserByVerificationToken(ctx context.Context, token string) (*database.User, error) {
	var user database.User
	if err := s.db.Where("verification_token = ?", token).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByRecoveryToken gets a user by recovery token
func (s *TestService) GetUserByRecoveryToken(ctx context.Context, token string) (*database.User, error) {
	var user database.User
	if err := s.db.Where("recovery_token = ?", token).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// VerifyUserEmail verifies a user's email
func (s *TestService) VerifyUserEmail(ctx context.Context, token string) (*database.User, error) {
	user, err := s.GetUserByVerificationToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	// Check if the token is expired
	if user.VerificationTokenExpiry.Before(time.Now()) {
		return nil, errors.New("verification token has expired")
	}

	// Mark the user as verified
	user.Verified = true
	user.VerificationToken = ""
	user.VerificationTokenExpiry = time.Time{}

	// Update the user
	if err := s.UpdateUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// UpdateUser updates a user
func (s *TestService) UpdateUser(ctx context.Context, user *database.User) error {
	return s.db.Save(user).Error
}

// RequestPasswordReset requests a password reset
func (s *TestService) RequestPasswordReset(ctx context.Context, email string) (*database.User, error) {
	user, err := s.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	// Generate a recovery token
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return nil, err
	}
	recoveryToken := base64.URLEncoding.EncodeToString(token)

	// Set the recovery token and expiry
	user.RecoveryToken = recoveryToken
	user.RecoveryTokenExpiry = time.Now().Add(24 * time.Hour)

	// Update the user
	if err := s.UpdateUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// ResetPassword resets a user's password
func (s *TestService) ResetPassword(ctx context.Context, token, newPassword string) error {
	user, err := s.GetUserByRecoveryToken(ctx, token)
	if err != nil {
		return err
	}
	if user == nil {
		return database.ErrInvalidToken
	}

	// Check if the token is expired
	if user.RecoveryTokenExpiry.Before(time.Now()) {
		return database.ErrTokenExpired
	}

	// Hash the new password
	hashedPassword, err := database.HashPassword(newPassword)
	if err != nil {
		return err
	}

	// Update the user
	user.Password = hashedPassword
	user.RecoveryToken = ""
	user.RecoveryTokenExpiry = time.Time{}

	return s.UpdateUser(ctx, user)
}

// GetUserByID gets a user by ID
func (s *TestService) GetUserByID(id uint) (*database.User, error) {
	var user database.User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByStripeCustomerID gets a user by Stripe customer ID
func (s *TestService) GetUserByStripeCustomerID(customerID string) (*database.User, error) {
	var user database.User
	if err := s.db.Where("stripe_customer_id = ?", customerID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// CreatePayment creates a payment
func (s *TestService) CreatePayment(payment *models.Payment) error {
	return s.db.Create(payment).Error
}

// GetPaymentsByUserID gets payments by user ID
func (s *TestService) GetPaymentsByUserID(userID uint) ([]models.Payment, error) {
	var payments []models.Payment
	if err := s.db.Where("user_id = ?", userID).Find(&payments).Error; err != nil {
		return nil, err
	}
	return payments, nil
}

// GetAllPayments gets all payments
func (s *TestService) GetAllPayments() ([]models.Payment, error) {
	var payments []models.Payment
	if err := s.db.Order("created_at desc").Find(&payments).Error; err != nil {
		return nil, err
	}
	return payments, nil
}

// FindPaymentByID finds a payment by ID
func (s *TestService) FindPaymentByID(id uint) (*models.Payment, error) {
	var payment models.Payment
	if err := s.db.First(&payment, id).Error; err != nil {
		return nil, err
	}
	return &payment, nil
}

// UpdatePayment updates a payment
func (s *TestService) UpdatePayment(payment *models.Payment) error {
	return s.db.Save(payment).Error
}

// FindAllManufacturers retrieves all manufacturers
func (s *TestService) FindAllManufacturers() ([]models.Manufacturer, error) {
	var manufacturers []models.Manufacturer
	if err := s.db.Find(&manufacturers).Error; err != nil {
		return nil, err
	}
	return manufacturers, nil
}

// FindManufacturerByID retrieves a manufacturer by ID
func (s *TestService) FindManufacturerByID(id uint) (*models.Manufacturer, error) {
	var manufacturer models.Manufacturer
	if err := s.db.First(&manufacturer, id).Error; err != nil {
		return nil, err
	}
	return &manufacturer, nil
}

// CreateManufacturer creates a new manufacturer
func (s *TestService) CreateManufacturer(manufacturer *models.Manufacturer) error {
	return s.db.Create(manufacturer).Error
}

// UpdateManufacturer updates a manufacturer
func (s *TestService) UpdateManufacturer(manufacturer *models.Manufacturer) error {
	return s.db.Save(manufacturer).Error
}

// DeleteManufacturer deletes a manufacturer
func (s *TestService) DeleteManufacturer(id uint) error {
	return s.db.Delete(&models.Manufacturer{}, id).Error
}

// FindAllCalibers retrieves all calibers
func (s *TestService) FindAllCalibers() ([]models.Caliber, error) {
	var calibers []models.Caliber
	if err := s.db.Find(&calibers).Error; err != nil {
		return nil, err
	}
	return calibers, nil
}

// FindCaliberByID retrieves a caliber by ID
func (s *TestService) FindCaliberByID(id uint) (*models.Caliber, error) {
	var caliber models.Caliber
	if err := s.db.First(&caliber, id).Error; err != nil {
		return nil, err
	}
	return &caliber, nil
}

// CreateCaliber creates a new caliber
func (s *TestService) CreateCaliber(caliber *models.Caliber) error {
	return s.db.Create(caliber).Error
}

// UpdateCaliber updates a caliber
func (s *TestService) UpdateCaliber(caliber *models.Caliber) error {
	return s.db.Save(caliber).Error
}

// DeleteCaliber deletes a caliber
func (s *TestService) DeleteCaliber(id uint) error {
	return s.db.Delete(&models.Caliber{}, id).Error
}

// FindAllWeaponTypes retrieves all weapon types
func (s *TestService) FindAllWeaponTypes() ([]models.WeaponType, error) {
	var weaponTypes []models.WeaponType
	if err := s.db.Find(&weaponTypes).Error; err != nil {
		return nil, err
	}
	return weaponTypes, nil
}

// FindWeaponTypeByID retrieves a weapon type by ID
func (s *TestService) FindWeaponTypeByID(id uint) (*models.WeaponType, error) {
	var weaponType models.WeaponType
	if err := s.db.First(&weaponType, id).Error; err != nil {
		return nil, err
	}
	return &weaponType, nil
}

// CreateWeaponType creates a new weapon type
func (s *TestService) CreateWeaponType(weaponType *models.WeaponType) error {
	return s.db.Create(weaponType).Error
}

// UpdateWeaponType updates a weapon type
func (s *TestService) UpdateWeaponType(weaponType *models.WeaponType) error {
	return s.db.Save(weaponType).Error
}

// DeleteWeaponType deletes a weapon type
func (s *TestService) DeleteWeaponType(id uint) error {
	return s.db.Delete(&models.WeaponType{}, id).Error
}

// GetDB returns the underlying *gorm.DB instance
func (s *TestService) GetDB() *gorm.DB {
	return s.db
}

// DeleteGun deletes a gun from the database
func (s *TestService) DeleteGun(db *gorm.DB, id uint, ownerID uint) error {
	return models.DeleteGun(s.db, id, ownerID)
}

// IsRecoveryExpired checks if a recovery token is expired in the test service
func (s *TestService) IsRecoveryExpired(ctx context.Context, token string) (bool, error) {
	user, err := s.GetUserByRecoveryToken(ctx, token)
	if err != nil {
		return true, err
	}
	return user.IsRecoveryExpired(), nil
}

// CountUsers returns the count of users
func (s *TestService) CountUsers() (int64, error) {
	var count int64
	err := s.db.Model(&database.User{}).Count(&count).Error
	return count, err
}

// FindRecentUsers returns a list of recent users with pagination and sorting
func (s *TestService) FindRecentUsers(offset, limit int, sortBy, sortOrder string) ([]database.User, error) {
	var users []database.User
	result := s.db.Model(&database.User{}).
		Order(sortBy + " " + sortOrder).
		Offset(offset).
		Limit(limit).
		Find(&users)
	return users, result.Error
}

// CountActiveSubscribers returns the number of users with active paid subscriptions
func (s *TestService) CountActiveSubscribers() (int64, error) {
	var count int64
	// This is a test implementation, so we'll return a dummy value
	err := s.db.Model(&database.User{}).
		Where("subscription_status = ? AND is_admin_granted = ? AND subscription_tier != ?",
			"active", false, "free").
		Count(&count).Error
	return count, err
}

// CountNewUsersThisMonth returns the number of users registered in the current month
func (s *TestService) CountNewUsersThisMonth() (int64, error) {
	var count int64
	// Get the first day of the current month
	now := time.Now()
	firstDay := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	// Count users created this month
	err := s.db.Model(&database.User{}).
		Where("created_at >= ?", firstDay).
		Count(&count).Error

	return count, err
}

// CountNewUsersLastMonth returns the number of users registered in the previous month
func (s *TestService) CountNewUsersLastMonth() (int64, error) {
	var count int64
	// Get the first day of the current and previous month
	now := time.Now()

	// Get previous month
	lastMonth := now.AddDate(0, -1, 0)
	firstDayLastMonth := time.Date(lastMonth.Year(), lastMonth.Month(), 1, 0, 0, 0, 0, now.Location())
	firstDayThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	// Count users created last month
	err := s.db.Model(&database.User{}).
		Where("created_at >= ? AND created_at < ?", firstDayLastMonth, firstDayThisMonth).
		Count(&count).Error

	return count, err
}

// CountNewSubscribersThisMonth returns the number of new subscriptions in the current month
func (s *TestService) CountNewSubscribersThisMonth() (int64, error) {
	var count int64
	// Get the first day of the current month
	now := time.Now()
	firstDay := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	// Count users who got a subscription this month (not admin granted)
	err := s.db.Model(&database.User{}).
		Where("subscription_status = ? AND is_admin_granted = ? AND updated_at >= ? AND subscription_tier != ?",
			"active", false, firstDay, "free").
		Count(&count).Error

	return count, err
}

// CountNewSubscribersLastMonth returns the number of new subscriptions in the previous month
func (s *TestService) CountNewSubscribersLastMonth() (int64, error) {
	var count int64
	// Get the first day of the current and previous month
	now := time.Now()

	// Get previous month
	lastMonth := now.AddDate(0, -1, 0)
	firstDayLastMonth := time.Date(lastMonth.Year(), lastMonth.Month(), 1, 0, 0, 0, 0, now.Location())
	firstDayThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	// Count users who got a subscription last month (not admin granted)
	err := s.db.Model(&database.User{}).
		Where("subscription_status = ? AND is_admin_granted = ? AND updated_at >= ? AND updated_at < ? AND subscription_tier != ?",
			"active", false, firstDayLastMonth, firstDayThisMonth, "free").
		Count(&count).Error

	return count, err
}

// FindAllGuns retrieves all guns
func (s *TestService) FindAllGuns() ([]models.Gun, error) {
	var guns []models.Gun
	if err := s.db.Find(&guns).Error; err != nil {
		return nil, err
	}
	return guns, nil
}

// FindAllUsers retrieves all users
func (s *TestService) FindAllUsers() ([]database.User, error) {
	var users []database.User
	if err := s.db.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// CountGunsByUser counts the number of guns owned by a user
func (s *TestService) CountGunsByUser(userID uint) (int64, error) {
	var count int64
	if err := s.db.Model(&models.Gun{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// FindAllCalibersByIDs retrieves all calibers with the given IDs
func (s *TestService) FindAllCalibersByIDs(ids []uint) ([]models.Caliber, error) {
	var calibers []models.Caliber
	if err := s.db.Where("id IN ?", ids).Find(&calibers).Error; err != nil {
		return nil, err
	}
	return calibers, nil
}

// FindAllWeaponTypesByIDs retrieves all weapon types with the given IDs
func (s *TestService) FindAllWeaponTypesByIDs(ids []uint) ([]models.WeaponType, error) {
	var weaponTypes []models.WeaponType
	if err := s.db.Where("id IN ?", ids).Find(&weaponTypes).Error; err != nil {
		return nil, err
	}
	return weaponTypes, nil
}

// Feature Flag-related methods implementation

// FindAllFeatureFlags retrieves all feature flags
func (s *TestService) FindAllFeatureFlags() ([]models.FeatureFlag, error) {
	var featureFlags []models.FeatureFlag
	if err := s.db.Find(&featureFlags).Error; err != nil {
		return nil, err
	}
	return featureFlags, nil
}

// FindFeatureFlagByID retrieves a feature flag by its ID
func (s *TestService) FindFeatureFlagByID(id uint) (*models.FeatureFlag, error) {
	var featureFlag models.FeatureFlag
	if err := s.db.First(&featureFlag, id).Error; err != nil {
		return nil, err
	}
	return &featureFlag, nil
}

// FindFeatureFlagByName retrieves a feature flag by its name
func (s *TestService) FindFeatureFlagByName(name string) (*models.FeatureFlag, error) {
	var featureFlag models.FeatureFlag
	if err := s.db.Where("name = ?", name).First(&featureFlag).Error; err != nil {
		return nil, err
	}
	return &featureFlag, nil
}

// CreateFeatureFlag creates a new feature flag
func (s *TestService) CreateFeatureFlag(flag *models.FeatureFlag) error {
	return s.db.Create(flag).Error
}

// UpdateFeatureFlag updates an existing feature flag
func (s *TestService) UpdateFeatureFlag(flag *models.FeatureFlag) error {
	return s.db.Save(flag).Error
}

// DeleteFeatureFlag deletes a feature flag
func (s *TestService) DeleteFeatureFlag(id uint) error {
	return s.db.Delete(&models.FeatureFlag{}, id).Error
}

// AddRoleToFeatureFlag adds a role to a feature flag
func (s *TestService) AddRoleToFeatureFlag(flagID uint, role string) error {
	return s.db.Create(&models.FeatureFlagRole{
		FeatureFlagID: flagID,
		Role:          role,
	}).Error
}

// RemoveRoleFromFeatureFlag removes a role from a feature flag
func (s *TestService) RemoveRoleFromFeatureFlag(flagID uint, role string) error {
	return s.db.Where("feature_flag_id = ? AND role = ?", flagID, role).
		Delete(&models.FeatureFlagRole{}).Error
}

// IsFeatureEnabled checks if a feature is enabled
func (s *TestService) IsFeatureEnabled(name string) (bool, error) {
	var featureFlag models.FeatureFlag
	if err := s.db.Where("name = ?", name).First(&featureFlag).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return featureFlag.Enabled, nil
}

// CanUserAccessFeature checks if a user can access a feature
func (s *TestService) CanUserAccessFeature(username, featureName string) (bool, error) {
	return models.CanAccessFeature(s.db, username, featureName)
}

// Casing-related methods implementation

// FindAllCasings retrieves all casings
func (s *TestService) FindAllCasings() ([]models.Casing, error) {
	return s.Casings, nil
}

// CreateCasing creates a new casing
func (s *TestService) CreateCasing(casing *models.Casing) error {
	// Check if there's a soft-deleted casing with the same type
	for i, existingCasing := range s.Casings {
		if existingCasing.Type == casing.Type && existingCasing.DeletedAt.Valid {
			// Found a soft-deleted casing with the same type, restore it
			s.Casings[i].DeletedAt.Valid = false
			s.Casings[i].DeletedAt.Time = time.Time{}
			s.Casings[i].Popularity = casing.Popularity // Update with new values
			*casing = s.Casings[i]                      // Return the updated casing
			return nil
		}
	}

	// If reaching here, check for unique constraint
	for _, existingCasing := range s.Casings {
		if existingCasing.Type == casing.Type && !existingCasing.DeletedAt.Valid {
			// Active casing with same type already exists
			return fmt.Errorf("ERROR: duplicate key value violates unique constraint \"uni_casings_type\"")
		}
	}

	// Set an ID if it doesn't have one
	if casing.ID == 0 {
		casing.ID = uint(len(s.Casings) + 1)
	}
	s.Casings = append(s.Casings, *casing)
	return nil
}

// FindCasingByID is a mock implementation for testing
func (s *TestService) FindCasingByID(id uint) (*models.Casing, error) {
	for _, casing := range s.Casings {
		if casing.ID == id {
			return &casing, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

// UpdateCasing is a mock implementation for testing
func (s *TestService) UpdateCasing(casing *models.Casing) error {
	for i, existingCasing := range s.Casings {
		if existingCasing.ID == casing.ID {
			// Update the casing in the array
			s.Casings[i] = *casing
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

// DeleteCasing is a mock implementation for testing
func (s *TestService) DeleteCasing(id uint) error {
	for i, casing := range s.Casings {
		if casing.ID == id {
			// Remove the casing from the array by replacing it with the last element
			// and then truncating the slice
			s.Casings[i] = s.Casings[len(s.Casings)-1]
			s.Casings = s.Casings[:len(s.Casings)-1]
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}
