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

// TestAdminGrainIndex tests the Index handler for grains
func TestAdminGrainIndex(t *testing.T) {
	// Setup test database and service
	db := testutils.NewTestDB()
	defer db.Close()                           // Close the database connection when done
	service := testutils.NewTestService(db.DB) // Pass db.DB

	// Setup test helper
	helper := testhelper.NewControllerTestHelper(db.DB, service)
	defer helper.CleanupTest() // Ensure cleanup after test

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)

	// Create some test grains using the service
	err := service.CreateGrain(&models.Grain{Weight: 115, Popularity: 100})
	assert.NoError(t, err)
	err = service.CreateGrain(&models.Grain{Weight: 230, Popularity: 90})
	assert.NoError(t, err)
	// Create a test grain with weight 0 which should display as "Other"
	err = service.CreateGrain(&models.Grain{Weight: 0, Popularity: 999})
	assert.NoError(t, err)

	// Create controller with the service
	grainController := controller.NewAdminGrainController(service)

	// Get an authenticated router using the test user
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)
	router.GET("/admin/grains", grainController.Index)

	// Perform the request using the helper's method
	w := helper.AssertViewRendered(
		t,
		router,
		http.MethodGet,
		"/admin/grains",
		nil,
		http.StatusOK,
	)

	// Assertions
	// Check if the response body contains expected text
	assert.Contains(t, w.Body.String(), "Grains")
	assert.Contains(t, w.Body.String(), "115")
	assert.Contains(t, w.Body.String(), "230")
	assert.Contains(t, w.Body.String(), "Other") // "Other" should be displayed for weight 0
}

// TestAdminGrainNew tests the New handler
func TestAdminGrainNew(t *testing.T) {
	// Setup
	db := testutils.NewTestDB()
	defer db.Close()
	service := testutils.NewTestService(db.DB)
	helper := testhelper.NewControllerTestHelper(db.DB, service)
	defer helper.CleanupTest()

	// Create controller
	grainController := controller.NewAdminGrainController(service)

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)

	// Register the route
	router.GET("/admin/grains/new", grainController.New)

	// Perform the request
	w := helper.AssertViewRendered(
		t,
		router,
		http.MethodGet,
		"/admin/grains/new",
		nil,
		http.StatusOK,
	)

	// Assertions
	assert.Contains(t, w.Body.String(), "New Grain")  // Check for expected title or content
	assert.Contains(t, w.Body.String(), "csrf_token") // Check for CSRF token input
}

// TestAdminGrainCreate tests the Create handler
func TestAdminGrainCreate(t *testing.T) {
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
	grainController := controller.NewAdminGrainController(service)

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)

	// Register the route for Create handler
	router.POST("/admin/grains", grainController.Create)

	// Define form data
	formData := url.Values{}
	formData.Set("weight", "124")    // Test weight
	formData.Set("popularity", "75") // Test popularity

	// Make request
	req, _ := http.NewRequest("POST", "/admin/grains", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusSeeOther, rr.Code, "Expected status code 303 See Other for successful create")

	// Verify redirection URL
	location, err := rr.Result().Location()
	assert.NoError(t, err, "Expected Location header")
	assert.Contains(t, location.Path, "/admin/grains", "Expected redirection to grains index")

	// Verify database record
	grains, err := service.FindAllGrains()
	assert.NoError(t, err, "Error finding grains")

	// Look for a grain with the test weight
	var found bool
	var createdGrain models.Grain
	for _, g := range grains {
		if g.Weight == 124 {
			found = true
			createdGrain = g
			break
		}
	}
	assert.True(t, found, "Expected to find a grain with weight 124")
	assert.Equal(t, 124, createdGrain.Weight)
	assert.Equal(t, 75, createdGrain.Popularity)
}

// TestAdminGrainCreateWithZeroWeight tests creating a grain with weight 0 (Other)
func TestAdminGrainCreateWithZeroWeight(t *testing.T) {
	middleware.EnableTestMode()
	defer middleware.DisableTestMode()

	db := testutils.NewTestDB()
	defer db.Close()
	service := testutils.NewTestService(db.DB)
	helper := testhelper.NewControllerTestHelper(db.DB, service)
	defer helper.CleanupTest()

	grainController := controller.NewAdminGrainController(service)
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)
	router.POST("/admin/grains", grainController.Create)

	formData := url.Values{}
	formData.Set("weight", "0") // "Other" grain
	formData.Set("popularity", "999")

	req, _ := http.NewRequest("POST", "/admin/grains", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusSeeOther, rr.Code)

	grains, _ := service.FindAllGrains()
	var found bool
	for _, g := range grains {
		if g.Weight == 0 {
			found = true
			assert.Equal(t, 999, g.Popularity)
			break
		}
	}
	assert.True(t, found, "Expected to find a grain with weight 0 (Other)")
}

// TestAdminGrainShow tests the Show handler
func TestAdminGrainShow(t *testing.T) {
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
	grainController := controller.NewAdminGrainController(service)

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)

	// Register the route for Show handler
	router.GET("/admin/grains/:id", grainController.Show)

	// Create a grain directly in the DB for testing
	testGrain := models.Grain{Weight: 168, Popularity: 85}
	err := service.CreateGrain(&testGrain)
	require.NoError(t, err, "Failed to create test grain")
	require.NotZero(t, testGrain.ID, "Test grain ID should not be zero")

	// Make request to the Show endpoint
	showURL := fmt.Sprintf("/admin/grains/%d", testGrain.ID)
	req, _ := http.NewRequest("GET", showURL, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code, "Expected status code 200 OK")

	// Check if the response body contains the grain's weight
	body := rr.Body.String()
	assert.Contains(t, body, "168", "Response body should contain the grain weight")
}

// TestAdminGrainShowWithZeroWeight tests the Show handler with a weight of 0 (Other)
func TestAdminGrainShowWithZeroWeight(t *testing.T) {
	middleware.EnableTestMode()
	defer middleware.DisableTestMode()

	db := testutils.NewTestDB()
	defer db.Close()
	service := testutils.NewTestService(db.DB)
	helper := testhelper.NewControllerTestHelper(db.DB, service)
	defer helper.CleanupTest()

	grainController := controller.NewAdminGrainController(service)
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)
	router.GET("/admin/grains/:id", grainController.Show)

	// Create a grain with weight 0
	testGrain := models.Grain{Weight: 0, Popularity: 999}
	err := service.CreateGrain(&testGrain)
	require.NoError(t, err)
	require.NotZero(t, testGrain.ID)

	showURL := fmt.Sprintf("/admin/grains/%d", testGrain.ID)
	req, _ := http.NewRequest("GET", showURL, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	assert.Contains(t, body, "Other", "Response body should show 'Other' for weight 0")
	assert.Contains(t, body, "(0)", "Response body should indicate the actual value is 0")
}

// TestAdminGrainEdit tests the Edit handler
func TestAdminGrainEdit(t *testing.T) {
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
	grainController := controller.NewAdminGrainController(service)

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)

	// Register the route for Edit handler
	router.GET("/admin/grains/:id/edit", grainController.Edit)

	// Create a grain for testing
	testGrain := models.Grain{Weight: 175, Popularity: 80}
	err := service.CreateGrain(&testGrain)
	require.NoError(t, err, "Failed to create test grain")
	require.NotZero(t, testGrain.ID, "Test grain ID should not be zero")

	// Make request to the Edit endpoint
	editURL := fmt.Sprintf("/admin/grains/%d/edit", testGrain.ID)
	req, _ := http.NewRequest("GET", editURL, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code, "Expected status code 200 OK")

	// Check if the response body contains form elements with the grain's data
	body := rr.Body.String()
	assert.Contains(t, body, "Edit Grain", "Response body should contain the title")
	assert.Contains(t, body, fmt.Sprintf("value=\"%d\"", testGrain.Weight), "Response body should contain the weight value")
	assert.Contains(t, body, fmt.Sprintf("value=\"%d\"", testGrain.Popularity), "Response body should contain the popularity value")
	assert.Contains(t, body, "csrf_token", "Response body should contain the CSRF token field")
}

// TestAdminGrainUpdate tests the Update handler
func TestAdminGrainUpdate(t *testing.T) {
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
	grainController := controller.NewAdminGrainController(service)

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)

	// Register the route for Update handler
	router.POST("/admin/grains/:id", grainController.Update)

	// Create a grain to update
	testGrain := models.Grain{Weight: 147, Popularity: 90}
	err := service.CreateGrain(&testGrain)
	require.NoError(t, err, "Failed to create test grain")
	require.NotZero(t, testGrain.ID, "Test grain ID should not be zero")

	// Prepare form data for update
	formData := url.Values{}
	formData.Set("weight", "147")    // Same weight
	formData.Set("popularity", "95") // Updated popularity

	// Make request to update the grain
	updateURL := fmt.Sprintf("/admin/grains/%d", testGrain.ID)
	req, _ := http.NewRequest("POST", updateURL, strings.NewReader(formData.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Assertions for redirect
	assert.Equal(t, http.StatusFound, rr.Code, "Expected status code 302 Found for successful update")
	location, err := rr.Result().Location()
	assert.NoError(t, err, "Expected Location header")
	assert.Contains(t, location.Path, "/admin/grains", "Expected redirection to grains index")

	// Verify the grain was updated in the database
	updatedGrain, err := service.FindGrainByID(testGrain.ID)
	assert.NoError(t, err, "Expected to find the updated grain")
	assert.Equal(t, 147, updatedGrain.Weight, "Expected grain weight to be unchanged")
	assert.Equal(t, 95, updatedGrain.Popularity, "Expected grain popularity to be updated")
}

// TestAdminGrainDelete tests the Delete handler
func TestAdminGrainDelete(t *testing.T) {
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
	grainController := controller.NewAdminGrainController(service)

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)

	// Register the route for Delete handler
	router.POST("/admin/grains/:id/delete", grainController.Delete)

	// Create a grain to delete
	testGrain := models.Grain{Weight: 123, Popularity: 65}
	err := service.CreateGrain(&testGrain)
	require.NoError(t, err, "Failed to create test grain")
	require.NotZero(t, testGrain.ID, "Test grain ID should not be zero")

	// Make request to delete the grain
	deleteURL := fmt.Sprintf("/admin/grains/%d/delete", testGrain.ID)
	req, _ := http.NewRequest("POST", deleteURL, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Assertions for redirect
	assert.Equal(t, http.StatusFound, rr.Code, "Expected status code 302 Found for successful delete")
	location, err := rr.Result().Location()
	assert.NoError(t, err, "Expected Location header")
	assert.Contains(t, location.Path, "/admin/grains", "Expected redirection to grains index")

	// Verify the grain was deleted from the database
	_, err = service.FindGrainByID(testGrain.ID)
	assert.Error(t, err, "Expected error when finding deleted grain")
	assert.Equal(t, gorm.ErrRecordNotFound, err, "Expected record not found error")
}

// TestAdminGrainRestoreSoftDeleted tests restoring a soft-deleted grain
func TestAdminGrainRestoreSoftDeleted(t *testing.T) {
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
	grainController := controller.NewAdminGrainController(service)

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)

	// Register routes
	router.POST("/admin/grains", grainController.Create)
	router.POST("/admin/grains/:id/delete", grainController.Delete)

	// Step 1: Create a grain
	formData := url.Values{}
	formData.Set("weight", "155")
	formData.Set("popularity", "50")

	req, _ := http.NewRequest("POST", "/admin/grains", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Verify creation was successful
	assert.True(t,
		rr.Code == http.StatusFound || rr.Code == http.StatusSeeOther,
		"Expected redirect status code (302 Found or 303 See Other), got %d", rr.Code)

	// Find the created grain
	grains, _ := service.FindAllGrains()
	var grainID uint
	for _, g := range grains {
		if g.Weight == 155 {
			grainID = g.ID
			break
		}
	}
	assert.NotZero(t, grainID, "Grain should have been created with a valid ID")

	// Step 2: Delete the grain
	deleteURL := fmt.Sprintf("/admin/grains/%d/delete", grainID)
	req, _ = http.NewRequest("POST", deleteURL, nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Verify delete was successful
	assert.True(t,
		rr.Code == http.StatusFound || rr.Code == http.StatusSeeOther,
		"Expected redirect status code (302 Found or 303 See Other), got %d", rr.Code)

	// Verify the grain is no longer in the active grains list
	grains, _ = service.FindAllGrains()
	var found bool
	for _, g := range grains {
		if g.Weight == 155 {
			found = true
			break
		}
	}
	assert.False(t, found, "Grain should not be found in active grains after deletion")

	// Step 3: Try to recreate the same grain (should restore the soft-deleted one)
	formData = url.Values{}
	formData.Set("weight", "155")
	formData.Set("popularity", "75") // Different popularity to verify it updates

	req, _ = http.NewRequest("POST", "/admin/grains", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Verify recreation/restoration was successful
	assert.True(t,
		rr.Code == http.StatusFound || rr.Code == http.StatusSeeOther,
		"Expected redirect status code (302 Found or 303 See Other), got %d", rr.Code)

	// Find the restored grain and verify its properties
	grains, _ = service.FindAllGrains()
	var restoredGrain models.Grain
	for _, g := range grains {
		if g.Weight == 155 {
			restoredGrain = g
			found = true
			break
		}
	}
	assert.True(t, found, "Grain should be found in active grains after restoration")
	assert.Equal(t, grainID, restoredGrain.ID, "Restored grain should have the same ID")
	assert.Equal(t, 75, restoredGrain.Popularity, "Restored grain should have updated popularity")
}
