package newrelic

import (
	"fmt"
	"os"

	"github.com/newrelic/go-agent/v3/newrelic"
)

// InitializeNewRelic creates and configures a new New Relic application instance
func InitializeNewRelic() (*newrelic.Application, error) {
	appName := os.Getenv("NEW_RELIC_APP_NAME")
	licenseKey := os.Getenv("NEW_RELIC_LICENSE_KEY")

	if appName == "" || licenseKey == "" {
		return nil, fmt.Errorf("New Relic environment variables not properly configured: NEW_RELIC_APP_NAME and NEW_RELIC_LICENSE_KEY must be set")
	}

	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName(appName),
		newrelic.ConfigLicense(licenseKey),
	)
	if err != nil {
		return nil, err
	}

	return app, nil
}

// GetNewRelicTransaction creates a new transaction for the given name
func GetNewRelicTransaction(app *newrelic.Application, name string) *newrelic.Transaction {
	return app.StartTransaction(name)
}
