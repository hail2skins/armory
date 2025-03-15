# Error Handling and Logging System

This project includes a comprehensive error handling and logging system for Gin-based applications. The system is composed of three main packages:

1. `internal/errors` - Custom error types and error handling functions
2. `internal/logger` - Structured logging system
3. `internal/middleware` - Gin middleware for error handling

## Features

- Custom error types for different scenarios (validation, auth, not found, etc.)
- Structured JSON logging with different log levels
- Error metrics collection
- HTML and JSON error responses
- Panic recovery
- Consistent error handling across the application

## Architecture

The system follows these principles:

1. **Separation of Concerns**: Each package has a specific responsibility
2. **Consistent Error Handling**: All errors are handled in a consistent way
3. **Structured Logging**: All logs are structured for easy parsing and analysis
4. **Metrics Collection**: Error metrics are collected for monitoring

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

### Error Handling in Controllers

```go
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
```

### Logging

```go
// Log an info message
logger.Info("User logged in", map[string]interface{}{
    "user_id": 123,
    "path": "/login",
})

// Log an error with additional fields
logger.Error("Database error", err, map[string]interface{}{
    "user_id": 123,
    "query": "SELECT * FROM users",
})
```

## Package Documentation

For more detailed documentation, see the README files in each package:

- [errors/README.md](internal/errors/README.md)
- [logger/README.md](internal/logger/README.md)
- [middleware/README.md](internal/middleware/README.md)

## Example

See the [example application](internal/middleware/example/main.go) for a complete example of how to use the error handling and logging system. 