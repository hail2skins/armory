package controller_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDB is a mock implementation of the database.Service interface
type MockDB struct {
	mock.Mock
}

// Health returns a mock health status
func (m *MockDB) Health() map[string]string {
	args := m.Called()
	return args.Get(0).(map[string]string)
}

// Close mocks closing the database connection
func (m *MockDB) Close() error {
	args := m.Called()
	return args.Error(0)
}

// CreateUser mocks creating a user
func (m *MockDB) CreateUser(ctx context.Context, email, password string) (*database.User, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// GetUserByEmail mocks retrieving a user by email
func (m *MockDB) GetUserByEmail(ctx context.Context, email string) (*database.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// AuthenticateUser mocks authenticating a user
func (m *MockDB) AuthenticateUser(ctx context.Context, email, password string) (*database.User, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// VerifyUserEmail mocks verifying a user's email
func (m *MockDB) VerifyUserEmail(ctx context.Context, token string) (*database.User, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// GetUserByVerificationToken mocks retrieving a user by verification token
func (m *MockDB) GetUserByVerificationToken(ctx context.Context, token string) (*database.User, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// GetUserByRecoveryToken mocks retrieving a user by recovery token
func (m *MockDB) GetUserByRecoveryToken(ctx context.Context, token string) (*database.User, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// UpdateUser mocks updating a user
func (m *MockDB) UpdateUser(ctx context.Context, user *database.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

// RequestPasswordReset mocks requesting a password reset
func (m *MockDB) RequestPasswordReset(ctx context.Context, email string) (*database.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// ResetPassword mocks resetting a password
func (m *MockDB) ResetPassword(ctx context.Context, token, newPassword string) error {
	args := m.Called(ctx, token, newPassword)
	return args.Error(0)
}

// CreatePayment mocks creating a payment
func (m *MockDB) CreatePayment(payment *database.Payment) error {
	args := m.Called(payment)
	return args.Error(0)
}

// GetPaymentsByUserID mocks retrieving payments by user ID
func (m *MockDB) GetPaymentsByUserID(userID uint) ([]database.Payment, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]database.Payment), args.Error(1)
}

// FindPaymentByID mocks retrieving a payment by ID
func (m *MockDB) FindPaymentByID(id uint) (*database.Payment, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.Payment), args.Error(1)
}

// UpdatePayment mocks updating a payment
func (m *MockDB) UpdatePayment(payment *database.Payment) error {
	args := m.Called(payment)
	return args.Error(0)
}

// GetUserByID mocks retrieving a user by ID
func (m *MockDB) GetUserByID(id uint) (*database.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// GetUserByStripeCustomerID mocks retrieving a user by Stripe customer ID
func (m *MockDB) GetUserByStripeCustomerID(customerID string) (*database.User, error) {
	args := m.Called(customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// MockUser implements the models.User interface for testing
type MockUser struct {
	Email string
}

func (m *MockUser) GetUserName() string {
	return m.Email
}

func (m *MockUser) GetID() uint {
	return 1
}

// MockAuthInfo implements the auth.Info interface for testing
type MockAuthInfo struct {
	Email string
}

func (m *MockAuthInfo) GetUserName() string {
	return m.Email
}

func (m *MockAuthInfo) GetID() string {
	return "1"
}

// MockAuthController is a mock implementation of the AuthController
type MockAuthController struct {
	User          *MockAuthInfo
	Authenticated bool
}

// GetCurrentUser returns a mock user and authentication status
func (m *MockAuthController) GetCurrentUser(c *gin.Context) (interface{ GetUserName() string }, bool) {
	return m.User, m.Authenticated
}

func TestPaymentController_PricingHandler(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockDB := &MockDB{}

	// Set up expectations for the mock DB
	mockDB.On("Health").Return(map[string]string{"status": "up"})

	tests := []struct {
		name          string
		authenticated bool
		email         string
		wantStatus    int
		wantContent   []string
	}{
		{
			name:          "Unauthenticated user",
			authenticated: false,
			email:         "",
			wantStatus:    http.StatusOK,
			wantContent:   []string{"Simple, transparent pricing", "Free", "Liking It", "Loving It", "Big Baller", "Frequently asked questions"},
		},
		{
			name:          "Authenticated user",
			authenticated: true,
			email:         "test@example.com",
			wantStatus:    http.StatusOK,
			wantContent:   []string{"Simple, transparent pricing", "Free", "Liking It", "Loving It", "Big Baller", "Frequently asked questions"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new gin context
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			// Set up the mock auth controller
			mockAuthController := &MockAuthController{
				User:          &MockAuthInfo{Email: tt.email},
				Authenticated: tt.authenticated,
			}

			// Create a new payment controller
			paymentController := controller.NewPaymentController(mockDB)

			// Set up mock expectations
			if tt.authenticated {
				// Create a mock user for the database response
				dbUser := &database.User{
					SubscriptionTier: "free",
				}
				mockDB.On("GetUserByEmail", mock.Anything, tt.email).Return(dbUser, nil)
			}

			// Register middleware to set the auth controller in the context
			r.Use(func(c *gin.Context) {
				c.Set("authController", mockAuthController)
				c.Next()
			})

			// Register the route
			r.GET("/pricing", paymentController.PricingHandler)

			// Create a new request
			req, _ := http.NewRequest(http.MethodGet, "/pricing", nil)

			// Serve the request
			r.ServeHTTP(w, req)

			// Assert the response
			assert.Equal(t, tt.wantStatus, w.Code)

			// Check for expected content
			for _, content := range tt.wantContent {
				assert.Contains(t, w.Body.String(), content)
			}
		})
	}
}
