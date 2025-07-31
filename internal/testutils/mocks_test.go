package testutils

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMockFramework_Creation(t *testing.T) {
	mf := NewMockFramework(t)
	defer mf.Cleanup()

	assert.NotNil(t, mf.FileSystem)
	assert.NotNil(t, mf.Network)
	assert.NotNil(t, mf.Time)
	assert.NotNil(t, mf.CommandRunner)
}

func TestMockFramework_Reset(t *testing.T) {
	mf := NewMockFramework(t)
	defer mf.Cleanup()

	// Set up some mock expectations
	mf.FileSystem.On("ReadFile", "test.txt").Return([]byte("content"), nil)
	mf.Network.On("Get", "http://example.com").Return(&http.Response{}, nil)
	mf.Time.On("Now").Return(time.Unix(0, 0))

	// Reset should clear all expectations
	mf.Reset()

	// Verify mocks are reset
	assert.Empty(t, mf.FileSystem.ExpectedCalls)
	assert.Empty(t, mf.Network.ExpectedCalls)
	assert.Empty(t, mf.Time.ExpectedCalls)
}

func TestMockFramework_Cleanup(t *testing.T) {
	mf := NewMockFramework(t)

	cleanupCalled := false
	mf.AddCleanup(func() {
		cleanupCalled = true
	})

	mf.Cleanup()
	assert.True(t, cleanupCalled)
}

func TestMockFileSystem_CreateAndReadFile(t *testing.T) {
	mfs := NewMockFileSystem()

	content := []byte("test content")
	mfs.CreateFile("/test/file.txt", content, 0o644)

	// Mock the ReadFile call
	mfs.On("ReadFile", "/test/file.txt").Return(content, nil)

	result, err := mfs.ReadFile("/test/file.txt")
	require.NoError(t, err)
	assert.Equal(t, content, result)

	mfs.AssertExpectations(t)
}

func TestMockFileSystem_WriteFile(t *testing.T) {
	mfs := NewMockFileSystem()

	content := []byte("new content")
	mfs.On("WriteFile", "/test/newfile.txt", content, os.FileMode(0o644)).Return(nil)

	err := mfs.WriteFile("/test/newfile.txt", content, 0o644)
	require.NoError(t, err)

	// Verify file was created in mock filesystem
	file, exists := mfs.files["/test/newfile.txt"]
	assert.True(t, exists)
	assert.Equal(t, content, file.Content)
	assert.Equal(t, os.FileMode(0o644), file.Mode)

	mfs.AssertExpectations(t)
}

func TestMockFileSystem_CreateDir(t *testing.T) {
	mfs := NewMockFileSystem()

	mfs.CreateDir("/test/dir", 0o755)

	// Mock Stat call
	mfs.On("Stat", "/test/dir").Return(nil, nil)

	_, err := mfs.Stat("/test/dir")
	require.NoError(t, err)

	// Verify directory was created
	file, exists := mfs.files["/test/dir"]
	assert.True(t, exists)
	assert.True(t, file.IsDir)
	assert.Equal(t, os.FileMode(0o755)|os.ModeDir, file.Mode)

	mfs.AssertExpectations(t)
}

func TestMockFileSystem_Stat(t *testing.T) {
	mfs := NewMockFileSystem()

	content := []byte("test content")
	mfs.CreateFile("/test/file.txt", content, 0o644)

	mfs.On("Stat", "/test/file.txt").Return(nil, nil)

	info, err := mfs.Stat("/test/file.txt")
	require.NoError(t, err)

	assert.Equal(t, "/test/file.txt", info.Name())
	assert.Equal(t, int64(len(content)), info.Size())
	assert.Equal(t, os.FileMode(0o644), info.Mode())
	assert.False(t, info.IsDir())

	mfs.AssertExpectations(t)
}

func TestMockFileSystem_MkdirAll(t *testing.T) {
	mfs := NewMockFileSystem()

	mfs.On("MkdirAll", "/test/path/deep", os.FileMode(0o755)).Return(nil)

	err := mfs.MkdirAll("/test/path/deep", 0o755)
	require.NoError(t, err)

	// Verify directory was created
	file, exists := mfs.files["/test/path/deep"]
	assert.True(t, exists)
	assert.True(t, file.IsDir)

	mfs.AssertExpectations(t)
}

func TestMockFileSystem_Remove(t *testing.T) {
	mfs := NewMockFileSystem()

	mfs.CreateFile("/test/file.txt", []byte("content"), 0o644)
	mfs.On("Remove", "/test/file.txt").Return(nil)

	err := mfs.Remove("/test/file.txt")
	require.NoError(t, err)

	// Verify file was removed
	_, exists := mfs.files["/test/file.txt"]
	assert.False(t, exists)

	mfs.AssertExpectations(t)
}

func TestMockFileSystem_ReadFileNotFound(t *testing.T) {
	mfs := NewMockFileSystem()

	mfs.On("ReadFile", "/nonexistent.txt").Return([]byte(nil), os.ErrNotExist)

	_, err := mfs.ReadFile("/nonexistent.txt")
	assert.ErrorIs(t, err, os.ErrNotExist)

	mfs.AssertExpectations(t)
}

func TestMockFileSystem_ReadDirectory(t *testing.T) {
	mfs := NewMockFileSystem()

	mfs.CreateDir("/test/dir", 0o755)
	mfs.On("ReadFile", "/test/dir").Return([]byte(nil), errors.New("read /test/dir: is a directory"))

	_, err := mfs.ReadFile("/test/dir")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is a directory")

	mfs.AssertExpectations(t)
}

func TestMockNetwork_HTTPResponse(t *testing.T) {
	mn := NewMockNetwork()

	// Create mock HTTP response
	response := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("response body")),
	}

	mn.MockHTTPResponse("http://example.com", response, nil)
	mn.On("Get", "http://example.com").Return(response, nil)

	result, err := mn.Get("http://example.com")
	require.NoError(t, err)
	defer func() { _ = result.Body.Close() }()
	assert.Equal(t, 200, result.StatusCode)

	body, err := io.ReadAll(result.Body)
	require.NoError(t, err)
	assert.Equal(t, "response body", string(body))

	mn.AssertExpectations(t)
}

func TestMockNetwork_HTTPError(t *testing.T) {
	mn := NewMockNetwork()

	expectedError := errors.New("network error")
	mn.MockHTTPResponse("http://bad-url.com", nil, expectedError)
	mn.On("Get", "http://bad-url.com").Return((*http.Response)(nil), expectedError)

	resp, err := mn.Get("http://bad-url.com")
	assert.Error(t, err)
	if resp != nil && resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	assert.Equal(t, expectedError, err)

	mn.AssertExpectations(t)
}

func TestMockNetwork_Post(t *testing.T) {
	mn := NewMockNetwork()

	response := &http.Response{
		StatusCode: http.StatusCreated,
		Body:       io.NopCloser(strings.NewReader("created")),
	}

	body := strings.NewReader("post data")
	mn.On("Post", "http://example.com/api", "application/json", body).Return(response, nil)

	result, err := mn.Post("http://example.com/api", "application/json", body)
	require.NoError(t, err)
	defer func() { _ = result.Body.Close() }()
	assert.Equal(t, 201, result.StatusCode)

	mn.AssertExpectations(t)
}

func TestMockTime_BasicFunctionality(t *testing.T) {
	mt := NewMockTime()

	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	mt.SetTime(fixedTime)
	mt.On("Now").Return(fixedTime)

	result := mt.Now()
	assert.Equal(t, fixedTime, result)

	mt.AssertExpectations(t)
}

func TestMockTime_FreezeAndUnfreeze(t *testing.T) {
	mt := NewMockTime()

	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	mt.SetTime(fixedTime)

	// Freeze time
	mt.FreezeTime()

	// Advance time should not change frozen time
	mt.AdvanceTime(1 * time.Hour)
	assert.Equal(t, fixedTime, mt.currentTime)

	// Unfreeze time
	mt.UnfreezeTime()

	// Now advance should work
	mt.AdvanceTime(1 * time.Hour)
	expected := fixedTime.Add(1 * time.Hour)
	assert.Equal(t, expected, mt.currentTime)
}

func TestMockTime_AdvanceTime(t *testing.T) {
	mt := NewMockTime()

	initialTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	mt.SetTime(initialTime)

	mt.AdvanceTime(2 * time.Hour)
	expected := initialTime.Add(2 * time.Hour)
	assert.Equal(t, expected, mt.currentTime)
}

func TestMockTime_Sleep(t *testing.T) {
	mt := NewMockTime()

	initialTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	mt.SetTime(initialTime)

	mt.On("Sleep", 30*time.Second)

	mt.Sleep(30 * time.Second)

	// Verify time advanced
	expected := initialTime.Add(30 * time.Second)
	assert.Equal(t, expected, mt.currentTime)

	mt.AssertExpectations(t)
}

func TestMockCommandRunner_BasicExecution(t *testing.T) {
	mcr := NewMockCommandRunner()

	result := &CommandResult{
		Stdout:   "command output",
		Stderr:   "",
		ExitCode: 0,
		Error:    nil,
	}

	ctx := context.Background()
	mcr.MockCommand("go version", result)
	mcr.On("RunCommand", ctx, "go", []string{"version"}).Return(result, nil)

	actualResult, err := mcr.RunCommand(ctx, "go", "version")
	require.NoError(t, err)
	assert.Equal(t, "command output", actualResult.Stdout)
	assert.Equal(t, 0, actualResult.ExitCode)

	mcr.AssertExpectations(t)
}

func TestMockCommandRunner_CommandError(t *testing.T) {
	mcr := NewMockCommandRunner()

	expectedError := errors.New("command failed")
	result := &CommandResult{
		Stdout:   "",
		Stderr:   "error message",
		ExitCode: 1,
		Error:    expectedError,
	}

	ctx := context.Background()
	mcr.MockCommand("failing-command", result)
	mcr.On("RunCommand", ctx, "failing-command", mock.AnythingOfType("[]string")).Return(result, expectedError)

	actualResult, err := mcr.RunCommand(ctx, "failing-command")
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Equal(t, "error message", actualResult.Stderr)
	assert.Equal(t, 1, actualResult.ExitCode)

	mcr.AssertExpectations(t)
}

func TestMockFramework_Integration(t *testing.T) {
	mf := NewMockFramework(t)
	defer mf.Cleanup()

	// Set up file system mock
	configContent := []byte(`{"server": {"port": 8080}}`)
	mf.FileSystem.CreateFile("/config/app.json", configContent, 0o644)
	mf.FileSystem.On("ReadFile", "/config/app.json").Return(configContent, nil)

	// Set up network mock
	response := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"status": "ok"}`)),
	}
	mf.Network.MockHTTPResponse("http://api.example.com/status", response, nil)
	mf.Network.On("Get", "http://api.example.com/status").Return(response, nil)

	// Set up time mock
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	mf.Time.SetTime(fixedTime)
	mf.Time.On("Now").Return(fixedTime)

	// Set up command runner mock
	buildResult := &CommandResult{
		Stdout:   "Build successful",
		Stderr:   "",
		ExitCode: 0,
		Error:    nil,
	}
	ctx := context.Background()
	mf.CommandRunner.MockCommand("go build .", buildResult)
	mf.CommandRunner.On("RunCommand", ctx, "go", []string{"build", "."}).Return(buildResult, nil)

	// Test file system
	content, err := mf.FileSystem.ReadFile("/config/app.json")
	require.NoError(t, err)
	assert.Equal(t, configContent, content)

	// Test network
	resp, err := mf.Network.Get("http://api.example.com/status")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, 200, resp.StatusCode)

	// Test time
	now := mf.Time.Now()
	assert.Equal(t, fixedTime, now)

	// Test command runner
	cmdResult, err := mf.CommandRunner.RunCommand(ctx, "go", "build", ".")
	require.NoError(t, err)
	assert.Equal(t, "Build successful", cmdResult.Stdout)

	// Verify all expectations
	mf.FileSystem.AssertExpectations(t)
	mf.Network.AssertExpectations(t)
	mf.Time.AssertExpectations(t)
	mf.CommandRunner.AssertExpectations(t)
}

func TestMockFramework_ConcurrentAccess(t *testing.T) {
	mf := NewMockFramework(t)
	defer mf.Cleanup()

	// Test concurrent access to file system mock
	mf.FileSystem.CreateFile("/test/file.txt", []byte("content"), 0o644)
	mf.FileSystem.On("ReadFile", "/test/file.txt").Return([]byte("content"), nil)

	done := make(chan bool, 10)

	// Launch multiple goroutines
	for range 10 {
		go func() {
			defer func() { done <- true }()

			content, err := mf.FileSystem.ReadFile("/test/file.txt")
			assert.NoError(t, err)
			assert.Equal(t, []byte("content"), content)
		}()
	}

	// Wait for all goroutines to complete
	for range 10 {
		<-done
	}

	mf.FileSystem.AssertExpectations(t)
}
