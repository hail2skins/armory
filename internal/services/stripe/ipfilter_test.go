package stripe

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockHTTPClient is a mock implementation of the HTTPClient interface
type MockHTTPClient struct {
	mock.Mock
}

// Do is a mocked implementation of the Do method
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

// TestNewIPFilterService tests the creation of a new IP filter service
func TestNewIPFilterService(t *testing.T) {
	// Create a mock HTTP client
	mockClient := new(MockHTTPClient)

	// Call the function
	service := NewIPFilterService(mockClient)

	// Assertions
	require.NotNil(t, service, "Service should not be nil")

	// Check if the correct type is returned
	_, ok := service.(*ipFilterService)
	assert.True(t, ok, "Service should be of type *ipFilterService")

	// Check if the HTTP client is correctly set
	ipService := service.(*ipFilterService)
	assert.Equal(t, mockClient, ipService.httpClient, "HTTP client should be correctly set")

	// Check if fields are properly initialized
	assert.NotNil(t, ipService.ipRanges, "IP ranges map should be initialized")
	assert.NotNil(t, ipService.mutex, "Mutex should be initialized")
	assert.False(t, ipService.lastUpdateFailed, "lastUpdateFailed should be initialized to false")
}

// Mock response for webhook IPs
func createMockWebhookResponse() string {
	return `{"WEBHOOKS": ["192.0.2.1", "192.0.2.2", "2001:db8::1"]}`
}

// Mock response for API IPs
func createMockAPIResponse() string {
	return `{"API": ["192.0.2.3", "192.0.2.4", "2001:db8::2"]}`
}

// Mock response for Armada/Gator IPs
func createMockArmadaGatorResponse() string {
	return `{"ARMADA_GATOR": ["192.0.2.5/24", "2001:db8:1::/48"]}`
}

// TestFetchIPRanges tests the FetchIPRanges method
func TestFetchIPRanges(t *testing.T) {
	// Create a mock HTTP client
	mockClient := new(MockHTTPClient)

	// Set up expectations for all three sources
	webhookResp := httptest.NewRecorder()
	webhookResp.WriteString(createMockWebhookResponse())

	apiResp := httptest.NewRecorder()
	apiResp.WriteString(createMockAPIResponse())

	armadaResp := httptest.NewRecorder()
	armadaResp.WriteString(createMockArmadaGatorResponse())

	// Set expectations on the mock client - order matters here
	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "ips_webhooks.json")
	})).Return(webhookResp.Result(), nil)

	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "ips_api.json")
	})).Return(apiResp.Result(), nil)

	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "ips_armada_gator.json")
	})).Return(armadaResp.Result(), nil)

	// Create the service
	service := NewIPFilterService(mockClient)

	// Call the method
	err := service.FetchIPRanges()

	// Assertions
	require.NoError(t, err, "FetchIPRanges should not return an error")

	ipService := service.(*ipFilterService)

	// Should have 6 IPs (3 from webhooks, 3 from API, possibly more from CIDR blocks in Armada/Gator)
	// Due to CIDR block expansion, we just check that we have at least the direct IPs
	assert.GreaterOrEqual(t, len(ipService.ipRanges), 6, "Service should have at least 6 IP ranges")
	assert.False(t, ipService.lastUpdateFailed, "lastUpdateFailed should be false")

	// Verify that the mock was called 3 times (once for each source)
	mockClient.AssertNumberOfCalls(t, "Do", 3)
}

// TestFetchIPRanges_PartialFailure tests handling partial failures in IP range fetching
func TestFetchIPRanges_PartialFailure(t *testing.T) {
	// Create a mock HTTP client
	mockClient := new(MockHTTPClient)

	// Set up expectations for all three sources
	webhookResp := httptest.NewRecorder()
	webhookResp.WriteString(createMockWebhookResponse())

	// Make the API request fail
	errorResp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       http.NoBody,
	}

	armadaResp := httptest.NewRecorder()
	armadaResp.WriteString(createMockArmadaGatorResponse())

	// Set expectations on the mock client - order matters here
	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "ips_webhooks.json")
	})).Return(webhookResp.Result(), nil)

	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "ips_api.json")
	})).Return(errorResp, nil)

	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "ips_armada_gator.json")
	})).Return(armadaResp.Result(), nil)

	// Create the service
	service := NewIPFilterService(mockClient)

	// Call the method
	err := service.FetchIPRanges()

	// Assertions - should still succeed as long as at least one source works
	require.NoError(t, err, "FetchIPRanges should not return an error with partial failures")

	ipService := service.(*ipFilterService)

	// Should have at least 3 IPs (3 from webhooks, potentially more from CIDR blocks)
	assert.GreaterOrEqual(t, len(ipService.ipRanges), 3, "Service should have at least 3 IP ranges")

	// lastUpdateFailed should be true since we had a partial failure
	assert.True(t, ipService.lastUpdateFailed, "lastUpdateFailed should be true with partial failures")

	// Verify that the mock was called 3 times (once for each source)
	mockClient.AssertNumberOfCalls(t, "Do", 3)
}

// TestFetchIPRanges_AllFailures tests error handling when all sources fail
func TestFetchIPRanges_AllFailures(t *testing.T) {
	// Create a mock HTTP client
	mockClient := new(MockHTTPClient)

	// Make all requests fail
	errorResp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       http.NoBody,
	}

	// Set expectations on the mock client for all three sources
	mockClient.On("Do", mock.Anything).Return(errorResp, nil)

	// Create the service
	service := NewIPFilterService(mockClient)
	ipService := service.(*ipFilterService)

	// Call the method
	err := service.FetchIPRanges()

	// Assertions - should fail since all sources failed
	require.Error(t, err, "FetchIPRanges should return an error when all sources fail")
	assert.True(t, ipService.lastUpdateFailed, "lastUpdateFailed should be true")

	// Verify that the mock was called 3 times (once for each source)
	mockClient.AssertNumberOfCalls(t, "Do", 3)
}

// TestIsStripeIP tests the IsStripeIP method
func TestIsStripeIP(t *testing.T) {
	// Create a mock HTTP client
	mockClient := new(MockHTTPClient)

	// Create a service with some predefined IP ranges
	service := NewIPFilterService(mockClient)
	ipService := service.(*ipFilterService)

	// Add test IP ranges to the service
	ipService.mutex.Lock()
	ipService.ipRanges = make(map[string]*net.IPNet)

	// Add a test IPv4 range
	_, network, _ := net.ParseCIDR("192.0.2.0/24")
	ipService.ipRanges["192.0.2.0/24"] = network

	// Add a test IPv6 range
	_, network, _ = net.ParseCIDR("2001:db8::/32")
	ipService.ipRanges["2001:db8::/32"] = network
	ipService.mutex.Unlock()

	// Test cases
	testCases := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"IPv4 in range", "192.0.2.10", true},
		{"IPv4 out of range", "192.0.3.10", false},
		{"IPv6 in range", "2001:db8::1", true},
		{"IPv6 out of range", "2001:db9::1", false},
		{"Invalid IP", "not-an-ip", false},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := service.IsStripeIP(tc.ip)
			assert.Equal(t, tc.expected, result, "IsStripeIP(%s) should return %v", tc.ip, tc.expected)
		})
	}
}

// TestIsStripeIPWithEmptyRanges tests the behavior when no IP ranges are available
func TestIsStripeIPWithEmptyRanges(t *testing.T) {
	// Create a mock HTTP client
	mockClient := new(MockHTTPClient)

	// Create a service with empty IP ranges
	service := NewIPFilterService(mockClient)

	// Test with an IP
	result := service.IsStripeIP("192.0.2.10")

	// In a production environment, we'd probably want to return true if we can't
	// verify the IP (fail open rather than fail closed)
	assert.False(t, result, "IsStripeIP should return false when no ranges are available")
}

// TestBackgroundRefresh tests the background refresh functionality
func TestBackgroundRefresh(t *testing.T) {
	// Create a mock HTTP client
	mockClient := new(MockHTTPClient)

	// Set up expectations for all three sources
	webhookResp := httptest.NewRecorder()
	webhookResp.WriteString(createMockWebhookResponse())

	apiResp := httptest.NewRecorder()
	apiResp.WriteString(createMockAPIResponse())

	armadaResp := httptest.NewRecorder()
	armadaResp.WriteString(createMockArmadaGatorResponse())

	// Set expectations on the mock client - called twice for each source (initial + background)
	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "ips_webhooks.json")
	})).Return(webhookResp.Result(), nil).Times(2)

	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "ips_api.json")
	})).Return(apiResp.Result(), nil).Times(2)

	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "ips_armada_gator.json")
	})).Return(armadaResp.Result(), nil).Times(2)

	// Create the service with a short refresh interval for testing
	service := NewIPFilterService(mockClient)
	ipService := service.(*ipFilterService)

	// Set a short refresh interval for testing
	ipService.refreshInterval = 100 * time.Millisecond

	// Start background refresh
	stop := make(chan struct{})
	service.StartBackgroundRefresh(stop)

	// Wait for at least one refresh to happen
	time.Sleep(150 * time.Millisecond)

	// Stop the background refresh
	close(stop)

	// Verify that the mock was called at least twice for each source
	mockClient.AssertNumberOfCalls(t, "Do", 6) // 3 sources x 2 calls each
}

// TestMiddleware tests the Middleware function
func TestMiddleware(t *testing.T) {
	// Create a mock HTTP client
	mockClient := new(MockHTTPClient)

	// Create a service with some predefined IP ranges
	service := NewIPFilterService(mockClient)
	ipService := service.(*ipFilterService)

	// Add test IP ranges to the service
	ipService.mutex.Lock()
	ipService.ipRanges = make(map[string]*net.IPNet)
	_, network, _ := net.ParseCIDR("192.0.2.0/24")
	ipService.ipRanges["192.0.2.0/24"] = network
	ipService.mutex.Unlock()

	// Create a test gin context
	gin.SetMode(gin.TestMode)

	// Test cases
	testCases := []struct {
		name           string
		ip             string
		path           string
		override       bool
		expectedStatus int
	}{
		{"Allowed IP", "192.0.2.10", "/webhook", false, http.StatusOK},
		{"Blocked IP", "192.0.3.10", "/webhook", false, http.StatusForbidden},
		{"Override Header", "192.0.3.10", "/webhook", true, http.StatusOK},
		{"Non-webhook path", "192.0.3.10", "/other", false, http.StatusOK},
	}

	// Set up override environment variable
	os.Setenv("STRIPE_OVERRIDE_SECRET", "test-secret")
	defer os.Unsetenv("STRIPE_OVERRIDE_SECRET")

	// Set up config
	os.Setenv("STRIPE_IP_FILTER_ENABLED", "true")
	defer os.Unsetenv("STRIPE_IP_FILTER_ENABLED")

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new recorder for each test
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Set up the request
			req := httptest.NewRequest("POST", tc.path, nil)
			req.RemoteAddr = tc.ip + ":12345" // Add a port

			// Add override header if needed
			if tc.override {
				req.Header.Set("X-Stripe-Override", "test-secret")
			}

			c.Request = req

			// Create a test handler
			testHandler := func(c *gin.Context) {
				c.Status(http.StatusOK)
			}

			// Use the middleware
			service.Middleware()(c)

			// Call the next handler if not aborted
			if !c.IsAborted() {
				testHandler(c)
			}

			// Assert the response
			assert.Equal(t, tc.expectedStatus, w.Code, "Response status should match expected")
		})
	}
}

// TestMiddlewareDisabled tests the middleware when disabled via environment
func TestMiddlewareDisabled(t *testing.T) {
	// Create a mock HTTP client
	mockClient := new(MockHTTPClient)

	// Create a service
	service := NewIPFilterService(mockClient)

	// Set up test context
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Set up the request with an IP that would be blocked
	req := httptest.NewRequest("POST", "/webhook", nil)
	req.RemoteAddr = "192.0.3.10:12345"
	c.Request = req

	// Set IP filtering to disabled
	os.Setenv("STRIPE_IP_FILTER_ENABLED", "false")
	defer os.Unsetenv("STRIPE_IP_FILTER_ENABLED")

	// Create a test handler
	testHandler := func(c *gin.Context) {
		c.Status(http.StatusOK)
	}

	// Use the middleware
	service.Middleware()(c)

	// Call the next handler if not aborted
	if !c.IsAborted() {
		testHandler(c)
	}

	// Assert the response - should be OK even for a non-Stripe IP
	assert.Equal(t, http.StatusOK, w.Code, "Response should be OK when filtering is disabled")
}

// TestGetLastUpdateStatus tests the GetLastUpdateStatus method
func TestGetLastUpdateStatus(t *testing.T) {
	// Create a mock HTTP client
	mockClient := new(MockHTTPClient)

	// Create a service
	service := NewIPFilterService(mockClient)
	ipService := service.(*ipFilterService)

	// Test initial state
	status := service.GetLastUpdateStatus()
	assert.False(t, status.Failed, "Initial failed status should be false")
	assert.Equal(t, time.Time{}, status.LastUpdate, "Initial last update time should be zero")
	assert.Equal(t, 0, status.NumRanges, "Initial number of ranges should be zero")

	// Set some test values
	ipService.mutex.Lock()
	ipService.lastUpdateTime = time.Now()
	ipService.lastUpdateFailed = true
	ipService.ipRanges = make(map[string]*net.IPNet)
	_, network, _ := net.ParseCIDR("192.0.2.0/24")
	ipService.ipRanges["192.0.2.0/24"] = network
	ipService.mutex.Unlock()

	// Test updated state
	status = service.GetLastUpdateStatus()
	assert.True(t, status.Failed, "Failed status should be true")
	assert.Equal(t, 1, status.NumRanges, "Number of ranges should be 1")
	assert.False(t, status.LastUpdate.IsZero(), "Last update time should not be zero")
}

// TestIPFilteringManagerIntegration tests the integration between components
func TestIPFilteringManagerIntegration(t *testing.T) {
	// Create a mock HTTP client
	mockClient := new(MockHTTPClient)

	// Set up expectations for all three sources
	webhookResp := httptest.NewRecorder()
	webhookResp.WriteString(createMockWebhookResponse())

	apiResp := httptest.NewRecorder()
	apiResp.WriteString(createMockAPIResponse())

	armadaResp := httptest.NewRecorder()
	armadaResp.WriteString(createMockArmadaGatorResponse())

	// Set expectations on the mock client - order matters here
	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "ips_webhooks.json")
	})).Return(webhookResp.Result(), nil)

	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "ips_api.json")
	})).Return(apiResp.Result(), nil)

	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "ips_armada_gator.json")
	})).Return(armadaResp.Result(), nil)

	// Create the service
	service := NewIPFilterService(mockClient)

	// Force an initial fetch
	err := service.FetchIPRanges()
	require.NoError(t, err, "FetchIPRanges should not return an error")

	// Test IPs from different sources
	assert.True(t, service.IsStripeIP("192.0.2.1"), "Webhook IP should be recognized")
	assert.True(t, service.IsStripeIP("192.0.2.3"), "API IP should be recognized")
	assert.True(t, service.IsStripeIP("192.0.2.10"), "IP in Armada/Gator range should be recognized")

	// Test an IP not in any range
	assert.False(t, service.IsStripeIP("192.0.3.1"), "IP outside all ranges should not be recognized")

	// Test GetLastUpdateStatus
	status := service.GetLastUpdateStatus()
	assert.False(t, status.Failed, "Update should not have failed")
	assert.GreaterOrEqual(t, status.NumRanges, 6, "Should have at least 6 IP ranges")

	// Verify expectations
	mockClient.AssertExpectations(t)
}
