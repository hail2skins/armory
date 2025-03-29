package controller_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/middleware"
	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/testutils"
	"github.com/hail2skins/armory/internal/testutils/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestAdminCasingIndex tests the Index handler for casings
func TestAdminCasingIndex(t *testing.T) {
	// Setup test database and service
	db := testutils.NewTestDB()
	defer db.Close()                           // Close the database connection when done
	service := testutils.NewTestService(db.DB) // Pass db.DB

	// Setup test helper
	helper := testhelper.NewControllerTestHelper(db.DB, service)
	defer helper.CleanupTest() // Ensure cleanup after test

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)

	// Create some test casings using the service
	err := service.CreateCasing(&models.Casing{Type: "Brass", Popularity: 1}) // Will fail until interface is updated
	assert.NoError(t, err)
	err = service.CreateCasing(&models.Casing{Type: "Steel", Popularity: 0}) // Will fail until interface is updated
	assert.NoError(t, err)

	// Create controller with the service
	casingController := controller.NewAdminCasingController(service) // Will fail until controller is created

	// Get an authenticated router using the test user
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)
	router.GET("/admin/casings", casingController.Index)

	// Perform the request using the helper's method
	w := helper.AssertViewRendered(
		t,
		router,
		http.MethodGet,
		"/admin/casings",
		nil,
		http.StatusOK,
	)

	// Assertions
	// Check if the response body contains expected text
	assert.Contains(t, w.Body.String(), "Casings")
	assert.Contains(t, w.Body.String(), "Brass")
	assert.Contains(t, w.Body.String(), "Steel")
}

// Add tests for New, Create, Show, Edit, Update, Delete here...

// TestAdminCasingNew tests the New handler
func TestAdminCasingNew(t *testing.T) {
	// Setup
	db := testutils.NewTestDB()
	defer db.Close()
	service := testutils.NewTestService(db.DB) // Pass db.DB
	helper := testhelper.NewControllerTestHelper(db.DB, service)
	defer helper.CleanupTest()

	// Create controller
	casingController := controller.NewAdminCasingController(service)

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)

	// Register the route
	router.GET("/admin/casings/new", casingController.New)

	// Perform the request
	w := helper.AssertViewRendered(
		t,
		router,
		http.MethodGet,
		"/admin/casings/new",
		nil,
		http.StatusOK,
	)

	// Assertions
	assert.Contains(t, w.Body.String(), "New Casing") // Check for expected title or content
	assert.Contains(t, w.Body.String(), "csrf_token") // Check for CSRF token input
}

// TestAdminCasingCreate tests the Create handler
func TestAdminCasingCreate(t *testing.T) {
	// Enable CSRF test mode
	middleware.EnableTestMode()
	defer middleware.DisableTestMode() // Ensure cleanup

	// Setup
	db := testutils.NewTestDB()
	defer db.Close()
	service := testutils.NewTestService(db.DB)
	helper := testhelper.NewControllerTestHelper(db.DB, service)
	defer helper.CleanupTest()

	// Create controller
	casingController := controller.NewAdminCasingController(service)

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)

	// Register the route for Create handler
	router.POST("/admin/casings", casingController.Create)

	// Define form data
	formData := url.Values{}
	formData.Set("type", "Test Casing Create")
	formData.Set("popularity", "99")

	// Make request
	req, _ := http.NewRequest("POST", "/admin/casings", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusSeeOther, rr.Code, "Expected status code 303 See Other for successful create")

	// Verify redirection URL
	location, err := rr.Result().Location()
	assert.NoError(t, err, "Expected Location header")
	assert.Contains(t, location.Path, "/admin/casings", "Expected redirection to casing index")

	// Verify database record
	casings, err := service.FindAllCasings()
	assert.NoError(t, err, "Error finding casings")

	// Look for a casing with the test type
	var found bool
	var createdCasing models.Casing
	for _, c := range casings {
		if c.Type == "Test Casing Create" {
			found = true
			createdCasing = c
			break
		}
	}
	assert.True(t, found, "Expected to find a casing with type 'Test Casing Create'")
	assert.Equal(t, "Test Casing Create", createdCasing.Type)
	assert.Equal(t, 99, createdCasing.Popularity)
}

// TestAdminCasingShow tests the Show handler
func TestAdminCasingShow(t *testing.T) {
	// Enable test mode for middleware
	middleware.EnableTestMode()
	defer middleware.DisableTestMode()

	// Setup
	db := testutils.NewTestDB()
	defer db.Close()
	service := testutils.NewTestService(db.DB)
	helper := testhelper.NewControllerTestHelper(db.DB, service)
	defer helper.CleanupTest()

	// Create controller
	casingController := controller.NewAdminCasingController(service)

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)

	// Register the route for Show handler - Skip middleware for testing
	router.GET("/admin/casings/:id", casingController.Show)

	// Create a casing directly in the DB for testing
	testCasing := models.Casing{Type: "Show Me Casing", Popularity: 5} // Use Type and Popularity
	err := service.CreateCasing(&testCasing)
	require.NoError(t, err, "Failed to create test casing")
	require.NotZero(t, testCasing.ID, "Test casing ID should not be zero")

	// Make request to the Show endpoint
	showURL := fmt.Sprintf("/admin/casings/%d", testCasing.ID)
	req, _ := http.NewRequest("GET", showURL, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code, "Expected status code 200 OK")

	// Check if the response body contains the casing's type
	body := rr.Body.String()
	assert.Contains(t, body, "Show Me Casing", "Response body should contain the casing type")
	// Popularity might not be directly displayed in a simple Show view, so we'll omit checking for it here for now.
}

// TestAdminCasingEdit tests the Edit handler
func TestAdminCasingEdit(t *testing.T) {
	// Enable test mode for middleware
	middleware.EnableTestMode()
	defer middleware.DisableTestMode()

	// Setup
	db := testutils.NewTestDB()
	defer db.Close()
	service := testutils.NewTestService(db.DB)
	helper := testhelper.NewControllerTestHelper(db.DB, service)
	defer helper.CleanupTest()

	// Create controller
	casingController := controller.NewAdminCasingController(service)

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)

	// Register the route for Edit handler
	router.GET("/admin/casings/:id/edit", casingController.Edit)

	// Create a casing for testing
	testCasing := models.Casing{Type: "Edit Me Casing", Popularity: 10}
	err := service.CreateCasing(&testCasing)
	require.NoError(t, err, "Failed to create test casing")
	require.NotZero(t, testCasing.ID, "Test casing ID should not be zero")

	// Make request to the Edit endpoint
	editURL := fmt.Sprintf("/admin/casings/%d/edit", testCasing.ID)
	req, _ := http.NewRequest("GET", editURL, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code, "Expected status code 200 OK")

	// Check if the response body contains form elements with the casing's data
	body := rr.Body.String()
	assert.Contains(t, body, "Edit Casing", "Response body should contain the title")
	assert.Contains(t, body, "Edit Me Casing", "Response body should contain the casing type")
	assert.Contains(t, body, fmt.Sprintf("value=\"%d\"", testCasing.Popularity), "Response body should contain the popularity value")
	assert.Contains(t, body, "csrf_token", "Response body should contain the CSRF token field")
}

// TestAdminCasingUpdate tests the Update handler
func TestAdminCasingUpdate(t *testing.T) {
	// Enable test mode for middleware
	middleware.EnableTestMode()
	defer middleware.DisableTestMode()

	// Setup
	db := testutils.NewTestDB()
	defer db.Close()
	service := testutils.NewTestService(db.DB)
	helper := testhelper.NewControllerTestHelper(db.DB, service)
	defer helper.CleanupTest()

	// Create controller
	casingController := controller.NewAdminCasingController(service)

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)

	// Register the route for Update handler
	router.POST("/admin/casings/:id", casingController.Update)

	// Create a casing to update
	testCasing := models.Casing{Type: "Before Update", Popularity: 5}
	err := service.CreateCasing(&testCasing)
	require.NoError(t, err, "Failed to create test casing")
	require.NotZero(t, testCasing.ID, "Test casing ID should not be zero")

	// Prepare form data for update
	formData := url.Values{}
	formData.Set("type", "After Update")
	formData.Set("popularity", "10")

	// Make request to update the casing
	updateURL := fmt.Sprintf("/admin/casings/%d", testCasing.ID)
	req, _ := http.NewRequest("POST", updateURL, strings.NewReader(formData.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Assertions for redirect
	assert.Equal(t, http.StatusFound, rr.Code, "Expected status code 302 Found for successful update")
	location, err := rr.Result().Location()
	assert.NoError(t, err, "Expected Location header")
	assert.Contains(t, location.Path, "/admin/casings", "Expected redirection to casing index")

	// Verify the casing was updated in the database
	updatedCasing, err := service.FindCasingByID(testCasing.ID)
	assert.NoError(t, err, "Expected to find the updated casing")
	assert.Equal(t, "After Update", updatedCasing.Type, "Expected casing type to be updated")
	assert.Equal(t, 10, updatedCasing.Popularity, "Expected casing popularity to be updated")
}

// TestAdminCasingDelete tests the Delete handler
func TestAdminCasingDelete(t *testing.T) {
	// Enable test mode for middleware
	middleware.EnableTestMode()
	defer middleware.DisableTestMode()

	// Setup
	db := testutils.NewTestDB()
	defer db.Close()
	service := testutils.NewTestService(db.DB)
	helper := testhelper.NewControllerTestHelper(db.DB, service)
	defer helper.CleanupTest()

	// Create controller
	casingController := controller.NewAdminCasingController(service)

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)

	// Register the route for Delete handler
	router.POST("/admin/casings/:id/delete", casingController.Delete)

	// Create a casing to delete
	testCasing := models.Casing{Type: "Delete Me Casing", Popularity: 3}
	err := service.CreateCasing(&testCasing)
	require.NoError(t, err, "Failed to create test casing")
	require.NotZero(t, testCasing.ID, "Test casing ID should not be zero")

	// Make request to delete the casing
	deleteURL := fmt.Sprintf("/admin/casings/%d/delete", testCasing.ID)
	req, _ := http.NewRequest("POST", deleteURL, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Assertions for redirect
	assert.Equal(t, http.StatusFound, rr.Code, "Expected status code 302 Found for successful delete")
	location, err := rr.Result().Location()
	assert.NoError(t, err, "Expected Location header")
	assert.Contains(t, location.Path, "/admin/casings", "Expected redirection to casing index")

	// Verify the casing was deleted from the database
	_, err = service.FindCasingByID(testCasing.ID)
	assert.Error(t, err, "Expected error when finding deleted casing")
	assert.Equal(t, gorm.ErrRecordNotFound, err, "Expected record not found error")
}

// TestAdminCasingRestoreSoftDeleted tests restoring a soft-deleted casing
func TestAdminCasingRestoreSoftDeleted(t *testing.T) {
	// Enable CSRF test mode
	middleware.EnableTestMode()
	defer middleware.DisableTestMode()

	// Setup
	db := testutils.NewTestDB()
	defer db.Close()
	service := testutils.NewTestService(db.DB)
	helper := testhelper.NewControllerTestHelper(db.DB, service)
	defer helper.CleanupTest()

	// Create controller
	casingController := controller.NewAdminCasingController(service)

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)

	// Register routes
	router.POST("/admin/casings", casingController.Create)
	router.POST("/admin/casings/:id/delete", casingController.Delete)

	// Step 1: Create a casing
	formData := url.Values{}
	formData.Set("type", "Restore Test Casing")
	formData.Set("popularity", "50")

	req, _ := http.NewRequest("POST", "/admin/casings", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Verify creation was successful
	assert.True(t,
		rr.Code == http.StatusFound || rr.Code == http.StatusSeeOther,
		"Expected redirect status code (302 Found or 303 See Other), got %d", rr.Code)

	// Find the created casing
	casings, _ := service.FindAllCasings()
	var casingID uint
	for _, c := range casings {
		if c.Type == "Restore Test Casing" {
			casingID = c.ID
			break
		}
	}
	assert.NotZero(t, casingID, "Casing should have been created with a valid ID")

	// Step 2: Delete the casing
	deleteURL := fmt.Sprintf("/admin/casings/%d/delete", casingID)
	req, _ = http.NewRequest("POST", deleteURL, nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Verify delete was successful
	assert.True(t,
		rr.Code == http.StatusFound || rr.Code == http.StatusSeeOther,
		"Expected redirect status code (302 Found or 303 See Other), got %d", rr.Code)

	// Verify the casing is no longer in the active casings list
	casings, _ = service.FindAllCasings()
	var found bool
	for _, c := range casings {
		if c.Type == "Restore Test Casing" {
			found = true
			break
		}
	}
	assert.False(t, found, "Casing should not be found in active casings after deletion")

	// Step 3: Try to recreate the same casing (should restore the soft-deleted one)
	formData = url.Values{}
	formData.Set("type", "Restore Test Casing")
	formData.Set("popularity", "75") // Different popularity to verify it updates

	req, _ = http.NewRequest("POST", "/admin/casings", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Verify recreation/restoration was successful
	assert.True(t,
		rr.Code == http.StatusFound || rr.Code == http.StatusSeeOther,
		"Expected redirect status code (302 Found or 303 See Other), got %d", rr.Code)

	// Find the restored casing and verify its properties
	casings, _ = service.FindAllCasings()
	var restoredCasing models.Casing
	for _, c := range casings {
		if c.Type == "Restore Test Casing" {
			restoredCasing = c
			found = true
			break
		}
	}
	assert.True(t, found, "Casing should be found in active casings after restoration")
	assert.Equal(t, casingID, restoredCasing.ID, "Restored casing should have the same ID")
	assert.Equal(t, 75, restoredCasing.Popularity, "Restored casing should have updated popularity")
}
