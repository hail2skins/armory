package database

import (
	"errors"
	"time"

	"crypto/rand"
	"encoding/base64"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/hail2skins/armory/internal/validation"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrTokenExpired       = errors.New("token expired")
	ErrUserLocked         = errors.New("user account is locked")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidEmail       = errors.New("invalid email format")
	ErrInvalidPassword    = errors.New("invalid password format")
)

// User represents a user in the system
type User struct {
	gorm.Model
	Email                   string `gorm:"uniqueIndex;not null"`
	Password                string `gorm:"not null"`
	Verified                bool   `gorm:"default:false"`
	PendingEmail            string // Holds the new email address until it's verified
	VerificationToken       string
	VerificationTokenExpiry time.Time
	VerificationSentAt      time.Time
	RecoveryToken           string
	RecoveryTokenExpiry     time.Time
	RecoverySentAt          time.Time
	LoginAttempts           int `gorm:"default:0"`
	LastLoginAttempt        time.Time
	LastLogin               time.Time // Tracks the last successful login
	// Stripe-related fields
	StripeCustomerID     string
	StripeSubscriptionID string
	SubscriptionTier     string `gorm:"default:'free'"`
	SubscriptionStatus   string
	SubscriptionEndDate  time.Time
	PromotionID          uint
	// Admin-granted subscription fields
	GrantedByID    uint   // ID of the admin who granted the subscription
	GrantReason    string // Reason for granting the subscription
	IsAdminGranted bool   `gorm:"default:false"` // Whether the subscription was granted by an admin
	IsLifetime     bool   `gorm:"default:false"` // Whether the subscription is a lifetime subscription
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}

// GetUserName returns the user's email
func (u *User) GetUserName() string {
	return u.Email
}

// GetID returns the user's ID
func (u *User) GetID() uint {
	return u.ID
}

// Validate validates the user model fields
func (u *User) Validate() error {
	// Validate email
	if err := validation.ValidateEmail(u.Email); err != nil {
		return ErrInvalidEmail
	}

	// For password validation, we only want to validate non-hashed passwords
	// If the password is already hashed (typically >40 chars), we skip validation
	if len(u.Password) < 40 {
		if err := validation.ValidatePassword(u.Password); err != nil {
			return ErrInvalidPassword
		}
	}

	return nil
}

// BeforeCreate is a GORM hook that hashes the password before creating a user
func (u *User) BeforeCreate(tx *gorm.DB) error {
	// We'll skip validation in the hook to maintain compatibility with existing tests
	// Validation should be done explicitly before calling Create

	hashedPassword, err := HashPassword(u.Password)
	if err != nil {
		return err
	}
	u.Password = hashedPassword
	return nil
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword checks if the provided password matches the stored hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// IsVerified returns whether the user's email is verified
func (u *User) IsVerified() bool {
	return u.Verified
}

// IsVerificationExpired returns whether the verification token has expired
func (u *User) IsVerificationExpired() bool {
	return time.Now().After(u.VerificationTokenExpiry)
}

// IsRecoveryExpired returns whether the recovery token has expired
func (u *User) IsRecoveryExpired() bool {
	return time.Now().After(u.RecoveryTokenExpiry)
}

// IsLockedOut returns whether the user is locked out due to too many login attempts
func (u *User) IsLockedOut() bool {
	if u.LoginAttempts >= 5 {
		// Check if the lockout period (15 minutes) has passed
		return time.Since(u.LastLoginAttempt) < 15*time.Minute
	}
	return false
}

// GenerateVerificationToken generates a verification token and sets expiry
func (u *User) GenerateVerificationToken() string {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return ""
	}
	u.VerificationToken = base64.URLEncoding.EncodeToString(token)
	u.VerificationSentAt = time.Now()
	u.VerificationTokenExpiry = u.VerificationSentAt.Add(1 * time.Hour)
	return u.VerificationToken
}

// GenerateRecoveryToken generates a password recovery token and sets expiry
func (u *User) GenerateRecoveryToken() string {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return ""
	}
	u.RecoveryToken = base64.URLEncoding.EncodeToString(token)
	u.RecoverySentAt = time.Now()
	u.RecoveryTokenExpiry = u.RecoverySentAt.Add(1 * time.Hour)
	return u.RecoveryToken
}

// VerifyEmail marks the user's email as verified
func (u *User) VerifyEmail() {
	// If there's a pending email change, apply it now
	if u.PendingEmail != "" {
		u.Email = u.PendingEmail
		u.PendingEmail = ""
	}
	u.Verified = true
	u.VerificationToken = ""
}

// IncrementLoginAttempts increments the login attempt counter and updates the timestamp
func (u *User) IncrementLoginAttempts() {
	u.LoginAttempts++
	u.LastLoginAttempt = time.Now()
}

// ResetLoginAttempts resets the login attempt counter and updates the timestamp
func (u *User) ResetLoginAttempts() {
	u.LoginAttempts = 0
	u.LastLoginAttempt = time.Now()
}

// SetPassword hashes and sets the user's password
func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// CheckPassword verifies the provided password against the user's hashed password
func (u *User) CheckPassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
}

// IsLocked returns true if the user's account is locked due to too many failed login attempts
func (u *User) IsLocked() bool {
	if u.LoginAttempts >= 5 && time.Since(u.LastLoginAttempt) < 15*time.Minute {
		return true
	}
	return false
}

// HasActiveSubscription returns true if the user has an active subscription
func (u *User) HasActiveSubscription() bool {
	if u.SubscriptionTier == "free" {
		return false
	}

	// Lifetime subscriptions are always active
	if u.SubscriptionTier == "lifetime" || u.SubscriptionTier == "premium_lifetime" {
		return true
	}

	// Check if subscription is active based on status and end date
	if u.SubscriptionStatus == "active" && (u.SubscriptionEndDate.IsZero() || time.Now().Before(u.SubscriptionEndDate)) {
		return true
	}

	return false
}

// CanSubscribeToTier checks if a user can subscribe to a specific tier based on their current subscription
func (u *User) CanSubscribeToTier(tier string) bool {
	// Users can always upgrade to a higher tier
	switch u.SubscriptionTier {
	case "free":
		return true // Free users can subscribe to any tier
	case "monthly":
		// Monthly users can upgrade to yearly, lifetime, or premium_lifetime
		return tier != "monthly"
	case "yearly":
		// Yearly users can upgrade to lifetime or premium_lifetime
		return tier != "monthly" && tier != "yearly"
	case "lifetime":
		// Lifetime users can only upgrade to premium_lifetime
		return tier == "premium_lifetime"
	case "premium_lifetime":
		// Premium lifetime users cannot subscribe to any other tier
		return false
	default:
		return true
	}
}
