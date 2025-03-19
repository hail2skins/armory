package mocks

import (
	"context"

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
	return m.Called().Error(0)
}

func (m *MockDB) Health() map[string]string {
	return m.Called().Get(0).(map[string]string)
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
	return m.Called(ctx, user).Error(0)
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
	return m.Called(ctx, token, newPassword).Error(0)
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
	return m.Called(payment).Error(0)
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
	return m.Called(payment).Error(0)
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
	return m.Called(manufacturer).Error(0)
}

func (m *MockDB) UpdateManufacturer(manufacturer *models.Manufacturer) error {
	return m.Called(manufacturer).Error(0)
}

func (m *MockDB) DeleteManufacturer(id uint) error {
	return m.Called(id).Error(0)
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
	return m.Called(caliber).Error(0)
}

func (m *MockDB) UpdateCaliber(caliber *models.Caliber) error {
	return m.Called(caliber).Error(0)
}

func (m *MockDB) DeleteCaliber(id uint) error {
	return m.Called(id).Error(0)
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
	return m.Called(weaponType).Error(0)
}

func (m *MockDB) UpdateWeaponType(weaponType *models.WeaponType) error {
	return m.Called(weaponType).Error(0)
}

func (m *MockDB) DeleteWeaponType(id uint) error {
	return m.Called(id).Error(0)
}

func (m *MockDB) DeleteGun(db *gorm.DB, gunID uint, userID uint) error {
	return m.Called(db, gunID, userID).Error(0)
}

// NewHomeController is a method used for reflection-based testing
func (m *MockDB) NewHomeController(db database.Service) interface{} {
	args := m.Called(db)
	return args.Get(0)
}

// Factory function to create a mock implementation of a controller
func (m *MockDB) MockController(name string, args ...interface{}) interface{} {
	callArgs := make([]interface{}, 0, len(args)+1)
	callArgs = append(callArgs, name)
	callArgs = append(callArgs, args...)
	return m.Called(callArgs...).Get(0)
}

// CountUsers mocks the CountUsers method
func (m *MockDB) CountUsers() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

// FindRecentUsers mocks the FindRecentUsers method
func (m *MockDB) FindRecentUsers(offset, limit int, sortBy, sortOrder string) ([]database.User, error) {
	args := m.Called(offset, limit, sortBy, sortOrder)
	return args.Get(0).([]database.User), args.Error(1)
}
