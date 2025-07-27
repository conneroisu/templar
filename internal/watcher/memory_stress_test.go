package watcher

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/interfaces"
)

// TestMemoryGrowthUnderHighLoad tests memory behavior under sustained high load.
func TestMemoryGrowthUnderHighLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory stress test in short mode")
	}

	// Create test directory in current working directory
	tempDir := filepath.Join(".", "test_watcher_memory_"+string(rune(time.Now().UnixNano()%10000)))
	err := os.MkdirAll(tempDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create watcher with very short debounce to maximize event processing
	fw, err := NewFileWatcher(1 * time.Millisecond)
	if err != nil {
		t.Fatalf(ErrFailedToCreateFileWatcher, err)
	}
	defer func() {
		if err := fw.Stop(); err != nil {
			t.Logf("Failed to stop file watcher: %v", err)
		}
	}()

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
	for cycle := range 5 {
		t.Logf("Memory stress cycle %d/5", cycle+1)

		// Create many files rapidly
		for i := range 200 {
			fileName := filepath.Join(
				tempDir,
				"stress_test_"+string(rune(cycle))+"_"+string(rune(i))+".templ",
			)
			content := make([]byte, 1024) // 1KB per file
			for j := range content {
				content[j] = byte(i % 256)
			}

			if err := os.WriteFile(fileName, content, 0o600); err != nil {
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
		for i := range 200 {
			fileName := filepath.Join(
				tempDir,
				"stress_test_"+string(rune(cycle))+"_"+string(rune(i))+".templ",
			)
			_ = os.Remove(fileName)
		}

		// Wait for deletion events
		time.Sleep(50 * time.Millisecond)

		// Force GC and measure memory growth
		runtime.GC()
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		memGrowth := int64(0)
		if m.Alloc > m1.Alloc {
			diff := m.Alloc - m1.Alloc
			// Safe conversion: check for overflow before converting
			if diff <= math.MaxInt64 {
				memGrowth = int64(
					diff,
				) //nolint:gosec // Overflow protection: checked diff <= math.MaxInt64
			} else {
				memGrowth = math.MaxInt64 // Cap at max int64
			}
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
		diff := m2.Alloc - m1.Alloc
		// Safe conversion: check for overflow before converting
		if diff <= math.MaxInt64 {
			finalGrowth = int64(
				diff,
			) //nolint:gosec // Overflow protection: checked diff <= math.MaxInt64
		} else {
			finalGrowth = math.MaxInt64 // Cap at max int64
		}
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

// TestChannelBufferOverflow tests behavior when event channels become full.
func TestChannelBufferOverflow(t *testing.T) {
	// Create watcher with very long debounce to prevent flushing
	fw, err := NewFileWatcher(10 * time.Second)
	if err != nil {
		t.Fatalf(ErrFailedToCreateFileWatcher, err)
	}
	defer func() {
		if err := fw.Stop(); err != nil {
			t.Logf("Failed to stop file watcher: %v", err)
		}
	}()

	// Add handler (won't be called due to long debounce)
	fw.AddHandler(func(events []ChangeEvent) error {
		return nil
	})

	// Overflow the event channel
	eventsToSend := 150 // More than channel capacity (100)
	sentEvents := 0

eventLoop:
	for range eventsToSend {
		event := ChangeEvent{
			Type:    EventTypeModified,
			Path:    TestFilePath,
			ModTime: time.Now(),
			Size:    1024,
		}

		select {
		case fw.debouncer.events <- event:
			sentEvents++
		default:
			// Channel full - this should happen
			break eventLoop
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

// BenchmarkMemoryEfficiency benchmarks memory efficiency improvements.
func BenchmarkMemoryEfficiency(b *testing.B) {
	fw, err := NewFileWatcher(10 * time.Millisecond)
	if err != nil {
		b.Fatalf(ErrFailedToCreateFileWatcher, err)
	}
	defer func() {
		if err := fw.Stop(); err != nil {
			b.Logf("Failed to stop file watcher: %v", err)
		}
	}()

	fw.AddHandler(func(events []ChangeEvent) error {
		return nil
	})

	b.ResetTimer()
	b.ReportAllocs()

	// Benchmark event processing with pooled objects
	for i := range b.N {
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
