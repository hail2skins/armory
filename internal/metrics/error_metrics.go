package metrics

import (
	"sort"
	"sync"
	"time"
)

// ErrorEntry represents a single error type with its statistics
type ErrorEntry struct {
	Count        int64
	LastOccurred time.Time
	Latencies    []float64
	Path         string
	Timestamps   []time.Time
	IPAddresses  []string // Store IP addresses for each occurrence
}

// AvgLatency calculates the average latency for this error type
func (e *ErrorEntry) AvgLatency() float64 {
	if len(e.Latencies) == 0 {
		return 0
	}
	sum := 0.0
	for _, l := range e.Latencies {
		sum += l
	}
	return sum / float64(len(e.Latencies))
}

// LastIP returns the IP address of the most recent occurrence
func (e *ErrorEntry) LastIP() string {
	if len(e.IPAddresses) == 0 {
		return "unknown"
	}
	return e.IPAddresses[len(e.IPAddresses)-1]
}

// ErrorMetrics tracks error statistics across the application
type ErrorMetrics struct {
	errors      map[string]*ErrorEntry // Key: error type
	endpoints   map[string]*ErrorEntry // Key: endpoint path
	statusCodes map[int]*ErrorEntry    // Key: HTTP status code
	mu          sync.RWMutex
}

// NewErrorMetrics creates a new error metrics collector
func NewErrorMetrics() *ErrorMetrics {
	return &ErrorMetrics{
		errors:      make(map[string]*ErrorEntry),
		endpoints:   make(map[string]*ErrorEntry),
		statusCodes: make(map[int]*ErrorEntry),
	}
}

// Record adds an error occurrence to the metrics
func (em *ErrorMetrics) Record(errorType string, statusCode int, latency float64, path string, ipAddress string) {
	em.recordWithTime(errorType, statusCode, latency, path, ipAddress, time.Now())
}

// recordWithTime is an internal method for testing that allows setting the timestamp
func (em *ErrorMetrics) recordWithTime(errorType string, statusCode int, latency float64, path string, ipAddress string, timestamp time.Time) {
	em.mu.Lock()
	defer em.mu.Unlock()

	// Record error type metrics
	if _, exists := em.errors[errorType]; !exists {
		em.errors[errorType] = &ErrorEntry{
			Timestamps:  make([]time.Time, 0),
			Latencies:   make([]float64, 0),
			IPAddresses: make([]string, 0),
		}
	}
	entry := em.errors[errorType]
	entry.Count++
	entry.LastOccurred = timestamp
	entry.Latencies = append(entry.Latencies, latency)
	entry.Path = path
	entry.Timestamps = append(entry.Timestamps, timestamp)
	entry.IPAddresses = append(entry.IPAddresses, ipAddress)

	// Record endpoint metrics
	if _, exists := em.endpoints[path]; !exists {
		em.endpoints[path] = &ErrorEntry{
			Timestamps:  make([]time.Time, 0),
			Latencies:   make([]float64, 0),
			IPAddresses: make([]string, 0),
		}
	}
	em.endpoints[path].Count++
	em.endpoints[path].LastOccurred = timestamp
	em.endpoints[path].Latencies = append(em.endpoints[path].Latencies, latency)
	em.endpoints[path].Timestamps = append(em.endpoints[path].Timestamps, timestamp)
	em.endpoints[path].IPAddresses = append(em.endpoints[path].IPAddresses, ipAddress)

	// Record status code metrics
	if _, exists := em.statusCodes[statusCode]; !exists {
		em.statusCodes[statusCode] = &ErrorEntry{
			Timestamps:  make([]time.Time, 0),
			Latencies:   make([]float64, 0),
			IPAddresses: make([]string, 0),
		}
	}
	em.statusCodes[statusCode].Count++
	em.statusCodes[statusCode].LastOccurred = timestamp
	em.statusCodes[statusCode].Latencies = append(em.statusCodes[statusCode].Latencies, latency)
	em.statusCodes[statusCode].Timestamps = append(em.statusCodes[statusCode].Timestamps, timestamp)
	em.statusCodes[statusCode].IPAddresses = append(em.statusCodes[statusCode].IPAddresses, ipAddress)
}

// GetStats returns all collected error statistics
func (em *ErrorMetrics) GetStats() map[string]interface{} {
	em.mu.RLock()
	defer em.mu.RUnlock()

	return map[string]interface{}{
		"error_counts":    em.errors,
		"status_counts":   em.statusCodes,
		"endpoint_counts": em.endpoints,
	}
}

// GetRecentErrors returns the most recent errors, up to the specified count
func (em *ErrorMetrics) GetRecentErrors(count int) []struct {
	ErrorType    string
	Count        int64
	LastOccurred time.Time
	Path         string
	IPAddress    string
} {
	em.mu.RLock()
	defer em.mu.RUnlock()

	errors := make([]struct {
		ErrorType    string
		Count        int64
		LastOccurred time.Time
		Path         string
		IPAddress    string
	}, 0, len(em.errors))

	for errType, entry := range em.errors {
		errors = append(errors, struct {
			ErrorType    string
			Count        int64
			LastOccurred time.Time
			Path         string
			IPAddress    string
		}{
			ErrorType:    errType,
			Count:        entry.Count,
			LastOccurred: entry.LastOccurred,
			Path:         entry.Path,
			IPAddress:    entry.LastIP(),
		})
	}

	// Sort by LastOccurred in descending order
	sort.Slice(errors, func(i, j int) bool {
		return errors[i].LastOccurred.After(errors[j].LastOccurred)
	})

	// Limit to the requested count
	if len(errors) > count {
		errors = errors[:count]
	}

	return errors
}

// GetErrorRates returns the error count for each error type within the specified duration
func (em *ErrorMetrics) GetErrorRates(duration time.Duration) map[string]float64 {
	return em.GetErrorRatesWithReference(duration, time.Now())
}

// GetErrorRatesWithReference is an internal method for testing that allows setting the reference time
func (em *ErrorMetrics) GetErrorRatesWithReference(duration time.Duration, referenceTime time.Time) map[string]float64 {
	em.mu.RLock()
	defer em.mu.RUnlock()

	rates := make(map[string]float64)
	cutoff := referenceTime.Add(-duration)

	for errType, entry := range em.errors {
		count := float64(0)
		for _, ts := range entry.Timestamps {
			if !ts.Before(cutoff) {
				count++
			}
		}
		rates[errType] = count
	}

	return rates
}

// GetLatencyPercentiles calculates the p50, p95, and p99 percentiles for all errors
func (em *ErrorMetrics) GetLatencyPercentiles() map[string]float64 {
	em.mu.RLock()
	defer em.mu.RUnlock()

	allLatencies := make([]float64, 0)
	for _, entry := range em.errors {
		allLatencies = append(allLatencies, entry.Latencies...)
	}

	if len(allLatencies) == 0 {
		return map[string]float64{
			"p50": 0,
			"p95": 0,
			"p99": 0,
		}
	}

	sort.Float64s(allLatencies)

	return map[string]float64{
		"p50": percentile(allLatencies, 0.5),
		"p95": percentile(allLatencies, 0.95),
		"p99": percentile(allLatencies, 0.99),
	}
}

// Cleanup removes error statistics older than the specified duration
func (em *ErrorMetrics) Cleanup(maxAge time.Duration) {
	em.mu.Lock()
	defer em.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)

	// Cleanup old error entries
	for _, entry := range em.errors {
		if entry != nil && entry.LastOccurred.Before(cutoff) {
			entry.Count = 0
			entry.Latencies = nil
			entry.Timestamps = nil
			entry.IPAddresses = nil
		}
	}

	// Cleanup old endpoint entries
	for _, entry := range em.endpoints {
		if entry != nil && entry.LastOccurred.Before(cutoff) {
			entry.Count = 0
			entry.Latencies = nil
			entry.Timestamps = nil
			entry.IPAddresses = nil
		}
	}

	// Cleanup old status code entries
	for _, entry := range em.statusCodes {
		if entry != nil && entry.LastOccurred.Before(cutoff) {
			entry.Count = 0
			entry.Latencies = nil
			entry.Timestamps = nil
			entry.IPAddresses = nil
		}
	}
}

// percentile calculates the percentile value from a sorted slice
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	index := int(float64(len(sorted)-1) * p)
	return sorted[index]
}
