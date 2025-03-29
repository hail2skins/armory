package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	// Redirect log output for testing
	oldOutput := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Reset logWriter to nil for testing to use direct stdout
	originalNrWriter := nrWriter
	nrWriter = nil
	defer func() {
		nrWriter = originalNrWriter
	}()

	// Restore original output when done
	defer func() {
		os.Stdout = oldOutput
	}()

	// Test cases
	testCases := []struct {
		name     string
		logFunc  func()
		expected string // Changed from LogLevel to string to match our New Relic formatting
		message  string
	}{
		{
			name: "Debug log",
			logFunc: func() {
				Debug("Debug message", nil)
			},
			expected: "DEBUG",
			message:  "Debug message",
		},
		{
			name: "Info log",
			logFunc: func() {
				Info("Info message", nil)
			},
			expected: "INFO",
			message:  "Info message",
		},
		{
			name: "Warn log",
			logFunc: func() {
				Warn("Warning message", nil)
			},
			expected: "WARNING", // Changed from WARN to WARNING for New Relic format
			message:  "Warning message",
		},
		{
			name: "Error log",
			logFunc: func() {
				Error("Error message", errors.New("test error"), nil)
			},
			expected: "ERROR",
			message:  "Error message",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call the log function
			tc.logFunc()

			// Flush the writer
			w.Close()

			// Read the output
			var buf bytes.Buffer
			io.Copy(&buf, r)

			// Create a new pipe for the next test
			r, w, _ = os.Pipe()
			os.Stdout = w

			// Parse the JSON log entry
			var entry map[string]interface{}
			err := json.Unmarshal(buf.Bytes(), &entry)

			// Verify the log entry
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, entry["level"])
			assert.Equal(t, tc.message, entry["message"])

			// For error logs, verify the error message
			if tc.expected == "ERROR" {
				assert.Equal(t, "test error", entry["error"])
			}
		})
	}
}

func TestLoggerWithFields(t *testing.T) {
	// Redirect log output for testing
	oldOutput := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Reset logWriter to nil for testing to use direct stdout
	originalNrWriter := nrWriter
	nrWriter = nil
	defer func() {
		nrWriter = originalNrWriter
	}()

	// Restore original output when done
	defer func() {
		os.Stdout = oldOutput
	}()

	// Test with additional fields
	fields := map[string]interface{}{
		"user_id":  uint(123),
		"path":     "/api/test",
		"trace_id": "abc123",
	}

	// Log with fields
	Info("Info with fields", fields)

	// Flush the writer
	w.Close()

	// Read the output
	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Parse the JSON log entry
	var entry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &entry)

	// Verify the log entry
	assert.NoError(t, err)
	assert.Equal(t, "INFO", entry["level"])
	assert.Equal(t, "Info with fields", entry["message"])
	assert.Equal(t, float64(123), entry["user_id"]) // JSON numbers are float64
	assert.Equal(t, "/api/test", entry["path"])
	assert.Equal(t, "abc123", entry["trace_id"])

	// Check that logtype exists
	assert.Equal(t, "application", entry["logtype"])
}

func TestSetupFileLogging(t *testing.T) {
	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "log_test_*.log")
	assert.NoError(t, err)

	// Get the file name and close it (we'll open it through the logger)
	fileName := tmpFile.Name()
	tmpFile.Close()

	// Remove the file when done
	defer os.Remove(fileName)

	// Save the original stdout
	originalStdout := os.Stdout

	// Setup file logging
	err = SetupFileLogging(fileName)
	assert.NoError(t, err)

	// Log a message
	testMsg := "Test file logging"
	log.Println(testMsg)

	// Reset logging
	ResetLogging()

	// Restore stdout
	os.Stdout = originalStdout

	// Read the file content
	content, err := os.ReadFile(fileName)
	assert.NoError(t, err)

	// Verify the log contains our message
	assert.Contains(t, string(content), testMsg)
}

func TestSetupFileLoggingError(t *testing.T) {
	// Try to log to an invalid path
	err := SetupFileLogging("/invalid/path/that/does/not/exist/log.txt")
	assert.Error(t, err)
}
