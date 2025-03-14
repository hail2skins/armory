package database

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// newTestDB creates a test database service with its own connection
func newTestDB(t *testing.T) *service {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	username := os.Getenv("DB_USERNAME")
	password := os.Getenv("DB_PASSWORD")
	database := os.Getenv("DB_DATABASE")
	schema := os.Getenv("DB_SCHEMA")
	if schema == "" {
		schema = "public"
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable search_path=%s",
		host, username, password, database, port, schema)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	svc := &service{db: db}
	if err := svc.AutoMigrate(); err != nil {
		t.Fatalf("Error auto migrating schema: %v", err)
	}

	return svc
}

func TestUserService(t *testing.T) {
	// Start a PostgreSQL container
	ctx := context.Background()
	postgresContainer, err := tcpostgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer postgresContainer.Terminate(ctx)

	// Get connection details
	host, err := postgresContainer.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := postgresContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatal(err)
	}

	// Set environment variables for the database connection
	t.Setenv("DB_HOST", host)
	t.Setenv("DB_PORT", port.Port())
	t.Setenv("DB_USERNAME", "postgres")
	t.Setenv("DB_PASSWORD", "postgres")
	t.Setenv("DB_DATABASE", "testdb")
	t.Setenv("DB_SCHEMA", "public")

	// Test creating a user
	t.Run("CreateUser", func(t *testing.T) {
		db := newTestDB(t)
		defer db.Close()

		user, err := db.CreateUser(ctx, "test@example.com", "password123")
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "test@example.com", user.Email)
		assert.NotEmpty(t, user.Password)
		assert.NotZero(t, user.ID)
		assert.False(t, user.CreatedAt.IsZero())
		assert.False(t, user.UpdatedAt.IsZero())
	})

	// Test getting a user by email
	t.Run("GetUserByEmail", func(t *testing.T) {
		db := newTestDB(t)
		defer db.Close()

		// Create a user first
		_, err := db.CreateUser(ctx, "get@example.com", "password123")
		assert.NoError(t, err)

		// Get the user
		user, err := db.GetUserByEmail(ctx, "get@example.com")
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "get@example.com", user.Email)

		// Try to get a non-existent user
		user, err = db.GetUserByEmail(ctx, "nonexistent@example.com")
		assert.NoError(t, err)
		assert.Nil(t, user)
	})

	// Test authenticating a user
	t.Run("AuthenticateUser", func(t *testing.T) {
		db := newTestDB(t)
		defer db.Close()

		// Create a user first
		_, err := db.CreateUser(ctx, "auth@example.com", "password123")
		assert.NoError(t, err)

		// Authenticate with correct credentials
		user, err := db.AuthenticateUser(ctx, "auth@example.com", "password123")
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "auth@example.com", user.Email)

		// Authenticate with incorrect password
		user, err = db.AuthenticateUser(ctx, "auth@example.com", "wrongpassword")
		assert.NoError(t, err)
		assert.Nil(t, user)

		// Authenticate with non-existent user
		user, err = db.AuthenticateUser(ctx, "nonexistent@example.com", "password123")
		assert.NoError(t, err)
		assert.Nil(t, user)
	})
}
