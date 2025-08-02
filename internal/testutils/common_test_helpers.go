// Package testutils provides comprehensive testing utilities for the Templar project.
// This follows the zero technical debt policy by providing reusable, well-tested helpers
// that prevent code duplication and ensure consistent testing patterns across the codebase.
package testutils

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContext provides a standardized test context with timeout and cleanup.
type TestContext struct {
	ctx     context.Context
	cancel  context.CancelFunc
	cleanup []func()
	tmpDir  string
	mu      sync.Mutex
	t       *testing.T
}

// TestingT represents the common interface between *testing.T and *testing.B.
type TestingT interface {
	Cleanup(func())
	TempDir() string
	Fatalf(format string, args ...interface{})
	Logf(format string, args ...interface{})
}

// NewTestContext creates a new test context with a 30-second timeout and cleanup tracking.
func NewTestContext(t TestingT) *TestContext {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	tc := &TestContext{
		ctx:     ctx,
		cancel:  cancel,
		cleanup: make([]func(), 0),
		t:       t.(*testing.T), // Safe cast for backwards compatibility
	}

	// Ensure cleanup is called even if test panics
	t.Cleanup(tc.Cleanup)

	return tc
}

// NewBenchmarkContext creates a new test context for benchmarks.
func NewBenchmarkContext(b *testing.B) *TestContext {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	tc := &TestContext{
		ctx:     ctx,
		cancel:  cancel,
		cleanup: make([]func(), 0),
		t:       nil, // B doesn't implement same interface as T
	}

	// Ensure cleanup is called even if benchmark panics
	b.Cleanup(tc.Cleanup)

	return tc
}

// Context returns the underlying context.
func (tc *TestContext) Context() context.Context {
	return tc.ctx
}

// AddCleanup adds a cleanup function to be called when the test ends.
func (tc *TestContext) AddCleanup(cleanup func()) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.cleanup = append(tc.cleanup, cleanup)
}

// TempDir creates a temporary directory for the test and schedules it for cleanup.
func (tc *TestContext) TempDir() string {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.tmpDir == "" {
		var err error
		tc.tmpDir, err = os.MkdirTemp("", "templar-test-*")
		require.NoError(tc.t, err, "Failed to create temporary directory")

		tc.cleanup = append(tc.cleanup, func() {
			if err := os.RemoveAll(tc.tmpDir); err != nil {
				tc.t.Logf("Warning: failed to remove temp dir %s: %v", tc.tmpDir, err)
			}
		})
	}

	return tc.tmpDir
}

// Cleanup executes all registered cleanup functions in reverse order.
func (tc *TestContext) Cleanup() {
	tc.cancel()

	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Execute cleanup functions in reverse order (LIFO)
	for i := len(tc.cleanup) - 1; i >= 0; i-- {
		func() {
			defer func() {
				if r := recover(); r != nil {
					tc.t.Logf("Cleanup function panicked: %v", r)
				}
			}()
			tc.cleanup[i]()
		}()
	}
}

// TestFileSystem provides utilities for creating test files and directories.
type TestFileSystem struct {
	baseDir string
	t       *testing.T
}

// NewTestFileSystem creates a new test filesystem in the given base directory.
func NewTestFileSystem(t *testing.T, baseDir string) *TestFileSystem {
	return &TestFileSystem{
		baseDir: baseDir,
		t:       t,
	}
}

// NewBenchmarkFileSystem creates a new test file system for benchmarks.
func NewBenchmarkFileSystem(b *testing.B, baseDir string) *TestFileSystem {
	return &TestFileSystem{
		baseDir: baseDir,
		t:       nil, // We'll handle this carefully in methods
	}
}

// CreateFile creates a file with the given content at the specified path.
func (tfs *TestFileSystem) CreateFile(relativePath, content string) string {
	fullPath := filepath.Join(tfs.baseDir, relativePath)

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	require.NoError(tfs.t, os.MkdirAll(dir, 0755), "Failed to create directory: %s", dir)

	// Write file
	require.NoError(tfs.t, os.WriteFile(fullPath, []byte(content), 0644), "Failed to write file: %s", fullPath)

	return fullPath
}

// CreateDir creates a directory at the specified path.
func (tfs *TestFileSystem) CreateDir(relativePath string) string {
	fullPath := filepath.Join(tfs.baseDir, relativePath)
	require.NoError(tfs.t, os.MkdirAll(fullPath, 0755), "Failed to create directory: %s", fullPath)

	return fullPath
}

// WriteTemplFile creates a templ component file with the given content.
func (tfs *TestFileSystem) WriteTemplFile(name, content string) string {
	filename := name + ".templ"

	return tfs.CreateFile(filename, content)
}

// WriteGoFile creates a Go file with the given content.
func (tfs *TestFileSystem) WriteGoFile(name, content string) string {
	filename := name + ".go"

	return tfs.CreateFile(filename, content)
}

// AssertFileExists verifies that a file exists at the given path.
func (tfs *TestFileSystem) AssertFileExists(relativePath string) {
	fullPath := filepath.Join(tfs.baseDir, relativePath)
	_, err := os.Stat(fullPath)
	assert.NoError(tfs.t, err, "File should exist: %s", fullPath)
}

// AssertFileNotExists verifies that a file does not exist at the given path.
func (tfs *TestFileSystem) AssertFileNotExists(relativePath string) {
	fullPath := filepath.Join(tfs.baseDir, relativePath)
	_, err := os.Stat(fullPath)
	assert.True(tfs.t, os.IsNotExist(err), "File should not exist: %s", fullPath)
}

// TestComponent represents a test component for testing purposes.
type TestComponent struct {
	Name       string
	Package    string
	FilePath   string
	Function   string
	Parameters []TestParameter
	Content    string
}

// TestParameter represents a parameter for testing purposes.
type TestParameter struct {
	Name string
	Type string
}

// GenerateTestComponent creates a test component with the given specifications.
func GenerateTestComponent(name, pkg string, params []TestParameter) TestComponent {
	var paramStrs []string
	for _, p := range params {
		paramStrs = append(paramStrs, fmt.Sprintf("%s %s", p.Name, p.Type))
	}

	content := fmt.Sprintf(`package %s

templ %s(%s) {
	<div class="test-component">
		<h1>Test Component: %s</h1>
	</div>
}
`, pkg, name, strings.Join(paramStrs, ", "), name)

	return TestComponent{
		Name:       name,
		Package:    pkg,
		FilePath:   strings.ToLower(name) + ".templ",
		Function:   name,
		Parameters: params,
		Content:    content,
	}
}

// TestServer provides utilities for testing HTTP servers.
type TestServer struct {
	listener net.Listener
	port     int
	t        *testing.T
}

// NewTestServer creates a test server that listens on a random available port.
func NewTestServer(t *testing.T) *TestServer {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "Failed to create test server listener")

	addr := listener.Addr().(*net.TCPAddr)

	return &TestServer{
		listener: listener,
		port:     addr.Port,
		t:        t,
	}
}

// Port returns the port the test server is listening on.
func (ts *TestServer) Port() int {
	return ts.port
}

// URL returns the base URL for the test server.
func (ts *TestServer) URL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", ts.port)
}

// Listener returns the underlying net.Listener.
func (ts *TestServer) Listener() net.Listener {
	return ts.listener
}

// Close closes the test server.
func (ts *TestServer) Close() error {
	return ts.listener.Close()
}

// CaptureOutput captures stdout and stderr for testing purposes.
type CaptureOutput struct {
	origStdout *os.File
	origStderr *os.File
	stdoutR    *os.File
	stdoutW    *os.File
	stderrR    *os.File
	stderrW    *os.File
	t          *testing.T
}

// NewCaptureOutput creates a new output capturer.
func NewCaptureOutput(t *testing.T) *CaptureOutput {
	stdoutR, stdoutW, err := os.Pipe()
	require.NoError(t, err, "Failed to create stdout pipe")

	stderrR, stderrW, err := os.Pipe()
	require.NoError(t, err, "Failed to create stderr pipe")

	return &CaptureOutput{
		origStdout: os.Stdout,
		origStderr: os.Stderr,
		stdoutR:    stdoutR,
		stdoutW:    stdoutW,
		stderrR:    stderrR,
		stderrW:    stderrW,
		t:          t,
	}
}

// Start begins capturing output.
func (co *CaptureOutput) Start() {
	os.Stdout = co.stdoutW
	os.Stderr = co.stderrW

	// Also redirect log output
	log.SetOutput(co.stderrW)
}

// Stop stops capturing and returns the captured stdout and stderr.
func (co *CaptureOutput) Stop() (stdout, stderr string) {
	// Restore original outputs
	os.Stdout = co.origStdout
	os.Stderr = co.origStderr
	log.SetOutput(co.origStderr)

	// Close writers
	if err := co.stdoutW.Close(); err != nil {
		co.t.Logf("Warning: failed to close stdout writer: %v", err)
	}
	if err := co.stderrW.Close(); err != nil {
		co.t.Logf("Warning: failed to close stderr writer: %v", err)
	}

	// Read captured output
	stdoutBytes, err := io.ReadAll(co.stdoutR)
	require.NoError(co.t, err, "Failed to read captured stdout")

	stderrBytes, err := io.ReadAll(co.stderrR)
	require.NoError(co.t, err, "Failed to read captured stderr")

	if err := co.stdoutR.Close(); err != nil {
		co.t.Logf("Warning: failed to close stdout reader: %v", err)
	}
	if err := co.stderrR.Close(); err != nil {
		co.t.Logf("Warning: failed to close stderr reader: %v", err)
	}

	return string(stdoutBytes), string(stderrBytes)
}

// TestTimer provides utilities for testing time-sensitive operations.
type TestTimer struct {
	start time.Time
	t     *testing.T
}

// NewTestTimer creates a new test timer.
func NewTestTimer(t *testing.T) *TestTimer {
	return &TestTimer{
		start: time.Now(),
		t:     t,
	}
}

// AssertDuration asserts that the elapsed time is within the expected range.
func (tt *TestTimer) AssertDuration(min, max time.Duration) {
	elapsed := time.Since(tt.start)
	assert.GreaterOrEqual(tt.t, elapsed, min, "Operation completed too quickly")
	assert.LessOrEqual(tt.t, elapsed, max, "Operation took too long")
}

// AssertWithinTimeout asserts that the elapsed time is less than the timeout.
func (tt *TestTimer) AssertWithinTimeout(timeout time.Duration) {
	elapsed := time.Since(tt.start)
	assert.Less(tt.t, elapsed, timeout, "Operation exceeded timeout")
}

// GetCaller returns information about the calling function for better test error messages.
func GetCaller() (file string, line int, function string) {
	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		return "unknown", 0, "unknown"
	}

	fn := runtime.FuncForPC(pc)
	if fn != nil {
		function = fn.Name()
	}

	return filepath.Base(file), line, function
}

// AssertEventually retries a condition until it passes or times out.
func AssertEventually(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	start := time.Now()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		if condition() {
			return
		}

		if time.Since(start) >= timeout {
			t.Fatalf("Condition never became true within %v: %s", timeout, message)
		}

		<-ticker.C
	}
}

// AssertNever asserts that a condition never becomes true within the timeout.
func AssertNever(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	start := time.Now()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		if condition() {
			t.Fatalf("Condition became true when it should never have: %s", message)
		}

		if time.Since(start) >= timeout {
			return // Success - condition never became true
		}

		<-ticker.C
	}
}
