package mocks

import (
	"github.com/stretchr/testify/mock"
)

// MockEmailService is a mock implementation of email.EmailService
type MockEmailService struct {
	mock.Mock

	// Track verification email calls
	VerificationEmailSent bool
	LastVerificationEmail string
	LastVerificationToken string
	LastBaseURL           string

	// Track password reset email calls
	PasswordResetEmailSent bool
	LastResetEmail         string
	LastResetToken         string
	LastResetBaseURL       string

	// Track email change verification calls
	EmailChangeVerificationSent bool
	LastChangeEmail             string
	LastChangeToken             string
	LastChangeBaseURL           string
}

// SendVerificationEmail implements email.EmailService
func (m *MockEmailService) SendVerificationEmail(email, token, baseURL string) error {
	args := m.Called(email, token, baseURL)
	m.VerificationEmailSent = true
	m.LastVerificationEmail = email
	m.LastVerificationToken = token
	m.LastBaseURL = baseURL
	return args.Error(0)
}

// SendPasswordResetEmail implements email.EmailService
func (m *MockEmailService) SendPasswordResetEmail(email, token, baseURL string) error {
	args := m.Called(email, token, baseURL)
	m.PasswordResetEmailSent = true
	m.LastResetEmail = email
	m.LastResetToken = token
	m.LastResetBaseURL = baseURL
	return args.Error(0)
}

// SendEmailChangeVerification implements email.EmailService
func (m *MockEmailService) SendEmailChangeVerification(email, token, baseURL string) error {
	args := m.Called(email, token, baseURL)
	m.EmailChangeVerificationSent = true
	m.LastChangeEmail = email
	m.LastChangeToken = token
	m.LastChangeBaseURL = baseURL
	return args.Error(0)
}

// SendContactEmail implements email.EmailService
func (m *MockEmailService) SendContactEmail(name, email, subject, message string) error {
	args := m.Called(name, email, subject, message)
	return args.Error(0)
}
