package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hail2skins/armory/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPageTitles(t *testing.T) {
	// Create a new database instance for testing
	// IMPORTANT: Use SharedTestService to avoid repeatedly seeding the database
	// The shared database is seeded only once and reused across tests
	db := testutils.SharedTestService()
	defer db.Close() // This is a no-op for shared service

	// Create a new server
	s := &Server{
		port: 8080,
		db:   db,
	}

	// Get the router
	router := s.RegisterRoutes()

	// Define test cases
	testCases := []struct {
		name      string
		path      string
		wantCode  int
		wantTitle string
	}{
		{
			name:      "Home page",
			path:      "/",
			wantCode:  http.StatusOK,
			wantTitle: "Home | The Virtual Armory",
		},
		{
			name:      "Login page",
			path:      "/login",
			wantCode:  http.StatusOK,
			wantTitle: "Login | The Virtual Armory",
		},
		{
			name:      "Register page",
			path:      "/register",
			wantCode:  http.StatusOK,
			wantTitle: "Register | The Virtual Armory",
		},
		{
			name:      "Logout page",
			path:      "/logout",
			wantCode:  http.StatusSeeOther, // Redirects to home page
			wantTitle: "",                  // No title since it redirects
		},
	}

	// Run the tests
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a request
			req, err := http.NewRequest("GET", tc.path, nil)
			require.NoError(t, err)

			// Create a response recorder
			resp := httptest.NewRecorder()

			// Serve the request
			router.ServeHTTP(resp, req)

			// Check the status code
			assert.Equal(t, tc.wantCode, resp.Code)

			// For redirects, we don't check the title
			if resp.Code == http.StatusSeeOther {
				// Verify the redirect location
				location := resp.Header().Get("Location")
				assert.Equal(t, "/", location, "Redirect should go to home page")
				return
			}

			// Check the title
			body := resp.Body.String()
			titleTag := extractTitle(body)
			assert.Contains(t, titleTag, tc.wantTitle, "Page title should contain the expected title")
		})
	}
}

// extractTitle extracts the title tag from an HTML string
func extractTitle(html string) string {
	titleStart := strings.Index(html, "<title>")
	if titleStart == -1 {
		return ""
	}
	titleStart += 7 // Length of "<title>"

	titleEnd := strings.Index(html[titleStart:], "</title>")
	if titleEnd == -1 {
		return ""
	}

	return html[titleStart : titleStart+titleEnd]
}
