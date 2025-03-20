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
	// Ensure the user has a zero last login time initially
	err := s.service.db.Model(s.testUser).Update("last_login", time.Time{}).Error
	require.NoError(s.T(), err)

	// Store the time before authentication
	beforeAuth := time.Now()

	// Perform successful authentication
	user, err := s.service.AuthenticateUser(context.Background(), "test@example.com", "password123")
	require.NoError(s.T(), err)
	require.NotNil(s.T(), user)

	// Verify last login time was updated to a recent timestamp
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

// TestIncrementLoginAttemptsOnFailedLogin tests that login attempts are incremented on failed login
func (s *AuthenticationTestSuite) TestIncrementLoginAttemptsOnFailedLogin() {
	// Reset the login attempts to 0 initially
	s.testUser.LoginAttempts = 0
	s.testUser.LastLoginAttempt = time.Time{}
	err := s.service.db.Save(s.testUser).Error
	require.NoError(s.T(), err)

	// Store the time before authentication attempt
	beforeAuth := time.Now()

	// Perform failed authentication with wrong password
	user, err := s.service.AuthenticateUser(context.Background(), "test@example.com", "wrong_password")
	require.NoError(s.T(), err)
	assert.Nil(s.T(), user, "User should be nil after failed authentication")

	// Get the updated user from the database
	var updatedUser User
	err = s.service.db.First(&updatedUser, s.testUser.ID).Error
	require.NoError(s.T(), err)

	// Verify login attempts were incremented
	assert.Equal(s.T(), 1, updatedUser.LoginAttempts, "Login attempts should be incremented to 1 after failed login")

	// Verify LastLoginAttempt was updated to a recent time
	assert.False(s.T(), updatedUser.LastLoginAttempt.IsZero(), "LastLoginAttempt should be updated, not remain zero time")
	assert.True(s.T(), updatedUser.LastLoginAttempt.After(beforeAuth) || updatedUser.LastLoginAttempt.Equal(beforeAuth),
		"LastLoginAttempt should be updated to the current time or after")

	// Perform another failed authentication attempt
	user, err = s.service.AuthenticateUser(context.Background(), "test@example.com", "another_wrong_password")
	require.NoError(s.T(), err)
	assert.Nil(s.T(), user, "User should be nil after failed authentication")

	// Get the updated user again
	err = s.service.db.First(&updatedUser, s.testUser.ID).Error
	require.NoError(s.T(), err)

	// Verify login attempts were incremented again
	assert.Equal(s.T(), 2, updatedUser.LoginAttempts, "Login attempts should be incremented to 2 after second failed login")
}

// TestPreserveLastLoginOnFailedAttempt tests that LastLogin is preserved during failed login attempts
func (s *AuthenticationTestSuite) TestPreserveLastLoginOnFailedAttempt() {
	// Set initial LastLogin to a known non-zero time
	initialLastLogin := time.Now().Add(-24 * time.Hour) // 1 day ago
	err := s.service.db.Model(s.testUser).Update("last_login", initialLastLogin).Error
	require.NoError(s.T(), err)

	// Verify the initial state
	var initialUser User
	err = s.service.db.First(&initialUser, s.testUser.ID).Error
	require.NoError(s.T(), err)
	assert.False(s.T(), initialUser.LastLogin.IsZero(), "LastLogin should be non-zero initially")
	assert.Equal(s.T(), initialLastLogin.Unix(), initialUser.LastLogin.Unix(), "LastLogin should be set to our initial time")

	// Perform failed authentication with wrong password
	user, err := s.service.AuthenticateUser(context.Background(), "test@example.com", "wrong_password")
	require.NoError(s.T(), err)
	assert.Nil(s.T(), user, "User should be nil after failed authentication")

	// Get the updated user from the database
	var updatedUser User
	err = s.service.db.First(&updatedUser, s.testUser.ID).Error
	require.NoError(s.T(), err)

	// Verify LastLogin is preserved after a failed login attempt
	assert.False(s.T(), updatedUser.LastLogin.IsZero(), "LastLogin should not be reset to zero time after failed login")
	assert.Equal(s.T(), initialLastLogin.Unix(), updatedUser.LastLogin.Unix(),
		"LastLogin should remain unchanged after failed login attempt")
}

// TestGenerateRecoveryToken tests that recovery tokens are generated properly
func (s *AuthenticationTestSuite) TestGenerateRecoveryToken() {
	// Store the time before generating the token
	beforeGeneration := time.Now()

	// Generate a recovery token
	token := s.testUser.GenerateRecoveryToken()

	// Assert that the token is not empty
	assert.NotEmpty(s.T(), token, "Recovery token should not be empty")
	assert.Equal(s.T(), token, s.testUser.RecoveryToken, "Token should be stored in user.RecoveryToken")

	// Verify that RecoverySentAt is set to the current time
	assert.False(s.T(), s.testUser.RecoverySentAt.IsZero(), "RecoverySentAt should be set")
	assert.True(s.T(), s.testUser.RecoverySentAt.After(beforeGeneration) ||
		s.testUser.RecoverySentAt.Equal(beforeGeneration),
		"RecoverySentAt should be set to the current time or after token generation")

	// Verify that the token expiry is set to 1 hour in the future
	expectedExpiry := s.testUser.RecoverySentAt.Add(1 * time.Hour)
	timeDiff := s.testUser.RecoveryTokenExpiry.Sub(expectedExpiry)

	// Allow for a small time difference (less than 1 second) due to processing time
	assert.True(s.T(), timeDiff < time.Second && timeDiff > -time.Second,
		"RecoveryTokenExpiry should be set to 1 hour after RecoverySentAt")
}

// Run the test suite
func TestAuthenticationSuite(t *testing.T) {
	suite.Run(t, new(AuthenticationTestSuite))
}
