package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/services/email"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ErrEmailSendFailed is used for testing email send failures
var ErrEmailSendFailed = errors.New("failed to send email")

func TestResetPasswordRequestGetRoute(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	mockDB := new(mocks.MockDB)
	mockEmailService := new(mocks.MockEmailService)

	controller := mocks.NewTestAuthController(mockDB)
	controller.SetEmailService(mockEmailService)

	// Create a render function that just records the data
	var renderedData data.AuthData
	controller.RenderForgotPassword = func(c *gin.Context, d interface{}) {
		renderedData = d.(data.AuthData)
		c.Status(http.StatusOK)
	}

	r.GET("/reset-password/new", controller.ForgotPasswordHandler)

	// Create a request
	req := httptest.NewRequest(http.MethodGet, "/reset-password/new", nil)
	w := httptest.NewRecorder()

	// Perform the request
	r.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Reset Password", renderedData.Title)
}

func TestForgotPasswordSubmitValidEmail(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	mockDB := new(mocks.MockDB)
	mockEmailService := new(mocks.MockEmailService)

	// Create a test user with a valid email
	testUser := &database.User{
		Email: "test@example.com",
	}

	// Setup mock behavior
	mockDB.On("GetUserByEmail", mock.Anything, "test@example.com").Return(testUser, nil)
	mockDB.On("RequestPasswordReset", mock.Anything, "test@example.com").Return(testUser, nil)
	mockEmailService.On("SendPasswordResetEmail", "test@example.com", mock.Anything).Return(nil)

	controller := mocks.NewTestAuthController(mockDB)
	controller.SetEmailService(mockEmailService)

	// Create a render function that records the data
	var renderedData data.AuthData
	controller.RenderForgotPassword = func(c *gin.Context, d interface{}) {
		renderedData = d.(data.AuthData)
		c.Status(http.StatusOK)
	}

	r.POST("/forgot-password", controller.ForgotPasswordHandler)

	// Create a request with form data
	form := url.Values{}
	form.Add("email", "test@example.com")
	req := httptest.NewRequest(http.MethodPost, "/forgot-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	// Perform the request
	r.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, renderedData.Success)
	mockDB.AssertExpectations(t)
	mockEmailService.AssertExpectations(t)
}

func TestForgotPasswordSubmitInvalidEmail(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	mockDB := new(mocks.MockDB)
	mockEmailService := new(mocks.MockEmailService)

	// Setup mock behavior - user not found
	mockDB.On("GetUserByEmail", mock.Anything, "invalid@example.com").Return(nil, database.ErrUserNotFound)

	controller := mocks.NewTestAuthController(mockDB)
	controller.SetEmailService(mockEmailService)

	// Create a render function that records the data
	var renderedData data.AuthData
	controller.RenderForgotPassword = func(c *gin.Context, d interface{}) {
		renderedData = d.(data.AuthData)
		c.Status(http.StatusOK)
	}

	r.POST("/forgot-password", controller.ForgotPasswordHandler)

	// Create a request with form data
	form := url.Values{}
	form.Add("email", "invalid@example.com")
	req := httptest.NewRequest(http.MethodPost, "/forgot-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	// Perform the request
	r.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, renderedData.Success) // We still show success to avoid revealing valid emails
	mockDB.AssertExpectations(t)
	// Email service should not be called for invalid email
	mockEmailService.AssertNotCalled(t, "SendPasswordResetEmail", mock.Anything, mock.Anything)
}

func TestForgotPasswordEmailServiceFailure(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	mockDB := new(mocks.MockDB)
	mockEmailService := new(mocks.MockEmailService)

	// Create a test user with a valid email
	testUser := &database.User{
		Email: "test@example.com",
	}

	// Setup mock behavior
	mockDB.On("GetUserByEmail", mock.Anything, "test@example.com").Return(testUser, nil)
	mockDB.On("RequestPasswordReset", mock.Anything, "test@example.com").Return(testUser, nil)
	mockEmailService.On("SendPasswordResetEmail", "test@example.com", mock.Anything).Return(
		email.ErrEmailSendFailed)

	controller := mocks.NewTestAuthController(mockDB)
	controller.SetEmailService(mockEmailService)

	// Create a render function that records the data
	var renderedData data.AuthData
	controller.RenderForgotPassword = func(c *gin.Context, d interface{}) {
		renderedData = d.(data.AuthData)
		c.Status(http.StatusOK)
	}

	r.POST("/forgot-password", controller.ForgotPasswordHandler)

	// Create a request with form data
	form := url.Values{}
	form.Add("email", "test@example.com")
	req := httptest.NewRequest(http.MethodPost, "/forgot-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	// Perform the request
	r.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, renderedData.Error)
	mockDB.AssertExpectations(t)
	mockEmailService.AssertExpectations(t)
}
