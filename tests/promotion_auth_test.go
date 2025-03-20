package tests

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/services"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// PromotionAuthTestSuite is a test suite for testing promotion integration with auth
type PromotionAuthTestSuite struct {
	suite.Suite
	Router           *gin.Engine
	MockDB           *mocks.MockDB
	AuthController   *controller.AuthController
	PromotionService *services.PromotionService
	ActivePromotion  *models.Promotion
}

// SetupTest sets up the test suite
func (s *PromotionAuthTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	s.MockDB = new(mocks.MockDB)
	s.PromotionService = services.NewPromotionService(s.MockDB)

	// Set up auth controller with promotion service
	s.AuthController = controller.NewAuthController(s.MockDB)

	// Set the promotion service on the auth controller using the proper type
	s.AuthController.SetPromotionService(s.PromotionService)

	// Mock the email service with a no-op implementation
	mockEmailService := &mockEmailService{}
	s.AuthController.SetEmailService(mockEmailService)

	// Configure test-specific render functions to simulate the real behavior
	s.AuthController.RenderRegister = func(c *gin.Context, data interface{}) {
		// In a real app, this would render a template, but for tests just simulate it
		// Since the test should hit the POST path, adding a redirect here
		c.Redirect(http.StatusSeeOther, "/")
	}

	s.AuthController.RenderVerificationSent = func(c *gin.Context, data interface{}) {
		// In a real app, this would render a template, but for tests just simulate it
		c.Redirect(http.StatusSeeOther, "/verification-sent")
	}

	// Set up router
	s.Router = gin.New()

	// Create test promotion
	now := time.Now()
	s.ActivePromotion = &models.Promotion{
		Model:         gorm.Model{ID: 1},
		Name:          "Test Promotion",
		Type:          "free_trial",
		Active:        true,
		StartDate:     now.AddDate(0, 0, -1), // Started yesterday
		EndDate:       now.AddDate(0, 0, 5),  // Ends in 5 days
		BenefitDays:   30,
		DisplayOnHome: true,
		Description:   "Test promotion description",
		Banner:        "/images/test-banner.jpg",
	}
}

// mockGormDB mocks a gorm DB that will be used for soft-delete checking
type mockGormDB struct {
	mock.Mock
}

// Where mocks GORM Where clause
func (m *mockGormDB) Where(query interface{}, args ...interface{}) *mockGormDB {
	m.Called(query, args)
	return m
}

// First mocks GORM First method
func (m *mockGormDB) First(dest interface{}, conds ...interface{}) *mockGormDB {
	m.Called(dest, conds)
	// Simulate "not found" for our tests
	return m
}

// Error property that will be checked
func (m *mockGormDB) Error() error {
	args := m.Called()
	return args.Error(0)
}

// mockEmailService provides a no-op implementation of the email service for testing
type mockEmailService struct{}

// SendVerificationEmail is a no-op implementation for testing
func (m *mockEmailService) SendVerificationEmail(email, token, baseURL string) error {
	return nil
}

// SendPasswordResetEmail is a no-op implementation for testing
func (m *mockEmailService) SendPasswordResetEmail(email, token, baseURL string) error {
	return nil
}

// SendContactFormEmail is a no-op implementation for testing
func (m *mockEmailService) SendContactFormEmail(name, email, subject, message string) error {
	return nil
}

// SendContactEmail is a no-op implementation for testing
func (m *mockEmailService) SendContactEmail(name, email, subject, message string) error {
	return nil
}

// SendEmailChangeVerification is a no-op implementation for testing
func (m *mockEmailService) SendEmailChangeVerification(email, token, baseURL string) error {
	return nil
}

// TestRegisterWithActivePromotion tests user registration when promotion is active
func (s *PromotionAuthTestSuite) TestRegisterWithActivePromotion() {
	// Set up the auth routes
	s.Router.POST("/register", s.AuthController.RegisterHandler)

	// Mock GetUserByEmail to indicate the user doesn't exist yet
	s.MockDB.On("GetUserByEmail", mock.Anything, "test@example.com").Return(nil, nil)

	// Skip unscoped check - we're setting up our controller not to use this in tests
	s.AuthController.SkipUnscopedChecks = true

	// Mock FindActivePromotions to return our active promotion
	s.MockDB.On("FindActivePromotions").Return([]models.Promotion{*s.ActivePromotion}, nil)

	// Mock CreateUser to simulate the user being created with context
	testUser := &database.User{
		Email: "test@example.com",
	}

	s.MockDB.On("CreateUser", mock.Anything, "test@example.com", mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			// Simulate successful user creation
			testUser.ID = 1
		}).
		Return(testUser, nil)

	// Mock updating user with verification token
	s.MockDB.On("UpdateUser", mock.Anything, mock.MatchedBy(func(u *database.User) bool {
		// This will match the first UpdateUser call to add the verification token
		return u.ID == testUser.ID && u.VerificationToken != ""
	})).Return(nil)

	// Mock to check if promotion benefits are applied to the user
	s.MockDB.On("UpdateUser", mock.Anything, mock.MatchedBy(func(u *database.User) bool {
		// Check that the user has the promotion benefit days applied
		expectedExpiry := time.Now().AddDate(0, 0, s.ActivePromotion.BenefitDays)
		expiryDiff := u.SubscriptionEndDate.Sub(expectedExpiry)

		// Allow a small time difference (1 minute) since there's processing time
		return u.ID == testUser.ID &&
			u.SubscriptionTier == "promotion" &&
			u.PromotionID == s.ActivePromotion.ID &&
			expiryDiff > -time.Minute && expiryDiff < time.Minute
	})).Return(nil)

	// Create form data
	form := url.Values{}
	form.Add("email", "test@example.com")
	form.Add("password", "password123")
	form.Add("password_confirm", "password123")

	// Create a test request
	req, _ := http.NewRequest("POST", "/register", bytes.NewBufferString(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Test", "true") // Set test header to trigger redirect to home
	resp := httptest.NewRecorder()

	// Make the request
	s.Router.ServeHTTP(resp, req)

	// Verify redirects to home page on success in test environment
	s.Equal(http.StatusSeeOther, resp.Code)
	s.Equal("/", resp.Header().Get("Location"))

	// Verify mocks were called as expected
	s.MockDB.AssertExpectations(s.T())
}

// TestRegisterWithNoActivePromotion tests user registration when no promotion is active
func (s *PromotionAuthTestSuite) TestRegisterWithNoActivePromotion() {
	// Set up the auth routes
	s.Router.POST("/register", s.AuthController.RegisterHandler)

	// Mock GetUserByEmail to indicate the user doesn't exist yet
	s.MockDB.On("GetUserByEmail", mock.Anything, "test@example.com").Return(nil, nil)

	// Skip unscoped check - we're setting up our controller not to use this in tests
	s.AuthController.SkipUnscopedChecks = true

	// Mock FindActivePromotions to return empty list (no active promotions)
	s.MockDB.On("FindActivePromotions").Return([]models.Promotion{}, nil)

	// Mock CreateUser to simulate the user being created with context
	testUser := &database.User{
		Email: "test@example.com",
	}

	s.MockDB.On("CreateUser", mock.Anything, "test@example.com", mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			// Simulate successful user creation
			testUser.ID = 1
		}).
		Return(testUser, nil)

	// Mock updating user with verification token
	s.MockDB.On("UpdateUser", mock.Anything, mock.MatchedBy(func(u *database.User) bool {
		// In this case, we only expect the verification token update
		return u.ID == testUser.ID && u.VerificationToken != ""
	})).Return(nil)

	// Create form data
	form := url.Values{}
	form.Add("email", "test@example.com")
	form.Add("password", "password123")
	form.Add("password_confirm", "password123")

	// Create a test request
	req, _ := http.NewRequest("POST", "/register", bytes.NewBufferString(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Test", "true") // Set test header to trigger redirect to home
	resp := httptest.NewRecorder()

	// Make the request
	s.Router.ServeHTTP(resp, req)

	// Verify redirects to home page on success in test environment
	s.Equal(http.StatusSeeOther, resp.Code)
	s.Equal("/", resp.Header().Get("Location"))

	// Verify mocks were called as expected
	s.MockDB.AssertExpectations(s.T())
}

// TestPromotionAuthSuite runs the test suite
func TestPromotionAuthSuite(t *testing.T) {
	suite.Run(t, new(PromotionAuthTestSuite))
}
