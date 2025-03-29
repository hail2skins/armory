package newrelic

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRelicInitialization(t *testing.T) {
	// Set up test environment variables with a 40-character license key
	os.Setenv("NEW_RELIC_LICENSE_KEY", "0123456789012345678901234567890123456789")
	os.Setenv("NEW_RELIC_APP_NAME", "test-app")
	defer os.Unsetenv("NEW_RELIC_LICENSE_KEY")
	defer os.Unsetenv("NEW_RELIC_APP_NAME")

	// Test initialization
	app, err := InitializeNewRelic()
	assert.NoError(t, err)
	assert.NotNil(t, app)

	// Test transaction creation
	txn := app.StartTransaction("test-transaction")
	assert.NotNil(t, txn)
	txn.End()
}
