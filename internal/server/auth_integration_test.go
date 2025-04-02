package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	authviews "github.com/hail2skins/armory/cmd/web/views/auth"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/middleware"
	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/testutils"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// MockDBWithContext is a mock implementation of the database.Service interface with context
type MockDBWithContext struct {
	mock.Mock
}

func (m *MockDBWithContext) Health() map[string]string {
	args := m.Called()
	return args.Get(0).(map[string]string)
}

func (m *MockDBWithContext) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDBWithContext) CreateUser(ctx context.Context, email, password string) (*database.User, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

func (m *MockDBWithContext) GetUserByEmail(ctx context.Context, email string) (*database.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

func (m *MockDBWithContext) AuthenticateUser(ctx context.Context, email, password string) (*database.User, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

func (m *MockDBWithContext) GetUserByVerificationToken(ctx context.Context, token string) (*database.User, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

func (m *MockDBWithContext) GetUserByRecoveryToken(ctx context.Context, token string) (*database.User, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

func (m *MockDBWithContext) VerifyUserEmail(ctx context.Context, token string) (*database.User, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

func (m *MockDBWithContext) UpdateUser(ctx context.Context, user *database.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockDBWithContext) RequestPasswordReset(ctx context.Context, email string) (*database.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

func (m *MockDBWithContext) ResetPassword(ctx context.Context, token, newPassword string) error {
	args := m.Called(ctx, token, newPassword)
	return args.Error(0)
}

// Payment methods
func (m *MockDBWithContext) CreatePayment(payment *models.Payment) error {
	args := m.Called(payment)
	return args.Error(0)
}

func (m *MockDBWithContext) GetPaymentsByUserID(userID uint) ([]models.Payment, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Payment), args.Error(1)
}

func (m *MockDBWithContext) FindPaymentByID(id uint) (*models.Payment, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Payment), args.Error(1)
}

func (m *MockDBWithContext) UpdatePayment(payment *models.Payment) error {
	args := m.Called(payment)
	return args.Error(0)
}

func (m *MockDBWithContext) GetUserByID(id uint) (*database.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

func (m *MockDBWithContext) GetUserByStripeCustomerID(customerID string) (*database.User, error) {
	args := m.Called(customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// Add admin methods to MockDBWithContext

// FindAllManufacturers retrieves all manufacturers
func (m *MockDBWithContext) FindAllManufacturers() ([]models.Manufacturer, error) {
	args := m.Called()
	return args.Get(0).([]models.Manufacturer), args.Error(1)
}

// FindManufacturerByID retrieves a manufacturer by ID
func (m *MockDBWithContext) FindManufacturerByID(id uint) (*models.Manufacturer, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Manufacturer), args.Error(1)
}

// CreateManufacturer creates a new manufacturer
func (m *MockDBWithContext) CreateManufacturer(manufacturer *models.Manufacturer) error {
	args := m.Called(manufacturer)
	return args.Error(0)
}

// UpdateManufacturer updates a manufacturer
func (m *MockDBWithContext) UpdateManufacturer(manufacturer *models.Manufacturer) error {
	args := m.Called(manufacturer)
	return args.Error(0)
}

// DeleteManufacturer deletes a manufacturer
func (m *MockDBWithContext) DeleteManufacturer(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

// FindAllCalibers retrieves all calibers
func (m *MockDBWithContext) FindAllCalibers() ([]models.Caliber, error) {
	args := m.Called()
	return args.Get(0).([]models.Caliber), args.Error(1)
}

// FindCaliberByID retrieves a caliber by ID
func (m *MockDBWithContext) FindCaliberByID(id uint) (*models.Caliber, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Caliber), args.Error(1)
}

// CreateCaliber creates a new caliber
func (m *MockDBWithContext) CreateCaliber(caliber *models.Caliber) error {
	args := m.Called(caliber)
	return args.Error(0)
}

// UpdateCaliber updates a caliber
func (m *MockDBWithContext) UpdateCaliber(caliber *models.Caliber) error {
	args := m.Called(caliber)
	return args.Error(0)
}

// DeleteCaliber deletes a caliber
func (m *MockDBWithContext) DeleteCaliber(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

// FindAllWeaponTypes retrieves all weapon types
func (m *MockDBWithContext) FindAllWeaponTypes() ([]models.WeaponType, error) {
	args := m.Called()
	return args.Get(0).([]models.WeaponType), args.Error(1)
}

// FindWeaponTypeByID retrieves a weapon type by ID
func (m *MockDBWithContext) FindWeaponTypeByID(id uint) (*models.WeaponType, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.WeaponType), args.Error(1)
}

// CreateWeaponType creates a new weapon type
func (m *MockDBWithContext) CreateWeaponType(weaponType *models.WeaponType) error {
	args := m.Called(weaponType)
	return args.Error(0)
}

// UpdateWeaponType updates a weapon type
func (m *MockDBWithContext) UpdateWeaponType(weaponType *models.WeaponType) error {
	args := m.Called(weaponType)
	return args.Error(0)
}

// DeleteWeaponType deletes a weapon type
func (m *MockDBWithContext) DeleteWeaponType(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

// GetDB returns the underlying *gorm.DB instance
func (m *MockDBWithContext) GetDB() *gorm.DB {
	args := m.Called()
	return args.Get(0).(*gorm.DB)
}

// DeleteGun deletes a gun from the database
func (m *MockDBWithContext) DeleteGun(db *gorm.DB, id uint, ownerID uint) error {
	args := m.Called(db, id, ownerID)
	return args.Error(0)
}

// IsRecoveryExpired implements the database.Service interface
func (m *MockDBWithContext) IsRecoveryExpired(ctx context.Context, token string) (bool, error) {
	args := m.Called(ctx, token)
	return args.Bool(0), args.Error(1)
}

// CountUsers returns the count of users
func (m *MockDBWithContext) CountUsers() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

// GetAllPayments retrieves all payments ordered by creation date descending
func (m *MockDBWithContext) GetAllPayments() ([]models.Payment, error) {
	args := m.Called()
	return args.Get(0).([]models.Payment), args.Error(1)
}

// FindRecentUsers finds recent users with pagination and sorting
func (m *MockDBWithContext) FindRecentUsers(offset, limit int, sortBy, sortOrder string) ([]database.User, error) {
	args := m.Called(offset, limit, sortBy, sortOrder)
	return args.Get(0).([]database.User), args.Error(1)
}

// Setup TestUserRegistration mock responses
func setupTestRouter(t *testing.T) (*gin.Engine, *MockDBWithContext, *mocks.MockEmailService) {
	// Create Gin router
	router := gin.Default()

	// Configure Gin for tests
	gin.SetMode(gin.TestMode)

	// Add session middleware
	store := cookie.NewStore([]byte("test-secret-key"))
	router.Use(sessions.Sessions("auth-session", store))

	// Add CSRF middleware after session middleware
	router.Use(middleware.CSRFMiddleware())

	// Create mock database
	mockDB := new(MockDBWithContext)

	// Create test user
	hashedPassword, err := database.HashPassword("Password123!")
	if err != nil {
		t.Fatal(err)
	}

	// Create a test user
	testUser := &database.User{
		Email:    "test@example.com",
		Password: hashedPassword,
		Verified: true,
	}
	testUser.ID = 1 // Set ID

	// Create a test DB instance to handle Unscoped queries
	testDb := testutils.SharedTestService()

	// Mock the GetDB method to return a real DB instance
	mockDB.On("GetDB").Return(testDb.GetDB())

	// Mock authentication
	mockDB.On("AuthenticateUser", mock.Anything, "test@example.com", "Password123!").Return(testUser, nil)
	mockDB.On("AuthenticateUser", mock.Anything, "test@example.com", "wrongpassword").Return(nil, nil)
	mockDB.On("AuthenticateUser", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

	// Mock get user by email - use this for specific cases
	// First mock it to return nil for first call in register test
	mockDB.On("GetUserByEmail", mock.Anything, "test@example.com").Return(nil, nil).Once()
	mockDB.On("GetUserByEmail", mock.Anything, "nonexistent@example.com").Return(nil, nil)
	mockDB.On("GetUserByEmail", mock.Anything, "duplicate@example.com").Return(testUser, nil)
	mockDB.On("GetUserByEmail", mock.Anything, mock.Anything).Return(nil, nil)

	// Create a mock email service
	mockEmail := new(mocks.MockEmailService)

	// Setup mock responses
	testUser2 := &database.User{
		Email:    "test2@example.com",
		Verified: true,
	}
	testUser2.ID = 2 // Set ID

	// Mock create user
	mockDB.On("CreateUser", mock.Anything, "test@example.com", "Password123!").Return(testUser, nil)
	mockDB.On("CreateUser", mock.Anything, "test2@example.com", "Password123!").Return(testUser2, nil)

	// Mock GetUserByEmail specifically for the first "test@example.com" calls
	mockDB.On("GetUserByEmail", mock.Anything, "test2@example.com").Return(nil, nil).Once()

	// Mock AuthenticateUser for test2@example.com
	mockDB.On("AuthenticateUser", mock.Anything, "test2@example.com", "Password123!").Return(testUser2, nil)

	// Mock UpdateUser to return nil error
	mockDB.On("UpdateUser", mock.Anything, mock.AnythingOfType("*database.User")).Return(nil)

	// Mock verification token methods
	mockDB.On("GetUserByVerificationToken", mock.Anything, "test-token").Return(testUser, nil)
	mockDB.On("VerifyUserEmail", mock.Anything, "test-token").Return(testUser, nil)
	mockDB.On("GetUserByVerificationToken", mock.Anything, "test-verification-token").Return(testUser, nil)
	mockDB.On("VerifyUserEmail", mock.Anything, "test-verification-token").Run(func(args mock.Arguments) {
		// Update the user's verified status
		testUser.Verified = true
	}).Return(testUser, nil)

	// Mock other methods that might be called
	mockDB.On("GetUserByVerificationToken", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil).Maybe()
	mockDB.On("GetUserByRecoveryToken", mock.Anything, mock.Anything).Return(nil, nil)
	mockDB.On("RequestPasswordReset", mock.Anything, mock.Anything).Return(nil, nil)
	mockDB.On("ResetPassword", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Health check
	mockDB.On("Health").Return(map[string]string{"status": "up"})

	// Create a new router
	router.Use(gin.Recovery())

	// Create a new auth controller
	authController := controller.NewAuthController(mockDB)

	// Set the mock email service - this is crucial
	authController.SetEmailService(mockEmail)

	// Set up middleware for auth data
	router.Use(func(c *gin.Context) {
		// Get the current user's authentication status and email
		userInfo, authenticated := authController.GetCurrentUser(c)

		// Create AuthData with authentication status and email
		authData := data.NewAuthData()
		authData.Authenticated = authenticated

		// Set email if authenticated
		if authenticated {
			authData.Email = userInfo.GetUserName()
		}

		// Add authData to context
		c.Set("authData", authData)
		c.Set("authController", authController)

		c.Next()
	})

	// Override the render methods to use our templates
	authController.RenderLogin = func(c *gin.Context, d interface{}) {
		authData := d.(data.AuthData)
		// Set authentication state
		_, authenticated := authController.GetCurrentUser(c)
		authData.Authenticated = authenticated
		// Set default title if not set
		if authData.Title == "" {
			authData.Title = "Login"
		}
		authviews.Login(authData).Render(c.Request.Context(), c.Writer)
	}

	authController.RenderRegister = func(c *gin.Context, d interface{}) {
		authData := d.(data.AuthData)
		// Set authentication state
		_, authenticated := authController.GetCurrentUser(c)
		authData.Authenticated = authenticated
		// Set default title if not set
		if authData.Title == "" {
			authData.Title = "Register"
		}
		authviews.Register(authData).Render(c.Request.Context(), c.Writer)
	}

	authController.RenderLogout = func(c *gin.Context, d interface{}) {
		authData := d.(data.AuthData)
		// Set authentication state - should be false after logout
		authData.Authenticated = false
		// Set default title if not set
		if authData.Title == "" {
			authData.Title = "Logout"
		}
		authviews.Logout(authData).Render(c.Request.Context(), c.Writer)
	}

	// Set up routes
	router.GET("/", func(c *gin.Context) {
		// Get auth data from context, but we don't need to use it here
		// as the HomeController will handle it
		homeController := controller.NewHomeController(mockDB)
		homeController.HomeHandler(c)
	})

	router.GET("/login", authController.LoginHandler)
	router.POST("/login", authController.LoginHandler)
	router.GET("/register", authController.RegisterHandler)
	router.POST("/register", authController.RegisterHandler)
	router.GET("/logout", authController.LogoutHandler)
	router.GET("/verification-sent", func(c *gin.Context) {
		c.String(http.StatusOK, "Verification email sent")
	})
	router.GET("/verify-email", authController.VerifyEmailHandler)

	// Add owner route for redirect after login
	router.GET("/owner", func(c *gin.Context) {
		c.String(http.StatusOK, "Owner page")
	})

	// Set up a mock gorm.DB for direct SQL operations
	mockGormDB := testDb.GetDB()

	// Mock the GetDB method to return a real DB instance
	mockDB.On("GetDB").Return(mockGormDB)

	return router, mockDB, mockEmail
}

// MockAuthService is defined in routes_test.go in the same package

// TestSimplifiedAuthFlow tests the authentication flow using mocks
func TestSimplifiedAuthFlow(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	t.Run("Authentication flow changes login state", func(t *testing.T) {
		// Create a fresh router for this test
		router := gin.New()

		// Create mock DB and controllers
		mockDB := new(mocks.MockDB)
		homeController := controller.NewHomeController(mockDB)

		// Create a test authService that can change state
		authService := &MockAuthService{
			authenticated: false,
			email:         "",
		}

		// Register middleware
		router.Use(func(c *gin.Context) {
			c.Set("auth", authService)
			c.Set("authController", authService)
			c.Set("authData", map[string]interface{}{
				"Authenticated": authService.authenticated,
				"Email":         authService.email,
				"Title":         "Test Page",
			})
			c.Next()
		})

		// Register routes
		router.GET("/login", authService.LoginHandler)
		router.POST("/login", func(c *gin.Context) {
			// Simulate login
			authService.authenticated = true
			authService.email = "test@example.com"
			c.Redirect(http.StatusSeeOther, "/")
		})
		router.GET("/register", authService.RegisterHandler)
		router.POST("/register", func(c *gin.Context) {
			// Simulate registration
			c.Redirect(http.StatusSeeOther, "/verification-sent")
		})
		router.GET("/verification-sent", func(c *gin.Context) {
			c.String(http.StatusOK, "Verification email sent")
		})
		router.GET("/verify-email", func(c *gin.Context) {
			// Simulate verification
			c.Redirect(http.StatusSeeOther, "/login?verified=true")
		})
		router.GET("/logout", func(c *gin.Context) {
			// Simulate logout
			authService.authenticated = false
			authService.email = ""
			c.Redirect(http.StatusSeeOther, "/")
		})
		router.GET("/", homeController.HomeHandler)

		// Test 1: Unauthenticated user
		t.Run("Unauthenticated state", func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/login", nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusOK, resp.Code)
			assert.Contains(t, resp.Body.String(), "Login page")
		})

		// Test 2: User can register
		t.Run("Registration redirects to verification page", func(t *testing.T) {
			form := url.Values{}
			form.Add("email", "test@example.com")
			form.Add("password", "Password123!")
			form.Add("confirm_password", "Password123!")

			req, _ := http.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusSeeOther, resp.Code)
			assert.Equal(t, "/verification-sent", resp.Header().Get("Location"))
		})

		// Test 3: User can verify email
		t.Run("Email verification", func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/verify-email?token=test-token", nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusSeeOther, resp.Code)
			assert.Equal(t, "/login?verified=true", resp.Header().Get("Location"))
		})

		// Test 4: User can login
		t.Run("Login changes authentication state", func(t *testing.T) {
			// First check that we're not authenticated
			assert.False(t, authService.authenticated)

			// Login
			form := url.Values{}
			form.Add("email", "test@example.com")
			form.Add("password", "Password123!")

			req, _ := http.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusSeeOther, resp.Code)
			assert.Equal(t, "/", resp.Header().Get("Location"))

			// Check authentication state was updated
			assert.True(t, authService.authenticated)
			assert.Equal(t, "test@example.com", authService.email)
		})

		// Test 5: User can logout
		t.Run("Logout changes authentication state", func(t *testing.T) {
			// First verify we're authenticated
			assert.True(t, authService.authenticated)

			// Logout
			req, _ := http.NewRequest("GET", "/logout", nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusSeeOther, resp.Code)
			assert.Equal(t, "/", resp.Header().Get("Location"))

			// Check authentication state was updated
			assert.False(t, authService.authenticated)
			assert.Equal(t, "", authService.email)
		})
	})
}

// Promotion-related methods

func (m *MockDBWithContext) FindAllPromotions() ([]models.Promotion, error) {
	args := m.Called()
	return args.Get(0).([]models.Promotion), args.Error(1)
}

func (m *MockDBWithContext) FindPromotionByID(id uint) (*models.Promotion, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Promotion), args.Error(1)
}

func (m *MockDBWithContext) CreatePromotion(promotion *models.Promotion) error {
	args := m.Called(promotion)
	return args.Error(0)
}

func (m *MockDBWithContext) UpdatePromotion(promotion *models.Promotion) error {
	args := m.Called(promotion)
	return args.Error(0)
}

func (m *MockDBWithContext) DeletePromotion(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

// FindActivePromotions mocks the database method to fetch active promotions
func (m *MockDBWithContext) FindActivePromotions() ([]models.Promotion, error) {
	args := m.Called()
	return args.Get(0).([]models.Promotion), args.Error(1)
}

// CountActiveSubscribers returns the number of users with active paid subscriptions
func (m *MockDBWithContext) CountActiveSubscribers() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

// CountNewUsersThisMonth returns the number of users registered in the current month
func (m *MockDBWithContext) CountNewUsersThisMonth() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

// CountNewUsersLastMonth returns the number of users registered in the previous month
func (m *MockDBWithContext) CountNewUsersLastMonth() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

// CountNewSubscribersThisMonth returns the number of new subscriptions in the current month
func (m *MockDBWithContext) CountNewSubscribersThisMonth() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

// CountNewSubscribersLastMonth returns the number of new subscriptions in the previous month
func (m *MockDBWithContext) CountNewSubscribersLastMonth() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

// FindAllGuns retrieves all guns
func (m *MockDBWithContext) FindAllGuns() ([]models.Gun, error) {
	args := m.Called()
	return args.Get(0).([]models.Gun), args.Error(1)
}

// FindAllUsers retrieves all users
func (m *MockDBWithContext) FindAllUsers() ([]database.User, error) {
	args := m.Called()
	return args.Get(0).([]database.User), args.Error(1)
}

// CountGunsByUser counts the number of guns owned by a user
func (m *MockDBWithContext) CountGunsByUser(userID uint) (int64, error) {
	args := m.Called(userID)
	return args.Get(0).(int64), args.Error(1)
}

// FindAllCalibersByIDs retrieves all calibers with the given IDs
func (m *MockDBWithContext) FindAllCalibersByIDs(ids []uint) ([]models.Caliber, error) {
	args := m.Called(ids)
	return args.Get(0).([]models.Caliber), args.Error(1)
}

// FindAllWeaponTypesByIDs retrieves all weapon types with the given IDs
func (m *MockDBWithContext) FindAllWeaponTypesByIDs(ids []uint) ([]models.WeaponType, error) {
	args := m.Called(ids)
	return args.Get(0).([]models.WeaponType), args.Error(1)
}

// LoginTestSuite is a test suite for login functionality
type LoginTestSuite struct {
	testutils.IntegrationTestSuite
}

// TestLoginLogout tests the complete login and logout flow
func (s *LoginTestSuite) TestLoginLogout() {
	// Create a test user
	user, err := s.DB.CreateUser(context.Background(), "test@example.com", "Password123!")
	s.NoError(err)
	s.NotNil(user)

	// Mark user as verified
	user.Verified = true
	err = s.DB.UpdateUser(context.Background(), user)
	s.NoError(err)

	// Test 1: Successful login
	loginForm := url.Values{}
	loginForm.Add("email", "test@example.com")
	loginForm.Add("password", "Password123!")

	resp, err := http.Post(s.Server.URL+"/login", "application/x-www-form-urlencoded", strings.NewReader(loginForm.Encode()))
	s.NoError(err)
	s.Equal(http.StatusSeeOther, resp.StatusCode)
	s.Equal("/owner", resp.Header.Get("Location"))

	// Save cookies from login
	s.SaveCookies(resp)

	// Test 2: Access owner page (should succeed)
	req, _ := http.NewRequest("GET", s.Server.URL+"/owner", nil)
	s.ApplyCookies(req)
	resp, err = http.DefaultClient.Do(req)
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)

	// Test 3: Logout
	req, _ = http.NewRequest("GET", s.Server.URL+"/logout", nil)
	s.ApplyCookies(req)
	resp, err = http.DefaultClient.Do(req)
	s.NoError(err)
	s.Equal(http.StatusSeeOther, resp.StatusCode)
	s.Equal("/", resp.Header.Get("Location"))

	// Test 4: Try to access owner page after logout (should fail)
	req, _ = http.NewRequest("GET", s.Server.URL+"/owner", nil)
	s.ApplyCookies(req)
	resp, err = http.DefaultClient.Do(req)
	s.NoError(err)
	s.Equal(http.StatusFound, resp.StatusCode) // Should redirect to login
}

// TestLoginFailures tests various login failure scenarios
func (s *LoginTestSuite) TestLoginFailures() {
	// Create a test user
	user, err := s.DB.CreateUser(context.Background(), "test@example.com", "Password123!")
	s.NoError(err)
	s.NotNil(user)

	// Test 1: Wrong password
	loginForm := url.Values{}
	loginForm.Add("email", "test@example.com")
	loginForm.Add("password", "wrongpassword")

	resp, err := http.Post(s.Server.URL+"/login", "application/x-www-form-urlencoded", strings.NewReader(loginForm.Encode()))
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode) // Shows login form with error

	// Test 2: Non-existent user
	loginForm = url.Values{}
	loginForm.Add("email", "nonexistent@example.com")
	loginForm.Add("password", "Password123!")

	resp, err = http.Post(s.Server.URL+"/login", "application/x-www-form-urlencoded", strings.NewReader(loginForm.Encode()))
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode) // Shows login form with error

	// Test 3: Unverified user
	user2, err := s.DB.CreateUser(context.Background(), "unverified@example.com", "Password123!")
	s.NoError(err)
	s.NotNil(user2)

	loginForm = url.Values{}
	loginForm.Add("email", "unverified@example.com")
	loginForm.Add("password", "Password123!")

	resp, err = http.Post(s.Server.URL+"/login", "application/x-www-form-urlencoded", strings.NewReader(loginForm.Encode()))
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode) // Shows login form with error
}

func TestLogin(t *testing.T) {
	suite := &LoginTestSuite{}
	testutils.RunIntegrationSuite(t, &suite.IntegrationTestSuite)
}

// Feature Flag-related methods for MockDBWithContext

func (m *MockDBWithContext) FindAllFeatureFlags() ([]models.FeatureFlag, error) {
	args := m.Called()
	return args.Get(0).([]models.FeatureFlag), args.Error(1)
}

func (m *MockDBWithContext) FindFeatureFlagByID(id uint) (*models.FeatureFlag, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.FeatureFlag), args.Error(1)
}

func (m *MockDBWithContext) FindFeatureFlagByName(name string) (*models.FeatureFlag, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.FeatureFlag), args.Error(1)
}

func (m *MockDBWithContext) CreateFeatureFlag(flag *models.FeatureFlag) error {
	return m.Called(flag).Error(0)
}

func (m *MockDBWithContext) UpdateFeatureFlag(flag *models.FeatureFlag) error {
	return m.Called(flag).Error(0)
}

func (m *MockDBWithContext) DeleteFeatureFlag(id uint) error {
	return m.Called(id).Error(0)
}

func (m *MockDBWithContext) AddRoleToFeatureFlag(flagID uint, role string) error {
	return m.Called(flagID, role).Error(0)
}

func (m *MockDBWithContext) RemoveRoleFromFeatureFlag(flagID uint, role string) error {
	return m.Called(flagID, role).Error(0)
}

func (m *MockDBWithContext) IsFeatureEnabled(name string) (bool, error) {
	args := m.Called(name)
	return args.Bool(0), args.Error(1)
}

func (m *MockDBWithContext) CanUserAccessFeature(username, featureName string) (bool, error) {
	args := m.Called(username, featureName)
	return args.Bool(0), args.Error(1)
}

// Casing-related methods for MockDBWithContext

func (m *MockDBWithContext) FindAllCasings() ([]models.Casing, error) {
	args := m.Called()
	return args.Get(0).([]models.Casing), args.Error(1)
}

func (m *MockDBWithContext) FindCasingByID(id uint) (*models.Casing, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Casing), args.Error(1)
}

func (m *MockDBWithContext) CreateCasing(casing *models.Casing) error {
	return m.Called(casing).Error(0)
}

func (m *MockDBWithContext) UpdateCasing(casing *models.Casing) error {
	return m.Called(casing).Error(0)
}

func (m *MockDBWithContext) DeleteCasing(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

// BulletStyle-related methods
func (m *MockDBWithContext) FindAllBulletStyles() ([]models.BulletStyle, error) {
	args := m.Called()
	return args.Get(0).([]models.BulletStyle), args.Error(1)
}

func (m *MockDBWithContext) FindBulletStyleByID(id uint) (*models.BulletStyle, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BulletStyle), args.Error(1)
}

func (m *MockDBWithContext) CreateBulletStyle(bulletStyle *models.BulletStyle) error {
	args := m.Called(bulletStyle)
	return args.Error(0)
}

func (m *MockDBWithContext) UpdateBulletStyle(bulletStyle *models.BulletStyle) error {
	args := m.Called(bulletStyle)
	return args.Error(0)
}

func (m *MockDBWithContext) DeleteBulletStyle(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

// Grain-related methods
func (m *MockDBWithContext) FindAllGrains() ([]models.Grain, error) {
	args := m.Called()
	return args.Get(0).([]models.Grain), args.Error(1)
}

func (m *MockDBWithContext) FindGrainByID(id uint) (*models.Grain, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Grain), args.Error(1)
}

func (m *MockDBWithContext) CreateGrain(grain *models.Grain) error {
	args := m.Called(grain)
	return args.Error(0)
}

func (m *MockDBWithContext) UpdateGrain(grain *models.Grain) error {
	args := m.Called(grain)
	return args.Error(0)
}

func (m *MockDBWithContext) DeleteGrain(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

// Brand related methods
func (m *MockDBWithContext) FindAllBrands() ([]models.Brand, error) {
	args := m.Called()
	return args.Get(0).([]models.Brand), args.Error(1)
}

func (m *MockDBWithContext) CreateBrand(brand *models.Brand) error {
	args := m.Called(brand)
	return args.Error(0)
}

func (m *MockDBWithContext) FindBrandByID(id uint) (*models.Brand, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Brand), args.Error(1)
}

func (m *MockDBWithContext) UpdateBrand(brand *models.Brand) error {
	args := m.Called(brand)
	return args.Error(0)
}

func (m *MockDBWithContext) DeleteBrand(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

// CheckExpiredPromotionSubscription checks if a user's subscription has expired and updates the status to "expired" if it has.
func (m *MockDBWithContext) CheckExpiredPromotionSubscription(user *database.User) (bool, error) {
	args := m.Called(user)
	return args.Bool(0), args.Error(1)
}

// Ammunition-related methods for MockDBWithContext
func (m *MockDBWithContext) CountAmmoByUser(userID uint) (int64, error) {
	args := m.Called(userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockDBWithContext) FindAllAmmo() ([]models.Ammo, error) {
	args := m.Called()
	return args.Get(0).([]models.Ammo), args.Error(1)
}

func (m *MockDBWithContext) FindAmmoByID(id uint) (*models.Ammo, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Ammo), args.Error(1)
}
