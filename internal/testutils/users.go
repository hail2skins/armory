package testutils

import (
	"context"

	"github.com/hail2skins/armory/internal/database"
)

// CreateTestUser creates a test user
func CreateTestUser(ctx context.Context, db database.Service, email, password string) (*database.User, error) {
	return db.CreateUser(ctx, email, password)
}

// GetTestUserByEmail gets a test user by email
func GetTestUserByEmail(ctx context.Context, db database.Service, email string) (*database.User, error) {
	return db.GetUserByEmail(ctx, email)
}

// AuthenticateTestUser authenticates a test user
func AuthenticateTestUser(ctx context.Context, db database.Service, email, password string) (*database.User, error) {
	return db.AuthenticateUser(ctx, email, password)
}
