package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// AdminPaymentRoutesSuite is a test suite for admin payment routes
type AdminPaymentRoutesSuite struct {
	ControllerTestSuite
}

// SetupTest sets up each test
func (s *AdminPaymentRoutesSuite) SetupTest() {
	// Run the parent's SetupTest
	s.ControllerTestSuite.SetupTest()

	// Set up mock authenticated state for admin access
	s.MockAuth.On("IsAuthenticated", mock.Anything).Return(true)

	// Set up GetCurrentUser to return admin user info
	s.MockAuth.On("GetCurrentUser", mock.Anything).Return(s.createMockAdminUser(), true)

	// Add middleware to set authData in context
	s.Router.Use(func(c *gin.Context) {
		// Create auth data for the context
		authData := data.NewAuthData()
		authData.Authenticated = true
		authData.Email = "admin@example.com"
		authData.Roles = []string{"admin"}
		authData.IsCasbinAdmin = true
		c.Set("authData", authData)
		c.Next()
	})
}

// Helper method to create a mock admin user
func (s *AdminPaymentRoutesSuite) createMockAdminUser() *mock.Mock {
	mockUser := new(mock.Mock)

	// Set up methods for MockAuthInfo interface
	mockUser.On("GetUserName").Return("admin@example.com")
	mockUser.On("GetGroups").Return([]string{"admin"})

	return mockUser
}

// CreateAdminPaymentController creates and returns an AdminPaymentController
func (s *AdminPaymentRoutesSuite) CreateAdminPaymentController() *controller.AdminPaymentController {
	if ctl, ok := s.Controllers["adminPayment"]; ok {
		return ctl.(*controller.AdminPaymentController)
	}

	adminPaymentController := controller.NewAdminPaymentController(s.MockDB)
	s.Controllers["adminPayment"] = adminPaymentController
	return adminPaymentController
}

// TestPaymentsHistoryPage tests the payments history page
func (s *AdminPaymentRoutesSuite) TestPaymentsHistoryPage() {
	// Create the controller
	controller := s.CreateAdminPaymentController()

	// Set up the route
	s.Router.GET("/admin/payments-history", controller.ShowPaymentsHistory)

	// Create test payments
	now := time.Now()
	payments := []models.Payment{
		{
			UserID:      1,
			Amount:      1999,
			Currency:    "usd",
			PaymentType: "subscription",
			Status:      "succeeded",
			Description: "Monthly Subscription",
			StripeID:    "pi_123456",
		},
		{
			UserID:      2,
			Amount:      9999,
			Currency:    "usd",
			PaymentType: "one-time",
			Status:      "succeeded",
			Description: "Lifetime Subscription",
			StripeID:    "pi_789012",
		},
	}

	// Set CreatedAt fields separately to avoid Model field directly
	payments[0].CreatedAt = now.Add(-24 * time.Hour)
	payments[0].ID = 1
	payments[1].CreatedAt = now
	payments[1].ID = 2

	// Mock the database call to return our test payments
	s.MockDB.On("GetAllPayments").Return(payments, nil)

	// Create a request to the endpoint
	req, _ := http.NewRequest("GET", "/admin/payments-history", nil)
	w := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(w, req)

	// Assert the response
	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.Contains(s.T(), w.Body.String(), "Payment History")
	assert.Contains(s.T(), w.Body.String(), "Monthly Subscription")
	assert.Contains(s.T(), w.Body.String(), "Lifetime Subscription")
}

// TestAdminPaymentRoutesSuite runs the test suite
func TestAdminPaymentRoutesSuite(t *testing.T) {
	suite.Run(t, new(AdminPaymentRoutesSuite))
}
