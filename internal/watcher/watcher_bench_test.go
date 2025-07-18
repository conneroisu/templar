package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// createTestDirStructure creates a directory structure with the specified number of files
func createTestDirStructure(fileCount int) string {
	tempDir := fmt.Sprintf("watcher_bench_%d_%d", fileCount, time.Now().UnixNano())
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		panic(err)
	}

	// Create subdirectories
	for i := 0; i < fileCount/10; i++ {
		subDir := filepath.Join(tempDir, fmt.Sprintf("subdir_%d", i))
		if err := os.MkdirAll(subDir, 0755); err != nil {
			panic(err)
		}
	}

	// Create files distributed across subdirectories
	for i := 0; i < fileCount; i++ {
		subDirIndex := i / 10
		if subDirIndex >= fileCount/10 {
			subDirIndex = 0
		}
		
		var filePath string
		if subDirIndex == 0 {
			filePath = filepath.Join(tempDir, fmt.Sprintf("file_%d.go", i))
		} else {
			filePath = filepath.Join(tempDir, fmt.Sprintf("subdir_%d", subDirIndex), fmt.Sprintf("file_%d.go", i))
		}
		
		content := fmt.Sprintf("package main\n\n// File %d content\nfunc main() {\n\tprintln(\"hello %d\")\n}\n", i, i)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			panic(err)
		}
	}

	return tempDir
}

// BenchmarkFileWatcher_AddRecursive benchmarks directory scanning performance
func BenchmarkFileWatcher_AddRecursive(b *testing.B) {
	sizes := []int{100, 500, 1000, 2000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("files-%d", size), func(b *testing.B) {
			// Create test directory structure
			testDir := createTestDirStructure(size)
			defer os.RemoveAll(testDir)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				watcher, err := NewFileWatcher(100 * time.Millisecond)
				if err != nil {
					b.Fatal(err)
				}

				err = watcher.AddRecursive(testDir)
				if err != nil {
					b.Fatal(err)
				}

				watcher.Stop()
			}
		})
	}
}

// BenchmarkFileWatcher_AddPath benchmarks single path addition
func BenchmarkFileWatcher_AddPath(b *testing.B) {
	watcher, err := NewFileWatcher(100 * time.Millisecond)
	if err != nil {
		b.Fatal(err)
	}
	defer watcher.Stop()

	// Create test directory
	testDir := createTestDirStructure(100)
	defer os.RemoveAll(testDir)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Add and immediately remove to reset state
		err := watcher.AddPath(testDir)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFileWatcher_EventProcessing benchmarks event processing under load
func BenchmarkFileWatcher_EventProcessing(b *testing.B) {
	testDir := createTestDirStructure(1000)
	defer os.RemoveAll(testDir)

	watcher, err := NewFileWatcher(50 * time.Millisecond)
	if err != nil {
		b.Fatal(err)
	}
	defer watcher.Stop()

	// Setup watcher
	err = watcher.AddRecursive(testDir)
	if err != nil {
		b.Fatal(err)
	}

	// Add a simple handler to process events
	eventCount := 0
	watcher.AddHandler(func(events []ChangeEvent) error {
		eventCount += len(events)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = watcher.Start(ctx)
	if err != nil {
		b.Fatal(err)
	}

	// Let the watcher start up
	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Create file changes
		filename := filepath.Join(testDir, fmt.Sprintf("bench_file_%d.txt", i%100))
		content := fmt.Sprintf("benchmark content %d", i)
		err := os.WriteFile(filename, []byte(content), 0644)
		if err != nil {
			b.Fatal(err)
		}
	}

	// Wait for events to be processed
	time.Sleep(200 * time.Millisecond)
}

// BenchmarkFileWatcher_FilterPerformance benchmarks filter application performance
func BenchmarkFileWatcher_FilterPerformance(b *testing.B) {
	filterTypes := []struct {
		name   string
		filter FileFilter
	}{
		{"TemplFilter", TemplFilter},
		{"GoFilter", GoFilter},
		{"NoTestFilter", NoTestFilter},
		{"NoVendorFilter", NoVendorFilter},
		{"NoGitFilter", NoGitFilter},
	}

	testPaths := []string{
		"main.go",
		"component.templ", 
		"main_test.go",
		"vendor/package/index.js",
		".git/config",
		"src/components/Button.templ",
		"internal/server/handler.go",
		"node_modules/react/index.js",
	}

	for _, ft := range filterTypes {
		b.Run(ft.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				for _, path := range testPaths {
					ft.filter(path)
				}
			}
		})
	}
}

// BenchmarkDebouncer_Performance benchmarks debouncing performance
func BenchmarkDebouncer_Performance(b *testing.B) {
	debouncer := &Debouncer{
		delay:   50 * time.Millisecond,
		events:  make(chan ChangeEvent, 1000),
		output:  make(chan []ChangeEvent, 100),
		pending: make([]ChangeEvent, 0),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start debouncer
	go debouncer.start(ctx)

	// Consumer to prevent channel blocking
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-debouncer.output:
				// Consume events
			}
		}
	}()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		event := ChangeEvent{
			Type:    EventTypeModified,
			Path:    fmt.Sprintf("file_%d.go", i%100),
			ModTime: time.Now(),
			Size:    1024,
		}

		select {
		case debouncer.events <- event:
		default:
			// Skip if channel is full
		}
	}

	// Wait for debouncing to complete
	time.Sleep(100 * time.Millisecond)
}

// BenchmarkFileWatcher_MemoryUsage benchmarks memory usage patterns
func BenchmarkFileWatcher_MemoryUsage(b *testing.B) {
	b.Run("SmallDirectory", func(b *testing.B) {
		benchmarkMemoryUsage(b, 100)
	})

	b.Run("MediumDirectory", func(b *testing.B) {
		benchmarkMemoryUsage(b, 1000)
	})

	b.Run("LargeDirectory", func(b *testing.B) {
		benchmarkMemoryUsage(b, 5000)
	})
}

func benchmarkMemoryUsage(b *testing.B, fileCount int) {
	testDir := createTestDirStructure(fileCount)
	defer os.RemoveAll(testDir)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		watcher, err := NewFileWatcher(100 * time.Millisecond)
		if err != nil {
			b.Fatal(err)
		}

		err = watcher.AddRecursive(testDir)
		if err != nil {
			b.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		err = watcher.Start(ctx)
		if err != nil {
			b.Fatal(err)
		}

		// Simulate some activity
		for j := 0; j < 10; j++ {
			filename := filepath.Join(testDir, fmt.Sprintf("temp_%d.txt", j))
			os.WriteFile(filename, []byte("content"), 0644)
		}

		cancel()
		watcher.Stop()
	}
}

// BenchmarkFileWatcher_ConcurrentOperations benchmarks concurrent file operations
func BenchmarkFileWatcher_ConcurrentOperations(b *testing.B) {
	testDir := createTestDirStructure(500)
	defer os.RemoveAll(testDir)

	watcher, err := NewFileWatcher(25 * time.Millisecond)
	if err != nil {
		b.Fatal(err)
	}
	defer watcher.Stop()

	err = watcher.AddRecursive(testDir)
	if err != nil {
		b.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = watcher.Start(ctx)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		fileIndex := 0
		for pb.Next() {
			filename := filepath.Join(testDir, fmt.Sprintf("concurrent_%d.txt", fileIndex))
			content := fmt.Sprintf("content %d", fileIndex)
			os.WriteFile(filename, []byte(content), 0644)
			fileIndex++
		}
	})

	// Allow time for event processing
	time.Sleep(100 * time.Millisecond)
}