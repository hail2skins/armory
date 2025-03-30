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

func (m *MockDB) GetAllPayments() ([]models.Payment, error) {
	args := m.Called()
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

// FindAllPromotions mocks the database method to fetch all promotions
func (m *MockDB) FindAllPromotions() ([]models.Promotion, error) {
	args := m.Called()
	return args.Get(0).([]models.Promotion), args.Error(1)
}

// FindPromotionByID mocks the database method to find a promotion by ID
func (m *MockDB) FindPromotionByID(id uint) (*models.Promotion, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Promotion), args.Error(1)
}

// CreatePromotion mocks the database method to create a promotion
func (m *MockDB) CreatePromotion(promotion *models.Promotion) error {
	return m.Called(promotion).Error(0)
}

// UpdatePromotion mocks the database method to update a promotion
func (m *MockDB) UpdatePromotion(promotion *models.Promotion) error {
	return m.Called(promotion).Error(0)
}

// DeletePromotion mocks the database method to delete a promotion
func (m *MockDB) DeletePromotion(id uint) error {
	return m.Called(id).Error(0)
}

// FindActivePromotions mocks the database method to fetch active promotions
func (m *MockDB) FindActivePromotions() ([]models.Promotion, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Promotion), args.Error(1)
}

// CountActiveSubscribers returns the number of users with active paid subscriptions
func (m *MockDB) CountActiveSubscribers() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

// CountNewUsersThisMonth returns the number of users registered in the current month
func (m *MockDB) CountNewUsersThisMonth() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

// CountNewUsersLastMonth returns the number of users registered in the previous month
func (m *MockDB) CountNewUsersLastMonth() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

// CountNewSubscribersThisMonth returns the number of new subscriptions in the current month
func (m *MockDB) CountNewSubscribersThisMonth() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

// CountNewSubscribersLastMonth returns the number of new subscriptions in the previous month
func (m *MockDB) CountNewSubscribersLastMonth() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockDB) FindAllGuns() ([]models.Gun, error) {
	args := m.Called()
	return args.Get(0).([]models.Gun), args.Error(1)
}

func (m *MockDB) FindAllUsers() ([]database.User, error) {
	args := m.Called()
	return args.Get(0).([]database.User), args.Error(1)
}

func (m *MockDB) CountGunsByUser(userID uint) (int64, error) {
	args := m.Called(userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockDB) FindAllCalibersByIDs(ids []uint) ([]models.Caliber, error) {
	args := m.Called(ids)
	return args.Get(0).([]models.Caliber), args.Error(1)
}

func (m *MockDB) FindAllWeaponTypesByIDs(ids []uint) ([]models.WeaponType, error) {
	args := m.Called(ids)
	return args.Get(0).([]models.WeaponType), args.Error(1)
}

// Feature Flag-related methods for MockDB

func (m *MockDB) FindAllFeatureFlags() ([]models.FeatureFlag, error) {
	args := m.Called()
	return args.Get(0).([]models.FeatureFlag), args.Error(1)
}

func (m *MockDB) FindFeatureFlagByID(id uint) (*models.FeatureFlag, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.FeatureFlag), args.Error(1)
}

func (m *MockDB) FindFeatureFlagByName(name string) (*models.FeatureFlag, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.FeatureFlag), args.Error(1)
}

func (m *MockDB) CreateFeatureFlag(flag *models.FeatureFlag) error {
	return m.Called(flag).Error(0)
}

func (m *MockDB) UpdateFeatureFlag(flag *models.FeatureFlag) error {
	return m.Called(flag).Error(0)
}

func (m *MockDB) DeleteFeatureFlag(id uint) error {
	return m.Called(id).Error(0)
}

func (m *MockDB) AddRoleToFeatureFlag(flagID uint, role string) error {
	return m.Called(flagID, role).Error(0)
}

func (m *MockDB) RemoveRoleFromFeatureFlag(flagID uint, role string) error {
	return m.Called(flagID, role).Error(0)
}

func (m *MockDB) IsFeatureEnabled(name string) (bool, error) {
	args := m.Called(name)
	return args.Bool(0), args.Error(1)
}

func (m *MockDB) CanUserAccessFeature(username, featureName string) (bool, error) {
	args := m.Called(username, featureName)
	return args.Bool(0), args.Error(1)
}

// --- Casing-related methods ---

func (m *MockDB) FindAllCasings() ([]models.Casing, error) {
	args := m.Called()
	return args.Get(0).([]models.Casing), args.Error(1)
}

func (m *MockDB) CreateCasing(casing *models.Casing) error {
	return m.Called(casing).Error(0)
}

func (m *MockDB) FindCasingByID(id uint) (*models.Casing, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Casing), args.Error(1)
}

func (m *MockDB) UpdateCasing(casing *models.Casing) error {
	return m.Called(casing).Error(0)
}

func (m *MockDB) DeleteCasing(id uint) error {
	return m.Called(id).Error(0)
}

// --- BulletStyle-related methods ---

func (m *MockDB) FindAllBulletStyles() ([]models.BulletStyle, error) {
	args := m.Called()
	return args.Get(0).([]models.BulletStyle), args.Error(1)
}

func (m *MockDB) CreateBulletStyle(bulletStyle *models.BulletStyle) error {
	return m.Called(bulletStyle).Error(0)
}

func (m *MockDB) FindBulletStyleByID(id uint) (*models.BulletStyle, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BulletStyle), args.Error(1)
}

func (m *MockDB) UpdateBulletStyle(bulletStyle *models.BulletStyle) error {
	return m.Called(bulletStyle).Error(0)
}

func (m *MockDB) DeleteBulletStyle(id uint) error {
	return m.Called(id).Error(0)
}

// Grain-related mock methods
func (m *MockDB) FindAllGrains() ([]models.Grain, error) {
	args := m.Called()
	return args.Get(0).([]models.Grain), args.Error(1)
}

func (m *MockDB) CreateGrain(grain *models.Grain) error {
	return m.Called(grain).Error(0)
}

func (m *MockDB) FindGrainByID(id uint) (*models.Grain, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Grain), args.Error(1)
}

func (m *MockDB) UpdateGrain(grain *models.Grain) error {
	return m.Called(grain).Error(0)
}

func (m *MockDB) DeleteGrain(id uint) error {
	return m.Called(id).Error(0)
}

// Brand related methods
// FindAllBrands mocks the FindAllBrands method
func (m *MockDB) FindAllBrands() ([]models.Brand, error) {
	args := m.Called()
	return args.Get(0).([]models.Brand), args.Error(1)
}

// CreateBrand mocks the CreateBrand method
func (m *MockDB) CreateBrand(brand *models.Brand) error {
	return m.Called(brand).Error(0)
}

// FindBrandByID mocks the FindBrandByID method
func (m *MockDB) FindBrandByID(id uint) (*models.Brand, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Brand), args.Error(1)
}

// UpdateBrand mocks the UpdateBrand method
func (m *MockDB) UpdateBrand(brand *models.Brand) error {
	return m.Called(brand).Error(0)
}

// DeleteBrand mocks the DeleteBrand method
func (m *MockDB) DeleteBrand(id uint) error {
	return m.Called(id).Error(0)
}
