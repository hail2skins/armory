package database

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserModelValidation(t *testing.T) {
	// Create a test database
	_, tempDir := setupUserTestDB(t)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name     string
		user     *User
		wantErr  bool
		errorMsg string
	}{
		{
			name: "Valid user",
			user: &User{
				Email:    "test@example.com",
				Password: "Password123!",
			},
			wantErr:  false,
			errorMsg: "",
		},
		{
			name: "Invalid email",
			user: &User{
				Email:    "invalid-email",
				Password: "Password123!",
			},
			wantErr:  true,
			errorMsg: ErrInvalidEmail.Error(),
		},
		{
			name: "Invalid password - too short",
			user: &User{
				Email:    "test@example.com",
				Password: "Pass1!",
			},
			wantErr:  true,
			errorMsg: ErrInvalidPassword.Error(),
		},
		{
			name: "Invalid password - no uppercase",
			user: &User{
				Email:    "test@example.com",
				Password: "password123!",
			},
			wantErr:  true,
			errorMsg: ErrInvalidPassword.Error(),
		},
		{
			name: "Invalid password - no special char",
			user: &User{
				Email:    "test@example.com",
				Password: "Password123",
			},
			wantErr:  true,
			errorMsg: ErrInvalidPassword.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test validation directly
			err := tt.user.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.errorMsg, err.Error())
			} else {
				assert.NoError(t, err)
			}

			// We no longer test BeforeCreate since it doesn't validate directly
			// Each controller should call Validate prior to creating a User
		})
	}
}
