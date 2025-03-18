package testing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// Initialize test mode for all controller tests
func init() {
	gin.SetMode(gin.TestMode)
}

// TestRequest represents a test HTTP request configuration
type TestRequest struct {
	Method string
	Path   string
	Body   interface{}
	Header map[string]string
}

// TestResponse helps verify HTTP responses
type TestResponse struct {
	Recorder       *httptest.ResponseRecorder
	ExpectedCode   int
	ExpectedBody   string
	ExpectedHeader map[string]string
}

// MockDB is a standardized mock database implementation for controller tests
type MockDB struct {
	mock.Mock
}

// -------- Mock DB implementation methods ----------

// GetDB mocks the GetDB method
func (m *MockDB) GetDB() *gorm.DB {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*gorm.DB)
}

// Health mocks the Health method
func (m *MockDB) Health() map[string]string {
	args := m.Called()
	return args.Get(0).(map[string]string)
}

// Close mocks the Close method
func (m *MockDB) Close() error {
	args := m.Called()
	return args.Error(0)
}

// GetUserByEmail mocks the GetUserByEmail method
func (m *MockDB) GetUserByEmail(ctx context.Context, email string) (*database.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// GetUserByID mocks the GetUserByID method
func (m *MockDB) GetUserByID(id uint) (*database.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// CreateUser mocks the CreateUser method
func (m *MockDB) CreateUser(ctx context.Context, email, password string) (*database.User, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// UpdateUser mocks the UpdateUser method
func (m *MockDB) UpdateUser(ctx context.Context, user *database.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

// AuthenticateUser mocks the AuthenticateUser method
func (m *MockDB) AuthenticateUser(ctx context.Context, email, password string) (*database.User, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// GetUserByVerificationToken mocks the GetUserByVerificationToken method
func (m *MockDB) GetUserByVerificationToken(ctx context.Context, token string) (*database.User, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// GetUserByRecoveryToken mocks the GetUserByRecoveryToken method
func (m *MockDB) GetUserByRecoveryToken(ctx context.Context, token string) (*database.User, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// VerifyUserEmail mocks the VerifyUserEmail method
func (m *MockDB) VerifyUserEmail(ctx context.Context, token string) (*database.User, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// RequestPasswordReset mocks the RequestPasswordReset method
func (m *MockDB) RequestPasswordReset(ctx context.Context, email string) (*database.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// ResetPassword mocks the ResetPassword method
func (m *MockDB) ResetPassword(ctx context.Context, token, newPassword string) error {
	args := m.Called(ctx, token, newPassword)
	return args.Error(0)
}

// IsRecoveryExpired mocks the IsRecoveryExpired method
func (m *MockDB) IsRecoveryExpired(ctx context.Context, token string) (bool, error) {
	args := m.Called(ctx, token)
	return args.Bool(0), args.Error(1)
}

// GetUserByStripeCustomerID mocks the GetUserByStripeCustomerID method
func (m *MockDB) GetUserByStripeCustomerID(customerID string) (*database.User, error) {
	args := m.Called(customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// FindAllWeaponTypes mocks the FindAllWeaponTypes method
func (m *MockDB) FindAllWeaponTypes() ([]models.WeaponType, error) {
	args := m.Called()
	return args.Get(0).([]models.WeaponType), args.Error(1)
}

// FindWeaponTypeByID mocks the FindWeaponTypeByID method
func (m *MockDB) FindWeaponTypeByID(id uint) (*models.WeaponType, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.WeaponType), args.Error(1)
}

// CreateWeaponType mocks the CreateWeaponType method
func (m *MockDB) CreateWeaponType(weaponType *models.WeaponType) error {
	args := m.Called(weaponType)
	return args.Error(0)
}

// UpdateWeaponType mocks the UpdateWeaponType method
func (m *MockDB) UpdateWeaponType(weaponType *models.WeaponType) error {
	args := m.Called(weaponType)
	return args.Error(0)
}

// DeleteWeaponType mocks the DeleteWeaponType method
func (m *MockDB) DeleteWeaponType(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

// FindAllCalibers mocks the FindAllCalibers method
func (m *MockDB) FindAllCalibers() ([]models.Caliber, error) {
	args := m.Called()
	return args.Get(0).([]models.Caliber), args.Error(1)
}

// FindCaliberByID mocks the FindCaliberByID method
func (m *MockDB) FindCaliberByID(id uint) (*models.Caliber, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Caliber), args.Error(1)
}

// CreateCaliber mocks the CreateCaliber method
func (m *MockDB) CreateCaliber(caliber *models.Caliber) error {
	args := m.Called(caliber)
	return args.Error(0)
}

// UpdateCaliber mocks the UpdateCaliber method
func (m *MockDB) UpdateCaliber(caliber *models.Caliber) error {
	args := m.Called(caliber)
	return args.Error(0)
}

// DeleteCaliber mocks the DeleteCaliber method
func (m *MockDB) DeleteCaliber(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

// FindAllManufacturers mocks the FindAllManufacturers method
func (m *MockDB) FindAllManufacturers() ([]models.Manufacturer, error) {
	args := m.Called()
	return args.Get(0).([]models.Manufacturer), args.Error(1)
}

// FindManufacturerByID mocks the FindManufacturerByID method
func (m *MockDB) FindManufacturerByID(id uint) (*models.Manufacturer, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Manufacturer), args.Error(1)
}

// CreateManufacturer mocks the CreateManufacturer method
func (m *MockDB) CreateManufacturer(manufacturer *models.Manufacturer) error {
	args := m.Called(manufacturer)
	return args.Error(0)
}

// UpdateManufacturer mocks the UpdateManufacturer method
func (m *MockDB) UpdateManufacturer(manufacturer *models.Manufacturer) error {
	args := m.Called(manufacturer)
	return args.Error(0)
}

// DeleteManufacturer mocks the DeleteManufacturer method
func (m *MockDB) DeleteManufacturer(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

// DeleteGun mocks the DeleteGun method
func (m *MockDB) DeleteGun(db *gorm.DB, id uint, ownerID uint) error {
	args := m.Called(db, id, ownerID)
	return args.Error(0)
}

// MockUser implements the models.User interface for testing
type MockUser struct {
	UserID   uint
	UserName string
}

// GetID returns the mock user ID
func (m *MockUser) GetID() uint {
	return m.UserID
}

// GetUserName returns the mock user name
func (m *MockUser) GetUserName() string {
	return m.UserName
}

// MockAuthController mocks the authentication controller
type MockAuthController struct {
	mock.Mock
}

// GetCurrentUser mocks the GetCurrentUser method
func (m *MockAuthController) GetCurrentUser(c *gin.Context) (models.User, bool) {
	args := m.Called(c)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(models.User), args.Bool(1)
}

// SetupTestRouter creates a new Gin router with common test middleware
func SetupTestRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	// Set up flash middleware
	router.Use(func(c *gin.Context) {
		c.Set("setFlash", func(msg string) {
			c.Set("flash_message", msg)
		})
		c.Next()
	})

	return router
}

// SetupTestRouterWithAuth creates a router with auth middleware and mock auth controller
func SetupTestRouterWithAuth() (*gin.Engine, *MockAuthController) {
	router := SetupTestRouter()

	// Create mock auth controller
	mockAuth := new(MockAuthController)

	// Make auth controller available in context
	router.Use(func(c *gin.Context) {
		c.Set("authController", mockAuth)
		c.Next()
	})

	return router, mockAuth
}

// SetupAuthUser configures a mock auth controller to return a test user
func SetupAuthUser(mockAuth *MockAuthController, userID uint, userName string) *MockUser {
	user := &MockUser{
		UserID:   userID,
		UserName: userName,
	}
	mockAuth.On("GetCurrentUser", mock.Anything).Return(user, true)
	return user
}

// CreateTestUser returns a test database.User for testing
func CreateTestUser() *database.User {
	return &database.User{
		Model: gorm.Model{
			ID: 1,
		},
		Email:    "test@example.com",
		Password: "hashed_password",
		Verified: true,
	}
}

// MakeRequest helper for making HTTP requests in tests
func MakeRequest(router *gin.Engine, method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	var reqBody string

	switch v := body.(type) {
	case string:
		reqBody = v
	case url.Values:
		reqBody = v.Encode()
	default:
		reqBody = ""
	}

	req, _ := http.NewRequest(method, path, strings.NewReader(reqBody))

	// Set default content type for forms
	if _, ok := body.(url.Values); ok {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	// Set custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Record the response
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w
}

// AssertResponse verifies that the response matches expectations
func AssertResponse(t *testing.T, resp *TestResponse) {
	if resp.ExpectedCode != 0 {
		assert.Equal(t, resp.ExpectedCode, resp.Recorder.Code)
	}

	if resp.ExpectedBody != "" {
		assert.Contains(t, resp.Recorder.Body.String(), resp.ExpectedBody)
	}

	for key, value := range resp.ExpectedHeader {
		assert.Equal(t, value, resp.Recorder.Header().Get(key))
	}
}

// SetupTestWithFlash creates a router with flash middleware and session data
func SetupTestWithFlash() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	// Mock session data
	router.Use(func(c *gin.Context) {
		// Create session data storage
		sessionData := make(map[string]interface{})
		c.Set("session", sessionData)

		// Add flash middleware
		c.Set("setFlash", func(msg string) {
			sessionData["flash"] = msg
		})

		// Add getFlash middleware
		c.Set("getFlash", func() string {
			if flash, exists := sessionData["flash"]; exists {
				delete(sessionData, "flash")
				return flash.(string)
			}
			return ""
		})

		// Add session method to store and retrieve arbitrary data
		c.Set("setSessionValue", func(key string, value interface{}) {
			sessionData[key] = value
		})

		c.Set("getSessionValue", func(key string) (interface{}, bool) {
			value, exists := sessionData[key]
			return value, exists
		})

		c.Next()
	})

	return router
}

// SetupTestWithAuthenticatedUser creates a router with flash middleware and authenticated user
func SetupTestWithAuthenticatedUser(userID uint, email string) (*gin.Engine, *MockUser) {
	router := SetupTestWithFlash()

	// Create mock user
	mockUser := &MockUser{
		UserID:   userID,
		UserName: email,
	}

	// Add authentication middleware
	router.Use(func(c *gin.Context) {
		c.Set("user", mockUser)
		c.Set("authenticated", true)
		c.Next()
	})

	return router, mockUser
}

// TestWithDB runs a test with a configured MockDB
func TestWithDB(t *testing.T, configureDB func(*MockDB), testFn func(*gin.Engine, *MockDB)) {
	mockDB := new(MockDB)

	// Allow the test to configure mock expectations
	configureDB(mockDB)

	// Setup router with flash middleware
	router := SetupTestWithFlash()

	// Run the test
	testFn(router, mockDB)

	// Verify all expectations were met
	mockDB.AssertExpectations(t)
}

// TestWithAuthenticatedDB runs a test with an authenticated user and MockDB
func TestWithAuthenticatedDB(t *testing.T, userID uint, email string, configureDB func(*MockDB),
	testFn func(*gin.Engine, *MockDB, *MockUser)) {
	mockDB := new(MockDB)

	// Allow the test to configure mock expectations
	configureDB(mockDB)

	// Setup router with flash middleware and authenticated user
	router, mockUser := SetupTestWithAuthenticatedUser(userID, email)

	// Run the test
	testFn(router, mockDB, mockUser)

	// Verify all expectations were met
	mockDB.AssertExpectations(t)
}
