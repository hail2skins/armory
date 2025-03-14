package web

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNavBarAuthentication(t *testing.T) {
	// Test NavAuth component directly
	t.Run("NavAuth shows Login and Register for unauthenticated users", func(t *testing.T) {
		// Render the NavAuth component for an unauthenticated user
		buf := &bytes.Buffer{}
		err := NavAuth(false).Render(context.Background(), buf)
		assert.NoError(t, err)

		// Check that the output contains Login and Register links
		html := buf.String()
		assert.Contains(t, html, `href="/login"`)
		assert.Contains(t, html, `href="/register"`)
		assert.NotContains(t, html, `href="/logout"`)
	})

	t.Run("NavAuth shows Logout for authenticated users", func(t *testing.T) {
		// Render the NavAuth component for an authenticated user
		buf := &bytes.Buffer{}
		err := NavAuth(true).Render(context.Background(), buf)
		assert.NoError(t, err)

		// Check that the output contains Logout link
		html := buf.String()
		assert.Contains(t, html, `href="/logout"`)
		assert.NotContains(t, html, `href="/login"`)
		assert.NotContains(t, html, `href="/register"`)
	})

	// Test Home page with different authentication states
	t.Run("Home page shows different content based on authentication", func(t *testing.T) {
		// Test unauthenticated state
		unauthData := map[string]interface{}{
			"authenticated": false,
		}

		bufUnauth := &bytes.Buffer{}
		err := Home(unauthData).Render(context.Background(), bufUnauth)
		assert.NoError(t, err)

		htmlUnauth := bufUnauth.String()
		assert.Contains(t, htmlUnauth, "Login</a>")
		assert.Contains(t, htmlUnauth, "Register</a>")

		// Test authenticated state
		authData := map[string]interface{}{
			"authenticated": true,
			"email":         "test@example.com",
		}

		bufAuth := &bytes.Buffer{}
		err = Home(authData).Render(context.Background(), bufAuth)
		assert.NoError(t, err)

		htmlAuth := bufAuth.String()
		assert.Contains(t, htmlAuth, "test@example.com")
		assert.Contains(t, htmlAuth, "Logout</a>")
	})

	// Test Base template with auth-links element
	t.Run("Base template includes auth-links with HTMX attributes", func(t *testing.T) {
		buf := &bytes.Buffer{}
		err := Base().Render(context.Background(), buf)
		assert.NoError(t, err)

		html := buf.String()
		assert.Contains(t, html, `id="auth-links"`)
		assert.Contains(t, html, `hx-get="/auth-links"`)
		assert.Contains(t, html, `hx-trigger="load"`)
	})
}
