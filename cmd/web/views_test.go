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

func TestBaseTemplateAndNavigation(t *testing.T) {
	t.Run("Base template includes navigation with correct structure", func(t *testing.T) {
		buf := &bytes.Buffer{}
		authData := data.AuthData{
			Authenticated: false,
		}
		// Create a simple component function
		children := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			_, err := io.WriteString(w, "<div>Test content</div>")
			return err
		})
		err := partialviews.Base(authData, children).Render(context.Background(), buf)
		assert.NoError(t, err)

		html := buf.String()
		// Check navigation structure
		assert.Contains(t, html, `nav id="header"`)
		assert.Contains(t, html, `class="fixed w-full z-30 top-0 text-white bg-gunmetal-800"`)
		// Check for navigation links
		assert.Contains(t, html, `href="/"`)
		assert.Contains(t, html, `href="/login"`)
		assert.Contains(t, html, `href="/register"`)
		assert.NotContains(t, html, `href="/logout"`)
	})

	t.Run("Base template shows correct navigation for authenticated users", func(t *testing.T) {
		buf := &bytes.Buffer{}
		authData := data.AuthData{
			Authenticated: true,
			Email:         "test@example.com",
		}
		children := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			_, err := io.WriteString(w, "<div>Test content</div>")
			return err
		})
		err := partialviews.Base(authData, children).Render(context.Background(), buf)
		assert.NoError(t, err)

		html := buf.String()
		// Check navigation structure
		assert.Contains(t, html, `nav id="header"`)
		assert.Contains(t, html, `class="fixed w-full z-30 top-0 text-white bg-gunmetal-800"`)
		// Check for navigation links
		assert.Contains(t, html, `href="/"`)
		assert.Contains(t, html, `href="/logout"`)
		assert.NotContains(t, html, `href="/login"`)
		assert.NotContains(t, html, `href="/register"`)
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

func TestAboutPageContent(t *testing.T) {
	// Test that the about page exists and contains the expected content
	t.Run("About page has correct title and content", func(t *testing.T) {
		// Create test data
		authData := data.AuthData{
			Authenticated: false,
		}

		// Create AboutData with the AuthData
		aboutData := homeviews.AboutData{
			AuthData: authData,
		}

		// Render the about page
		buf := &bytes.Buffer{}
		err := homeviews.About(aboutData).Render(context.Background(), buf)
		assert.NoError(t, err)

		// Check for the presence of key content
		html := buf.String()

		// Title should include "About"
		assert.Contains(t, html, "About")

		// Check for required content
		assert.Contains(t, html, "Privacy & Security")
		assert.Contains(t, html, "Comprehensive Tracking")
		assert.Contains(t, html, "Future Proof")

		// Should NOT contain "Join thousands"
		assert.NotContains(t, html, "Join thousands")
	})
}
