package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hail2skins/armory/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

// testDB holds the test database connection
var testDB *gorm.DB
var testDBPath string

// Test configuration: in-memory sqlite database for testing
func init() {
	if err := godotenv.Load(".env.test"); err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error loading .env.test file: %v", err)
		}
	}
	// Set default test environment variables
	if os.Getenv("DB_DRIVER") == "" {
		os.Setenv("DB_DRIVER", "sqlite")
	}
	if os.Getenv("DB_NAME") == "" {
		os.Setenv("DB_NAME", ":memory:")
	}
}

// NewContext creates a new context for testing
func NewContext() context.Context {
	return context.Background()
}

// NewTestGormDB creates a new GORM DB instance for testing
func NewTestGormDB() *gorm.DB {
	// Set up logging
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Silent,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	// Connect to in-memory SQLite database for testing
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: newLogger,
	})

	if err != nil {
		panic(fmt.Sprintf("failed to connect database: %v", err))
	}

	return db
}

// NewTestDatabase creates a fully initialized test database service
func NewTestDatabase() Service {
	// Create a new in-memory SQLite GORM DB
	db := NewTestGormDB()

	// Initialize the database schema
	initTestSchema(db)

	// Return the database service
	return &service{db: db}
}

// initTestSchema initializes the database schema for testing
func initTestSchema(db *gorm.DB) {
	// Auto-migrate all models
	if err := db.AutoMigrate(
		&User{},
		&models.WeaponType{},
		&models.Caliber{},
		&models.Manufacturer{},
		&models.Gun{},
		&models.Payment{},
	); err != nil {
		log.Fatalf("Failed to migrate test database: %v", err)
	}

	// Seed test data
	seedTestData(db)
}

// seedTestData adds minimal test data for testing
func seedTestData(db *gorm.DB) {
	// Create a test weapon type
	db.Create(&models.WeaponType{
		Type:       "Test Rifle",
		Nickname:   "Test Rifle",
		Popularity: 100,
	})

	// Create a test caliber
	db.Create(&models.Caliber{
		Caliber:    "Test 5.56",
		Nickname:   "Test 5.56",
		Popularity: 90,
	})

	// Create a test manufacturer
	db.Create(&models.Manufacturer{
		Name:       "Test Glock",
		Country:    "Test Country",
		Popularity: 80,
	})
}

// TestInMemoryDatabaseConnection checks the in-memory database connection
func TestInMemoryDatabaseConnection(t *testing.T) {
	// Arrange
	svc := NewTestDatabase()
	defer svc.Close()

	// Act
	health := svc.Health()

	// Assert
	assert.Equal(t, "up", health["status"])
}

// TestInMemoryGetDB retrieves the in-memory database instance
func TestInMemoryGetDB(t *testing.T) {
	// Arrange
	svc := NewTestDatabase()
	defer svc.Close()

	// Act
	db := svc.GetDB()

	// Assert
	assert.NotNil(t, db)
}

// TestInMemoryClose tests closing the in-memory database connection
func TestInMemoryClose(t *testing.T) {
	// Arrange
	svc := NewTestDatabase()

	// Act & Assert
	// This should not panic
	err := svc.Close()
	assert.NoError(t, err)
}

// TestInMemoryHealth checks the in-memory database health
func TestInMemoryHealth(t *testing.T) {
	// Arrange
	svc := NewTestDatabase()
	defer svc.Close()

	// Act
	health := svc.Health()

	// Assert
	assert.Equal(t, "up", health["status"])
}

// prepareTestDB sets up a SQLite database for testing
func prepareTestDB(t *testing.T) {
	if testDB != nil {
		return
	}

	// Create a temporary file for the test database
	tempDir := os.TempDir()
	testDBPath = filepath.Join(tempDir, "armory_test.db")

	// Remove any existing test database
	os.Remove(testDBPath)

	var err error
	testDB, err = gorm.Open(sqlite.Open(testDBPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})

	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Auto-migrate the test database
	err = testDB.AutoMigrate(
		&User{},
		&models.WeaponType{},
		&models.Caliber{},
		&models.Manufacturer{},
		&models.Gun{},
		&models.Payment{},
	)

	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}
}

// cleanupTestDB cleans up the test database
func cleanupTestDB() {
	if testDB != nil {
		// Close the database connection
		db, err := testDB.DB()
		if err == nil {
			db.Close()
		}
		testDB = nil

		// Remove the test database file
		os.Remove(testDBPath)
	}
}

// TestDatabaseOperations tests the database operations
func TestDatabaseOperations(t *testing.T) {
	prepareTestDB(t)
	defer cleanupTestDB()

	// Create a new database service using the test database
	svc := &service{db: testDB}

	// Test health check
	health := svc.Health()
	assert.Equal(t, "up", health["status"])

	// Test user operations
	ctx := NewContext()
	user, err := svc.CreateUser(ctx, "test@example.com", "password123")
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "test@example.com", user.Email)

	userByEmail, err := svc.GetUserByEmail(ctx, "test@example.com")
	assert.NoError(t, err)
	assert.NotNil(t, userByEmail)
	assert.Equal(t, user.ID, userByEmail.ID)

	// Test user authentication
	authUser, err := svc.AuthenticateUser(ctx, "test@example.com", "password123")
	assert.NoError(t, err)
	assert.NotNil(t, authUser)
	assert.Equal(t, user.ID, authUser.ID)

	// Test user not found
	noUser, err := svc.GetUserByEmail(ctx, "nonexistent@example.com")
	assert.NoError(t, err)
	assert.Nil(t, noUser)

	// Test failed authentication
	failedAuth, err := svc.AuthenticateUser(ctx, "test@example.com", "wrongpassword")
	assert.NoError(t, err)
	assert.Nil(t, failedAuth)
}

// TestHealth tests the Health method
func TestHealth(t *testing.T) {
	prepareTestDB(t)
	defer cleanupTestDB()

	// Create a new database service using the test database
	svc := &service{db: testDB}

	// Test health check
	health := svc.Health()
	assert.Equal(t, "up", health["status"])
}

// TestClose tests the Close method
func TestClose(t *testing.T) {
	prepareTestDB(t)
	defer cleanupTestDB()

	// Create a new database service using the test database
	svc := &service{db: testDB}

	// Test close
	err := svc.Close()
	assert.NoError(t, err)
}
