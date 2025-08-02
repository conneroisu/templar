package testutils

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// FileSystemAdapter provides an interface for file system operations that can be mocked.
type FileSystemAdapter interface {
	ReadFile(filename string) ([]byte, error)
	WriteFile(filename string, data []byte, perm os.FileMode) error
	Stat(name string) (fs.FileInfo, error)
	MkdirAll(path string, perm os.FileMode) error
	Remove(name string) error
	RemoveAll(path string) error
	Chmod(name string, mode os.FileMode) error
	Open(name string) (*os.File, error)
}

// NetworkAdapter provides an interface for network operations that can be mocked.
type NetworkAdapter interface {
	Get(url string) (*http.Response, error)
	Post(url, contentType string, body any) (*http.Response, error)
	Do(req *http.Request) (*http.Response, error)
}

// TimeAdapter provides an interface for time operations that can be mocked.
type TimeAdapter interface {
	Now() time.Time
	Sleep(d time.Duration)
	Since(t time.Time) time.Duration
	Until(t time.Time) time.Duration
	After(d time.Duration) <-chan time.Time
	NewTimer(d time.Duration) *time.Timer
	NewTicker(d time.Duration) *time.Ticker
}

// CommandAdapter provides an interface for command execution that can be mocked.
type CommandAdapter interface {
	RunCommand(ctx context.Context, name string, args ...string) (*CommandResult, error)
	RunCommandWithDir(ctx context.Context, dir, name string, args ...string) (*CommandResult, error)
	LookPath(file string) (string, error)
}

// RealFileSystemAdapter implements FileSystemAdapter using real os operations.
type RealFileSystemAdapter struct{}

// NewRealFileSystemAdapter creates a new real file system adapter.
func NewRealFileSystemAdapter() *RealFileSystemAdapter {
	return &RealFileSystemAdapter{}
}

func (r *RealFileSystemAdapter) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

func (r *RealFileSystemAdapter) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return os.WriteFile(filename, data, perm)
}

func (r *RealFileSystemAdapter) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

func (r *RealFileSystemAdapter) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (r *RealFileSystemAdapter) Remove(name string) error {
	return os.Remove(name)
}

func (r *RealFileSystemAdapter) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (r *RealFileSystemAdapter) Chmod(name string, mode os.FileMode) error {
	return os.Chmod(name, mode)
}

func (r *RealFileSystemAdapter) Open(name string) (*os.File, error) {
	return os.Open(name)
}

// RealNetworkAdapter implements NetworkAdapter using real http operations.
type RealNetworkAdapter struct {
	client *http.Client
}

// NewRealNetworkAdapter creates a new real network adapter.
func NewRealNetworkAdapter() *RealNetworkAdapter {
	return &RealNetworkAdapter{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (r *RealNetworkAdapter) Get(url string) (*http.Response, error) {
	return r.client.Get(url)
}

func (r *RealNetworkAdapter) Post(url, contentType string, body any) (*http.Response, error) {
	// Validate URL to prevent security issues
	if url == "" {
		return nil, errors.New("empty URL not allowed")
	}
	// For tests, only allow localhost and 127.0.0.1
	if !strings.HasPrefix(url, "http://localhost:") && !strings.HasPrefix(url, "http://127.0.0.1:") {
		return nil, errors.New("only localhost URLs allowed in tests")
	}
	// Validated URL is safe to use
	safeURL := url

	return http.Post(safeURL, contentType, nil) //nolint:gosec // URL validated above
}

func (r *RealNetworkAdapter) Do(req *http.Request) (*http.Response, error) {
	return r.client.Do(req)
}

// RealTimeAdapter implements TimeAdapter using real time operations.
type RealTimeAdapter struct{}

// NewRealTimeAdapter creates a new real time adapter.
func NewRealTimeAdapter() *RealTimeAdapter {
	return &RealTimeAdapter{}
}

func (r *RealTimeAdapter) Now() time.Time {
	return time.Now()
}

func (r *RealTimeAdapter) Sleep(d time.Duration) {
	time.Sleep(d)
}

func (r *RealTimeAdapter) Since(t time.Time) time.Duration {
	return time.Since(t)
}

func (r *RealTimeAdapter) Until(t time.Time) time.Duration {
	return time.Until(t)
}

func (r *RealTimeAdapter) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

func (r *RealTimeAdapter) NewTimer(d time.Duration) *time.Timer {
	return time.NewTimer(d)
}

func (r *RealTimeAdapter) NewTicker(d time.Duration) *time.Ticker {
	return time.NewTicker(d)
}

// RealCommandAdapter implements CommandAdapter using real command execution.
type RealCommandAdapter struct{}

// NewRealCommandAdapter creates a new real command adapter.
func NewRealCommandAdapter() *RealCommandAdapter {
	return &RealCommandAdapter{}
}

func (r *RealCommandAdapter) RunCommand(ctx context.Context, name string, args ...string) (*CommandResult, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()

	result := &CommandResult{
		Stdout:   string(output),
		Stderr:   "",
		ExitCode: 0,
		Error:    err,
	}

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		}
	}

	return result, err
}

func (r *RealCommandAdapter) RunCommandWithDir(ctx context.Context, dir, name string, args ...string) (*CommandResult, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()

	result := &CommandResult{
		Stdout:   string(output),
		Stderr:   "",
		ExitCode: 0,
		Error:    err,
	}

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		}
	}

	return result, err
}

func (r *RealCommandAdapter) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

// MockFileSystemAdapter adapts MockFileSystem to FileSystemAdapter interface.
type MockFileSystemAdapter struct {
	mock *MockFileSystem
}

// NewMockFileSystemAdapter creates a new mock file system adapter.
func NewMockFileSystemAdapter(mock *MockFileSystem) *MockFileSystemAdapter {
	return &MockFileSystemAdapter{mock: mock}
}

func (m *MockFileSystemAdapter) ReadFile(filename string) ([]byte, error) {
	return m.mock.ReadFile(filename)
}

func (m *MockFileSystemAdapter) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return m.mock.WriteFile(filename, data, perm)
}

func (m *MockFileSystemAdapter) Stat(name string) (fs.FileInfo, error) {
	return m.mock.Stat(name)
}

func (m *MockFileSystemAdapter) MkdirAll(path string, perm os.FileMode) error {
	return m.mock.MkdirAll(path, perm)
}

func (m *MockFileSystemAdapter) Remove(name string) error {
	return m.mock.Remove(name)
}

func (m *MockFileSystemAdapter) RemoveAll(path string) error {
	args := m.mock.Called(path)

	return args.Error(0)
}

func (m *MockFileSystemAdapter) Chmod(name string, mode os.FileMode) error {
	args := m.mock.Called(name, mode)

	return args.Error(0)
}

func (m *MockFileSystemAdapter) Open(name string) (*os.File, error) {
	args := m.mock.Called(name)

	return args.Get(0).(*os.File), args.Error(1)
}

// MockNetworkAdapter adapts MockNetwork to NetworkAdapter interface.
type MockNetworkAdapter struct {
	mock *MockNetwork
}

// NewMockNetworkAdapter creates a new mock network adapter.
func NewMockNetworkAdapter(mock *MockNetwork) *MockNetworkAdapter {
	return &MockNetworkAdapter{mock: mock}
}

func (m *MockNetworkAdapter) Get(url string) (*http.Response, error) {
	return m.mock.Get(url)
}

func (m *MockNetworkAdapter) Post(url, contentType string, body any) (*http.Response, error) {
	return m.mock.Post(url, contentType, nil)
}

func (m *MockNetworkAdapter) Do(req *http.Request) (*http.Response, error) {
	args := m.mock.Called(req)

	return args.Get(0).(*http.Response), args.Error(1)
}

// MockTimeAdapter adapts MockTime to TimeAdapter interface.
type MockTimeAdapter struct {
	mock *MockTime
}

// NewMockTimeAdapter creates a new mock time adapter.
func NewMockTimeAdapter(mock *MockTime) *MockTimeAdapter {
	return &MockTimeAdapter{mock: mock}
}

func (m *MockTimeAdapter) Now() time.Time {
	return m.mock.Now()
}

func (m *MockTimeAdapter) Sleep(d time.Duration) {
	m.mock.Sleep(d)
}

func (m *MockTimeAdapter) Since(t time.Time) time.Duration {
	args := m.mock.Called(t)

	return args.Get(0).(time.Duration)
}

func (m *MockTimeAdapter) Until(t time.Time) time.Duration {
	args := m.mock.Called(t)

	return args.Get(0).(time.Duration)
}

func (m *MockTimeAdapter) After(d time.Duration) <-chan time.Time {
	args := m.mock.Called(d)

	return args.Get(0).(<-chan time.Time)
}

func (m *MockTimeAdapter) NewTimer(d time.Duration) *time.Timer {
	args := m.mock.Called(d)

	return args.Get(0).(*time.Timer)
}

func (m *MockTimeAdapter) NewTicker(d time.Duration) *time.Ticker {
	args := m.mock.Called(d)

	return args.Get(0).(*time.Ticker)
}

// MockCommandAdapter adapts MockCommandRunner to CommandAdapter interface.
type MockCommandAdapter struct {
	mock *MockCommandRunner
}

// NewMockCommandAdapter creates a new mock command adapter.
func NewMockCommandAdapter(mock *MockCommandRunner) *MockCommandAdapter {
	return &MockCommandAdapter{mock: mock}
}

func (m *MockCommandAdapter) RunCommand(ctx context.Context, name string, args ...string) (*CommandResult, error) {
	return m.mock.RunCommand(ctx, name, args...)
}

func (m *MockCommandAdapter) RunCommandWithDir(ctx context.Context, dir, name string, args ...string) (*CommandResult, error) {
	mockArgs := m.mock.Called(ctx, dir, name, args)

	return mockArgs.Get(0).(*CommandResult), mockArgs.Error(1)
}

func (m *MockCommandAdapter) LookPath(file string) (string, error) {
	args := m.mock.Called(file)

	return args.String(0), args.Error(1)
}

// TestEnvironmentAdapters contains all adapters for testing environment.
type TestEnvironmentAdapters struct {
	FileSystem FileSystemAdapter
	Network    NetworkAdapter
	Time       TimeAdapter
	Command    CommandAdapter
}

// NewRealAdapters creates adapters using real implementations.
func NewRealAdapters() *TestEnvironmentAdapters {
	return &TestEnvironmentAdapters{
		FileSystem: NewRealFileSystemAdapter(),
		Network:    NewRealNetworkAdapter(),
		Time:       NewRealTimeAdapter(),
		Command:    NewRealCommandAdapter(),
	}
}

// NewMockAdapters creates adapters using mock implementations.
func NewMockAdapters(mf *MockFramework) *TestEnvironmentAdapters {
	return &TestEnvironmentAdapters{
		FileSystem: NewMockFileSystemAdapter(mf.FileSystem),
		Network:    NewMockNetworkAdapter(mf.Network),
		Time:       NewMockTimeAdapter(mf.Time),
		Command:    NewMockCommandAdapter(mf.CommandRunner),
	}
}

// WithMockFileSystem creates adapters with only file system mocked.
func (tea *TestEnvironmentAdapters) WithMockFileSystem(mf *MockFramework) *TestEnvironmentAdapters {
	return &TestEnvironmentAdapters{
		FileSystem: NewMockFileSystemAdapter(mf.FileSystem),
		Network:    tea.Network,
		Time:       tea.Time,
		Command:    tea.Command,
	}
}

// WithMockNetwork creates adapters with only network mocked.
func (tea *TestEnvironmentAdapters) WithMockNetwork(mf *MockFramework) *TestEnvironmentAdapters {
	return &TestEnvironmentAdapters{
		FileSystem: tea.FileSystem,
		Network:    NewMockNetworkAdapter(mf.Network),
		Time:       tea.Time,
		Command:    tea.Command,
	}
}

// WithMockTime creates adapters with only time mocked.
func (tea *TestEnvironmentAdapters) WithMockTime(mf *MockFramework) *TestEnvironmentAdapters {
	return &TestEnvironmentAdapters{
		FileSystem: tea.FileSystem,
		Network:    tea.Network,
		Time:       NewMockTimeAdapter(mf.Time),
		Command:    tea.Command,
	}
}

// WithMockCommand creates adapters with only command execution mocked.
func (tea *TestEnvironmentAdapters) WithMockCommand(mf *MockFramework) *TestEnvironmentAdapters {
	return &TestEnvironmentAdapters{
		FileSystem: tea.FileSystem,
		Network:    tea.Network,
		Time:       tea.Time,
		Command:    NewMockCommandAdapter(mf.CommandRunner),
	}
}
