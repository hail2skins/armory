package stripe

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/logger"
)

// IPRangesResponse represents the structure of Stripe's IP ranges API response formats
type IPRangesResponse struct {
	// Webhooks format
	Webhooks []string `json:"WEBHOOKS,omitempty"`

	// API format
	API []string `json:"API,omitempty"`

	// Armada/Gator format (files/armada/gator service) - can include subnets
	ArmadaGator []string `json:"ARMADA_GATOR,omitempty"`
}

// UpdateStatus contains information about the last IP ranges update
type UpdateStatus struct {
	LastUpdate time.Time
	NumRanges  int
	Failed     bool
}

// HTTPClient interface for making HTTP requests
// This allows for easier mocking in tests
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// IPFilterService defines the interface for Stripe IP filtering operations
type IPFilterService interface {
	// FetchIPRanges fetches the latest IP ranges from Stripe API
	FetchIPRanges() error

	// IsStripeIP checks if an IP belongs to Stripe
	IsStripeIP(ip string) bool

	// StartBackgroundRefresh starts a background goroutine to periodically refresh IP ranges
	StartBackgroundRefresh(stop chan struct{})

	// Middleware returns a Gin middleware that filters requests based on IP
	Middleware() gin.HandlerFunc

	// GetLastUpdateStatus returns information about the last update
	GetLastUpdateStatus() UpdateStatus
}

// ipFilterService implements the IPFilterService interface
type ipFilterService struct {
	httpClient       HTTPClient
	ipRanges         map[string]*net.IPNet
	mutex            sync.RWMutex
	lastUpdateTime   time.Time
	lastUpdateFailed bool
	refreshInterval  time.Duration
}

// NewIPFilterService creates a new Stripe IP filter service
func NewIPFilterService(client HTTPClient) IPFilterService {
	// If no client is provided, use the default HTTP client
	if client == nil {
		client = &http.Client{
			Timeout: 10 * time.Second,
		}
	}

	return &ipFilterService{
		httpClient:      client,
		ipRanges:        make(map[string]*net.IPNet),
		mutex:           sync.RWMutex{},
		refreshInterval: 24 * time.Hour, // Default refresh every 24 hours
	}
}

// FetchIPRanges fetches the latest IP ranges from Stripe API
func (s *ipFilterService) FetchIPRanges() error {
	// Create a map to store all IP ranges
	allRanges := make(map[string]*net.IPNet)

	// List of URL and their respective field names in the response
	ipSources := []struct {
		url   string
		field string
	}{
		{"https://stripe.com/files/ips/ips_webhooks.json", "WEBHOOKS"},
		{"https://stripe.com/files/ips/ips_api.json", "API"},
		{"https://stripe.com/files/ips/ips_armada_gator.json", "ARMADA_GATOR"},
	}

	// Fetch from each source
	hasError := false
	var ipCounts = make(map[string]int)

	for _, source := range ipSources {
		ips, err := s.fetchIPsFromSource(source.url, source.field)
		if err != nil {
			logger.Error("Failed to fetch IP ranges", err, map[string]interface{}{
				"source": source.url,
			})
			hasError = true
			continue
		}

		// Add all IPs to the combined map
		for cidr, ipNet := range ips {
			allRanges[cidr] = ipNet
		}

		// Track count for logging
		ipCounts[source.field] = len(ips)
	}

	// If we have at least some IPs, update the ranges
	if len(allRanges) > 0 {
		s.mutex.Lock()
		s.ipRanges = allRanges
		s.lastUpdateTime = time.Now()
		s.lastUpdateFailed = hasError // Mark as failed only if all sources failed
		s.mutex.Unlock()

		logger.Info("Stripe IP ranges fetched successfully", map[string]interface{}{
			"webhook_ips_count":      ipCounts["WEBHOOKS"],
			"api_ips_count":          ipCounts["API"],
			"armada_gator_ips_count": ipCounts["ARMADA_GATOR"],
			"total_ips_count":        len(allRanges),
		})
		return nil
	}

	// If all sources failed
	s.setLastUpdateFailed(true)
	return fmt.Errorf("failed to fetch any IP ranges from all sources")
}

// fetchIPsFromSource fetches IP ranges from a specific Stripe source
func (s *ipFilterService) fetchIPsFromSource(url, fieldName string) (map[string]*net.IPNet, error) {
	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Send the request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch IP ranges: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the response JSON based on field name
	ips := make([]string, 0)

	// Use different unmarshaling approach based on field name
	switch fieldName {
	case "WEBHOOKS":
		var data struct {
			Webhooks []string `json:"WEBHOOKS"`
		}
		if err := json.Unmarshal(body, &data); err != nil {
			return nil, fmt.Errorf("failed to parse webhook IP ranges: %w", err)
		}
		ips = data.Webhooks
	case "API":
		var data struct {
			API []string `json:"API"`
		}
		if err := json.Unmarshal(body, &data); err != nil {
			return nil, fmt.Errorf("failed to parse API IP ranges: %w", err)
		}
		ips = data.API
	case "ARMADA_GATOR":
		var data struct {
			ArmadaGator []string `json:"ARMADA_GATOR"`
		}
		if err := json.Unmarshal(body, &data); err != nil {
			return nil, fmt.Errorf("failed to parse armada/gator IP ranges: %w", err)
		}
		ips = data.ArmadaGator
	default:
		return nil, fmt.Errorf("unknown field name: %s", fieldName)
	}

	// Parse the IP ranges
	result := make(map[string]*net.IPNet)
	for _, ipStr := range ips {
		// If it's already a CIDR block, use it as is
		// Otherwise convert individual IPs to CIDR blocks with /32 mask (single IP)
		cidr := ipStr
		if !strings.Contains(ipStr, "/") {
			cidr = ipStr + "/32"
		}

		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			logger.Warn("Failed to parse IP CIDR", map[string]interface{}{
				"cidr":   cidr,
				"source": fieldName,
				"error":  err.Error(),
			})
			continue
		}
		result[cidr] = ipNet
	}

	return result, nil
}

// storeIPRanges stores the IP ranges in memory with proper locking
// This is for backward compatibility with tests - we use individual fetches now
func (s *ipFilterService) storeIPRanges(ipRanges IPRangesResponse) {
	// Create a new map to store the IP ranges
	newRanges := make(map[string]*net.IPNet)

	// Process webhook IPs
	for _, ipStr := range ipRanges.Webhooks {
		cidr := ipStr
		if !strings.Contains(ipStr, "/") {
			cidr = ipStr + "/32"
		}

		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			logger.Warn("Failed to parse webhook IP CIDR", map[string]interface{}{
				"cidr":  cidr,
				"error": err.Error(),
			})
			continue
		}
		newRanges[cidr] = ipNet
	}

	// Process API IPs
	for _, ipStr := range ipRanges.API {
		cidr := ipStr
		if !strings.Contains(ipStr, "/") {
			cidr = ipStr + "/32"
		}

		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			logger.Warn("Failed to parse API IP CIDR", map[string]interface{}{
				"cidr":  cidr,
				"error": err.Error(),
			})
			continue
		}
		newRanges[cidr] = ipNet
	}

	// Process Armada/Gator IPs
	for _, ipStr := range ipRanges.ArmadaGator {
		cidr := ipStr
		if !strings.Contains(ipStr, "/") {
			cidr = ipStr + "/32"
		}

		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			logger.Warn("Failed to parse Armada/Gator IP CIDR", map[string]interface{}{
				"cidr":  cidr,
				"error": err.Error(),
			})
			continue
		}
		newRanges[cidr] = ipNet
	}

	// Replace the old ranges with the new ones (with locking)
	s.mutex.Lock()
	s.ipRanges = newRanges
	s.lastUpdateTime = time.Now()
	s.lastUpdateFailed = false
	s.mutex.Unlock()
}

// setLastUpdateFailed marks the last update as failed with proper locking
func (s *ipFilterService) setLastUpdateFailed(failed bool) {
	s.mutex.Lock()
	s.lastUpdateFailed = failed
	s.lastUpdateTime = time.Now()
	s.mutex.Unlock()
}

// IsStripeIP checks if an IP belongs to Stripe
func (s *ipFilterService) IsStripeIP(ipStr string) bool {
	// Parse the IP address
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// Check if the IP belongs to any of our ranges (with read lock)
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// If we have no IP ranges, fail safely (return false)
	if len(s.ipRanges) == 0 {
		logger.Warn("No IP ranges available for validation", nil)
		return false
	}

	// Check each IP range
	for _, ipNet := range s.ipRanges {
		if ipNet.Contains(ip) {
			return true
		}
	}

	return false
}

// StartBackgroundRefresh starts a background goroutine to periodically refresh IP ranges
func (s *ipFilterService) StartBackgroundRefresh(stop chan struct{}) {
	// Start a goroutine to refresh IP ranges periodically
	go func() {
		ticker := time.NewTicker(s.refreshInterval)
		defer ticker.Stop()

		// Fetch IP ranges immediately on startup
		if err := s.FetchIPRanges(); err != nil {
			logger.Error("Failed to fetch initial IP ranges", err, map[string]interface{}{
				"error": err.Error(),
			})
		}

		for {
			select {
			case <-ticker.C:
				// Fetch IP ranges periodically
				if err := s.FetchIPRanges(); err != nil {
					logger.Error("Failed to refresh IP ranges", err, map[string]interface{}{
						"error": err.Error(),
					})
				}
			case <-stop:
				// Stop the goroutine when signaled
				logger.Info("Stopping Stripe IP ranges refresh", nil)
				return
			}
		}
	}()
}

// Middleware returns a Gin middleware that filters requests based on IP
func (s *ipFilterService) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only apply filtering to webhook endpoints
		if !strings.HasPrefix(c.Request.URL.Path, "/webhook") {
			c.Next()
			return
		}

		// Check if IP filtering is enabled
		if enabled := os.Getenv("STRIPE_IP_FILTER_ENABLED"); enabled != "true" {
			// IP filtering is disabled, allow all requests
			logger.Debug("Stripe IP filtering is disabled", nil)
			c.Next()
			return
		}

		// Check for override header
		overrideSecret := os.Getenv("STRIPE_OVERRIDE_SECRET")
		if overrideSecret != "" && c.GetHeader("X-Stripe-Override") == overrideSecret {
			logger.Info("Stripe webhook request allowed by override header", map[string]interface{}{
				"path": c.Request.URL.Path,
			})
			c.Next()
			return
		}

		// Extract client IP from request
		clientIP := c.ClientIP()
		if clientIP == "" {
			// Try to extract from RemoteAddr
			clientIP = c.Request.RemoteAddr
			if i := strings.LastIndex(clientIP, ":"); i > -1 {
				clientIP = clientIP[:i]
			}
		}

		// Check if the IP belongs to Stripe
		if s.IsStripeIP(clientIP) {
			// IP is allowed, continue processing
			logger.Debug("Stripe webhook request allowed", map[string]interface{}{
				"ip":   clientIP,
				"path": c.Request.URL.Path,
			})
			c.Next()
		} else {
			// IP is not allowed, abort with forbidden
			logger.Warn("Blocked non-Stripe webhook request", map[string]interface{}{
				"ip":   clientIP,
				"path": c.Request.URL.Path,
			})
			c.AbortWithStatus(http.StatusForbidden)
		}
	}
}

// GetLastUpdateStatus returns information about the last update
func (s *ipFilterService) GetLastUpdateStatus() UpdateStatus {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return UpdateStatus{
		LastUpdate: s.lastUpdateTime,
		NumRanges:  len(s.ipRanges),
		Failed:     s.lastUpdateFailed,
	}
}
