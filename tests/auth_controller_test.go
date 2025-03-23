package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hail2skins/armory/internal/database"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// AuthControllerTestSuite is a test suite for the AuthController
type AuthControllerTestSuite struct {
	ControllerTestSuite
}

// TestLoginPage tests the GET handler for the login page
func (s *AuthControllerTestSuite) TestLoginPage() {
	// Create the controller
	authController := s.CreateAuthController()

	// Register routes
	s.Router.GET("/login", authController.LoginHandler)

	// Create a test request
	req, _ := http.NewRequest("GET", "/login", nil)
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// Assert response
	s.Equal(http.StatusOK, resp.Code)
	s.Contains(resp.Body.String(), "Login")
	s.Contains(resp.Body.String(), "Email")
	s.Contains(resp.Body.String(), "Password")
}

// TestSuccessfulLogin tests a successful login
func (s *AuthControllerTestSuite) TestSuccessfulLogin() {
	// Create the controller
	authController := s.CreateAuthController()

	// Register routes
	s.Router.POST("/login", authController.LoginHandler)

	// Create mock verified user
	mockUser := &database.User{
		Email:    "test@example.com",
		Verified: true, // Important: User must be verified for successful login
	}

	// Set up mock expectations for AuthenticateUser with context.Background()
	s.MockDB.On("AuthenticateUser", mock.Anything, "test@example.com", "password123").Return(mockUser, nil)

	// Create form data instead of JSON
	formData := "email=test@example.com&password=password123"

	// Create a test request with form data
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// The successful login should result in a redirect to /owner
	s.Equal(http.StatusSeeOther, resp.Code)
	s.Equal("/owner", resp.Header().Get("Location"))

	// Verify DB mock expectations
	s.MockDB.AssertExpectations(s.T())
}

// TestFailedLogin tests a failed login attempt
func (s *AuthControllerTestSuite) TestFailedLogin() {
	// Create the controller
	authController := s.CreateAuthController()

	// Register routes
	s.Router.POST("/login", authController.LoginHandler)

	// Set up mock expectations for AuthenticateUser to return nil user with error
	s.MockDB.On("AuthenticateUser", mock.Anything, "test@example.com", "wrongpassword").Return(nil, database.ErrInvalidCredentials)

	// Create form data instead of JSON
	formData := "email=test@example.com&password=wrongpassword"

	// Create a test request
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(resp, req)

	// The error could be either:
	// 1. A 401 response with "Invalid email or password"
	// 2. A 200 response with an error message in the HTML
	if resp.Code == http.StatusUnauthorized {
		s.Contains(resp.Body.String(), "Invalid email or password")
	} else {
		s.Equal(http.StatusOK, resp.Code)
		s.Contains(resp.Body.String(), "Invalid email or password")
	}

	// Verify mock expectations
	s.MockDB.AssertExpectations(s.T())
}

// Run the tests
func TestAuthControllerSuite(t *testing.T) {
	suite.Run(t, new(AuthControllerTestSuite))
}
