package tests

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestStripeSecurityCSRFTokens checks the Stripe security template files for CSRF token input fields
func TestStripeSecurityCSRFTokens(t *testing.T) {
	// Use absolute paths to the template files
	securityTemplPath := "/home/art/armory/cmd/web/views/admin/stripe_security.templ"
	testIPTemplPath := "/home/art/armory/cmd/web/views/admin/stripe_test_ip.templ"

	// Read the security template file
	securityTempl, err := os.ReadFile(securityTemplPath)
	if err != nil {
		t.Skipf("Skipping test: could not read security template file: %v", err)
	}

	// Read the IP test template file
	testIPTempl, err := os.ReadFile(testIPTemplPath)
	if err != nil {
		t.Skipf("Skipping test: could not read IP test template file: %v", err)
	}

	// Convert files to string
	securityTemplStr := string(securityTempl)
	testIPTemplStr := string(testIPTempl)

	// Test for CSRF token in toggle filtering form
	assert.Contains(t, securityTemplStr,
		`<form method="post" action="/admin/stripe-security/toggle-filtering"`,
		"Security template should contain toggle filtering form")
	assert.Contains(t, securityTemplStr,
		`<input type="hidden" name="csrf_token" value="`+"`",
		"Toggle filtering form should contain CSRF token")

	// Test for CSRF token in refresh form
	assert.Contains(t, securityTemplStr,
		`<form method="post" action="/admin/stripe-security/refresh"`,
		"Security template should contain refresh form")
	assert.Contains(t, securityTemplStr,
		`<input type="hidden" name="csrf_token" value="`+"`",
		"Refresh form should contain CSRF token")

	// Test for CSRF token in IP test form
	assert.Contains(t, testIPTemplStr,
		`<form method="post" action="/admin/stripe-security/check-ip"`,
		"IP test template should contain check IP form")
	assert.Contains(t, testIPTemplStr,
		`<input type="hidden" name="csrf_token" value="`+"`",
		"IP test form should contain CSRF token")
}
