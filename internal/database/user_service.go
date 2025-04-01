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

	// CountActiveSubscribers returns the number of users with active paid subscriptions
	CountActiveSubscribers() (int64, error)

	// CountNewUsersThisMonth returns the number of users registered in the current month
	CountNewUsersThisMonth() (int64, error)

	// CountNewUsersLastMonth returns the number of users registered in the previous month
	CountNewUsersLastMonth() (int64, error)

	// CountNewSubscribersThisMonth returns the number of new subscriptions in the current month
	CountNewSubscribersThisMonth() (int64, error)

	// CountNewSubscribersLastMonth returns the number of new subscriptions in the previous month
	CountNewSubscribersLastMonth() (int64, error)

	// CheckExpiredPromotionSubscription checks if a user's subscription has expired
	// and updates the status to "expired" if it has.
	// It also resets the subscription tier to "free" and clears the expiration date.
	// Returns true if the subscription status was updated, false otherwise.
	CheckExpiredPromotionSubscription(user *User) (bool, error)
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

		// Only update the specific fields related to login attempts
		// This avoids resetting the LastLogin field
		if err := s.db.WithContext(ctx).Model(user).
			Updates(map[string]interface{}{
				"login_attempts":     user.LoginAttempts,
				"last_login_attempt": user.LastLoginAttempt,
			}).Error; err != nil {
			return nil, err
		}

		return nil, nil // Password doesn't match
	}

	// On successful login, reset the login attempts counter and update last login time
	user.ResetLoginAttempts()
	user.LastLogin = time.Now()
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

// CountActiveSubscribers returns the number of users with active paid subscriptions
func (s *service) CountActiveSubscribers() (int64, error) {
	var count int64

	// Count users with active subscriptions who have not been granted by admin
	// This counts only paid subscribers
	err := s.db.Model(&User{}).
		Where("subscription_status = ? AND is_admin_granted = ? AND subscription_tier != ?",
			"active", false, "free").
		Count(&count).Error

	return count, err
}

// getFirstDayOfMonth returns the first day of the current month
func getFirstDayOfMonth() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
}

// getFirstDayOfLastMonth returns the first day of the previous month
func getFirstDayOfLastMonth() time.Time {
	now := time.Now()

	// Go back one month
	lastMonth := now.AddDate(0, -1, 0)

	// First day of the last month
	return time.Date(lastMonth.Year(), lastMonth.Month(), 1, 0, 0, 0, 0, now.Location())
}

// getFirstDayOfNextMonth returns the first day of the next month
func getFirstDayOfNextMonth() time.Time {
	firstOfThisMonth := getFirstDayOfMonth()
	return firstOfThisMonth.AddDate(0, 1, 0)
}

// CountNewUsersThisMonth returns the number of users registered in the current month
func (s *service) CountNewUsersThisMonth() (int64, error) {
	var count int64

	// Get the first day of the current month
	firstDay := getFirstDayOfMonth()

	// Count users created this month
	err := s.db.Model(&User{}).
		Where("created_at >= ?", firstDay).
		Count(&count).Error

	return count, err
}

// CountNewUsersLastMonth returns the number of users registered in the previous month
func (s *service) CountNewUsersLastMonth() (int64, error) {
	var count int64

	// Get the first day of the previous month
	firstDayLastMonth := getFirstDayOfLastMonth()

	// Get the first day of the current month
	firstDayThisMonth := getFirstDayOfMonth()

	// Count users created last month
	err := s.db.Model(&User{}).
		Where("created_at >= ? AND created_at < ?", firstDayLastMonth, firstDayThisMonth).
		Count(&count).Error

	return count, err
}

// CountNewSubscribersThisMonth returns the number of new subscriptions in the current month
func (s *service) CountNewSubscribersThisMonth() (int64, error) {
	var count int64

	// Get the first day of the current month
	firstDay := getFirstDayOfMonth()

	// Count users who got a subscription this month (not admin granted)
	err := s.db.Model(&User{}).
		Where("subscription_status = ? AND is_admin_granted = ? AND updated_at >= ? AND subscription_tier != ?",
			"active", false, firstDay, "free").
		Count(&count).Error

	return count, err
}

// CountNewSubscribersLastMonth returns the number of new subscriptions in the previous month
func (s *service) CountNewSubscribersLastMonth() (int64, error) {
	var count int64

	// Get the first day of the previous month
	firstDayLastMonth := getFirstDayOfLastMonth()

	// Get the first day of the current month
	firstDayThisMonth := getFirstDayOfMonth()

	// Count users who got a subscription last month (not admin granted)
	err := s.db.Model(&User{}).
		Where("subscription_status = ? AND is_admin_granted = ? AND updated_at >= ? AND updated_at < ? AND subscription_tier != ?",
			"active", false, firstDayLastMonth, firstDayThisMonth, "free").
		Count(&count).Error

	return count, err
}

// CheckExpiredPromotionSubscription checks if a user's subscription has expired
// and updates the status to "expired" if it has.
// It also resets the subscription tier to "free" and clears the expiration date.
// Returns true if the subscription status was updated, false otherwise.
func (s *service) CheckExpiredPromotionSubscription(user *User) (bool, error) {
	// If no subscription tier, already expired or free tier, nothing to do
	if user.SubscriptionTier == "" || user.SubscriptionStatus == "expired" || user.SubscriptionTier == "free" {
		return false, nil
	}

	// If subscription end date is in the past, mark as expired
	if !user.SubscriptionEndDate.IsZero() && time.Now().After(user.SubscriptionEndDate) {
		// Update subscription status to expired, reset tier to free, and clear end date
		user.SubscriptionStatus = "expired"
		user.SubscriptionTier = "free"
		user.SubscriptionEndDate = time.Time{} // zero time

		// Save the updated user
		err := s.db.Save(user).Error
		if err != nil {
			return false, err
		}

		return true, nil
	}

	// Subscription is still active
	return false, nil
}
