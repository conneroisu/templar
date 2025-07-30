package testutils

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ExampleBasicMockUsage demonstrates basic mock framework usage.
func ExampleBasicMockUsage() {
	// This example shows how to use the mock framework in a test
	
	var t *testing.T // In real code, this comes from the test function parameter
	
	// Create mock framework
	mf := NewMockFramework(t)
	defer mf.Cleanup()

	// Set up file system mock
	mf.FileSystem.CreateFile("/config/app.json", []byte(`{"port": 8080}`), 0o644)
	mf.FileSystem.On("ReadFile", "/config/app.json").Return([]byte(`{"port": 8080}`), nil)

	// Set up network mock
	response := &http.Response{StatusCode: http.StatusOK}
	mf.Network.On("Get", "http://api.example.com").Return(response, nil)

	// Set up time mock for deterministic testing
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	mf.Time.SetTime(fixedTime)
	mf.Time.On("Now").Return(fixedTime)

	// Your test code here...
	
	// Verify expectations at the end
	AssertMockExpectations(t, mf)
}

// ExampleIsolatedTestSuite demonstrates complete test isolation.
func ExampleIsolatedTestSuite() {
	var t *testing.T // In real code, this comes from the test function parameter
	
	// Create completely isolated test environment
	_, adapters, cleanup := CreateIsolatedTestSuite(t)
	defer cleanup()

	// Use adapters instead of direct os/http/time calls
	content, err := adapters.FileSystem.ReadFile("/config/app.json")
	if err != nil {
		// Handle error
		_ = content
	}

	now := adapters.Time.Now()
	_ = now // Use deterministic time
	
	// All expectations are automatically verified in cleanup()
}

// ExampleMockingSpecificOperations shows how to mock specific operations.
func ExampleMockingSpecificOperations() {
	var t *testing.T
	
	mf := NewMockFramework(t)
	defer mf.Cleanup()

	// Mock build failure
	MockBuildFailure(mf, "syntax error in template")

	// Mock network error
	MockNetworkError(mf, "http://api.example.com", "connection timeout")

	// Mock file system error
	MockFileSystemError(mf, "ReadFile", "/nonexistent.txt", "file not found")

	// Test your code that should handle these errors...
}

// ExampleConvertingExistingTest shows how to convert an existing test to use mocks.
//
// Before: Test that directly uses file system
// func TestReadConfig(t *testing.T) {
//     tempDir := t.TempDir()
//     configFile := filepath.Join(tempDir, "config.json")
//     
//     // Create real file
//     err := os.WriteFile(configFile, []byte(`{"port": 8080}`), 0644)
//     require.NoError(t, err)
//     
//     // Test function that reads file
//     config, err := ReadConfig(configFile)
//     require.NoError(t, err)
//     assert.Equal(t, 8080, config.Port)
// }
//
// After: Test using mocks for better isolation
func ExampleConvertingExistingTest(t *testing.T) {
	// Create mock framework
	mf := NewMockFramework(t)
	defer mf.Cleanup()

	// Set up mock file system
	configContent := []byte(`{"port": 8080}`)
	mf.FileSystem.CreateFile("/config/config.json", configContent, 0o644)
	mf.FileSystem.On("ReadFile", "/config/config.json").Return(configContent, nil)

	// Create adapters
	_ = NewMockAdapters(mf)

	// Test function that uses adapters instead of direct os calls
	// config, err := ReadConfigWithAdapter(adapters.FileSystem, "/config/config.json")
	// require.NoError(t, err)
	// assert.Equal(t, 8080, config.Port)

	// Verify all expectations
	AssertMockExpectations(t, mf)
}

// ExampleTestingWithDeterministicTime shows how to test time-dependent code.
func ExampleTestingWithDeterministicTime(t *testing.T) {
	mf := NewMockFramework(t)
	defer mf.Cleanup()

	// Set up deterministic time
	baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	SetupMockTime(mf, baseTime)

	adapter := NewMockTimeAdapter(mf.Time)

	// Test time-dependent operations
	start := adapter.Now()
	require.Equal(t, baseTime, start)

	// Advance time for testing
	mf.Time.AdvanceTime(1 * time.Hour)
	later := adapter.Now()
	expected := baseTime.Add(1 * time.Hour)
	require.Equal(t, expected, later)

	AssertMockExpectations(t, mf)
}

// ExampleMockingCommandExecution shows how to test command execution.
func ExampleMockingCommandExecution(t *testing.T) {
	mf := NewMockFramework(t)
	defer mf.Cleanup()

	// Set up command mocks
	SetupMockCommands(mf)

	adapter := NewMockCommandAdapter(mf.CommandRunner)
	ctx := context.Background()

	// Test successful command
	result, err := adapter.RunCommand(ctx, "templ", "generate")
	require.NoError(t, err)
	assert.Equal(t, "Generated templates successfully", result.Stdout)
	assert.Equal(t, 0, result.ExitCode)

	AssertMockExpectations(t, mf)
}

// ExampleTestingErrorHandling demonstrates testing error conditions.
func ExampleTestingErrorHandling(t *testing.T) {
	mf := NewMockFramework(t)
	defer mf.Cleanup()

	// Set up error conditions
	MockFileSystemError(mf, "ReadFile", "/config/app.json", "permission denied")
	MockNetworkError(mf, "http://api.example.com", "connection refused")
	MockBuildFailure(mf, "template compilation failed")

	adapters := NewMockAdapters(mf)

	// Test file system error handling
	_, err := adapters.FileSystem.ReadFile("/config/app.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")

	// Test network error handling
	resp, err := adapters.Network.Get("http://api.example.com")
	assert.Error(t, err)
	if resp != nil && resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	assert.Contains(t, err.Error(), "connection refused")

	// Test command error handling
	ctx := context.Background()
	result, err := adapters.Command.RunCommand(ctx, "templ", "generate")
	assert.Error(t, err)
	assert.Contains(t, result.Stderr, "template compilation failed")

	AssertMockExpectations(t, mf)
}

// ExampleConcurrentTesting shows how to test concurrent operations with mocks.
func ExampleConcurrentTesting(t *testing.T) {
	mf := NewMockFramework(t)
	defer mf.Cleanup()

	// Set up mocks for concurrent access
	for i := 0; i < 10; i++ {
		filename := fmt.Sprintf("/data/file_%d.txt", i)
		content := []byte(fmt.Sprintf("content_%d", i))
		mf.FileSystem.CreateFile(filename, content, 0o644)
		mf.FileSystem.On("ReadFile", filename).Return(content, nil)
	}

	adapter := NewMockFileSystemAdapter(mf.FileSystem)

	// Test concurrent file access
	results := make(chan string, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			filename := fmt.Sprintf("/data/file_%d.txt", id)
			content, err := adapter.ReadFile(filename)
			if err == nil {
				results <- string(content)
			} else {
				results <- ""
			}
		}(i)
	}

	// Collect results
	for i := 0; i < 10; i++ {
		result := <-results
		assert.NotEmpty(t, result)
	}

	AssertMockExpectations(t, mf)
}

// ExampleMockingWebSocketConnections shows how to mock WebSocket operations.
func ExampleMockingWebSocketConnections(t *testing.T) {
	mf := NewMockFramework(t)
	defer mf.Cleanup()

	// Set up WebSocket connection mock
	// Note: This would require extending the mock framework for WebSocket support
	// For now, this shows the pattern

	// Mock HTTP upgrade request for WebSocket
	upgradeResponse := &http.Response{
		StatusCode: http.StatusSwitchingProtocols,
		Header:     make(http.Header),
	}
	upgradeResponse.Header.Set("Upgrade", "websocket")
	upgradeResponse.Header.Set("Connection", "Upgrade")

	mf.Network.On("Do", mock.AnythingOfType("*http.Request")).Return(upgradeResponse, nil)

	// Your WebSocket testing code here...
	
	AssertMockExpectations(t, mf)
}

// ExamplePerformanceTesting shows how to use mocks for performance testing.
func ExamplePerformanceTesting(t *testing.T) {
	mf := NewMockFramework(t)
	defer mf.Cleanup()

	// Set up lightweight mocks for performance testing
	content := []byte("test content")
	mf.FileSystem.CreateFile("/test/file.txt", content, 0o644)
	mf.FileSystem.On("ReadFile", "/test/file.txt").Return(content, nil)

	adapter := NewMockFileSystemAdapter(mf.FileSystem)

	// Performance test with mocks
	start := time.Now()
	iterations := 1000
	
	for i := 0; i < iterations; i++ {
		_, err := adapter.ReadFile("/test/file.txt")
		require.NoError(t, err)
	}

	duration := time.Since(start)
	t.Logf("Performed %d mock operations in %v", iterations, duration)

	// Should be very fast with mocks
	assert.Less(t, duration, 100*time.Millisecond)

	AssertMockExpectations(t, mf)
}