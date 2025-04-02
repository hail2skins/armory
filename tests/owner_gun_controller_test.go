package tests

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/middleware"
	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/testutils"
	"github.com/hail2skins/armory/internal/testutils/testhelper"
	"github.com/shaj13/go-guardian/v2/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAuthControllerAdapter adapts the testhelper.MockAuthService to what the controller expects
type MockAuthControllerAdapter struct {
	mockService *testhelper.MockAuthService
	userID      uint
	userEmail   string
}

// GetCurrentUser implements the required interface method
func (m *MockAuthControllerAdapter) GetCurrentUser(c *gin.Context) (auth.Info, bool) {
	return &testhelper.MockAuthInfo{
		Username: m.userEmail,
		ID:       fmt.Sprintf("%d", m.userID),
	}, true
}

// TestNewGun tests the new gun form page
func TestNewGun(t *testing.T) {
	// Enable CSRF test mode
	middleware.EnableTestMode()
	defer middleware.DisableTestMode()

	// Setup test database and service
	db := testutils.NewTestDB()
	defer db.Close()
	service := testutils.NewTestService(db.DB)
	helper := testhelper.NewControllerTestHelper(db.DB, service)
	defer helper.CleanupTest()

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)

	// Create controller and setup the route
	controller := controller.NewOwnerController(service)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)
	router.GET("/owner/gun/new", controller.New)

	// Make request
	w := helper.AssertViewRendered(
		t,
		router,
		http.MethodGet,
		"/owner/gun/new",
		nil,
		http.StatusOK,
	)

	// Assert response
	assert.Contains(t, w.Body.String(), "Add New Firearm")
}

// TestGunCreate tests the creation of a gun
func TestGunCreate(t *testing.T) {
	// Enable CSRF test mode
	middleware.EnableTestMode()
	defer middleware.DisableTestMode()

	// Setup test database and service
	db := testutils.NewTestDB()
	defer db.Close()
	service := testutils.NewTestService(db.DB)
	helper := testhelper.NewControllerTestHelper(db.DB, service)
	defer helper.CleanupTest()

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)

	// Create some test data
	weaponType := models.WeaponType{Type: "Test Pistol Creation", Popularity: 1}
	err := service.CreateWeaponType(&weaponType)
	require.NoError(t, err)

	caliber := models.Caliber{Caliber: "Test 9mm Creation", Popularity: 1}
	err = service.CreateCaliber(&caliber)
	require.NoError(t, err)

	manufacturer := models.Manufacturer{Name: "Test Glock Creation", Country: "USA", Popularity: 1}
	err = service.CreateManufacturer(&manufacturer)
	require.NoError(t, err)

	// Define form data for gun creation
	formData := url.Values{}
	formData.Set("name", "Test Gun")
	formData.Set("serial_number", "TEST123")
	formData.Set("purpose", "Home Defense")
	formData.Set("weapon_type_id", fmt.Sprintf("%d", weaponType.ID))
	formData.Set("caliber_id", fmt.Sprintf("%d", caliber.ID))
	formData.Set("manufacturer_id", fmt.Sprintf("%d", manufacturer.ID))
	formData.Set("csrf_token", "test_token")

	// Create controller and setup the route
	gunController := controller.NewOwnerController(service)

	// Create a custom router with our adapter
	router := gin.New()
	router.Use(func(c *gin.Context) {
		// Set up the auth controller adapter
		mockAuthService := helper.AuthService.(*testhelper.MockAuthService)
		adapter := &MockAuthControllerAdapter{
			mockService: mockAuthService,
			userID:      testUser.ID,
			userEmail:   testUser.Email,
		}
		c.Set("authController", adapter)

		// Set authenticated flag
		c.Set("authenticated", true)

		// Set user data
		c.Set("user", gin.H{"id": testUser.ID, "email": testUser.Email})

		// Set CSRF token
		c.Set("csrf_token", "test_token")

		// Set flash function
		c.Set("setFlash", func(msg string) {
			// No-op for tests
		})

		// Set auth data for views
		c.Set("authData", gin.H{
			"IsAuthenticated": true,
			"UserEmail":       testUser.Email,
			"UserID":          testUser.ID,
			"CSRFToken":       "test_token",
		})

		c.Next()
	})

	// Add the gun controller route with the correct path
	router.POST("/owner/guns", gunController.Create)

	// Make request with error handling
	req, err := http.NewRequest("POST", "/owner/guns", strings.NewReader(formData.Encode()))
	require.NoError(t, err, "Failed to create request")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-CSRF-TEST-MODE", "1")
	rr := httptest.NewRecorder()

	// Serve the request through the router
	router.ServeHTTP(rr, req)

	// Verify redirection to owner dashboard
	assert.Equal(t, http.StatusSeeOther, rr.Code)
	assert.Equal(t, "/owner", rr.Header().Get("Location"))

	// Verify the gun was created in the database
	gunList := []models.Gun{}
	err = db.DB.Where("owner_id = ? AND name = ?", testUser.ID, "Test Gun").Find(&gunList).Error
	require.NoError(t, err, "Failed to query gun")

	// Verify the gun was created
	assert.Greater(t, len(gunList), 0, "No gun was created in the database")

	if len(gunList) > 0 {
		assert.Equal(t, "Test Gun", gunList[0].Name, "Gun name mismatch")
		assert.Equal(t, "TEST123", gunList[0].SerialNumber, "Serial number mismatch")
		assert.Equal(t, "Home Defense", gunList[0].Purpose, "Purpose mismatch")
		assert.Equal(t, weaponType.ID, gunList[0].WeaponTypeID, "Weapon type ID mismatch")
		assert.Equal(t, caliber.ID, gunList[0].CaliberID, "Caliber ID mismatch")
		assert.Equal(t, manufacturer.ID, gunList[0].ManufacturerID, "Manufacturer ID mismatch")
	}
}

// TestGunCreateValidation tests validation during gun creation
func TestGunCreateValidation(t *testing.T) {
	// Enable CSRF test mode
	middleware.EnableTestMode()
	defer middleware.DisableTestMode()

	// Setup test database
	db := testutils.NewTestDB()
	defer db.Close()

	// First, create valid weapon type, caliber, and manufacturer records
	weaponType := models.WeaponType{Type: "Test Type Validation", Popularity: 1}
	err := db.DB.Create(&weaponType).Error
	require.NoError(t, err, "Failed to create test weapon type")

	caliber := models.Caliber{Caliber: "Test Caliber Validation", Popularity: 1}
	err = db.DB.Create(&caliber).Error
	require.NoError(t, err, "Failed to create test caliber")

	manufacturer := models.Manufacturer{Name: "Test Manufacturer Validation", Country: "USA", Popularity: 1}
	err = db.DB.Create(&manufacturer).Error
	require.NoError(t, err, "Failed to create test manufacturer")

	// Test cases for validation failures
	testCases := []struct {
		name        string
		gun         *models.Gun
		expectedErr string
	}{
		{
			name: "Too Long Name",
			gun: &models.Gun{
				Name:           strings.Repeat("X", 101),
				SerialNumber:   "TEST123",
				WeaponTypeID:   weaponType.ID,
				CaliberID:      caliber.ID,
				ManufacturerID: manufacturer.ID,
				OwnerID:        1,
			},
			expectedErr: "exceeds maximum length",
		},
		{
			name: "Invalid Weapon Type",
			gun: &models.Gun{
				Name:           "Test Gun",
				SerialNumber:   "TEST123",
				WeaponTypeID:   999,
				CaliberID:      caliber.ID,
				ManufacturerID: manufacturer.ID,
				OwnerID:        1,
			},
			expectedErr: "weapon type",
		},
		{
			name: "Invalid Caliber",
			gun: &models.Gun{
				Name:           "Test Gun",
				SerialNumber:   "TEST123",
				WeaponTypeID:   weaponType.ID,
				CaliberID:      999,
				ManufacturerID: manufacturer.ID,
				OwnerID:        1,
			},
			expectedErr: "caliber",
		},
		{
			name: "Invalid Manufacturer",
			gun: &models.Gun{
				Name:           "Test Gun",
				SerialNumber:   "TEST123",
				WeaponTypeID:   weaponType.ID,
				CaliberID:      caliber.ID,
				ManufacturerID: 999,
				OwnerID:        1,
			},
			expectedErr: "manufacturer",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Validate using the model's validation method
			err := tc.gun.Validate(db.DB)

			// Check that validation failed as expected
			assert.Error(t, err, "Expected validation to fail")
			if assert.NotNil(t, err, "Error should not be nil") {
				assert.Contains(t, err.Error(), tc.expectedErr,
					"Error message should contain %q, but got: %q",
					tc.expectedErr, err.Error())
			}
		})
	}
}

// TestCreateGunValidationErrors tests validation during gun creation
func TestCreateGunValidationErrors(t *testing.T) {
	// Enable CSRF test mode
	middleware.EnableTestMode()
	defer middleware.DisableTestMode()

	// Setup test database and service
	db := testutils.NewTestDB()
	defer db.Close()
	service := testutils.NewTestService(db.DB)
	helper := testhelper.NewControllerTestHelper(db.DB, service)
	defer helper.CleanupTest()

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)

	// Create some test data for form validation
	weaponType := models.WeaponType{Type: "Test Validation Type", Popularity: 1}
	err := service.CreateWeaponType(&weaponType)
	require.NoError(t, err)

	caliber := models.Caliber{Caliber: "Test Validation Caliber", Popularity: 1}
	err = service.CreateCaliber(&caliber)
	require.NoError(t, err)

	manufacturer := models.Manufacturer{Name: "Test Validation Manufacturer", Country: "USA", Popularity: 1}
	err = service.CreateManufacturer(&manufacturer)
	require.NoError(t, err)

	// Create form data with validation errors
	formData := url.Values{
		"name":            {"Test Gun With Error"}, // Valid name, but validation will catch other errors
		"serial_number":   {"TEST123"},
		"purpose":         {string(make([]byte, 101))}, // Purpose too long
		"paid":            {"-100"},                    // Negative price
		"weapon_type_id":  {fmt.Sprintf("%d", weaponType.ID)},
		"caliber_id":      {fmt.Sprintf("%d", caliber.ID)},
		"manufacturer_id": {fmt.Sprintf("%d", manufacturer.ID)},
		"csrf_token":      {"test_token"},
	}

	// Create controller and setup the route
	gunController := controller.NewOwnerController(service)

	// Create a custom router with our adapter
	router := gin.New()
	router.Use(func(c *gin.Context) {
		// Set up the auth controller adapter
		mockAuthService := helper.AuthService.(*testhelper.MockAuthService)
		adapter := &MockAuthControllerAdapter{
			mockService: mockAuthService,
			userID:      testUser.ID,
			userEmail:   testUser.Email,
		}
		c.Set("authController", adapter)

		// Set authenticated flag
		c.Set("authenticated", true)

		// Set user data
		c.Set("user", gin.H{"id": testUser.ID, "email": testUser.Email})

		// Set CSRF token
		c.Set("csrf_token", "test_token")

		// Set flash function
		c.Set("setFlash", func(msg string) {
			// No-op for tests
		})

		// Set auth data for views
		c.Set("authData", gin.H{
			"IsAuthenticated": true,
			"UserEmail":       testUser.Email,
			"UserID":          testUser.ID,
			"CSRFToken":       "test_token",
		})

		c.Next()
	})

	// Add the gun controller route with the correct path
	router.POST("/owner/guns", gunController.Create)

	// Make request with error handling
	req, err := http.NewRequest("POST", "/owner/guns", strings.NewReader(formData.Encode()))
	require.NoError(t, err, "Failed to create request")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-CSRF-TEST-MODE", "1")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert response contains form rendering with validation errors
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "Add New Firearm") // Should re-render the form

	// Verify no gun was created
	var count int64
	db.DB.Model(&models.Gun{}).Where("owner_id = ?", testUser.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

// TestCreateGunUnauthenticated tests gun creation when not authenticated
func TestCreateGunUnauthenticated(t *testing.T) {
	// Enable CSRF test mode
	middleware.EnableTestMode()
	defer middleware.DisableTestMode()

	// Setup test database and service
	db := testutils.NewTestDB()
	defer db.Close()
	service := testutils.NewTestService(db.DB)
	helper := testhelper.NewControllerTestHelper(db.DB, service)
	defer helper.CleanupTest()

	// Create controller and setup a router with no auth controller
	controller := controller.NewOwnerController(service)
	router := gin.Default()

	// Use middleware without authentication
	router.Use(middleware.CSRFMiddleware())

	// Add the route
	router.POST("/owner/guns", controller.Create)

	// Make request
	formData := url.Values{"csrf_token": {"test_token"}}
	req, err := http.NewRequest("POST", "/owner/guns", strings.NewReader(formData.Encode()))
	require.NoError(t, err, "Failed to create request")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-CSRF-TEST-MODE", "1")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert redirect to login
	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/login", w.Header().Get("Location"))

	// Verify no gun was created
	var count int64
	db.DB.Model(&models.Gun{}).Count(&count)
	assert.Equal(t, int64(0), count)
}

// TestGunIndex tests the index action for guns
func TestGunIndex(t *testing.T) {
	// Enable CSRF test mode
	middleware.EnableTestMode()
	defer middleware.DisableTestMode()

	// Setup test database and service
	db := testutils.NewTestDB()
	defer db.Close()
	service := testutils.NewTestService(db.DB)
	helper := testhelper.NewControllerTestHelper(db.DB, service)
	defer helper.CleanupTest()

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)

	// Get existing data from seeded database
	var weaponType models.WeaponType
	var caliber models.Caliber
	var manufacturer models.Manufacturer

	err := db.DB.First(&weaponType).Error
	require.NoError(t, err, "Failed to get weapon type")

	err = db.DB.First(&caliber).Error
	require.NoError(t, err, "Failed to get caliber")

	err = db.DB.First(&manufacturer).Error
	require.NoError(t, err, "Failed to get manufacturer")

	// Create two test guns
	testGun1 := models.Gun{
		Name:           "Test Gun 1",
		SerialNumber:   "SN-TEST-1",
		WeaponTypeID:   weaponType.ID,
		CaliberID:      caliber.ID,
		ManufacturerID: manufacturer.ID,
		OwnerID:        testUser.ID,
	}
	err = db.DB.Create(&testGun1).Error
	require.NoError(t, err, "Failed to create test gun 1")

	testGun2 := models.Gun{
		Name:           "Test Gun 2",
		SerialNumber:   "SN-TEST-2",
		WeaponTypeID:   weaponType.ID,
		CaliberID:      caliber.ID,
		ManufacturerID: manufacturer.ID,
		OwnerID:        testUser.ID,
	}
	err = db.DB.Create(&testGun2).Error
	require.NoError(t, err, "Failed to create test gun 2")

	// Create controller and setup the route
	controller := controller.NewOwnerController(service)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)
	router.GET("/owner/guns/arsenal", controller.Arsenal)

	// Make the request
	req, err := http.NewRequest("GET", "/owner/guns/arsenal", nil)
	require.NoError(t, err, "Failed to create request")
	req.Header.Set("X-CSRF-TEST-MODE", "1")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Verify response
	assert.Equal(t, http.StatusOK, rr.Code, "Expected status OK")
	assert.Contains(t, rr.Body.String(), "Test Gun 1", "Response should contain first gun name")
	assert.Contains(t, rr.Body.String(), "Test Gun 2", "Response should contain second gun name")
}

// TestGunShow tests displaying a single gun record
func TestGunShow(t *testing.T) {
	// Enable CSRF test mode
	middleware.EnableTestMode()
	defer middleware.DisableTestMode()

	// Setup test database and service
	db := testutils.NewTestDB()
	defer db.Close()
	service := testutils.NewTestService(db.DB)
	helper := testhelper.NewControllerTestHelper(db.DB, service)
	defer helper.CleanupTest()

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)

	// Get existing data from seeded database
	var weaponType models.WeaponType
	var caliber models.Caliber
	var manufacturer models.Manufacturer

	err := db.DB.First(&weaponType).Error
	require.NoError(t, err, "Failed to get weapon type")

	err = db.DB.First(&caliber).Error
	require.NoError(t, err, "Failed to get caliber")

	err = db.DB.First(&manufacturer).Error
	require.NoError(t, err, "Failed to get manufacturer")

	// Create a test gun with optional fields
	paid := 1299.99
	acquiredDate := time.Now().AddDate(0, -1, 0) // 1 month ago

	testGun := models.Gun{
		Name:           "Test Detail Gun",
		SerialNumber:   "SN-DETAIL-1",
		Purpose:        "Home Defense",
		WeaponTypeID:   weaponType.ID,
		CaliberID:      caliber.ID,
		ManufacturerID: manufacturer.ID,
		OwnerID:        testUser.ID,
		Paid:           &paid,
		Acquired:       &acquiredDate,
	}
	err = db.DB.Create(&testGun).Error
	require.NoError(t, err, "Failed to create test gun")

	// Create controller and setup the route
	controller := controller.NewOwnerController(service)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)
	router.GET("/owner/guns/:id", controller.Show)

	// Make the request
	req, err := http.NewRequest("GET", fmt.Sprintf("/owner/guns/%d", testGun.ID), nil)
	require.NoError(t, err, "Failed to create request")
	req.Header.Set("X-CSRF-TEST-MODE", "1")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Verify response
	assert.Equal(t, http.StatusOK, rr.Code, "Expected status OK")
	assert.Contains(t, rr.Body.String(), "Test Detail Gun", "Response should contain gun name")
	assert.Contains(t, rr.Body.String(), "SN-DETAIL-1", "Response should contain serial number")
	assert.Contains(t, rr.Body.String(), "Home Defense", "Response should contain purpose")
	assert.Contains(t, rr.Body.String(), manufacturer.Name, "Response should contain manufacturer name")
	assert.Contains(t, rr.Body.String(), caliber.Caliber, "Response should contain caliber")
	assert.Contains(t, rr.Body.String(), weaponType.Type, "Response should contain weapon type")
	assert.Contains(t, rr.Body.String(), "$1299.99", "Response should contain paid amount")
}

// TestGunEdit tests the edit action for firearms
func TestGunEdit(t *testing.T) {
	// Setup test environment
	middleware.EnableTestMode()
	defer middleware.DisableTestMode()

	db := testutils.NewTestDB()
	defer db.Close()
	service := testutils.NewTestService(db.DB)
	helper := testhelper.NewControllerTestHelper(db.DB, service)
	defer helper.CleanupTest()

	// Create a test user
	testUser := helper.CreateTestUser(t)

	// Create test data
	weaponType := models.WeaponType{Type: "Pistol", Popularity: 1}
	err := service.CreateWeaponType(&weaponType)
	require.NoError(t, err)

	caliber := models.Caliber{Caliber: "9mm", Popularity: 1}
	err = service.CreateCaliber(&caliber)
	require.NoError(t, err)

	manufacturer := models.Manufacturer{Name: "Glock", Country: "Austria", Popularity: 1}
	err = service.CreateManufacturer(&manufacturer)
	require.NoError(t, err)

	paid := 599.99
	acquiredDate := time.Now().AddDate(0, -2, 0) // 2 months ago

	testGun := models.Gun{
		Name:           "Test Edit Gun",
		SerialNumber:   "SN-EDIT-1",
		Purpose:        "Home Defense",
		WeaponTypeID:   weaponType.ID,
		CaliberID:      caliber.ID,
		ManufacturerID: manufacturer.ID,
		OwnerID:        testUser.ID,
		Paid:           &paid,
		Acquired:       &acquiredDate,
	}
	err = db.DB.Create(&testGun).Error
	require.NoError(t, err, "Failed to create test gun")

	// Setup the controller and router
	controller := controller.NewOwnerController(service)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)
	router.GET("/owner/guns/:id/edit", controller.Edit)

	// Make the request
	req, err := http.NewRequest("GET", fmt.Sprintf("/owner/guns/%d/edit", testGun.ID), nil)
	require.NoError(t, err, "Failed to create request")
	req.Header.Set("X-CSRF-TEST-MODE", "1")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Verify response
	assert.Equal(t, http.StatusOK, rr.Code, "Expected status OK")
	assert.Contains(t, rr.Body.String(), "Edit Firearm", "Response should contain page title")
	assert.Contains(t, rr.Body.String(), "Test Edit Gun", "Response should contain gun name")
	assert.Contains(t, rr.Body.String(), "SN-EDIT-1", "Response should contain serial number")
	assert.Contains(t, rr.Body.String(), "Home Defense", "Response should contain purpose")
	assert.Contains(t, rr.Body.String(), "599.99", "Response should contain price")
}

// TestGunUpdate tests the update action for firearms
func TestGunUpdate(t *testing.T) {
	// Setup test environment
	middleware.EnableTestMode()
	defer middleware.DisableTestMode()

	db := testutils.NewTestDB()
	defer db.Close()
	service := testutils.NewTestService(db.DB)
	helper := testhelper.NewControllerTestHelper(db.DB, service)
	defer helper.CleanupTest()

	// Create a test user
	testUser := helper.CreateTestUser(t)

	// Create test data
	weaponType := models.WeaponType{Type: "Pistol", Popularity: 1}
	err := service.CreateWeaponType(&weaponType)
	require.NoError(t, err)

	newWeaponType := models.WeaponType{Type: "Rifle", Popularity: 1}
	err = service.CreateWeaponType(&newWeaponType)
	require.NoError(t, err)

	caliber := models.Caliber{Caliber: "9mm", Popularity: 1}
	err = service.CreateCaliber(&caliber)
	require.NoError(t, err)

	newCaliber := models.Caliber{Caliber: "5.56 NATO", Popularity: 1}
	err = service.CreateCaliber(&newCaliber)
	require.NoError(t, err)

	manufacturer := models.Manufacturer{Name: "Glock", Country: "Austria", Popularity: 1}
	err = service.CreateManufacturer(&manufacturer)
	require.NoError(t, err)

	newManufacturer := models.Manufacturer{Name: "Smith & Wesson", Country: "USA", Popularity: 1}
	err = service.CreateManufacturer(&newManufacturer)
	require.NoError(t, err)

	// Create a test gun
	testGun := models.Gun{
		Name:           "Test Update Gun",
		SerialNumber:   "SN-UPDATE-1",
		Purpose:        "Home Defense",
		WeaponTypeID:   weaponType.ID,
		CaliberID:      caliber.ID,
		ManufacturerID: manufacturer.ID,
		OwnerID:        testUser.ID,
	}
	err = db.DB.Create(&testGun).Error
	require.NoError(t, err, "Failed to create test gun")

	// Create form data for update
	newPaid := "899.99"
	newAcquiredDate := time.Now().AddDate(0, -1, 0).Format("2006-01-02") // 1 month ago

	formData := url.Values{}
	formData.Set("name", "Updated Gun Name")
	formData.Set("serial_number", "SN-UPDATED")
	formData.Set("purpose", "Range & Competition")
	formData.Set("weapon_type_id", fmt.Sprintf("%d", newWeaponType.ID))
	formData.Set("caliber_id", fmt.Sprintf("%d", newCaliber.ID))
	formData.Set("manufacturer_id", fmt.Sprintf("%d", newManufacturer.ID))
	formData.Set("paid", newPaid)
	formData.Set("acquired_date", newAcquiredDate)
	formData.Set("csrf_token", "test_token")

	// Setup the controller and router
	controller := controller.NewOwnerController(service)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)
	router.POST("/owner/guns/:id", controller.Update)

	// Make request with error handling
	req, err := http.NewRequest("POST", fmt.Sprintf("/owner/guns/%d", testGun.ID), strings.NewReader(formData.Encode()))
	require.NoError(t, err, "Failed to create request")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-CSRF-TEST-MODE", "1")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Check if we were redirected - update to match actual controller behavior
	assert.Equal(t, http.StatusSeeOther, rr.Code, "Expected redirect status")
	assert.Equal(t, "/owner", rr.Header().Get("Location"), "Expected redirect to owner dashboard")

	// Verify the update was successful
	var updatedGun models.Gun
	err = db.DB.First(&updatedGun, testGun.ID).Error
	require.NoError(t, err, "Failed to retrieve updated gun")

	assert.Equal(t, "Updated Gun Name", updatedGun.Name, "Name should be updated")
	assert.Equal(t, "SN-UPDATED", updatedGun.SerialNumber, "Serial number should be updated")
	assert.Equal(t, "Range & Competition", updatedGun.Purpose, "Purpose should be updated")
	assert.Equal(t, newWeaponType.ID, updatedGun.WeaponTypeID, "Weapon type should be updated")
	assert.Equal(t, newCaliber.ID, updatedGun.CaliberID, "Caliber should be updated")
	assert.Equal(t, newManufacturer.ID, updatedGun.ManufacturerID, "Manufacturer should be updated")

	// Check the optional fields
	require.NotNil(t, updatedGun.Paid, "Paid should be set")
	assert.InDelta(t, 899.99, *updatedGun.Paid, 0.01, "Paid amount should be updated")

	require.NotNil(t, updatedGun.Acquired, "Acquired date should be set")
	expectedDate, _ := time.Parse("2006-01-02", newAcquiredDate)
	assert.Equal(t, expectedDate.Format("2006-01-02"), updatedGun.Acquired.Format("2006-01-02"), "Acquired date should be updated")
}

// TestGunUpdateValidationErrors tests validation errors during gun updates
func TestGunUpdateValidationErrors(t *testing.T) {
	// Setup test environment
	middleware.EnableTestMode()
	defer middleware.DisableTestMode()

	db := testutils.NewTestDB()
	defer db.Close()
	service := testutils.NewTestService(db.DB)
	helper := testhelper.NewControllerTestHelper(db.DB, service)
	defer helper.CleanupTest()

	// Create a test user
	testUser := helper.CreateTestUser(t)

	// Create test data
	weaponType := models.WeaponType{Type: "Pistol", Popularity: 1}
	err := service.CreateWeaponType(&weaponType)
	require.NoError(t, err)

	caliber := models.Caliber{Caliber: "9mm", Popularity: 1}
	err = service.CreateCaliber(&caliber)
	require.NoError(t, err)

	manufacturer := models.Manufacturer{Name: "Glock", Country: "Austria", Popularity: 1}
	err = service.CreateManufacturer(&manufacturer)
	require.NoError(t, err)

	// Create a test gun
	testGun := models.Gun{
		Name:           "Test Validation Gun",
		SerialNumber:   "SN-VALIDATE-1",
		Purpose:        "Home Defense",
		WeaponTypeID:   weaponType.ID,
		CaliberID:      caliber.ID,
		ManufacturerID: manufacturer.ID,
		OwnerID:        testUser.ID,
	}
	err = db.DB.Create(&testGun).Error
	require.NoError(t, err, "Failed to create test gun")

	// Create form data with validation errors
	formData := url.Values{}
	formData.Set("name", "") // Empty name should fail validation
	formData.Set("serial_number", "SN-VALIDATE-1")
	formData.Set("purpose", string(make([]byte, 101))) // Purpose too long (over 100 chars)
	formData.Set("weapon_type_id", fmt.Sprintf("%d", weaponType.ID))
	formData.Set("caliber_id", fmt.Sprintf("%d", caliber.ID))
	formData.Set("manufacturer_id", fmt.Sprintf("%d", manufacturer.ID))
	formData.Set("paid", "-100") // Negative price should fail validation
	formData.Set("csrf_token", "test_token")

	// Setup the controller and router
	controller := controller.NewOwnerController(service)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)
	router.POST("/owner/guns/:id", controller.Update)

	// Make request with error handling
	req, err := http.NewRequest("POST", fmt.Sprintf("/owner/guns/%d", testGun.ID), strings.NewReader(formData.Encode()))
	require.NoError(t, err, "Failed to create request")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-CSRF-TEST-MODE", "1")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Update assertions to match actual controller behavior
	assert.Equal(t, http.StatusSeeOther, rr.Code, "Expected redirect status")
	expectedRedirect := fmt.Sprintf("/owner/guns/%d/edit", testGun.ID)
	assert.Equal(t, expectedRedirect, rr.Header().Get("Location"), "Expected redirect to edit page with validation errors")

	// Verify gun was not updated (the validation should still prevent updates)
	var unchangedGun models.Gun
	err = db.DB.First(&unchangedGun, testGun.ID).Error
	require.NoError(t, err, "Failed to retrieve gun")

	assert.Equal(t, "Test Validation Gun", unchangedGun.Name, "Name should remain unchanged")
	assert.Equal(t, "Home Defense", unchangedGun.Purpose, "Purpose should remain unchanged")
}

// TestGunDelete tests the deletion of a gun
func TestGunDelete(t *testing.T) {
	// Setup test environment
	middleware.EnableTestMode()
	defer middleware.DisableTestMode()

	db := testutils.NewTestDB()
	defer db.Close()
	service := testutils.NewTestService(db.DB)
	helper := testhelper.NewControllerTestHelper(db.DB, service)
	defer helper.CleanupTest()

	// Create a test user and test gun
	testUser := helper.CreateTestUser(t)

	// Create test data
	weaponType := models.WeaponType{Type: "Pistol for Deletion", Popularity: 1}
	err := service.CreateWeaponType(&weaponType)
	require.NoError(t, err)

	caliber := models.Caliber{Caliber: "9mm for Deletion", Popularity: 1}
	err = service.CreateCaliber(&caliber)
	require.NoError(t, err)

	manufacturer := models.Manufacturer{Name: "Glock for Deletion", Country: "Austria", Popularity: 1}
	err = service.CreateManufacturer(&manufacturer)
	require.NoError(t, err)

	testGun := models.Gun{
		Name:           "Test Delete Gun",
		SerialNumber:   "SN-DELETE-1",
		Purpose:        "Testing Deletion",
		WeaponTypeID:   weaponType.ID,
		CaliberID:      caliber.ID,
		ManufacturerID: manufacturer.ID,
		OwnerID:        testUser.ID,
	}
	err = db.DB.Create(&testGun).Error
	require.NoError(t, err, "Failed to create test gun")

	// Setup the controller and router
	controller := controller.NewOwnerController(service)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)
	router.POST("/owner/guns/:id/delete", controller.Delete)

	// Make the delete request
	req, err := http.NewRequest("POST", fmt.Sprintf("/owner/guns/%d/delete", testGun.ID), nil)
	require.NoError(t, err, "Failed to create request")
	req.Header.Set("X-CSRF-TEST-MODE", "1")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Verify we were redirected to the owner dashboard
	assert.Equal(t, http.StatusSeeOther, rr.Code, "Expected redirect status")
	assert.Equal(t, "/owner", rr.Header().Get("Location"), "Expected redirect to owner dashboard")

	// Verify the gun was soft deleted (should have DeletedAt set)
	var deletedGun models.Gun
	err = db.DB.Unscoped().First(&deletedGun, testGun.ID).Error
	require.NoError(t, err, "Failed to retrieve deleted gun")

	// Check if deleted_at is set (not nil), which indicates soft deletion
	assert.NotNil(t, deletedGun.DeletedAt, "Gun should be soft deleted")
}
