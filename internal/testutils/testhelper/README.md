# Test Helper Utilities

This package provides utilities to simplify and standardize testing controllers and authentication flows in the Armory application.

## Overview

The test helper utilities include:

1. **AuthService Interface**: Abstracts authentication-related operations for testing
2. **ControllerTestHelper**: Provides utilities for testing controllers including router setup, form submission, etc.
3. **TestData**: Manages test fixtures and provides helper methods for creating test data

## Using the Test Helpers

### Basic Controller Test Example

```go
func TestMyController(t *testing.T) {
    // Create a test database
    db, _ := database.NewTestDB()
    service := database.NewTestService(db)
    
    // Create the test helper
    helper := testhelper.NewControllerTestHelper(db, service)
    
    // Create a test user
    testUser := helper.CreateTestUser(t)
    
    // Set up your controller with dependencies
    controller := &YourController{DB: service}
    
    // Get a router with authentication middleware
    router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)
    
    // Register your routes
    router.GET("/protected", controller.ProtectedHandler)
    
    // Make a request and check the response
    w := helper.AssertViewRendered(
        t,
        router,
        http.MethodGet,
        "/protected",
        nil,
        http.StatusOK,
    )
    
    // Check the response body
    assert.Contains(t, w.Body.String(), "Expected Content")
    
    // Clean up test data
    helper.CleanupTest()
}
```

### Testing Form Submission

```go
func TestFormSubmission(t *testing.T) {
    // Set up test helpers
    db, _ := database.NewTestDB()
    service := database.NewTestService(db)
    helper := testhelper.NewControllerTestHelper(db, service)
    
    // Set up your controller with dependencies
    controller := &YourController{DB: service}
    
    // Get a router for the test
    router := helper.GetUnauthenticatedRouter()
    
    // Register your routes
    router.POST("/login", controller.LoginHandler)
    
    // Create form values
    form := url.Values{}
    form.Add("email", "test@example.com")
    form.Add("password", "Password123!")
    
    // Submit the form and check the response
    w := helper.SubmitForm(
        t,
        router,
        http.MethodPost,
        "/login",
        form,
        http.StatusSeeOther, // 303 redirect expected
        "/dashboard", // Expected redirect location
    )
    
    // Check for flash messages or other side effects if needed
    // Clean up
    helper.CleanupTest()
}
```

### Using TestData for Fixtures

```go
func TestWithFixtures(t *testing.T) {
    // Set up test database and service
    db, _ := database.NewTestDB()
    service := database.NewTestService(db)
    
    // Create TestData directly if you don't need the full helper
    testData := testhelper.NewTestData(db, service)
    
    // Create standard test fixtures
    testUser := testData.CreateTestUser(context.Background())
    testGun := testData.CreateTestGun(testUser)
    
    // Run your tests with the fixtures
    
    // Clean up when done
    testData.CleanupTestData()
}
```

## Test Suite Pattern

For more complex tests, consider using the test suite pattern:

```go
type MyControllerSuite struct {
    suite.Suite
    DB       *gorm.DB
    Service  database.Service
    Helper   *testhelper.ControllerTestHelper
    TestUser *database.User
}

func (s *MyControllerSuite) SetupSuite() {
    // Set up test database
    db, err := database.NewTestDB()
    s.Require().NoError(err)
    s.DB = db
    
    // Set up service
    s.Service = database.NewTestService(db)
    
    // Set up helper
    s.Helper = testhelper.NewControllerTestHelper(db, s.Service)
}

func (s *MyControllerSuite) SetupTest() {
    // Set up fixtures for each test
    s.TestUser = s.Helper.CreateTestUser(s.T())
}

func (s *MyControllerSuite) TearDownTest() {
    // Clean up after each test
    s.Helper.CleanupTest()
}

func (s *MyControllerSuite) TestSomething() {
    // Your test code here
}

func TestMyControllerSuite(t *testing.T) {
    suite.Run(t, new(MyControllerSuite))
}
```

## Auth Service Interface

The `AuthService` interface provides a standardized way to mock authentication for tests. It includes methods for:

- Authenticating users
- Getting the current user from context
- Checking if a request is authenticated
- Setting up authentication middleware for testing
- Creating pre-authenticated routers for testing protected routes

This allows controllers to be tested in isolation without needing a full implementation of the auth controller. 