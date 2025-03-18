package tests

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/hail2skins/armory/internal/services/email"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// HomeControllerTestSuite is a test suite for the HomeController
type HomeControllerTestSuite struct {
	ControllerTestSuite
}

// TestContactPage tests the GET handler for the contact page
func (s *HomeControllerTestSuite) TestContactPage() {
	// Create the controller
	homeController := s.CreateHomeController()

	// Register routes
	s.Router.GET("/contact", homeController.ContactHandler)

	// Create a test request
	req, _ := http.NewRequest("GET", "/contact", nil)
	resp := httptest.NewRecorder()

	// Mock the auth controller's GetCurrentUser method
	// Use mock.Anything to match any gin.Context passed to the method
	s.MockAuth.On("GetCurrentUser", mock.Anything).Return(nil, false)

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response
	s.Equal(http.StatusOK, resp.Code)
	s.Contains(resp.Body.String(), "Contact Us")
	s.Contains(resp.Body.String(), "Have questions or feedback?")
}

// TestContactFormSubmission tests submitting the contact form
func (s *HomeControllerTestSuite) TestContactFormSubmission() {
	// Create the controller
	homeController := s.CreateHomeController()

	// Register routes
	s.Router.POST("/contact", homeController.ContactHandler)

	// Set up expectations for email service
	s.MockEmail.On("SendContactEmail", "John Doe", "john@example.com", "Test Subject", "Test Message").Return(nil)

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

	// Mock the auth controller's GetCurrentUser method
	// Use mock.Anything to match any gin.Context passed to the method
	s.MockAuth.On("GetCurrentUser", mock.Anything).Return(nil, false)

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response
	s.Equal(http.StatusOK, resp.Code)
	s.Contains(resp.Body.String(), "Thank you for your message")
	s.Contains(resp.Body.String(), "bg-green-100")

	// Verify mock expectations
	s.MockEmail.AssertExpectations(s.T())
}

// TestContactFormWithMissingFields tests the form validation
func (s *HomeControllerTestSuite) TestContactFormWithMissingFields() {
	// Create the controller
	homeController := s.CreateHomeController()

	// Register routes
	s.Router.POST("/contact", homeController.ContactHandler)

	// Create form data with missing fields
	form := url.Values{}
	form.Add("name", "John Doe")
	form.Add("email", "john@example.com")
	// Missing subject and message

	// Create a test request
	req, _ := http.NewRequest("POST", "/contact", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()

	// Mock the auth controller's GetCurrentUser method
	// Use mock.Anything to match any gin.Context passed to the method
	s.MockAuth.On("GetCurrentUser", mock.Anything).Return(nil, false)

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response
	s.Equal(http.StatusOK, resp.Code)
	s.Contains(resp.Body.String(), "Please fill out all fields")
	s.Contains(resp.Body.String(), "bg-red-100")
}

// TestContactFormWithEmailError tests handling email service errors
func (s *HomeControllerTestSuite) TestContactFormWithEmailError() {
	// Create the controller
	homeController := s.CreateHomeController()

	// Register routes
	s.Router.POST("/contact", homeController.ContactHandler)

	// Set up expectations for email service to return an error
	s.MockEmail.On("SendContactEmail", "Error User", "error@example.com", "Error Subject", "Error Message").Return(email.ErrEmailSendFailed)

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

	// Mock the auth controller's GetCurrentUser method
	// Use mock.Anything to match any gin.Context passed to the method
	s.MockAuth.On("GetCurrentUser", mock.Anything).Return(nil, false)

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response
	s.Equal(http.StatusOK, resp.Code)
	s.Contains(resp.Body.String(), "There was an error sending your message")
	s.Contains(resp.Body.String(), "bg-red-100")

	// Verify mock expectations
	s.MockEmail.AssertExpectations(s.T())
}

// Run the tests
func TestHomeControllerSuite(t *testing.T) {
	suite.Run(t, new(HomeControllerTestSuite))
}
