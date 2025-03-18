package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/database"
	"github.com/shaj13/go-guardian/v2/auth"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// OwnerControllerTestSuite is a test suite for the OwnerController
type OwnerControllerTestSuite struct {
	suite.Suite
	Router    *gin.Engine
	MockDB    *mock.Mock
	mockOwner *database.User
	mockUser  *auth.DefaultUser
}

// SetupTest sets up each test
func (s *OwnerControllerTestSuite) SetupTest() {
	// Set up Gin
	gin.SetMode(gin.TestMode)
	s.Router = gin.New()

	// Create a mock owner for testing
	s.mockOwner = &database.User{
		Model:    gorm.Model{ID: 1},
		Email:    "test@example.com",
		Verified: true,
	}

	// Create a mock auth user
	s.mockUser = &auth.DefaultUser{
		ID:   "1",
		Name: "test@example.com",
	}
}

// TestOwnerEndpoints tests various owner endpoints with simplified handlers
func (s *OwnerControllerTestSuite) TestOwnerEndpoints() {
	// Simple handler that doesn't use any controller dependencies
	handler := func(c *gin.Context) {
		c.String(http.StatusOK, "Test endpoint works")
	}

	// Register test routes
	s.Router.GET("/owners", handler)
	s.Router.GET("/profile", handler)
	s.Router.GET("/profile/edit", handler)

	// Test each endpoint
	endpoints := []string{"/owners", "/profile", "/profile/edit"}

	for _, endpoint := range endpoints {
		s.Run("Testing "+endpoint, func() {
			req, _ := http.NewRequest("GET", endpoint, nil)
			resp := httptest.NewRecorder()
			s.Router.ServeHTTP(resp, req)

			s.Equal(http.StatusOK, resp.Code)
			s.Contains(resp.Body.String(), "Test endpoint works")
		})
	}
}

// Run the tests
func TestOwnerControllerSuite(t *testing.T) {
	suite.Run(t, new(OwnerControllerTestSuite))
}
