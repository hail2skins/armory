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

// TestAdminBulletStyleIndex tests the Index handler for bullet styles
func TestAdminBulletStyleIndex(t *testing.T) {
	// Setup test database and service
	db := testutils.NewTestDB()
	defer db.Close()                           // Close the database connection when done
	service := testutils.NewTestService(db.DB) // Pass db.DB

	// Setup test helper
	helper := testhelper.NewControllerTestHelper(db.DB, service)
	defer helper.CleanupTest() // Ensure cleanup after test

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)

	// Create some test bullet styles using the service
	err := service.CreateBulletStyle(&models.BulletStyle{Type: "FMJ", Nickname: "Full Metal Jacket", Popularity: 100})
	assert.NoError(t, err)
	err = service.CreateBulletStyle(&models.BulletStyle{Type: "JHP", Nickname: "Jacketed Hollow Point", Popularity: 90})
	assert.NoError(t, err)

	// Create controller with the service
	bulletStyleController := controller.NewAdminBulletStyleController(service)

	// Get an authenticated router using the test user
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)
	router.GET("/admin/bullet_styles", bulletStyleController.Index)

	// Perform the request using the helper's method
	w := helper.AssertViewRendered(
		t,
		router,
		http.MethodGet,
		"/admin/bullet_styles",
		nil,
		http.StatusOK,
	)

	// Assertions
	// Check if the response body contains expected text
	assert.Contains(t, w.Body.String(), "Bullet Styles")
	assert.Contains(t, w.Body.String(), "FMJ")
	assert.Contains(t, w.Body.String(), "JHP")
}

// TestAdminBulletStyleNew tests the New handler
func TestAdminBulletStyleNew(t *testing.T) {
	// Setup
	db := testutils.NewTestDB()
	defer db.Close()
	service := testutils.NewTestService(db.DB)
	helper := testhelper.NewControllerTestHelper(db.DB, service)
	defer helper.CleanupTest()

	// Create controller
	bulletStyleController := controller.NewAdminBulletStyleController(service)

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)

	// Register the route
	router.GET("/admin/bullet_styles/new", bulletStyleController.New)

	// Perform the request
	w := helper.AssertViewRendered(
		t,
		router,
		http.MethodGet,
		"/admin/bullet_styles/new",
		nil,
		http.StatusOK,
	)

	// Assertions
	assert.Contains(t, w.Body.String(), "New Bullet Style") // Check for expected title or content
	assert.Contains(t, w.Body.String(), "csrf_token")       // Check for CSRF token input
}

// TestAdminBulletStyleCreate tests the Create handler
func TestAdminBulletStyleCreate(t *testing.T) {
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
	bulletStyleController := controller.NewAdminBulletStyleController(service)

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)

	// Register the route for Create handler
	router.POST("/admin/bullet_styles", bulletStyleController.Create)

	// Define form data
	formData := url.Values{}
	formData.Set("type", "Test BulletStyle Create")
	formData.Set("nickname", "TBSC")
	formData.Set("popularity", "99")

	// Make request
	req, _ := http.NewRequest("POST", "/admin/bullet_styles", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusSeeOther, rr.Code, "Expected status code 303 See Other for successful create")

	// Verify redirection URL
	location, err := rr.Result().Location()
	assert.NoError(t, err, "Expected Location header")
	assert.Contains(t, location.Path, "/admin/bullet_styles", "Expected redirection to bullet_styles index")

	// Verify database record
	bulletStyles, err := service.FindAllBulletStyles()
	assert.NoError(t, err, "Error finding bullet styles")

	// Look for a bullet style with the test type
	var found bool
	var createdBulletStyle models.BulletStyle
	for _, bs := range bulletStyles {
		if bs.Type == "Test BulletStyle Create" {
			found = true
			createdBulletStyle = bs
			break
		}
	}
	assert.True(t, found, "Expected to find a bullet style with type 'Test BulletStyle Create'")
	assert.Equal(t, "Test BulletStyle Create", createdBulletStyle.Type)
	assert.Equal(t, "TBSC", createdBulletStyle.Nickname)
	assert.Equal(t, 99, createdBulletStyle.Popularity)
}

// TestAdminBulletStyleShow tests the Show handler
func TestAdminBulletStyleShow(t *testing.T) {
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
	bulletStyleController := controller.NewAdminBulletStyleController(service)

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)

	// Register the route for Show handler
	router.GET("/admin/bullet_styles/:id", bulletStyleController.Show)

	// Create a bullet style directly in the DB for testing
	testBulletStyle := models.BulletStyle{Type: "Show Me BulletStyle", Nickname: "SMS", Popularity: 5}
	err := service.CreateBulletStyle(&testBulletStyle)
	require.NoError(t, err, "Failed to create test bullet style")
	require.NotZero(t, testBulletStyle.ID, "Test bullet style ID should not be zero")

	// Make request to the Show endpoint
	showURL := fmt.Sprintf("/admin/bullet_styles/%d", testBulletStyle.ID)
	req, _ := http.NewRequest("GET", showURL, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code, "Expected status code 200 OK")

	// Check if the response body contains the bullet style's type and nickname
	body := rr.Body.String()
	assert.Contains(t, body, "Show Me BulletStyle", "Response body should contain the bullet style type")
	assert.Contains(t, body, "SMS", "Response body should contain the bullet style nickname")
}

// TestAdminBulletStyleEdit tests the Edit handler
func TestAdminBulletStyleEdit(t *testing.T) {
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
	bulletStyleController := controller.NewAdminBulletStyleController(service)

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)

	// Register the route for Edit handler
	router.GET("/admin/bullet_styles/:id/edit", bulletStyleController.Edit)

	// Create a bullet style for testing
	testBulletStyle := models.BulletStyle{Type: "Edit Me BulletStyle", Nickname: "EMB", Popularity: 10}
	err := service.CreateBulletStyle(&testBulletStyle)
	require.NoError(t, err, "Failed to create test bullet style")
	require.NotZero(t, testBulletStyle.ID, "Test bullet style ID should not be zero")

	// Make request to the Edit endpoint
	editURL := fmt.Sprintf("/admin/bullet_styles/%d/edit", testBulletStyle.ID)
	req, _ := http.NewRequest("GET", editURL, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code, "Expected status code 200 OK")

	// Check if the response body contains form elements with the bullet style's data
	body := rr.Body.String()
	assert.Contains(t, body, "Edit Bullet Style", "Response body should contain the title")
	assert.Contains(t, body, "Edit Me BulletStyle", "Response body should contain the bullet style type")
	assert.Contains(t, body, "EMB", "Response body should contain the bullet style nickname")
	assert.Contains(t, body, fmt.Sprintf("value=\"%d\"", testBulletStyle.Popularity), "Response body should contain the popularity value")
	assert.Contains(t, body, "csrf_token", "Response body should contain the CSRF token field")
}

// TestAdminBulletStyleUpdate tests the Update handler
func TestAdminBulletStyleUpdate(t *testing.T) {
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
	bulletStyleController := controller.NewAdminBulletStyleController(service)

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)

	// Register the route for Update handler
	router.POST("/admin/bullet_styles/:id", bulletStyleController.Update)

	// Create a bullet style to update
	testBulletStyle := models.BulletStyle{Type: "Before Update", Nickname: "BU", Popularity: 5}
	err := service.CreateBulletStyle(&testBulletStyle)
	require.NoError(t, err, "Failed to create test bullet style")
	require.NotZero(t, testBulletStyle.ID, "Test bullet style ID should not be zero")

	// Prepare form data for update
	formData := url.Values{}
	formData.Set("type", "After Update")
	formData.Set("nickname", "AU")
	formData.Set("popularity", "10")

	// Make request to update the bullet style
	updateURL := fmt.Sprintf("/admin/bullet_styles/%d", testBulletStyle.ID)
	req, _ := http.NewRequest("POST", updateURL, strings.NewReader(formData.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Assertions for redirect
	assert.Equal(t, http.StatusFound, rr.Code, "Expected status code 302 Found for successful update")
	location, err := rr.Result().Location()
	assert.NoError(t, err, "Expected Location header")
	assert.Contains(t, location.Path, "/admin/bullet_styles", "Expected redirection to bullet_styles index")

	// Verify the bullet style was updated in the database
	updatedBulletStyle, err := service.FindBulletStyleByID(testBulletStyle.ID)
	assert.NoError(t, err, "Expected to find the updated bullet style")
	assert.Equal(t, "After Update", updatedBulletStyle.Type, "Expected bullet style type to be updated")
	assert.Equal(t, "AU", updatedBulletStyle.Nickname, "Expected bullet style nickname to be updated")
	assert.Equal(t, 10, updatedBulletStyle.Popularity, "Expected bullet style popularity to be updated")
}

// TestAdminBulletStyleDelete tests the Delete handler
func TestAdminBulletStyleDelete(t *testing.T) {
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
	bulletStyleController := controller.NewAdminBulletStyleController(service)

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)

	// Register the route for Delete handler
	router.POST("/admin/bullet_styles/:id/delete", bulletStyleController.Delete)

	// Create a bullet style to delete
	testBulletStyle := models.BulletStyle{Type: "Delete Me BulletStyle", Nickname: "DMB", Popularity: 3}
	err := service.CreateBulletStyle(&testBulletStyle)
	require.NoError(t, err, "Failed to create test bullet style")
	require.NotZero(t, testBulletStyle.ID, "Test bullet style ID should not be zero")

	// Make request to delete the bullet style
	deleteURL := fmt.Sprintf("/admin/bullet_styles/%d/delete", testBulletStyle.ID)
	req, _ := http.NewRequest("POST", deleteURL, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Assertions for redirect
	assert.Equal(t, http.StatusFound, rr.Code, "Expected status code 302 Found for successful delete")
	location, err := rr.Result().Location()
	assert.NoError(t, err, "Expected Location header")
	assert.Contains(t, location.Path, "/admin/bullet_styles", "Expected redirection to bullet_styles index")

	// Verify the bullet style was deleted from the database
	_, err = service.FindBulletStyleByID(testBulletStyle.ID)
	assert.Error(t, err, "Expected error when finding deleted bullet style")
	assert.Equal(t, gorm.ErrRecordNotFound, err, "Expected record not found error")
}

// TestAdminBulletStyleRestoreSoftDeleted tests restoring a soft-deleted bullet style
func TestAdminBulletStyleRestoreSoftDeleted(t *testing.T) {
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
	bulletStyleController := controller.NewAdminBulletStyleController(service)

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)

	// Register routes
	router.POST("/admin/bullet_styles", bulletStyleController.Create)
	router.POST("/admin/bullet_styles/:id/delete", bulletStyleController.Delete)

	// Step 1: Create a bullet style
	formData := url.Values{}
	formData.Set("type", "Restore Test BulletStyle")
	formData.Set("nickname", "RTB")
	formData.Set("popularity", "50")

	req, _ := http.NewRequest("POST", "/admin/bullet_styles", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Verify creation was successful
	assert.True(t,
		rr.Code == http.StatusFound || rr.Code == http.StatusSeeOther,
		"Expected redirect status code (302 Found or 303 See Other), got %d", rr.Code)

	// Find the created bullet style
	bulletStyles, _ := service.FindAllBulletStyles()
	var bulletStyleID uint
	for _, bs := range bulletStyles {
		if bs.Type == "Restore Test BulletStyle" {
			bulletStyleID = bs.ID
			break
		}
	}
	assert.NotZero(t, bulletStyleID, "Bullet style should have been created with a valid ID")

	// Step 2: Delete the bullet style
	deleteURL := fmt.Sprintf("/admin/bullet_styles/%d/delete", bulletStyleID)
	req, _ = http.NewRequest("POST", deleteURL, nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Verify delete was successful
	assert.True(t,
		rr.Code == http.StatusFound || rr.Code == http.StatusSeeOther,
		"Expected redirect status code (302 Found or 303 See Other), got %d", rr.Code)

	// Verify the bullet style is no longer in the active bullet styles list
	bulletStyles, _ = service.FindAllBulletStyles()
	var found bool
	for _, bs := range bulletStyles {
		if bs.Type == "Restore Test BulletStyle" {
			found = true
			break
		}
	}
	assert.False(t, found, "Bullet style should not be found in active bullet styles after deletion")

	// Step 3: Try to recreate the same bullet style (should restore the soft-deleted one)
	formData = url.Values{}
	formData.Set("type", "Restore Test BulletStyle")
	formData.Set("nickname", "Updated RTB") // Different nickname
	formData.Set("popularity", "75")        // Different popularity to verify it updates

	req, _ = http.NewRequest("POST", "/admin/bullet_styles", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Verify recreation/restoration was successful
	assert.True(t,
		rr.Code == http.StatusFound || rr.Code == http.StatusSeeOther,
		"Expected redirect status code (302 Found or 303 See Other), got %d", rr.Code)

	// Find the restored bullet style and verify its properties
	bulletStyles, _ = service.FindAllBulletStyles()
	var restoredBulletStyle models.BulletStyle
	for _, bs := range bulletStyles {
		if bs.Type == "Restore Test BulletStyle" {
			restoredBulletStyle = bs
			found = true
			break
		}
	}
	assert.True(t, found, "Bullet style should be found in active bullet styles after restoration")
	assert.Equal(t, bulletStyleID, restoredBulletStyle.ID, "Restored bullet style should have the same ID")
	assert.Equal(t, "Updated RTB", restoredBulletStyle.Nickname, "Restored bullet style should have updated nickname")
	assert.Equal(t, 75, restoredBulletStyle.Popularity, "Restored bullet style should have updated popularity")
}
