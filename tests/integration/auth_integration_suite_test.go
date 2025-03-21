package integration

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/testutils"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthIntegrationSuite provides a test suite for auth-related integration tests
type AuthIntegrationSuite struct {
	suite.Suite
	DB              *gorm.DB
	Service         database.Service
	Router          *gin.Engine
	AuthController  *controller.AuthController
	OwnerController *controller.OwnerController
}

// SetupSuite runs once before all tests in the suite
func (s *AuthIntegrationSuite) SetupSuite() {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a real test database service
	s.Service = testutils.SharedTestService()
	s.DB = s.Service.GetDB()

	// Create controllers with real DB
	s.AuthController = controller.NewAuthController(s.Service)
	s.OwnerController = controller.NewOwnerController(s.Service)

	// Set up router
	s.Router = gin.New()
	s.setupRoutes()
}

// setupRoutes configures all the routes needed for testing
func (s *AuthIntegrationSuite) setupRoutes() {
	// Add middleware to set required context values
	s.Router.Use(func(c *gin.Context) {
		c.Set("auth", s.AuthController)
		c.Set("authController", s.AuthController) // Required by LandingPage
		c.Next()
	})

	// Set up flash middleware
	s.Router.Use(func(c *gin.Context) {
		c.Set("setFlash", func(msg string) {
			c.SetCookie("flash", msg, 3600, "/", "", false, false)
		})

		// Get flash message from cookie
		if flash, err := c.Cookie("flash"); err == nil && flash != "" {
			c.Set("flash", flash)
			// Clear the cookie
			c.SetCookie("flash", "", -1, "/", "", false, false)
		}

		c.Next()
	})

	// Auth routes
	s.Router.GET("/login", s.AuthController.LoginHandler)
	s.Router.POST("/login", s.AuthController.LoginHandler)
	s.Router.GET("/logout", s.AuthController.LogoutHandler)

	// Owner routes
	s.Router.GET("/owner", s.OwnerController.LandingPage)
	s.Router.GET("/owner/guns/arsenal", s.OwnerController.Arsenal)
}

// TestLoginAndOwnerPage tests the complete login flow and owner page content
func (s *AuthIntegrationSuite) TestLoginAndOwnerPage() {
	// Create a test user with plain text password (BeforeCreate hook will hash it)
	testUser := &database.User{
		Email:            "test@example.com",
		Password:         "password123", // Plain password - will be hashed by BeforeCreate
		Verified:         true,
		SubscriptionTier: "free",
	}

	// Save the user directly to the database
	err := s.DB.Create(testUser).Error
	s.NoError(err)

	// Verify the user was saved correctly
	var checkUser database.User
	result := s.DB.Where("email = ?", testUser.Email).First(&checkUser)
	s.NoError(result.Error)

	// Test password verification works (should work now since model hashed it correctly)
	err = bcrypt.CompareHashAndPassword([]byte(checkUser.Password), []byte("password123"))
	s.NoError(err, "Password verification should succeed")

	// Test successful login using the actual login endpoint
	form := url.Values{}
	form.Add("email", "test@example.com")
	form.Add("password", "password123")

	req, _ := http.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()
	s.Router.ServeHTTP(resp, req)

	// Should redirect to owner page
	s.Equal(http.StatusSeeOther, resp.Code, "Login should redirect with status 303")
	s.Equal("/owner", resp.Header().Get("Location"), "Login should redirect to /owner")

	// Extract cookies for next request
	cookies := resp.Result().Cookies()

	// Now test the owner page content
	ownerReq, _ := http.NewRequest("GET", "/owner", nil)
	for _, cookie := range cookies {
		ownerReq.AddCookie(cookie)
	}
	ownerResp := httptest.NewRecorder()
	s.Router.ServeHTTP(ownerResp, ownerReq)

	// Check status code
	s.Equal(http.StatusOK, ownerResp.Code, "Owner page should return 200 OK")

	// Get the response body as a string
	body := ownerResp.Body.String()

	// Verify all the required elements
	s.Contains(body, "You haven't added any firearms yet", "Owner page should show empty state message")
	s.Contains(body, "Total Firearms:</strong> 0", "Owner page should show zero firearms count")
	s.Contains(body, "Total Paid:</strong> $0.00", "Owner page should show zero total paid")
	s.Contains(body, "No firearms added yet", "Owner page should indicate no guns in recently added")
	s.Contains(body, "Current Plan:</strong> Free", "Owner page should show Free plan")

	// Verify the Under Construction section
	s.Contains(body, "Under Construction", "Owner page should have ammo inventory marked as under construction")
	s.Contains(body, `href="#"`, "Under Construction link should point to #")

	// Verify the buttons
	s.Contains(body, `href="/owner/guns/arsenal"`, "Owner page should have View Arsenal button")
	s.Contains(body, "View Arsenal", "Owner page should have View Arsenal text")

	s.Contains(body, `href="/owner/guns/new"`, "Owner page should have Add New Firearm button")
	s.Contains(body, "Add New Firearm", "Owner page should have Add New Firearm text")

	s.Contains(body, "Add Your First Firearm", "Owner page should have Add Your First Firearm button")

	// Flash message should appear
	s.Contains(body, "Enjoy adding to your armory!", "Owner page should display the welcome flash message")

	// Now test clicking the "View Arsenal" button - user is taken to arsenal page
	arsenalReq, _ := http.NewRequest("GET", "/owner/guns/arsenal", nil)
	for _, cookie := range cookies {
		arsenalReq.AddCookie(cookie)
	}
	arsenalResp := httptest.NewRecorder()
	s.Router.ServeHTTP(arsenalResp, arsenalReq)

	// Check status code
	s.Equal(http.StatusOK, arsenalResp.Code, "Arsenal page should return 200 OK")

	// Get the response body as a string
	arsenalBody := arsenalResp.Body.String()

	// Verify the content for an empty arsenal
	s.Contains(arsenalBody, "No firearms found", "Arsenal page should show empty state message")

	// Verify navigation bar shows authenticated state
	s.Contains(arsenalBody, `href="/owner"`, "Arsenal page should have My Armory link")
	s.Contains(arsenalBody, `href="/logout"`, "Arsenal page should have Logout link")

	// Verify navigation bar does NOT show unauthenticated links
	s.NotContains(arsenalBody, `href="/login"`, "Arsenal page should not have Login link")
	s.NotContains(arsenalBody, `href="/register"`, "Arsenal page should not have Register link")

	// Clean up - using the DB directly since Service doesn't expose Delete
	result = s.DB.Unscoped().Delete(testUser)
	s.NoError(result.Error)
}

// TearDownSuite cleans up after all tests
func (s *AuthIntegrationSuite) TearDownSuite() {
	// Clean up database if needed
}

func TestAuthIntegrationSuite(t *testing.T) {
	suite.Run(t, new(AuthIntegrationSuite))
}
