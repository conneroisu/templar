package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventTypeString(t *testing.T) {
	testCases := []struct {
		eventType EventType
		expected  string
	}{
		{EventTypeCreated, "created"},
		{EventTypeModified, "modified"},
		{EventTypeDeleted, "deleted"},
		{EventTypeRenamed, "renamed"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.eventType.String())
		})
	}
}

func TestNewFileWatcher(t *testing.T) {
	watcher, err := NewFileWatcher(100 * time.Millisecond)
	require.NoError(t, err)
	defer watcher.Stop()

	assert.NotNil(t, watcher.watcher)
	assert.NotNil(t, watcher.debouncer)
	assert.Empty(t, watcher.filters)
	assert.Empty(t, watcher.handlers)
}

func TestFileWatcherAddFilter(t *testing.T) {
	watcher, err := NewFileWatcher(100 * time.Millisecond)
	require.NoError(t, err)
	defer watcher.Stop()

	// Add templ filter
	watcher.AddFilter(TemplFilter)
	assert.Len(t, watcher.filters, 1)

	// Add go filter
	watcher.AddFilter(GoFilter)
	assert.Len(t, watcher.filters, 2)
}

func TestFileWatcherAddHandler(t *testing.T) {
	watcher, err := NewFileWatcher(100 * time.Millisecond)
	require.NoError(t, err)
	defer watcher.Stop()

	handlerCalled := false
	handler := func(events []ChangeEvent) error {
		handlerCalled = true
		return nil
	}

	watcher.AddHandler(handler)
	assert.Len(t, watcher.handlers, 1)

	// Simulate calling handler
	watcher.mutex.RLock()
	for _, h := range watcher.handlers {
		h([]ChangeEvent{{Type: EventTypeCreated, Path: "test.go"}})
	}
	watcher.mutex.RUnlock()

	assert.True(t, handlerCalled)
}

func TestFileWatcherAddPath(t *testing.T) {
	watcher, err := NewFileWatcher(100 * time.Millisecond)
	require.NoError(t, err)
	defer watcher.Stop()

	// Create temporary directory within current working directory
	tempDir := "test_temp_dir"
	err = os.MkdirAll(tempDir, 0755)
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Test watching directory
	err = watcher.AddPath(tempDir)
	assert.NoError(t, err)

	// Test watching non-existent path
	err = watcher.AddPath("/non/existent/path")
	assert.Error(t, err)
}

func TestFileWatcherStartStop(t *testing.T) {
	watcher, err := NewFileWatcher(50 * time.Millisecond)
	require.NoError(t, err)
	defer watcher.Stop()

	// Create temporary directory within current working directory
	tempDir := "test_temp_start_stop"
	err = os.MkdirAll(tempDir, 0755)
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	err = watcher.AddPath(tempDir)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var eventReceived bool
	var eventMutex sync.Mutex

	watcher.AddHandler(func(events []ChangeEvent) error {
		eventMutex.Lock()
		eventReceived = true
		eventMutex.Unlock()
		return nil
	})

	// Start watching
	err = watcher.Start(ctx)
	require.NoError(t, err)

	// Give watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Create a file to trigger event
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	// Wait for debouncing and event processing
	time.Sleep(200 * time.Millisecond)

	eventMutex.Lock()
	received := eventReceived
	eventMutex.Unlock()

	assert.True(t, received)

	// Test stop
	cancel()
	err = watcher.Stop()
	assert.NoError(t, err)
}

func TestTemplFilter(t *testing.T) {
	testCases := []struct {
		path     string
		expected bool
	}{
		{"main.go", false},
		{"component.templ", true},
		{"script.js", false},
		{"style.css", false},
		{"README.md", false},
		{"test", false},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			result := TemplFilter(tc.path)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGoFilter(t *testing.T) {
	testCases := []struct {
		path     string
		expected bool
	}{
		{"main.go", true},
		{"component.templ", false},
		{"script.js", false},
		{"style.css", false},
		{"README.md", false},
		{"test", false},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			result := GoFilter(tc.path)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNoTestFilter(t *testing.T) {
	testCases := []struct {
		path     string
		expected bool
	}{
		{"main.go", true},
		{"main_test.go", false},
		{"component.templ", true},
		{"component_test.templ", false},
		{"other.js", true},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			result := NoTestFilter(tc.path)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNoVendorFilter(t *testing.T) {
	testCases := []struct {
		path     string
		expected bool
	}{
		{"src/main.go", true},
		{"vendor/package/index.js", false},
		{"src/vendor/test.go", false},
		{"main.go", true},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			result := NoVendorFilter(tc.path)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNoGitFilter(t *testing.T) {
	testCases := []struct {
		path     string
		expected bool
	}{
		{"src/main.go", true},
		{".git/config", false},
		{"src/.git/test.go", false},
		{"main.go", true},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			result := NoGitFilter(tc.path)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDebouncer(t *testing.T) {
	debouncer := &Debouncer{
		delay:   50 * time.Millisecond,
		events:  make(chan ChangeEvent, 100),
		output:  make(chan []ChangeEvent, 10),
		pending: make([]ChangeEvent, 0),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start debouncer
	go debouncer.start(ctx)

	var receivedEvents [][]ChangeEvent
	var eventMutex sync.Mutex

	// Listen for debounced events
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case events := <-debouncer.output:
				eventMutex.Lock()
				receivedEvents = append(receivedEvents, events)
				eventMutex.Unlock()
			}
		}
	}()

	// Send multiple events quickly
	debouncer.events <- ChangeEvent{Path: "test1.go", Type: EventTypeModified}
	debouncer.events <- ChangeEvent{Path: "test1.go", Type: EventTypeModified}
	debouncer.events <- ChangeEvent{Path: "test2.go", Type: EventTypeModified}

	// Wait for debouncing
	time.Sleep(150 * time.Millisecond)

	eventMutex.Lock()
	finalEvents := receivedEvents
	eventMutex.Unlock()

	// Should have received at least one batch of events
	assert.Greater(t, len(finalEvents), 0)
	if len(finalEvents) > 0 {
		// Should have deduplicated test1.go and kept test2.go
		assert.LessOrEqual(t, len(finalEvents[0]), 2)
	}
}

func TestChangeEvent(t *testing.T) {
	now := time.Now()
	event := ChangeEvent{
		Type:    EventTypeModified,
		Path:    "/path/to/file.go",
		ModTime: now,
		Size:    1024,
	}

	assert.Equal(t, EventTypeModified, event.Type)
	assert.Equal(t, "/path/to/file.go", event.Path)
	assert.Equal(t, now, event.ModTime)
	assert.Equal(t, int64(1024), event.Size)
}

func TestFileWatcherValidation(t *testing.T) {
	watcher, err := NewFileWatcher(100 * time.Millisecond)
	require.NoError(t, err)
	defer watcher.Stop()

	// Test watching with path traversal
	err = watcher.AddPath("../../../etc")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path")

	// Test watching relative path that resolves outside cwd
	err = watcher.AddPath("./../../..")
	assert.Error(t, err)
}

func TestFileWatcherConcurrency(t *testing.T) {
	watcher, err := NewFileWatcher(50 * time.Millisecond)
	require.NoError(t, err)
	defer watcher.Stop()

	// Create temporary directory within current working directory
	tempDir := "test_temp_concurrency"
	err = os.MkdirAll(tempDir, 0755)
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	err = watcher.AddPath(tempDir)
	require.NoError(t, err)

	var wg sync.WaitGroup
	var eventCount int
	var eventMutex sync.Mutex

	// Add handler
	watcher.AddHandler(func(events []ChangeEvent) error {
		eventMutex.Lock()
		eventCount += len(events)
		eventMutex.Unlock()
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start watcher
	err = watcher.Start(ctx)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	// Create multiple files concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			testFile := filepath.Join(tempDir, fmt.Sprintf("test%d.txt", i))
			err := os.WriteFile(testFile, []byte("test"), 0644)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Wait for all events to be processed
	time.Sleep(200 * time.Millisecond)

	eventMutex.Lock()
	finalCount := eventCount
	eventMutex.Unlock()

	// Should have received events (exact count may vary due to debouncing)
	assert.Greater(t, finalCount, 0)
	assert.LessOrEqual(t, finalCount, 10)
}

func TestFileWatcherErrorHandling(t *testing.T) {
	watcher, err := NewFileWatcher(100 * time.Millisecond)
	require.NoError(t, err)

	// Test double stop
	err = watcher.Stop()
	assert.NoError(t, err)
	err = watcher.Stop()
	assert.NoError(t, err) // Should not error on double stop
}

func TestAddRecursive(t *testing.T) {
	watcher, err := NewFileWatcher(100 * time.Millisecond)
	require.NoError(t, err)
	defer watcher.Stop()

	// Create temporary directory with subdirectories within current working directory
	tempDir := "test_temp_recursive"
	subDir := filepath.Join(tempDir, "subdir")
	err = os.MkdirAll(subDir, 0755)
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Test adding recursively
	err = watcher.AddRecursive(tempDir)
	assert.NoError(t, err)

	// Test with invalid path
	err = watcher.AddRecursive("../../../etc")
	assert.Error(t, err)
}