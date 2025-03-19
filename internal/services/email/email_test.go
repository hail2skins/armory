package email

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockEmailService is a mock implementation of EmailService
type MockEmailService struct {
	mock.Mock
}

func (m *MockEmailService) SendVerificationEmail(email, token, baseURL string) error {
	args := m.Called(email, token, baseURL)
	return args.Error(0)
}

func (m *MockEmailService) SendPasswordResetEmail(email, token, baseURL string) error {
	args := m.Called(email, token, baseURL)
	return args.Error(0)
}

func (m *MockEmailService) SendContactEmail(name, email, subject, message string) error {
	args := m.Called(name, email, subject, message)
	return args.Error(0)
}

func (m *MockEmailService) SendEmailChangeVerification(email, token, baseURL string) error {
	args := m.Called(email, token, baseURL)
	return args.Error(0)
}

func TestNewMailjetService(t *testing.T) {
	// Save original env vars
	origAPIKey := os.Getenv("MAILJET_API_KEY")
	origSecretKey := os.Getenv("MAILJET_SECRET_KEY")
	origSenderEmail := os.Getenv("MAILJET_SENDER_EMAIL")
	origSenderName := os.Getenv("MAILJET_SENDER_NAME")
	origAdminEmail := os.Getenv("ADMIN_EMAIL")

	// Restore env vars after test
	defer func() {
		os.Setenv("MAILJET_API_KEY", origAPIKey)
		os.Setenv("MAILJET_SECRET_KEY", origSecretKey)
		os.Setenv("MAILJET_SENDER_EMAIL", origSenderEmail)
		os.Setenv("MAILJET_SENDER_NAME", origSenderName)
		os.Setenv("ADMIN_EMAIL", origAdminEmail)
	}()

	// Test missing API key
	os.Setenv("MAILJET_API_KEY", "")
	os.Setenv("MAILJET_SECRET_KEY", "secret")
	os.Setenv("MAILJET_SENDER_EMAIL", "test@example.com")
	os.Setenv("MAILJET_SENDER_NAME", "Test")
	os.Setenv("ADMIN_EMAIL", "admin@example.com")

	service, err := NewMailjetService()
	assert.Error(t, err)
	assert.Nil(t, service)

	// Test missing secret key
	os.Setenv("MAILJET_API_KEY", "key")
	os.Setenv("MAILJET_SECRET_KEY", "")
	service, err = NewMailjetService()
	assert.Error(t, err)
	assert.Nil(t, service)

	// Test missing sender email
	os.Setenv("MAILJET_API_KEY", "key")
	os.Setenv("MAILJET_SECRET_KEY", "secret")
	os.Setenv("MAILJET_SENDER_EMAIL", "")
	service, err = NewMailjetService()
	assert.Error(t, err)
	assert.Nil(t, service)

	// Test missing sender name
	os.Setenv("MAILJET_SENDER_EMAIL", "test@example.com")
	os.Setenv("MAILJET_SENDER_NAME", "")
	service, err = NewMailjetService()
	assert.Error(t, err)
	assert.Nil(t, service)

	// Test missing admin email
	os.Setenv("MAILJET_SENDER_NAME", "Test")
	os.Setenv("ADMIN_EMAIL", "")
	service, err = NewMailjetService()
	assert.Error(t, err)
	assert.Nil(t, service)

	// Test successful creation
	os.Setenv("MAILJET_API_KEY", "key")
	os.Setenv("MAILJET_SECRET_KEY", "secret")
	os.Setenv("MAILJET_SENDER_EMAIL", "test@example.com")
	os.Setenv("MAILJET_SENDER_NAME", "Test")
	os.Setenv("ADMIN_EMAIL", "admin@example.com")

	service, err = NewMailjetService()
	assert.NoError(t, err)
	assert.NotNil(t, service)
}

func TestMailjetService_SendVerificationEmail(t *testing.T) {
	mockService := new(MockEmailService)

	// Test successful email sending
	mockService.On("SendVerificationEmail", "user@example.com", "test-token", "http://localhost:3000").Return(nil)
	err := mockService.SendVerificationEmail("user@example.com", "test-token", "http://localhost:3000")
	assert.NoError(t, err)
	mockService.AssertExpectations(t)

	// Test error case
	mockService.On("SendVerificationEmail", "error@example.com", "test-token", "http://localhost:3000").Return(ErrEmailSendFailed)
	err = mockService.SendVerificationEmail("error@example.com", "test-token", "http://localhost:3000")
	assert.Error(t, err)
	assert.Equal(t, ErrEmailSendFailed, err)
	mockService.AssertExpectations(t)
}

func TestMailjetService_SendPasswordResetEmail(t *testing.T) {
	mockService := new(MockEmailService)

	// Test successful email sending
	mockService.On("SendPasswordResetEmail", "user@example.com", "test-token", "http://localhost:3000").Return(nil)
	err := mockService.SendPasswordResetEmail("user@example.com", "test-token", "http://localhost:3000")
	assert.NoError(t, err)
	mockService.AssertExpectations(t)

	// Test error case
	mockService.On("SendPasswordResetEmail", "error@example.com", "test-token", "http://localhost:3000").Return(ErrEmailSendFailed)
	err = mockService.SendPasswordResetEmail("error@example.com", "test-token", "http://localhost:3000")
	assert.Error(t, err)
	assert.Equal(t, ErrEmailSendFailed, err)
	mockService.AssertExpectations(t)
}

func TestMailjetService_SendEmailChangeVerification(t *testing.T) {
	mockService := new(MockEmailService)

	// Test successful email sending
	mockService.On("SendEmailChangeVerification", "user@example.com", "test-token", "http://localhost:3000").Return(nil)
	err := mockService.SendEmailChangeVerification("user@example.com", "test-token", "http://localhost:3000")
	assert.NoError(t, err)
	mockService.AssertExpectations(t)

	// Test error case
	mockService.On("SendEmailChangeVerification", "error@example.com", "test-token", "http://localhost:3000").Return(ErrEmailSendFailed)
	err = mockService.SendEmailChangeVerification("error@example.com", "test-token", "http://localhost:3000")
	assert.Error(t, err)
	assert.Equal(t, ErrEmailSendFailed, err)
	mockService.AssertExpectations(t)
}

func TestMailjetService_SendContactEmail(t *testing.T) {
	mockService := new(MockEmailService)

	// Test successful email sending
	mockService.On("SendContactEmail", "John Doe", "john@example.com", "Test Subject", "Test Message").Return(nil)
	err := mockService.SendContactEmail("John Doe", "john@example.com", "Test Subject", "Test Message")
	assert.NoError(t, err)
	mockService.AssertExpectations(t)

	// Test error case
	mockService.On("SendContactEmail", "Error User", "error@example.com", "Error Subject", "Error Message").Return(ErrEmailSendFailed)
	err = mockService.SendContactEmail("Error User", "error@example.com", "Error Subject", "Error Message")
	assert.Error(t, err)
	assert.Equal(t, ErrEmailSendFailed, err)
	mockService.AssertExpectations(t)
}
