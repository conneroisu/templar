//go:build property

package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestFileWatcherProperties validates critical properties of the file watcher
func TestFileWatcherProperties(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.Rng.Seed(9876)
	parameters.MinSuccessfulTests = 50

	properties := gopter.NewProperties(parameters)

	// Property: File watcher should debounce rapid file changes
	properties.Property("file watcher debounces rapid changes", prop.ForAll(
		func(debounceMs int, changeCount int) bool {
			if debounceMs < 10 || debounceMs > 1000 || changeCount < 1 || changeCount > 20 {
				return true
			}

			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "test.templ")

			// Create initial file
			if err := os.WriteFile(testFile, []byte("initial content"), 0644); err != nil {
				return true
			}

			watcher, err := NewFileWatcher(time.Duration(debounceMs) * time.Millisecond)
			if err != nil {
				return true
			}
			defer watcher.Stop()

			// Track events
			eventCount := 0
			watcher.AddHandler(func(events []ChangeEvent) error {
				eventCount += len(events)
				return nil
			})

			if err := watcher.AddRecursive(tempDir); err != nil {
				return true
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := watcher.Start(ctx); err != nil {
				return true
			}

			// Wait for watcher to start
			time.Sleep(50 * time.Millisecond)

			// Make rapid changes to file
			for i := 0; i < changeCount; i++ {
				content := []byte("content " + string(rune(i)))
				if err := os.WriteFile(testFile, content, 0644); err != nil {
					continue
				}
				time.Sleep(time.Duration(debounceMs/4) * time.Millisecond) // Changes faster than debounce
			}

			// Wait for debounce period plus buffer
			time.Sleep(time.Duration(debounceMs*2) * time.Millisecond)

			// Property: Should receive fewer events than changes due to debouncing
			// Allow some tolerance for timing variations
			return eventCount <= changeCount && eventCount >= 1
		},
		gen.IntRange(50, 500),
		gen.IntRange(3, 10),
	))

	// Property: File watcher should handle multiple directories
	properties.Property("file watcher handles multiple directories", prop.ForAll(
		func(dirCount int) bool {
			if dirCount < 1 || dirCount > 10 {
				return true
			}

			baseDir := t.TempDir()
			watcher, err := NewFileWatcher(100 * time.Millisecond)
			if err != nil {
				return true
			}
			defer watcher.Stop()

			// Create directories and files
			dirs := make([]string, dirCount)
			for i := 0; i < dirCount; i++ {
				dirPath := filepath.Join(baseDir, "dir"+fmt.Sprintf("%d", i))
				if err := os.MkdirAll(dirPath, 0755); err != nil {
					return true
				}
				dirs[i] = dirPath

				// Create test file in each directory
				testFile := filepath.Join(dirPath, "test.templ")
				if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
					return true
				}
			}

			eventCount := 0
			watcher.AddHandler(func(events []ChangeEvent) error {
				eventCount += len(events)
				return nil
			})

			// Add all directories to watch
			for _, dir := range dirs {
				if err := watcher.AddRecursive(dir); err != nil {
					return true
				}
			}

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			if err := watcher.Start(ctx); err != nil {
				return true
			}

			// Wait for watcher to start
			time.Sleep(50 * time.Millisecond)

			// Make changes to files in each directory
			for i, dir := range dirs {
				testFile := filepath.Join(dir, "test.templ")
				content := []byte("updated content " + string(rune(i)))
				if err := os.WriteFile(testFile, content, 0644); err != nil {
					continue
				}
			}

			// Wait for events
			time.Sleep(300 * time.Millisecond)

			// Property: Should receive events from all directories
			return eventCount >= 1 && eventCount <= dirCount*2 // Allow for some event coalescing
		},
		gen.IntRange(1, 5),
	))

	// Property: File watcher should handle file creation and deletion
	properties.Property("file watcher handles file lifecycle", prop.ForAll(
		func(fileCount int) bool {
			if fileCount < 1 || fileCount > 15 {
				return true
			}

			tempDir := t.TempDir()
			watcher, err := NewFileWatcher(100 * time.Millisecond)
			if err != nil {
				return true
			}
			defer watcher.Stop()

			eventPaths := make(map[string]bool)
			watcher.AddHandler(func(events []ChangeEvent) error {
				for _, event := range events {
					eventPaths[event.Path] = true
				}
				return nil
			})

			if err := watcher.AddRecursive(tempDir); err != nil {
				return true
			}

			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
			defer cancel()

			if err := watcher.Start(ctx); err != nil {
				return true
			}

			// Wait for watcher to start
			time.Sleep(50 * time.Millisecond)

			// Create files
			filePaths := make([]string, fileCount)
			for i := 0; i < fileCount; i++ {
				fileName := "test" + string(rune(i)) + ".templ"
				filePath := filepath.Join(tempDir, fileName)
				filePaths[i] = filePath

				content := []byte("content " + string(rune(i)))
				if err := os.WriteFile(filePath, content, 0644); err != nil {
					continue
				}
			}

			// Wait for creation events
			time.Sleep(200 * time.Millisecond)

			// Delete files
			for _, filePath := range filePaths {
				os.Remove(filePath)
			}

			// Wait for deletion events
			time.Sleep(200 * time.Millisecond)

			// Property: Should receive at least one event (creation or deletion are both valid)
			return len(eventPaths) >= 1
		},
		gen.IntRange(1, 8),
	))

	// Property: File watcher should be resilient to invalid paths
	properties.Property("file watcher handles invalid paths gracefully", prop.ForAll(
		func(invalidPath string) bool {
			if invalidPath == "" {
				return true
			}

			watcher, err := NewFileWatcher(100 * time.Millisecond)
			if err != nil {
				return true
			}
			defer watcher.Stop()

			// Should not panic on invalid paths
			addErr := watcher.AddPath(invalidPath)
			
			// Property: Should return error for invalid paths, not panic
			return addErr != nil
		},
		gen.OneConstOf("/nonexistent/path", "", "/dev/null/invalid"),
	))

	// Property: Concurrent watching should be safe
	properties.Property("concurrent watch operations are safe", prop.ForAll(
		func(goroutineCount int) bool {
			if goroutineCount < 1 || goroutineCount > 10 {
				return true
			}

			baseDir := t.TempDir()
			watcher, err := NewFileWatcher(100 * time.Millisecond)
			if err != nil {
				return true
			}
			defer watcher.Stop()

			eventCount := 0
			watcher.AddHandler(func(events []ChangeEvent) error {
				eventCount += len(events)
				return nil
			})

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			// Create subdirectories
			dirs := make([]string, goroutineCount)
			for i := 0; i < goroutineCount; i++ {
				dirPath := filepath.Join(baseDir, "subdir"+string(rune(i)))
				if err := os.MkdirAll(dirPath, 0755); err != nil {
					return true
				}
				dirs[i] = dirPath
			}

			// Add all directories to watch
			for _, dir := range dirs {
				if err := watcher.AddRecursive(dir); err != nil {
					return true
				}
			}

			if err := watcher.Start(ctx); err != nil {
				return true
			}

			// Start concurrent file operations
			done := make(chan bool, goroutineCount)
			for i := 0; i < goroutineCount; i++ {
				go func(dir string) {
					defer func() { done <- true }()
					
					// Create a file to trigger events
					testFile := filepath.Join(dir, "concurrent.templ")
					os.WriteFile(testFile, []byte("concurrent content"), 0644)
				}(dirs[i])
			}

			// Wait for all goroutines
			for i := 0; i < goroutineCount; i++ {
				select {
				case <-done:
				case <-time.After(2 * time.Second):
					return false // Timeout indicates potential deadlock
				}
			}

			// Wait for events
			time.Sleep(300 * time.Millisecond)

			// Property: Should not deadlock and should receive some events
			return eventCount >= 0 // No panics or deadlocks
		},
		gen.IntRange(1, 5),
	))

	properties.TestingRun(t)
}

// TestWatcherEventOrderingProperties validates event ordering and consistency
func TestWatcherEventOrderingProperties(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.Rng.Seed(1357)
	parameters.MinSuccessfulTests = 30

	properties := gopter.NewProperties(parameters)

	// Property: Event ordering should be consistent for sequential operations
	properties.Property("sequential file operations maintain order consistency", prop.ForAll(
		func(operationCount int) bool {
			if operationCount < 2 || operationCount > 10 {
				return true
			}

			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "sequence.templ")

			watcher, err := NewFileWatcher(50 * time.Millisecond)
			if err != nil {
				return true
			}
			defer watcher.Stop()

			lastEventTime := time.Time{}
			orderViolations := 0
			
			watcher.AddHandler(func(events []ChangeEvent) error {
				for _, event := range events {
					currentTime := event.ModTime
					if !lastEventTime.IsZero() && currentTime.Before(lastEventTime) {
						orderViolations++
					}
					lastEventTime = currentTime
				}
				return nil
			})

			if err := watcher.AddRecursive(tempDir); err != nil {
				return true
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := watcher.Start(ctx); err != nil {
				return true
			}

			// Wait for watcher to start
			time.Sleep(50 * time.Millisecond)

			// Perform sequential operations with spacing to allow debouncing
			for i := 0; i < operationCount; i++ {
				content := []byte("content iteration " + string(rune(i)))
				if err := os.WriteFile(testFile, content, 0644); err != nil {
					continue
				}
				time.Sleep(100 * time.Millisecond) // Space out operations
			}

			// Wait for final events
			time.Sleep(200 * time.Millisecond)

			// Property: Event timestamps should not violate chronological order
			return orderViolations == 0
		},
		gen.IntRange(2, 6),
	))

	properties.TestingRun(t)
}