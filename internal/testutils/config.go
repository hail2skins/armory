package testutils

import (
    "os"
    "path/filepath"
)

// TestConfig holds test configuration
type TestConfig struct {
    DBPath string
    AssetsPath string
    // Add other test configuration
}

// LoadTestConfig loads test configuration
func LoadTestConfig() *TestConfig {
    return &TestConfig{
        DBPath: filepath.Join(os.TempDir(), "test.db"),
        AssetsPath: "./testdata",
    }
}