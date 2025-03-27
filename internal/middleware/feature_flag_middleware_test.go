package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAuthController for testing
type MockAuthController struct {
	mock.Mock
}

func (m *MockAuthController) IsAuthenticated(c *gin.Context) bool {
	args := m.Called(c)
	return args.Bool(0)
}

func (m *MockAuthController) IsAdmin(c *gin.Context) bool {
	args := m.Called(c)
	return args.Bool(0)
}

// MockUser for testing
type MockUser struct {
	mock.Mock
}

func (m *MockUser) GetUserName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockUser) GetGroups() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

// TestRequireFeature tests the RequireFeature middleware
func TestRequireFeature(t *testing.T) {
	// Save and restore original database.New
	originalNew := databaseNewFunc
	defer func() { databaseNewFunc = originalNew }()

	// Test cases
	tests := []struct {
		name           string
		setup          func(*MockAuthController, *MockUser, *mocks.MockDB) (*http.Request, *gin.Engine)
		featureName    string
		expectedStatus int
		expectedPath   string
	}{
		{
			name: "No auth controller",
			setup: func(authCtrl *MockAuthController, user *MockUser, db *mocks.MockDB) (*http.Request, *gin.Engine) {
				// Create a router with the middleware
				router := gin.New()
				router.GET("/test", RequireFeature("test_feature"), func(c *gin.Context) {
					c.Status(http.StatusOK)
				})
				// Create a request (no auth controller set)
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				return req, router
			},
			featureName:    "test_feature",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Not authenticated",
			setup: func(authCtrl *MockAuthController, user *MockUser, db *mocks.MockDB) (*http.Request, *gin.Engine) {
				// Set up the mock expectations
				authCtrl.On("IsAuthenticated", mock.Anything).Return(false)

				// Create a router with the middleware and a handler to set the auth controller
				router := gin.New()
				router.Use(func(c *gin.Context) {
					c.Set("authController", authCtrl)
					c.Next()
				})
				router.GET("/test", RequireFeature("test_feature"), func(c *gin.Context) {
					c.Status(http.StatusOK)
				})

				// Create a request
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				return req, router
			},
			featureName:    "test_feature",
			expectedStatus: http.StatusFound, // Redirect to login
			expectedPath:   "/login",
		},
		{
			name: "Admin user",
			setup: func(authCtrl *MockAuthController, user *MockUser, db *mocks.MockDB) (*http.Request, *gin.Engine) {
				// Set up the mock expectations
				authCtrl.On("IsAuthenticated", mock.Anything).Return(true)
				authCtrl.On("IsAdmin", mock.Anything).Return(true)

				// Create a router with the middleware and a handler to set the auth controller
				router := gin.New()
				router.Use(func(c *gin.Context) {
					c.Set("authController", authCtrl)
					c.Next()
				})
				router.GET("/test", RequireFeature("test_feature"), func(c *gin.Context) {
					c.Status(http.StatusOK)
				})

				// Create a request
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				return req, router
			},
			featureName:    "test_feature",
			expectedStatus: http.StatusOK,
		},
		{
			name: "No user in context",
			setup: func(authCtrl *MockAuthController, user *MockUser, db *mocks.MockDB) (*http.Request, *gin.Engine) {
				// Set up the mock expectations
				authCtrl.On("IsAuthenticated", mock.Anything).Return(true)
				authCtrl.On("IsAdmin", mock.Anything).Return(false)

				// Create a router with the middleware and a handler to set the auth controller
				router := gin.New()
				router.Use(func(c *gin.Context) {
					c.Set("authController", authCtrl)
					// Don't set the user
					c.Next()
				})
				router.GET("/test", RequireFeature("test_feature"), func(c *gin.Context) {
					c.Status(http.StatusOK)
				})

				// Create a request
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				return req, router
			},
			featureName:    "test_feature",
			expectedStatus: http.StatusFound, // Redirect to login
			expectedPath:   "/login",
		},
		{
			name: "User without access",
			setup: func(authCtrl *MockAuthController, user *MockUser, db *mocks.MockDB) (*http.Request, *gin.Engine) {
				// Set up the mock expectations
				authCtrl.On("IsAuthenticated", mock.Anything).Return(true)
				authCtrl.On("IsAdmin", mock.Anything).Return(false)
				user.On("GetUserName").Return("test@example.com")
				// No need to expect GetGroups since the middleware doesn't use it
				db.On("CanUserAccessFeature", "test@example.com", "test_feature").Return(false, nil)

				// Set the mock database
				databaseNewFunc = func() database.Service { return db }

				// Create a router with the middleware and a handler to set the auth controller and user
				router := gin.New()
				router.Use(func(c *gin.Context) {
					c.Set("authController", authCtrl)
					c.Set("auth", user)
					c.Next()
				})
				router.GET("/test", RequireFeature("test_feature"), func(c *gin.Context) {
					c.Status(http.StatusOK)
				})

				// Create a request
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				return req, router
			},
			featureName:    "test_feature",
			expectedStatus: http.StatusFound, // Redirect to dashboard
			expectedPath:   "/dashboard",
		},
		{
			name: "User with access",
			setup: func(authCtrl *MockAuthController, user *MockUser, db *mocks.MockDB) (*http.Request, *gin.Engine) {
				// Set up the mock expectations
				authCtrl.On("IsAuthenticated", mock.Anything).Return(true)
				authCtrl.On("IsAdmin", mock.Anything).Return(false)
				user.On("GetUserName").Return("test@example.com")
				// No need to expect GetGroups since the middleware doesn't use it
				db.On("CanUserAccessFeature", "test@example.com", "test_feature").Return(true, nil)

				// Set the mock database
				databaseNewFunc = func() database.Service { return db }

				// Create a router with the middleware and handlers
				router := gin.New()
				router.Use(func(c *gin.Context) {
					c.Set("authController", authCtrl)
					c.Set("auth", user)
					c.Next()
				})
				router.GET("/test", RequireFeature("test_feature"), func(c *gin.Context) {
					// Add flag to verify middleware set it
					hasAccess, exists := c.Get("has_test_feature_access")
					assert.True(t, exists)
					assert.True(t, hasAccess.(bool))
					c.Status(http.StatusOK)
				})

				// Create a request
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				return req, router
			},
			featureName:    "test_feature",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup Gin test mode
			gin.SetMode(gin.TestMode)

			// Create mocks
			authCtrl := new(MockAuthController)
			user := new(MockUser)
			db := new(mocks.MockDB)

			// Setup the test case
			req, router := tt.setup(authCtrl, user, db)

			// Perform the request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, w.Code,
				"Expected status %d, got %d", tt.expectedStatus, w.Code)

			if tt.expectedPath != "" {
				assert.Equal(t, tt.expectedPath, w.Header().Get("Location"),
					"Expected redirect to %s, got %s", tt.expectedPath, w.Header().Get("Location"))
			}

			// Verify mocks
			authCtrl.AssertExpectations(t)
			user.AssertExpectations(t)
			db.AssertExpectations(t)
		})
	}
}

// TestCheckFeatureAccessForTemplates tests the CheckFeatureAccessForTemplates middleware
func TestCheckFeatureAccessForTemplates(t *testing.T) {
	// Save and restore original database.New
	originalNew := databaseNewFunc
	defer func() { databaseNewFunc = originalNew }()

	// Test cases
	tests := []struct {
		name           string
		setup          func(*gin.Context) // Direct context setup for simplicity
		expectedAccess map[string]bool
	}{
		{
			name: "No auth controller",
			setup: func(c *gin.Context) {
				// Don't set auth controller
			},
			expectedAccess: nil,
		},
		{
			name: "Not authenticated",
			setup: func(c *gin.Context) {
				// Create auth controller mock directly in the test
				auth := &MockAuthController{}
				auth.On("IsAuthenticated", mock.Anything).Return(false)
				c.Set("authController", auth)
			},
			expectedAccess: nil,
		},
		{
			name: "Admin user",
			setup: func(c *gin.Context) {
				// Create auth controller mock directly in the test
				auth := &MockAuthController{}
				auth.On("IsAuthenticated", mock.Anything).Return(true)
				auth.On("IsAdmin", mock.Anything).Return(true).Once()
				c.Set("authController", auth)

				// Create user mock and add to context as "auth"
				userMock := &MockUser{}
				userMock.On("GetUserName").Return("admin").Once()
				c.Set("auth", userMock)

				// Create database mock that will be returned by our mock function
				mockDB := new(mocks.MockDB)
				mockDB.On("FindAllFeatureFlags").Return([]models.FeatureFlag{
					{ID: 1, Name: "feature1"},
					{ID: 2, Name: "feature2"},
				}, nil).Once()

				// Override databaseNewFunc during this test
				databaseNewFunc = func() database.Service {
					return mockDB
				}
			},
			expectedAccess: map[string]bool{
				"feature1": true,
				"feature2": true,
			},
		},
		{
			name: "Regular user with access to one feature",
			setup: func(c *gin.Context) {
				// Create auth controller mock
				auth := &MockAuthController{}
				auth.On("IsAuthenticated", mock.Anything).Return(true)
				auth.On("IsAdmin", mock.Anything).Return(false)
				c.Set("authController", auth)

				// Create user mock and add to context as "auth"
				userMock := &MockUser{}
				userMock.On("GetUserName").Return("regular").Once()
				c.Set("auth", userMock)

				// Create database mock
				mockDB := new(mocks.MockDB)
				mockDB.On("FindAllFeatureFlags").Return([]models.FeatureFlag{
					{ID: 1, Name: "feature1"},
					{ID: 2, Name: "feature2"},
				}, nil)

				// Mock database check for user permissions
				mockDB.On("CanUserAccessFeature", "regular", "feature1").Return(true, nil)
				mockDB.On("CanUserAccessFeature", "regular", "feature2").Return(false, nil)

				databaseNewFunc = func() database.Service {
					return mockDB
				}
			},
			expectedAccess: map[string]bool{
				"feature1": true,
				"feature2": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up Gin test mode
			gin.SetMode(gin.TestMode)

			// Create a request for the context
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Set up the test case by applying setup to the context
			tt.setup(c)

			// Run the middleware directly on our context
			middleware := CheckFeatureAccessForTemplates()
			middleware(c)

			// Check feature access in context
			if tt.expectedAccess == nil {
				// Expect no feature_access in context
				_, exists := c.Get("feature_access")
				assert.False(t, exists, "feature_access should not be set in context")
			} else {
				// Expect feature_access to match expected map
				access, exists := c.Get("feature_access")
				if !exists {
					// Print debug info when the feature_access is not set
					authCtl, hasAuth := c.Get("authController")
					if !hasAuth {
						t.Error("No authController in context")
					} else {
						t.Logf("authController type: %T", authCtl)

						// Check if IsAdmin is working
						if auth, ok := authCtl.(*MockAuthController); ok {
							t.Logf("IsAdmin returns: %v", auth.IsAdmin(c))
						}
					}
					t.Error("feature_access not set in context")
				} else {
					accessMap, ok := access.(map[string]bool)
					assert.True(t, ok, "feature_access should be map[string]bool")

					assert.Equal(t, tt.expectedAccess, accessMap, "feature_access values should match expected")
				}
			}
		})
	}
}

// TestHasFeatureAccess tests the HasFeatureAccess helper function
func TestHasFeatureAccess(t *testing.T) {
	// Create a gin context for testing
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Test cases
	tests := []struct {
		name         string
		featureName  string
		contextSetup func(*gin.Context)
		expected     bool
	}{
		{
			name:        "No feature_access in context",
			featureName: "test_feature",
			contextSetup: func(c *gin.Context) {
				// Don't set anything
			},
			expected: false,
		},
		{
			name:        "Invalid type in context",
			featureName: "test_feature",
			contextSetup: func(c *gin.Context) {
				c.Set("feature_access", "not a map")
			},
			expected: false,
		},
		{
			name:        "Feature not in map",
			featureName: "unknown_feature",
			contextSetup: func(c *gin.Context) {
				c.Set("feature_access", map[string]bool{
					"test_feature": true,
				})
			},
			expected: false,
		},
		{
			name:        "Feature in map but false",
			featureName: "test_feature",
			contextSetup: func(c *gin.Context) {
				c.Set("feature_access", map[string]bool{
					"test_feature": false,
				})
			},
			expected: false,
		},
		{
			name:        "Feature in map and true",
			featureName: "test_feature",
			contextSetup: func(c *gin.Context) {
				c.Set("feature_access", map[string]bool{
					"test_feature": true,
				})
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup context
			tt.contextSetup(c)

			// Test function
			result := HasFeatureAccess(c, tt.featureName)

			// Assertion
			assert.Equal(t, tt.expected, result)
		})
	}
}
