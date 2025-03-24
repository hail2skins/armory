package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		wantErr  bool
		errorMsg string
	}{
		{
			name:     "Valid email",
			email:    "test@example.com",
			wantErr:  false,
			errorMsg: "",
		},
		{
			name:     "Empty email",
			email:    "",
			wantErr:  true,
			errorMsg: ErrInvalidEmail.Error(),
		},
		{
			name:     "No @ symbol",
			email:    "testexample.com",
			wantErr:  true,
			errorMsg: ErrInvalidEmail.Error(),
		},
		{
			name:     "No domain part",
			email:    "test@",
			wantErr:  true,
			errorMsg: ErrInvalidEmail.Error(),
		},
		{
			name:     "No TLD",
			email:    "test@example",
			wantErr:  true,
			errorMsg: ErrInvalidEmail.Error(),
		},
		{
			name:     "Valid email with subdomain",
			email:    "test@sub.example.com",
			wantErr:  false,
			errorMsg: "",
		},
		{
			name:     "Valid email with plus",
			email:    "test+123@example.com",
			wantErr:  false,
			errorMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.errorMsg, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
		errorMsg string
	}{
		{
			name:     "Valid password",
			password: "Password123!",
			wantErr:  false,
			errorMsg: "",
		},
		{
			name:     "Too short",
			password: "Pass1!",
			wantErr:  true,
			errorMsg: ErrPasswordTooShort.Error(),
		},
		{
			name:     "No uppercase",
			password: "password123!",
			wantErr:  true,
			errorMsg: ErrPasswordNoUppercase.Error(),
		},
		{
			name:     "No special character",
			password: "Password123",
			wantErr:  true,
			errorMsg: ErrPasswordNoSpecialChar.Error(),
		},
		{
			name:     "Empty password",
			password: "",
			wantErr:  true,
			errorMsg: ErrPasswordTooShort.Error(),
		},
		{
			name:     "Just long enough with requirements",
			password: "Pass123!",
			wantErr:  false,
			errorMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.errorMsg, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
