package database

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTestDB creates a test database
func setupTestDB(t *testing.T) (*gorm.DB, string) {
	// Create a temporary directory for the SQLite database
	tempDir, err := os.MkdirTemp("", "auth-test-*")
	require.NoError(t, err)

	// Create a SQLite database in the temporary directory
	dbPath := filepath.Join(tempDir, "auth-test.db")

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

// AuthenticationTestSuite is a test suite for authentication-related functionality
type AuthenticationTestSuite struct {
	suite.Suite
	service  *service
	testUser *User
	cleanup  func()
}

// SetupSuite sets up the test suite
func (s *AuthenticationTestSuite) SetupSuite() {
	// Create a test database
	db, tempDir := setupTestDB(s.T())

	// Run migrations to ensure all required tables exist
	err := db.AutoMigrate(&User{})
	require.NoError(s.T(), err)

	// Initialize the service
	s.service = &service{db: db}

	// Setup cleanup function
	s.cleanup = func() {
		// Close the database connection
		sqlDB, err := db.DB()
		if err == nil {
			sqlDB.Close()
		}
		// Remove the test database file
		os.RemoveAll(tempDir)
	}

	// Create a test user with plain password
	// (the BeforeCreate hook will hash it automatically)
	s.testUser = &User{
		Email:    "test@example.com",
		Password: "password123", // Plain password - will be hashed by BeforeCreate
		Verified: true,          // User is verified
	}

	// Save the user
	err = db.Create(s.testUser).Error
	require.NoError(s.T(), err)

	// Verify user was saved correctly
	var savedUser User
	err = db.First(&savedUser, s.testUser.ID).Error
	require.NoError(s.T(), err)

	// Verify the password works after hashing by BeforeCreate
	require.True(s.T(), CheckPassword("password123", savedUser.Password),
		"Password verification failed after saving user")
}

// TearDownSuite cleans up after the test suite
func (s *AuthenticationTestSuite) TearDownSuite() {
	s.cleanup()
}

// TestResetLoginAttemptsOnSuccessfulLogin tests that login attempts are reset on successful login
func (s *AuthenticationTestSuite) TestResetLoginAttemptsOnSuccessfulLogin() {
	// Set up initial state with failed login attempts
	s.testUser.LoginAttempts = 3
	s.testUser.LastLoginAttempt = time.Now().Add(-10 * time.Minute)
	err := s.service.db.Save(s.testUser).Error
	require.NoError(s.T(), err)

	// Store the time before authentication
	beforeAuth := time.Now()

	// Verify we can find the user by email
	userCheck, err := s.service.GetUserByEmail(context.Background(), "test@example.com")
	require.NoError(s.T(), err)
	require.NotNil(s.T(), userCheck, "User should exist in the database before test")

	// Perform successful authentication
	user, err := s.service.AuthenticateUser(context.Background(), "test@example.com", "password123")
	require.NoError(s.T(), err)
	require.NotNil(s.T(), user, "User should not be nil after successful authentication")

	// Verify login attempts were reset
	assert.Equal(s.T(), 0, user.LoginAttempts, "Login attempts should be reset to 0 after successful login")

	// Verify LastLoginAttempt was updated to a recent time (not reset to zero)
	assert.False(s.T(), user.LastLoginAttempt.IsZero(), "LastLoginAttempt should be updated, not reset to zero time")
	assert.True(s.T(), user.LastLoginAttempt.After(beforeAuth) || user.LastLoginAttempt.Equal(beforeAuth),
		"LastLoginAttempt should be updated to the current time or after")

	// Verify changes were saved to the database
	var updatedUser User
	err = s.service.db.First(&updatedUser, s.testUser.ID).Error
	require.NoError(s.T(), err)

	assert.Equal(s.T(), 0, updatedUser.LoginAttempts, "Login attempts should be persisted as 0 in the database")
	assert.False(s.T(), updatedUser.LastLoginAttempt.IsZero(), "LastLoginAttempt should not be persisted as zero time")
	assert.True(s.T(), updatedUser.LastLoginAttempt.After(beforeAuth) || updatedUser.LastLoginAttempt.Equal(beforeAuth),
		"LastLoginAttempt should be persisted as the current time or after")
}

// TestTrackLastLoginTimeOnSuccessfulLogin tests that last login time is updated on successful login
func (s *AuthenticationTestSuite) TestTrackLastLoginTimeOnSuccessfulLogin() {
	// This test expects a LastLogin field to be added to the User model
	// It will fail until we implement that field in a later step
	s.T().Skip("LastLogin field not yet implemented - this test will be enabled later")

	// Ensure the user has a zero last login time initially
	err := s.service.db.Model(s.testUser).Update("last_login", time.Time{}).Error
	require.NoError(s.T(), err)

	// Store the time before authentication
	// beforeAuth := time.Now()

	// Perform successful authentication
	user, err := s.service.AuthenticateUser(context.Background(), "test@example.com", "password123")
	require.NoError(s.T(), err)
	require.NotNil(s.T(), user)

	// Verify last login time was updated to a recent timestamp
	// These assertions will be uncommented once the LastLogin field is added
	/*
		assert.False(s.T(), user.LastLogin.IsZero(), "LastLogin should not be zero after successful login")
		assert.True(s.T(), user.LastLogin.After(beforeAuth) || user.LastLogin.Equal(beforeAuth),
			"LastLogin should be set to the current time or after authentication time")

		// Verify changes were saved to the database
		var updatedUser User
		err = s.service.db.First(&updatedUser, s.testUser.ID).Error
		require.NoError(s.T(), err)

		assert.False(s.T(), updatedUser.LastLogin.IsZero(), "LastLogin should be persisted as current time in the database")
		assert.True(s.T(), updatedUser.LastLogin.After(beforeAuth) || updatedUser.LastLogin.Equal(beforeAuth),
			"LastLogin should be persisted as the current time or after authentication time")
	*/
}

// TestGenerateVerificationToken tests that verification tokens are generated properly
func (s *AuthenticationTestSuite) TestGenerateVerificationToken() {
	// Store the time before generating the token
	beforeGeneration := time.Now()

	// Generate a verification token
	token := s.testUser.GenerateVerificationToken()

	// Assert that the token is not empty
	assert.NotEmpty(s.T(), token, "Verification token should not be empty")
	assert.Equal(s.T(), token, s.testUser.VerificationToken, "Token should be stored in user.VerificationToken")

	// Verify that VerificationSentAt is set to the current time
	assert.False(s.T(), s.testUser.VerificationSentAt.IsZero(), "VerificationSentAt should be set")
	assert.True(s.T(), s.testUser.VerificationSentAt.After(beforeGeneration) ||
		s.testUser.VerificationSentAt.Equal(beforeGeneration),
		"VerificationSentAt should be set to the current time or after token generation")

	// Verify that the token expiry is set to 1 hour in the future
	expectedExpiry := s.testUser.VerificationSentAt.Add(1 * time.Hour)
	timeDiff := s.testUser.VerificationTokenExpiry.Sub(expectedExpiry)

	// Allow for a small time difference (less than 1 second) due to processing time
	assert.True(s.T(), timeDiff < time.Second && timeDiff > -time.Second,
		"VerificationTokenExpiry should be set to 1 hour after VerificationSentAt")
}

// Run the test suite
func TestAuthenticationSuite(t *testing.T) {
	suite.Run(t, new(AuthenticationTestSuite))
}
