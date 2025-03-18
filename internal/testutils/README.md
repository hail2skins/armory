# Test Utilities for Armory

This package provides utilities for testing the Armory application, including mocks, test helpers, and utilities for controller testing.

## Components

### 1. AuthService Interface

The `AuthService` interface provides an abstraction for authentication-related operations, making it easier to mock auth functionality during testing. It helps decouple the auth controller from the rest of the codebase, making controllers more testable.

To use it in your application:

1. **Implement the interface** in your auth controller:

```go
// In internal/controller/auth.go

// Ensure AuthController implements the AuthService interface
var _ controller.AuthService = (*AuthController)(nil)

// Implement any missing methods required by the interface
func (a *AuthController) IsAuthenticated(c *gin.Context) bool {
    return a.isAuthenticated(c)
}

// Other implementations...
```

2. **Use the interface in your controllers**:

```go
// In other controllers

type YourController struct {
    DB          database.Service
    AuthService controller.AuthService // Use the interface instead of concrete type
}

func NewYourController(db database.Service, auth controller.AuthService) *YourController {
    return &YourController{
        DB:          db,
        AuthService: auth,
    }
}

func (y *YourController) SomeProtectedHandler(c *gin.Context) {
    // Now you can use the auth service interface methods
    if !y.AuthService.IsAuthenticated(c) {
        c.Redirect(http.StatusFound, "/login")
        return
    }
    
    // Rest of your handler...
}
```

3. **In middleware**, set the auth service using the interface:

```go
// In your middleware or router setup
router.Use(func(c *gin.Context) {
    c.Set("auth", authService) // Using interface, not concrete implementation
    c.Next()
})
```

4. **For testing**, use the mock implementation:

```go
// In your tests

// Create mock auth service
mockAuth := &MockAuthService{
    authenticated: false,
    email: "test@example.com",
}

// Create controller with the mock
controller := &YourController{
    DB: dbService,
    AuthService: mockAuth,
}

// Test your controller...
```

### 2. Server Testing with MockAuthService

When testing server routes that depend on authentication, use the MockAuthService in your test files:

```go
// In your server test file
func TestRouteWithAuth(t *testing.T) {
    router := gin.Default()
    mockAuth := &MockAuthService{
        authenticated: true,
        email: "test@example.com", 
    }
    
    // Set the auth service in middleware
    router.Use(func(c *gin.Context) {
        c.Set("auth", mockAuth)
        c.Next()
    })
    
    // Setup routes and test
    // ...
}
```

**Important Note**: When running individual test files that use MockAuthService, you need to include the file that defines this mock:

```
go test ./internal/server/routes_test.go ./internal/server/your_test_file.go -v
```

This is because the mock definitions are shared across test files in the same package.

### 3. ControllerTestHelper

The `ControllerTestHelper` provides utilities for testing controllers, including:

- Setting up routers with common middleware
- Creating test users and fixtures
- Simplifying form submission and response checking
- Handling authentication for tests

For detailed usage, see the [testhelper README](./testhelper/README.md).

### 4. TestData

The `TestData` structure manages test fixtures, providing methods to create and manage test entities like users, guns, weapon types, etc.

```go
testData := testhelper.NewTestData(db, service)
user := testData.CreateTestUser(context.Background())
gun := testData.CreateTestGun(user)
// ... use fixtures in your tests
testData.CleanupTestData() // Clean up when done
```

## Best Practices

1. **Use interfaces** to decouple components and make them easier to test.
2. **Create test suites** for more complex controllers using testify/suite.
3. **Clean up your test data** after tests to avoid test interference.
4. **Prefer controller with dependency injection** over global variables or singletons.
5. **Mock external services** (like email) for testing by using setter methods:
   ```go
   // In your test
   mockEmailService := &MockEmailService{}
   controller.SetEmailService(mockEmailService)
   ```

## Examples

See the [example tests](./example_test.go) for patterns on how to use these utilities in your tests. 