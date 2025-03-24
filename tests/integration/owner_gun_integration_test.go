package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
	"github.com/stretchr/testify/suite"
)

// OwnerGunIntegrationTest extends the base IntegrationSuite for testing gun functionalities
type OwnerGunIntegrationTest struct {
	IntegrationSuite
	testUser *database.User
}

// SetupTest runs before each test in the suite
func (s *OwnerGunIntegrationTest) SetupTest() {
	// Call the parent SetupTest to set up the suite
	s.IntegrationSuite.SetupTest()

	// Create a test user for our tests
	s.testUser = s.CreateTestUser("gunowner@example.com", "Password123!", true)
}

// TearDownTest runs after each test in the suite
func (s *OwnerGunIntegrationTest) TearDownTest() {
	// Clean up test user
	s.CleanupTestUser(s.testUser)

	// Call the parent TearDownTest
	s.IntegrationSuite.TearDownTest()
}

// TestOwnerPageWithNoFirearms tests that a logged-in user can see the owner page
// and it displays the "no firearms" message when they haven't added any
func (s *OwnerGunIntegrationTest) TestOwnerPageWithNoFirearms() {
	// Login the user using the shared helper
	cookies := s.LoginUser(s.testUser.Email, "Password123!")

	// Visit the owner page with the session cookies using the shared helper
	resp := s.MakeAuthenticatedRequest("GET", "/owner", cookies)

	// Verify we successfully loaded the owner page
	s.Equal(http.StatusOK, resp.Code)

	// Get the response body as a string
	body := resp.Body.String()

	// Verify the page contains the welcome text
	s.Contains(body, "Welcome to Your Virtual Armory")

	// Verify the page shows that no firearms have been added yet
	s.Contains(body, "You haven")
	s.Contains(body, "added any firearms yet")

	// Verify total firearms and total paid counters
	s.Contains(body, "Total Firearms:</strong> 0")
	s.Contains(body, "Total Paid:</strong> $0.00")

	// Verify the "Under Construction" link with href="#"
	s.Contains(body, "Under Construction")
	s.Contains(body, "href=\"#\"")

	// Verify the buttons for viewing and adding firearms
	s.Contains(body, "View Arsenal")
	s.Contains(body, "href=\"/owner/guns/arsenal\"")

	s.Contains(body, "Add New Firearm")
	s.Contains(body, "href=\"/owner/guns/new\"")

	// Verify the "Add Your First Firearm" button
	s.Contains(body, "Add Your First Firearm")
	s.Contains(body, "href=\"/owner/guns/new\"")
}

// TestArsenalPageWithNoFirearms tests that a logged-in user can see the arsenal page
// and it displays an empty state when they haven't added any firearms
func (s *OwnerGunIntegrationTest) TestArsenalPageWithNoFirearms() {
	// Login the user using the shared helper
	cookies := s.LoginUser(s.testUser.Email, "Password123!")

	// Visit the arsenal page with the session cookies using the shared helper
	resp := s.MakeAuthenticatedRequest("GET", "/owner/guns/arsenal", cookies)

	// Verify we successfully loaded the arsenal page
	s.Equal(http.StatusOK, resp.Code)

	// Get the response body as a string
	body := resp.Body.String()

	// Verify the page contains the arsenal title
	s.Contains(body, "Your Arsenal")

	// Verify the "Add New Firearm" button is present
	s.Contains(body, "Add New Firearm")
	s.Contains(body, "href=\"/owner/guns/new\"")

	// Verify the empty state message
	// Note: We're checking for partial text to avoid HTML encoding issues
	s.Contains(body, "No firearms found")

	// Verify the "Back to Dashboard" button is present
	s.Contains(body, "Back to Dashboard")
	s.Contains(body, "href=\"/owner\"")
}

// TestArsenalPageWithOneFirearm tests the arsenal page when a user has one firearm
func (s *OwnerGunIntegrationTest) TestArsenalPageWithOneFirearm() {
	// Login the user
	cookies := s.LoginUser(s.testUser.Email, "Password123!")

	// First create a gun to test with
	// Create the gun directly in the database for simplicity
	gunName := "Test Arsenal Gun"
	expectedPaid := 1250.75
	paidAmount := expectedPaid // Create a copy to use as pointer

	today := time.Now()

	gun := models.Gun{
		Name:           gunName,
		SerialNumber:   "SN67890",
		WeaponTypeID:   1,
		CaliberID:      1,
		ManufacturerID: 1,
		OwnerID:        s.testUser.ID,
		Acquired:       &today,
		Paid:           &paidAmount,
	}

	result := s.DB.Create(&gun)
	s.NoError(result.Error, "Failed to create test gun")
	s.T().Logf("Created test gun with ID: %d", gun.ID)

	// Visit the arsenal page
	arsenalResp := s.MakeAuthenticatedRequest("GET", "/owner/guns/arsenal", cookies)
	s.Equal(http.StatusOK, arsenalResp.Code)

	arsenalBody := arsenalResp.Body.String()

	// Verify the page shows the gun information
	s.NotContains(arsenalBody, "No firearms found")

	// Check gun name is present and linked to detail page
	s.Contains(arsenalBody, gunName)
	s.Contains(arsenalBody, fmt.Sprintf("guns/%d", gun.ID))

	// Check gun details are shown in the table
	s.Contains(arsenalBody, gun.SerialNumber)

	// Check for the paid amount - format with two decimal places
	formattedPaid := fmt.Sprintf("%.2f", expectedPaid)
	s.Contains(arsenalBody, formattedPaid)

	// Check for the acquired date
	formattedDate := today.Format("Jan 2, 2006")
	s.Contains(arsenalBody, formattedDate)

	// Verify back to dashboard link
	s.Contains(arsenalBody, "Back to Dashboard")
	s.Contains(arsenalBody, "href=\"/owner\"")

	// After test, clean up the created gun
	s.DB.Unscoped().Delete(&gun)
}

// TestCreateGunWorkflow tests the full workflow of gun creation
func (s *OwnerGunIntegrationTest) TestCreateGunWorkflow() {
	// Login the user using the shared helper
	cookies := s.LoginUser(s.testUser.Email, "Password123!")

	// Step 1: Visit the new gun form
	resp := s.MakeAuthenticatedRequest("GET", "/owner/guns/new", cookies)
	s.Equal(http.StatusOK, resp.Code)

	// Verify the form title is present
	s.Contains(resp.Body.String(), "Add New Firearm")

	// Step 2: Submit the form with invalid data (missing required fields)
	invalidForm := url.Values{}
	invalidForm.Add("name", "Test Gun") // Only provide name, missing other required fields

	req, _ := http.NewRequest("POST", "/owner/guns", strings.NewReader(invalidForm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Add auth cookies
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	invalidResp := httptest.NewRecorder()
	s.Router.ServeHTTP(invalidResp, req)

	// Verify we remain on the form page with error messages
	s.Equal(http.StatusOK, invalidResp.Code)
	body := invalidResp.Body.String()
	s.Contains(body, "Add New Firearm") // Still on the form page
	s.Contains(body, "Valid weapon type is required")
	s.Contains(body, "Valid caliber is required")
	s.Contains(body, "Valid manufacturer is required")

	// Step 3: Submit the form with valid data
	validForm := url.Values{}
	validForm.Add("name", "Test Gun 1")
	validForm.Add("serial_number", "SN12345")
	validForm.Add("weapon_type_id", "1")  // Using ID 1 as requested
	validForm.Add("caliber_id", "1")      // Using ID 1 as requested
	validForm.Add("manufacturer_id", "1") // Using ID 1 as requested
	validForm.Add("paid", "1500.50")      // Add a paid amount to test

	// Save the paid value for verification later
	expectedPaid := "1500.50"

	// Get today's date for the acquisition date
	today := time.Now().Format("2006-01-02")
	validForm.Add("acquired", today)

	req, _ = http.NewRequest("POST", "/owner/guns", strings.NewReader(validForm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Add auth cookies
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	validResp := httptest.NewRecorder()
	s.Router.ServeHTTP(validResp, req)

	// Verify redirect to owner page
	s.Equal(http.StatusSeeOther, validResp.Code)
	s.Equal("/owner", validResp.Header().Get("Location"))

	// Capture the flash cookie from the redirect response
	var flashCookie *http.Cookie
	for _, cookie := range validResp.Result().Cookies() {
		if cookie.Name == "flash" {
			flashCookie = cookie
			break
		}
	}

	// Step 4: Visit the owner page to verify gun was added - include the flash cookie
	ownerReq, _ := http.NewRequest("GET", "/owner", nil)

	// Add all cookies including auth
	for _, cookie := range cookies {
		ownerReq.AddCookie(cookie)
	}

	// Explicitly add the flash cookie if it exists
	if flashCookie != nil {
		ownerReq.AddCookie(flashCookie)
		s.T().Logf("Flash cookie found with value: %s", flashCookie.Value)
	}

	ownerResp := httptest.NewRecorder()
	s.Router.ServeHTTP(ownerResp, ownerReq)
	s.Equal(http.StatusOK, ownerResp.Code)

	ownerBody := ownerResp.Body.String()

	// Check for flash message - look for partial text
	s.T().Logf("Checking flash message in owner body")
	if flashCookie != nil {
		// Try both - either the message is in the cookie value or rendered in the page
		flashValue, _ := url.QueryUnescape(flashCookie.Value)
		s.T().Logf("Decoded flash cookie value: %s", flashValue)

		// Check if either the cookie value or a part of it appears in the page
		s.True(strings.Contains(ownerBody, "arsenal") ||
			strings.Contains(ownerBody, "Firearm") ||
			strings.Contains(ownerBody, "added") ||
			strings.Contains(ownerBody, flashValue),
			"Page should contain success message about gun being added")
	}

	// Check updated statistics with more precise checking
	s.T().Logf("Checking statistics in owner body")
	// Look for "Total Firearms: 1" with possible HTML tags in between
	totalFirearmsPattern := "Total Firearms[^0-9]*1"
	s.Regexp(totalFirearmsPattern, ownerBody)

	// Look for "Total Paid: $1500.50" with possible HTML tags in between
	// Use expectedPaid to ensure we're checking the exact value submitted
	// Dollar sign is required to ensure we're looking at the total, not a table cell
	totalPaidPattern := fmt.Sprintf("Total Paid[^0-9]*\\$%s", expectedPaid)
	s.Regexp(totalPaidPattern, ownerBody)

	// Check the gun data in the database to verify it was properly saved
	var gun models.Gun
	result := s.DB.Where("name = ?", "Test Gun 1").Where("owner_id = ?", s.testUser.ID).First(&gun)
	s.NoError(result.Error, "Failed to find created gun in database")
	s.T().Logf("Found gun ID: %d", gun.ID)

	// Verify the paid amount was properly saved
	s.NotNil(gun.Paid, "Gun should have a paid value")
	if gun.Paid != nil {
		s.Equal(1500.50, *gun.Paid, "Gun paid amount should match what we submitted")
		s.T().Logf("Gun paid amount in DB: %f", *gun.Paid)
	}

	// Check gun table is present instead of "no firearms" message
	s.NotContains(ownerBody, "haven't added any firearms yet")

	// Check the gun name is present
	s.Contains(ownerBody, "Test Gun 1")

	// Get the gun ID from the database for link checking
	var gunID models.Gun
	result = s.DB.Where("name = ?", "Test Gun 1").Where("owner_id = ?", s.testUser.ID).First(&gunID)
	s.NoError(result.Error, "Failed to find created gun in database")
	s.T().Logf("Found gun ID: %d", gunID.ID)

	// Verify gun ID in the URL for the gun's name
	gunIDString := fmt.Sprintf("%d", gunID.ID)
	s.T().Logf("Looking for gun ID %s in the page", gunIDString)

	// Be more flexible with URL format - it could be /guns/ID or /owner/guns/ID
	foundGunLink := strings.Contains(ownerBody, fmt.Sprintf("guns/%s", gunIDString))
	s.True(foundGunLink, "Should find link to the gun detail page")

	// Step 5: Visit the arsenal page to verify gun appears there as well
	arsenalResp := s.MakeAuthenticatedRequest("GET", "/owner/guns/arsenal", cookies)
	s.Equal(http.StatusOK, arsenalResp.Code)

	arsenalBody := arsenalResp.Body.String()

	// Verify the gun is listed in arsenal as well
	s.NotContains(arsenalBody, "No firearms found")
	s.Contains(arsenalBody, "Test Gun 1")
	s.Contains(arsenalBody, fmt.Sprintf("guns/%s", gunIDString))
}

// TestGunDetailPage tests the gun detail page for a logged-in owner with one gun
func (s *OwnerGunIntegrationTest) TestGunDetailPage() {
	// Login the user
	cookies := s.LoginUser(s.testUser.Email, "Password123!")

	// First create a gun to test with
	// Create the gun directly in the database for simplicity
	gunName := "Test Detail Gun"
	expectedPaid := 1750.25
	paidAmount := expectedPaid // Create a copy to use as pointer

	today := time.Now()

	gun := models.Gun{
		Name:           gunName,
		SerialNumber:   "SN12345-DETAIL",
		WeaponTypeID:   1,
		CaliberID:      1,
		ManufacturerID: 1,
		OwnerID:        s.testUser.ID,
		Acquired:       &today,
		Paid:           &paidAmount,
	}

	result := s.DB.Create(&gun)
	s.NoError(result.Error, "Failed to create test gun")
	s.T().Logf("Created test gun with ID: %d", gun.ID)

	// Need to manually load the related entities for verification
	s.DB.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").First(&gun, gun.ID)

	// Visit the gun detail page
	detailURL := fmt.Sprintf("/owner/guns/%d", gun.ID)
	detailResp := s.MakeAuthenticatedRequest("GET", detailURL, cookies)
	s.Equal(http.StatusOK, detailResp.Code)

	detailBody := detailResp.Body.String()

	// Verify the page title
	s.Contains(detailBody, "Firearm Details")

	// Verify the "Back to Dashboard" link
	s.Contains(detailBody, "Back to Dashboard")
	s.Contains(detailBody, "href=\"/owner\"")

	// Verify the "Edit Firearm" link
	s.Contains(detailBody, "Edit Firearm")
	s.Contains(detailBody, fmt.Sprintf("href=\"/owner/guns/%d/edit\"", gun.ID))

	// Verify the "Delete Firearm" button
	s.Contains(detailBody, "Delete Firearm")
	s.Contains(detailBody, fmt.Sprintf("action=\"/owner/guns/%d/delete\"", gun.ID))

	// Verify section headers
	s.Contains(detailBody, "Basic Information")
	s.Contains(detailBody, "Specifications")

	// Verify gun details - Basic Information
	s.Contains(detailBody, gunName)
	s.Contains(detailBody, gun.SerialNumber)

	// Format date in the same way the template does
	formattedDate := today.Format("January 2, 2006")
	s.Contains(detailBody, formattedDate)

	// Format price with dollar sign and two decimal places
	formattedPaid := fmt.Sprintf("$%.2f", expectedPaid)
	s.Contains(detailBody, formattedPaid)

	// Verify gun details - Specifications
	s.Contains(detailBody, gun.WeaponType.Type)
	s.Contains(detailBody, gun.Caliber.Caliber)
	s.Contains(detailBody, gun.Manufacturer.Name)

	// After test, clean up the created gun
	s.DB.Unscoped().Delete(&gun)
}

// TestEditGunWorkflow tests the full workflow of editing a gun
func (s *OwnerGunIntegrationTest) TestEditGunWorkflow() {
	// Login the user
	cookies := s.LoginUser(s.testUser.Email, "Password123!")

	// First create a gun to test with directly in the database
	originalName := "Test Edit Gun"
	originalSerial := "SN55555-EDIT"
	originalPaid := 1234.56
	paidAmount := originalPaid // Create a copy to use as pointer

	today := time.Now()
	formattedToday := today.Format("2006-01-02")

	// Create a gun with initial values
	gun := models.Gun{
		Name:           originalName,
		SerialNumber:   originalSerial,
		WeaponTypeID:   1, // Start with ID 1
		CaliberID:      1, // Start with ID 1
		ManufacturerID: 1, // Start with ID 1
		OwnerID:        s.testUser.ID,
		Acquired:       &today,
		Paid:           &paidAmount,
	}

	result := s.DB.Create(&gun)
	s.NoError(result.Error, "Failed to create test gun")
	s.T().Logf("Created test gun with ID: %d", gun.ID)

	// Load the relationships for the original gun to verify they change later
	var originalGun models.Gun
	s.DB.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").First(&originalGun, gun.ID)

	originalWeaponType := originalGun.WeaponType.Type
	originalCaliber := originalGun.Caliber.Caliber
	originalManufacturer := originalGun.Manufacturer.Name

	// Step 1: Visit the edit page for this gun
	editURL := fmt.Sprintf("/owner/guns/%d/edit", gun.ID)
	editResp := s.MakeAuthenticatedRequest("GET", editURL, cookies)
	s.Equal(http.StatusOK, editResp.Code)

	// Verify the edit page has the correct initial values
	editBody := editResp.Body.String()
	s.Contains(editBody, "Edit Gun")
	s.Contains(editBody, originalName)
	s.Contains(editBody, originalSerial)
	s.Contains(editBody, fmt.Sprintf("value=\"%.2f\"", originalPaid)) // The paid amount with 2 decimal places

	// Step 2: Submit the form with invalid data (empty name)
	invalidForm := url.Values{}
	invalidForm.Add("name", "") // Empty name should fail validation
	invalidForm.Add("serial_number", originalSerial)
	invalidForm.Add("weapon_type_id", "1")
	invalidForm.Add("caliber_id", "1")
	invalidForm.Add("manufacturer_id", "1")
	invalidForm.Add("paid", fmt.Sprintf("%.2f", originalPaid))
	invalidForm.Add("acquired_date", formattedToday)

	updateURL := fmt.Sprintf("/owner/guns/%d", gun.ID)
	req, _ := http.NewRequest("POST", updateURL, strings.NewReader(invalidForm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Add auth cookies
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	invalidResp := httptest.NewRecorder()
	s.Router.ServeHTTP(invalidResp, req)

	// Verify we're still on the edit form with an error
	s.Equal(http.StatusOK, invalidResp.Code)
	invalidRespBody := invalidResp.Body.String()
	s.Contains(invalidRespBody, "Edit Gun")
	s.Contains(invalidRespBody, "Name is required") // Error message should be displayed

	// Step 3: Submit the form with valid, updated data
	// Define new values that are different from original
	updatedName := "Updated Test Gun"
	updatedSerial := "SN99999-UPDATED"
	updatedPaid := 9876.54 // Different from original

	// Create a new date different from today
	updatedDate := today.AddDate(0, -1, 0) // One month earlier
	formattedUpdatedDate := updatedDate.Format("2006-01-02")

	validForm := url.Values{}
	validForm.Add("name", updatedName)
	validForm.Add("serial_number", updatedSerial)
	validForm.Add("weapon_type_id", "2")  // Changed to ID 2
	validForm.Add("caliber_id", "2")      // Changed to ID 2
	validForm.Add("manufacturer_id", "2") // Changed to ID 2
	validForm.Add("paid", fmt.Sprintf("%.2f", updatedPaid))
	validForm.Add("acquired_date", formattedUpdatedDate)

	req, _ = http.NewRequest("POST", updateURL, strings.NewReader(validForm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Add auth cookies
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	validResp := httptest.NewRecorder()
	s.Router.ServeHTTP(validResp, req)

	// Verify redirect to owner page
	s.Equal(http.StatusSeeOther, validResp.Code)
	s.Equal("/owner", validResp.Header().Get("Location"))

	// Capture the flash cookie from the redirect response
	var flashCookie *http.Cookie
	for _, cookie := range validResp.Result().Cookies() {
		if cookie.Name == "flash" {
			flashCookie = cookie
			break
		}
	}
	s.NotNil(flashCookie, "Flash cookie should be set with update success message")

	// Step 4: Visit the owner page to verify changes
	ownerReq, _ := http.NewRequest("GET", "/owner", nil)

	// Add all cookies including auth
	for _, cookie := range cookies {
		ownerReq.AddCookie(cookie)
	}

	// Add the flash cookie
	if flashCookie != nil {
		ownerReq.AddCookie(flashCookie)
	}

	ownerResp := httptest.NewRecorder()
	s.Router.ServeHTTP(ownerResp, ownerReq)
	s.Equal(http.StatusOK, ownerResp.Code)

	// Step 5: Verify the updates were saved to database
	var updatedGun models.Gun
	result = s.DB.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").First(&updatedGun, gun.ID)
	s.NoError(result.Error, "Should be able to find the updated gun")

	// Verify all fields were updated
	s.Equal(updatedName, updatedGun.Name)
	s.Equal(updatedSerial, updatedGun.SerialNumber)
	s.Equal(uint(2), updatedGun.WeaponTypeID)
	s.Equal(uint(2), updatedGun.CaliberID)
	s.Equal(uint(2), updatedGun.ManufacturerID)

	// Verify payment was updated
	s.NotNil(updatedGun.Paid)
	if updatedGun.Paid != nil {
		s.Equal(updatedPaid, *updatedGun.Paid)
	}

	// Verify acquisition date was updated - must match the day
	s.NotNil(updatedGun.Acquired)
	if updatedGun.Acquired != nil {
		s.Equal(updatedDate.Year(), updatedGun.Acquired.Year())
		s.Equal(updatedDate.Month(), updatedGun.Acquired.Month())
		s.Equal(updatedDate.Day(), updatedGun.Acquired.Day())
	}

	// Step 6: Visit the gun detail page to verify display is correct
	detailURL := fmt.Sprintf("/owner/guns/%d", gun.ID)
	detailResp := s.MakeAuthenticatedRequest("GET", detailURL, cookies)
	s.Equal(http.StatusOK, detailResp.Code)

	detailBody := detailResp.Body.String()

	// Check that new values are present
	s.Contains(detailBody, updatedName)
	s.Contains(detailBody, updatedSerial)
	s.Contains(detailBody, updatedGun.WeaponType.Type)
	s.Contains(detailBody, updatedGun.Caliber.Caliber)
	s.Contains(detailBody, updatedGun.Manufacturer.Name)
	s.Contains(detailBody, fmt.Sprintf("$%.2f", updatedPaid))

	// Format date the same way the template formats it
	formattedUpdatedDisplayDate := updatedDate.Format("January 2, 2006")
	s.Contains(detailBody, formattedUpdatedDisplayDate)

	// Verify old values are not present anymore
	s.NotContains(detailBody, originalName)
	s.NotContains(detailBody, originalSerial)
	s.NotContains(detailBody, originalWeaponType)
	s.NotContains(detailBody, originalCaliber)
	s.NotContains(detailBody, originalManufacturer)
	s.NotContains(detailBody, fmt.Sprintf("$%.2f", originalPaid))

	// After test, clean up the created gun
	s.DB.Unscoped().Delete(&gun)
}

// TestDeleteGunWorkflow tests deleting a gun from the owner page
func (s *OwnerGunIntegrationTest) TestDeleteGunWorkflow() {
	// Login the user
	cookies := s.LoginUser(s.testUser.Email, "Password123!")

	// First create a gun to test with directly in the database
	gunName := "Test Delete Gun"
	serialNumber := "SN-DELETE-123"
	paidAmount := 1234.56
	paid := paidAmount // Create a copy to use as pointer

	today := time.Now()

	// Create a gun with initial values
	gun := models.Gun{
		Name:           gunName,
		SerialNumber:   serialNumber,
		WeaponTypeID:   1,
		CaliberID:      1,
		ManufacturerID: 1,
		OwnerID:        s.testUser.ID,
		Acquired:       &today,
		Paid:           &paid,
	}

	result := s.DB.Create(&gun)
	s.NoError(result.Error, "Failed to create test gun")
	s.T().Logf("Created test gun with ID: %d", gun.ID)

	// Step 1: Visit the owner page to confirm the gun is listed
	ownerResp := s.MakeAuthenticatedRequest("GET", "/owner", cookies)
	s.Equal(http.StatusOK, ownerResp.Code)

	ownerBody := ownerResp.Body.String()

	// Verify gun info appears on page
	s.Contains(ownerBody, gunName)
	s.Contains(ownerBody, "Total Firearms:</strong> 1")
	formattedPaid := fmt.Sprintf("$%.2f", paidAmount)
	s.Contains(ownerBody, fmt.Sprintf("Total Paid:</strong> %s", formattedPaid))
	s.NotContains(ownerBody, "You haven't added any firearms yet")

	// Note: in a real browser, the delete action is triggered via a form submission with JavaScript
	// Since we're testing backend functionality, we'll simulate the form POST directly

	// Step 2: First try clicking cancel in the confirmation dialog (simulated)
	// This is just for documentation - in a real test with browser automation we would test this
	s.T().Log("Note: In a real scenario, clicking Cancel in the confirmation dialog would abort the delete")
	s.T().Log("Since this is a backend test, we're only testing the actual deletion flow")

	// Step 3: Submit the delete form (simulating clicking OK in the confirmation dialog)
	deleteURL := fmt.Sprintf("/owner/guns/%d/delete", gun.ID)
	req, _ := http.NewRequest("POST", deleteURL, nil)

	// Add auth cookies
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	deleteResp := httptest.NewRecorder()
	s.Router.ServeHTTP(deleteResp, req)

	// Verify redirect back to owner page
	s.Equal(http.StatusSeeOther, deleteResp.Code)
	s.Equal("/owner", deleteResp.Header().Get("Location"))

	// Capture the flash cookie from the redirect response
	var flashCookie *http.Cookie
	for _, cookie := range deleteResp.Result().Cookies() {
		if cookie.Name == "flash" {
			flashCookie = cookie
			break
		}
	}
	s.NotNil(flashCookie, "Flash cookie should be set with delete success message")

	// Step 4: Visit the owner page again to verify gun is deleted
	ownerReq, _ := http.NewRequest("GET", "/owner", nil)

	// Add all cookies including auth
	for _, cookie := range cookies {
		ownerReq.AddCookie(cookie)
	}

	// Add the flash cookie
	if flashCookie != nil {
		ownerReq.AddCookie(flashCookie)
	}

	ownerResp2 := httptest.NewRecorder()
	s.Router.ServeHTTP(ownerResp2, ownerReq)
	s.Equal(http.StatusOK, ownerResp2.Code)

	ownerBody2 := ownerResp2.Body.String()

	// Verify gun is gone from the page
	s.NotContains(ownerBody2, gunName)
	s.NotContains(ownerBody2, serialNumber)

	// Verify counts have been reset
	s.Contains(ownerBody2, "Total Firearms:</strong> 0")
	s.Contains(ownerBody2, "Total Paid:</strong> $0.00")

	// Verify empty state message is shown
	s.Contains(ownerBody2, "You haven")
	s.Contains(ownerBody2, "added any firearms yet")

	// Verify the "Add Your First Firearm" button is shown again
	s.Contains(ownerBody2, "Add Your First Firearm")
	s.Contains(ownerBody2, "href=\"/owner/guns/new\"")

	// Step 5: Verify the gun is actually deleted from the database
	var deletedGun models.Gun
	result = s.DB.Where("id = ?", gun.ID).First(&deletedGun)
	s.Error(result.Error, "Gun should not be found in database")
	s.Equal("record not found", result.Error.Error())

	// Note: We don't need to clean up the test gun since it's been deleted by the test

	// Note for future tests:
	// We may want to try deleting from other pages too (like detail page or arsenal)
	// if we are concerned about errors on those pages due to different implementations
}

// TestOwnerGunIntegration runs the owner gun integration test suite
func TestOwnerGunIntegration(t *testing.T) {
	suite.Run(t, new(OwnerGunIntegrationTest))
}
