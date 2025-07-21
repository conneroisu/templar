// Package watcher provides real-time file system monitoring with debouncing
// and recursive directory watching capabilities.
//
// The watcher monitors file system changes for .templ files and triggers
// component rescanning and rebuilding. It implements debouncing to prevent
// excessive rebuilds during rapid file changes, supports recursive directory
// monitoring with configurable ignore patterns, and provides safe goroutine
// lifecycle management with proper context cancellation.
package watcher

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Constants for memory management
const (
	MaxPendingEvents = 1000             // Maximum events to queue before dropping
	CleanupInterval  = 30 * time.Second // How often to cleanup old state
)

// Object pools for memory efficiency
var (
	eventPool = sync.Pool{
		New: func() interface{} {
			return make([]ChangeEvent, 0, 100)
		},
	}

	eventMapPool = sync.Pool{
		New: func() interface{} {
			return make(map[string]ChangeEvent, 100)
		},
	}
)

// FileWatcher watches for file changes with intelligent debouncing
type FileWatcher struct {
	watcher   *fsnotify.Watcher
	debouncer *Debouncer
	filters   []FileFilter
	handlers  []ChangeHandler
	mutex     sync.RWMutex
	stopped   bool
}

// ChangeEvent represents a file change event
type ChangeEvent struct {
	Type    EventType
	Path    string
	ModTime time.Time
	Size    int64
}

// EventType represents the type of file change
type EventType int

const (
	EventTypeCreated EventType = iota
	EventTypeModified
	EventTypeDeleted
	EventTypeRenamed
)

// String returns the string representation of the EventType
func (e EventType) String() string {
	switch e {
	case EventTypeCreated:
		return "created"
	case EventTypeModified:
		return "modified"
	case EventTypeDeleted:
		return "deleted"
	case EventTypeRenamed:
		return "renamed"
	default:
		return "unknown"
	}
}

// FileFilter determines if a file should be watched
type FileFilter func(path string) bool

// ChangeHandler handles file change events
type ChangeHandler func(events []ChangeEvent) error

// Debouncer groups rapid file changes together
type Debouncer struct {
	delay        time.Duration
	events       chan ChangeEvent
	output       chan []ChangeEvent
	timer        *time.Timer
	pending      []ChangeEvent
	mutex        sync.Mutex
	cleanupTimer *time.Timer
	lastCleanup  time.Time
}

// NewFileWatcher creates a new file watcher
func NewFileWatcher(debounceDelay time.Duration) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	debouncer := &Debouncer{
		delay:       debounceDelay,
		events:      make(chan ChangeEvent, 100),
		output:      make(chan []ChangeEvent, 10),
		pending:     make([]ChangeEvent, 0, 100),
		lastCleanup: time.Now(),
	}

	fw := &FileWatcher{
		watcher:   watcher,
		debouncer: debouncer,
		filters:   make([]FileFilter, 0),
		handlers:  make([]ChangeHandler, 0),
	}

	return fw, nil
}

// AddFilter adds a file filter
func (fw *FileWatcher) AddFilter(filter FileFilter) {
	fw.mutex.Lock()
	defer fw.mutex.Unlock()
	fw.filters = append(fw.filters, filter)
}

// AddHandler adds a change handler
func (fw *FileWatcher) AddHandler(handler ChangeHandler) {
	fw.mutex.Lock()
	defer fw.mutex.Unlock()
	fw.handlers = append(fw.handlers, handler)
}

// AddPath adds a path to watch
func (fw *FileWatcher) AddPath(path string) error {
	// Validate and clean the path
	cleanPath, err := fw.validatePath(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	return fw.watcher.Add(cleanPath)
}

// AddRecursive adds a directory and all subdirectories to watch
func (fw *FileWatcher) AddRecursive(root string) error {
	// Validate and clean the root path
	cleanRoot, err := fw.validatePath(root)
	if err != nil {
		return fmt.Errorf("invalid root path: %w", err)
	}

	return filepath.Walk(cleanRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Validate each directory path before adding
			cleanPath, err := fw.validatePath(path)
			if err != nil {
				log.Printf("Skipping invalid directory path: %s", path)
				return nil
			}
			return fw.watcher.Add(cleanPath)
		}

		return nil
	})
}

// validatePath validates and cleans a file path to prevent directory traversal
func (fw *FileWatcher) validatePath(path string) (string, error) {
	// Clean the path to resolve . and .. elements
	cleanPath := filepath.Clean(path)

	// Get absolute path to normalize
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("getting absolute path: %w", err)
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting current directory: %w", err)
	}

	// Ensure the path is within the current working directory or its subdirectories
	// This prevents directory traversal attacks
	if !strings.HasPrefix(absPath, cwd) {
		return "", fmt.Errorf("path %s is outside current working directory", path)
	}

	// Additional security check: reject paths with suspicious patterns
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("path contains directory traversal: %s", path)
	}

	return cleanPath, nil
}

// Start starts the file watcher
func (fw *FileWatcher) Start(ctx context.Context) error {
	// Start debouncer
	go fw.debouncer.start(ctx)

	// Start event processor
	go fw.processEvents(ctx)

	// Start main watcher loop
	go fw.watchLoop(ctx)

	return nil
}

// Stop stops the file watcher and cleans up resources
func (fw *FileWatcher) Stop() error {
	fw.mutex.Lock()
	defer fw.mutex.Unlock()

	// Check if already stopped to prevent double-close
	if fw.stopped {
		return nil
	}
	fw.stopped = true

	fw.debouncer.mutex.Lock()
	defer fw.debouncer.mutex.Unlock()

	// Stop and cleanup all timers
	if fw.debouncer.timer != nil {
		fw.debouncer.timer.Stop()
		fw.debouncer.timer = nil
	}

	if fw.debouncer.cleanupTimer != nil {
		fw.debouncer.cleanupTimer.Stop()
		fw.debouncer.cleanupTimer = nil
	}

	// Clear pending events to release memory
	fw.debouncer.pending = nil

	// Close the file system watcher (this will close its internal channels)
	return fw.watcher.Close()
}

func (fw *FileWatcher) watchLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-fw.watcher.Events:
			fw.handleFsnotifyEvent(event)
		case err := <-fw.watcher.Errors:
			// Log error but continue watching
			log.Printf("File watcher error: %v", err)
		}
	}
}

func (fw *FileWatcher) handleFsnotifyEvent(event fsnotify.Event) {
	// Apply filters
	fw.mutex.RLock()
	filters := fw.filters
	fw.mutex.RUnlock()

	for _, filter := range filters {
		if !filter(event.Name) {
			return
		}
	}

	// Get file info
	info, err := os.Stat(event.Name)
	var modTime time.Time
	var size int64

	if err == nil {
		modTime = info.ModTime()
		size = info.Size()
	}

	// Convert to our event type
	var eventType EventType
	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		eventType = EventTypeCreated
	case event.Op&fsnotify.Write == fsnotify.Write:
		eventType = EventTypeModified
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		eventType = EventTypeDeleted
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		eventType = EventTypeRenamed
	default:
		eventType = EventTypeModified
	}

	changeEvent := ChangeEvent{
		Type:    eventType,
		Path:    event.Name,
		ModTime: modTime,
		Size:    size,
	}

	// Send to debouncer
	select {
	case fw.debouncer.events <- changeEvent:
	default:
		// Channel full, skip this event
	}
}

func (fw *FileWatcher) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case events := <-fw.debouncer.output:
			fw.mutex.RLock()
			handlers := fw.handlers
			fw.mutex.RUnlock()

			for _, handler := range handlers {
				if err := handler(events); err != nil {
					// Log error but continue processing
					log.Printf("File watcher handler error: %v", err)
				}
			}
		}
	}
}

// Debouncer implementation
func (d *Debouncer) start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-d.events:
			d.addEvent(event)
		}
	}
}

func (d *Debouncer) addEvent(event ChangeEvent) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// Prevent unbounded growth - drop old events if we exceed limit
	if len(d.pending) >= MaxPendingEvents {
		// Remove oldest events (FIFO)
		copy(d.pending, d.pending[1:])
		d.pending = d.pending[:len(d.pending)-1]
	}

	// Add event to pending list
	d.pending = append(d.pending, event)

	// Reset timer
	if d.timer != nil {
		d.timer.Stop()
	}

	d.timer = time.AfterFunc(d.delay, func() {
		d.flush()
	})

	// Periodic cleanup to prevent memory growth
	if time.Since(d.lastCleanup) > CleanupInterval {
		d.cleanup()
		d.lastCleanup = time.Now()
	}
}

func (d *Debouncer) flush() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if len(d.pending) == 0 {
		return
	}

	// Get objects from pools to reduce allocations
	eventMap := eventMapPool.Get().(map[string]ChangeEvent)
	events := eventPool.Get().([]ChangeEvent)

	// Clear the map and slice for reuse
	for k := range eventMap {
		delete(eventMap, k)
	}
	events = events[:0]

	// Deduplicate events by path
	for _, event := range d.pending {
		eventMap[event.Path] = event
	}

	// Convert back to slice
	for _, event := range eventMap {
		events = append(events, event)
	}

	// Make a copy for sending since we'll reuse the slice
	eventsCopy := make([]ChangeEvent, len(events))
	copy(eventsCopy, events)

	// Return objects to pools for reuse
	eventMapPool.Put(eventMap)
	eventPool.Put(events)

	// Send debounced events (non-blocking)
	select {
	case d.output <- eventsCopy:
	default:
		// Channel full, skip - this prevents goroutine leaks
	}

	// Clear pending events - reuse underlying array if capacity is reasonable
	if cap(d.pending) <= MaxPendingEvents*2 {
		d.pending = d.pending[:0]
	} else {
		// Reallocate if capacity grew too large
		d.pending = make([]ChangeEvent, 0, 100)
	}
}

// cleanup performs periodic memory cleanup
func (d *Debouncer) cleanup() {
	// This function is called while holding the mutex in addEvent

	// If pending slice has grown too large, reallocate with smaller capacity
	if cap(d.pending) > MaxPendingEvents*2 {
		newPending := make([]ChangeEvent, len(d.pending), MaxPendingEvents)
		copy(newPending, d.pending)
		d.pending = newPending
	}

	// Force garbage collection of any unreferenced timer objects
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
}

// Common file filters
func TemplFilter(path string) bool {
	return filepath.Ext(path) == ".templ"
}

func GoFilter(path string) bool {
	return filepath.Ext(path) == ".go"
}

func NoTestFilter(path string) bool {
	base := filepath.Base(path)
	matched1, _ := filepath.Match("*_test.go", base)
	matched2, _ := filepath.Match("*_test.templ", base)
	return !matched1 && !matched2
}

func NoVendorFilter(path string) bool {
	return !filepath.HasPrefix(path, "vendor/") && !strings.Contains(path, "/vendor/")
}

func NoGitFilter(path string) bool {
	return !filepath.HasPrefix(path, ".git/") && !strings.Contains(path, "/.git/")
}
