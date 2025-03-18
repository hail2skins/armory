package controller

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestResetPasswordGetRoute(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	mockDB := new(mocks.MockDB)
	mockEmailService := new(mocks.MockEmailService)

	// Setup mock behavior
	testUser := &database.User{
		Email:         "test@example.com",
		RecoveryToken: "valid-token",
	}
	mockDB.On("GetUserByRecoveryToken", mock.Anything, "valid-token").Return(testUser, nil)

	// Mock the IsRecoveryExpired method on the testUser
	mockDB.On("IsRecoveryExpired", mock.Anything, "valid-token").Return(false, nil)

	controller := mocks.NewTestAuthController(mockDB)
	controller.SetEmailService(mockEmailService)

	// Create a render function that records the data
	var renderedData data.AuthData
	controller.RenderResetPassword = func(c *gin.Context, d interface{}) {
		renderedData = d.(data.AuthData)
		c.Status(http.StatusOK)
	}

	r.GET("/reset-password", controller.ResetPasswordHandler)

	// Create a request
	req := httptest.NewRequest(http.MethodGet, "/reset-password?token=valid-token", nil)
	w := httptest.NewRecorder()

	// Perform the request
	r.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Reset Password", renderedData.Title)
	assert.Equal(t, "valid-token", renderedData.Token)
	mockDB.AssertExpectations(t)
}

func TestResetPasswordInvalidToken(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	mockDB := new(mocks.MockDB)
	mockEmailService := new(mocks.MockEmailService)

	// Setup mock behavior - invalid token
	mockDB.On("GetUserByRecoveryToken", mock.Anything, "invalid-token").Return(nil, database.ErrUserNotFound)

	controller := mocks.NewTestAuthController(mockDB)
	controller.SetEmailService(mockEmailService)

	r.GET("/reset-password", controller.ResetPasswordHandler)

	// Create a request with invalid token
	req := httptest.NewRequest(http.MethodGet, "/reset-password?token=invalid-token", nil)
	w := httptest.NewRecorder()

	// Perform the request
	r.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusSeeOther, w.Code) // Should redirect to login
	assert.Equal(t, "/login", w.Header().Get("Location"))
	mockDB.AssertExpectations(t)
}

func TestResetPasswordExpiredToken(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	mockDB := new(mocks.MockDB)
	mockEmailService := new(mocks.MockEmailService)

	// Setup mock behavior - expired token
	testUser := &database.User{
		Email:         "test@example.com",
		RecoveryToken: "expired-token",
	}
	mockDB.On("GetUserByRecoveryToken", mock.Anything, "expired-token").Return(testUser, nil)

	// Mock the IsRecoveryExpired method to return true for expired token
	mockDB.On("IsRecoveryExpired", mock.Anything, "expired-token").Return(true, nil)

	controller := mocks.NewTestAuthController(mockDB)
	controller.SetEmailService(mockEmailService)

	r.GET("/reset-password", controller.ResetPasswordHandler)

	// Create a request with expired token
	req := httptest.NewRequest(http.MethodGet, "/reset-password?token=expired-token", nil)
	w := httptest.NewRecorder()

	// Perform the request
	r.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusSeeOther, w.Code) // Should redirect to login
	assert.Equal(t, "/login", w.Header().Get("Location"))
	mockDB.AssertExpectations(t)
}

func TestResetPasswordSubmitSuccess(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	mockDB := new(mocks.MockDB)
	mockEmailService := new(mocks.MockEmailService)

	// Setup mock behavior
	testUser := &database.User{
		Email:         "test@example.com",
		RecoveryToken: "valid-token",
	}
	mockDB.On("GetUserByRecoveryToken", mock.Anything, "valid-token").Return(testUser, nil)
	mockDB.On("IsRecoveryExpired", mock.Anything, "valid-token").Return(false, nil)
	mockDB.On("ResetPassword", mock.Anything, "valid-token", "newpassword123").Return(nil)

	controller := mocks.NewTestAuthController(mockDB)
	controller.SetEmailService(mockEmailService)

	r.POST("/reset-password", controller.ResetPasswordHandler)

	// Create a request with form data
	form := url.Values{}
	form.Add("token", "valid-token")
	form.Add("password", "newpassword123")
	form.Add("confirm_password", "newpassword123")
	req := httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	// Perform the request
	r.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusSeeOther, w.Code) // Should redirect to login
	assert.Equal(t, "/login", w.Header().Get("Location"))
	mockDB.AssertExpectations(t)
}

func TestResetPasswordSubmitPasswordMismatch(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	mockDB := new(mocks.MockDB)
	mockEmailService := new(mocks.MockEmailService)

	// Setup mock behavior
	testUser := &database.User{
		Email:         "test@example.com",
		RecoveryToken: "valid-token",
	}
	mockDB.On("GetUserByRecoveryToken", mock.Anything, "valid-token").Return(testUser, nil)
	mockDB.On("IsRecoveryExpired", mock.Anything, "valid-token").Return(false, nil)

	controller := mocks.NewTestAuthController(mockDB)
	controller.SetEmailService(mockEmailService)

	// Create a render function that records the data
	var renderedData data.AuthData
	controller.RenderResetPassword = func(c *gin.Context, d interface{}) {
		renderedData = d.(data.AuthData)
		c.Status(http.StatusOK)
	}

	r.POST("/reset-password", controller.ResetPasswordHandler)

	// Create a request with mismatched passwords
	form := url.Values{}
	form.Add("token", "valid-token")
	form.Add("password", "newpassword123")
	form.Add("confirm_password", "differentpassword")
	req := httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	// Perform the request
	r.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, renderedData.Error)
	assert.Equal(t, "Passwords do not match", renderedData.Error)
	mockDB.AssertExpectations(t)
}

func TestResetPasswordSubmitDatabaseError(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	mockDB := new(mocks.MockDB)
	mockEmailService := new(mocks.MockEmailService)

	// Setup mock behavior - database error
	testUser := &database.User{
		Email:         "test@example.com",
		RecoveryToken: "valid-token",
	}
	mockDB.On("GetUserByRecoveryToken", mock.Anything, "valid-token").Return(testUser, nil)
	mockDB.On("IsRecoveryExpired", mock.Anything, "valid-token").Return(false, nil)
	mockDB.On("ResetPassword", mock.Anything, "valid-token", "newpassword123").Return(database.ErrInvalidCredentials)

	controller := mocks.NewTestAuthController(mockDB)
	controller.SetEmailService(mockEmailService)

	// Create a render function that records the data
	var renderedData data.AuthData
	controller.RenderResetPassword = func(c *gin.Context, d interface{}) {
		renderedData = d.(data.AuthData)
		c.Status(http.StatusOK)
	}

	r.POST("/reset-password", controller.ResetPasswordHandler)

	// Create a request with form data
	form := url.Values{}
	form.Add("token", "valid-token")
	form.Add("password", "newpassword123")
	form.Add("confirm_password", "newpassword123")
	req := httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	// Perform the request
	r.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, renderedData.Error)
	assert.Contains(t, renderedData.Error, "Failed to reset password", "Error message should contain 'Failed to reset password'")
	mockDB.AssertExpectations(t)
}
