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
)

// TestAdminBrandIndex tests the Index handler for brands
func TestAdminBrandIndex(t *testing.T) {
	// Setup test database and service
	db := testutils.NewTestDB()
	defer db.Close()
	service := testutils.NewTestService(db.DB)

	// Setup test helper
	helper := testhelper.NewControllerTestHelper(db.DB, service)
	defer helper.CleanupTest()

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)

	// Create some test brands using the service
	err := service.CreateBrand(&models.Brand{Name: "Federal", Nickname: "Fed", Popularity: 10})
	assert.NoError(t, err)
	err = service.CreateBrand(&models.Brand{Name: "Hornady", Nickname: "Horn", Popularity: 5})
	assert.NoError(t, err)

	// Create controller with the service
	brandController := controller.NewAdminBrandController(service)

	// Get an authenticated router using the test user
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)
	router.GET("/admin/brands", brandController.Index)

	// Perform the request using the helper's method
	w := helper.AssertViewRendered(
		t,
		router,
		http.MethodGet,
		"/admin/brands",
		nil,
		http.StatusOK,
	)

	// Assertions
	// Check if the response body contains expected text
	assert.Contains(t, w.Body.String(), "Brands")
	assert.Contains(t, w.Body.String(), "Federal")
	assert.Contains(t, w.Body.String(), "Hornady")
}

// TestAdminBrandNew tests the New handler
func TestAdminBrandNew(t *testing.T) {
	// Setup
	db := testutils.NewTestDB()
	defer db.Close()
	service := testutils.NewTestService(db.DB)
	helper := testhelper.NewControllerTestHelper(db.DB, service)
	defer helper.CleanupTest()

	// Create controller
	brandController := controller.NewAdminBrandController(service)

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)

	// Register the route
	router.GET("/admin/brands/new", brandController.New)

	// Perform the request
	w := helper.AssertViewRendered(
		t,
		router,
		http.MethodGet,
		"/admin/brands/new",
		nil,
		http.StatusOK,
	)

	// Assertions
	assert.Contains(t, w.Body.String(), "New Brand")
	assert.Contains(t, w.Body.String(), "csrf_token")
}

// TestAdminBrandCreate tests the Create handler
func TestAdminBrandCreate(t *testing.T) {
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
	brandController := controller.NewAdminBrandController(service)

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)

	// Register the route for Create handler
	router.POST("/admin/brands", brandController.Create)

	// Define form data
	formData := url.Values{}
	formData.Set("name", "Test Brand Create")
	formData.Set("nickname", "TBC")
	formData.Set("popularity", "99")

	// Make request
	req, _ := http.NewRequest("POST", "/admin/brands", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusSeeOther, rr.Code, "Expected status code 303 See Other for successful create")

	// Verify redirection URL
	location, err := rr.Result().Location()
	assert.NoError(t, err, "Expected Location header")
	assert.Contains(t, location.Path, "/admin/brands", "Expected redirection to brand index")

	// Verify database record
	brands, err := service.FindAllBrands()
	assert.NoError(t, err, "Error finding brands")

	// Look for a brand with the test name
	var found bool
	var createdBrand models.Brand
	for _, b := range brands {
		if b.Name == "Test Brand Create" {
			found = true
			createdBrand = b
			break
		}
	}
	assert.True(t, found, "Expected to find a brand with name 'Test Brand Create'")
	assert.Equal(t, "Test Brand Create", createdBrand.Name)
	assert.Equal(t, "TBC", createdBrand.Nickname)
	assert.Equal(t, 99, createdBrand.Popularity)
}

// TestAdminBrandShow tests the Show handler
func TestAdminBrandShow(t *testing.T) {
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
	brandController := controller.NewAdminBrandController(service)

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)

	// Register the route for Show handler
	router.GET("/admin/brands/:id", brandController.Show)

	// Create a brand directly in the DB for testing
	testBrand := models.Brand{Name: "Show Me Brand", Nickname: "SMB", Popularity: 5}
	err := service.CreateBrand(&testBrand)
	require.NoError(t, err, "Failed to create test brand")
	require.NotZero(t, testBrand.ID, "Test brand ID should not be zero")

	// Make request to the Show endpoint
	showURL := fmt.Sprintf("/admin/brands/%d", testBrand.ID)
	req, _ := http.NewRequest("GET", showURL, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code, "Expected status code 200 OK")
	assert.Contains(t, rr.Body.String(), "Brand Details")
	assert.Contains(t, rr.Body.String(), "Show Me Brand")
	assert.Contains(t, rr.Body.String(), "SMB")
}

// TestAdminBrandEdit tests the Edit handler
func TestAdminBrandEdit(t *testing.T) {
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
	brandController := controller.NewAdminBrandController(service)

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)

	// Register the route for Edit handler
	router.GET("/admin/brands/:id/edit", brandController.Edit)

	// Create a brand directly in the DB for testing
	testBrand := models.Brand{Name: "Edit Me Brand", Nickname: "EMB", Popularity: 7}
	err := service.CreateBrand(&testBrand)
	require.NoError(t, err, "Failed to create test brand")
	require.NotZero(t, testBrand.ID, "Test brand ID should not be zero")

	// Make request to the Edit endpoint
	editURL := fmt.Sprintf("/admin/brands/%d/edit", testBrand.ID)
	req, _ := http.NewRequest("GET", editURL, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code, "Expected status code 200 OK")
	assert.Contains(t, rr.Body.String(), "Edit Brand")
	assert.Contains(t, rr.Body.String(), "Edit Me Brand")
	assert.Contains(t, rr.Body.String(), "EMB")
	assert.Contains(t, rr.Body.String(), "7")
}

// TestAdminBrandUpdate tests the Update handler
func TestAdminBrandUpdate(t *testing.T) {
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
	brandController := controller.NewAdminBrandController(service)

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)

	// Register the route for Update handler
	router.POST("/admin/brands/:id", brandController.Update)

	// Create a brand directly in the DB for testing
	testBrand := models.Brand{Name: "Update Me Brand", Nickname: "UMB", Popularity: 8}
	err := service.CreateBrand(&testBrand)
	require.NoError(t, err, "Failed to create test brand")
	require.NotZero(t, testBrand.ID, "Test brand ID should not be zero")

	// Define form data for update
	formData := url.Values{}
	formData.Set("name", "Updated Brand Name")
	formData.Set("nickname", "UBN")
	formData.Set("popularity", "15")

	// Make request to the Update endpoint
	updateURL := fmt.Sprintf("/admin/brands/%d", testBrand.ID)
	req, _ := http.NewRequest("POST", updateURL, strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Assertions for redirect
	assert.Equal(t, http.StatusFound, rr.Code, "Expected status code 302 Found for successful update")
	location, err := rr.Result().Location()
	assert.NoError(t, err, "Expected Location header")
	assert.Contains(t, location.Path, "/admin/brands", "Expected redirection to brand index")

	// Verify the brand was updated
	updatedBrand, err := service.FindBrandByID(testBrand.ID)
	assert.NoError(t, err, "Error finding updated brand")
	assert.Equal(t, "Updated Brand Name", updatedBrand.Name)
	assert.Equal(t, "UBN", updatedBrand.Nickname)
	assert.Equal(t, 15, updatedBrand.Popularity)
}

// TestAdminBrandDelete tests the Delete handler
func TestAdminBrandDelete(t *testing.T) {
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
	brandController := controller.NewAdminBrandController(service)

	// Create a test user for authentication context
	testUser := helper.CreateTestUser(t)
	router := helper.GetAuthenticatedRouter(testUser.ID, testUser.Email)

	// Register the route for Delete handler
	router.POST("/admin/brands/:id/delete", brandController.Delete)

	// Create a brand directly in the DB for testing
	testBrand := models.Brand{Name: "Delete Me Brand", Nickname: "DMB", Popularity: 3}
	err := service.CreateBrand(&testBrand)
	require.NoError(t, err, "Failed to create test brand")
	require.NotZero(t, testBrand.ID, "Test brand ID should not be zero")

	// Make request to the Delete endpoint
	deleteURL := fmt.Sprintf("/admin/brands/%d/delete", testBrand.ID)
	req, _ := http.NewRequest("POST", deleteURL, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Assertions for redirect
	assert.Equal(t, http.StatusFound, rr.Code, "Expected status code 302 Found for successful delete")
	location, err := rr.Result().Location()
	assert.NoError(t, err, "Expected Location header")
	assert.Contains(t, location.Path, "/admin/brands", "Expected redirection to brand index")

	// Verify the brand was deleted
	_, err = service.FindBrandByID(testBrand.ID)
	assert.Error(t, err, "Expected error finding deleted brand")
}
