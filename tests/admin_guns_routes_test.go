package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// AdminGunsRoutesSuite is a test suite for admin guns routes
type AdminGunsRoutesSuite struct {
	ControllerTestSuite
}

// SetupTest sets up each test
func (s *AdminGunsRoutesSuite) SetupTest() {
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
func (s *AdminGunsRoutesSuite) createMockAdminUser() *mock.Mock {
	mockUser := new(mock.Mock)

	// Set up methods for MockAuthInfo interface
	mockUser.On("GetUserName").Return("admin@example.com")
	mockUser.On("GetGroups").Return([]string{"admin"})

	return mockUser
}

// CreateAdminGunsController creates and returns an AdminGunsController
func (s *AdminGunsRoutesSuite) CreateAdminGunsController() *controller.AdminGunsController {
	if ctl, ok := s.Controllers["adminGuns"]; ok {
		return ctl.(*controller.AdminGunsController)
	}

	adminGunsController := controller.NewAdminGunsController(s.MockDB)
	s.Controllers["adminGuns"] = adminGunsController
	return adminGunsController
}

// TestGunsIndexPage tests the guns index page
func (s *AdminGunsRoutesSuite) TestGunsIndexPage() {
	// Create the controller
	controller := s.CreateAdminGunsController()

	// Set up the route
	s.Router.GET("/admin/guns", controller.Index)

	// Create test guns and users
	user1 := &database.User{
		Model: gorm.Model{ID: 1},
		Email: "user1@example.com",
	}

	user2 := &database.User{
		Model: gorm.Model{ID: 2},
		Email: "user2@example.com",
	}

	guns := []models.Gun{
		{
			Model:          gorm.Model{ID: 1},
			Name:           "Glock 19",
			SerialNumber:   "ABC123",
			Purpose:        "EDC",
			WeaponTypeID:   1,
			CaliberID:      1,
			ManufacturerID: 1,
			OwnerID:        1,
			Acquired:       func() *time.Time { t := time.Now().AddDate(0, -6, 0); return &t }(),
		},
		{
			Model:          gorm.Model{ID: 2},
			Name:           "AR-15",
			SerialNumber:   "DEF456",
			Purpose:        "Home Defense",
			WeaponTypeID:   2,
			CaliberID:      2,
			ManufacturerID: 2,
			OwnerID:        1,
			Acquired:       func() *time.Time { t := time.Now().AddDate(0, -3, 0); return &t }(),
		},
		{
			Model:          gorm.Model{ID: 3},
			Name:           "Remington 870",
			SerialNumber:   "GHI789",
			Purpose:        "Hunting",
			WeaponTypeID:   3,
			CaliberID:      3,
			ManufacturerID: 3,
			OwnerID:        2,
			Acquired:       func() *time.Time { t := time.Now().AddDate(0, -1, 0); return &t }(),
		},
	}

	// Mock database calls
	s.MockDB.On("FindAllUsers").Return([]database.User{*user1, *user2}, nil)
	s.MockDB.On("FindAllGuns").Return(guns, nil)
	s.MockDB.On("CountGunsByUser", uint(1)).Return(int64(2), nil)
	s.MockDB.On("CountGunsByUser", uint(2)).Return(int64(1), nil)
	s.MockDB.On("FindAllManufacturers").Return([]models.Manufacturer{
		{Model: gorm.Model{ID: 1}, Name: "Glock"},
		{Model: gorm.Model{ID: 2}, Name: "Colt"},
		{Model: gorm.Model{ID: 3}, Name: "Remington"},
	}, nil)
	s.MockDB.On("FindAllCalibersByIDs", mock.Anything).Return([]models.Caliber{
		{Model: gorm.Model{ID: 1}, Caliber: "9mm"},
		{Model: gorm.Model{ID: 2}, Caliber: "5.56mm"},
		{Model: gorm.Model{ID: 3}, Caliber: "12 Gauge"},
	}, nil)
	s.MockDB.On("FindAllWeaponTypesByIDs", mock.Anything).Return([]models.WeaponType{
		{Model: gorm.Model{ID: 1}, Type: "Handgun"},
		{Model: gorm.Model{ID: 2}, Type: "Rifle"},
		{Model: gorm.Model{ID: 3}, Type: "Shotgun"},
	}, nil)

	// Create a request to the endpoint
	req, _ := http.NewRequest("GET", "/admin/guns", nil)
	w := httptest.NewRecorder()

	// Serve the request
	s.Router.ServeHTTP(w, req)

	// Assert the response
	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.Contains(s.T(), w.Body.String(), "Guns Tracked")
	assert.Contains(s.T(), w.Body.String(), "user1@example.com")
	assert.Contains(s.T(), w.Body.String(), "Glock 19")
	assert.Contains(s.T(), w.Body.String(), "AR-15")
	assert.Contains(s.T(), w.Body.String(), "user2@example.com")
	assert.Contains(s.T(), w.Body.String(), "Remington 870")
}

// TestAdminGunsRoutesSuite runs the test suite
func TestAdminGunsRoutesSuite(t *testing.T) {
	suite.Run(t, new(AdminGunsRoutesSuite))
}
