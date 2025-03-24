package database

import (
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupUserTestDB(t *testing.T) (*gorm.DB, string) {
	// Create a temporary directory for the SQLite database
	tempDir, err := os.MkdirTemp("", "user-test-*")
	require.NoError(t, err)

	// Create a SQLite database in the temporary directory
	dbPath := filepath.Join(tempDir, "user-test.db")

	// Open connection to database
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Run migrations
	err = db.AutoMigrate(&User{})
	require.NoError(t, err)

	return db, tempDir
}

func TestUserVerificationFields(t *testing.T) {
	// Create a test database
	db, tempDir := setupUserTestDB(t)
	defer os.RemoveAll(tempDir)

	// Create a test user
	user := &User{
		Email:    "test@example.com",
		Password: "Password123!",
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
