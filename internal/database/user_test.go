package database

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) (*gorm.DB, *tcpostgres.PostgresContainer) {
	ctx := context.Background()

	container, err := tcpostgres.RunContainer(ctx,
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
	require.NoError(t, err)

	// Get the container's host and port
	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err)

	// Construct the database URL
	dbURL := fmt.Sprintf("postgres://postgres:postgres@%s:%s/testdb?sslmode=disable", host, port.Port())

	// Connect to the database
	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	require.NoError(t, err)

	// Run migrations
	err = db.AutoMigrate(&User{})
	require.NoError(t, err)

	return db, container
}

func TestUserVerificationFields(t *testing.T) {
	// Create a test database
	db, container := setupTestDB(t)
	defer container.Terminate(context.Background())

	// Create a test user
	user := &User{
		Email:    "test@example.com",
		Password: "password123",
	}

	// Save the user
	err := db.Create(user).Error
	require.NoError(t, err)

	// Test verification token
	token := make([]byte, 32)
	_, err = rand.Read(token)
	require.NoError(t, err)
	verificationToken := base64.URLEncoding.EncodeToString(token)

	user.VerificationToken = verificationToken
	user.VerificationTokenExpiry = time.Now().Add(24 * time.Hour)
	err = db.Save(user).Error
	require.NoError(t, err)

	// Test recovery token
	_, err = rand.Read(token)
	require.NoError(t, err)
	recoveryToken := base64.URLEncoding.EncodeToString(token)

	user.RecoveryToken = recoveryToken
	user.RecoveryTokenExpiry = time.Now().Add(1 * time.Hour)
	err = db.Save(user).Error
	require.NoError(t, err)

	// Test verification status
	user.Verified = true
	user.VerificationToken = ""
	user.VerificationTokenExpiry = time.Time{}
	err = db.Save(user).Error
	require.NoError(t, err)

	// Test login attempts
	user.LoginAttempts = 1
	user.LastLoginAttempt = time.Now()
	err = db.Save(user).Error
	require.NoError(t, err)

	// Retrieve the user and verify all fields
	var retrievedUser User
	err = db.First(&retrievedUser, user.ID).Error
	require.NoError(t, err)

	assert.Equal(t, true, retrievedUser.Verified)
	assert.Empty(t, retrievedUser.VerificationToken)
	assert.True(t, retrievedUser.VerificationTokenExpiry.IsZero())
	assert.Equal(t, recoveryToken, retrievedUser.RecoveryToken)
	assert.False(t, retrievedUser.RecoveryTokenExpiry.IsZero())
	assert.Equal(t, 1, retrievedUser.LoginAttempts)
	assert.False(t, retrievedUser.LastLoginAttempt.IsZero())
}
