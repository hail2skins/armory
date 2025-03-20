package database

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// UserService defines the interface for user-related operations
type UserService interface {
	// CreateUser creates a new user with the given email and password
	CreateUser(ctx context.Context, email, password string) (*User, error)

	// GetUserByEmail retrieves a user by their email
	GetUserByEmail(ctx context.Context, email string) (*User, error)

	// AuthenticateUser authenticates a user with the given email and password
	AuthenticateUser(ctx context.Context, email, password string) (*User, error)

	// VerifyUserEmail verifies a user's email with the given token
	VerifyUserEmail(ctx context.Context, token string) (*User, error)

	// GetUserByVerificationToken gets a user by their verification token
	GetUserByVerificationToken(ctx context.Context, token string) (*User, error)

	// GetUserByRecoveryToken gets a user by their recovery token
	GetUserByRecoveryToken(ctx context.Context, token string) (*User, error)

	// UpdateUser updates a user's information
	UpdateUser(ctx context.Context, user *User) error

	// RequestPasswordReset initiates a password reset for a user
	RequestPasswordReset(ctx context.Context, email string) (*User, error)

	// ResetPassword resets a user's password using a recovery token
	ResetPassword(ctx context.Context, token, newPassword string) error

	// GetUserByID retrieves a user by their ID
	GetUserByID(id uint) (*User, error)

	// GetUserByStripeCustomerID retrieves a user by their Stripe customer ID
	GetUserByStripeCustomerID(customerID string) (*User, error)

	// IsRecoveryExpired checks if a recovery token is expired
	IsRecoveryExpired(ctx context.Context, token string) (bool, error)

	// CountUsers returns the total number of users in the database
	CountUsers() (int64, error)

	// FindRecentUsers returns a list of recent users with pagination and sorting
	FindRecentUsers(offset, limit int, sortBy, sortOrder string) ([]User, error)
}

// Ensure service implements UserService
var _ UserService = (*service)(nil)

// CreateUser creates a new user with the given email and password
func (s *service) CreateUser(ctx context.Context, email, password string) (*User, error) {
	// Create a new user
	user := &User{
		Email:    email,
		Password: password,
	}

	// Save the user to the database
	result := s.db.WithContext(ctx).Create(user)
	if result.Error != nil {
		return nil, result.Error
	}

	return user, nil
}

// GetUserByEmail retrieves a user by their email
func (s *service) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	result := s.db.WithContext(ctx).Where("email = ?", email).First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil // User not found
		}
		return nil, result.Error
	}

	return &user, nil
}

// AuthenticateUser authenticates a user with the given email and password
func (s *service) AuthenticateUser(ctx context.Context, email, password string) (*User, error) {
	// Get the user by email
	user, err := s.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	// User not found
	if user == nil {
		return nil, nil
	}

	// Check the password
	if !CheckPassword(password, user.Password) {
		// On failed login attempts, increment the counter
		user.IncrementLoginAttempts()
		if err := s.UpdateUser(ctx, user); err != nil {
			return nil, err
		}
		return nil, nil // Password doesn't match
	}

	// On successful login, reset the login attempts counter
	user.ResetLoginAttempts()
	if err := s.UpdateUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByVerificationToken retrieves a user by their verification token
func (s *service) GetUserByVerificationToken(ctx context.Context, token string) (*User, error) {
	var user User
	result := s.db.WithContext(ctx).Where("verification_token = ?", token).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &user, nil
}

// GetUserByRecoveryToken retrieves a user by their recovery token
func (s *service) GetUserByRecoveryToken(ctx context.Context, token string) (*User, error) {
	var user User
	result := s.db.WithContext(ctx).Where("recovery_token = ?", token).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &user, nil
}

// VerifyUserEmail verifies a user's email with the given token
func (s *service) VerifyUserEmail(ctx context.Context, token string) (*User, error) {
	user, err := s.GetUserByVerificationToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	if user.IsVerificationExpired() {
		return nil, errors.New("verification token has expired")
	}

	user.VerifyEmail()
	if err := s.UpdateUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// UpdateUser updates a user's information
func (s *service) UpdateUser(ctx context.Context, user *User) error {
	return s.db.WithContext(ctx).Save(user).Error
}

// RequestPasswordReset initiates a password reset for a user
func (s *service) RequestPasswordReset(ctx context.Context, email string) (*User, error) {
	user, err := s.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	user.GenerateRecoveryToken()
	if err := s.UpdateUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// ResetPassword resets a user's password using a recovery token
func (s *service) ResetPassword(ctx context.Context, token, newPassword string) error {
	user, err := s.GetUserByRecoveryToken(ctx, token)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrInvalidToken
	}

	if user.IsRecoveryExpired() {
		return ErrTokenExpired
	}

	hashedPassword, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	user.Password = hashedPassword
	user.RecoveryToken = ""
	user.RecoveryTokenExpiry = time.Time{}
	user.RecoverySentAt = time.Time{}

	return s.UpdateUser(ctx, user)
}

// GetUserByID retrieves a user by their ID
func (s *service) GetUserByID(id uint) (*User, error) {
	var user User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByStripeCustomerID retrieves a user by their Stripe customer ID
func (s *service) GetUserByStripeCustomerID(customerID string) (*User, error) {
	var user User
	if err := s.db.Where("stripe_customer_id = ?", customerID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// IsRecoveryExpired checks if a recovery token is expired
func (s *service) IsRecoveryExpired(ctx context.Context, token string) (bool, error) {
	user, err := s.GetUserByRecoveryToken(ctx, token)
	if err != nil {
		return true, err
	}
	return user.IsRecoveryExpired(), nil
}

// CountUsers returns the total number of users in the database
func (s *service) CountUsers() (int64, error) {
	var count int64
	result := s.db.Model(&User{}).Count(&count)
	return count, result.Error
}

// FindRecentUsers returns a list of recent users with pagination and sorting
func (s *service) FindRecentUsers(offset, limit int, sortBy, sortOrder string) ([]User, error) {
	var users []User
	result := s.db.Model(&User{}).
		Order(sortBy + " " + sortOrder).
		Offset(offset).
		Limit(limit).
		Find(&users)
	return users, result.Error
}
