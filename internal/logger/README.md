# Logger Package

This package provides a structured logging system for Go applications. It includes:

- Different log levels (DEBUG, INFO, WARN, ERROR)
- Structured JSON logging
- Support for additional fields
- File logging

## Log Levels

The package defines several log levels:

- `DEBUG` - For detailed debugging information
- `INFO` - For general information
- `WARN` - For warnings
- `ERROR` - For errors

## Usage

### Basic Logging

```go
// Log a debug message
logger.Debug("Debug message", nil)

// Log an info message
logger.Info("Info message", nil)

// Log a warning message
logger.Warn("Warning message", nil)

// Log an error message
logger.Error("Error message", err, nil)
```

### Logging with Additional Fields

```go
// Log with additional fields
logger.Info("User logged in", map[string]interface{}{
    "user_id": 123,
    "path": "/login",
    "trace_id": "abc123",
})

// Log an error with additional fields
logger.Error("Database error", err, map[string]interface{}{
    "user_id": 123,
    "query": "SELECT * FROM users",
})
```

### File Logging

```go
// Set up logging to a file
err := logger.SetupFileLogging("/path/to/log/file.log")
if err != nil {
    // Handle error
}

// Reset logging to stdout
logger.ResetLogging()
```

## Log Format

Logs are output in JSON format:

```json
{
  "timestamp": "2025-03-14T19:45:08.854506621-05:00",
  "level": "ERROR",
  "message": "Database error",
  "error": "sql: no rows in result set",
  "user_id": 123,
  "path": "/users/456",
  "trace_id": "abc123",
  "fields": {
    "query": "SELECT * FROM users WHERE id = 456"
  }
}
```

## Integration

This package is designed to work with the `errors` and `middleware` packages for comprehensive error handling and logging in Gin applications. 