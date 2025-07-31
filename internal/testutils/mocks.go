package testutils

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
)

// MockFramework provides comprehensive mocking capabilities for testing.
type MockFramework struct {
	t             *testing.T
	FileSystem    *MockFileSystem
	Network       *MockNetwork
	Time          *MockTime
	CommandRunner *MockCommandRunner
	cleanupFuncs  []func()
	mu            sync.RWMutex
}

// NewMockFramework creates a new mock framework instance.
func NewMockFramework(t *testing.T) *MockFramework {
	return &MockFramework{
		t:             t,
		FileSystem:    NewMockFileSystem(),
		Network:       NewMockNetwork(),
		Time:          NewMockTime(),
		CommandRunner: NewMockCommandRunner(),
		cleanupFuncs:  make([]func(), 0),
	}
}

// Reset resets all mocks to their initial state.
func (mf *MockFramework) Reset() {
	mf.mu.Lock()
	defer mf.mu.Unlock()

	mf.FileSystem.Reset()
	mf.Network.Reset()
	mf.Time.Reset()
	mf.CommandRunner.Reset()
}

// Cleanup performs cleanup operations for all mocks.
func (mf *MockFramework) Cleanup() {
	mf.mu.Lock()
	defer mf.mu.Unlock()

	for i := len(mf.cleanupFuncs) - 1; i >= 0; i-- {
		mf.cleanupFuncs[i]()
	}
	mf.cleanupFuncs = mf.cleanupFuncs[:0]
}

// AddCleanup adds a cleanup function to be called during framework cleanup.
func (mf *MockFramework) AddCleanup(cleanup func()) {
	mf.mu.Lock()
	defer mf.mu.Unlock()
	mf.cleanupFuncs = append(mf.cleanupFuncs, cleanup)
}

// MockFileSystem provides comprehensive file system mocking.
type MockFileSystem struct {
	mock.Mock
	files       map[string]*MockFile
	permissions map[string]os.FileMode
	mu          sync.RWMutex
}

// MockFile represents a mock file with content and metadata.
type MockFile struct {
	Content  []byte
	Mode     os.FileMode
	ModTime  time.Time
	IsDir    bool
	Entries  []fs.DirEntry
	ReadPos  int64
	WritePos int64
	closed   bool         //nolint:unused  // TODO: implement file closing tracking
	mu       sync.RWMutex //nolint:unused  // TODO: implement proper concurrency control
}

// NewMockFileSystem creates a new mock file system.
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		files:       make(map[string]*MockFile),
		permissions: make(map[string]os.FileMode),
	}
}

// Reset resets the mock file system to its initial state.
func (mfs *MockFileSystem) Reset() {
	mfs.mu.Lock()
	defer mfs.mu.Unlock()

	mfs.ExpectedCalls = nil
	mfs.Calls = nil
	mfs.files = make(map[string]*MockFile)
	mfs.permissions = make(map[string]os.FileMode)
}

// CreateFile creates a mock file with the specified content.
func (mfs *MockFileSystem) CreateFile(path string, content []byte, mode os.FileMode) {
	mfs.mu.Lock()
	defer mfs.mu.Unlock()

	mfs.files[path] = &MockFile{
		Content: content,
		Mode:    mode,
		ModTime: time.Now(),
		IsDir:   false,
	}
	mfs.permissions[path] = mode
}

// CreateDir creates a mock directory.
func (mfs *MockFileSystem) CreateDir(path string, mode os.FileMode) {
	mfs.mu.Lock()
	defer mfs.mu.Unlock()

	mfs.files[path] = &MockFile{
		Mode:    mode | fs.ModeDir,
		ModTime: time.Now(),
		IsDir:   true,
		Entries: make([]fs.DirEntry, 0),
	}
	mfs.permissions[path] = mode
}

// ReadFile mocks os.ReadFile functionality.
func (mfs *MockFileSystem) ReadFile(filename string) ([]byte, error) {
	args := mfs.Called(filename)

	mfs.mu.RLock()
	defer mfs.mu.RUnlock()

	if file, exists := mfs.files[filename]; exists {
		if file.IsDir {
			return nil, fmt.Errorf("read %s: is a directory", filename)
		}

		return file.Content, args.Error(1)
	}

	return args.Get(0).([]byte), args.Error(1)
}

// WriteFile mocks os.WriteFile functionality.
func (mfs *MockFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	args := mfs.Called(filename, data, perm)

	mfs.mu.Lock()
	defer mfs.mu.Unlock()

	mfs.files[filename] = &MockFile{
		Content: data,
		Mode:    perm,
		ModTime: time.Now(),
		IsDir:   false,
	}
	mfs.permissions[filename] = perm

	return args.Error(0)
}

// Stat mocks os.Stat functionality.
func (mfs *MockFileSystem) Stat(name string) (fs.FileInfo, error) {
	args := mfs.Called(name)

	mfs.mu.RLock()
	defer mfs.mu.RUnlock()

	if file, exists := mfs.files[name]; exists {
		return &MockFileInfo{
			name:    name,
			size:    int64(len(file.Content)),
			mode:    file.Mode,
			modTime: file.ModTime,
			isDir:   file.IsDir,
		}, args.Error(1)
	}

	if args.Get(0) != nil {
		return args.Get(0).(fs.FileInfo), args.Error(1)
	}

	return nil, args.Error(1)
}

// MkdirAll mocks os.MkdirAll functionality.
func (mfs *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	args := mfs.Called(path, perm)

	mfs.mu.Lock()
	defer mfs.mu.Unlock()

	mfs.files[path] = &MockFile{
		Mode:    perm | fs.ModeDir,
		ModTime: time.Now(),
		IsDir:   true,
		Entries: make([]fs.DirEntry, 0),
	}
	mfs.permissions[path] = perm

	return args.Error(0)
}

// Remove mocks os.Remove functionality.
func (mfs *MockFileSystem) Remove(name string) error {
	args := mfs.Called(name)

	mfs.mu.Lock()
	defer mfs.mu.Unlock()

	delete(mfs.files, name)
	delete(mfs.permissions, name)

	return args.Error(0)
}

// MockFileInfo implements fs.FileInfo for mock files.
type MockFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (mfi *MockFileInfo) Name() string       { return mfi.name }
func (mfi *MockFileInfo) Size() int64        { return mfi.size }
func (mfi *MockFileInfo) Mode() os.FileMode  { return mfi.mode }
func (mfi *MockFileInfo) ModTime() time.Time { return mfi.modTime }
func (mfi *MockFileInfo) IsDir() bool        { return mfi.isDir }
func (mfi *MockFileInfo) Sys() interface{}   { return nil }

// MockNetwork provides comprehensive network mocking capabilities.
type MockNetwork struct {
	mock.Mock
	httpResponses map[string]*http.Response
	httpErrors    map[string]error
	mu            sync.RWMutex
}

// NewMockNetwork creates a new mock network.
func NewMockNetwork() *MockNetwork {
	return &MockNetwork{
		httpResponses: make(map[string]*http.Response),
		httpErrors:    make(map[string]error),
	}
}

// Reset resets the mock network to its initial state.
func (mn *MockNetwork) Reset() {
	mn.mu.Lock()
	defer mn.mu.Unlock()

	mn.ExpectedCalls = nil
	mn.Calls = nil
	mn.httpResponses = make(map[string]*http.Response)
	mn.httpErrors = make(map[string]error)
}

// MockHTTPResponse configures a mock HTTP response for a given URL.
func (mn *MockNetwork) MockHTTPResponse(url string, response *http.Response, err error) {
	mn.mu.Lock()
	defer mn.mu.Unlock()

	mn.httpResponses[url] = response
	mn.httpErrors[url] = err
}

// Get mocks http.Get functionality.
func (mn *MockNetwork) Get(url string) (*http.Response, error) {
	args := mn.Called(url)

	mn.mu.RLock()
	defer mn.mu.RUnlock()

	if response, exists := mn.httpResponses[url]; exists {
		return response, mn.httpErrors[url]
	}

	return args.Get(0).(*http.Response), args.Error(1)
}

// Post mocks http.Post functionality.
func (mn *MockNetwork) Post(url, contentType string, body io.Reader) (*http.Response, error) {
	args := mn.Called(url, contentType, body)

	return args.Get(0).(*http.Response), args.Error(1)
}

// MockTime provides deterministic time mocking.
type MockTime struct {
	mock.Mock
	currentTime time.Time
	frozen      bool
	mu          sync.RWMutex
}

// NewMockTime creates a new mock time.
func NewMockTime() *MockTime {
	return &MockTime{
		currentTime: time.Now(),
		frozen:      false,
	}
}

// Reset resets the mock time to its initial state.
func (mt *MockTime) Reset() {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	mt.ExpectedCalls = nil
	mt.Calls = nil
	mt.currentTime = time.Now()
	mt.frozen = false
}

// Now returns the current mock time.
func (mt *MockTime) Now() time.Time {
	mt.Called()

	mt.mu.RLock()
	defer mt.mu.RUnlock()

	return mt.currentTime
}

// FreezeTime freezes time at the current value.
func (mt *MockTime) FreezeTime() {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	mt.frozen = true
}

// UnfreezeTime unfreezes time.
func (mt *MockTime) UnfreezeTime() {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	mt.frozen = false
}

// SetTime sets the current mock time.
func (mt *MockTime) SetTime(t time.Time) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	mt.currentTime = t
}

// AdvanceTime advances the mock time by the specified duration.
func (mt *MockTime) AdvanceTime(d time.Duration) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	if !mt.frozen {
		mt.currentTime = mt.currentTime.Add(d)
	}
}

// Sleep mocks time.Sleep functionality.
func (mt *MockTime) Sleep(d time.Duration) {
	mt.Called(d)
	mt.AdvanceTime(d)
}

// MockCommandRunner provides command execution mocking.
type MockCommandRunner struct {
	mock.Mock
	commands map[string]*CommandResult
	mu       sync.RWMutex
}

// CommandResult represents the result of a command execution.
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Error    error
}

// NewMockCommandRunner creates a new mock command runner.
func NewMockCommandRunner() *MockCommandRunner {
	return &MockCommandRunner{
		commands: make(map[string]*CommandResult),
	}
}

// Reset resets the mock command runner to its initial state.
func (mcr *MockCommandRunner) Reset() {
	mcr.mu.Lock()
	defer mcr.mu.Unlock()

	mcr.ExpectedCalls = nil
	mcr.Calls = nil
	mcr.commands = make(map[string]*CommandResult)
}

// MockCommand configures a mock command result.
func (mcr *MockCommandRunner) MockCommand(cmd string, result *CommandResult) {
	mcr.mu.Lock()
	defer mcr.mu.Unlock()
	mcr.commands[cmd] = result
}

// RunCommand mocks command execution.
func (mcr *MockCommandRunner) RunCommand(ctx context.Context, name string, args ...string) (*CommandResult, error) {
	cmdStr := fmt.Sprintf("%s %v", name, args)
	mockArgs := mcr.Called(ctx, name, args)

	mcr.mu.RLock()
	defer mcr.mu.RUnlock()

	if result, exists := mcr.commands[cmdStr]; exists {
		return result, result.Error
	}

	return mockArgs.Get(0).(*CommandResult), mockArgs.Error(1)
}
