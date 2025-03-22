package integration

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
)

// LoginUser is a helper method to login a test user and return the session cookies
func (s *IntegrationSuite) LoginUser(email, password string) []*http.Cookie {
	// Create login form data with user credentials
	form := url.Values{}
	form.Add("email", email)
	form.Add("password", password)

	// Submit login request
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()
	s.Router.ServeHTTP(resp, req)

	// Verify login was successful (redirects to /owner)
	s.Equal(http.StatusSeeOther, resp.Code)
	s.Equal("/owner", resp.Header().Get("Location"))

	// Return cookies for subsequent requests
	return resp.Result().Cookies()
}

// MakeAuthenticatedRequest is a helper to make a request with authentication cookies
func (s *IntegrationSuite) MakeAuthenticatedRequest(method, path string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, nil)

	// Add authentication cookies to the request
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	// Make the request
	resp := httptest.NewRecorder()
	s.Router.ServeHTTP(resp, req)

	return resp
}
