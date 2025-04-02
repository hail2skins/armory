package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/hail2skins/armory/internal/database/seed"
	"github.com/hail2skins/armory/internal/models"
	_ "github.com/joho/godotenv/autoload"
)

// Service represents a service that interacts with a database.
type Service interface {
	// Health returns a map of health status information.
	// The keys and values in the map are service-specific.
	Health() map[string]string

	// Close terminates the database connection.
	// It returns an error if the connection cannot be closed.
	Close() error

	// Include UserService methods
	UserService

	// Include AdminService methods
	AdminService

	// Payment-related methods
	CreatePayment(payment *models.Payment) error
	GetPaymentsByUserID(userID uint) ([]models.Payment, error)
	GetAllPayments() ([]models.Payment, error)
	FindPaymentByID(id uint) (*models.Payment, error)
	UpdatePayment(payment *models.Payment) error

	// Promotion-related methods
	FindAllPromotions() ([]models.Promotion, error)
	FindPromotionByID(id uint) (*models.Promotion, error)
	CreatePromotion(promotion *models.Promotion) error
	UpdatePromotion(promotion *models.Promotion) error
	DeletePromotion(id uint) error
	FindActivePromotions() ([]models.Promotion, error)

	// Feature Flag-related methods
	FindAllFeatureFlags() ([]models.FeatureFlag, error)
	FindFeatureFlagByID(id uint) (*models.FeatureFlag, error)
	FindFeatureFlagByName(name string) (*models.FeatureFlag, error)
	CreateFeatureFlag(flag *models.FeatureFlag) error
	UpdateFeatureFlag(flag *models.FeatureFlag) error
	DeleteFeatureFlag(id uint) error
	AddRoleToFeatureFlag(flagID uint, role string) error
	RemoveRoleFromFeatureFlag(flagID uint, role string) error
	IsFeatureEnabled(name string) (bool, error)
	CanUserAccessFeature(username, featureName string) (bool, error)

	// Additional user methods
	GetUserByID(id uint) (*User, error)
	GetUserByStripeCustomerID(customerID string) (*User, error)

	// Gun-related methods
	DeleteGun(db *gorm.DB, id uint, ownerID uint) error
	FindAllGuns() ([]models.Gun, error)
	FindAllUsers() ([]User, error)
	CountGunsByUser(userID uint) (int64, error)
	FindAllCalibersByIDs(ids []uint) ([]models.Caliber, error)
	FindAllWeaponTypesByIDs(ids []uint) ([]models.WeaponType, error)

	// Ammo-related methods
	FindAllAmmo() ([]models.Ammo, error)
	FindAmmoByID(id uint) (*models.Ammo, error)
	CountAmmoByUser(userID uint) (int64, error)

	// Casing-related methods
	FindAllCasings() ([]models.Casing, error)
	CreateCasing(casing *models.Casing) error
	FindCasingByID(id uint) (*models.Casing, error)
	UpdateCasing(casing *models.Casing) error
	DeleteCasing(id uint) error

	// BulletStyle-related methods
	FindAllBulletStyles() ([]models.BulletStyle, error)
	CreateBulletStyle(bulletStyle *models.BulletStyle) error
	FindBulletStyleByID(id uint) (*models.BulletStyle, error)
	UpdateBulletStyle(bulletStyle *models.BulletStyle) error
	DeleteBulletStyle(id uint) error

	// Grain-related methods
	FindAllGrains() ([]models.Grain, error)
	CreateGrain(grain *models.Grain) error
	FindGrainByID(id uint) (*models.Grain, error)
	UpdateGrain(grain *models.Grain) error
	DeleteGrain(id uint) error

	// Brand-related methods
	FindAllBrands() ([]models.Brand, error)
	CreateBrand(brand *models.Brand) error
	FindBrandByID(id uint) (*models.Brand, error)
	UpdateBrand(brand *models.Brand) error
	DeleteBrand(id uint) error

	// GetDB returns the underlying *gorm.DB instance
	GetDB() *gorm.DB
}

type service struct {
	db *gorm.DB
	mu sync.Mutex // Mutex to protect the db instance
}

var (
	database   = os.Getenv("DB_DATABASE")
	password   = os.Getenv("DB_PASSWORD")
	username   = os.Getenv("DB_USERNAME")
	port       = os.Getenv("DB_PORT")
	host       = os.Getenv("DB_HOST")
	schema     = os.Getenv("DB_SCHEMA")
	dbInstance *service
	mu         sync.Mutex // Mutex to protect the dbInstance
)

func New() Service {
	mu.Lock()
	defer mu.Unlock()

	// Reuse Connection
	if dbInstance != nil && dbInstance.db != nil {
		return dbInstance
	}

	// Set default schema if not provided
	searchPath := schema
	if searchPath == "" {
		searchPath = "public"
	}

	// Create DSN string for PostgreSQL
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable search_path=%s",
		host, username, password, database, port, searchPath)

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
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Create service instance
	dbInstance = &service{
		db: db,
	}

	// Auto migrate the schema
	if err := dbInstance.AutoMigrate(); err != nil {
		log.Printf("Error auto migrating schema: %v", err)
	}

	// Seed the database with initial data
	seed.RunSeeds(dbInstance.db)

	return dbInstance
}

// AutoMigrate automatically migrates the schema
func (s *service) AutoMigrate() error {
	return s.db.AutoMigrate(
		&User{},
		&models.Payment{},
		&models.Manufacturer{},
		&models.Caliber{},
		&models.WeaponType{},
		&models.Gun{},
		&models.Promotion{},
		&models.CasbinRule{},
		&models.FeatureFlag{},
		&models.FeatureFlagRole{},
		&models.Casing{},
		&models.BulletStyle{},
		&models.Grain{},
		&models.Brand{},
		&models.Ammo{},
	)
}

// Health checks the health of the database connection by pinging the database.
// It returns a map with keys indicating various health statistics.
func (s *service) Health() map[string]string {
	stats := make(map[string]string)

	// Get SQL DB instance
	sqlDB, err := s.db.DB()
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		log.Printf("db down: %v", err)
		return stats
	}

	// Ping the database
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = sqlDB.PingContext(ctx)
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		log.Printf("db down: %v", err)
		return stats
	}

	// Database is up, add more statistics
	stats["status"] = "up"
	stats["message"] = "It's healthy"

	// Get database stats (like open connections, in use, idle, etc.)
	dbStats := sqlDB.Stats()
	stats["open_connections"] = strconv.Itoa(dbStats.OpenConnections)
	stats["in_use"] = strconv.Itoa(dbStats.InUse)
	stats["idle"] = strconv.Itoa(dbStats.Idle)
	stats["wait_count"] = strconv.FormatInt(dbStats.WaitCount, 10)
	stats["wait_duration"] = dbStats.WaitDuration.String()
	stats["max_idle_closed"] = strconv.FormatInt(dbStats.MaxIdleClosed, 10)
	stats["max_lifetime_closed"] = strconv.FormatInt(dbStats.MaxLifetimeClosed, 10)

	// Evaluate stats to provide a health message
	if dbStats.OpenConnections > 40 { // Assuming 50 is the max for this example
		stats["message"] = "The database is experiencing heavy load."
	}

	if dbStats.WaitCount > 1000 {
		stats["message"] = "The database has a high number of wait events, indicating potential bottlenecks."
	}

	if dbStats.MaxIdleClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many idle connections are being closed, consider revising the connection pool settings."
	}

	if dbStats.MaxLifetimeClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many connections are being closed due to max lifetime, consider increasing max lifetime or revising the connection usage pattern."
	}

	return stats
}

// Close closes the database connection.
// It logs a message indicating the disconnection from the specific database.
// If the connection is successfully closed, it returns nil.
// If an error occurs while closing the connection, it returns the error.
func (s *service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db == nil {
		return nil // Already closed
	}

	log.Printf("Disconnected from database: %s", database)

	// Get SQL DB instance
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}

	err = sqlDB.Close()
	if err == nil {
		// Only set db to nil if close was successful
		s.db = nil
	}
	return err
}

// GetDB returns the underlying *gorm.DB instance
func (s *service) GetDB() *gorm.DB {
	return s.db
}

// DeleteGun deletes a gun from the database
func (s *service) DeleteGun(db *gorm.DB, id uint, ownerID uint) error {
	return models.DeleteGun(s.db, id, ownerID)
}

// Payment-related methods implementation
// CreatePayment creates a new payment record
func (s *service) CreatePayment(payment *models.Payment) error {
	return s.db.Create(payment).Error
}

// GetPaymentsByUserID retrieves all payments for a user
func (s *service) GetPaymentsByUserID(userID uint) ([]models.Payment, error) {
	var payments []models.Payment
	if err := s.db.Where("user_id = ?", userID).Order("created_at desc").Find(&payments).Error; err != nil {
		return nil, err
	}
	return payments, nil
}

// GetAllPayments retrieves all payments ordered by creation date descending
func (s *service) GetAllPayments() ([]models.Payment, error) {
	var payments []models.Payment
	if err := s.db.Order("created_at desc").Find(&payments).Error; err != nil {
		return nil, err
	}
	return payments, nil
}

// FindPaymentByID retrieves a payment by its ID
func (s *service) FindPaymentByID(id uint) (*models.Payment, error) {
	var payment models.Payment
	if err := s.db.First(&payment, id).Error; err != nil {
		return nil, err
	}
	return &payment, nil
}

// UpdatePayment updates an existing payment in the database
func (s *service) UpdatePayment(payment *models.Payment) error {
	return s.db.Save(payment).Error
}

// Promotion-related methods implementation
// FindAllPromotions retrieves all promotions
func (s *service) FindAllPromotions() ([]models.Promotion, error) {
	var promotions []models.Promotion
	if err := s.db.Find(&promotions).Error; err != nil {
		return nil, err
	}
	return promotions, nil
}

// FindPromotionByID retrieves a promotion by its ID
func (s *service) FindPromotionByID(id uint) (*models.Promotion, error) {
	var promotion models.Promotion
	if err := s.db.First(&promotion, id).Error; err != nil {
		return nil, err
	}
	return &promotion, nil
}

// CreatePromotion creates a new promotion
func (s *service) CreatePromotion(promotion *models.Promotion) error {
	return s.db.Create(promotion).Error
}

// UpdatePromotion updates an existing promotion in the database
func (s *service) UpdatePromotion(promotion *models.Promotion) error {
	return s.db.Save(promotion).Error
}

// DeletePromotion deletes a promotion from the database
func (s *service) DeletePromotion(id uint) error {
	return s.db.Delete(&models.Promotion{}, id).Error
}

// FindActivePromotions finds all active promotions for the current time period
func (s *service) FindActivePromotions() ([]models.Promotion, error) {
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

// Gun-related methods implementation
// FindAllGuns retrieves all guns from the database
func (s *service) FindAllGuns() ([]models.Gun, error) {
	var guns []models.Gun
	if err := s.db.Find(&guns).Error; err != nil {
		return nil, err
	}
	return guns, nil
}

// FindAllUsers retrieves all users from the database
func (s *service) FindAllUsers() ([]User, error) {
	var users []User
	if err := s.db.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// CountGunsByUser retrieves the count of guns for a specific user
func (s *service) CountGunsByUser(userID uint) (int64, error) {
	var count int64
	if err := s.db.Model(&models.Gun{}).Where("owner_id = ?", userID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// FindAllCalibersByIDs retrieves all calibers for a list of IDs
func (s *service) FindAllCalibersByIDs(ids []uint) ([]models.Caliber, error) {
	var calibers []models.Caliber
	if err := s.db.Where("id IN ?", ids).Find(&calibers).Error; err != nil {
		return nil, err
	}
	return calibers, nil
}

// FindAllWeaponTypesByIDs retrieves all weapon types for a list of IDs
func (s *service) FindAllWeaponTypesByIDs(ids []uint) ([]models.WeaponType, error) {
	var weaponTypes []models.WeaponType
	if err := s.db.Where("id IN ?", ids).Find(&weaponTypes).Error; err != nil {
		return nil, err
	}
	return weaponTypes, nil
}

// Feature Flag-related methods implementation
// FindAllFeatureFlags retrieves all feature flags
func (s *service) FindAllFeatureFlags() ([]models.FeatureFlag, error) {
	var featureFlags []models.FeatureFlag
	if err := s.db.Find(&featureFlags).Error; err != nil {
		return nil, err
	}
	return featureFlags, nil
}

// FindFeatureFlagByID retrieves a feature flag by its ID
func (s *service) FindFeatureFlagByID(id uint) (*models.FeatureFlag, error) {
	var featureFlag models.FeatureFlag
	if err := s.db.First(&featureFlag, id).Error; err != nil {
		return nil, err
	}
	return &featureFlag, nil
}

// FindFeatureFlagByName retrieves a feature flag by its name
func (s *service) FindFeatureFlagByName(name string) (*models.FeatureFlag, error) {
	var featureFlag models.FeatureFlag
	if err := s.db.Where("name = ?", name).First(&featureFlag).Error; err != nil {
		return nil, err
	}
	return &featureFlag, nil
}

// CreateFeatureFlag creates a new feature flag
func (s *service) CreateFeatureFlag(flag *models.FeatureFlag) error {
	return s.db.Create(flag).Error
}

// UpdateFeatureFlag updates an existing feature flag
func (s *service) UpdateFeatureFlag(flag *models.FeatureFlag) error {
	return s.db.Save(flag).Error
}

// DeleteFeatureFlag deletes a feature flag
func (s *service) DeleteFeatureFlag(id uint) error {
	// Start a transaction to ensure all operations succeed or fail together
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Delete associated roles first
		if err := tx.Where("feature_flag_id = ?", id).Delete(&models.FeatureFlagRole{}).Error; err != nil {
			return err
		}
		// Then delete the feature flag
		return tx.Delete(&models.FeatureFlag{}, id).Error
	})
}

// AddRoleToFeatureFlag adds a role to a feature flag
func (s *service) AddRoleToFeatureFlag(flagID uint, role string) error {
	return s.db.Model(&models.FeatureFlagRole{}).Create(&models.FeatureFlagRole{
		FeatureFlagID: flagID,
		Role:          role,
	}).Error
}

// RemoveRoleFromFeatureFlag removes a role from a feature flag
func (s *service) RemoveRoleFromFeatureFlag(flagID uint, role string) error {
	return s.db.Delete(&models.FeatureFlagRole{}, "feature_flag_id = ? AND role = ?", flagID, role).Error
}

// IsFeatureEnabled checks if a feature is enabled
func (s *service) IsFeatureEnabled(name string) (bool, error) {
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
func (s *service) CanUserAccessFeature(username, featureName string) (bool, error) {
	return models.CanAccessFeature(s.db, username, featureName)
}

// Casing-related methods implementation
// FindAllCasings retrieves all casings from the database
func (s *service) FindAllCasings() ([]models.Casing, error) {
	var casings []models.Casing
	// Order by Popularity descending, then by Type ascending for consistent ordering
	if err := s.db.Order("popularity DESC, type ASC").Find(&casings).Error; err != nil {
		return nil, err
	}
	return casings, nil
}

// CreateCasing creates a new casing record
func (s *service) CreateCasing(casing *models.Casing) error {
	// Check if there's a soft-deleted casing with the same type
	var existingCasing models.Casing
	result := s.db.Unscoped().Where("type = ?", casing.Type).First(&existingCasing)

	if result.Error == nil && existingCasing.DeletedAt.Valid {
		// Record exists and is soft-deleted, restore it
		existingCasing.DeletedAt.Valid = false // Clear the deleted_at timestamp
		existingCasing.DeletedAt.Time = time.Time{}
		existingCasing.Popularity = casing.Popularity // Update with new values

		// Update the existing record (restore it)
		return s.db.Unscoped().Save(&existingCasing).Error
	} else if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// Some error other than "record not found" occurred
		return result.Error
	}

	// No soft-deleted record found with this type, create a new one
	return s.db.Create(casing).Error
}

// FindCasingByID finds a casing by its ID using the models function
func (s *service) FindCasingByID(id uint) (*models.Casing, error) {
	return models.FindCasingByID(s.db, id)
}

// UpdateCasing updates an existing casing in the database
func (s *service) UpdateCasing(casing *models.Casing) error {
	return s.db.Save(casing).Error
}

// DeleteCasing deletes a casing from the database
func (s *service) DeleteCasing(id uint) error {
	return s.db.Delete(&models.Casing{}, id).Error
}

// BulletStyle-related methods implementation
// FindAllBulletStyles retrieves all bullet styles from the database
func (s *service) FindAllBulletStyles() ([]models.BulletStyle, error) {
	var bulletStyles []models.BulletStyle
	// Order by Popularity descending, then by Type ascending for consistent ordering
	if err := s.db.Order("popularity DESC, type ASC").Find(&bulletStyles).Error; err != nil {
		return nil, err
	}
	return bulletStyles, nil
}

// CreateBulletStyle creates a new bullet style record
func (s *service) CreateBulletStyle(bulletStyle *models.BulletStyle) error {
	// Check if there's a soft-deleted bullet style with the same type
	var existingBulletStyle models.BulletStyle
	result := s.db.Unscoped().Where("type = ?", bulletStyle.Type).First(&existingBulletStyle)

	if result.Error == nil && existingBulletStyle.DeletedAt.Valid {
		// Record exists and is soft-deleted, restore it
		existingBulletStyle.DeletedAt.Valid = false // Clear the deleted_at timestamp
		existingBulletStyle.DeletedAt.Time = time.Time{}
		existingBulletStyle.Nickname = bulletStyle.Nickname     // Update with new values
		existingBulletStyle.Popularity = bulletStyle.Popularity // Update with new values

		// Update the existing record (restore it)
		return s.db.Unscoped().Save(&existingBulletStyle).Error
	} else if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// Some error other than "record not found" occurred
		return result.Error
	}

	// No soft-deleted record found with this type, create a new one
	return s.db.Create(bulletStyle).Error
}

// FindBulletStyleByID retrieves a bullet style by its ID
func (s *service) FindBulletStyleByID(id uint) (*models.BulletStyle, error) {
	var bulletStyle models.BulletStyle
	if err := s.db.First(&bulletStyle, id).Error; err != nil {
		return nil, err
	}
	return &bulletStyle, nil
}

// UpdateBulletStyle updates an existing bullet style in the database
func (s *service) UpdateBulletStyle(bulletStyle *models.BulletStyle) error {
	return s.db.Save(bulletStyle).Error
}

// DeleteBulletStyle deletes a bullet style from the database
func (s *service) DeleteBulletStyle(id uint) error {
	return s.db.Delete(&models.BulletStyle{}, id).Error
}

// Grain-related methods implementation
// FindAllGrains retrieves all grains from the database
func (s *service) FindAllGrains() ([]models.Grain, error) {
	var grains []models.Grain
	// Order by Popularity descending, then by Weight ascending for consistent ordering
	if err := s.db.Order("popularity DESC, weight ASC").Find(&grains).Error; err != nil {
		return nil, err
	}
	return grains, nil
}

// CreateGrain creates a new grain record
func (s *service) CreateGrain(grain *models.Grain) error {
	// Check if there's a soft-deleted grain with the same weight
	var existingGrain models.Grain
	result := s.db.Unscoped().Where("weight = ?", grain.Weight).First(&existingGrain)

	if result.Error == nil && existingGrain.DeletedAt.Valid {
		// Record exists and is soft-deleted, restore it
		existingGrain.DeletedAt.Valid = false // Clear the deleted_at timestamp
		existingGrain.DeletedAt.Time = time.Time{}
		existingGrain.Popularity = grain.Popularity // Update with new values

		// Update the existing record (restore it)
		return s.db.Unscoped().Save(&existingGrain).Error
	} else if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// Some error other than "record not found" occurred
		return result.Error
	}

	// No soft-deleted record found with this weight, create a new one
	return s.db.Create(grain).Error
}

// FindGrainByID retrieves a grain by its ID
func (s *service) FindGrainByID(id uint) (*models.Grain, error) {
	var grain models.Grain
	if err := s.db.First(&grain, id).Error; err != nil {
		return nil, err
	}
	return &grain, nil
}

// UpdateGrain updates an existing grain in the database
func (s *service) UpdateGrain(grain *models.Grain) error {
	return s.db.Save(grain).Error
}

// DeleteGrain deletes a grain from the database
func (s *service) DeleteGrain(id uint) error {
	return s.db.Delete(&models.Grain{}, id).Error
}

// Brand-related methods implementation
// FindAllBrands retrieves all brands from the database
func (s *service) FindAllBrands() ([]models.Brand, error) {
	var brands []models.Brand
	if err := s.db.Find(&brands).Error; err != nil {
		return nil, err
	}
	return brands, nil
}

// CreateBrand creates a new brand record
func (s *service) CreateBrand(brand *models.Brand) error {
	return s.db.Create(brand).Error
}

// FindBrandByID retrieves a brand by its ID
func (s *service) FindBrandByID(id uint) (*models.Brand, error) {
	var brand models.Brand
	if err := s.db.First(&brand, id).Error; err != nil {
		return nil, err
	}
	return &brand, nil
}

// UpdateBrand updates an existing brand in the database
func (s *service) UpdateBrand(brand *models.Brand) error {
	return s.db.Save(brand).Error
}

// DeleteBrand deletes a brand from the database
func (s *service) DeleteBrand(id uint) error {
	return s.db.Delete(&models.Brand{}, id).Error
}

// Ammo-related methods implementation
// FindAllAmmo retrieves all ammo from the database
func (s *service) FindAllAmmo() ([]models.Ammo, error) {
	var ammo []models.Ammo
	if err := s.db.Find(&ammo).Error; err != nil {
		return nil, err
	}
	return ammo, nil
}

// FindAmmoByID retrieves ammo by its ID
func (s *service) FindAmmoByID(id uint) (*models.Ammo, error) {
	var ammo models.Ammo
	if err := s.db.First(&ammo, id).Error; err != nil {
		return nil, err
	}
	return &ammo, nil
}

// CountAmmoByUser retrieves the count of ammo for a specific user
func (s *service) CountAmmoByUser(userID uint) (int64, error) {
	var count int64
	if err := s.db.Model(&models.Ammo{}).Where("owner_id = ?", userID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
