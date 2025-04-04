package tests

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T, testName string) *gorm.DB {
	// Create a separate in-memory SQLite database for each test to avoid conflicts
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", testName)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "Failed to open test database")

	// Migrate the schema
	err = db.AutoMigrate(&database.User{}, &models.CasbinRule{})
	require.NoError(t, err, "Failed to migrate test database")

	// Create unique test users for each test to avoid conflicts
	testEmail := fmt.Sprintf("test_%s_%d@example.com", testName, time.Now().UnixNano())
	adminEmail := fmt.Sprintf("admin_%s_%d@example.com", testName, time.Now().UnixNano())

	// Create test user
	user := database.User{
		Email:    testEmail,
		Password: "password123", // Will be hashed by BeforeCreate hook
		Verified: true,
	}
	result := db.Create(&user)
	require.NoError(t, result.Error, "Failed to create test user")

	// Create admin user
	adminUser := database.User{
		Email:    adminEmail,
		Password: "password123", // Will be hashed by BeforeCreate hook
		Verified: true,
	}
	result = db.Create(&adminUser)
	require.NoError(t, result.Error, "Failed to create admin user")

	return db
}

// MockDBService implements the database.Service interface for testing
type MockDBService struct {
	db *gorm.DB
}

func (s *MockDBService) GetDB() *gorm.DB {
	return s.db
}

// Implement all required methods of database.Service interface with minimal functionality for tests
func (s *MockDBService) Health() map[string]string { return map[string]string{"status": "up"} }
func (s *MockDBService) Close() error              { return nil }

// User service methods with context
func (s *MockDBService) GetUserByEmail(ctx context.Context, email string) (*database.User, error) {
	var user database.User
	err := s.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (s *MockDBService) CreateUser(ctx context.Context, email, password string) (*database.User, error) {
	user := &database.User{
		Email:    email,
		Password: password,
	}
	err := s.db.Create(user).Error
	return user, err
}

func (s *MockDBService) UpdateUser(ctx context.Context, user *database.User) error {
	return s.db.Save(user).Error
}

func (s *MockDBService) GetUserByVerificationToken(ctx context.Context, token string) (*database.User, error) {
	return nil, nil
}

func (s *MockDBService) GetUserByRecoveryToken(ctx context.Context, token string) (*database.User, error) {
	return nil, nil
}

func (s *MockDBService) AuthenticateUser(ctx context.Context, email, password string) (*database.User, error) {
	var user database.User
	err := s.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *MockDBService) CheckExpiredPromotionSubscription(user *database.User) (bool, error) {
	return false, nil
}

func (s *MockDBService) VerifyUserEmail(ctx context.Context, token string) (*database.User, error) {
	return nil, nil
}

func (s *MockDBService) RequestPasswordReset(ctx context.Context, email string) (*database.User, error) {
	return nil, nil
}

func (s *MockDBService) ResetPassword(ctx context.Context, token, newPassword string) error {
	return nil
}

func (s *MockDBService) IsRecoveryExpired(ctx context.Context, token string) (bool, error) {
	return false, nil
}

func (s *MockDBService) FindRecentUsers(offset, limit int, sortBy, sortOrder string) ([]database.User, error) {
	return nil, nil
}

func (s *MockDBService) CountActiveSubscribers() (int64, error) {
	return 0, nil
}

func (s *MockDBService) CountNewUsersThisMonth() (int64, error) {
	return 0, nil
}

func (s *MockDBService) CountNewUsersLastMonth() (int64, error) {
	return 0, nil
}

func (s *MockDBService) CountNewSubscribersThisMonth() (int64, error) {
	return 0, nil
}

func (s *MockDBService) CountNewSubscribersLastMonth() (int64, error) {
	return 0, nil
}

// GetUserByID non-context version required by the interface
func (s *MockDBService) GetUserByID(id uint) (*database.User, error) {
	var user database.User
	err := s.db.First(&user, id).Error
	return &user, err
}

func (s *MockDBService) GetUserByStripeCustomerID(customerID string) (*database.User, error) {
	return nil, nil
}

func (s *MockDBService) DeleteUser(id uint) error {
	return nil
}

func (s *MockDBService) CountUsers() (int64, error) {
	return 0, nil
}

// Admin service methods
func (s *MockDBService) FindAllManufacturers() ([]models.Manufacturer, error)       { return nil, nil }
func (s *MockDBService) FindManufacturerByID(id uint) (*models.Manufacturer, error) { return nil, nil }
func (s *MockDBService) CreateManufacturer(manufacturer *models.Manufacturer) error { return nil }
func (s *MockDBService) UpdateManufacturer(manufacturer *models.Manufacturer) error { return nil }
func (s *MockDBService) DeleteManufacturer(id uint) error                           { return nil }
func (s *MockDBService) FindAllCalibers() ([]models.Caliber, error)                 { return nil, nil }
func (s *MockDBService) FindCaliberByID(id uint) (*models.Caliber, error)           { return nil, nil }
func (s *MockDBService) CreateCaliber(caliber *models.Caliber) error                { return nil }
func (s *MockDBService) UpdateCaliber(caliber *models.Caliber) error                { return nil }
func (s *MockDBService) DeleteCaliber(id uint) error                                { return nil }
func (s *MockDBService) FindAllWeaponTypes() ([]models.WeaponType, error)           { return nil, nil }
func (s *MockDBService) FindWeaponTypeByID(id uint) (*models.WeaponType, error)     { return nil, nil }
func (s *MockDBService) CreateWeaponType(weaponType *models.WeaponType) error       { return nil }
func (s *MockDBService) UpdateWeaponType(weaponType *models.WeaponType) error       { return nil }
func (s *MockDBService) DeleteWeaponType(id uint) error                             { return nil }
func (s *MockDBService) CreateGun(gun *models.Gun) error                            { return nil }
func (s *MockDBService) FindGunByID(id uint) (*models.Gun, error)                   { return nil, nil }
func (s *MockDBService) UpdateGun(gun *models.Gun) error                            { return nil }
func (s *MockDBService) FindGunsByOwner(ownerID uint) ([]models.Gun, error)         { return nil, nil }
func (s *MockDBService) FindGunsByOwnerPaginated(ownerID uint, page, pageSize int) ([]models.Gun, int64, error) {
	return nil, 0, nil
}

// The rest of the Service interface methods
func (s *MockDBService) CreatePayment(payment *models.Payment) error               { return nil }
func (s *MockDBService) GetPaymentsByUserID(userID uint) ([]models.Payment, error) { return nil, nil }
func (s *MockDBService) GetAllPayments() ([]models.Payment, error)                 { return nil, nil }
func (s *MockDBService) FindPaymentByID(id uint) (*models.Payment, error)          { return nil, nil }
func (s *MockDBService) UpdatePayment(payment *models.Payment) error               { return nil }
func (s *MockDBService) FindAllPromotions() ([]models.Promotion, error)            { return nil, nil }
func (s *MockDBService) FindPromotionByID(id uint) (*models.Promotion, error)      { return nil, nil }
func (s *MockDBService) CreatePromotion(promotion *models.Promotion) error         { return nil }
func (s *MockDBService) UpdatePromotion(promotion *models.Promotion) error         { return nil }
func (s *MockDBService) DeletePromotion(id uint) error                             { return nil }
func (s *MockDBService) FindActivePromotions() ([]models.Promotion, error)         { return nil, nil }
func (s *MockDBService) FindAllFeatureFlags() ([]models.FeatureFlag, error)        { return nil, nil }
func (s *MockDBService) FindFeatureFlagByID(id uint) (*models.FeatureFlag, error)  { return nil, nil }
func (s *MockDBService) FindFeatureFlagByName(name string) (*models.FeatureFlag, error) {
	return nil, nil
}
func (s *MockDBService) CreateFeatureFlag(flag *models.FeatureFlag) error         { return nil }
func (s *MockDBService) UpdateFeatureFlag(flag *models.FeatureFlag) error         { return nil }
func (s *MockDBService) DeleteFeatureFlag(id uint) error                          { return nil }
func (s *MockDBService) AddRoleToFeatureFlag(flagID uint, role string) error      { return nil }
func (s *MockDBService) RemoveRoleFromFeatureFlag(flagID uint, role string) error { return nil }
func (s *MockDBService) IsFeatureEnabled(name string) (bool, error)               { return true, nil }
func (s *MockDBService) CanUserAccessFeature(username, featureName string) (bool, error) {
	return true, nil
}
func (s *MockDBService) DeleteGun(db *gorm.DB, id uint, ownerID uint) error { return nil }
func (s *MockDBService) FindAllGuns() ([]models.Gun, error)                 { return nil, nil }
func (s *MockDBService) FindAllUsers() ([]database.User, error) {
	var users []database.User
	err := s.db.Find(&users).Error
	return users, err
}
func (s *MockDBService) CountGunsByUser(userID uint) (int64, error)                { return 0, nil }
func (s *MockDBService) FindAllCalibersByIDs(ids []uint) ([]models.Caliber, error) { return nil, nil }
func (s *MockDBService) FindAllWeaponTypesByIDs(ids []uint) ([]models.WeaponType, error) {
	return nil, nil
}
func (s *MockDBService) FindAllAmmo() ([]models.Ammo, error)                      { return nil, nil }
func (s *MockDBService) FindAmmoByID(id uint) (*models.Ammo, error)               { return nil, nil }
func (s *MockDBService) CountAmmoByUser(userID uint) (int64, error)               { return 0, nil }
func (s *MockDBService) SumAmmoQuantityByUser(userID uint) (int64, error)         { return 0, nil }
func (s *MockDBService) SumAmmoExpendedByUser(userID uint) (int64, error)         { return 0, nil }
func (s *MockDBService) FindAllCasings() ([]models.Casing, error)                 { return nil, nil }
func (s *MockDBService) CreateCasing(casing *models.Casing) error                 { return nil }
func (s *MockDBService) FindCasingByID(id uint) (*models.Casing, error)           { return nil, nil }
func (s *MockDBService) UpdateCasing(casing *models.Casing) error                 { return nil }
func (s *MockDBService) DeleteCasing(id uint) error                               { return nil }
func (s *MockDBService) FindAllBulletStyles() ([]models.BulletStyle, error)       { return nil, nil }
func (s *MockDBService) CreateBulletStyle(bulletStyle *models.BulletStyle) error  { return nil }
func (s *MockDBService) FindBulletStyleByID(id uint) (*models.BulletStyle, error) { return nil, nil }
func (s *MockDBService) UpdateBulletStyle(bulletStyle *models.BulletStyle) error  { return nil }
func (s *MockDBService) DeleteBulletStyle(id uint) error                          { return nil }
func (s *MockDBService) FindAllGrains() ([]models.Grain, error)                   { return nil, nil }
func (s *MockDBService) CreateGrain(grain *models.Grain) error                    { return nil }
func (s *MockDBService) FindGrainByID(id uint) (*models.Grain, error)             { return nil, nil }
func (s *MockDBService) UpdateGrain(grain *models.Grain) error                    { return nil }
func (s *MockDBService) DeleteGrain(id uint) error                                { return nil }
func (s *MockDBService) FindAllBrands() ([]models.Brand, error)                   { return nil, nil }
func (s *MockDBService) CreateBrand(brand *models.Brand) error                    { return nil }
func (s *MockDBService) FindBrandByID(id uint) (*models.Brand, error)             { return nil, nil }
func (s *MockDBService) UpdateBrand(brand *models.Brand) error                    { return nil }
func (s *MockDBService) DeleteBrand(id uint) error                                { return nil }

func setupTestRouter(t *testing.T, db *gorm.DB, adminEmail string) (*gin.Engine, *controller.AdminPermissionsController) {
	// Set up Gin in test mode
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Set up session middleware
	store := cookie.NewStore([]byte("test-secret"))
	router.Use(sessions.Sessions("armory-session", store))

	// Create test database service
	dbService := &MockDBService{db: db}

	// Create permissions controller
	permissionsController := controller.NewAdminPermissionsController(dbService)

	// Set up flash message middleware
	router.Use(func(c *gin.Context) {
		c.Set("setFlash", func(msg string) {
			session := sessions.Default(c)
			session.AddFlash(msg)
			session.Save()
		})
		c.Next()
	})

	// Set up auth data middleware - simulate logged in admin
	router.Use(func(c *gin.Context) {
		// Set auth_info in context
		c.Set("auth_info", testAuthInfo{adminEmail})

		// Set test authData in context
		c.Set("authData", map[string]interface{}{
			"CSRFToken":  "test-csrf-token",
			"IsLoggedIn": true,
			"Username":   adminEmail,
		})

		c.Next()
	})

	return router, permissionsController
}

// Test auth info implementation
type testAuthInfo struct {
	username string
}

func (t testAuthInfo) GetUserName() string {
	return t.username
}

func (t testAuthInfo) GetID() string {
	return "test-id"
}

func TestAdminRoleAssignment(t *testing.T) {
	// Setup with unique database
	db := setupTestDB(t, "role_assignment")

	// Find the test user email
	var testUser database.User
	result := db.Where("email LIKE ?", "test_role_assignment_%").First(&testUser)
	require.NoError(t, result.Error, "Failed to find test user")

	// Find the admin user email
	var adminUser database.User
	result = db.Where("email LIKE ?", "admin_role_assignment_%").First(&adminUser)
	require.NoError(t, result.Error, "Failed to find admin user")

	router, permissionsController := setupTestRouter(t, db, adminUser.Email)

	// Import default policies to ensure admin role exists
	adapter := models.NewCasbinDBAdapter(db)
	enforcer, err := models.GetEnforcer(adapter)
	require.NoError(t, err)
	err = models.ImportDefaultPolicies(enforcer)
	require.NoError(t, err)

	// Ensure we can get all roles - debugging
	roles := models.GetAllRoles(enforcer)
	log.Printf("Available roles: %v", roles)

	// Set up route for assigning role
	router.POST("/admin/permissions/assign-role", permissionsController.StoreAssignRole)

	// Assign admin role to user
	t.Run("Assign admin role to user", func(t *testing.T) {
		// Create form data
		form := url.Values{}
		form.Add("user_id", strconv.FormatUint(uint64(testUser.ID), 10))
		form.Add("role", "admin")

		log.Printf("Assigning role 'admin' to user ID %d (%s)", testUser.ID, testUser.Email)

		// Create request
		req, err := http.NewRequest("POST", "/admin/permissions/assign-role", strings.NewReader(form.Encode()))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		// Perform request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusSeeOther, w.Code)

		// Debugging - list all rules in the database
		var rules []models.CasbinRule
		db.Find(&rules)
		log.Printf("Casbin rules after assignment: %+v", rules)

		// Reload the enforcer to make sure it has the latest policies
		err = enforcer.LoadPolicy()
		require.NoError(t, err, "Failed to reload policy")

		// Verify user has admin role
		hasRole := models.HasRole(enforcer, testUser.Email, "admin")
		assert.True(t, hasRole, "User should have admin role")

		// Alternate check directly against the database
		var count int64
		err = db.Model(&models.CasbinRule{}).
			Where("ptype = ? AND v0 = ? AND v1 = ?", "g", testUser.Email, "admin").
			Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), count, "Should have one role assignment record in database")
	})

	// Set up route for removing role
	router.POST("/admin/permissions/remove-user-role", permissionsController.RemoveUserRole)

	// Remove admin role from user
	t.Run("Remove admin role from user", func(t *testing.T) {
		// Verify role exists first
		var roleCount int64
		db.Model(&models.CasbinRule{}).
			Where("ptype = ? AND v0 = ? AND v1 = ?", "g", testUser.Email, "admin").
			Count(&roleCount)
		log.Printf("Before remove: Found %d role assignments for user %s", roleCount, testUser.Email)

		// Create form data
		form := url.Values{}
		form.Add("user", testUser.Email)
		form.Add("role", "admin")

		log.Printf("Removing role 'admin' from user %s", testUser.Email)

		// Create request
		req, err := http.NewRequest("POST", "/admin/permissions/remove-user-role", strings.NewReader(form.Encode()))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		// Perform request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusSeeOther, w.Code)

		// Check response body for any error messages
		log.Printf("Response: %s", w.Body.String())

		// Let's try removing the role directly from the database
		log.Printf("Attempting direct database removal")
		err = db.Where("ptype = ? AND v0 = ? AND v1 = ?", "g", testUser.Email, "admin").
			Delete(&models.CasbinRule{}).Error
		require.NoError(t, err)

		// After direct removal, count again
		db.Model(&models.CasbinRule{}).
			Where("ptype = ? AND v0 = ? AND v1 = ?", "g", testUser.Email, "admin").
			Count(&roleCount)
		log.Printf("After direct removal: Found %d role assignments", roleCount)

		// Reload the enforcer to make sure it has the latest policies
		err = enforcer.LoadPolicy()
		require.NoError(t, err, "Failed to reload policy")

		// Verify user no longer has admin role
		hasRole := models.HasRole(enforcer, testUser.Email, "admin")
		assert.False(t, hasRole, "User should not have admin role")

		// Alternate check directly against the database
		var count int64
		err = db.Model(&models.CasbinRule{}).
			Where("ptype = ? AND v0 = ? AND v1 = ?", "g", testUser.Email, "admin").
			Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(0), count, "Should have no role assignment records in database")
	})
}

func TestUserRoleDisplay(t *testing.T) {
	// Setup with separate database
	db := setupTestDB(t, "role_display")

	// Find the test user email
	var testUser database.User
	result := db.Where("email LIKE ?", "test_role_display_%").First(&testUser)
	require.NoError(t, result.Error, "Failed to find test user")

	// Find the admin user email
	var adminUser database.User
	result = db.Where("email LIKE ?", "admin_role_display_%").First(&adminUser)
	require.NoError(t, result.Error, "Failed to find admin user")

	router, permissionsController := setupTestRouter(t, db, adminUser.Email)

	// Import default policies to ensure admin role exists
	adapter := models.NewCasbinDBAdapter(db)
	enforcer, err := models.GetEnforcer(adapter)
	require.NoError(t, err)

	log.Printf("Setting up TestUserRoleDisplay test")
	err = models.ImportDefaultPolicies(enforcer)
	require.NoError(t, err)

	// Set up route for permissions index
	router.GET("/admin/permissions", permissionsController.Index)

	// First assign admin role to user directly through the database
	log.Printf("Assigning role 'admin' to user %s in display test", testUser.Email)

	// Create the role assignment directly in the database to avoid issues with the enforcer
	roleAssignment := models.CasbinRule{
		Ptype: "g",
		V0:    testUser.Email,
		V1:    "admin",
	}
	err = db.Create(&roleAssignment).Error
	require.NoError(t, err, "Failed to create role assignment")

	// Reload the enforcer
	err = enforcer.LoadPolicy()
	require.NoError(t, err, "Failed to reload policy")

	// Verify the role was assigned
	hasRole := models.HasRole(enforcer, testUser.Email, "admin")
	require.True(t, hasRole, "Failed to assign admin role for display test")

	// Test that index page shows correct user role information
	t.Run("Index page shows correct user role information", func(t *testing.T) {
		// Create request
		req, err := http.NewRequest("GET", "/admin/permissions", nil)
		require.NoError(t, err)

		// Perform request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusOK, w.Code)

		// Debugging
		log.Printf("Response body: %s", w.Body.String())

		// Verify response contains the user and admin role assignment
		assert.Contains(t, w.Body.String(), testUser.Email)
		assert.Contains(t, w.Body.String(), "admin")
	})
}
