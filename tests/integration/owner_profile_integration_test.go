package integration

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/hail2skins/armory/internal/database"
	"github.com/stretchr/testify/suite"
)

// OwnerProfileIntegrationTest extends the base IntegrationSuite for testing profile functionalities
type OwnerProfileIntegrationTest struct {
	IntegrationSuite
	testUser *database.User
}

// SetupTest runs before each test in the suite
func (s *OwnerProfileIntegrationTest) SetupTest() {
	// Call the parent SetupTest to set up the suite
	s.IntegrationSuite.SetupTest()

	// Create a test user for our tests
	s.testUser = s.CreateTestUser("profileowner@example.com", "Password123!", true)
}

// TearDownTest runs after each test in the suite
func (s *OwnerProfileIntegrationTest) TearDownTest() {
	// Clean up test user
	s.CleanupTestUser(s.testUser)

	// Call the parent TearDownTest
	s.IntegrationSuite.TearDownTest()
}

// TestProfilePage tests that a logged-in user can see the profile page
// and it displays all the required elements
func (s *OwnerProfileIntegrationTest) TestProfilePage() {
	// Login the user using the shared helper
	cookies := s.LoginUser(s.testUser.Email, "Password123!")

	// Visit the owner page to verify the Profile link exists
	ownerResp := s.MakeAuthenticatedRequest("GET", "/owner", cookies)
	s.Equal(http.StatusOK, ownerResp.Code)

	// Verify the Profile link is present on the owner page
	ownerBody := ownerResp.Body.String()
	s.Contains(ownerBody, "Profile")
	s.Contains(ownerBody, "href=\"/owner/profile\"")

	// Visit the profile page with the session cookies
	profileResp := s.MakeAuthenticatedRequest("GET", "/owner/profile", cookies)
	s.Equal(http.StatusOK, profileResp.Code)

	// Get the response body as a string
	profileBody := profileResp.Body.String()

	// Verify page title
	s.Contains(profileBody, "Your Profile")

	// Verify back to dashboard link
	s.Contains(profileBody, "Back to Dashboard")
	s.Contains(profileBody, "href=\"/owner\"")

	// Verify Account Information section
	s.Contains(profileBody, "Account Information")

	// Verify user email is displayed
	s.Contains(profileBody, s.testUser.Email)

	// Verify Subscription section
	s.Contains(profileBody, "Subscription")

	// Verify Account Management section
	s.Contains(profileBody, "Account Management")

	// Verify Edit Profile link
	s.Contains(profileBody, "Edit Profile")
	s.Contains(profileBody, "href=\"/owner/profile/edit\"")

	// Verify Manage Subscription link
	s.Contains(profileBody, "Manage Subscription")
	s.Contains(profileBody, "href=\"/owner/profile/subscription\"")

	// Verify account deletion section text
	s.Contains(profileBody, "Need to take a break? You can delete your account. Come back any time.")

	// Verify Delete Account link
	s.Contains(profileBody, "Delete Account")
	s.Contains(profileBody, "href=\"/owner/profile/delete\"")
}

// TestPaymentHistoryPage tests that a logged-in user can see the payment history page
// and it displays the expected elements (including the missing Back to Dashboard link)
func (s *OwnerProfileIntegrationTest) TestPaymentHistoryPage() {
	// Login the user using the shared helper
	cookies := s.LoginUser(s.testUser.Email, "Password123!")

	// Visit the payment history page
	paymentHistoryResp := s.MakeAuthenticatedRequest("GET", "/owner/payment-history", cookies)
	s.Equal(http.StatusOK, paymentHistoryResp.Code)

	// Get the response body as a string
	paymentHistoryBody := paymentHistoryResp.Body.String()

	// Verify page title
	s.Contains(paymentHistoryBody, "Payment History")

	// Verify empty state message
	s.Contains(paymentHistoryBody, "You don't have any payments yet")

	// This will fail because the Back to Dashboard link doesn't exist on this page
	s.Contains(paymentHistoryBody, "Back to Dashboard")
	s.Contains(paymentHistoryBody, "href=\"/owner\"")
}

// TestSubscriptionPage tests that a logged-in user can see the subscription management page
// and it displays all the expected elements
func (s *OwnerProfileIntegrationTest) TestSubscriptionPage() {
	// Login the user using the shared helper
	cookies := s.LoginUser(s.testUser.Email, "Password123!")

	// Visit the subscription page
	subscriptionResp := s.MakeAuthenticatedRequest("GET", "/owner/profile/subscription", cookies)
	s.Equal(http.StatusOK, subscriptionResp.Code)

	// Get the response body as a string
	subscriptionBody := subscriptionResp.Body.String()

	// Verify page title
	s.Contains(subscriptionBody, "Subscription Management")

	// Verify back to profile link
	s.Contains(subscriptionBody, "Back to My Profile")
	s.Contains(subscriptionBody, "href=\"/owner/profile\"")

	// Verify Current Plan section
	s.Contains(subscriptionBody, "Current Plan")

	// Verify Free plan for new users
	s.Contains(subscriptionBody, "Free")

	// Verify Upgrade Plan button
	s.Contains(subscriptionBody, "Upgrade Plan")
	s.Contains(subscriptionBody, "href=\"/pricing\"")

	// Verify Payment History section
	s.Contains(subscriptionBody, "Payment History")

	// Verify empty payment history message
	s.Contains(subscriptionBody, "No payment history available")
}

// TestDeleteAccountFlow tests the account deletion confirmation page and the actual deletion process
func (s *OwnerProfileIntegrationTest) TestDeleteAccountFlow() {
	// 1. Login
	cookies := s.LoginUser(s.testUser.Email, "Password123!")

	// 2. Click the profile link from /owner page
	profileReq, _ := http.NewRequest("GET", "/owner/profile", nil)
	for _, cookie := range cookies {
		profileReq.AddCookie(cookie)
	}
	profileResp := httptest.NewRecorder()
	s.Router.ServeHTTP(profileResp, profileReq)
	s.Equal(http.StatusOK, profileResp.Code)

	// 3. Click delete account link from profile page
	deleteReq, _ := http.NewRequest("GET", "/owner/profile/delete", nil)
	for _, cookie := range cookies {
		deleteReq.AddCookie(cookie)
	}
	deleteResp := httptest.NewRecorder()
	s.Router.ServeHTTP(deleteResp, deleteReq)
	s.Equal(http.StatusOK, deleteResp.Code)

	// Verify delete confirmation page content
	deleteBody := deleteResp.Body.String()
	s.Contains(deleteBody, "Delete Account")
	s.Contains(deleteBody, "Are you sure you want to delete your account?")
	s.Contains(deleteBody, "This will remove your access to the Virtual Armory")
	s.Contains(deleteBody, "All your data will be retained")
	s.Contains(deleteBody, "sign up again with the same email address")
	s.Contains(deleteBody, "No, Keep My Account")
	s.Contains(deleteBody, "Yes, Delete My Account")

	// 4. Click "Yes, Delete My Account" button
	form := url.Values{}
	form.Add("confirm", "true")
	confirmReq, _ := http.NewRequest("POST", "/owner/profile/delete", strings.NewReader(form.Encode()))
	confirmReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, cookie := range cookies {
		confirmReq.AddCookie(cookie)
	}
	confirmResp := httptest.NewRecorder()
	s.Router.ServeHTTP(confirmResp, confirmReq)

	// Should redirect to home with flash
	s.Equal(http.StatusSeeOther, confirmResp.Code)
	s.Equal("/", confirmResp.Header().Get("Location"))

	// Get the updated cookies after deletion
	deletionCookies := confirmResp.Result().Cookies()

	// 5. Try to access /owner page with the new cookies
	ownerReq, _ := http.NewRequest("GET", "/owner", nil)
	for _, cookie := range deletionCookies {
		ownerReq.AddCookie(cookie)
	}
	ownerResp := httptest.NewRecorder()
	s.Router.ServeHTTP(ownerResp, ownerReq)

	// Should be redirected to login
	s.Equal(http.StatusSeeOther, ownerResp.Code)
	s.Equal("/login", ownerResp.Header().Get("Location"))
}

// TestEditProfileWithEmailChange tests the email change functionality
// When a user changes their email, they should be redirected to the verification sent page
func (s *OwnerProfileIntegrationTest) TestEditProfileWithEmailChange() {
	// Login the user using the shared helper
	cookies := s.LoginUser(s.testUser.Email, "Password123!")

	// Step 1: Visit the profile edit page
	editProfileResp := s.MakeAuthenticatedRequest("GET", "/owner/profile/edit", cookies)
	s.Equal(http.StatusOK, editProfileResp.Code)

	// Get the response body as a string
	editProfileBody := editProfileResp.Body.String()

	// Verify we're on the edit profile page
	s.Contains(editProfileBody, "Edit Profile")
	s.Contains(editProfileBody, "form")
	s.Contains(editProfileBody, "email")
	s.Contains(editProfileBody, s.testUser.Email) // Current email should be displayed

	// Step 2: Submit the form with a new email address
	// Create form data with the new email
	form := url.Values{}
	form.Add("email", "newemail@example.com")

	// Create a POST request with the form data
	req, err := http.NewRequest("POST", "/owner/profile/update", strings.NewReader(form.Encode()))
	s.Require().NoError(err)

	// Set content type for form data
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Add auth cookies
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	// Send the request
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)
	updateResp := w.Result()

	// Step 3: Verify redirection to verification-sent page
	s.Equal(http.StatusSeeOther, updateResp.StatusCode)
	s.Equal("/verification-sent", updateResp.Header.Get("Location"))

	// That's it! We've verified the redirect, which was the goal of this test
}

// TestPasswordResetFlow tests that a user can access the password reset flow from the profile edit page
func (s *OwnerProfileIntegrationTest) TestPasswordResetFlow() {
	// Login the user using the shared helper
	cookies := s.LoginUser(s.testUser.Email, "Password123!")

	// Step 1: Visit the profile edit page
	editProfileResp := s.MakeAuthenticatedRequest("GET", "/owner/profile/edit", cookies)
	s.Equal(http.StatusOK, editProfileResp.Code)

	// Verify the profile edit page contains a link to reset password
	editProfileBody := editProfileResp.Body.String()
	s.Contains(editProfileBody, "Reset Password")
	s.Contains(editProfileBody, "href=\"/reset-password/new\"")

	// Step 2: Visit the reset password page directly
	resetPasswordReq, err := http.NewRequest("GET", "/reset-password/new", nil)
	s.Require().NoError(err)

	// Add auth cookies
	for _, cookie := range cookies {
		resetPasswordReq.AddCookie(cookie)
	}

	resetPasswordW := httptest.NewRecorder()
	s.Router.ServeHTTP(resetPasswordW, resetPasswordReq)
	resetPasswordResp := resetPasswordW.Result()

	// Should get a 200 OK status code
	s.Equal(http.StatusOK, resetPasswordResp.StatusCode)

	// Read the response body
	resetPasswordBody, err := io.ReadAll(resetPasswordResp.Body)
	s.Require().NoError(err)
	resetPasswordBodyStr := string(resetPasswordBody)

	// Verify the reset password page contains the expected elements
	// Title
	s.Contains(resetPasswordBodyStr, "Reset your password")

	// Instructions
	s.Contains(resetPasswordBodyStr, "Enter your email address and we'll send you a link to reset your password")

	// Email form field
	s.Contains(resetPasswordBodyStr, "<input")
	s.Contains(resetPasswordBodyStr, "type=\"email\"")
	s.Contains(resetPasswordBodyStr, "name=\"email\"")

	// Expiration notice
	s.Contains(resetPasswordBodyStr, "Note:")
	s.Contains(resetPasswordBodyStr, "password reset link will expire in 60 minutes")

	// Back to login link
	s.Contains(resetPasswordBodyStr, "Back to login")
	s.Contains(resetPasswordBodyStr, "href=\"/login\"")

	// Submit button
	s.Contains(resetPasswordBodyStr, "Send reset link")
}

// MakeRequestWithCookie is a helper to make HTTP requests with a specific cookie
func (s *OwnerProfileIntegrationTest) MakeRequestWithCookie(method, path string, body io.Reader, cookie *http.Cookie) *httptest.ResponseRecorder {
	req, err := http.NewRequest(method, path, body)
	s.Require().NoError(err)

	if cookie != nil {
		req.AddCookie(cookie)
	}

	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)
	return w
}

// TestOwnerProfileIntegration runs the owner profile integration test suite
func TestOwnerProfileIntegration(t *testing.T) {
	suite.Run(t, new(OwnerProfileIntegrationTest))
}
