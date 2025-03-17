package controller

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/testutils/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestOwnerGunNew tests the New function for gun creation
func TestOwnerGunNew(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create a new mock DB
	mockDB := new(mocks.MockDB)

	// Create a new mock auth controller
	mockAuthController := new(mocks.MockAuthController)

	// Create a test user
	user := &database.User{
		Model: gorm.Model{
			ID: 1,
		},
		Email: "test@example.com",
	}

	// Create test data
	weaponTypes := []models.WeaponType{
		{ID: 1, Type: "Rifle", Popularity: 10},
		{ID: 2, Type: "Pistol", Popularity: 20},
	}

	calibers := []models.Caliber{
		{Model: gorm.Model{ID: 1}, Caliber: ".223", Popularity: 10},
		{Model: gorm.Model{ID: 2}, Caliber: "9mm", Popularity: 20},
	}

	manufacturers := []models.Manufacturer{
		{Model: gorm.Model{ID: 1}, Name: "Colt", Popularity: 10},
		{Model: gorm.Model{ID: 2}, Name: "Glock", Popularity: 20},
	}

	// Set up expectations
	mockAuthInfo := &mocks.MockAuthInfo{}
	mockAuthInfo.SetUserName("test@example.com")

	// Expect GetCurrentUser to be called and return the mock auth info and true
	mockAuthController.On("GetCurrentUser", mock.Anything).Return(mockAuthInfo, true)

	// Expect GetUserByEmail to be called with the user's email and return the user
	mockDB.On("GetUserByEmail", mock.Anything, "test@example.com").Return(user, nil)

	// Expect FindAllWeaponTypes to be called and return the weapon types
	mockDB.On("FindAllWeaponTypes").Return(weaponTypes, nil)

	// Expect FindAllCalibers to be called and return the calibers
	mockDB.On("FindAllCalibers").Return(calibers, nil)

	// Expect FindAllManufacturers to be called and return the manufacturers
	mockDB.On("FindAllManufacturers").Return(manufacturers, nil)

	// Create a new owner controller with the mock DB
	ownerController := NewOwnerController(mockDB)

	// Create a new gin context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create a new request
	req, _ := http.NewRequest("GET", "/owner/guns/new", nil)
	c.Request = req

	// Set the auth controller in the context
	c.Set("authController", mockAuthController)

	// Call the New method
	ownerController.New(c)

	// Assert that the response code is 200
	assert.Equal(t, http.StatusOK, w.Code)
	// Assert that the response body contains the expected content
	assert.Contains(t, w.Body.String(), "Add New Firearm")

	// Verify expectations
	mockAuthController.AssertExpectations(t)
	mockDB.AssertExpectations(t)
}

// TestOwnerGunNewUnauthenticated tests that unauthenticated users are redirected
func TestOwnerGunNewUnauthenticated(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create a new mock DB
	mockDB := new(mocks.MockDB)

	// Create a new mock auth controller
	mockAuthController := new(mocks.MockAuthController)

	// Set up expectations
	// Expect GetCurrentUser to be called and return nil and false
	mockAuthController.On("GetCurrentUser", mock.Anything).Return(nil, false)

	// Create a new owner controller with the mock DB
	ownerController := NewOwnerController(mockDB)

	// Create a new gin context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create a new request
	req, _ := http.NewRequest("GET", "/owner/guns/new", nil)
	c.Request = req

	// Set the auth controller in the context
	c.Set("authController", mockAuthController)

	// Set a mock setFlash function
	c.Set("setFlash", func(msg string) {
		assert.Equal(t, "You must be logged in to access this page", msg)
	})

	// Call the New method
	ownerController.New(c)

	// Assert that the response is a redirect to /login
	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/login", w.Header().Get("Location"))

	// Verify expectations
	mockAuthController.AssertExpectations(t)
}

// TestOwnerGunCreate tests the Create function for gun creation
func TestOwnerGunCreate(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create a new mock DB
	mockDB := new(mocks.MockDB)

	// Create a new mock auth controller
	mockAuthController := new(mocks.MockAuthController)

	// Create a test user
	user := &database.User{
		Model: gorm.Model{
			ID: 1,
		},
		Email: "test@example.com",
	}

	// Set up expectations
	mockAuthInfo := &mocks.MockAuthInfo{}
	mockAuthInfo.SetUserName("test@example.com")

	// Expect GetCurrentUser to be called and return the mock auth info and true
	mockAuthController.On("GetCurrentUser", mock.Anything).Return(mockAuthInfo, true)

	// Expect GetUserByEmail to be called with the user's email and return the user
	mockDB.On("GetUserByEmail", mock.Anything, "test@example.com").Return(user, nil)

	// Create a mock DB connection with the guns table
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})

	// Create the related tables
	err := db.AutoMigrate(&models.WeaponType{}, &models.Caliber{}, &models.Manufacturer{}, &models.Gun{})
	assert.NoError(t, err, "Failed to create tables")

	// Create test data in the database
	weaponType := models.WeaponType{Type: "Rifle", Popularity: 10}
	err = db.Create(&weaponType).Error
	assert.NoError(t, err, "Failed to create weapon type")

	caliber := models.Caliber{Caliber: ".223", Popularity: 10}
	err = db.Create(&caliber).Error
	assert.NoError(t, err, "Failed to create caliber")

	manufacturer := models.Manufacturer{Name: "Colt", Popularity: 10}
	err = db.Create(&manufacturer).Error
	assert.NoError(t, err, "Failed to create manufacturer")

	// Expect GetDB to be called and return the mock DB connection
	mockDB.On("GetDB").Return(db)

	// Create a new owner controller with the mock DB
	ownerController := NewOwnerController(mockDB)

	// Create a new gin router
	router := gin.New()

	// Set up middleware
	router.Use(func(c *gin.Context) {
		c.Set("authController", mockAuthController)
		c.Set("setFlash", func(msg string) {
			t.Logf("Flash message: %s", msg)
			assert.Equal(t, "Weapon added to your arsenal", msg)
		})
		c.Next()
	})

	// Register the route
	router.POST("/owner/guns", ownerController.Create)

	// Create a new recorder
	w := httptest.NewRecorder()

	// Create form data
	form := url.Values{}
	form.Add("name", "Test Gun")
	form.Add("serial_number", "123456")
	form.Add("weapon_type_id", "1")
	form.Add("caliber_id", "1")
	form.Add("manufacturer_id", "1")
	form.Add("acquired_date", "2023-01-01")

	// Create a new request
	req, _ := http.NewRequest("POST", "/owner/guns", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Serve the request through the router
	router.ServeHTTP(w, req)

	// Print the response body for debugging
	t.Logf("Response body: %s", w.Body.String())
	t.Logf("Response code: %d", w.Code)
	t.Logf("Response headers: %v", w.Header())

	// Assert that the response is a redirect to /owner
	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/owner", w.Header().Get("Location"))

	// Verify expectations
	mockAuthController.AssertExpectations(t)
	mockDB.AssertExpectations(t)
}

// TestOwnerGunCreateUnauthenticated tests that unauthenticated users are redirected
func TestOwnerGunCreateUnauthenticated(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create a new mock DB
	mockDB := new(mocks.MockDB)

	// Create a new mock auth controller
	mockAuthController := new(mocks.MockAuthController)

	// Set up expectations
	// Expect GetCurrentUser to be called and return nil and false
	mockAuthController.On("GetCurrentUser", mock.Anything).Return(nil, false)

	// Create a new owner controller with the mock DB
	ownerController := NewOwnerController(mockDB)

	// Create a new gin router
	router := gin.New()

	// Set up middleware
	router.Use(func(c *gin.Context) {
		c.Set("authController", mockAuthController)
		c.Set("setFlash", func(msg string) {
			assert.Equal(t, "You must be logged in to access this page", msg)
		})
		c.Next()
	})

	// Register the route
	router.POST("/owner/guns", ownerController.Create)

	// Create a new recorder
	w := httptest.NewRecorder()

	// Create a new request
	req, _ := http.NewRequest("POST", "/owner/guns", nil)

	// Serve the request through the router
	router.ServeHTTP(w, req)

	// Assert that the response is a redirect to /login
	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/login", w.Header().Get("Location"))

	// Verify expectations
	mockAuthController.AssertExpectations(t)
}

// TestOwnerGunCreateValidation tests validation for gun creation
func TestOwnerGunCreateValidation(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create a new mock DB
	mockDB := new(mocks.MockDB)

	// Create a new mock auth controller
	mockAuthController := new(mocks.MockAuthController)

	// Create a test user
	user := &database.User{
		Model: gorm.Model{
			ID: 1,
		},
		Email: "test@example.com",
	}

	// Create test data
	weaponTypes := []models.WeaponType{
		{ID: 1, Type: "Rifle", Popularity: 10},
		{ID: 2, Type: "Pistol", Popularity: 20},
	}

	calibers := []models.Caliber{
		{Model: gorm.Model{ID: 1}, Caliber: ".223", Popularity: 10},
		{Model: gorm.Model{ID: 2}, Caliber: "9mm", Popularity: 20},
	}

	manufacturers := []models.Manufacturer{
		{Model: gorm.Model{ID: 1}, Name: "Colt", Popularity: 10},
		{Model: gorm.Model{ID: 2}, Name: "Glock", Popularity: 20},
	}

	// Set up expectations
	mockAuthInfo := &mocks.MockAuthInfo{}
	mockAuthInfo.SetUserName("test@example.com")

	// Expect GetCurrentUser to be called and return the mock auth info and true
	mockAuthController.On("GetCurrentUser", mock.Anything).Return(mockAuthInfo, true)

	// Expect GetUserByEmail to be called with the user's email and return the user
	mockDB.On("GetUserByEmail", mock.Anything, "test@example.com").Return(user, nil)

	// Expect FindAllWeaponTypes to be called and return the weapon types
	mockDB.On("FindAllWeaponTypes").Return(weaponTypes, nil)

	// Expect FindAllCalibers to be called and return the calibers
	mockDB.On("FindAllCalibers").Return(calibers, nil)

	// Expect FindAllManufacturers to be called and return the manufacturers
	mockDB.On("FindAllManufacturers").Return(manufacturers, nil)

	// Create a new owner controller with the mock DB
	ownerController := NewOwnerController(mockDB)

	// Create a new gin context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create form data with missing required fields
	form := url.Values{}

	// Create a new request
	req, _ := http.NewRequest("POST", "/owner/guns", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Request = req

	// Set the auth controller in the context
	c.Set("authController", mockAuthController)

	// Call the Create method
	ownerController.Create(c)

	// Assert that the response code is 200 (form re-rendered with errors)
	assert.Equal(t, http.StatusOK, w.Code)
	// Assert that the response body contains the expected content
	assert.Contains(t, w.Body.String(), "Add New Firearm")
	assert.Contains(t, w.Body.String(), "Name is required")
	assert.Contains(t, w.Body.String(), "Weapon type is required")
	assert.Contains(t, w.Body.String(), "Caliber is required")
	assert.Contains(t, w.Body.String(), "Manufacturer is required")

	// Verify expectations
	mockAuthController.AssertExpectations(t)
	mockDB.AssertExpectations(t)
}

// TestOwnerGunShow tests the Show function for displaying a gun
func TestOwnerGunShow(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create a new mock DB
	mockDB := new(mocks.MockDB)

	// Create a new mock auth controller
	mockAuthController := new(mocks.MockAuthController)

	// Create a test user
	user := &database.User{
		Model: gorm.Model{
			ID: 1,
		},
		Email: "test@example.com",
	}

	// Create a test gun with preloaded relationships
	gun := models.Gun{
		Model: gorm.Model{
			ID: 1,
		},
		Name:         "Test Gun",
		SerialNumber: "123456",
		OwnerID:      1,
		WeaponTypeID: 1,
		WeaponType: models.WeaponType{
			ID:   1,
			Type: "Rifle",
		},
		CaliberID: 1,
		Caliber: models.Caliber{
			Model:   gorm.Model{ID: 1},
			Caliber: ".223",
		},
		ManufacturerID: 1,
		Manufacturer: models.Manufacturer{
			Model: gorm.Model{ID: 1},
			Name:  "Colt",
		},
	}

	// Create a mock DB that will return our gun
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	// Migrate the schema
	err = db.AutoMigrate(&models.Gun{}, &models.WeaponType{}, &models.Caliber{}, &models.Manufacturer{})
	assert.NoError(t, err)

	// Create the gun in the database
	err = db.Create(&gun).Error
	assert.NoError(t, err)

	// Set up expectations
	mockAuthInfo := &mocks.MockAuthInfo{}
	mockAuthInfo.SetUserName("test@example.com")

	// Expect GetCurrentUser to be called and return the mock auth info and true
	mockAuthController.On("GetCurrentUser", mock.Anything).Return(mockAuthInfo, true)

	// Expect GetUserByEmail to be called with the user's email and return the user
	mockDB.On("GetUserByEmail", mock.Anything, "test@example.com").Return(user, nil)

	// Expect GetDB to be called and return the test DB
	mockDB.On("GetDB").Return(db)

	// Create a new owner controller with the mock DB
	ownerController := NewOwnerController(mockDB)

	// Create a new gin context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create a new request
	req, _ := http.NewRequest("GET", "/owner/guns/1", nil)
	c.Request = req
	c.Params = []gin.Param{{Key: "id", Value: "1"}}

	// Set the auth controller in the context
	c.Set("authController", mockAuthController)

	// Call the Show method
	ownerController.Show(c)

	// Assert that the response code is 200
	assert.Equal(t, http.StatusOK, w.Code)

	// Assert that the response body contains the expected content
	assert.Contains(t, w.Body.String(), "Firearm Details")
	assert.Contains(t, w.Body.String(), "Test Gun")
	assert.Contains(t, w.Body.String(), "123456")
	assert.Contains(t, w.Body.String(), "Rifle")
	assert.Contains(t, w.Body.String(), ".223")
	assert.Contains(t, w.Body.String(), "Colt")

	// Verify expectations
	mockAuthController.AssertExpectations(t)
	mockDB.AssertExpectations(t)
}

// TestOwnerGunShowUnauthenticated tests the Show function when the user is not authenticated
func TestOwnerGunShowUnauthenticated(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create a new mock DB
	mockDB := new(mocks.MockDB)

	// Create a new mock auth controller
	mockAuthController := new(mocks.MockAuthController)

	// Set up expectations
	// Expect GetCurrentUser to be called and return nil and false (not authenticated)
	mockAuthController.On("GetCurrentUser", mock.Anything).Return(nil, false)

	// Create a new owner controller with the mock DB
	ownerController := NewOwnerController(mockDB)

	// Create a new gin context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create a new request
	req, _ := http.NewRequest("GET", "/owner/guns/1", nil)
	c.Request = req
	c.Params = []gin.Param{{Key: "id", Value: "1"}}

	// Set the auth controller in the context
	c.Set("authController", mockAuthController)

	// Call the Show method
	ownerController.Show(c)

	// Assert that the response is a redirect to login
	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/login", w.Header().Get("Location"))

	// Verify expectations
	mockAuthController.AssertExpectations(t)
}

// TestOwnerGunArsenal tests the Arsenal function for displaying all guns
func TestOwnerGunArsenal(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create a new mock DB
	mockDB := new(mocks.MockDB)

	// Create a new mock auth controller
	mockAuthController := new(mocks.MockAuthController)

	// Create a test user
	user := &database.User{
		Model: gorm.Model{
			ID: 1,
		},
		Email: "test@example.com",
	}

	// Create test guns
	guns := []models.Gun{
		{
			Name:         "Test Gun 1",
			SerialNumber: "123456",
			OwnerID:      1,
			WeaponTypeID: 1,
			WeaponType: models.WeaponType{
				ID:   1,
				Type: "Rifle",
			},
			CaliberID: 1,
			Caliber: models.Caliber{
				Model:   gorm.Model{ID: 1},
				Caliber: ".223",
			},
			ManufacturerID: 1,
			Manufacturer: models.Manufacturer{
				Model: gorm.Model{ID: 1},
				Name:  "Colt",
			},
		},
		{
			Name:         "Test Gun 2",
			SerialNumber: "654321",
			OwnerID:      1,
			WeaponTypeID: 2,
			WeaponType: models.WeaponType{
				ID:   2,
				Type: "Pistol",
			},
			CaliberID: 2,
			Caliber: models.Caliber{
				Model:   gorm.Model{ID: 2},
				Caliber: "9mm",
			},
			ManufacturerID: 2,
			Manufacturer: models.Manufacturer{
				Model: gorm.Model{ID: 2},
				Name:  "Glock",
			},
		},
	}

	// Create a mock DB that will return our guns
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	// Migrate the schema
	err = db.AutoMigrate(&models.Gun{}, &models.WeaponType{}, &models.Caliber{}, &models.Manufacturer{})
	assert.NoError(t, err)

	// Create the guns in the database
	for _, gun := range guns {
		err = db.Create(&gun).Error
		assert.NoError(t, err)
	}

	// Set up expectations
	mockAuthInfo := &mocks.MockAuthInfo{}
	mockAuthInfo.SetUserName("test@example.com")

	// Expect GetCurrentUser to be called and return the mock auth info and true
	mockAuthController.On("GetCurrentUser", mock.Anything).Return(mockAuthInfo, true)

	// Expect GetUserByEmail to be called with the user's email and return the user
	mockDB.On("GetUserByEmail", mock.Anything, "test@example.com").Return(user, nil)

	// Expect GetDB to be called and return the test DB
	mockDB.On("GetDB").Return(db)

	// Create a new owner controller with the mock DB
	ownerController := NewOwnerController(mockDB)

	// Create a new gin context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create a new request
	req, _ := http.NewRequest("GET", "/owner/guns/arsenal", nil)
	c.Request = req

	// Set the auth controller in the context
	c.Set("authController", mockAuthController)

	// Call the Arsenal method
	ownerController.Arsenal(c)

	// Assert that the response code is 200
	assert.Equal(t, http.StatusOK, w.Code)

	// Assert that the response body contains the expected content
	body := w.Body.String()
	assert.Contains(t, body, "Your Arsenal")
	assert.Contains(t, body, "Test Gun 1")
	assert.Contains(t, body, "Test Gun 2")
	assert.Contains(t, body, "Rifle")
	assert.Contains(t, body, "Pistol")
	assert.Contains(t, body, ".223")
	assert.Contains(t, body, "9mm")
	assert.Contains(t, body, "Colt")
	assert.Contains(t, body, "Glock")

	// Verify expectations
	mockAuthController.AssertExpectations(t)
	mockDB.AssertExpectations(t)
}

// TestOwnerGunArsenalUnauthenticated tests the Arsenal function when the user is not authenticated
func TestOwnerGunArsenalUnauthenticated(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create a new mock DB
	mockDB := new(mocks.MockDB)

	// Create a new mock auth controller
	mockAuthController := new(mocks.MockAuthController)

	// Set up expectations
	// Expect GetCurrentUser to be called and return nil and false (not authenticated)
	mockAuthController.On("GetCurrentUser", mock.Anything).Return(nil, false)

	// Create a new owner controller with the mock DB
	ownerController := NewOwnerController(mockDB)

	// Create a new gin context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create a new request
	req, _ := http.NewRequest("GET", "/owner/guns/arsenal", nil)
	c.Request = req

	// Set the auth controller in the context
	c.Set("authController", mockAuthController)

	// Call the Arsenal method
	ownerController.Arsenal(c)

	// Assert that the response is a redirect to login
	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/login", w.Header().Get("Location"))

	// Verify expectations
	mockAuthController.AssertExpectations(t)
}

// TestOwnerGunEdit tests the Edit function for gun editing
func TestOwnerGunEdit(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create a new mock DB
	mockDB := new(mocks.MockDB)

	// Create a new mock auth controller
	mockAuthController := new(mocks.MockAuthController)

	// Create a test user
	user := &database.User{
		Model: gorm.Model{
			ID: 1,
		},
		Email: "test@example.com",
	}

	// Create test data
	weaponTypes := []models.WeaponType{
		{ID: 1, Type: "Rifle", Popularity: 10},
		{ID: 2, Type: "Pistol", Popularity: 20},
	}

	calibers := []models.Caliber{
		{Model: gorm.Model{ID: 1}, Caliber: ".223", Popularity: 10},
		{Model: gorm.Model{ID: 2}, Caliber: "9mm", Popularity: 20},
	}

	manufacturers := []models.Manufacturer{
		{Model: gorm.Model{ID: 1}, Name: "Colt", Popularity: 10},
		{Model: gorm.Model{ID: 2}, Name: "Glock", Popularity: 20},
	}

	// Create a test gun
	gun := &models.Gun{
		Model: gorm.Model{
			ID: 1,
		},
		Name:           "Test Gun",
		SerialNumber:   "123456",
		OwnerID:        1,
		WeaponTypeID:   1,
		CaliberID:      1,
		ManufacturerID: 1,
	}

	// Create a mock DB connection
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})

	// Create the necessary tables
	err := db.AutoMigrate(&models.Gun{}, &models.WeaponType{}, &models.Caliber{}, &models.Manufacturer{})
	assert.NoError(t, err, "Failed to create tables")

	// Insert test data
	err = db.Create(gun).Error
	assert.NoError(t, err, "Failed to create gun")

	for _, wt := range weaponTypes {
		err = db.Create(&wt).Error
		assert.NoError(t, err, "Failed to create weapon type")
	}

	for _, cal := range calibers {
		err = db.Create(&cal).Error
		assert.NoError(t, err, "Failed to create caliber")
	}

	for _, mfr := range manufacturers {
		err = db.Create(&mfr).Error
		assert.NoError(t, err, "Failed to create manufacturer")
	}

	// Set up expectations
	mockAuthInfo := &mocks.MockAuthInfo{}
	mockAuthInfo.SetUserName("test@example.com")

	// Expect GetCurrentUser to be called and return the mock auth info and true
	mockAuthController.On("GetCurrentUser", mock.Anything).Return(mockAuthInfo, true)

	// Expect GetUserByEmail to be called with the user's email and return the user
	mockDB.On("GetUserByEmail", mock.Anything, "test@example.com").Return(user, nil)

	// Expect GetDB to be called and return the mock DB connection
	mockDB.On("GetDB").Return(db)

	// Create a new owner controller with the mock DB
	ownerController := NewOwnerController(mockDB)

	// Create a new gin context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create a new request
	req, _ := http.NewRequest("GET", "/owner/guns/1/edit", nil)
	c.Request = req
	c.Params = []gin.Param{{Key: "id", Value: "1"}}

	// Set the auth controller in the context
	c.Set("authController", mockAuthController)
	c.Set("setFlash", func(msg string) {})

	// Call the Edit function
	ownerController.Edit(c)

	// Assert expectations
	mockAuthController.AssertExpectations(t)
	mockDB.AssertExpectations(t)

	// Assert the response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Edit Firearm")
	assert.Contains(t, w.Body.String(), "Test Gun")
	assert.Contains(t, w.Body.String(), "123456")
}

// TestOwnerGunEditUnauthenticated tests the Edit function for gun editing when user is not authenticated
func TestOwnerGunEditUnauthenticated(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create a new mock DB
	mockDB := new(mocks.MockDB)

	// Create a new mock auth controller
	mockAuthController := new(mocks.MockAuthController)

	// Set up expectations
	mockAuthController.On("GetCurrentUser", mock.Anything).Return(nil, false)

	// Create a new gin context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create a new request
	req, _ := http.NewRequest("GET", "/owner/guns/1/edit", nil)
	c.Request = req
	c.Params = []gin.Param{{Key: "id", Value: "1"}}

	// Set the auth controller
	c.Set("authController", mockAuthController)
	c.Set("setFlash", func(msg string) {})

	// Create a new owner controller
	ownerController := NewOwnerController(mockDB)

	// Call the Edit function
	ownerController.Edit(c)

	// Assert expectations
	mockAuthController.AssertExpectations(t)

	// Assert the response
	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/login", w.Header().Get("Location"))
}

// TestOwnerGunUpdate tests the Update function for gun updating
func TestOwnerGunUpdate(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create a new mock DB
	mockDB := new(mocks.MockDB)

	// Create a new mock auth controller
	mockAuthController := new(mocks.MockAuthController)

	// Create a test user
	user := &database.User{
		Model: gorm.Model{
			ID: 1,
		},
		Email: "test@example.com",
	}

	// Create a mock DB connection with SQL logging
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	// Create the necessary tables
	err := db.AutoMigrate(&models.Gun{}, &models.WeaponType{}, &models.Caliber{}, &models.Manufacturer{})
	assert.NoError(t, err, "Failed to create tables")

	// Create test weapon types
	weaponTypes := []models.WeaponType{
		{ID: 1, Type: "Rifle", Popularity: 10},
		{ID: 2, Type: "Pistol", Popularity: 20},
	}
	for _, wt := range weaponTypes {
		err = db.Create(&wt).Error
		assert.NoError(t, err, "Failed to create weapon type")
	}

	// Create test calibers
	calibers := []models.Caliber{
		{Model: gorm.Model{ID: 1}, Caliber: ".223", Popularity: 10},
		{Model: gorm.Model{ID: 2}, Caliber: "9mm", Popularity: 20},
	}
	for _, cal := range calibers {
		err = db.Create(&cal).Error
		assert.NoError(t, err, "Failed to create caliber")
	}

	// Create test manufacturers
	manufacturers := []models.Manufacturer{
		{Model: gorm.Model{ID: 1}, Name: "Colt", Popularity: 10},
		{Model: gorm.Model{ID: 2}, Name: "Glock", Popularity: 20},
	}
	for _, mfr := range manufacturers {
		err = db.Create(&mfr).Error
		assert.NoError(t, err, "Failed to create manufacturer")
	}

	// Create a test gun
	gun := &models.Gun{
		Model: gorm.Model{
			ID: 1,
		},
		Name:           "Test Gun",
		SerialNumber:   "123456",
		OwnerID:        1,
		WeaponTypeID:   1, // Rifle
		CaliberID:      1, // .223
		ManufacturerID: 1, // Colt
	}

	// Insert test data
	err = db.Create(gun).Error
	assert.NoError(t, err, "Failed to create gun")

	// Verify the gun was created with the correct relationships
	var createdGun models.Gun
	err = db.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").First(&createdGun, 1).Error
	assert.NoError(t, err, "Failed to load created gun")
	t.Logf("Created gun: ID=%d, Name=%s, WeaponTypeID=%d, CaliberID=%d, ManufacturerID=%d",
		createdGun.ID, createdGun.Name, createdGun.WeaponTypeID, createdGun.CaliberID, createdGun.ManufacturerID)
	t.Logf("Created gun relationships: WeaponType=%s, Caliber=%s, Manufacturer=%s",
		createdGun.WeaponType.Type, createdGun.Caliber.Caliber, createdGun.Manufacturer.Name)

	// Set up expectations
	mockAuthInfo := &mocks.MockAuthInfo{}
	mockAuthInfo.SetUserName("test@example.com")

	// Expect GetCurrentUser to be called and return the mock auth info and true
	mockAuthController.On("GetCurrentUser", mock.Anything).Return(mockAuthInfo, true)

	// Expect GetUserByEmail to be called with the user's email and return the user
	mockDB.On("GetUserByEmail", mock.Anything, "test@example.com").Return(user, nil)

	// Expect GetDB to be called and return the mock DB connection
	mockDB.On("GetDB").Return(db)

	// Create a new owner controller with the mock DB
	ownerController := NewOwnerController(mockDB)

	// Create a new gin router
	router := gin.New()

	// Set up middleware
	router.Use(func(c *gin.Context) {
		c.Set("authController", mockAuthController)
		c.Set("setFlash", func(msg string) {
			t.Logf("Flash message: %s", msg)
			assert.Equal(t, "Your gun has been updated.", msg)
		})
		c.Next()
	})

	// Register the route
	router.POST("/owner/guns/:id", ownerController.Update)

	// Create form data - updating to Pistol, 9mm, and Glock
	form := url.Values{}
	form.Add("name", "Updated Gun")
	form.Add("serial_number", "654321")
	form.Add("weapon_type_id", "2")  // Pistol
	form.Add("caliber_id", "2")      // 9mm
	form.Add("manufacturer_id", "2") // Glock
	form.Add("acquired_date", "2023-01-01")

	// Create a new recorder
	w := httptest.NewRecorder()

	// Create a new request
	req, _ := http.NewRequest("POST", "/owner/guns/1", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Serve the request through the router
	router.ServeHTTP(w, req)

	// Print the response body for debugging
	t.Logf("Response body: %s", w.Body.String())
	t.Logf("Response code: %d", w.Code)
	t.Logf("Response headers: %v", w.Header())

	// Assert expectations
	mockAuthController.AssertExpectations(t)
	mockDB.AssertExpectations(t)

	// Assert the response
	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/owner", w.Header().Get("Location"))

	// Verify the gun was updated in the database with all relationships
	var updatedGun models.Gun
	err = db.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").First(&updatedGun, 1).Error
	assert.NoError(t, err)

	// Print detailed information about the updated gun
	t.Logf("Updated gun: ID=%d, Name=%s, WeaponTypeID=%d, CaliberID=%d, ManufacturerID=%d",
		updatedGun.ID, updatedGun.Name, updatedGun.WeaponTypeID, updatedGun.CaliberID, updatedGun.ManufacturerID)
	t.Logf("Updated gun relationships: WeaponType=%s, Caliber=%s, Manufacturer=%s",
		updatedGun.WeaponType.Type, updatedGun.Caliber.Caliber, updatedGun.Manufacturer.Name)

	// Check basic fields
	assert.Equal(t, "Updated Gun", updatedGun.Name)
	assert.Equal(t, "654321", updatedGun.SerialNumber)

	// Check relationship IDs
	assert.Equal(t, uint(2), updatedGun.WeaponTypeID)
	assert.Equal(t, uint(2), updatedGun.CaliberID)
	assert.Equal(t, uint(2), updatedGun.ManufacturerID)

	// Check that the relationships are properly loaded
	assert.NotNil(t, updatedGun.WeaponType)
	assert.Equal(t, "Pistol", updatedGun.WeaponType.Type)

	assert.NotNil(t, updatedGun.Caliber)
	assert.Equal(t, "9mm", updatedGun.Caliber.Caliber)

	assert.NotNil(t, updatedGun.Manufacturer)
	assert.Equal(t, "Glock", updatedGun.Manufacturer.Name)
}

// TestOwnerGunUpdateUnauthenticated tests the Update function for gun updating when user is not authenticated
func TestOwnerGunUpdateUnauthenticated(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create a new mock DB
	mockDB := new(mocks.MockDB)

	// Create a new mock auth controller
	mockAuthController := new(mocks.MockAuthController)

	// Set up expectations
	mockAuthController.On("GetCurrentUser", mock.Anything).Return(nil, false)

	// Create a new owner controller with the mock DB
	ownerController := NewOwnerController(mockDB)

	// Create a new gin router
	router := gin.New()

	// Set up middleware
	router.Use(func(c *gin.Context) {
		c.Set("authController", mockAuthController)
		c.Set("setFlash", func(msg string) {
			t.Logf("Flash message: %s", msg)
			assert.Equal(t, "You must be logged in to access this page", msg)
		})
		c.Next()
	})

	// Register the route
	router.POST("/owner/guns/:id", ownerController.Update)

	// Create a new recorder
	w := httptest.NewRecorder()

	// Create a new request
	req, _ := http.NewRequest("POST", "/owner/guns/1", nil)

	// Serve the request through the router
	router.ServeHTTP(w, req)

	// Print the response body for debugging
	t.Logf("Response body: %s", w.Body.String())
	t.Logf("Response code: %d", w.Code)
	t.Logf("Response headers: %v", w.Header())

	// Assert expectations
	mockAuthController.AssertExpectations(t)

	// Assert the response
	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/login", w.Header().Get("Location"))
}

// TestOwnerGunUpdateValidation tests the Update function for gun updating with validation errors
func TestOwnerGunUpdateValidation(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create a new mock DB
	mockDB := new(mocks.MockDB)

	// Create a new mock auth controller
	mockAuthController := new(mocks.MockAuthController)

	// Create a test user
	user := &database.User{
		Model: gorm.Model{
			ID: 1,
		},
		Email: "test@example.com",
	}

	// Create a test gun
	gun := &models.Gun{
		Model: gorm.Model{
			ID: 1,
		},
		Name:           "Test Gun",
		SerialNumber:   "123456",
		OwnerID:        1,
		WeaponTypeID:   1,
		CaliberID:      1,
		ManufacturerID: 1,
	}

	// Create test data
	weaponTypes := []models.WeaponType{
		{ID: 1, Type: "Rifle", Popularity: 10},
		{ID: 2, Type: "Pistol", Popularity: 20},
	}

	calibers := []models.Caliber{
		{Model: gorm.Model{ID: 1}, Caliber: ".223", Popularity: 10},
		{Model: gorm.Model{ID: 2}, Caliber: "9mm", Popularity: 20},
	}

	manufacturers := []models.Manufacturer{
		{Model: gorm.Model{ID: 1}, Name: "Colt", Popularity: 10},
		{Model: gorm.Model{ID: 2}, Name: "Glock", Popularity: 20},
	}

	// Create a mock DB connection
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})

	// Create the necessary tables
	err := db.AutoMigrate(&models.Gun{}, &models.WeaponType{}, &models.Caliber{}, &models.Manufacturer{})
	assert.NoError(t, err, "Failed to create tables")

	// Insert test data
	err = db.Create(gun).Error
	assert.NoError(t, err, "Failed to create gun")

	for _, wt := range weaponTypes {
		err = db.Create(&wt).Error
		assert.NoError(t, err, "Failed to create weapon type")
	}

	for _, cal := range calibers {
		err = db.Create(&cal).Error
		assert.NoError(t, err, "Failed to create caliber")
	}

	for _, mfr := range manufacturers {
		err = db.Create(&mfr).Error
		assert.NoError(t, err, "Failed to create manufacturer")
	}

	// Set up expectations
	mockAuthInfo := &mocks.MockAuthInfo{}
	mockAuthInfo.SetUserName("test@example.com")

	// Expect GetCurrentUser to be called and return the mock auth info and true
	mockAuthController.On("GetCurrentUser", mock.Anything).Return(mockAuthInfo, true)

	// Expect GetUserByEmail to be called with the user's email and return the user
	mockDB.On("GetUserByEmail", mock.Anything, "test@example.com").Return(user, nil)

	// Expect GetDB to be called and return the mock DB connection
	mockDB.On("GetDB").Return(db)

	// Create a new owner controller
	ownerController := NewOwnerController(mockDB)

	// Create a new gin router
	router := gin.New()

	// Set up middleware
	router.Use(func(c *gin.Context) {
		c.Set("authController", mockAuthController)
		c.Set("setFlash", func(msg string) {
			t.Logf("Flash message: %s", msg)
		})
		c.Next()
	})

	// Register the route
	router.POST("/owner/guns/:id", ownerController.Update)

	// Create form data with empty name (required field)
	form := url.Values{}
	form.Add("name", "")
	form.Add("serial_number", "654321")
	form.Add("weapon_type_id", "2")
	form.Add("caliber_id", "2")
	form.Add("manufacturer_id", "2")

	// Create a new recorder
	w := httptest.NewRecorder()

	// Create a new request
	req, _ := http.NewRequest("POST", "/owner/guns/1", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Serve the request through the router
	router.ServeHTTP(w, req)

	// Print the response body for debugging
	t.Logf("Response body: %s", w.Body.String())
	t.Logf("Response code: %d", w.Code)

	// Assert expectations
	mockAuthController.AssertExpectations(t)
	mockDB.AssertExpectations(t)

	// Assert the response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Edit Firearm")
	assert.Contains(t, w.Body.String(), "Name is required")
}
