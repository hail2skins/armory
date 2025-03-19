package testutils

import (
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/mock"
)

// MockAuthController is a mock of AuthController
type MockAuthController struct {
	mock.Mock
}

// GetCurrentUser mocks the GetCurrentUser method
func (m *MockAuthController) GetCurrentUser(c *gin.Context) (interface{}, bool) {
	args := m.Called(c)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0), args.Bool(1)
}

// IsAuthenticated mocks the IsAuthenticated method
func (m *MockAuthController) IsAuthenticated(c *gin.Context) bool {
	return m.Called(c).Bool(0)
}

// NewMockTestSetup creates a new test setup with mocks
func NewMockTestSetup() (*gin.Engine, *mocks.MockDB, *MockAuthController) {
	// Create router
	router := gin.New()

	// Create mock database
	mockDB := new(mocks.MockDB)

	// Create mock auth controller
	mockAuth := new(MockAuthController)

	// Set up middleware to inject auth controller
	router.Use(func(c *gin.Context) {
		c.Set("auth", mockAuth)
		c.Set("authController", mockAuth)
		c.Next()
	})

	// Return router and mocks
	return router, mockDB, mockAuth
}

// SetupTestUser creates a test user
func SetupTestUser() *database.User {
	user := &database.User{
		Email:    "test@example.com",
		Password: "password",
	}
	// Set the ID using gorm.Model embedded struct
	user.Model.ID = 1
	return user
}

// SetupMockUser sets up a mock user in the database and auth controller
func SetupMockUser(mockDB *mocks.MockDB, mockAuth *MockAuthController, user *database.User) {
	// Set up mock DB to return the user when queried by email
	mockDB.On("GetUserByEmail", mock.Anything, user.Email).Return(user, nil)
	mockDB.On("GetUserByID", user.ID).Return(user, nil)

	// Set up mock auth controller to recognize the user as authenticated
	mockUserInfo := &mocks.MockAuthInfo{}
	mockUserInfo.SetUserName(user.Email)
	mockUserInfo.SetID("1") // Default ID
	mockAuth.On("GetCurrentUser", mock.Anything).Return(mockUserInfo, true)
	mockAuth.On("IsAuthenticated", mock.Anything).Return(true)
}

// CreateMockControllers creates controllers with the mock database
func CreateMockControllers(mockDB *mocks.MockDB) (*controller.AuthController, *controller.OwnerController) {
	authController := controller.NewAuthController(mockDB)
	ownerController := controller.NewOwnerController(mockDB)
	return authController, ownerController
}
