package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
	"github.com/shaj13/go-guardian/v2/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// MockAuthController is a mock implementation of the AuthController
type MockAuthController struct {
	mock.Mock
}

// GetCurrentUser returns a mock user for testing
func (m *MockAuthController) GetCurrentUser(c *gin.Context) (auth.Info, bool) {
	return auth.NewUserInfo("test@example.com", "1", nil, nil), true
}

// MockAdminDB is a mock implementation of the database.Service interface
type MockAdminDB struct {
	mock.Mock
}

// Health returns a map of health status information
func (m *MockAdminDB) Health() map[string]string {
	args := m.Called()
	return args.Get(0).(map[string]string)
}

// Close terminates the database connection
func (m *MockAdminDB) Close() error {
	args := m.Called()
	return args.Error(0)
}

// CreateUser creates a new user
func (m *MockAdminDB) CreateUser(ctx context.Context, email, password string) (*database.User, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// GetUserByEmail retrieves a user by email
func (m *MockAdminDB) GetUserByEmail(ctx context.Context, email string) (*database.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// AuthenticateUser authenticates a user
func (m *MockAdminDB) AuthenticateUser(ctx context.Context, email, password string) (*database.User, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// VerifyUserEmail verifies a user's email
func (m *MockAdminDB) VerifyUserEmail(ctx context.Context, token string) (*database.User, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// GetUserByVerificationToken retrieves a user by verification token
func (m *MockAdminDB) GetUserByVerificationToken(ctx context.Context, token string) (*database.User, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// GetUserByRecoveryToken retrieves a user by recovery token
func (m *MockAdminDB) GetUserByRecoveryToken(ctx context.Context, token string) (*database.User, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// UpdateUser updates a user
func (m *MockAdminDB) UpdateUser(ctx context.Context, user *database.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

// RequestPasswordReset requests a password reset
func (m *MockAdminDB) RequestPasswordReset(ctx context.Context, email string) (*database.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// ResetPassword resets a user's password
func (m *MockAdminDB) ResetPassword(ctx context.Context, token, newPassword string) error {
	args := m.Called(ctx, token, newPassword)
	return args.Error(0)
}

// CreatePayment creates a payment
func (m *MockAdminDB) CreatePayment(payment *models.Payment) error {
	args := m.Called(payment)
	return args.Error(0)
}

// GetPaymentsByUserID retrieves payments by user ID
func (m *MockAdminDB) GetPaymentsByUserID(userID uint) ([]models.Payment, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Payment), args.Error(1)
}

// FindPaymentByID retrieves a payment by ID
func (m *MockAdminDB) FindPaymentByID(id uint) (*models.Payment, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Payment), args.Error(1)
}

// UpdatePayment updates a payment
func (m *MockAdminDB) UpdatePayment(payment *models.Payment) error {
	args := m.Called(payment)
	return args.Error(0)
}

// GetUserByID retrieves a user by ID
func (m *MockAdminDB) GetUserByID(id uint) (*database.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// GetUserByStripeCustomerID retrieves a user by Stripe customer ID
func (m *MockAdminDB) GetUserByStripeCustomerID(customerID string) (*database.User, error) {
	args := m.Called(customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// FindAllManufacturers retrieves all manufacturers
func (m *MockAdminDB) FindAllManufacturers() ([]models.Manufacturer, error) {
	args := m.Called()
	manufacturers := args.Get(0).([]*models.Manufacturer)
	result := make([]models.Manufacturer, len(manufacturers))
	for i, m := range manufacturers {
		result[i] = *m
	}
	return result, args.Error(1)
}

// FindManufacturerByID retrieves a manufacturer by ID
func (m *MockAdminDB) FindManufacturerByID(id uint) (*models.Manufacturer, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Manufacturer), args.Error(1)
}

// CreateManufacturer creates a new manufacturer
func (m *MockAdminDB) CreateManufacturer(manufacturer *models.Manufacturer) error {
	args := m.Called(manufacturer)
	return args.Error(0)
}

// UpdateManufacturer updates a manufacturer
func (m *MockAdminDB) UpdateManufacturer(manufacturer *models.Manufacturer) error {
	args := m.Called(manufacturer)
	return args.Error(0)
}

// DeleteManufacturer deletes a manufacturer
func (m *MockAdminDB) DeleteManufacturer(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

// FindAllCalibers retrieves all calibers
func (m *MockAdminDB) FindAllCalibers() ([]models.Caliber, error) {
	args := m.Called()
	calibers := args.Get(0).([]*models.Caliber)
	result := make([]models.Caliber, len(calibers))
	for i, c := range calibers {
		result[i] = *c
	}
	return result, args.Error(1)
}

// FindCaliberByID retrieves a caliber by ID
func (m *MockAdminDB) FindCaliberByID(id uint) (*models.Caliber, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Caliber), args.Error(1)
}

// CreateCaliber creates a new caliber
func (m *MockAdminDB) CreateCaliber(caliber *models.Caliber) error {
	args := m.Called(caliber)
	return args.Error(0)
}

// UpdateCaliber updates a caliber
func (m *MockAdminDB) UpdateCaliber(caliber *models.Caliber) error {
	args := m.Called(caliber)
	return args.Error(0)
}

// DeleteCaliber deletes a caliber
func (m *MockAdminDB) DeleteCaliber(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

// FindAllWeaponTypes retrieves all weapon types
func (m *MockAdminDB) FindAllWeaponTypes() ([]models.WeaponType, error) {
	args := m.Called()
	weaponTypes := args.Get(0).([]*models.WeaponType)
	result := make([]models.WeaponType, len(weaponTypes))
	for i, w := range weaponTypes {
		result[i] = *w
	}
	return result, args.Error(1)
}

// FindWeaponTypeByID retrieves a weapon type by ID
func (m *MockAdminDB) FindWeaponTypeByID(id uint) (*models.WeaponType, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.WeaponType), args.Error(1)
}

// CreateWeaponType creates a new weapon type
func (m *MockAdminDB) CreateWeaponType(weaponType *models.WeaponType) error {
	args := m.Called(weaponType)
	return args.Error(0)
}

// UpdateWeaponType updates a weapon type
func (m *MockAdminDB) UpdateWeaponType(weaponType *models.WeaponType) error {
	args := m.Called(weaponType)
	return args.Error(0)
}

// GetDB returns the underlying *gorm.DB instance
func (m *MockAdminDB) GetDB() *gorm.DB {
	args := m.Called()
	return args.Get(0).(*gorm.DB)
}

// DeleteWeaponType deletes a weapon type
func (m *MockAdminDB) DeleteWeaponType(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

// DeleteGun deletes a gun from the database
func (m *MockAdminDB) DeleteGun(db *gorm.DB, id uint, ownerID uint) error {
	args := m.Called(db, id, ownerID)
	return args.Error(0)
}

// IsRecoveryExpired is a mock method to satisfy the database.Service interface
func (m *MockAdminDB) IsRecoveryExpired(ctx context.Context, token string) (bool, error) {
	args := m.Called(ctx, token)
	return args.Bool(0), args.Error(1)
}

// MockResponseWriter is a mock implementation of http.ResponseWriter
type MockResponseWriter struct {
	mock.Mock
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// Header returns a mock header
func (m *MockResponseWriter) Header() http.Header {
	if m.Headers == nil {
		m.Headers = make(http.Header)
	}
	return m.Headers
}

// Write writes to the mock body
func (m *MockResponseWriter) Write(b []byte) (int, error) {
	if m.StatusCode == 0 {
		m.StatusCode = http.StatusOK
	}
	m.Body = append(m.Body, b...)
	return len(b), nil
}

// WriteHeader sets the mock status code
func (m *MockResponseWriter) WriteHeader(statusCode int) {
	m.StatusCode = statusCode
}

// TestAdminManufacturerRoutes tests the admin manufacturer routes
func TestAdminManufacturerRoutes(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create a mock DB
	mockDB := new(MockAdminDB)

	// Create a mock manufacturer
	manufacturer := &models.Manufacturer{
		Model:      gorm.Model{ID: 1},
		Name:       "Test Manufacturer",
		Nickname:   "Test",
		Country:    "Test Country",
		Popularity: 1,
	}

	// Set up expectations
	mockDB.On("FindAllManufacturers").Return([]*models.Manufacturer{manufacturer}, nil)
	mockDB.On("FindManufacturerByID", uint(1)).Return(manufacturer, nil)
	mockDB.On("CreateManufacturer", mock.AnythingOfType("*models.Manufacturer")).Return(nil)
	mockDB.On("UpdateManufacturer", mock.AnythingOfType("*models.Manufacturer")).Return(nil)
	mockDB.On("DeleteManufacturer", uint(1)).Return(nil)
	mockDB.On("GetDB").Return(&gorm.DB{}).Maybe()

	// Create a test HTTP server
	router := gin.New()

	// Create an admin manufacturer controller
	adminController := NewAdminManufacturerController(mockDB)

	// Register the routes
	router.GET("/admin/manufacturers", adminController.Index)
	router.GET("/admin/manufacturers/new", adminController.New)
	router.POST("/admin/manufacturers", adminController.Create)
	router.GET("/admin/manufacturers/:id", adminController.Show)
	router.GET("/admin/manufacturers/:id/edit", adminController.Edit)
	router.POST("/admin/manufacturers/:id", adminController.Update)
	router.POST("/admin/manufacturers/:id/delete", adminController.Delete)

	// Test cases for GET routes
	getTests := []struct {
		name       string
		path       string
		statusCode int
	}{
		{
			name:       "Index",
			path:       "/admin/manufacturers",
			statusCode: http.StatusOK,
		},
		{
			name:       "New",
			path:       "/admin/manufacturers/new",
			statusCode: http.StatusOK,
		},
		{
			name:       "Show",
			path:       "/admin/manufacturers/1",
			statusCode: http.StatusOK,
		},
		{
			name:       "Edit",
			path:       "/admin/manufacturers/1/edit",
			statusCode: http.StatusOK,
		},
	}

	for _, tt := range getTests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test request
			req, _ := http.NewRequest("GET", tt.path, nil)
			resp := httptest.NewRecorder()

			// Serve the request
			router.ServeHTTP(resp, req)

			// Assert response
			assert.Equal(t, tt.statusCode, resp.Code)
		})
	}

	// Test Create route
	t.Run("Create", func(t *testing.T) {
		formData := "name=New+Manufacturer&nickname=New&country=New+Country"
		req, _ := http.NewRequest("POST", "/admin/manufacturers", strings.NewReader(formData))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusSeeOther, resp.Code)
	})

	// Test Update route
	t.Run("Update", func(t *testing.T) {
		formData := "name=Updated+Manufacturer&nickname=Updated&country=Updated+Country"
		req, _ := http.NewRequest("POST", "/admin/manufacturers/1", strings.NewReader(formData))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusSeeOther, resp.Code)
	})

	// Test Delete route
	t.Run("Delete", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/admin/manufacturers/1/delete", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusSeeOther, resp.Code)
	})
}

// TestAdminCaliberRoutes tests the admin caliber routes
func TestAdminCaliberRoutes(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create a mock DB
	mockDB := new(MockAdminDB)

	// Create a mock caliber
	caliber := &models.Caliber{
		Model:      gorm.Model{ID: 1},
		Caliber:    "Test Caliber",
		Nickname:   "Test",
		Popularity: 1,
	}

	// Set up expectations
	mockDB.On("FindAllCalibers").Return([]*models.Caliber{caliber}, nil)
	mockDB.On("FindCaliberByID", uint(1)).Return(caliber, nil)
	mockDB.On("CreateCaliber", mock.AnythingOfType("*models.Caliber")).Return(nil)
	mockDB.On("UpdateCaliber", mock.AnythingOfType("*models.Caliber")).Return(nil)
	mockDB.On("DeleteCaliber", uint(1)).Return(nil)

	// Create a test HTTP server
	router := gin.New()

	// Create an admin caliber controller
	adminController := NewAdminCaliberController(mockDB)

	// Register the routes
	router.GET("/admin/calibers", adminController.Index)
	router.GET("/admin/calibers/new", adminController.New)
	router.POST("/admin/calibers", adminController.Create)
	router.GET("/admin/calibers/:id", adminController.Show)
	router.GET("/admin/calibers/:id/edit", adminController.Edit)
	router.POST("/admin/calibers/:id", adminController.Update)
	router.POST("/admin/calibers/:id/delete", adminController.Delete)

	// Test cases for GET routes
	getTests := []struct {
		name       string
		path       string
		statusCode int
	}{
		{
			name:       "Index",
			path:       "/admin/calibers",
			statusCode: http.StatusOK,
		},
		{
			name:       "New",
			path:       "/admin/calibers/new",
			statusCode: http.StatusOK,
		},
		{
			name:       "Show",
			path:       "/admin/calibers/1",
			statusCode: http.StatusOK,
		},
		{
			name:       "Edit",
			path:       "/admin/calibers/1/edit",
			statusCode: http.StatusOK,
		},
	}

	for _, tt := range getTests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test request
			req, _ := http.NewRequest("GET", tt.path, nil)
			resp := httptest.NewRecorder()

			// Serve the request
			router.ServeHTTP(resp, req)

			// Assert response
			assert.Equal(t, tt.statusCode, resp.Code)
		})
	}

	// Test Create route
	t.Run("Create", func(t *testing.T) {
		formData := "caliber=New+Caliber&nickname=New"
		req, _ := http.NewRequest("POST", "/admin/calibers", strings.NewReader(formData))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusSeeOther, resp.Code)
	})

	// Test Update route
	t.Run("Update", func(t *testing.T) {
		formData := "caliber=Updated+Caliber&nickname=Updated"
		req, _ := http.NewRequest("POST", "/admin/calibers/1", strings.NewReader(formData))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusSeeOther, resp.Code)
	})

	// Test Delete route
	t.Run("Delete", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/admin/calibers/1/delete", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusSeeOther, resp.Code)
	})
}

// TestAdminWeaponTypeRoutes tests the admin weapon type routes
func TestAdminWeaponTypeRoutes(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create a mock DB
	mockDB := new(MockAdminDB)

	// Create a mock weapon type
	weaponType := &models.WeaponType{
		ID:         1,
		Type:       "Test Type",
		Nickname:   "Test",
		Popularity: 1,
	}

	// Set up expectations
	mockDB.On("FindAllWeaponTypes").Return([]*models.WeaponType{weaponType}, nil)
	mockDB.On("FindWeaponTypeByID", uint(1)).Return(weaponType, nil)
	mockDB.On("CreateWeaponType", mock.AnythingOfType("*models.WeaponType")).Return(nil)
	mockDB.On("UpdateWeaponType", mock.AnythingOfType("*models.WeaponType")).Return(nil)
	mockDB.On("DeleteWeaponType", uint(1)).Return(nil)

	// Create a test HTTP server
	router := gin.New()

	// Create an admin weapon type controller
	adminController := NewAdminWeaponTypeController(mockDB)

	// Register the routes
	router.GET("/admin/weapon_types", adminController.Index)
	router.GET("/admin/weapon_types/new", adminController.New)
	router.POST("/admin/weapon_types", adminController.Create)
	router.GET("/admin/weapon_types/:id", adminController.Show)
	router.GET("/admin/weapon_types/:id/edit", adminController.Edit)
	router.POST("/admin/weapon_types/:id", adminController.Update)
	router.POST("/admin/weapon_types/:id/delete", adminController.Delete)

	// Test cases for GET routes
	getTests := []struct {
		name       string
		path       string
		statusCode int
	}{
		{
			name:       "Index",
			path:       "/admin/weapon_types",
			statusCode: http.StatusOK,
		},
		{
			name:       "New",
			path:       "/admin/weapon_types/new",
			statusCode: http.StatusOK,
		},
		{
			name:       "Show",
			path:       "/admin/weapon_types/1",
			statusCode: http.StatusOK,
		},
		{
			name:       "Edit",
			path:       "/admin/weapon_types/1/edit",
			statusCode: http.StatusOK,
		},
	}

	for _, tt := range getTests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test request
			req, _ := http.NewRequest("GET", tt.path, nil)
			resp := httptest.NewRecorder()

			// Serve the request
			router.ServeHTTP(resp, req)

			// Assert response
			assert.Equal(t, tt.statusCode, resp.Code)
		})
	}

	// Test Create route
	t.Run("Create", func(t *testing.T) {
		formData := "type=New+Type&nickname=New"
		req, _ := http.NewRequest("POST", "/admin/weapon_types", strings.NewReader(formData))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusSeeOther, resp.Code)
	})

	// Test Update route
	t.Run("Update", func(t *testing.T) {
		formData := "type=Updated+Type&nickname=Updated"
		req, _ := http.NewRequest("POST", "/admin/weapon_types/1", strings.NewReader(formData))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusSeeOther, resp.Code)
	})

	// Test Delete route
	t.Run("Delete", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/admin/weapon_types/1/delete", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusSeeOther, resp.Code)
	})
}
