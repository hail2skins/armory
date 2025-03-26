package database

import (
	"context"
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

	// Additional user methods
	GetUserByID(id uint) (*User, error)
	GetUserByStripeCustomerID(customerID string) (*User, error)

	// Gun-related methods
	DeleteGun(db *gorm.DB, id uint, ownerID uint) error

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
