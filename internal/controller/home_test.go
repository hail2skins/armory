package controller

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/services/email"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockEmailServiceWithContact is a mock implementation of the email.EmailService interface
type MockEmailServiceWithContact struct {
	mock.Mock
}

func (m *MockEmailServiceWithContact) SendVerificationEmail(email, token string) error {
	args := m.Called(email, token)
	return args.Error(0)
}

func (m *MockEmailServiceWithContact) SendPasswordResetEmail(email, token string) error {
	args := m.Called(email, token)
	return args.Error(0)
}

func (m *MockEmailServiceWithContact) SendContactEmail(name, email, subject, message string) error {
	args := m.Called(name, email, subject, message)
	return args.Error(0)
}

func TestHomeController_ContactHandler(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create a mock DB and email service
	mockDB := new(MockDB)
	mockEmailService := new(MockEmailServiceWithContact)

	// Create a test HTTP server
	router := gin.New()

	// Create a mock auth controller
	mockAuthController := new(AuthController)
	mockAuthController.db = mockDB

	// Set up middleware for auth data
	router.Use(func(c *gin.Context) {
		c.Set("authController", mockAuthController)
		c.Next()
	})

	// Create a home controller with the mock services
	homeController := &HomeController{
		db:           mockDB,
		emailService: mockEmailService,
	}

	// Register the contact route
	router.GET("/contact", homeController.ContactHandler)
	router.POST("/contact", homeController.ContactHandler)

	t.Run("GET contact page", func(t *testing.T) {
		// Create a test request
		req, _ := http.NewRequest("GET", "/contact", nil)
		resp := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(resp, req)

		// Assert response
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), "Contact Us")
		assert.Contains(t, resp.Body.String(), "Have questions or feedback?")
	})

	t.Run("POST contact form - success", func(t *testing.T) {
		// Set up the mock email service to return success
		mockEmailService.On("SendContactEmail", "John Doe", "john@example.com", "Test Subject", "Test Message").Return(nil).Once()

		// Create form data
		form := url.Values{}
		form.Add("name", "John Doe")
		form.Add("email", "john@example.com")
		form.Add("subject", "Test Subject")
		form.Add("message", "Test Message")

		// Create a test request
		req, _ := http.NewRequest("POST", "/contact", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(resp, req)

		// Assert response
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), "Thank you for your message")
		assert.Contains(t, resp.Body.String(), "bg-green-100")

		// Verify that the email service was called
		mockEmailService.AssertExpectations(t)
	})

	t.Run("POST contact form - missing fields", func(t *testing.T) {
		// Create form data with missing fields
		form := url.Values{}
		form.Add("name", "John Doe")
		form.Add("email", "john@example.com")
		// Missing subject and message

		// Create a test request
		req, _ := http.NewRequest("POST", "/contact", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(resp, req)

		// Assert response
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), "Please fill out all fields")
		assert.Contains(t, resp.Body.String(), "bg-red-100")
	})

	t.Run("POST contact form - email service error", func(t *testing.T) {
		// Set up the mock email service to return an error
		mockEmailService.On("SendContactEmail", "Error User", "error@example.com", "Error Subject", "Error Message").Return(email.ErrEmailSendFailed).Once()

		// Create form data
		form := url.Values{}
		form.Add("name", "Error User")
		form.Add("email", "error@example.com")
		form.Add("subject", "Error Subject")
		form.Add("message", "Error Message")

		// Create a test request
		req, _ := http.NewRequest("POST", "/contact", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(resp, req)

		// Assert response
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), "There was an error sending your message")
		assert.Contains(t, resp.Body.String(), "bg-red-100")

		// Verify that the email service was called
		mockEmailService.AssertExpectations(t)
	})
}
