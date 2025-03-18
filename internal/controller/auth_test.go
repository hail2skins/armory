package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/testutils"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// MockDB is a mock implementation of the database.Service interface
type MockDB struct {
	mock.Mock
}

func (m *MockDB) Health() map[string]string {
	args := m.Called()
	return args.Get(0).(map[string]string)
}

func (m *MockDB) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDB) CreateUser(ctx context.Context, email, password string) (*database.User, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

func (m *MockDB) GetUserByEmail(ctx context.Context, email string) (*database.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

func (m *MockDB) AuthenticateUser(ctx context.Context, email, password string) (*database.User, error) {
	args := m.Called(ctx, email, password)
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

func (m *MockDB) GetUserByVerificationToken(ctx context.Context, token string) (*database.User, error) {
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

func (m *MockDB) UpdateUser(ctx context.Context, user *database.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

// Payment methods
func (m *MockDB) CreatePayment(payment *models.Payment) error {
	args := m.Called(payment)
	return args.Error(0)
}

func (m *MockDB) GetPaymentsByUserID(userID uint) ([]models.Payment, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
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

func (m *MockDB) GetUserByID(id uint) (*database.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

func (m *MockDB) GetUserByStripeCustomerID(customerID string) (*database.User, error) {
	args := m.Called(customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

// Add admin methods to MockDB

// FindAllManufacturers retrieves all manufacturers
func (m *MockDB) FindAllManufacturers() ([]models.Manufacturer, error) {
	args := m.Called()
	return args.Get(0).([]models.Manufacturer), args.Error(1)
}

// FindManufacturerByID retrieves a manufacturer by ID
func (m *MockDB) FindManufacturerByID(id uint) (*models.Manufacturer, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Manufacturer), args.Error(1)
}

// CreateManufacturer creates a new manufacturer
func (m *MockDB) CreateManufacturer(manufacturer *models.Manufacturer) error {
	args := m.Called(manufacturer)
	return args.Error(0)
}

// UpdateManufacturer updates a manufacturer
func (m *MockDB) UpdateManufacturer(manufacturer *models.Manufacturer) error {
	args := m.Called(manufacturer)
	return args.Error(0)
}

// DeleteManufacturer deletes a manufacturer
func (m *MockDB) DeleteManufacturer(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

// FindAllCalibers retrieves all calibers
func (m *MockDB) FindAllCalibers() ([]models.Caliber, error) {
	args := m.Called()
	return args.Get(0).([]models.Caliber), args.Error(1)
}

// FindCaliberByID retrieves a caliber by ID
func (m *MockDB) FindCaliberByID(id uint) (*models.Caliber, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Caliber), args.Error(1)
}

// CreateCaliber creates a new caliber
func (m *MockDB) CreateCaliber(caliber *models.Caliber) error {
	args := m.Called(caliber)
	return args.Error(0)
}

// UpdateCaliber updates a caliber
func (m *MockDB) UpdateCaliber(caliber *models.Caliber) error {
	args := m.Called(caliber)
	return args.Error(0)
}

// DeleteCaliber deletes a caliber
func (m *MockDB) DeleteCaliber(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

// FindAllWeaponTypes retrieves all weapon types
func (m *MockDB) FindAllWeaponTypes() ([]models.WeaponType, error) {
	args := m.Called()
	return args.Get(0).([]models.WeaponType), args.Error(1)
}

// FindWeaponTypeByID retrieves a weapon type by ID
func (m *MockDB) FindWeaponTypeByID(id uint) (*models.WeaponType, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.WeaponType), args.Error(1)
}

// CreateWeaponType creates a new weapon type
func (m *MockDB) CreateWeaponType(weaponType *models.WeaponType) error {
	args := m.Called(weaponType)
	return args.Error(0)
}

// UpdateWeaponType updates a weapon type
func (m *MockDB) UpdateWeaponType(weaponType *models.WeaponType) error {
	args := m.Called(weaponType)
	return args.Error(0)
}

// DeleteWeaponType deletes a weapon type
func (m *MockDB) DeleteWeaponType(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

// GetDB returns the underlying *gorm.DB instance
func (m *MockDB) GetDB() *gorm.DB {
	args := m.Called()
	return args.Get(0).(*gorm.DB)
}

// MockEmailService is a mock implementation of the email.EmailService interface
type MockEmailService struct {
	mock.Mock
}

func (m *MockEmailService) SendVerificationEmail(email, token string) error {
	args := m.Called(email, token)
	return args.Error(0)
}

func (m *MockEmailService) SendPasswordResetEmail(email, token string) error {
	args := m.Called(email, token)
	return args.Error(0)
}

func (m *MockEmailService) SendContactEmail(name, email, subject, message string) error {
	args := m.Called(name, email, subject, message)
	return args.Error(0)
}

func TestAuthenticationFlow(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	t.Run("Authentication flow changes navigation bar", func(t *testing.T) {
		mockDB := new(mocks.MockDB)
		mockEmail := new(MockEmailService)

		// Create a test user
		testUser := &database.User{
			Email:    "test@example.com",
			Password: "$2a$10$1234567890123456789012", // Fake hashed password
			Verified: false,
		}
		database.SetUserID(testUser, 1)

		// Setup mock responses
		mockDB.On("GetUserByEmail", mock.Anything, "test@example.com").Return(nil, nil).Once()
		mockDB.On("GetUserByEmail", mock.Anything, "test@example.com").Return(testUser, nil).Times(3)
		mockDB.On("CreateUser", mock.Anything, "test@example.com", "password123").Return(testUser, nil)
		mockDB.On("AuthenticateUser", mock.Anything, "test@example.com", "password123").Return(testUser, nil)
		mockDB.On("GetUserByVerificationToken", mock.Anything, "test-token").Return(testUser, nil)
		mockDB.On("VerifyUserEmail", mock.Anything, "test-token").Return(testUser, nil)

		// Mock UpdateUser to return nil error
		mockDB.On("UpdateUser", mock.Anything, mock.AnythingOfType("*database.User")).Return(nil)

		// Mock email service
		mockEmail.On("SendVerificationEmail", mock.Anything, mock.Anything).Return(nil)

		// Create auth controller
		authController := NewAuthController(mockDB)

		// Set the email service
		authController.emailService = mockEmail

		// Create a test HTTP server
		router := gin.New()

		// Setup render functions to capture HTML output
		var loginHTML, registerHTML string

		authController.RenderLogin = func(c *gin.Context, data interface{}) {
			loginHTML = "Login Form"
			c.String(http.StatusOK, loginHTML)
		}

		authController.RenderRegister = func(c *gin.Context, data interface{}) {
			registerHTML = "Register Form"
			c.String(http.StatusOK, registerHTML)
		}

		// Setup routes
		router.GET("/login", authController.LoginHandler)
		router.POST("/login", authController.LoginHandler)
		router.GET("/register", authController.RegisterHandler)
		router.POST("/register", authController.RegisterHandler)
		router.GET("/logout", authController.LogoutHandler)
		router.GET("/verify-email", authController.VerifyEmailHandler)
		router.GET("/verification-sent", func(c *gin.Context) {
			c.String(http.StatusOK, "Verification email sent")
		})

		// Add auth-links endpoint
		router.GET("/auth-links", func(c *gin.Context) {
			_, authenticated := authController.GetCurrentUser(c)
			c.Header("Content-Type", "text/html")

			if authenticated {
				c.String(http.StatusOK, `<a href="/logout">Logout</a>`)
			} else {
				c.String(http.StatusOK, `<a href="/login">Login</a><a href="/register">Register</a>`)
			}
		})

		// Add home page
		router.GET("/", func(c *gin.Context) {
			_, authenticated := authController.GetCurrentUser(c)

			// Simplified home page with navigation
			html := `
			<nav>
				<a href="/">Home</a>
				<span id="auth-links">
			`

			if authenticated {
				html += `<a href="/logout">Logout</a>`
			} else {
				html += `<a href="/login">Login</a><a href="/register">Register</a>`
			}

			html += `
				</span>
			</nav>
			<main>Welcome to Armory</main>
			`

			c.Header("Content-Type", "text/html")
			c.String(http.StatusOK, html)
		})

		// Step 1: Check unauthenticated state
		t.Run("Unauthenticated user sees login and register links", func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/auth-links", nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusOK, resp.Code)
			assert.Contains(t, resp.Body.String(), `<a href="/login">Login</a>`)
			assert.Contains(t, resp.Body.String(), `<a href="/register">Register</a>`)
		})

		// Step 2: Register a new user
		t.Run("User can register", func(t *testing.T) {
			form := url.Values{}
			form.Add("email", "test@example.com")
			form.Add("password", "password123")
			form.Add("password_confirm", "password123")

			req, _ := http.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			// Don't set X-Test header so we get redirected to verification-sent page
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			// Should redirect to verification-sent page
			assert.Equal(t, http.StatusSeeOther, resp.Code)
			assert.Equal(t, "/verification-sent", resp.Header().Get("Location"))
		})

		// Step 3: Verify email and login
		t.Run("User can verify email and login", func(t *testing.T) {
			// Verify the email
			req, _ := http.NewRequest("GET", "/verify-email?token=test-token", nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			// Check that the user is redirected to the login page
			assert.Equal(t, 303, resp.Code)
			assert.Equal(t, "/login?verified=true", resp.Header().Get("Location"))

			// Login with the verified user
			loginForm := url.Values{}
			loginForm.Add("email", "test@example.com")
			loginForm.Add("password", "password123")
			req, _ = http.NewRequest("POST", "/login", strings.NewReader(loginForm.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			resp = httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			// Check that the user is redirected to the owner page
			assert.Equal(t, 303, resp.Code)
			assert.Equal(t, "/owner", resp.Header().Get("Location"))
		})

		// Step 4: Check authenticated state
		t.Run("Authenticated user sees logout link", func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/auth-links", nil)
			// Add auth cookie from previous step
			req.AddCookie(&http.Cookie{
				Name:  "auth-session",
				Value: "1", // User ID from the test user
			})
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusOK, resp.Code)
			assert.Contains(t, resp.Body.String(), `<a href="/logout">Logout</a>`)
			assert.NotContains(t, resp.Body.String(), `<a href="/login">Login</a>`)
		})

		// Step 5: Logout
		t.Run("User can logout", func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/logout", nil)
			// Add auth cookie
			req.AddCookie(&http.Cookie{
				Name:  "auth-session",
				Value: "1", // User ID from the test user
			})
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			// Should redirect to home page
			assert.Equal(t, http.StatusSeeOther, resp.Code)
			assert.Equal(t, "/", resp.Header().Get("Location"))

			// Check that auth cookie is cleared
			cookies := resp.Result().Cookies()
			var authCookie *http.Cookie
			for _, cookie := range cookies {
				if cookie.Name == "auth-session" {
					authCookie = cookie
					break
				}
			}
			assert.NotNil(t, authCookie, "Auth cookie should be present")
			assert.Equal(t, "", authCookie.Value, "Auth cookie should be cleared")
			assert.Less(t, authCookie.MaxAge, 0, "Auth cookie should be expired")
		})

		// Step 6: Check unauthenticated state again
		t.Run("After logout, user sees login and register links", func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/auth-links", nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusOK, resp.Code)
			assert.Contains(t, resp.Body.String(), `<a href="/login">Login</a>`)
			assert.Contains(t, resp.Body.String(), `<a href="/register">Register</a>`)
		})
	})
}

func TestLogoutRedirectsToHome(t *testing.T) {
	// Create a test database
	db := testutils.NewTestService()
	defer db.Close()

	// Create a test user
	ctx := context.Background()
	user, err := testutils.CreateTestUser(ctx, db, "test@example.com", "password123")
	require.NoError(t, err)
	require.NotNil(t, user)

	// Create a new auth controller
	authController := NewAuthController(db)

	// Create a new router
	router := gin.New()

	// Set up the routes
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Home Page")
	})
	router.GET("/logout", authController.LogoutHandler)

	// Create a request to logout
	req, _ := http.NewRequest("GET", "/logout", nil)
	// Add auth cookie
	req.AddCookie(&http.Cookie{
		Name:  "auth-session",
		Value: strconv.FormatUint(uint64(user.ID), 10),
	})

	// Create a response recorder
	resp := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(resp, req)

	// Check that we get redirected to the home page
	assert.Equal(t, http.StatusSeeOther, resp.Code)
	assert.Equal(t, "/", resp.Header().Get("Location"))

	// Check that the auth cookie is cleared
	cookies := resp.Result().Cookies()
	var authCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "auth-session" {
			authCookie = cookie
			break
		}
	}
	assert.NotNil(t, authCookie, "Auth cookie should be present")
	assert.Equal(t, "", authCookie.Value, "Auth cookie should be cleared")
	assert.Less(t, authCookie.MaxAge, 0, "Auth cookie should be expired")
}

func TestEmailVerification(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockDB := new(mocks.MockDB)

	// Setup mock responses
	testUser := &database.User{
		Email:             "test@example.com",
		VerificationToken: "test-token",
		Verified:          false,
	}
	database.SetUserID(testUser, 1)

	// Mock GetUserByVerificationToken
	mockDB.On("GetUserByVerificationToken", mock.Anything, "test-token").Return(testUser, nil)
	mockDB.On("GetUserByVerificationToken", mock.Anything, "invalid-token").Return(nil, nil)

	// Mock VerifyUserEmail
	mockDB.On("VerifyUserEmail", mock.Anything, "test-token").Return(testUser, nil)
	mockDB.On("VerifyUserEmail", mock.Anything, "invalid-token").Return(nil, nil)

	// Create auth controller
	authController := NewAuthController(mockDB)

	// Create a test HTTP server
	router := gin.New()

	// Setup routes
	router.GET("/verify-email", authController.VerifyEmailHandler)

	t.Run("User can verify email with valid token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/verify-email?token=test-token", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		// Should redirect to login page with success message
		assert.Equal(t, http.StatusSeeOther, resp.Code)
		assert.Equal(t, "/login?verified=true", resp.Header().Get("Location"))
	})

	t.Run("Invalid verification token returns error", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/verify-email?token=invalid-token", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "Invalid verification token")
	})
}

func TestRegisterUser(t *testing.T) {
	// Create a test database
	db := testutils.NewTestService()
	defer db.Close()

	// Create a new auth controller
	authController := NewAuthController(db)

	// Create a mock email service
	mockEmail := new(MockEmailService)
	mockEmail.On("SendVerificationEmail", mock.Anything, mock.Anything).Return(nil)
	authController.SetEmailService(mockEmail)

	// Create a new router
	router := gin.New()

	// Set up the routes
	router.POST("/register", authController.RegisterHandler)
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Home Page")
	})

	// Create registration form data
	form := url.Values{}
	form.Add("email", "newuser@example.com")
	form.Add("password", "password123")
	form.Add("password_confirm", "password123")

	// Create a request to register
	req, _ := http.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Don't set X-Test header so we get redirected to verification-sent page

	// Create a response recorder
	resp := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(resp, req)

	// Check that we get redirected to the verification-sent page
	assert.Equal(t, http.StatusSeeOther, resp.Code)
	assert.Equal(t, "/verification-sent", resp.Header().Get("Location"))

	// Verify the user was created in the database
	ctx := context.Background()
	user, err := db.GetUserByEmail(ctx, "newuser@example.com")
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "newuser@example.com", user.Email)
	assert.NotEmpty(t, user.Password, "Password should be hashed and stored")
	assert.False(t, user.Verified, "User should not be verified initially")
	assert.NotEmpty(t, user.VerificationToken, "Verification token should be generated")

	// Verify that the verification email was sent
	mockEmail.AssertCalled(t, "SendVerificationEmail", "newuser@example.com", mock.Anything)
}

// TestLoginRedirectsToOwner tests that users are redirected to /owner after successful login
func TestLoginRedirectsToOwner(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create mock objects
	mockDB := new(mocks.MockDB)

	// Create a test user
	testUser := &database.User{
		Model: gorm.Model{
			ID: 1,
		},
		Email:    "test@example.com",
		Password: "hashed_password",
	}

	// Setup expectations
	mockDB.On("AuthenticateUser", mock.Anything, "test@example.com", "password123").Return(testUser, nil)

	// Create the controller
	authController := NewAuthController(mockDB)

	// Setup router
	router := gin.Default()
	router.POST("/login", authController.LoginHandler)

	// Create a login request
	form := url.Values{}
	form.Add("email", "test@example.com")
	form.Add("password", "password123")
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(resp, req)

	// Check the response - should be a redirect to /owner
	assert.Equal(t, 303, resp.Code)
	assert.Equal(t, "/owner", resp.Header().Get("Location"))

	// Verify that all expectations were met
	mockDB.AssertExpectations(t)
}

// TestLoginRedirectsToOwnerWithWelcomeMessage tests that users are redirected to /owner after successful login with a welcome message
func TestLoginRedirectsToOwnerWithWelcomeMessage(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create mock objects
	mockDB := new(mocks.MockDB)

	// Create a test user
	testUser := &database.User{
		Model: gorm.Model{
			ID: 1,
		},
		Email:    "test@example.com",
		Password: "hashed_password",
	}

	// Setup expectations
	mockDB.On("AuthenticateUser", mock.Anything, "test@example.com", "password123").Return(testUser, nil)

	// Create the controller
	authController := NewAuthController(mockDB)

	// Setup router with a custom middleware to capture flash messages
	router := gin.Default()
	var capturedFlash string
	router.Use(func(c *gin.Context) {
		c.Set("setFlash", func(msg string) {
			capturedFlash = msg
		})
		c.Next()
	})
	router.POST("/login", authController.LoginHandler)

	// Create a login request
	form := url.Values{}
	form.Add("email", "test@example.com")
	form.Add("password", "password123")
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(resp, req)

	// Check the response - should be a redirect to /owner
	assert.Equal(t, 303, resp.Code)
	assert.Equal(t, "/owner", resp.Header().Get("Location"))

	// Check that a welcome flash message was set
	assert.Contains(t, capturedFlash, "Welcome back")

	// Verify that all expectations were met
	mockDB.AssertExpectations(t)
}

// IsRecoveryExpired implements the database.Service interface
func (m *MockDB) IsRecoveryExpired(ctx context.Context, token string) (bool, error) {
	args := m.Called(ctx, token)
	return args.Bool(0), args.Error(1)
}
