package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	authviews "github.com/hail2skins/armory/cmd/web/views/auth"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// MockEmailService is a mock implementation of the EmailService interface for testing
type MockEmailService struct {
	mock.Mock

	// Track calls for verification
	SendVerificationEmailCalled bool
	SendVerificationEmailEmail  string
	SendVerificationEmailToken  string
	SendVerificationEmailError  error

	SendPasswordResetEmailCalled bool
	SendPasswordResetEmailEmail  string
	SendPasswordResetEmailToken  string
	SendPasswordResetEmailError  error

	IsConfiguredCalled bool
	IsConfiguredResult bool

	SendContactEmailCalled  bool
	SendContactEmailName    string
	SendContactEmailEmail   string
	SendContactEmailSubject string
	SendContactEmailMessage string
	SendContactEmailError   error

	SendEmailChangeVerificationCalled bool
	SendEmailChangeVerificationEmail  string
	SendEmailChangeVerificationToken  string
	SendEmailChangeVerificationError  error
}

// SendVerificationEmail is a mock implementation that records the call
func (m *MockEmailService) SendVerificationEmail(email, token string) error {
	m.SendVerificationEmailCalled = true
	m.SendVerificationEmailEmail = email
	m.SendVerificationEmailToken = token
	return m.SendVerificationEmailError
}

// SendPasswordResetEmail is a mock implementation that records the call
func (m *MockEmailService) SendPasswordResetEmail(email, token string) error {
	m.SendPasswordResetEmailCalled = true
	m.SendPasswordResetEmailEmail = email
	m.SendPasswordResetEmailToken = token
	return m.SendPasswordResetEmailError
}

// SendContactEmail is a mock implementation that records the call
func (m *MockEmailService) SendContactEmail(name, email, subject, message string) error {
	m.SendContactEmailCalled = true
	m.SendContactEmailName = name
	m.SendContactEmailEmail = email
	m.SendContactEmailSubject = subject
	m.SendContactEmailMessage = message
	return m.SendContactEmailError
}

// SendEmailChangeVerification is a mock implementation that records the call
func (m *MockEmailService) SendEmailChangeVerification(email, token string) error {
	m.SendEmailChangeVerificationCalled = true
	m.SendEmailChangeVerificationEmail = email
	m.SendEmailChangeVerificationToken = token
	return m.SendEmailChangeVerificationError
}

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

// Setup TestUserRegistration mock responses
func setupTestRouter(t *testing.T) (*gin.Engine, *MockDBWithContext, *MockEmailService) {
	// Create Gin router
	router := gin.Default()

	// Configure Gin for tests
	gin.SetMode(gin.TestMode)

	// Create mock database
	mockDB := new(MockDBWithContext)

	// Create test user
	hashedPassword, err := database.HashPassword("password123")
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
	mockDB.On("AuthenticateUser", mock.Anything, "test@example.com", "password123").Return(testUser, nil)
	mockDB.On("AuthenticateUser", mock.Anything, "test@example.com", "wrongpassword").Return(nil, nil)
	mockDB.On("AuthenticateUser", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

	// Mock get user by email - use this for specific cases
	// First mock it to return nil for first call in register test
	mockDB.On("GetUserByEmail", mock.Anything, "test@example.com").Return(nil, nil).Once()
	mockDB.On("GetUserByEmail", mock.Anything, "nonexistent@example.com").Return(nil, nil)
	mockDB.On("GetUserByEmail", mock.Anything, "duplicate@example.com").Return(testUser, nil)
	mockDB.On("GetUserByEmail", mock.Anything, mock.Anything).Return(nil, nil)

	// Create a mock email service
	mockEmail := &MockEmailService{
		IsConfiguredResult: true,
	}

	// Setup mock responses
	testUser2 := &database.User{
		Email:    "test2@example.com",
		Verified: true,
	}
	testUser2.ID = 2 // Set ID

	// Mock create user
	mockDB.On("CreateUser", mock.Anything, "test@example.com", "password123").Return(testUser, nil)
	mockDB.On("CreateUser", mock.Anything, "test2@example.com", "password123").Return(testUser2, nil)

	// Mock GetUserByEmail specifically for the first "test@example.com" calls
	mockDB.On("GetUserByEmail", mock.Anything, "test2@example.com").Return(nil, nil).Once()

	// Mock AuthenticateUser for test2@example.com
	mockDB.On("AuthenticateUser", mock.Anything, "test2@example.com", "password123").Return(testUser2, nil)

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

func TestAuthenticationFlow(t *testing.T) {
	// Setup
	router, _, mockEmail := setupTestRouter(t)

	// Test the full authentication flow
	t.Run("Authentication flow changes navigation bar in HTML", func(t *testing.T) {
		// Step 1: Check unauthenticated state - Home page should show Login and Register
		t.Run("Unauthenticated user sees login and register links in HTML", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			require.Equal(t, http.StatusOK, resp.Code)
			body := resp.Body.String()

			// Verify the actual HTML contains login and register links
			assert.Contains(t, body, `href="/login"`)
			assert.Contains(t, body, `href="/register"`)
			assert.NotContains(t, body, `You are logged in as`)
		})

		// Step 2: Register a new user
		t.Run("User can register and is redirected to verification page", func(t *testing.T) {
			// Submit registration form
			form := url.Values{}
			form.Add("email", "test@example.com")
			form.Add("password", "password123")
			form.Add("password_confirm", "password123")

			req := httptest.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			// Should redirect to verification-sent page
			require.Equal(t, http.StatusSeeOther, resp.Code)
			require.Equal(t, "/verification-sent", resp.Header().Get("Location"))

			// Verify that the verification email was sent
			assert.True(t, mockEmail.SendVerificationEmailCalled, "SendVerificationEmail should have been called")
			assert.Equal(t, "test@example.com", mockEmail.SendVerificationEmailEmail, "SendVerificationEmail should have been called with the correct email")

			// Check that user is still unauthenticated after registration
			req = httptest.NewRequest("GET", "/", nil)
			resp = httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			// Verify the HTML still shows unauthenticated state
			homeHTML := resp.Body.String()
			assert.Contains(t, homeHTML, `/login`)
			assert.Contains(t, homeHTML, `/register`)
			assert.NotContains(t, homeHTML, `/logout`)
		})

		// Step 3: Verify email
		t.Run("User can verify email and then login", func(t *testing.T) {
			// Simulate email verification
			req := httptest.NewRequest("GET", "/verify-email?token=test-token", nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			// Should redirect to login page with verified parameter
			require.Equal(t, http.StatusSeeOther, resp.Code)
			require.Equal(t, "/login?verified=true", resp.Header().Get("Location"))

			// Now login with the verified user
			form := url.Values{}
			form.Add("email", "test@example.com")
			form.Add("password", "password123")

			req = httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			resp = httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			// Should redirect to owner page
			require.Equal(t, http.StatusSeeOther, resp.Code)
			require.Equal(t, "/owner", resp.Header().Get("Location"))

			// Extract auth cookie
			cookies := resp.Result().Cookies()
			var authCookie *http.Cookie
			for _, cookie := range cookies {
				if cookie.Name == "auth-session" {
					authCookie = cookie
					break
				}
			}
			require.NotNil(t, authCookie, "Auth cookie should be set after login")

			// Now check the home page with the auth cookie
			req = httptest.NewRequest("GET", "/", nil)
			req.AddCookie(authCookie)
			resp = httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			// Check that the home page shows the logout link
			homeHTML := resp.Body.String()
			assert.Contains(t, homeHTML, `/logout`)
			assert.NotContains(t, homeHTML, `/login`)
			assert.NotContains(t, homeHTML, `/register`)
		})

		// Step 4: Logout
		t.Run("User can logout and HTML changes back to show login/register links", func(t *testing.T) {
			// Logout
			req := httptest.NewRequest("GET", "/logout", nil)
			req.AddCookie(&http.Cookie{
				Name:  "auth-session",
				Value: "1", // User ID from the test user
			})
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			// Should redirect to home page
			require.Equal(t, http.StatusSeeOther, resp.Code)
			require.Equal(t, "/", resp.Header().Get("Location"))

			// Check that auth cookie is cleared
			cookies := resp.Result().Cookies()
			var authCookie *http.Cookie
			for _, cookie := range cookies {
				if cookie.Name == "auth-session" {
					authCookie = cookie
					break
				}
			}
			require.NotNil(t, authCookie, "Auth cookie should be present")
			assert.Equal(t, "", authCookie.Value, "Auth cookie should be cleared")
			assert.Less(t, authCookie.MaxAge, 0, "Auth cookie should be expired")

			// Now check the home page to verify unauthenticated state
			req = httptest.NewRequest("GET", "/", nil)
			resp = httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			// Verify the HTML shows unauthenticated state
			homeHTML := resp.Body.String()
			assert.Contains(t, homeHTML, `/login`)
			assert.Contains(t, homeHTML, `/register`)
			assert.NotContains(t, homeHTML, `/logout`)
		})
	})
}

// TestRealHTMLOutput tests the actual HTML output of the templates with different authentication states
func TestRealHTMLOutput(t *testing.T) {
	// Create Gin router and mock objects
	router := gin.Default()
	gin.SetMode(gin.TestMode)

	// Create mock database
	mockDB := new(MockDBWithContext)

	// Create test DB
	testDb := testutils.SharedTestService()
	mockDB.On("GetDB").Return(testDb.GetDB())

	// Setup test user with proper hashed password
	hashedPassword, err := database.HashPassword("password123")
	if err != nil {
		t.Fatal(err)
	}

	// Create authenticated test user
	testUser := &database.User{
		Email:    "test3@example.com",
		Password: hashedPassword,
		Verified: true,
	}
	testUser.ID = 3

	// Create mock email service
	mockEmail := &MockEmailService{
		IsConfiguredResult: true,
	}

	// Setup all required mocks
	mockDB.On("Health").Return(map[string]string{"status": "up"})
	mockDB.On("CreateUser", mock.Anything, "test3@example.com", "password123").Return(testUser, nil)
	mockDB.On("GetUserByEmail", mock.Anything, "test3@example.com").Return(nil, nil).Once()
	mockDB.On("GetUserByEmail", mock.Anything, "test3@example.com").Return(testUser, nil).Maybe()
	mockDB.On("AuthenticateUser", mock.Anything, "test3@example.com", "password123").Return(testUser, nil)
	mockDB.On("UpdateUser", mock.Anything, mock.AnythingOfType("*database.User")).Return(nil)

	// Setup controllers
	authController := controller.NewAuthController(mockDB)
	authController.SetEmailService(mockEmail)

	// Setup middleware
	router.Use(func(c *gin.Context) {
		// Set IP address
		c.Set("remote_ip", "192.0.2.1")

		// Get user auth info
		userInfo, authenticated := authController.GetCurrentUser(c)

		// Create auth data
		authData := data.NewAuthData()
		authData.Authenticated = authenticated

		// Set email if authenticated
		if authenticated {
			authData.Email = userInfo.GetUserName()
		}

		// Set to context
		c.Set("authData", authData)
		c.Set("authController", authController)

		c.Next()
	})

	// Override render methods
	authController.RenderLogin = func(c *gin.Context, d interface{}) {
		authData := d.(data.AuthData)
		_, authenticated := authController.GetCurrentUser(c)
		authData.Authenticated = authenticated
		if authData.Title == "" {
			authData.Title = "Login"
		}
		authviews.Login(authData).Render(c.Request.Context(), c.Writer)
	}

	authController.RenderRegister = func(c *gin.Context, d interface{}) {
		authData := d.(data.AuthData)
		_, authenticated := authController.GetCurrentUser(c)
		authData.Authenticated = authenticated
		if authData.Title == "" {
			authData.Title = "Register"
		}
		authviews.Register(authData).Render(c.Request.Context(), c.Writer)
	}

	// Setup routes
	router.GET("/", func(c *gin.Context) {
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
	router.GET("/owner", func(c *gin.Context) {
		c.String(http.StatusOK, "Owner page")
	})

	t.Run("Registration and login flow works correctly", func(t *testing.T) {
		// Test unauthenticated state
		req := httptest.NewRequest("GET", "/", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		// Verify HTML contains login and register links
		unauthHTML := resp.Body.String()
		assert.Contains(t, unauthHTML, `href="/login"`)
		assert.Contains(t, unauthHTML, `href="/register"`)
		assert.NotContains(t, unauthHTML, `href="/logout"`)

		// Step 1: Register
		form := url.Values{}
		form.Add("email", "test3@example.com")
		form.Add("password", "password123")
		form.Add("password_confirm", "password123")

		regReq := httptest.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
		regReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		regResp := httptest.NewRecorder()
		router.ServeHTTP(regResp, regReq)

		// Should redirect to verification-sent page
		require.Equal(t, http.StatusSeeOther, regResp.Code)
		require.Equal(t, "/verification-sent", regResp.Header().Get("Location"))

		// Step 2: Login (skipping verification since we already mocked the user as verified)
		loginForm := url.Values{}
		loginForm.Add("email", "test3@example.com")
		loginForm.Add("password", "password123")

		loginReq := httptest.NewRequest("POST", "/login", strings.NewReader(loginForm.Encode()))
		loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		loginResp := httptest.NewRecorder()
		router.ServeHTTP(loginResp, loginReq)

		// Should redirect to owner page
		require.Equal(t, http.StatusSeeOther, loginResp.Code, "Login should redirect to owner page")
		require.Equal(t, "/owner", loginResp.Header().Get("Location"))

		// Extract auth cookie
		cookies := loginResp.Result().Cookies()
		var authCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == "auth-session" {
				authCookie = cookie
				break
			}
		}
		require.NotNil(t, authCookie, "Auth cookie should be set after login")

		// Check home page with auth cookie
		authReq := httptest.NewRequest("GET", "/", nil)
		authReq.AddCookie(authCookie)
		authResp := httptest.NewRecorder()
		router.ServeHTTP(authResp, authReq)

		// Verify HTML changes for authenticated user
		authHTML := authResp.Body.String()
		assert.Contains(t, authHTML, "Logout")
		assert.NotContains(t, authHTML, `href="/login"`)
		assert.NotContains(t, authHTML, `href="/register"`)
	})
}

// TestEmailVerificationFlow tests the complete email verification flow
func TestEmailVerificationFlow(t *testing.T) {
	// Setup
	router, mockDB, mockEmail := setupTestRouter(t)

	// Mock the verification token and user
	testUser := &database.User{
		Email:             "verify@example.com",
		VerificationToken: "test-verification-token",
	}
	database.SetUserID(testUser, 3)

	// Setup mock responses for verification
	mockDB.On("GetUserByEmail", mock.Anything, "verify@example.com").Return(nil, nil).Once()
	mockDB.On("CreateUser", mock.Anything, "verify@example.com", "password123").Return(testUser, nil)
	mockDB.On("UpdateUser", mock.Anything, mock.AnythingOfType("*database.User")).Return(nil)
	mockDB.On("GetUserByVerificationToken", mock.Anything, "test-verification-token").Return(testUser, nil)
	mockDB.On("VerifyUserEmail", mock.Anything, "test-verification-token").Return(testUser, nil)

	t.Run("Registration redirects to verification-sent page", func(t *testing.T) {
		// Submit registration form
		form := url.Values{}
		form.Add("email", "verify@example.com")
		form.Add("password", "password123")
		form.Add("password_confirm", "password123")

		req := httptest.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		// Don't set X-Test header so we get the real flow
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		// Should redirect to verification-sent page
		require.Equal(t, http.StatusSeeOther, resp.Code)
		require.Equal(t, "/verification-sent", resp.Header().Get("Location"))

		// Verify that the verification email was sent
		assert.True(t, mockEmail.SendVerificationEmailCalled, "SendVerificationEmail should have been called")
		assert.Equal(t, "verify@example.com", mockEmail.SendVerificationEmailEmail, "SendVerificationEmail should have been called with the correct email")
		assert.NotEmpty(t, mockEmail.SendVerificationEmailToken, "SendVerificationEmail should have been called with a token")
	})

	t.Run("Verification token redirects to login page with verified parameter", func(t *testing.T) {
		// Visit verification link
		req := httptest.NewRequest("GET", "/verify-email?token=test-verification-token", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		// Should redirect to login page with verified=true
		require.Equal(t, http.StatusSeeOther, resp.Code)
		require.Equal(t, "/login?verified=true", resp.Header().Get("Location"))
	})
}

// TestRegistrationWithoutAuthentication tests that a user is not authenticated after registration
// until they verify their email and log in
func TestRegistrationWithoutAuthentication(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a mock database
	mockDB := new(MockDBWithContext)

	// Create a mock email service
	mockEmail := &MockEmailService{
		IsConfiguredResult: true,
	}

	// Create a test DB instance to handle Unscoped queries
	testDb := testutils.SharedTestService()

	// Mock the GetDB method to return a real DB instance
	mockDB.On("GetDB").Return(testDb.GetDB())

	// Mock the verification token and user
	testUser := &database.User{
		Email:             "unauthenticated@example.com",
		VerificationToken: "test-verification-token",
	}
	database.SetUserID(testUser, 4)

	// Setup mock responses for verification
	mockDB.On("GetUserByEmail", mock.Anything, "unauthenticated@example.com").Return(nil, nil).Once()
	mockDB.On("CreateUser", mock.Anything, "unauthenticated@example.com", "password123").Return(testUser, nil)
	mockDB.On("UpdateUser", mock.Anything, mock.AnythingOfType("*database.User")).Return(nil)
	mockDB.On("GetUserByVerificationToken", mock.Anything, "test-verification-token").Return(testUser, nil)
	mockDB.On("VerifyUserEmail", mock.Anything, "test-verification-token").Run(func(args mock.Arguments) {
		// Update the user's verified status
		testUser.Verified = true
	}).Return(testUser, nil)
	mockDB.On("AuthenticateUser", mock.Anything, "unauthenticated@example.com", "password123").Return(testUser, nil)

	// Add default mocks for other methods that might be called
	mockDB.On("Health").Return(map[string]string{"status": "up"})

	// Create a new router
	router := gin.New()

	// Add recovery middleware
	router.Use(gin.Recovery())

	// Create a new auth controller
	authController := controller.NewAuthController(mockDB)

	// Set the mock email service
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

	// Add a route to check authentication status
	router.GET("/check-auth", func(c *gin.Context) {
		// Get the auth data from the context
		authData, exists := c.Get("authData")
		if !exists {
			c.String(http.StatusOK, "not-authenticated")
			return
		}

		// Check if authenticated
		if authData.(data.AuthData).Authenticated {
			c.String(http.StatusOK, "authenticated")
		} else {
			c.String(http.StatusOK, "not-authenticated")
		}
	})

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

	t.Run("User is not authenticated after registration", func(t *testing.T) {
		// Submit registration form
		form := url.Values{}
		form.Add("email", "unauthenticated@example.com")
		form.Add("password", "password123")
		form.Add("password_confirm", "password123")

		req := httptest.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		// Don't set X-Test header so we get the real flow
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		// Should redirect to verification-sent page
		require.Equal(t, http.StatusSeeOther, resp.Code)
		require.Equal(t, "/verification-sent", resp.Header().Get("Location"))

		// Check if auth cookie is set - it should NOT be set or should be invalid
		cookies := resp.Result().Cookies()
		var authCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == "auth-session" {
				authCookie = cookie
				break
			}
		}

		// Check authentication status
		authCheckReq := httptest.NewRequest("GET", "/check-auth", nil)
		if authCookie != nil {
			authCheckReq.AddCookie(authCookie)
		}
		authCheckResp := httptest.NewRecorder()
		router.ServeHTTP(authCheckResp, authCheckReq)

		// Should not be authenticated
		assert.Equal(t, "not-authenticated", authCheckResp.Body.String(), "User should not be authenticated after registration")
	})

	t.Run("User can verify email and then login", func(t *testing.T) {
		// Visit verification link
		verifyReq := httptest.NewRequest("GET", "/verify-email?token=test-verification-token", nil)
		verifyResp := httptest.NewRecorder()
		router.ServeHTTP(verifyResp, verifyReq)

		// Should redirect to login page with verified parameter
		require.Equal(t, http.StatusSeeOther, verifyResp.Code)
		require.Equal(t, "/login?verified=true", verifyResp.Header().Get("Location"))

		// Now login with the verified account
		loginForm := url.Values{}
		loginForm.Add("email", "unauthenticated@example.com")
		loginForm.Add("password", "password123")

		loginReq := httptest.NewRequest("POST", "/login", strings.NewReader(loginForm.Encode()))
		loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		loginResp := httptest.NewRecorder()
		router.ServeHTTP(loginResp, loginReq)

		// Should redirect to owner page
		require.Equal(t, http.StatusSeeOther, loginResp.Code)
		require.Equal(t, "/owner", loginResp.Header().Get("Location"))

		// Check that auth cookie is now set
		loginCookies := loginResp.Result().Cookies()
		var loginAuthCookie *http.Cookie
		for _, cookie := range loginCookies {
			if cookie.Name == "auth-session" {
				loginAuthCookie = cookie
				break
			}
		}
		require.NotNil(t, loginAuthCookie, "Auth cookie should be set after login")
		assert.NotEmpty(t, loginAuthCookie.Value, "Auth cookie should have a value after login")

		// Check authentication status
		authCheckReq := httptest.NewRequest("GET", "/check-auth", nil)
		authCheckReq.AddCookie(loginAuthCookie)
		authCheckResp := httptest.NewRecorder()
		router.ServeHTTP(authCheckResp, authCheckReq)

		// Should be authenticated
		assert.Equal(t, "authenticated", authCheckResp.Body.String(), "User should be authenticated after login")
	})
}
