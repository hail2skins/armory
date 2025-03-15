# Error Handling Package

This package provides a comprehensive error handling system for Gin-based applications. It includes:

- Custom error types for different scenarios
- Structured error responses
- HTML and JSON response formats
- Panic recovery middleware

## Error Types

The package defines several error types:

- `ValidationError` - For input validation errors (400 Bad Request)
- `AuthError` - For authentication/authorization errors (401 Unauthorized)
- `NotFoundError` - For resource not found errors (404 Not Found)
- `PaymentError` - For payment processing errors (400 Bad Request)

Each error type implements the standard Go `error` interface and provides an additional `ErrorType()` method for metrics and logging.

## Usage

### Creating Errors

```go
// Create a validation error
err := errors.NewValidationError("Invalid input")

// Create an auth error
err := errors.NewAuthError("Unauthorized access")

// Create a not found error
err := errors.NewNotFoundError("User not found")

// Create a payment error with a code
err := errors.NewPaymentError("Payment failed", "CARD_DECLINED")
```

### Handling Errors

```go
// Handle an error directly
errors.HandleError(c, err)

// Or use the middleware (see the middleware package)
c.Error(err)
```

## Integration

This package is designed to work with the `middleware` package, which provides middleware for handling errors in Gin applications. See the `middleware` package for more details.

## Error Response Format

JSON responses have the following format:

```json
{
  "code": 400,
  "message": "Invalid input",
  "id": "a1b2c3d4e5f6" // Only for 500 errors, for tracking
}
```

HTML responses will render the error message in a template, or fall back to a simple text response if the template is not available. 