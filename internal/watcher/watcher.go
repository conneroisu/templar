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

// FileWatcher watches for file changes with intelligent debouncing
type FileWatcher struct {
	watcher   *fsnotify.Watcher
	debouncer *Debouncer
	filters   []FileFilter
	handlers  []ChangeHandler
	mutex     sync.RWMutex
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
	delay   time.Duration
	events  chan ChangeEvent
	output  chan []ChangeEvent
	timer   *time.Timer
	pending []ChangeEvent
	mutex   sync.Mutex
}

// NewFileWatcher creates a new file watcher
func NewFileWatcher(debounceDelay time.Duration) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	debouncer := &Debouncer{
		delay:   debounceDelay,
		events:  make(chan ChangeEvent, 100),
		output:  make(chan []ChangeEvent, 10),
		pending: make([]ChangeEvent, 0),
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
	// Stop the debouncer timer if it exists
	if fw.debouncer.timer != nil {
		fw.debouncer.timer.Stop()
	}

	// Close the watcher
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

	// Add event to pending list
	d.pending = append(d.pending, event)

	// Reset timer
	if d.timer != nil {
		d.timer.Stop()
	}

	d.timer = time.AfterFunc(d.delay, func() {
		d.flush()
	})
}

func (d *Debouncer) flush() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if len(d.pending) == 0 {
		return
	}

	// Deduplicate events by path
	eventMap := make(map[string]ChangeEvent)
	for _, event := range d.pending {
		eventMap[event.Path] = event
	}

	// Convert back to slice
	events := make([]ChangeEvent, 0, len(eventMap))
	for _, event := range eventMap {
		events = append(events, event)
	}

	// Send debounced events
	select {
	case d.output <- events:
	default:
		// Channel full, skip
	}

	// Clear pending events
	d.pending = d.pending[:0]
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
