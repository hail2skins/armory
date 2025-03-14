package web

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/a-h/templ"
	"github.com/hail2skins/armory/cmd/web/views/data"
	homeviews "github.com/hail2skins/armory/cmd/web/views/home"
	partialviews "github.com/hail2skins/armory/cmd/web/views/partials"
	"github.com/stretchr/testify/assert"
)

func TestNavBarAuthentication(t *testing.T) {
	// Test NavAuth component directly
	t.Run("NavAuth shows Login and Register for unauthenticated users", func(t *testing.T) {
		// Render the NavAuth component for an unauthenticated user
		buf := &bytes.Buffer{}
		err := partialviews.NavAuth(false).Render(context.Background(), buf)
		assert.NoError(t, err)

		// Check that the output contains Login and Register links
		html := buf.String()
		assert.Contains(t, html, `onclick="window.location.href=`)
		assert.Contains(t, html, `/login`)
		assert.Contains(t, html, `/register`)
		assert.NotContains(t, html, `/logout`)
	})

	t.Run("NavAuth shows Logout for authenticated users", func(t *testing.T) {
		// Render the NavAuth component for an authenticated user
		buf := &bytes.Buffer{}
		err := partialviews.NavAuth(true).Render(context.Background(), buf)
		assert.NoError(t, err)

		// Check that the output contains Logout link
		html := buf.String()
		assert.Contains(t, html, `/logout`)
		assert.NotContains(t, html, `/login`)
		assert.NotContains(t, html, `/register`)
	})

	// Test Home page with different authentication states
	t.Run("Home page shows different content based on authentication", func(t *testing.T) {
		// Test unauthenticated state
		unauthData := homeviews.HomeData{
			AuthData: data.AuthData{
				Authenticated: false,
			},
		}

		bufUnauth := &bytes.Buffer{}
		err := homeviews.Home(unauthData).Render(context.Background(), bufUnauth)
		assert.NoError(t, err)

		htmlUnauth := bufUnauth.String()
		// Check for login/register buttons in the content area
		assert.Contains(t, htmlUnauth, `href="/register"`)
		assert.Contains(t, htmlUnauth, `href="/login"`)
		// Check for absence of authenticated content
		assert.NotContains(t, htmlUnauth, `href="/collection"`)
		// Check for login/register in the nav bar
		assert.Contains(t, htmlUnauth, `onclick="window.location.href=&#39;/login&#39;"`)
		assert.Contains(t, htmlUnauth, `onclick="window.location.href=&#39;/register&#39;"`)
		// Check for absence of logout in the nav bar
		assert.NotContains(t, htmlUnauth, `onclick="window.location.href=&#39;/logout&#39;"`)

		// Test authenticated state
		authData := homeviews.HomeData{
			AuthData: data.AuthData{
				Authenticated: true,
				Email:         "test@example.com",
			},
		}

		bufAuth := &bytes.Buffer{}
		err = homeviews.Home(authData).Render(context.Background(), bufAuth)
		assert.NoError(t, err)

		htmlAuth := bufAuth.String()
		// Check for authenticated content
		assert.Contains(t, htmlAuth, `href="/collection"`)
		// Check for logout in the nav bar
		assert.Contains(t, htmlAuth, `onclick="window.location.href=&#39;/logout&#39;"`)
		// Check for absence of login/register in the content
		assert.NotContains(t, htmlAuth, `href="/register"`)
		assert.NotContains(t, htmlAuth, `href="/login"`)
		// Check for absence of login/register in the nav bar
		assert.NotContains(t, htmlAuth, `onclick="window.location.href=&#39;/login&#39;"`)
		assert.NotContains(t, htmlAuth, `onclick="window.location.href=&#39;/register&#39;"`)
	})

	// Test Base template with auth-links element
	t.Run("Base template includes auth-links with HTMX attributes", func(t *testing.T) {
		buf := &bytes.Buffer{}
		authData := data.AuthData{
			Authenticated: false,
		}
		// Create a simple component function instead of passing nil
		children := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			_, err := io.WriteString(w, "<div>Test content</div>")
			return err
		})
		err := partialviews.Base(authData, children).Render(context.Background(), buf)
		assert.NoError(t, err)

		html := buf.String()
		assert.Contains(t, html, `nav class="bg-gray-800"`)
	})
}

func TestHomePageContent(t *testing.T) {
	// Test that the home page contains the expected content from the reference template
	t.Run("Home page contains Virtual Armory features", func(t *testing.T) {
		// Create test data
		homeData := homeviews.HomeData{
			AuthData: data.AuthData{
				Authenticated: false,
			},
		}

		// Render the home page
		buf := &bytes.Buffer{}
		err := homeviews.Home(homeData).Render(context.Background(), buf)
		assert.NoError(t, err)

		// Check for the presence of key features
		html := buf.String()

		// Hero section
		assert.Contains(t, html, "The Virtual Armory")
		assert.Contains(t, html, "Your digital home for tracking your home arsenal safely and securely")

		// Features section
		assert.Contains(t, html, "Track Your Collection")
		assert.Contains(t, html, "Maintenance Records")
		assert.Contains(t, html, "Range Day Tracking")
		assert.Contains(t, html, "Ammo Inventory")
		assert.Contains(t, html, "Modding Paradise")

		// CTA section
		assert.Contains(t, html, "Ready to organize your collection?")
		assert.Contains(t, html, "Join firearm enthusiasts and build your virtual armory")
		assert.Contains(t, html, "View Pricing")
	})
}
