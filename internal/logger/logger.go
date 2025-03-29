package logger

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/newrelic/go-agent/v3/integrations/logcontext-v2/logWriter"
	"github.com/newrelic/go-agent/v3/newrelic"
)

type LogLevel string

const (
	DEBUG LogLevel = "DEBUG"
	INFO  LogLevel = "INFO"
	WARN  LogLevel = "WARN"
	ERROR LogLevel = "ERROR"
)

type Environment string

const (
	Development Environment = "development"
	Staging     Environment = "staging"
	Production  Environment = "production"
)

var (
	currentEnv     Environment = Development
	minLevel       LogLevel    = DEBUG
	botPatterns                = []string{"/wp-", ".php", "wlwmanifest", "wordpress", "/admin", "/wp-admin"}
	botAttemptsMap             = make(map[string]int)
	mapMutex       sync.RWMutex
	lastCleanup    = time.Now()
	nrWriter       *logWriter.LogWriter
	stdLogger      *log.Logger
)

// Configure sets up the logger configuration
func Configure(env Environment) {
	currentEnv = env

	// Set minimum log level based on environment
	switch env {
	case Production:
		minLevel = INFO
	case Staging:
		minLevel = DEBUG
	case Development:
		minLevel = DEBUG
	}

	// Initialize standard logger
	stdLogger = log.New(os.Stdout, "", 0)
}

// ConfigureNewRelic sets up New Relic logging
func ConfigureNewRelic(app *newrelic.Application) {
	if app != nil {
		// Create a base logWriter that writes to stdout
		writer := logWriter.New(os.Stdout, app)
		nrWriter = &writer
	}
}

type LogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       LogLevel               `json:"level"`
	Message     string                 `json:"message"`
	Error       string                 `json:"error,omitempty"`
	UserID      uint                   `json:"user_id,omitempty"`
	Path        string                 `json:"path,omitempty"`
	TraceID     string                 `json:"trace_id,omitempty"`
	Fields      map[string]interface{} `json:"attributes,omitempty"`
	LogType     string                 `json:"logtype,omitempty"`
	EntityName  string                 `json:"entity.name,omitempty"`
	ServiceName string                 `json:"service.name,omitempty"`
}

// shouldLog determines if the entry should be logged based on environment and content
func shouldLog(entry LogEntry) bool {
	// Always log ERROR and WARN
	if entry.Level == ERROR || entry.Level == WARN {
		return true
	}

	// Check minimum level
	switch entry.Level {
	case DEBUG:
		if minLevel != DEBUG {
			return false
		}
	case INFO:
		if minLevel == ERROR || minLevel == WARN {
			return false
		}
	}

	// In production, filter out common bot patterns and aggregate them
	if currentEnv == Production && entry.Path != "" {
		for _, pattern := range botPatterns {
			if strings.Contains(strings.ToLower(entry.Path), pattern) {
				mapMutex.Lock()
				defer mapMutex.Unlock()

				// Increment bot attempt counter
				botAttemptsMap[pattern]++

				// Log aggregated bot attempts every 100 attempts or every hour
				if time.Since(lastCleanup) > time.Hour {
					for pattern, count := range botAttemptsMap {
						if count > 0 {
							Info("Aggregated bot attempts", map[string]interface{}{
								"pattern": pattern,
								"count":   count,
								"period":  "1h",
							})
						}
					}
					botAttemptsMap = make(map[string]int)
					lastCleanup = time.Now()
				}

				return false
			}
		}
	}

	// In production, filter out template rendering debug logs
	if currentEnv == Production && entry.Level == DEBUG {
		debugMsgs := []string{
			"Rendering HTML template",
			"Rendering template",
			"Successfully rendered",
			"Template data",
			"Parsed gin.H data",
		}

		for _, msg := range debugMsgs {
			if strings.Contains(entry.Message, msg) {
				return false
			}
		}
	}

	return true
}

// Debug logs a debug message
func Debug(msg string, fields map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     DEBUG,
		Message:   msg,
	}
	addFields(&entry, fields)
	if shouldLog(entry) {
		writeLog(entry)
	}
}

// Info logs an info message
func Info(msg string, fields map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     INFO,
		Message:   msg,
	}
	addFields(&entry, fields)
	if shouldLog(entry) {
		writeLog(entry)
	}
}

// Warn logs a warning message
func Warn(msg string, fields map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     WARN,
		Message:   msg,
	}
	addFields(&entry, fields)
	if shouldLog(entry) {
		writeLog(entry)
	}
}

// Error logs an error message
func Error(msg string, err error, fields map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     ERROR,
		Message:   msg,
	}

	if err != nil {
		entry.Error = err.Error()
	}

	addFields(&entry, fields)
	if shouldLog(entry) {
		writeLog(entry)
	}
}

// addFields adds additional fields to the log entry
func addFields(entry *LogEntry, fields map[string]interface{}) {
	if fields == nil {
		return
	}

	for k, v := range fields {
		switch k {
		case "user_id":
			if userID, ok := v.(uint); ok {
				entry.UserID = userID
			}
		case "path":
			if path, ok := v.(string); ok {
				entry.Path = path
			}
		case "trace_id":
			if traceID, ok := v.(string); ok {
				entry.TraceID = traceID
			}
		default:
			// Store other fields in the Fields map
			if entry.Fields == nil {
				entry.Fields = make(map[string]interface{})
			}
			entry.Fields[k] = v
		}
	}
}

// writeLog writes a log entry
func writeLog(entry LogEntry) {
	// Set New Relic specific fields
	entry.LogType = "application"
	entry.EntityName = os.Getenv("NEW_RELIC_APP_NAME")
	entry.ServiceName = os.Getenv("NEW_RELIC_APP_NAME")

	// Map our log levels to New Relic's expected format
	switch entry.Level {
	case DEBUG:
		entry.Level = "DEBUG"
	case INFO:
		entry.Level = "INFO"
	case WARN:
		entry.Level = "WARNING"
	case ERROR:
		entry.Level = "ERROR"
	}

	// Convert to JSON
	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Error marshaling log entry: %v", err)
		return
	}

	// Get the appropriate writer
	var writer io.Writer = os.Stdout
	if nrWriter != nil {
		writer = nrWriter
	}

	// Write the log entry
	writer.Write(append(jsonBytes, '\n'))
}

// SetupFileLogging configures logging to a file
func SetupFileLogging(filePath string) error {
	// Create or open the log file
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	// Set the log output to the file
	log.SetOutput(file)

	return nil
}

// ResetLogging resets logging to stdout
func ResetLogging() {
	log.SetOutput(os.Stdout)
}

// GetContextLogger returns a new logger with the given gin context
func GetContextLogger(c *gin.Context) *log.Logger {
	if nrWriter == nil || c == nil {
		return stdLogger
	}

	// Get transaction from context
	txn := newrelic.FromContext(c.Request.Context())
	if txn != nil {
		// Create a new writer with the transaction
		txnWriter := nrWriter.WithTransaction(txn)
		return log.New(txnWriter, "", 0)
	}

	return stdLogger
}

// GetTransactionLogger returns a new logger with the given transaction
func GetTransactionLogger(txn *newrelic.Transaction) *log.Logger {
	if nrWriter == nil || txn == nil {
		return stdLogger
	}

	// Create a new writer with the transaction
	txnWriter := nrWriter.WithTransaction(txn)
	return log.New(txnWriter, "", 0)
}
