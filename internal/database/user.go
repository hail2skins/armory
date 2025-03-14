package database

import (
	"errors"
	"time"

	"crypto/rand"
	"encoding/base64"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrTokenExpired       = errors.New("token expired")
	ErrUserLocked         = errors.New("user account is locked")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// User represents a user in the system
type User struct {
	gorm.Model
	Email                   string `gorm:"uniqueIndex;not null"`
	Password                string `gorm:"not null"`
	Verified                bool   `gorm:"default:false"`
	VerificationToken       string
	VerificationTokenExpiry time.Time
	VerificationSentAt      time.Time
	RecoveryToken           string
	RecoveryTokenExpiry     time.Time
	RecoverySentAt          time.Time
	LoginAttempts           int `gorm:"default:0"`
	LastLoginAttempt        time.Time
}

// BeforeCreate is a GORM hook that hashes the password before creating a user
func (u *User) BeforeCreate(tx *gorm.DB) error {
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
	u.VerificationTokenExpiry = time.Now().Add(24 * time.Hour)
	return u.VerificationToken
}

// GenerateRecoveryToken generates a password recovery token and sets expiry
func (u *User) GenerateRecoveryToken() string {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return ""
	}
	u.RecoveryToken = base64.URLEncoding.EncodeToString(token)
	u.RecoveryTokenExpiry = time.Now().Add(1 * time.Hour)
	return u.RecoveryToken
}

// VerifyEmail marks the user's email as verified
func (u *User) VerifyEmail() {
	u.Verified = true
	u.VerificationToken = ""
}

// IncrementLoginAttempts increments the login attempt counter and updates the timestamp
func (u *User) IncrementLoginAttempts() {
	u.LoginAttempts++
	u.LastLoginAttempt = time.Now()
}

// ResetLoginAttempts resets the login attempt counter and clears the timestamp
func (u *User) ResetLoginAttempts() {
	u.LoginAttempts = 0
	u.LastLoginAttempt = time.Time{}
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
