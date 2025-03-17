package database

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/hail2skins/armory/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testDB holds the test database connection
var testDB *gorm.DB
var testDBPath string

// prepareTestDB sets up a SQLite database for testing
func prepareTestDB() error {
	// Create a temporary directory for the SQLite database
	tempDir, err := os.MkdirTemp("", "armory-db-test-*")
	if err != nil {
		return err
	}

	// Create a SQLite database in the temporary directory
	testDBPath = filepath.Join(tempDir, "test.db")

	// Configure GORM logger
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			LogLevel: logger.Silent, // Silent logging for tests
		},
	)

	// Open connection to database
	db, err := gorm.Open(sqlite.Open(testDBPath), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return err
	}

	testDB = db

	// Auto migrate tables - only include concrete structs
	err = db.AutoMigrate(
		&User{}, // The concrete User struct from database package
		&models.Payment{},
	)
	if err != nil {
		return err
	}

	return nil
}

// cleanupTestDB cleans up the test database
func cleanupTestDB() {
	// Close connection
	if testDB != nil {
		sqlDB, err := testDB.DB()
		if err == nil {
			sqlDB.Close()
		}
	}

	// Remove temp file
	if testDBPath != "" {
		os.Remove(testDBPath)
	}
}

func TestMain(m *testing.M) {
	// Setup test database
	if err := prepareTestDB(); err != nil {
		log.Printf("Failed to setup test database: %v", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	cleanupTestDB()

	os.Exit(code)
}

// TestPaymentMethods tests all the payment-related methods
func TestPaymentMethods(t *testing.T) {
	// Create a separate test DB for this test to avoid interference with other tests
	tempDir, err := os.MkdirTemp("", "payment-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "payment-test.db")

	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			LogLevel: logger.Silent,
		},
	)

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Auto migrate tables
	err = db.AutoMigrate(
		&User{},
		&models.Payment{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate tables: %v", err)
	}

	// Create a test service with our dedicated DB
	testSrv := &service{db: db}

	// Create test user
	testUser := &User{
		Email:    "test@example.com",
		Password: "password",
	}
	if err := testSrv.db.Create(testUser).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Test CreatePayment
	payment := &models.Payment{
		UserID:      testUser.ID,
		Amount:      1000, // $10.00
		Currency:    "usd",
		PaymentType: "test",
		Status:      "succeeded",
		Description: "Test payment",
		StripeID:    "test_stripe_id",
	}

	err = testSrv.CreatePayment(payment)
	if err != nil {
		t.Fatalf("CreatePayment failed: %v", err)
	}

	// Test GetPaymentsByUserID
	payments, err := testSrv.GetPaymentsByUserID(testUser.ID)
	if err != nil {
		t.Fatalf("GetPaymentsByUserID failed: %v", err)
	}
	if len(payments) != 1 {
		t.Fatalf("Expected 1 payment, got %d", len(payments))
	}

	// Test FindPaymentByID
	foundPayment, err := testSrv.FindPaymentByID(payment.ID)
	if err != nil {
		t.Fatalf("FindPaymentByID failed: %v", err)
	}
	if foundPayment.ID != payment.ID {
		t.Fatalf("Expected payment ID %d, got %d", payment.ID, foundPayment.ID)
	}

	// Test UpdatePayment
	payment.Description = "Updated test payment"
	err = testSrv.UpdatePayment(payment)
	if err != nil {
		t.Fatalf("UpdatePayment failed: %v", err)
	}

	// Verify the update
	updatedPayment, err := testSrv.FindPaymentByID(payment.ID)
	if err != nil {
		t.Fatalf("FindPaymentByID after update failed: %v", err)
	}
	if updatedPayment.Description != "Updated test payment" {
		t.Fatalf("Expected description 'Updated test payment', got '%s'", updatedPayment.Description)
	}
}

func TestNew(t *testing.T) {
	// Replace global db instance with our test one
	oldDBInstance := dbInstance
	dbInstance = &service{db: testDB}
	defer func() { dbInstance = oldDBInstance }()

	srv := New()
	if srv == nil {
		t.Fatal("New() returned nil")
	}
}

func TestHealth(t *testing.T) {
	// Replace global db instance with our test one
	oldDBInstance := dbInstance
	dbInstance = &service{db: testDB}
	defer func() { dbInstance = oldDBInstance }()

	srv := New()

	stats := srv.Health()

	if stats["status"] != "up" {
		t.Fatalf("expected status to be up, got %s", stats["status"])
	}
}

func TestClose(t *testing.T) {
	// Replace global db instance with our test one
	oldDBInstance := dbInstance
	dbInstance = &service{db: testDB}
	defer func() { dbInstance = oldDBInstance }()

	srv := New()

	if srv.Close() != nil {
		t.Fatalf("expected Close() to return nil")
	}
}
