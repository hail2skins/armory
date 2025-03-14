package controller

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockEmailService struct {
	mock.Mock
}

func (m *mockEmailService) SendVerificationEmail(email, token string) error {
	args := m.Called(email, token)
	return args.Error(0)
}

func (m *mockEmailService) SendPasswordResetEmail(email, token string) error {
	args := m.Called(email, token)
	return args.Error(0)
}

func TestVerificationFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDB)
	mockEmailSvc := new(mockEmailService)

	// Create test user
	testUser := &database.User{
		Email: "test@example.com",
	}
	database.SetUserID(testUser, 1)

	// Setup mock responses for registration
	mockDB.On("GetUserByEmail", mock.Anything, "test@example.com").Return(nil, nil).Once()
	mockDB.On("CreateUser", mock.Anything, "test@example.com", "password123").Return(testUser, nil).Once()
	mockDB.On("UpdateUser", mock.Anything, mock.AnythingOfType("*database.User")).Return(nil).Once()

	// Setup mock responses for verification
	mockDB.On("GetUserByVerificationToken", mock.Anything, "valid-token").Return(testUser, nil).Once()
	mockDB.On("GetUserByVerificationToken", mock.Anything, "invalid-token").Return(nil, nil).Once()
	mockDB.On("VerifyUserEmail", mock.Anything, "valid-token").Return(testUser, nil).Once()
	// We don't need to expect VerifyUserEmail for invalid-token because the function returns early

	mockEmailSvc.On("SendVerificationEmail", "test@example.com", mock.AnythingOfType("string")).Return(nil).Once()

	// Create auth controller
	authController := NewAuthController(mockDB)
	authController.emailService = mockEmailSvc

	// Create test router
	router := gin.New()
	router.POST("/register", authController.RegisterHandler)
	router.GET("/verify", authController.VerifyEmailHandler)

	// Test registration with verification email
	t.Run("Registration sends verification email", func(t *testing.T) {
		form := url.Values{}
		form.Add("email", "test@example.com")
		form.Add("password", "password123")
		form.Add("password_confirm", "password123")

		req := httptest.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Test", "true")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusSeeOther, w.Code)
	})

	// Test email verification with valid token
	t.Run("Valid verification token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/verify?token=valid-token", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusSeeOther, w.Code)
	})

	// Test email verification with invalid token
	t.Run("Invalid verification token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/verify?token=invalid-token", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// Assert all expectations at the end
	mockDB.AssertExpectations(t)
	mockEmailSvc.AssertExpectations(t)
}

func TestPasswordResetFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDB)
	mockEmailSvc := new(mockEmailService)

	// Create test user
	testUser := &database.User{
		Email:         "test@example.com",
		RecoveryToken: "valid-token",
	}
	database.SetUserID(testUser, 1)

	// Setup mock responses
	mockDB.On("GetUserByEmail", mock.Anything, "test@example.com").Return(testUser, nil).Once()
	mockDB.On("GetUserByEmail", mock.Anything, "nonexistent@example.com").Return(nil, nil).Once()
	mockDB.On("RequestPasswordReset", mock.Anything, "test@example.com").Return(testUser, nil).Once()
	mockDB.On("ResetPassword", mock.Anything, "valid-token", "newpassword123").Return(nil).Once()
	mockDB.On("ResetPassword", mock.Anything, "invalid-token", mock.Anything).Return(database.ErrInvalidToken).Once()

	mockEmailSvc.On("SendPasswordResetEmail", "test@example.com", mock.AnythingOfType("string")).Return(nil).Once()

	// Create auth controller
	authController := NewAuthController(mockDB)
	authController.emailService = mockEmailSvc

	// Create test router
	router := gin.New()
	router.POST("/forgot-password", authController.ForgotPasswordHandler)
	router.POST("/reset-password", authController.ResetPasswordHandler)

	// Test requesting password reset
	t.Run("Request password reset for existing user", func(t *testing.T) {
		form := url.Values{}
		form.Add("email", "test@example.com")

		req := httptest.NewRequest("POST", "/forgot-password", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Test", "true")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusSeeOther, w.Code)
	})

	// Test requesting password reset for non-existent user
	t.Run("Request password reset for non-existent user", func(t *testing.T) {
		form := url.Values{}
		form.Add("email", "nonexistent@example.com")

		req := httptest.NewRequest("POST", "/forgot-password", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Test", "true")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusSeeOther, w.Code)
	})

	// Test resetting password with valid token
	t.Run("Reset password with valid token", func(t *testing.T) {
		form := url.Values{}
		form.Add("token", "valid-token")
		form.Add("password", "newpassword123")
		form.Add("confirm_password", "newpassword123")

		req := httptest.NewRequest("POST", "/reset-password", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Test", "true")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusSeeOther, w.Code)
	})

	// Test resetting password with invalid token
	t.Run("Reset password with invalid token", func(t *testing.T) {
		form := url.Values{}
		form.Add("token", "invalid-token")
		form.Add("password", "newpassword123")
		form.Add("confirm_password", "newpassword123")

		req := httptest.NewRequest("POST", "/reset-password", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Test", "true")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// Assert all expectations at the end
	mockDB.AssertExpectations(t)
	mockEmailSvc.AssertExpectations(t)
}
