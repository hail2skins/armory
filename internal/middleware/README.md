# Error Handling System

This package provides a comprehensive error handling system for Gin-based applications. It includes:

- Custom error types for different scenarios (validation, auth, not found, etc.)
- Structured logging of errors
- Error metrics collection
- HTML and JSON error responses
- Panic recovery

## Usage

### Basic Setup

To set up all error handling components at once:

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/hail2skins/armory/internal/middleware"
)

func main() {
    router := gin.Default()
    
    // Set up all error handling
    middleware.SetupErrorHandling(router)
    
    // ... add your routes
    
    router.Run(":8080")
}
```

### Using Error Types

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/hail2skins/armory/internal/errors"
)

func GetUser(c *gin.Context) {
    id := c.Param("id")
    
    user, err := userService.GetByID(id)
    if err != nil {
        if err == sql.ErrNoRows {
            // Return a 404 error
            c.Error(errors.NewNotFoundError("User not found"))
            return
        }
        // Return a 500 error
        c.Error(err)
        return
    }
    
    c.JSON(200, user)
}

func ValidateUser(c *gin.Context) {
    var user User
    if err := c.ShouldBindJSON(&user); err != nil {
        // Return a 400 error
        c.Error(errors.NewValidationError("Invalid user data"))
        return
    }
    
    // ... process the user
    
    c.JSON(200, gin.H{"status": "ok"})
}
```

### Direct Error Handling

You can also handle errors directly without using the middleware:

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/hail2skins/armory/internal/errors"
)

func GetUser(c *gin.Context) {
    id := c.Param("id")
    
    user, err := userService.GetByID(id)
    if err != nil {
        if err == sql.ErrNoRows {
            // Handle the error directly
            errors.HandleError(c, errors.NewNotFoundError("User not found"))
            return
        }
        errors.HandleError(c, err)
        return
    }
    
    c.JSON(200, user)
}
```

## Error Types

The system provides several error types:

- `ValidationError` - For input validation errors (400 Bad Request)
- `AuthError` - For authentication/authorization errors (401 Unauthorized)
- `NotFoundError` - For resource not found errors (404 Not Found)
- `PaymentError` - For payment processing errors (400 Bad Request)

You can also use standard Go errors, which will be treated as internal server errors (500 Internal Server Error).

## Response Format

JSON responses have the following format:

```json
{
  "code": 400,
  "message": "Invalid input",
  "id": "a1b2c3d4e5f6" // Only for 500 errors, for tracking
}
```

HTML responses will render the error message in a template, or fall back to a simple text response if the template is not available.

## Feature Flag Middleware

The feature flag system provides a way to control access to features based on user roles through feature flags.

### RequireFeature Middleware

This middleware checks if a user has access to a specific feature based on feature flags and their associated roles.

```go
// Example usage in route setup
ownerGroup.Group("/ammo").Use(middleware.RequireFeature("ammo_features")).
    GET("", ammoController.Index)
```

If a user doesn't have access, they will be redirected to the dashboard with a flash message.

### CheckFeatureAccessForTemplates Middleware

This middleware adds feature access flags to the context for all registered feature flags, to be used in templates.

```go
// Example usage in route setup
router.Use(middleware.CheckFeatureAccessForTemplates())
```

In templates, you can then check if a user has access to a feature using:

```html
<!-- In templ templates -->
if HasFeatureAccess(ctx, "ammo_features") {
  <!-- Show ammo feature UI elements -->
}
```

Admin users automatically have access to all features. 