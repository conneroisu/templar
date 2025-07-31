//go:build integration
// +build integration

package integrationtests

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestServerConfig contains configuration for test server setup
type TestServerConfig struct {
	Host                string
	Port                int
	ReadinessTimeout    time.Duration
	HealthCheckInterval time.Duration
	MaxRetries          int
	BaseRetryDelay      time.Duration
}

// DefaultTestConfig returns a default test configuration with optimized timeouts
func DefaultTestConfig() *TestServerConfig {
	return &TestServerConfig{
		Host:                "localhost",
		Port:                0,                     // Use random available port
		ReadinessTimeout:    15 * time.Second,      // Reduced from 30s
		HealthCheckInterval: 50 * time.Millisecond, // Reduced from 100ms for faster response
		MaxRetries:          3,                     // Reduced from 5 for faster failure detection
		BaseRetryDelay:      50 * time.Millisecond, // Reduced from 100ms
	}
}

// FastTestConfig returns a configuration optimized for speed
func FastTestConfig() *TestServerConfig {
	return &TestServerConfig{
		Host:                "localhost",
		Port:                0,
		ReadinessTimeout:    5 * time.Second,
		HealthCheckInterval: 25 * time.Millisecond,
		MaxRetries:          2,
		BaseRetryDelay:      25 * time.Millisecond,
	}
}

// ServerReadiness represents server readiness status
type ServerReadiness struct {
	URL      string
	Port     int
	Ready    bool
	Healthy  bool
	Retries  int
	Duration time.Duration
}

// HealthResponse represents the structure of health check response
type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Version   string                 `json:"version"`
	Checks    map[string]interface{} `json:"checks"`
}

// WaitForServerReadiness waits for a server to become ready and healthy
// Returns ServerReadiness with detailed information about the readiness check
func WaitForServerReadiness(
	ctx context.Context,
	baseURL string,
	config *TestServerConfig,
) (*ServerReadiness, error) {
	if config == nil {
		config = DefaultTestConfig()
	}

	result := &ServerReadiness{
		URL:     baseURL,
		Ready:   false,
		Healthy: false,
	}

	start := time.Now()

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, config.ReadinessTimeout)
	defer cancel()

	ticker := time.NewTicker(config.HealthCheckInterval)
	defer ticker.Stop()

	// Extract port from URL if possible
	if strings.Contains(baseURL, ":") {
		parts := strings.Split(baseURL, ":")
		if len(parts) >= 3 {
			portStr := parts[2]
			if parsed, err := net.LookupPort("tcp", portStr); err == nil {
				result.Port = parsed
			}
		}
	}

	for result.Retries < config.MaxRetries {
		select {
		case <-timeoutCtx.Done():
			result.Duration = time.Since(start)
			return result, fmt.Errorf("server readiness timeout after %v (retries: %d)",
				config.ReadinessTimeout, result.Retries)

		case <-ticker.C:
			result.Retries++

			// First check if we can connect at all
			if !result.Ready {
				if err := checkServerConnection(baseURL); err != nil {
					continue // Server not accepting connections yet
				}
				result.Ready = true
			}

			// Then check health endpoint
			if result.Ready && !result.Healthy {
				if healthy, err := checkServerHealth(baseURL); err != nil {
					continue // Health check failed, retry
				} else if healthy {
					result.Healthy = true
					result.Duration = time.Since(start)
					return result, nil
				}
			}
		}
	}

	result.Duration = time.Since(start)
	return result, fmt.Errorf("server failed to become healthy after %d retries in %v",
		config.MaxRetries, result.Duration)
}

// checkServerConnection verifies that the server is accepting connections
func checkServerConnection(baseURL string) error {
	client := &http.Client{
		Timeout: 1 * time.Second,
	}

	req, err := http.NewRequest("GET", baseURL+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// Any response (including errors) means server is accepting connections
	return nil
}

// checkServerHealth performs a comprehensive health check
func checkServerHealth(baseURL string) (bool, error) {
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return false, fmt.Errorf("failed to decode health response: %w", err)
	}

	// Check overall status
	if health.Status != "healthy" {
		return false, fmt.Errorf("server status is %s", health.Status)
	}

	// Verify all subsystem checks are healthy
	for service, check := range health.Checks {
		if checkMap, ok := check.(map[string]interface{}); ok {
			if status, exists := checkMap["status"]; exists {
				if statusStr, ok := status.(string); ok && statusStr != "healthy" {
					return false, fmt.Errorf("service %s is not healthy: %s", service, statusStr)
				}
			}
		}
	}

	return true, nil
}

// RetryOperation executes an operation with exponential backoff retry logic
func RetryOperation(ctx context.Context, operation func() error, config *TestServerConfig) error {
	if config == nil {
		config = DefaultTestConfig()
	}

	var lastErr error
	delay := config.BaseRetryDelay

	for attempt := 0; attempt < config.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("operation cancelled: %w", ctx.Err())
		default:
		}

		if err := operation(); err != nil {
			lastErr = err
			if attempt < config.MaxRetries-1 {
				// Exponential backoff with jitter
				jitter := time.Duration(float64(delay) * 0.1)
				sleepTime := delay + time.Duration(float64(jitter)*2*(0.5-float64(attempt%2)))

				select {
				case <-ctx.Done():
					return fmt.Errorf("operation cancelled during retry: %w", ctx.Err())
				case <-time.After(sleepTime):
				}

				delay *= 2 // Exponential backoff
				if delay > 5*time.Second {
					delay = 5 * time.Second // Cap maximum delay
				}
			}
			continue
		}
		return nil // Success
	}

	return fmt.Errorf("operation failed after %d attempts, last error: %w",
		config.MaxRetries, lastErr)
}

// FindAvailablePort finds an available port on the system
func FindAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = listener.Close() }()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

// CleanupTestDirectory removes test directory and handles errors appropriately with retries
func CleanupTestDirectory(t *testing.T, dir string) {
	if dir == "" {
		return
	}

	for attempt := 0; attempt < 3; attempt++ {
		if err := os.RemoveAll(dir); err != nil {
			if attempt == 2 { // Last attempt
				t.Logf("Warning: failed to cleanup test directory %s after %d attempts: %v", dir, attempt+1, err)
			} else {
				time.Sleep(50 * time.Millisecond) // Brief pause before retry
			}
		} else {
			break // Success
		}
	}
}

// SafeTestCleanup provides a safe cleanup mechanism for tests with panic recovery
func SafeTestCleanup(t *testing.T, cleanup func() error) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Panic during cleanup: %v", r)
		}
	}()

	if cleanup != nil {
		if err := cleanup(); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}
}

// CreateTestComponent creates a component file with the given content
func CreateTestComponent(t *testing.T, dir, name, content string) string {
	filePath := filepath.Join(dir, name+".templ")
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err, "Failed to create test component %s", name)
	return filePath
}

// WaitForFileSystemSync waits for file system operations to complete with optimized timing
// This helps with race conditions between file creation and file watching
func WaitForFileSystemSync() {
	// Reduced base delay for faster execution
	time.Sleep(25 * time.Millisecond)
}

// WaitForFileSystemSyncLong provides a longer wait for bulk operations
func WaitForFileSystemSyncLong() {
	time.Sleep(50 * time.Millisecond)
}

// WaitForComponentProcessing waits for component scanning and processing to complete
// Optimized with shorter delays while maintaining reliability
func WaitForComponentProcessing() {
	// Single optimized delay instead of exponential backoff
	time.Sleep(150 * time.Millisecond)
}

// WaitForComponentProcessingBatch waits for batch component processing
func WaitForComponentProcessingBatch() {
	// Longer delay for batch operations
	time.Sleep(300 * time.Millisecond)
}

// AssertEventuallyEqual checks that a condition becomes true within a timeout
func AssertEventuallyEqual(t *testing.T, expected interface{}, getValue func() interface{},
	timeout time.Duration, message string) {

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	timeoutChan := time.After(timeout)

	for {
		select {
		case <-timeoutChan:
			actual := getValue()
			require.Equal(t, expected, actual, "Timed out waiting for condition: %s", message)
			return
		case <-ticker.C:
			if getValue() == expected {
				return
			}
		}
	}
}

// WaitForRegistryStable waits for the registry to reach a stable state with optimized timing
// If expectedCount is -1, it waits for any stable state regardless of count
func WaitForRegistryStable(registry interface{ Count() int }, expectedCount int, timeout time.Duration) error {
	if timeout == 0 {
		timeout = 3 * time.Second // Reduced default timeout
	}

	stable := false
	stableCount := 0
	lastCount := -1

	ticker := time.NewTicker(10 * time.Millisecond) // Faster polling
	defer ticker.Stop()

	timeoutChan := time.After(timeout)

	for {
		select {
		case <-timeoutChan:
			if expectedCount == -1 {
				return fmt.Errorf("registry never stabilized, last count: %d", lastCount)
			}
			return fmt.Errorf("registry never stabilized at count %d, last count: %d", expectedCount, lastCount)
		case <-ticker.C:
			currentCount := registry.Count()

			// Check if count matches expectation or if we don't care about specific count
			countMatches := (expectedCount == -1) || (currentCount == expectedCount)

			if countMatches {
				// Count matches (or we don't care) - check if it's stable
				if currentCount == lastCount {
					stableCount++
					if stableCount >= 2 { // Reduced stability requirement for faster execution
						stable = true
					}
				} else {
					stableCount = 0 // Reset stability counter
				}

				if stable {
					return nil
				}
			} else {
				stableCount = 0 // Reset if count doesn't match
			}

			lastCount = currentCount
		}
	}
}

// WaitForServerHealthy performs comprehensive server health checking with retries
func WaitForServerHealthy(baseURL string, timeout time.Duration) error {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timeoutChan := time.After(timeout)

	consecutiveSuccess := 0
	requiredConsecutive := 3 // Require 3 consecutive successful health checks

	for {
		select {
		case <-timeoutChan:
			return fmt.Errorf("server health check timed out after %v", timeout)
		case <-ticker.C:
			resp, err := client.Get(baseURL + "/health")
			if err != nil {
				consecutiveSuccess = 0
				continue
			}

			if resp.StatusCode == http.StatusOK {
				consecutiveSuccess++
				resp.Body.Close()

				if consecutiveSuccess >= requiredConsecutive {
					return nil
				}
			} else {
				consecutiveSuccess = 0
				resp.Body.Close()
			}
		}
	}
}

// ValidateServerURL validates and normalizes a server URL
func ValidateServerURL(url string) (string, error) {
	if url == "" {
		return "", fmt.Errorf("empty server URL")
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}

	return url, nil
}

// TestTimeout returns appropriate test timeout based on testing mode
func TestTimeout() time.Duration {
	if testing.Short() {
		return 5 * time.Second
	}
	return 30 * time.Second
}

// ComponentTemplate provides common component templates for testing
var ComponentTemplate = struct {
	Button string
	Card   string
	Modal  string
}{
	Button: `package components

templ Button(text string) {
	<button class="btn">{text}</button>
}`,
	Card: `package components

templ Card(title string, content string) {
	<div class="card">
		<h3>{title}</h3>
		<p>{content}</p>
	</div>
}`,
	Modal: `package components

templ Modal(title string, visible bool) {
	if visible {
		<div class="modal">
			<h2>{title}</h2>
		</div>
	}
}`,
}
