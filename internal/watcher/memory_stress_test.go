package watcher

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/interfaces"
)

// TestMemoryGrowthUnderHighLoad tests memory behavior under sustained high load
func TestMemoryGrowthUnderHighLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory stress test in short mode")
	}

	// Create test directory in current working directory
	tempDir := filepath.Join(".", "test_watcher_memory_"+string(rune(time.Now().UnixNano()%10000)))
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create watcher with very short debounce to maximize event processing
	fw, err := NewFileWatcher(1 * time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to create file watcher: %v", err)
	}
	defer fw.Stop()

	// Add handler that processes events
	eventCount := 0
	fw.AddHandler(func(events []ChangeEvent) error {
		eventCount += len(events)
		return nil
	})

	fw.AddFilter(interfaces.FileFilterFunc(TemplFilter))

	if err := fw.AddRecursive(tempDir); err != nil {
		t.Fatalf("Failed to add path: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := fw.Start(ctx); err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}

	// Measure memory before stress test
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Create sustained file activity to stress memory management
	for cycle := 0; cycle < 5; cycle++ {
		t.Logf("Memory stress cycle %d/5", cycle+1)

		// Create many files rapidly
		for i := 0; i < 200; i++ {
			fileName := filepath.Join(
				tempDir,
				"stress_test_"+string(rune(cycle))+"_"+string(rune(i))+".templ",
			)
			content := make([]byte, 1024) // 1KB per file
			for j := range content {
				content[j] = byte(i % 256)
			}

			if err := os.WriteFile(fileName, content, 0644); err != nil {
				continue
			}

			// Small delay to allow event processing
			if i%10 == 0 {
				time.Sleep(1 * time.Millisecond)
			}
		}

		// Wait for event processing
		time.Sleep(50 * time.Millisecond)

		// Delete files to create more events
		for i := 0; i < 200; i++ {
			fileName := filepath.Join(
				tempDir,
				"stress_test_"+string(rune(cycle))+"_"+string(rune(i))+".templ",
			)
			os.Remove(fileName)
		}

		// Wait for deletion events
		time.Sleep(50 * time.Millisecond)

		// Force GC and measure memory growth
		runtime.GC()
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		memGrowth := int64(0)
		if m.Alloc > m1.Alloc {
			memGrowth = int64(m.Alloc - m1.Alloc)
		}

		t.Logf(
			"Cycle %d: Memory growth: %d bytes, Events processed: %d",
			cycle+1,
			memGrowth,
			eventCount,
		)

		// Check for excessive growth (should be sub-linear)
		expectedMaxGrowth := int64((cycle + 1) * 100 * 1024) // 100KB per cycle max
		if memGrowth > expectedMaxGrowth {
			t.Logf("Warning: Memory growth %d bytes exceeds expected %d bytes for cycle %d",
				memGrowth, expectedMaxGrowth, cycle+1)
		}
	}

	// Final memory measurement
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	finalGrowth := int64(0)
	if m2.Alloc > m1.Alloc {
		finalGrowth = int64(m2.Alloc - m1.Alloc)
	}

	t.Logf("Final results:")
	t.Logf("- Memory before: %d bytes", m1.Alloc)
	t.Logf("- Memory after: %d bytes", m2.Alloc)
	t.Logf("- Total growth: %d bytes", finalGrowth)
	t.Logf("- Total events processed: %d", eventCount)

	// Memory growth should be reasonable (under 2MB for this workload)
	maxAllowedGrowth := int64(2 * 1024 * 1024)
	if finalGrowth > maxAllowedGrowth {
		t.Errorf(
			"Excessive memory growth: %d bytes (max allowed: %d bytes)",
			finalGrowth,
			maxAllowedGrowth,
		)
	}

	// Should have processed a reasonable number of events
	if eventCount < 100 {
		t.Errorf("Too few events processed: %d (expected at least 100)", eventCount)
	}
}

// TestChannelBufferOverflow tests behavior when event channels become full
func TestChannelBufferOverflow(t *testing.T) {
	// Create watcher with very long debounce to prevent flushing
	fw, err := NewFileWatcher(10 * time.Second)
	if err != nil {
		t.Fatalf("Failed to create file watcher: %v", err)
	}
	defer fw.Stop()

	// Add handler (won't be called due to long debounce)
	fw.AddHandler(func(events []ChangeEvent) error {
		return nil
	})

	// Overflow the event channel
	eventsToSend := 150 // More than channel capacity (100)
	sentEvents := 0

	for i := 0; i < eventsToSend; i++ {
		event := ChangeEvent{
			Type:    EventTypeModified,
			Path:    "/test/file.templ",
			ModTime: time.Now(),
			Size:    1024,
		}

		select {
		case fw.debouncer.events <- event:
			sentEvents++
		default:
			// Channel full - this should happen
			break
		}
	}

	t.Logf("Sent %d events out of %d attempted", sentEvents, eventsToSend)

	// Should not be able to send all events due to channel buffer limit
	if sentEvents >= eventsToSend {
		t.Errorf("Channel buffer overflow protection not working: sent all %d events", sentEvents)
	}

	// Should have sent at least the buffer capacity
	if sentEvents < 100 {
		t.Errorf("Channel buffer too small: only sent %d events", sentEvents)
	}
}

// BenchmarkMemoryEfficiency benchmarks memory efficiency improvements
func BenchmarkMemoryEfficiency(b *testing.B) {
	fw, err := NewFileWatcher(10 * time.Millisecond)
	if err != nil {
		b.Fatalf("Failed to create file watcher: %v", err)
	}
	defer fw.Stop()

	fw.AddHandler(func(events []ChangeEvent) error {
		return nil
	})

	b.ResetTimer()
	b.ReportAllocs()

	// Benchmark event processing with pooled objects
	for i := 0; i < b.N; i++ {
		event := ChangeEvent{
			Type:    EventTypeModified,
			Path:    "/test/file" + string(rune(i%100)) + ".templ",
			ModTime: time.Now(),
			Size:    1024,
		}

		// Simulate the full event flow through debouncer
		fw.debouncer.addEvent(event)

		// Occasionally flush to test pooling
		if i%50 == 0 {
			fw.debouncer.flush()
		}
	}
}
