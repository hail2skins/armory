package logger

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

type LogLevel string

const (
	DEBUG LogLevel = "DEBUG"
	INFO  LogLevel = "INFO"
	WARN  LogLevel = "WARN"
	ERROR LogLevel = "ERROR"
)

type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Error     string                 `json:"error,omitempty"`
	UserID    uint                   `json:"user_id,omitempty"`
	Path      string                 `json:"path,omitempty"`
	TraceID   string                 `json:"trace_id,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// Debug logs a debug message
func Debug(msg string, fields map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     DEBUG,
		Message:   msg,
	}
	addFields(&entry, fields)
	writeLog(entry)
}

// Info logs an info message
func Info(msg string, fields map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     INFO,
		Message:   msg,
	}
	addFields(&entry, fields)
	writeLog(entry)
}

// Warn logs a warning message
func Warn(msg string, fields map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     WARN,
		Message:   msg,
	}
	addFields(&entry, fields)
	writeLog(entry)
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
	writeLog(entry)
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

// writeLog writes the log entry to the configured output
func writeLog(entry LogEntry) {
	// Convert to JSON
	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Error marshaling log entry: %v", err)
		return
	}

	// Write to stdout (can be replaced with file output or other destinations)
	os.Stdout.Write(jsonBytes)
	os.Stdout.Write([]byte("\n"))
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
