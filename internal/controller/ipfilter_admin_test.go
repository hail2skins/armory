package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/services/stripe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockIPFilterService is a mock implementation of the stripe.IPFilterService interface
type MockIPFilterService struct {
	mock.Mock
}

// FetchIPRanges is a mocked implementation of the FetchIPRanges method
func (m *MockIPFilterService) FetchIPRanges() error {
	args := m.Called()
	return args.Error(0)
}

// IsStripeIP is a mocked implementation of the IsStripeIP method
func (m *MockIPFilterService) IsStripeIP(ip string) bool {
	args := m.Called(ip)
	return args.Bool(0)
}

// StartBackgroundRefresh is a mocked implementation of the StartBackgroundRefresh method
func (m *MockIPFilterService) StartBackgroundRefresh(stop chan struct{}) {
	m.Called(stop)
}

// Middleware is a mocked implementation of the Middleware method
func (m *MockIPFilterService) Middleware() gin.HandlerFunc {
	args := m.Called()
	return args.Get(0).(gin.HandlerFunc)
}

// GetLastUpdateStatus is a mocked implementation of the GetLastUpdateStatus method
func (m *MockIPFilterService) GetLastUpdateStatus() stripe.UpdateStatus {
	args := m.Called()
	return args.Get(0).(stripe.UpdateStatus)
}

func TestNewIPFilterAdminController(t *testing.T) {
	// Create a mock service
	mockService := new(MockIPFilterService)

	// Create a controller with the mock
	controller := NewIPFilterAdminController(mockService)

	// Assert that the controller is created correctly
	require.NotNil(t, controller, "Controller should not be nil")
	assert.Equal(t, mockService, controller.ipFilterService, "Service should be correctly set")
}

func TestRefreshIPRanges(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create test cases
	testCases := []struct {
		name           string
		setupMock      func(*MockIPFilterService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			setupMock: func(m *MockIPFilterService) {
				m.On("FetchIPRanges").Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"IP ranges refreshed successfully"}`,
		},
		{
			name: "Error",
			setupMock: func(m *MockIPFilterService) {
				m.On("FetchIPRanges").Return(assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"Failed to refresh IP ranges: assert.AnError general error for testing"}`,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock service
			mockService := new(MockIPFilterService)

			// Setup the mock
			tc.setupMock(mockService)

			// Create a controller with the mock
			controller := NewIPFilterAdminController(mockService)

			// Create a test recorder and context
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Call the method
			controller.RefreshIPRanges(c)

			// Assert the response
			assert.Equal(t, tc.expectedStatus, w.Code, "Status code should match expected")
			assert.JSONEq(t, tc.expectedBody, w.Body.String(), "Response body should match expected")

			// Verify expectations
			mockService.AssertExpectations(t)
		})
	}
}

func TestGetIPRangeStatus(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Save and restore environment variables
	origEnabled := os.Getenv("STRIPE_IP_FILTER_ENABLED")
	origSecret := os.Getenv("STRIPE_OVERRIDE_SECRET")
	defer func() {
		os.Setenv("STRIPE_IP_FILTER_ENABLED", origEnabled)
		os.Setenv("STRIPE_OVERRIDE_SECRET", origSecret)
	}()

	// Set test environment variables
	os.Setenv("STRIPE_IP_FILTER_ENABLED", "true")
	os.Setenv("STRIPE_OVERRIDE_SECRET", "test-secret")

	// Create a mock status
	mockStatus := stripe.UpdateStatus{
		LastUpdate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		NumRanges:  10,
		Failed:     false,
	}

	// Create a mock service
	mockService := new(MockIPFilterService)
	mockService.On("GetLastUpdateStatus").Return(mockStatus)

	// Create a controller with the mock
	controller := NewIPFilterAdminController(mockService)

	// Create a test recorder and context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Call the method
	controller.GetIPRangeStatus(c)

	// Assert the response
	assert.Equal(t, http.StatusOK, w.Code, "Status code should be 200")

	// Expected body with formatted time
	expectedTime := "2023-01-01T00:00:00Z" // RFC3339 format

	// Check that the JSON response contains expected values
	assert.Contains(t, w.Body.String(), `"last_update":"`+expectedTime+`"`)
	assert.Contains(t, w.Body.String(), `"num_ranges":10`)
	assert.Contains(t, w.Body.String(), `"last_failed":false`)
	assert.Contains(t, w.Body.String(), `"is_enabled":true`)
	assert.Contains(t, w.Body.String(), `"override_set":true`)

	// Verify expectations
	mockService.AssertExpectations(t)
}

func TestToggleIPFilter(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Save and restore environment variable
	origEnabled := os.Getenv("STRIPE_IP_FILTER_ENABLED")
	defer os.Setenv("STRIPE_IP_FILTER_ENABLED", origEnabled)

	// Test cases for different initial states
	testCases := []struct {
		name          string
		initialValue  string
		expectedValue string
	}{
		{
			name:          "Enable",
			initialValue:  "false",
			expectedValue: "true",
		},
		{
			name:          "Disable",
			initialValue:  "true",
			expectedValue: "false",
		},
		{
			name:          "Empty to Enabled",
			initialValue:  "",
			expectedValue: "true",
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set initial environment value
			os.Setenv("STRIPE_IP_FILTER_ENABLED", tc.initialValue)

			// Create a mock service
			mockService := new(MockIPFilterService)

			// Create a controller with the mock
			controller := NewIPFilterAdminController(mockService)

			// Create a test recorder and context
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Call the method
			controller.ToggleIPFilter(c)

			// Assert the response
			assert.Equal(t, http.StatusOK, w.Code, "Status code should be 200")

			// Check the environment variable was updated
			assert.Equal(t, tc.expectedValue, os.Getenv("STRIPE_IP_FILTER_ENABLED"), "Environment variable should be updated")

			// Expected message in response
			expectedEnabledStr := "true"
			if tc.expectedValue == "false" {
				expectedEnabledStr = "false"
			}
			assert.Contains(t, w.Body.String(), `"enabled":`+expectedEnabledStr)
			assert.Contains(t, w.Body.String(), `"message":"IP filter toggled successfully"`)
		})
	}
}

func TestIsIPAllowed(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Test cases
	testCases := []struct {
		name           string
		ip             string
		setupMock      func(*MockIPFilterService)
		expectedStatus int
		expectedJSON   map[string]interface{}
	}{
		{
			name: "IP Allowed",
			ip:   "192.0.2.1",
			setupMock: func(m *MockIPFilterService) {
				m.On("IsStripeIP", "192.0.2.1").Return(true)

				// Mock status with non-zero number of IPs
				m.On("GetLastUpdateStatus").Return(stripe.UpdateStatus{
					NumRanges: 5,
				})
			},
			expectedStatus: http.StatusOK,
			expectedJSON: map[string]interface{}{
				"ip":       "192.0.2.1",
				"allowed":  true,
				"num_ips":  5,
				"is_valid": true,
			},
		},
		{
			name: "IP Not Allowed",
			ip:   "192.0.3.1",
			setupMock: func(m *MockIPFilterService) {
				m.On("IsStripeIP", "192.0.3.1").Return(false)

				// Mock status
				m.On("GetLastUpdateStatus").Return(stripe.UpdateStatus{
					NumRanges: 5,
				})
			},
			expectedStatus: http.StatusOK,
			expectedJSON: map[string]interface{}{
				"ip":       "192.0.3.1",
				"allowed":  false,
				"num_ips":  5,
				"is_valid": true,
			},
		},
		{
			name:           "Missing IP",
			ip:             "",
			setupMock:      func(m *MockIPFilterService) {},
			expectedStatus: http.StatusBadRequest,
			expectedJSON: map[string]interface{}{
				"error": "IP parameter is required",
			},
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock service
			mockService := new(MockIPFilterService)

			// Setup the mock
			tc.setupMock(mockService)

			// Create a controller with the mock
			controller := NewIPFilterAdminController(mockService)

			// Create a test request
			req, _ := http.NewRequest("GET", "/admin/ip-check?ip="+tc.ip, nil)

			// Create a test recorder and context
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Set query parameters
			if tc.ip != "" {
				c.Request.URL.RawQuery = "ip=" + tc.ip
			}

			// Call the method
			controller.IsIPAllowed(c)

			// Assert the response status
			assert.Equal(t, tc.expectedStatus, w.Code, "Status code should match expected")

			// For successful responses, check JSON values
			if tc.expectedStatus == http.StatusOK {
				for key, value := range tc.expectedJSON {
					switch v := value.(type) {
					case string:
						assert.Contains(t, w.Body.String(), `"`+key+`":"`+v+`"`)
					case bool:
						if v {
							assert.Contains(t, w.Body.String(), `"`+key+`":true`)
						} else {
							assert.Contains(t, w.Body.String(), `"`+key+`":false`)
						}
					case int:
						assert.Contains(t, w.Body.String(), fmt.Sprintf(`"%s":%d`, key, v))
					}
				}
			} else {
				// For error responses, check the error message
				assert.Contains(t, w.Body.String(), tc.expectedJSON["error"].(string))
			}

			// Verify expectations
			mockService.AssertExpectations(t)
		})
	}
}
