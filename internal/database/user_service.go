package database

import (
	"context"
	"errors"

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
		return nil, nil // Password doesn't match
	}

	return user, nil
}
