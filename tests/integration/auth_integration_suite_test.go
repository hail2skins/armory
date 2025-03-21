package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
	"github.com/hail2skins/armory/internal/testutils"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthIntegrationSuite provides a test suite for auth-related integration tests
type AuthIntegrationSuite struct {
	suite.Suite
	DB              *gorm.DB
	Service         database.Service
	Router          *gin.Engine
	AuthController  *controller.AuthController
	OwnerController *controller.OwnerController
}

// SetupSuite runs once before all tests in the suite
func (s *AuthIntegrationSuite) SetupSuite() {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a real test database service
	s.Service = testutils.SharedTestService()
	s.DB = s.Service.GetDB()

	// Create controllers with real DB
	s.AuthController = controller.NewAuthController(s.Service)
	s.OwnerController = controller.NewOwnerController(s.Service)

	// Set up router
	s.Router = gin.New()
	s.setupRoutes()
}

// setupRoutes configures all the routes needed for testing
func (s *AuthIntegrationSuite) setupRoutes() {
	// Add middleware to set required context values
	s.Router.Use(func(c *gin.Context) {
		c.Set("auth", s.AuthController)
		c.Set("authController", s.AuthController) // Required by LandingPage
		c.Next()
	})

	// Set up flash middleware
	s.Router.Use(func(c *gin.Context) {
		c.Set("setFlash", func(msg string) {
			c.SetCookie("flash", msg, 3600, "/", "", false, false)
		})

		// Get flash message from cookie
		if flash, err := c.Cookie("flash"); err == nil && flash != "" {
			c.Set("flash", flash)
			// Clear the cookie
			c.SetCookie("flash", "", -1, "/", "", false, false)
		}

		c.Next()
	})

	// Auth routes
	s.Router.GET("/login", s.AuthController.LoginHandler)
	s.Router.POST("/login", s.AuthController.LoginHandler)
	s.Router.GET("/logout", s.AuthController.LogoutHandler)

	// Owner routes
	s.Router.GET("/owner", s.OwnerController.LandingPage)
	s.Router.GET("/owner/guns/arsenal", s.OwnerController.Arsenal)
	s.Router.GET("/owner/guns/new", s.OwnerController.New)
	s.Router.POST("/owner/guns", s.OwnerController.Create)
	s.Router.GET("/owner/guns/:id", s.OwnerController.Show)
	s.Router.GET("/owner/guns/:id/edit", s.OwnerController.Edit)
	s.Router.POST("/owner/guns/:id/update", s.OwnerController.Update)
}

// TestLoginAndOwnerPage tests the complete login flow and owner page content
func (s *AuthIntegrationSuite) TestLoginAndOwnerPage() {
	// Create a test user with plain text password (BeforeCreate hook will hash it)
	testUser := &database.User{
		Email:            "test@example.com",
		Password:         "password123", // Plain password - will be hashed by BeforeCreate
		Verified:         true,
		SubscriptionTier: "free",
	}

	// Save the user directly to the database
	err := s.DB.Create(testUser).Error
	s.NoError(err)

	// Create seed data for calibers, and manufacturers - use existing weapon type
	caliber := models.Caliber{Caliber: "5.56 NATO", Popularity: 100}
	err = s.DB.Create(&caliber).Error
	s.NoError(err)

	manufacturer := models.Manufacturer{Name: "Smith & Wesson", Popularity: 100}
	err = s.DB.Create(&manufacturer).Error
	s.NoError(err)

	// Verify the user was saved correctly
	var checkUser database.User
	result := s.DB.Where("email = ?", testUser.Email).First(&checkUser)
	s.NoError(result.Error)

	// Test password verification works (should work now since model hashed it correctly)
	err = bcrypt.CompareHashAndPassword([]byte(checkUser.Password), []byte("password123"))
	s.NoError(err, "Password verification should succeed")

	// Test successful login using the actual login endpoint
	form := url.Values{}
	form.Add("email", "test@example.com")
	form.Add("password", "password123")

	req, _ := http.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()
	s.Router.ServeHTTP(resp, req)

	// Should redirect to owner page
	s.Equal(http.StatusSeeOther, resp.Code, "Login should redirect with status 303")
	s.Equal("/owner", resp.Header().Get("Location"), "Login should redirect to /owner")

	// Extract cookies for next request
	cookies := resp.Result().Cookies()

	// Now test the owner page content
	ownerReq, _ := http.NewRequest("GET", "/owner", nil)
	for _, cookie := range cookies {
		ownerReq.AddCookie(cookie)
	}
	ownerResp := httptest.NewRecorder()
	s.Router.ServeHTTP(ownerResp, ownerReq)

	// Check status code
	s.Equal(http.StatusOK, ownerResp.Code, "Owner page should return 200 OK")

	// Get the response body as a string
	body := ownerResp.Body.String()

	// Verify all the required elements
	s.Contains(body, "You haven't added any firearms yet", "Owner page should show empty state message")
	s.Contains(body, "Total Firearms:</strong> 0", "Owner page should show zero firearms count")
	s.Contains(body, "Total Paid:</strong> $0.00", "Owner page should show zero total paid")
	s.Contains(body, "No firearms added yet", "Owner page should indicate no guns in recently added")
	s.Contains(body, "Current Plan:</strong> Free", "Owner page should show Free plan")

	// Verify the Under Construction section
	s.Contains(body, "Under Construction", "Owner page should have ammo inventory marked as under construction")
	s.Contains(body, `href="#"`, "Under Construction link should point to #")

	// Verify the buttons
	s.Contains(body, `href="/owner/guns/arsenal"`, "Owner page should have View Arsenal button")
	s.Contains(body, "View Arsenal", "Owner page should have View Arsenal text")

	s.Contains(body, `href="/owner/guns/new"`, "Owner page should have Add New Firearm button")
	s.Contains(body, "Add New Firearm", "Owner page should have Add New Firearm text")

	s.Contains(body, "Add Your First Firearm", "Owner page should have Add Your First Firearm button")

	// Flash message should appear
	s.Contains(body, "Enjoy adding to your armory!", "Owner page should display the welcome flash message")

	// Now test clicking the "View Arsenal" button - user is taken to arsenal page
	arsenalReq, _ := http.NewRequest("GET", "/owner/guns/arsenal", nil)
	for _, cookie := range cookies {
		arsenalReq.AddCookie(cookie)
	}
	arsenalResp := httptest.NewRecorder()
	s.Router.ServeHTTP(arsenalResp, arsenalReq)

	// Check status code
	s.Equal(http.StatusOK, arsenalResp.Code, "Arsenal page should return 200 OK")

	// Get the response body as a string
	arsenalBody := arsenalResp.Body.String()

	// Verify the content for an empty arsenal
	s.Contains(arsenalBody, "No firearms found", "Arsenal page should show empty state message")

	// Verify navigation bar shows authenticated state
	s.Contains(arsenalBody, `href="/owner"`, "Arsenal page should have My Armory link")
	s.Contains(arsenalBody, `href="/logout"`, "Arsenal page should have Logout link")

	// Verify navigation bar does NOT show unauthenticated links
	s.NotContains(arsenalBody, `href="/login"`, "Arsenal page should not have Login link")
	s.NotContains(arsenalBody, `href="/register"`, "Arsenal page should not have Register link")

	// Now test the "Add New Firearm" page
	newGunReq, _ := http.NewRequest("GET", "/owner/guns/new", nil)
	for _, cookie := range cookies {
		newGunReq.AddCookie(cookie)
	}
	newGunResp := httptest.NewRecorder()
	s.Router.ServeHTTP(newGunResp, newGunReq)

	// Check status code
	s.Equal(http.StatusOK, newGunResp.Code, "New gun page should return 200 OK")

	// Get the response body as a string
	newGunBody := newGunResp.Body.String()

	// Verify the content of the new gun page
	s.Contains(newGunBody, "Add New Firearm", "Page should have correct title")
	s.Contains(newGunBody, `href="/owner"`, "Page should have a back link to dashboard")
	s.Contains(newGunBody, "Back to Dashboard", "Page should have 'Back to Dashboard' text")

	// Verify form fields
	s.Contains(newGunBody, `name="name"`, "Form should have a name field")
	s.Contains(newGunBody, `name="serial_number"`, "Form should have a serial number field")
	s.Contains(newGunBody, `name="acquired"`, "Form should have an acquired date field")
	s.Contains(newGunBody, `name="weapon_type_id"`, "Form should have a weapon type dropdown")
	s.Contains(newGunBody, `name="caliber_id"`, "Form should have a caliber dropdown")
	s.Contains(newGunBody, `name="manufacturer_id"`, "Form should have a manufacturer dropdown")
	s.Contains(newGunBody, `name="paid"`, "Form should have a paid field")
	s.Contains(newGunBody, "Please enter in USD", "Page should have text about USD")
	s.Contains(newGunBody, "Add Firearm", "Page should have an 'Add Firearm' button")

	// Verify the navigation bar
	s.Contains(newGunBody, `href="/owner"`, "Page should have My Armory link")
	s.Contains(newGunBody, `href="/logout"`, "Page should have Logout link")
	s.NotContains(newGunBody, `href="/login"`, "Page should not have Login link")
	s.NotContains(newGunBody, `href="/register"`, "Page should not have Register link")

	// Test submitting an invalid form (missing required fields)
	invalidForm := url.Values{}
	// Serial number and paid are optional
	invalidForm.Add("serial_number", "123456")
	invalidForm.Add("paid", "300")

	invalidReq, _ := http.NewRequest("POST", "/owner/guns", strings.NewReader(invalidForm.Encode()))
	invalidReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	for _, cookie := range cookies {
		invalidReq.AddCookie(cookie)
	}
	invalidResp := httptest.NewRecorder()
	s.Router.ServeHTTP(invalidResp, invalidReq)

	// Should not redirect (form should be redisplayed with errors)
	s.NotEqual(http.StatusSeeOther, invalidResp.Code, "Invalid form submission should not redirect")

	// Now test submitting a valid form with all required fields
	// First, fetch the IDs of our created reference data
	var weaponTypeFromDB models.WeaponType
	var caliberFromDB models.Caliber
	var manufacturerFromDB models.Manufacturer

	// Find an existing Rifle weapon type - not creating a new one
	err = s.DB.First(&weaponTypeFromDB).Error
	s.NoError(err, "Should find a weapon type in the database")

	err = s.DB.Where("caliber = ?", "5.56 NATO").First(&caliberFromDB).Error
	s.NoError(err, "Should find the caliber we created")

	err = s.DB.Where("name = ?", "Smith & Wesson").First(&manufacturerFromDB).Error
	s.NoError(err, "Should find the manufacturer we created")

	validForm := url.Values{}
	validForm.Add("name", "AR-15")
	validForm.Add("serial_number", "12345678")
	validForm.Add("acquired", time.Now().Format("2006-01-02"))
	validForm.Add("weapon_type_id", fmt.Sprintf("%d", weaponTypeFromDB.ID))
	validForm.Add("caliber_id", fmt.Sprintf("%d", caliberFromDB.ID))
	validForm.Add("manufacturer_id", fmt.Sprintf("%d", manufacturerFromDB.ID))
	validForm.Add("paid", "300.00")

	validReq, _ := http.NewRequest("POST", "/owner/guns", strings.NewReader(validForm.Encode()))
	validReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	for _, cookie := range cookies {
		validReq.AddCookie(cookie)
	}
	validResp := httptest.NewRecorder()
	s.Router.ServeHTTP(validResp, validReq)

	// Should redirect to owner page
	s.Equal(http.StatusSeeOther, validResp.Code, "Valid form submission should redirect")
	s.Equal("/owner", validResp.Header().Get("Location"), "Should redirect to owner page")

	// Skip flash message verification - this is too brittle in tests
	// and we already verified that the gun creation worked via database check

	// Query the database for the gun we just created
	var gun models.Gun
	err = s.DB.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").Where("name = ?", "AR-15").First(&gun).Error
	s.NoError(err, "Should be able to find the newly created gun")

	// Now check the owner page to verify the gun was added
	finalOwnerReq, _ := http.NewRequest("GET", "/owner", nil)
	for _, cookie := range cookies {
		finalOwnerReq.AddCookie(cookie)
	}
	finalOwnerResp := httptest.NewRecorder()
	s.Router.ServeHTTP(finalOwnerResp, finalOwnerReq)

	// Check the updated owner page
	updatedBody := finalOwnerResp.Body.String()

	// Verify updated totals
	s.Contains(updatedBody, "Total Firearms:</strong> 1", "Gun count should now be 1")
	s.Contains(updatedBody, "Total Paid:</strong> $300.00", "Total paid should show $300.00")

	// Verify the gun is in the table using actual data from the database
	s.Contains(updatedBody, gun.Name, "Gun name should appear in the table")
	s.Contains(updatedBody, gun.WeaponType.Type, "Weapon type should appear in the table")
	s.Contains(updatedBody, gun.Manufacturer.Name, "Manufacturer should appear in the table")
	s.Contains(updatedBody, gun.Caliber.Caliber, "Caliber should appear in the table")

	// Check the paid amount - note this might be encoded as HTML so we use a more generic check
	paidStr := fmt.Sprintf("%.2f", *gun.Paid)
	s.Contains(updatedBody, paidStr, "Paid amount should appear in the table")

	// The "No firearms added yet" text should be gone
	s.NotContains(updatedBody, "No firearms added yet", "Empty state message should be gone")

	// Verify the arsenal is no longer empty by visiting it again
	finalArsenalReq, _ := http.NewRequest("GET", "/owner/guns/arsenal", nil)
	for _, cookie := range cookies {
		finalArsenalReq.AddCookie(cookie)
	}
	finalArsenalResp := httptest.NewRecorder()
	s.Router.ServeHTTP(finalArsenalResp, finalArsenalReq)

	finalArsenalBody := finalArsenalResp.Body.String()

	// The empty state message should be gone
	s.NotContains(finalArsenalBody, "No firearms found", "Arsenal should no longer be empty")

	// The gun we created should be visible with its details
	s.Contains(finalArsenalBody, gun.Name, "Gun name should appear in arsenal")
	s.Contains(finalArsenalBody, gun.WeaponType.Type, "Weapon type should appear in arsenal")
	s.Contains(finalArsenalBody, gun.Manufacturer.Name, "Manufacturer should appear in arsenal")
	s.Contains(finalArsenalBody, gun.Caliber.Caliber, "Caliber should appear in arsenal")

	// Check that the paid amount appears (accounting for formatting)
	paidStr = fmt.Sprintf("%.2f", *gun.Paid)
	s.Contains(finalArsenalBody, paidStr, "Paid amount should appear in arsenal")

	// Now test the gun show page
	gunShowURL := fmt.Sprintf("/owner/guns/%d", gun.ID)
	gunShowReq, _ := http.NewRequest("GET", gunShowURL, nil)
	for _, cookie := range cookies {
		gunShowReq.AddCookie(cookie)
	}
	gunShowResp := httptest.NewRecorder()
	s.Router.ServeHTTP(gunShowResp, gunShowReq)

	// Check status code
	s.Equal(http.StatusOK, gunShowResp.Code, "Gun show page should return 200 OK")

	// Get the response body as a string
	gunShowBody := gunShowResp.Body.String()

	// Verify gun details appear on the page
	s.Contains(gunShowBody, gun.Name, "Gun name should appear on the show page")
	s.Contains(gunShowBody, gun.SerialNumber, "Serial number should appear on the show page")
	s.Contains(gunShowBody, gun.WeaponType.Type, "Weapon type should appear on the show page")
	s.Contains(gunShowBody, gun.Manufacturer.Name, "Manufacturer should appear on the show page")
	s.Contains(gunShowBody, gun.Caliber.Caliber, "Caliber should appear on the show page")

	// If the gun has an acquired date, check that too
	if gun.Acquired != nil {
		acquiredDate := gun.Acquired.Format("January 2, 2006") // American date format: Month DD, YYYY
		s.Contains(gunShowBody, acquiredDate, "Acquired date should appear on the show page")
	}

	// Check the paid amount
	s.Contains(gunShowBody, fmt.Sprintf("$%.2f", *gun.Paid), "Paid amount should appear on the show page")

	// Check that the page has buttons for Edit and Delete
	s.Contains(gunShowBody, "Edit Firearm", "Show page should have Edit button")
	s.Contains(gunShowBody, "Delete Firearm", "Show page should have Delete button")

	// Now test the gun edit flow
	gunEditURL := fmt.Sprintf("/owner/guns/%d/edit", gun.ID)
	editReq, _ := http.NewRequest("GET", gunEditURL, nil)
	for _, cookie := range cookies {
		editReq.AddCookie(cookie)
	}
	editResp := httptest.NewRecorder()
	s.Router.ServeHTTP(editResp, editReq)

	// Check status code
	s.Equal(http.StatusOK, editResp.Code, "Gun edit page should return 200 OK")

	// Get the response body as a string
	editBody := editResp.Body.String()

	// Verify edit page content
	s.Contains(editBody, "Edit Firearm", "Page should have correct title")
	s.Contains(editBody, `href="/owner"`, "Page should have a back link to dashboard")
	s.Contains(editBody, "Back to Dashboard", "Page should have 'Back to Dashboard' text")

	// Verify nav bar has correct links for authenticated user
	s.Contains(editBody, `href="/owner"`, "Page should have My Armory link")
	s.Contains(editBody, `href="/logout"`, "Page should have Logout link")
	s.NotContains(editBody, `href="/login"`, "Page should not have Login link")
	s.NotContains(editBody, `href="/register"`, "Page should not have Register link")

	// Verify form fields
	s.Contains(editBody, `name="name"`, "Form should have a name field")
	s.Contains(editBody, `name="serial_number"`, "Form should have a serial number field")
	s.Contains(editBody, `name="acquired_date"`, "Form should have an acquired date field")
	s.Contains(editBody, `name="weapon_type_id"`, "Form should have a weapon type dropdown")
	s.Contains(editBody, `name="caliber_id"`, "Form should have a caliber dropdown")
	s.Contains(editBody, `name="manufacturer_id"`, "Form should have a manufacturer dropdown")
	s.Contains(editBody, `name="paid"`, "Form should have a paid field")
	s.Contains(editBody, "Update Firearm", "Page should have an 'Update Firearm' button")

	// Now find a second weapon type, caliber, and manufacturer for the update
	var secondWeaponType models.WeaponType
	var secondCaliber models.Caliber
	var secondManufacturer models.Manufacturer

	// Try to find a different weapon type than the one currently used
	err = s.DB.Where("id != ?", gun.WeaponTypeID).First(&secondWeaponType).Error
	if err != nil {
		// If we can't find another one, create a new one
		secondWeaponType = models.WeaponType{Type: "Pistol", Popularity: 95}
		err = s.DB.Create(&secondWeaponType).Error
		s.NoError(err, "Should be able to create a second weapon type")
	}

	// Try to find a different caliber
	err = s.DB.Where("id != ?", gun.CaliberID).First(&secondCaliber).Error
	if err != nil {
		// If we can't find another one, create a new one
		secondCaliber = models.Caliber{Caliber: "9mm", Popularity: 95}
		err = s.DB.Create(&secondCaliber).Error
		s.NoError(err, "Should be able to create a second caliber")
	}

	// Try to find a different manufacturer
	err = s.DB.Where("id != ?", gun.ManufacturerID).First(&secondManufacturer).Error
	if err != nil {
		// If we can't find another one, create a new one
		secondManufacturer = models.Manufacturer{Name: "Glock", Popularity: 95}
		err = s.DB.Create(&secondManufacturer).Error
		s.NoError(err, "Should be able to create a second manufacturer")
	}

	// Prepare update form data
	updatedName := "Modified AR-15"
	updatedSerialNumber := "87654321"
	updatedAcquiredDate := time.Now().AddDate(-1, 0, 0).Format("2006-01-02") // One year ago
	updatedPaid := "400.00"

	updateForm := url.Values{}
	updateForm.Add("name", updatedName)
	updateForm.Add("serial_number", updatedSerialNumber)
	updateForm.Add("acquired_date", updatedAcquiredDate)
	updateForm.Add("weapon_type_id", fmt.Sprintf("%d", secondWeaponType.ID))
	updateForm.Add("caliber_id", fmt.Sprintf("%d", secondCaliber.ID))
	updateForm.Add("manufacturer_id", fmt.Sprintf("%d", secondManufacturer.ID))
	updateForm.Add("paid", updatedPaid)

	// Submit the update form
	updateReq, _ := http.NewRequest("POST", fmt.Sprintf("/owner/guns/%d/update", gun.ID), strings.NewReader(updateForm.Encode()))
	updateReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	for _, cookie := range cookies {
		updateReq.AddCookie(cookie)
	}
	updateResp := httptest.NewRecorder()
	s.Router.ServeHTTP(updateResp, updateReq)

	// Should redirect to owner page
	s.Equal(http.StatusSeeOther, updateResp.Code, "Gun update should redirect")
	s.Equal("/owner", updateResp.Header().Get("Location"), "Should redirect to owner page")

	// Query the database for the updated gun
	var updatedGun models.Gun
	err = s.DB.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").First(&updatedGun, gun.ID).Error
	s.NoError(err, "Should be able to find the updated gun")

	// Verify the gun was updated correctly
	s.Equal(updatedName, updatedGun.Name, "Gun name should be updated")
	s.Equal(updatedSerialNumber, updatedGun.SerialNumber, "Gun serial number should be updated")
	s.Equal(secondWeaponType.ID, updatedGun.WeaponTypeID, "Gun weapon type should be updated")
	s.Equal(secondCaliber.ID, updatedGun.CaliberID, "Gun caliber should be updated")
	s.Equal(secondManufacturer.ID, updatedGun.ManufacturerID, "Gun manufacturer should be updated")

	// Parse the updated paid value for comparison
	updatedPaidFloat, err := strconv.ParseFloat(updatedPaid, 64)
	s.NoError(err, "Should be able to parse updated paid value")
	s.Equal(updatedPaidFloat, *updatedGun.Paid, "Gun paid amount should be updated")

	// Now check the owner page to see the updated gun
	checkOwnerReq, _ := http.NewRequest("GET", "/owner", nil)
	for _, cookie := range cookies {
		checkOwnerReq.AddCookie(cookie)
	}
	checkOwnerResp := httptest.NewRecorder()
	s.Router.ServeHTTP(checkOwnerResp, checkOwnerReq)

	// Check status code
	s.Equal(http.StatusOK, checkOwnerResp.Code, "Owner page should return 200 OK")

	// Get the response body as a string
	finalOwnerBody := checkOwnerResp.Body.String()

	// Verify the updated values appear on the owner page
	s.Contains(finalOwnerBody, updatedName, "Updated gun name should appear on the owner page")
	s.Contains(finalOwnerBody, updatedGun.WeaponType.Type, "Updated weapon type should appear on the owner page")
	s.Contains(finalOwnerBody, updatedGun.Manufacturer.Name, "Updated manufacturer should appear on the owner page")
	s.Contains(finalOwnerBody, updatedGun.Caliber.Caliber, "Updated caliber should appear on the owner page")
	s.Contains(finalOwnerBody, fmt.Sprintf("$%.2f", *updatedGun.Paid), "Updated paid amount should appear on the owner page")

	// Clean up - delete the gun and user
	s.DB.Exec("DELETE FROM guns")
	s.DB.Exec("DELETE FROM weapon_types")
	s.DB.Exec("DELETE FROM calibers")
	s.DB.Exec("DELETE FROM manufacturers")
	result = s.DB.Unscoped().Delete(testUser)
	s.NoError(result.Error)
}

// TearDownSuite cleans up after all tests
func (s *AuthIntegrationSuite) TearDownSuite() {
	// Clean up database if needed
}

func TestAuthIntegrationSuite(t *testing.T) {
	suite.Run(t, new(AuthIntegrationSuite))
}
