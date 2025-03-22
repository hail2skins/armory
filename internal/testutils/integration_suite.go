package testutils

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/database"
	"github.com/stretchr/testify/suite"
)

// IntegrationTestSuite is a test suite for integration tests
type IntegrationTestSuite struct {
	suite.Suite
	DB      database.Service
	Router  *gin.Engine
	Server  *httptest.Server
	Cookies []*http.Cookie
}

// SetupSuite sets up the test suite before running tests
func (s *IntegrationTestSuite) SetupSuite() {
	// Create a new test database using the shared service
	s.DB = SharedTestService()

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a new router
	s.Router = gin.New()

	// Create controllers
	authController := controller.NewAuthController(s.DB)

	// Set up middleware
	s.Router.Use(func(c *gin.Context) {
		c.Set("auth", authController)
		c.Set("authController", authController)
		c.Next()
	})

	// Set up routes
	s.Router.GET("/login", authController.LoginHandler)
	s.Router.POST("/login", authController.LoginHandler)
	s.Router.GET("/logout", authController.LogoutHandler)
	s.Router.GET("/owner", func(c *gin.Context) {
		c.String(http.StatusOK, "Owner page")
	})

	// Create test server
	s.Server = httptest.NewServer(s.Router)
}

// TearDownSuite cleans up after all tests have run
func (s *IntegrationTestSuite) TearDownSuite() {
	s.Server.Close()
	s.DB.Close()
}

// SaveCookies saves cookies from a response
func (s *IntegrationTestSuite) SaveCookies(resp *http.Response) {
	s.Cookies = resp.Cookies()
}

// ApplyCookies applies saved cookies to a request
func (s *IntegrationTestSuite) ApplyCookies(req *http.Request) {
	for _, cookie := range s.Cookies {
		req.AddCookie(cookie)
	}
}

// RunIntegrationSuite runs the integration test suite
func RunIntegrationSuite(t *testing.T, testSuite suite.TestingSuite) {
	suite.Run(t, testSuite)
}
