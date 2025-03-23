package integration

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
)

// LoginUser is a helper method to login a test user and return the session cookies
func (s *IntegrationSuite) LoginUser(email, password string) []*http.Cookie {
	return s.LoginUserWithRedirect(email, password, true)
}

// LoginUserWithRedirect is a helper method to login a test user and optionally follow the redirect
func (s *IntegrationSuite) LoginUserWithRedirect(email, password string, followRedirect bool) []*http.Cookie {
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

	// Get cookies from response
	cookies := resp.Result().Cookies()

	if followRedirect {
		// Follow redirect to /owner to ensure session is properly initialized
		ownerReq, _ := http.NewRequest("GET", "/owner", nil)
		for _, cookie := range cookies {
			ownerReq.AddCookie(cookie)
		}
		ownerResp := httptest.NewRecorder()
		s.Router.ServeHTTP(ownerResp, ownerReq)

		// Return all cookies for subsequent requests
		return ownerResp.Result().Cookies()
	}

	return cookies
}

// MakeAuthenticatedRequest is a helper to make a request with authentication cookies
func (s *IntegrationSuite) MakeAuthenticatedRequest(method, path string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	return s.MakeAuthenticatedRequestWithRedirect(method, path, cookies, true)
}

// MakeAuthenticatedRequestWithRedirect is a helper to make a request with authentication cookies
// and control whether to follow redirects
func (s *IntegrationSuite) MakeAuthenticatedRequestWithRedirect(method, path string, cookies []*http.Cookie, followRedirects bool) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, nil)

	// Add all cookies to the request
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	// Make the request
	resp := httptest.NewRecorder()
	s.Router.ServeHTTP(resp, req)

	// If followRedirects is true and this is a redirect response, follow it once to get the final response
	if followRedirects && (resp.Code == http.StatusSeeOther || resp.Code == http.StatusFound) {
		redirectURL := resp.Header().Get("Location")
		if redirectURL != "" {
			redirectReq, _ := http.NewRequest("GET", redirectURL, nil)
			// Add cookies to redirect request
			for _, cookie := range cookies {
				redirectReq.AddCookie(cookie)
			}
			redirectResp := httptest.NewRecorder()
			s.Router.ServeHTTP(redirectResp, redirectReq)
			return redirectResp
		}
	}

	return resp
}

// VerifySessionAuth verifies that a request has valid session authentication
func (s *IntegrationSuite) VerifySessionAuth(cookies []*http.Cookie, expectedEmail string) {
	// Make a request to check session data
	checkReq, _ := http.NewRequest("GET", "/", nil)
	for _, cookie := range cookies {
		checkReq.AddCookie(cookie)
	}
	checkResp := httptest.NewRecorder()
	s.Router.ServeHTTP(checkResp, checkReq)

	// Get the response body as a string
	body := checkResp.Body.String()

	// Verify that the navigation bar shows authenticated state
	s.NotContains(body, "Login")
	s.NotContains(body, "Register")
	s.Contains(body, "My Armory")
	s.Contains(body, "Logout")
}
