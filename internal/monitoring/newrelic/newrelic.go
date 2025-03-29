package newrelic

import (
	"os"

	"github.com/newrelic/go-agent/v3/newrelic"
)

// InitializeNewRelic creates and configures a new New Relic application instance
func InitializeNewRelic() (*newrelic.Application, error) {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName(os.Getenv("NEW_RELIC_APP_NAME")),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
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
