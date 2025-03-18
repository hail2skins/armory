package mocks

import (
	"context"

	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// MockDB is a mock implementation of the database.Service interface
type MockDB struct {
	mock.Mock
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

// GetDB mocks the GetDB method
func (m *MockDB) GetDB() *gorm.DB {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*gorm.DB)
}

// CreateUser mocks the CreateUser method
func (m *MockDB) CreateUser(ctx context.Context, email, password string) (*database.User, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// GetUserByEmail mocks the GetUserByEmail method
func (m *MockDB) GetUserByEmail(ctx context.Context, email string) (*database.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// AuthenticateUser mocks the AuthenticateUser method
func (m *MockDB) AuthenticateUser(ctx context.Context, email, password string) (*database.User, error) {
	args := m.Called(ctx, email, password)
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

// UpdateUser mocks the UpdateUser method
func (m *MockDB) UpdateUser(ctx context.Context, user *database.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
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

// GetUserByID mocks the GetUserByID method
func (m *MockDB) GetUserByID(id uint) (*database.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// GetUserByStripeCustomerID mocks the GetUserByStripeCustomerID method
func (m *MockDB) GetUserByStripeCustomerID(customerID string) (*database.User, error) {
	args := m.Called(customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
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

// CreatePayment mocks the CreatePayment method
func (m *MockDB) CreatePayment(payment *models.Payment) error {
	args := m.Called(payment)
	return args.Error(0)
}

// GetPaymentsByUserID mocks the GetPaymentsByUserID method
func (m *MockDB) GetPaymentsByUserID(userID uint) ([]models.Payment, error) {
	args := m.Called(userID)
	return args.Get(0).([]models.Payment), args.Error(1)
}

// FindPaymentByID mocks the FindPaymentByID method
func (m *MockDB) FindPaymentByID(id uint) (*models.Payment, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Payment), args.Error(1)
}

// UpdatePayment mocks the UpdatePayment method
func (m *MockDB) UpdatePayment(payment *models.Payment) error {
	args := m.Called(payment)
	return args.Error(0)
}

// DeleteGun mocks the DeleteGun method
func (m *MockDB) DeleteGun(db *gorm.DB, id uint, ownerID uint) error {
	args := m.Called(db, id, ownerID)
	return args.Error(0)
}

// IsRecoveryExpired mocks the checking of whether a recovery token is expired
func (m *MockDB) IsRecoveryExpired(ctx context.Context, token string) (bool, error) {
	args := m.Called(ctx, token)
	return args.Bool(0), args.Error(1)
}
