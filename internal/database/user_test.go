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
	"golang.org/x/crypto/bcrypt"
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

func TestUser_Validate(t *testing.T) {
	tests := []struct {
		name    string
		user    User
		wantErr bool
		errType error // Expected error type if wantErr is true
	}{
		{
			name:    "Valid user",
			user:    User{Email: "test@example.com", Password: "ValidPassword123!"},
			wantErr: false,
		},
		{
			name:    "Invalid email",
			user:    User{Email: "invalid-email", Password: "ValidPassword123!"},
			wantErr: true,
			errType: ErrInvalidEmail,
		},
		{
			name:    "Invalid password - too short",
			user:    User{Email: "test@example.com", Password: "short"},
			wantErr: true,
			errType: ErrInvalidPassword,
		},
		{
			name:    "Invalid password - no uppercase",
			user:    User{Email: "test@example.com", Password: "nouppercase123!"},
			wantErr: true,
			errType: ErrInvalidPassword,
		},
		{
			name:    "Invalid password - no lowercase",
			user:    User{Email: "test@example.com", Password: "NOLOWERCASE123!"},
			wantErr: true,
			errType: ErrInvalidPassword,
		},
		{
			name:    "Invalid password - no digit",
			user:    User{Email: "test@example.com", Password: "NoDigitPassword!"},
			wantErr: true,
			errType: ErrInvalidPassword,
		},
		{
			name:    "Invalid password - no special char",
			user:    User{Email: "test@example.com", Password: "NoSpecialChar123"},
			wantErr: true,
			errType: ErrInvalidPassword,
		},
		{
			name:    "Password looks like hash - skip validation",
			user:    User{Email: "test@example.com", Password: "$2a$10$abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ012345"}, // Example bcrypt hash structure
			wantErr: false,                                                                                                          // Should skip password validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.errType) // Check if the error is of the expected type
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPasswordHashingAndChecking(t *testing.T) {
	password := "ValidPass123!"

	// Test hashing
	hash, err := HashPassword(password)
	require.NoError(t, err, "Hashing password should not produce an error")
	require.NotEmpty(t, hash, "Hashed password should not be empty")
	assert.NotEqual(t, password, hash, "Hashed password should be different from the original password")

	// Test checking with correct password
	match := CheckPassword(password, hash)
	assert.True(t, match, "CheckPassword should return true for the correct password")

	// Test checking with incorrect password
	incorrectPassword := "WrongPass123!"
	match = CheckPassword(incorrectPassword, hash)
	assert.False(t, match, "CheckPassword should return false for an incorrect password")
}

func TestTokenGenerationAndExpiry(t *testing.T) {
	user := &User{}
	now := time.Now()

	// Test Verification Token
	verificationToken := user.GenerateVerificationToken()
	assert.NotEmpty(t, verificationToken, "Verification token should not be empty")
	assert.Equal(t, verificationToken, user.VerificationToken, "Generated token should match user's token field")
	assert.WithinDuration(t, now.Add(1*time.Hour), user.VerificationTokenExpiry, 5*time.Second, "Verification token expiry should be approx 1 hour from now")
	assert.False(t, user.IsVerificationExpired(), "Verification token should not be expired immediately after generation")

	// Test expired verification token
	user.VerificationTokenExpiry = now.Add(-1 * time.Minute) // Set expiry to the past
	assert.True(t, user.IsVerificationExpired(), "Verification token should be expired when expiry is in the past")

	// Reset user fields for recovery test
	user.RecoveryToken = ""
	user.RecoveryTokenExpiry = time.Time{}
	now = time.Now() // Recapture 'now' for accurate timing

	// Test Recovery Token
	recoveryToken := user.GenerateRecoveryToken()
	assert.NotEmpty(t, recoveryToken, "Recovery token should not be empty")
	assert.Equal(t, recoveryToken, user.RecoveryToken, "Generated token should match user's token field")
	assert.WithinDuration(t, now.Add(1*time.Hour), user.RecoveryTokenExpiry, 5*time.Second, "Recovery token expiry should be approx 1 hour from now")
	assert.False(t, user.IsRecoveryExpired(), "Recovery token should not be expired immediately after generation")

	// Test expired recovery token
	user.RecoveryTokenExpiry = now.Add(-1 * time.Minute) // Set expiry to the past
	assert.True(t, user.IsRecoveryExpired(), "Recovery token should be expired when expiry is in the past")
}

func TestAccountLockoutLogic(t *testing.T) {
	user := &User{}

	// Initial state
	assert.False(t, user.IsLocked(), "User should not be locked initially")
	assert.Zero(t, user.LoginAttempts, "Login attempts should be 0 initially")

	// Increment attempts (less than lockout threshold)
	for i := 1; i < 5; i++ {
		user.IncrementLoginAttempts()
		assert.Equal(t, i, user.LoginAttempts, "Login attempts should increment correctly")
		assert.False(t, user.IsLocked(), "User should not be locked after %d attempts", i)
		time.Sleep(1 * time.Millisecond) // Ensure LastLoginAttempt timestamp changes slightly
	}

	// Fifth attempt - should trigger lockout
	user.IncrementLoginAttempts()
	assert.Equal(t, 5, user.LoginAttempts, "Login attempts should be 5")
	assert.True(t, user.IsLocked(), "User should be locked after 5 attempts")

	// Test lockout expiry
	user.LastLoginAttempt = time.Now().Add(-16 * time.Minute) // Set last attempt time to 16 mins ago
	assert.False(t, user.IsLocked(), "User should not be locked after 15 minutes have passed")

	// Test reset
	user.ResetLoginAttempts()
	assert.Zero(t, user.LoginAttempts, "Login attempts should be 0 after reset")
	assert.False(t, user.IsLocked(), "User should not be locked after reset")
}

func TestUserEmailVerification(t *testing.T) {
	// Case 1: Standard verification
	t.Run("Standard verification", func(t *testing.T) {
		user := &User{
			Email:             "initial@example.com",
			Verified:          false,
			VerificationToken: "some_token",
		}

		assert.False(t, user.IsVerified(), "User should not be verified initially")

		user.VerifyEmail()

		assert.True(t, user.IsVerified(), "User should be verified after calling VerifyEmail")
		assert.Empty(t, user.VerificationToken, "Verification token should be cleared after verification")
		assert.Equal(t, "initial@example.com", user.Email, "Email should remain unchanged")
		assert.Empty(t, user.PendingEmail, "PendingEmail should be empty")
	})

	// Case 2: Verification with pending email change
	t.Run("Verification with pending email", func(t *testing.T) {
		user := &User{
			Email:             "initial@example.com",
			Verified:          false,
			PendingEmail:      "new@example.com",
			VerificationToken: "another_token",
		}

		assert.False(t, user.IsVerified(), "User should not be verified initially")

		user.VerifyEmail()

		assert.True(t, user.IsVerified(), "User should be verified after calling VerifyEmail")
		assert.Empty(t, user.VerificationToken, "Verification token should be cleared after verification")
		assert.Equal(t, "new@example.com", user.Email, "Email should be updated to PendingEmail")
		assert.Empty(t, user.PendingEmail, "PendingEmail should be cleared after verification")
	})
}

func TestUserSetAndCheckPassword(t *testing.T) {
	user := &User{}
	plainPassword := "ValidPass123!@#"

	// Test SetPassword
	err := user.SetPassword(plainPassword)
	require.NoError(t, err, "SetPassword should not return an error")
	assert.NotEmpty(t, user.Password, "User password hash should not be empty after SetPassword")
	assert.NotEqual(t, plainPassword, user.Password, "User password hash should be different from plain password")

	// Test CheckPassword with correct password
	err = user.CheckPassword(plainPassword)
	assert.NoError(t, err, "CheckPassword should not return an error for the correct password")

	// Test CheckPassword with incorrect password
	incorrectPassword := "WrongPassword123!@#"
	err = user.CheckPassword(incorrectPassword)
	assert.Error(t, err, "CheckPassword should return an error for an incorrect password")
	assert.ErrorIs(t, err, bcrypt.ErrMismatchedHashAndPassword, "Error should be bcrypt.ErrMismatchedHashAndPassword for incorrect password")
}

func TestHasActiveSubscription(t *testing.T) {
	now := time.Now()
	futureDate := now.Add(24 * time.Hour)
	pastDate := now.Add(-24 * time.Hour)

	tests := []struct {
		name           string
		user           User
		expectedResult bool
	}{
		{
			name:           "Free tier",
			user:           User{SubscriptionTier: "free"},
			expectedResult: false,
		},
		{
			name:           "Lifetime tier",
			user:           User{SubscriptionTier: "lifetime", SubscriptionStatus: "inactive"}, // Status shouldn't matter
			expectedResult: true,
		},
		{
			name:           "Premium Lifetime tier",
			user:           User{SubscriptionTier: "premium_lifetime", SubscriptionStatus: "past_due"}, // Status shouldn't matter
			expectedResult: true,
		},
		{
			name:           "Monthly tier, active status, future end date",
			user:           User{SubscriptionTier: "monthly", SubscriptionStatus: "active", SubscriptionEndDate: futureDate},
			expectedResult: true,
		},
		{
			name:           "Yearly tier, active status, zero end date",
			user:           User{SubscriptionTier: "yearly", SubscriptionStatus: "active", SubscriptionEndDate: time.Time{}},
			expectedResult: true,
		},
		{
			name:           "Monthly tier, active status, past end date",
			user:           User{SubscriptionTier: "monthly", SubscriptionStatus: "active", SubscriptionEndDate: pastDate},
			expectedResult: false,
		},
		{
			name:           "Yearly tier, inactive status, future end date",
			user:           User{SubscriptionTier: "yearly", SubscriptionStatus: "inactive", SubscriptionEndDate: futureDate},
			expectedResult: false,
		},
		{
			name:           "Monthly tier, past_due status, future end date",
			user:           User{SubscriptionTier: "monthly", SubscriptionStatus: "past_due", SubscriptionEndDate: futureDate},
			expectedResult: false,
		},
		{
			name:           "Admin granted, IsLifetime true",
			user:           User{SubscriptionTier: "premium_lifetime", IsLifetime: true, SubscriptionStatus: "inactive"}, // Should still be active due to IsLifetime
			expectedResult: true,                                                                                         // Note: Current code doesn't explicitly check IsLifetime, relies on tier name. Test reflects code.
		},
		{
			name:           "Admin granted, IsLifetime false, but tier is lifetime",
			user:           User{SubscriptionTier: "lifetime", IsLifetime: false, SubscriptionStatus: "active"},
			expectedResult: true, // Active because tier name is lifetime
		},
		{
			name:           "Admin granted, IsLifetime false, tier is monthly, active, future date",
			user:           User{SubscriptionTier: "monthly", IsLifetime: false, SubscriptionStatus: "active", SubscriptionEndDate: futureDate},
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedResult, tt.user.HasActiveSubscription())
		})
	}
}

func TestHasStripeManagedSubscription(t *testing.T) {
	now := time.Now()
	futureDate := now.Add(24 * time.Hour)
	pastDate := now.Add(-24 * time.Hour)

	tests := []struct {
		name           string
		user           User
		expectedResult bool
	}{
		{
			name: "Active monthly Stripe subscription",
			user: User{
				SubscriptionTier:     "monthly",
				SubscriptionStatus:   "active",
				SubscriptionEndDate:  futureDate,
				StripeSubscriptionID: "sub_123",
			},
			expectedResult: true,
		},
		{
			name: "Pending cancellation Stripe subscription",
			user: User{
				SubscriptionTier:     "yearly",
				SubscriptionStatus:   "pending_cancellation",
				SubscriptionEndDate:  futureDate,
				StripeSubscriptionID: "sub_123",
			},
			expectedResult: true,
		},
		{
			name: "No Stripe subscription ID",
			user: User{
				SubscriptionTier:    "monthly",
				SubscriptionStatus:  "active",
				SubscriptionEndDate: futureDate,
			},
			expectedResult: false,
		},
		{
			name: "Promotion subscription should not be Stripe managed",
			user: User{
				SubscriptionTier:     "promotion",
				SubscriptionStatus:   "active",
				SubscriptionEndDate:  futureDate,
				StripeSubscriptionID: "",
			},
			expectedResult: false,
		},
		{
			name: "Expired subscription",
			user: User{
				SubscriptionTier:     "monthly",
				SubscriptionStatus:   "active",
				SubscriptionEndDate:  pastDate,
				StripeSubscriptionID: "sub_123",
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedResult, tt.user.HasStripeManagedSubscription())
		})
	}
}

func TestCanCancelStripeSubscription(t *testing.T) {
	now := time.Now()
	futureDate := now.Add(24 * time.Hour)

	tests := []struct {
		name           string
		user           User
		expectedResult bool
	}{
		{
			name: "Can cancel active Stripe subscription",
			user: User{
				SubscriptionTier:     "monthly",
				SubscriptionStatus:   "active",
				SubscriptionEndDate:  futureDate,
				StripeSubscriptionID: "sub_123",
			},
			expectedResult: true,
		},
		{
			name: "Cannot cancel when pending cancellation",
			user: User{
				SubscriptionTier:     "monthly",
				SubscriptionStatus:   "pending_cancellation",
				SubscriptionEndDate:  futureDate,
				StripeSubscriptionID: "sub_123",
			},
			expectedResult: false,
		},
		{
			name: "Cannot cancel without Stripe subscription",
			user: User{
				SubscriptionTier:    "monthly",
				SubscriptionStatus:  "active",
				SubscriptionEndDate: futureDate,
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedResult, tt.user.CanCancelStripeSubscription())
		})
	}
}

func TestCanSubscribeToTier(t *testing.T) {
	tests := []struct {
		name         string
		currentUser  User
		targetTier   string
		expectedBool bool
	}{
		// Free tier can subscribe to anything
		{name: "Free to Monthly", currentUser: User{SubscriptionTier: "free"}, targetTier: "monthly", expectedBool: true},
		{name: "Free to Yearly", currentUser: User{SubscriptionTier: "free"}, targetTier: "yearly", expectedBool: true},
		{name: "Free to Lifetime", currentUser: User{SubscriptionTier: "free"}, targetTier: "lifetime", expectedBool: true},
		{name: "Free to Premium Lifetime", currentUser: User{SubscriptionTier: "free"}, targetTier: "premium_lifetime", expectedBool: true},

		// Monthly tier
		{name: "Monthly to Monthly", currentUser: User{SubscriptionTier: "monthly"}, targetTier: "monthly", expectedBool: false},
		{name: "Monthly to Yearly", currentUser: User{SubscriptionTier: "monthly"}, targetTier: "yearly", expectedBool: true},
		{name: "Monthly to Lifetime", currentUser: User{SubscriptionTier: "monthly"}, targetTier: "lifetime", expectedBool: true},
		{name: "Monthly to Premium Lifetime", currentUser: User{SubscriptionTier: "monthly"}, targetTier: "premium_lifetime", expectedBool: true},
		{name: "Monthly to Free", currentUser: User{SubscriptionTier: "monthly"}, targetTier: "free", expectedBool: true}, // Current code allows this

		// Yearly tier
		{name: "Yearly to Monthly", currentUser: User{SubscriptionTier: "yearly"}, targetTier: "monthly", expectedBool: false},
		{name: "Yearly to Yearly", currentUser: User{SubscriptionTier: "yearly"}, targetTier: "yearly", expectedBool: false},
		{name: "Yearly to Lifetime", currentUser: User{SubscriptionTier: "yearly"}, targetTier: "lifetime", expectedBool: true},
		{name: "Yearly to Premium Lifetime", currentUser: User{SubscriptionTier: "yearly"}, targetTier: "premium_lifetime", expectedBool: true},
		{name: "Yearly to Free", currentUser: User{SubscriptionTier: "yearly"}, targetTier: "free", expectedBool: true}, // Current code allows this

		// Lifetime tier
		{name: "Lifetime to Monthly", currentUser: User{SubscriptionTier: "lifetime"}, targetTier: "monthly", expectedBool: false},
		{name: "Lifetime to Yearly", currentUser: User{SubscriptionTier: "lifetime"}, targetTier: "yearly", expectedBool: false},
		{name: "Lifetime to Lifetime", currentUser: User{SubscriptionTier: "lifetime"}, targetTier: "lifetime", expectedBool: false},
		{name: "Lifetime to Premium Lifetime", currentUser: User{SubscriptionTier: "lifetime"}, targetTier: "premium_lifetime", expectedBool: true},
		{name: "Lifetime to Free", currentUser: User{SubscriptionTier: "lifetime"}, targetTier: "free", expectedBool: false},

		// Premium Lifetime tier
		{name: "Premium Lifetime to Monthly", currentUser: User{SubscriptionTier: "premium_lifetime"}, targetTier: "monthly", expectedBool: false},
		{name: "Premium Lifetime to Yearly", currentUser: User{SubscriptionTier: "premium_lifetime"}, targetTier: "yearly", expectedBool: false},
		{name: "Premium Lifetime to Lifetime", currentUser: User{SubscriptionTier: "premium_lifetime"}, targetTier: "lifetime", expectedBool: false},
		{name: "Premium Lifetime to Premium Lifetime", currentUser: User{SubscriptionTier: "premium_lifetime"}, targetTier: "premium_lifetime", expectedBool: false},
		{name: "Premium Lifetime to Free", currentUser: User{SubscriptionTier: "premium_lifetime"}, targetTier: "free", expectedBool: false},

		// Default case (should behave like free)
		{name: "Unknown Tier to Monthly", currentUser: User{SubscriptionTier: "unknown"}, targetTier: "monthly", expectedBool: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedBool, tt.currentUser.CanSubscribeToTier(tt.targetTier))
		})
	}
}

func TestUserGetters(t *testing.T) {
	user := User{
		Model: gorm.Model{ID: 99},
		Email: "getter@example.com",
	}

	assert.Equal(t, uint(99), user.GetID(), "GetID should return the correct user ID")
	assert.Equal(t, "getter@example.com", user.GetUserName(), "GetUserName should return the correct user email")
}

func TestUserBeforeCreateHook(t *testing.T) {
	db, tempDir := setupUserTestDB(t)
	defer os.RemoveAll(tempDir)

	plainPassword := "HookMeUp123!"
	user := User{
		Email:    "hooktest@example.com",
		Password: plainPassword, // Set plain text password
	}

	// Create the user, triggering the BeforeCreate hook
	err := db.Create(&user).Error
	require.NoError(t, err, "Creating user should not produce an error")
	require.NotZero(t, user.ID, "User ID should be populated after creation")

	// Retrieve the user directly from the database
	var retrievedUser User
	err = db.First(&retrievedUser, user.ID).Error
	require.NoError(t, err, "Retrieving user from DB should not produce an error")

	// Verify the password stored in the database is hashed
	assert.NotEmpty(t, retrievedUser.Password, "Stored password should not be empty")
	assert.NotEqual(t, plainPassword, retrievedUser.Password, "Stored password should be hashed, not plain text")

	// Verify the stored hash matches the original plain password
	match := CheckPassword(plainPassword, retrievedUser.Password)
	assert.True(t, match, "Stored password hash should match the original password")
}

// Add tests for other User methods here...
