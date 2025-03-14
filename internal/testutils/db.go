package testutils

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/hail2skins/armory/internal/database"
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
	if err := db.AutoMigrate(&database.User{}); err != nil {
		log.Fatalf("Error auto migrating schema: %v", err)
	}

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

// NewTestService creates a new test database service
func NewTestService() database.Service {
	return &TestService{
		db: NewTestDB().DB,
	}
}

// TestService is a test implementation of the database.Service interface
type TestService struct {
	db *gorm.DB
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
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
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
		return nil, nil
	}

	// Check the password
	if !database.CheckPassword(password, user.Password) {
		return nil, nil
	}

	return user, nil
}

func (s *TestService) GetUserByRecoveryToken(ctx context.Context, token string) (*database.User, error) {
	var user database.User
	result := s.db.Where("recovery_token = ?", token).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, database.ErrInvalidToken
		}
		return nil, result.Error
	}
	return &user, nil
}

func (s *TestService) GetUserByVerificationToken(ctx context.Context, token string) (*database.User, error) {
	var user database.User
	result := s.db.Where("verification_token = ?", token).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, database.ErrInvalidToken
		}
		return nil, result.Error
	}
	return &user, nil
}

func (s *TestService) VerifyUserEmail(ctx context.Context, token string) (*database.User, error) {
	user, err := s.GetUserByVerificationToken(ctx, token)
	if err != nil {
		return nil, err
	}
	user.Verified = true
	user.VerificationToken = ""
	if err := s.UpdateUser(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *TestService) RequestPasswordReset(ctx context.Context, email string) (*database.User, error) {
	user, err := s.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return nil, err
	}
	user.RecoveryToken = base64.URLEncoding.EncodeToString(token)
	user.RecoveryTokenExpiry = time.Now().Add(1 * time.Hour)
	if err := s.UpdateUser(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *TestService) ResetPassword(ctx context.Context, token, newPassword string) error {
	user, err := s.GetUserByRecoveryToken(ctx, token)
	if err != nil {
		return err
	}
	if time.Now().After(user.RecoveryTokenExpiry) {
		return database.ErrTokenExpired
	}
	if err := user.SetPassword(newPassword); err != nil {
		return err
	}
	user.RecoveryToken = ""
	user.RecoveryTokenExpiry = time.Time{}
	return s.UpdateUser(ctx, user)
}

func (s *TestService) UpdateUser(ctx context.Context, user *database.User) error {
	return s.db.Save(user).Error
}

// CreatePayment creates a new payment record
func (s *TestService) CreatePayment(payment *database.Payment) error {
	return s.db.Create(payment).Error
}

// GetPaymentsByUserID retrieves all payments for a user
func (s *TestService) GetPaymentsByUserID(userID uint) ([]database.Payment, error) {
	var payments []database.Payment
	if err := s.db.Where("user_id = ?", userID).Order("created_at desc").Find(&payments).Error; err != nil {
		return nil, err
	}
	return payments, nil
}

// FindPaymentByID retrieves a payment by its ID
func (s *TestService) FindPaymentByID(id uint) (*database.Payment, error) {
	var payment database.Payment
	if err := s.db.First(&payment, id).Error; err != nil {
		return nil, err
	}
	return &payment, nil
}

// UpdatePayment updates an existing payment in the database
func (s *TestService) UpdatePayment(payment *database.Payment) error {
	return s.db.Save(payment).Error
}

// GetUserByID retrieves a user by their ID
func (s *TestService) GetUserByID(id uint) (*database.User, error) {
	var user database.User
	if err := s.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByStripeCustomerID retrieves a user by their Stripe customer ID
func (s *TestService) GetUserByStripeCustomerID(customerID string) (*database.User, error) {
	var user database.User
	if err := s.db.Where("stripe_customer_id = ?", customerID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}
