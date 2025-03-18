package testutils

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// MockDB is a standardized mock database implementation
type MockDB struct {
	mock.Mock
}

// Set up standard mock methods to implement database.Service
func (m *MockDB) GetDB() *gorm.DB {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*gorm.DB)
}

func (m *MockDB) Close() error {
	return nil
}

func (m *MockDB) Health() map[string]string {
	return map[string]string{"status": "up"}
}

func (m *MockDB) GetUserByEmail(ctx context.Context, email string) (*database.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

func (m *MockDB) CreateUser(ctx context.Context, email string, password string) (*database.User, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

func (m *MockDB) UpdateUser(ctx context.Context, user *database.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockDB) AuthenticateUser(ctx context.Context, email, password string) (*database.User, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

func (m *MockDB) GetUserByID(id uint) (*database.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

func (m *MockDB) GetUserByVerificationToken(ctx context.Context, token string) (*database.User, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

func (m *MockDB) GetUserByRecoveryToken(ctx context.Context, token string) (*database.User, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

func (m *MockDB) VerifyUserEmail(ctx context.Context, token string) (*database.User, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

func (m *MockDB) RequestPasswordReset(ctx context.Context, email string) (*database.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

func (m *MockDB) ResetPassword(ctx context.Context, token, newPassword string) error {
	args := m.Called(ctx, token, newPassword)
	return args.Error(0)
}

func (m *MockDB) IsRecoveryExpired(ctx context.Context, token string) (bool, error) {
	args := m.Called(ctx, token)
	return args.Bool(0), args.Error(1)
}

func (m *MockDB) GetUserByStripeCustomerID(customerID string) (*database.User, error) {
	args := m.Called(customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

func (m *MockDB) CreatePayment(payment *models.Payment) error {
	args := m.Called(payment)
	return args.Error(0)
}

func (m *MockDB) GetPaymentsByUserID(userID uint) ([]models.Payment, error) {
	args := m.Called(userID)
	return args.Get(0).([]models.Payment), args.Error(1)
}

func (m *MockDB) FindPaymentByID(id uint) (*models.Payment, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Payment), args.Error(1)
}

func (m *MockDB) UpdatePayment(payment *models.Payment) error {
	args := m.Called(payment)
	return args.Error(0)
}

func (m *MockDB) FindAllManufacturers() ([]models.Manufacturer, error) {
	args := m.Called()
	return args.Get(0).([]models.Manufacturer), args.Error(1)
}

func (m *MockDB) FindManufacturerByID(id uint) (*models.Manufacturer, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Manufacturer), args.Error(1)
}

func (m *MockDB) CreateManufacturer(manufacturer *models.Manufacturer) error {
	args := m.Called(manufacturer)
	return args.Error(0)
}

func (m *MockDB) UpdateManufacturer(manufacturer *models.Manufacturer) error {
	args := m.Called(manufacturer)
	return args.Error(0)
}

func (m *MockDB) DeleteManufacturer(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockDB) FindAllCalibers() ([]models.Caliber, error) {
	args := m.Called()
	return args.Get(0).([]models.Caliber), args.Error(1)
}

func (m *MockDB) FindCaliberByID(id uint) (*models.Caliber, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Caliber), args.Error(1)
}

func (m *MockDB) CreateCaliber(caliber *models.Caliber) error {
	args := m.Called(caliber)
	return args.Error(0)
}

func (m *MockDB) UpdateCaliber(caliber *models.Caliber) error {
	args := m.Called(caliber)
	return args.Error(0)
}

func (m *MockDB) DeleteCaliber(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockDB) FindAllWeaponTypes() ([]models.WeaponType, error) {
	args := m.Called()
	return args.Get(0).([]models.WeaponType), args.Error(1)
}

func (m *MockDB) FindWeaponTypeByID(id uint) (*models.WeaponType, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.WeaponType), args.Error(1)
}

func (m *MockDB) CreateWeaponType(weaponType *models.WeaponType) error {
	args := m.Called(weaponType)
	return args.Error(0)
}

func (m *MockDB) UpdateWeaponType(weaponType *models.WeaponType) error {
	args := m.Called(weaponType)
	return args.Error(0)
}

func (m *MockDB) DeleteWeaponType(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockDB) DeleteGun(db *gorm.DB, gunID uint, userID uint) error {
	args := m.Called(db, gunID, userID)
	return args.Error(0)
}

// MockAuthController is a standardized mock auth controller implementation
type MockAuthController struct {
	mock.Mock
}

// Implement AuthControllerInterface
func (m *MockAuthController) GetCurrentUser(c *gin.Context) (interface{}, bool) {
	args := m.Called(c)
	return args.Get(0), args.Bool(1)
}

// NewMockTestSetup creates a complete mock test setup
func NewMockTestSetup() (*gin.Engine, *MockDB, *MockAuthController) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a new router
	router := gin.New()
	router.Use(gin.Recovery())

	// Create mock objects
	mockDB := new(MockDB)
	mockAuthController := new(MockAuthController)

	// Set up flash middleware
	router.Use(func(c *gin.Context) {
		c.Set("setFlash", func(msg string) {
			c.Set("flash_message", msg)
		})
		c.Next()
	})

	// Make auth controller available in context
	router.Use(func(c *gin.Context) {
		c.Set("authController", mockAuthController)
		c.Next()
	})

	return router, mockDB, mockAuthController
}

// SetupMockUser configures standard mocks for an authenticated user
func SetupMockUser(mockDB *MockDB, mockAuth *MockAuthController, user *database.User) {
	// Setup user retrieval
	mockDB.On("GetUserByEmail", mock.Anything, user.Email).Return(user, nil)

	// Setup authentication
	mockAuth.On("GetCurrentUser", mock.Anything).Return(user, true)
}

// SetupTestUser creates a test user for use in tests
func SetupTestUser() *database.User {
	user := &database.User{
		Email:    "test@example.com",
		Verified: true,
	}
	// Set the ID using gorm.Model embedded struct
	user.Model.ID = 1
	return user
}

// CreateMockControllers creates controllers with the mock database
func CreateMockControllers(mockDB *MockDB) (*controller.AuthController, *controller.OwnerController) {
	authController := controller.NewAuthController(mockDB)
	ownerController := controller.NewOwnerController(mockDB)
	return authController, ownerController
}
