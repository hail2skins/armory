# New Relic Integration

This package provides integration with New Relic for APM (Application Performance Monitoring) and logging in The Virtual Armory.

## Features

- APM (Application Performance Monitoring)
- Transaction tracking
- Log forwarding with transaction correlation
- Custom attribute support

## Configuration

The integration requires the following environment variables:

- `NEW_RELIC_LICENSE_KEY`: Your New Relic license key
- `NEW_RELIC_APP_NAME`: The name of your application in New Relic

## Components

### 1. APM Integration

The APM integration is initialized in `server.New()` and provides:
- Automatic transaction tracking
- Performance monitoring
- Error tracking

### 2. Logging Integration

The logging integration uses New Relic's `logWriter` to:
- Decorate logs with APM context
- Associate logs with transactions
- Forward logs to New Relic

## Usage

### Transaction Tracking

```go
// In your Gin handlers, logs will automatically be associated with the request transaction
func YourHandler(c *gin.Context) {
    reqLogger := logger.GetContextLogger(c)
    reqLogger.Println("Processing request")
}

// For background tasks or custom transactions
func YourFunction(txn *newrelic.Transaction) {
    txnLogger := logger.GetTransactionLogger(txn)
    txnLogger.Println("Processing task")
}
```

### Logging

The standard logger functions will automatically include New Relic context:
```go
logger.Info("Something happened", map[string]interface{}{
    "user_id": userId,
    "action": "login",
})
```

## Removal Instructions

To remove New Relic integration:

1. Remove Environment Variables:
   - Remove `NEW_RELIC_LICENSE_KEY`
   - Remove `NEW_RELIC_APP_NAME`

2. Remove Code:
   ```bash
   # Remove the New Relic package
   rm -rf internal/monitoring/newrelic

   # Remove dependencies
   go mod tidy
   ```

3. Code Changes Required:
   - In `internal/server/server.go`:
     - Remove the New Relic import
     - Remove the `newRelicApp` field from Server struct
     - Remove New Relic initialization in `New()`
     - Remove New Relic middleware from `RegisterMiddleware()`

   - In `internal/logger/logger.go`:
     - Remove New Relic imports
     - Remove `nrWriter` variable
     - Remove `ConfigureNewRelic` function
     - Remove `GetContextLogger` and `GetTransactionLogger` functions
     - Modify `writeLog` to only use standard logger

4. Update Dependencies:
   ```bash
   # Remove New Relic dependencies
   go get -u github.com/newrelic/go-agent/v3@none
   go get -u github.com/newrelic/go-agent/v3/integrations/nrgin@none
   go get -u github.com/newrelic/go-agent/v3/integrations/logcontext-v2/logWriter@none
   go mod tidy
   ```

## Dependencies

- github.com/newrelic/go-agent/v3
- github.com/newrelic/go-agent/v3/integrations/nrgin
- github.com/newrelic/go-agent/v3/integrations/logcontext-v2/logWriter

## Notes

- The integration is designed to work with our custom logger package
- Log levels are automatically mapped to New Relic severity levels
- All logs maintain their JSON format for consistency
- Transaction context is automatically added when available 