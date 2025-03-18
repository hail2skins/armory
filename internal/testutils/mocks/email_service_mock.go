package mocks

import (
	"github.com/stretchr/testify/mock"
)

// MockEmailService is a mock implementation of the email.EmailService interface
// that can be shared across tests
type MockEmailService struct {
	mock.Mock
}

// SendVerificationEmail mocks sending a verification email
func (m *MockEmailService) SendVerificationEmail(email, token string) error {
	args := m.Called(email, token)
	return args.Error(0)
}

// SendPasswordResetEmail mocks sending a password reset email
func (m *MockEmailService) SendPasswordResetEmail(email, token string) error {
	args := m.Called(email, token)
	return args.Error(0)
}

// SendContactEmail mocks sending a contact form email
func (m *MockEmailService) SendContactEmail(name, email, subject, message string) error {
	args := m.Called(name, email, subject, message)
	return args.Error(0)
}

// SendEmailChangeVerification mocks sending an email change verification
func (m *MockEmailService) SendEmailChangeVerification(email, token string) error {
	args := m.Called(email, token)
	return args.Error(0)
}
