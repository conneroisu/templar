package testutils

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRealFileSystemAdapter(t *testing.T) {
	adapter := NewRealFileSystemAdapter()
	tempDir := t.TempDir()

	// Test WriteFile and ReadFile
	testFile := tempDir + "/test.txt"
	content := []byte("test content")

	err := adapter.WriteFile(testFile, content, 0o644)
	require.NoError(t, err)

	readContent, err := adapter.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, content, readContent)

	// Test Stat
	info, err := adapter.Stat(testFile)
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), info.Size())
	assert.False(t, info.IsDir())

	// Test MkdirAll
	testDir := tempDir + "/nested/dir"
	err = adapter.MkdirAll(testDir, 0o755)
	require.NoError(t, err)

	info, err = adapter.Stat(testDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Test Remove
	err = adapter.Remove(testFile)
	require.NoError(t, err)

	_, err = adapter.Stat(testFile)
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestRealNetworkAdapter(t *testing.T) {
	// Skip if network is not available
	adapter := NewRealNetworkAdapter()

	// Test with a mock HTTP server would be better, but for now just test creation
	assert.NotNil(t, adapter)
	assert.NotNil(t, adapter.client)
	assert.Equal(t, 30*time.Second, adapter.client.Timeout)
}

func TestRealTimeAdapter(t *testing.T) {
	adapter := NewRealTimeAdapter()

	start := time.Now()
	now := adapter.Now()

	// Should be very close to actual time
	assert.WithinDuration(t, start, now, 10*time.Millisecond)

	// Test Since
	time.Sleep(1 * time.Millisecond)
	duration := adapter.Since(start)
	assert.Greater(t, duration, time.Duration(0))

	// Test Until
	future := now.Add(1 * time.Hour)
	until := adapter.Until(future)
	assert.Greater(t, until, 59*time.Minute)

	// Test Timer and Ticker creation
	timer := adapter.NewTimer(1 * time.Millisecond)
	assert.NotNil(t, timer)
	timer.Stop()

	ticker := adapter.NewTicker(1 * time.Millisecond)
	assert.NotNil(t, ticker)
	ticker.Stop()
}

func TestRealCommandAdapter(t *testing.T) {
	adapter := NewRealCommandAdapter()
	ctx := context.Background()

	// Test with echo command (should be available on most systems)
	result, err := adapter.RunCommand(ctx, "echo", "hello world")
	require.NoError(t, err)
	assert.Contains(t, result.Stdout, "hello world")
	assert.Equal(t, 0, result.ExitCode)

	// Test LookPath
	path, err := adapter.LookPath("echo")
	if err == nil { // echo might not be available on all systems
		assert.NotEmpty(t, path)
	}
}

func TestMockFileSystemAdapter(t *testing.T) {
	mf := NewMockFramework(t)
	defer mf.Cleanup()

	adapter := NewMockFileSystemAdapter(mf.FileSystem)

	// Set up mock expectations
	content := []byte("mock content")
	mf.FileSystem.On("WriteFile", "/test/file.txt", content, os.FileMode(0o644)).Return(nil)
	mf.FileSystem.On("ReadFile", "/test/file.txt").Return(content, nil)

	// Test WriteFile
	err := adapter.WriteFile("/test/file.txt", content, 0o644)
	require.NoError(t, err)

	// Test ReadFile
	readContent, err := adapter.ReadFile("/test/file.txt")
	require.NoError(t, err)
	assert.Equal(t, content, readContent)

	mf.FileSystem.AssertExpectations(t)
}

func TestMockNetworkAdapter(t *testing.T) {
	mf := NewMockFramework(t)
	defer mf.Cleanup()

	adapter := NewMockNetworkAdapter(mf.Network)

	// Set up mock expectations
	response := &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
	}
	mf.Network.On("Get", "http://example.com").Return(response, nil)

	// Test Get
	resp, err := adapter.Get("http://example.com")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, 200, resp.StatusCode)

	mf.Network.AssertExpectations(t)
}

func TestMockTimeAdapter(t *testing.T) {
	mf := NewMockFramework(t)
	defer mf.Cleanup()

	adapter := NewMockTimeAdapter(mf.Time)

	// Set up mock expectations
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	mf.Time.SetTime(fixedTime)
	mf.Time.On("Now").Return(fixedTime)
	mf.Time.On("Sleep", 1*time.Second)

	// Test Now
	now := adapter.Now()
	assert.Equal(t, fixedTime, now)

	// Test Sleep
	adapter.Sleep(1 * time.Second)

	mf.Time.AssertExpectations(t)
}

func TestMockCommandAdapter(t *testing.T) {
	mf := NewMockFramework(t)
	defer mf.Cleanup()

	adapter := NewMockCommandAdapter(mf.CommandRunner)

	// Set up mock expectations
	result := &CommandResult{
		Stdout:   "mock output",
		Stderr:   "",
		ExitCode: 0,
		Error:    nil,
	}
	ctx := context.Background()
	mf.CommandRunner.On("RunCommand", ctx, "echo", []string{"hello"}).Return(result, nil)

	// Test RunCommand
	actualResult, err := adapter.RunCommand(ctx, "echo", "hello")
	require.NoError(t, err)
	assert.Equal(t, "mock output", actualResult.Stdout)
	assert.Equal(t, 0, actualResult.ExitCode)

	mf.CommandRunner.AssertExpectations(t)
}

func TestNewRealAdapters(t *testing.T) {
	adapters := NewRealAdapters()

	assert.IsType(t, &RealFileSystemAdapter{}, adapters.FileSystem)
	assert.IsType(t, &RealNetworkAdapter{}, adapters.Network)
	assert.IsType(t, &RealTimeAdapter{}, adapters.Time)
	assert.IsType(t, &RealCommandAdapter{}, adapters.Command)
}

func TestNewMockAdapters(t *testing.T) {
	mf := NewMockFramework(t)
	defer mf.Cleanup()

	adapters := NewMockAdapters(mf)

	assert.IsType(t, &MockFileSystemAdapter{}, adapters.FileSystem)
	assert.IsType(t, &MockNetworkAdapter{}, adapters.Network)
	assert.IsType(t, &MockTimeAdapter{}, adapters.Time)
	assert.IsType(t, &MockCommandAdapter{}, adapters.Command)
}

func TestTestEnvironmentAdapters_WithMockFileSystem(t *testing.T) {
	mf := NewMockFramework(t)
	defer mf.Cleanup()

	realAdapters := NewRealAdapters()
	adapters := realAdapters.WithMockFileSystem(mf)

	assert.IsType(t, &MockFileSystemAdapter{}, adapters.FileSystem)
	assert.IsType(t, &RealNetworkAdapter{}, adapters.Network)
	assert.IsType(t, &RealTimeAdapter{}, adapters.Time)
	assert.IsType(t, &RealCommandAdapter{}, adapters.Command)
}

func TestTestEnvironmentAdapters_WithMockNetwork(t *testing.T) {
	mf := NewMockFramework(t)
	defer mf.Cleanup()

	realAdapters := NewRealAdapters()
	adapters := realAdapters.WithMockNetwork(mf)

	assert.IsType(t, &RealFileSystemAdapter{}, adapters.FileSystem)
	assert.IsType(t, &MockNetworkAdapter{}, adapters.Network)
	assert.IsType(t, &RealTimeAdapter{}, adapters.Time)
	assert.IsType(t, &RealCommandAdapter{}, adapters.Command)
}

func TestTestEnvironmentAdapters_WithMockTime(t *testing.T) {
	mf := NewMockFramework(t)
	defer mf.Cleanup()

	realAdapters := NewRealAdapters()
	adapters := realAdapters.WithMockTime(mf)

	assert.IsType(t, &RealFileSystemAdapter{}, adapters.FileSystem)
	assert.IsType(t, &RealNetworkAdapter{}, adapters.Network)
	assert.IsType(t, &MockTimeAdapter{}, adapters.Time)
	assert.IsType(t, &RealCommandAdapter{}, adapters.Command)
}

func TestTestEnvironmentAdapters_WithMockCommand(t *testing.T) {
	mf := NewMockFramework(t)
	defer mf.Cleanup()

	realAdapters := NewRealAdapters()
	adapters := realAdapters.WithMockCommand(mf)

	assert.IsType(t, &RealFileSystemAdapter{}, adapters.FileSystem)
	assert.IsType(t, &RealNetworkAdapter{}, adapters.Network)
	assert.IsType(t, &RealTimeAdapter{}, adapters.Time)
	assert.IsType(t, &MockCommandAdapter{}, adapters.Command)
}

func TestIntegrationWithExistingHelpers(t *testing.T) {
	// Test that mock framework works with existing helpers
	mf := NewMockFramework(t)
	defer mf.Cleanup()

	// Use existing CreateTempProject helper
	projectDir := CreateTempProject(t)

	// Set up file system mock for config file
	configContent := []byte(`{"test": true}`)
	configPath := projectDir + "/config.json"

	mf.FileSystem.CreateFile(configPath, configContent, 0o644)
	mf.FileSystem.On("ReadFile", configPath).Return(configContent, nil)

	// Create adapter and test
	adapter := NewMockFileSystemAdapter(mf.FileSystem)

	content, err := adapter.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, configContent, content)

	mf.FileSystem.AssertExpectations(t)
}

func TestMockFrameworkPerformance(t *testing.T) {
	mf := NewMockFramework(t)
	defer mf.Cleanup()

	// Set up expectations
	mf.FileSystem.On("ReadFile", "/test/file.txt").Return([]byte("content"), nil)

	adapter := NewMockFileSystemAdapter(mf.FileSystem)

	// Measure performance of many calls
	start := time.Now()
	iterations := 1000

	for range iterations {
		content, err := adapter.ReadFile("/test/file.txt")
		require.NoError(t, err)
		assert.Equal(t, []byte("content"), content)
	}

	duration := time.Since(start)
	t.Logf("Performed %d mock operations in %v (%.2f µs per operation)",
		iterations, duration, float64(duration.Nanoseconds())/float64(iterations)/1000)

	// Should be fast
	assert.Less(t, duration, 100*time.Millisecond)
}

func TestMockFrameworkMemoryUsage(t *testing.T) {
	mf := NewMockFramework(t)
	defer mf.Cleanup()

	// Create many files in mock filesystem
	for i := range 1000 {
		filename := "/test/file_" + strings.Repeat("x", i%100) + ".txt"
		content := []byte(strings.Repeat("content", i%50))
		mf.FileSystem.CreateFile(filename, content, 0o644)
	}

	// Reset should clean up memory
	mf.Reset()

	// Verify cleanup
	assert.Empty(t, mf.FileSystem.files)
	assert.Empty(t, mf.FileSystem.permissions)
}
