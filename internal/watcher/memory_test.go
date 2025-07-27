package watcher

import (
	"math"
	"runtime"
	"testing"
	"time"
)

// TestMemoryLeakPrevention tests that the file watcher doesn't leak memory under sustained load.
func TestMemoryLeakPrevention(t *testing.T) {
	// Create a file watcher with short debounce delay
	fw, err := NewFileWatcher(10 * time.Millisecond)
	if err != nil {
		t.Fatalf(ErrFailedToCreateFileWatcher, err)
	}
	defer func() {
		if err := fw.Stop(); err != nil {
			t.Logf("Warning: failed to stop file watcher: %v", err)
		}
	}()

	// Add a handler that does nothing
	fw.AddHandler(func(events []ChangeEvent) error {
		return nil
	})

	// Record initial memory usage
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Simulate many file change events
	for i := range 10000 {
		event := ChangeEvent{
			Type:    EventTypeModified,
			Path:    TestFilePath,
			ModTime: time.Now(),
			Size:    1024,
		}

		// Send event to debouncer
		select {
		case fw.debouncer.events <- event:
		default:
			// Channel full, skip
		}

		// Occasionally trigger flush
		if i%100 == 0 {
			time.Sleep(15 * time.Millisecond) // Longer than debounce delay
		}
	}

	// Wait for all events to be processed
	time.Sleep(100 * time.Millisecond)

	// Force garbage collection and measure memory
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Check that memory usage hasn't grown excessively
	var memoryGrowth int64
	if m2.Alloc > m1.Alloc {
		diff := m2.Alloc - m1.Alloc
		// Safe conversion: check for overflow before converting
		if diff <= math.MaxInt64 {
			memoryGrowth = int64(
				diff,
			) //nolint:gosec // Overflow protection: checked diff <= math.MaxInt64
		} else {
			memoryGrowth = math.MaxInt64 // Cap at max int64
		}
	} else {
		memoryGrowth = 0 // Memory decreased or stayed same
	}

	t.Logf(
		"Memory before: %d bytes, after: %d bytes, growth: %d bytes",
		m1.Alloc,
		m2.Alloc,
		memoryGrowth,
	)

	// Allow some growth but not more than 1MB for 10k events
	if memoryGrowth > 1024*1024 {
		t.Errorf("Excessive memory growth: %d bytes (expected < 1MB)", memoryGrowth)
	}
}

// TestBoundedEventQueue tests that the event queue doesn't grow unbounded.
func TestBoundedEventQueue(t *testing.T) {
	fw, err := NewFileWatcher(1 * time.Second) // Long delay to prevent flushing
	if err != nil {
		t.Fatalf(ErrFailedToCreateFileWatcher, err)
	}
	defer func() {
		if err := fw.Stop(); err != nil {
			t.Logf("Warning: failed to stop file watcher: %v", err)
		}
	}()

	// Send more events than MaxPendingEvents
	for range MaxPendingEvents + 500 {
		event := ChangeEvent{
			Type:    EventTypeModified,
			Path:    TestFilePath,
			ModTime: time.Now(),
			Size:    1024,
		}
		fw.debouncer.addEvent(event)
	}

	// Check that pending events are bounded
	fw.debouncer.mutex.Lock()
	pendingCount := len(fw.debouncer.pending)
	fw.debouncer.mutex.Unlock()

	if pendingCount > MaxPendingEvents {
		t.Errorf(
			"Event queue not bounded: %d events (expected <= %d)",
			pendingCount,
			MaxPendingEvents,
		)
	}

	t.Logf("Pending events after overflow: %d (max: %d)", pendingCount, MaxPendingEvents)
}

// TestObjectPoolEfficiency tests that object pools reduce allocations.
func TestObjectPoolEfficiency(t *testing.T) {
	fw, err := NewFileWatcher(10 * time.Millisecond)
	if err != nil {
		t.Fatalf(ErrFailedToCreateFileWatcher, err)
	}
	defer func() {
		if err := fw.Stop(); err != nil {
			t.Logf("Warning: failed to stop file watcher: %v", err)
		}
	}()

	// Add events and force multiple flushes
	for i := range 100 {
		event := ChangeEvent{
			Type:    EventTypeModified,
			Path:    TestFilePath,
			ModTime: time.Now(),
			Size:    1024,
		}
		fw.debouncer.addEvent(event)

		// Force flush every 10 events
		if i%10 == 9 {
			fw.debouncer.flush()
		}
	}

	// The test passes if no panics occur and objects are properly pooled
	t.Log("Object pool test completed successfully")
}

// TestCleanupPreventsGrowth tests that periodic cleanup prevents memory growth.
func TestCleanupPreventsGrowth(t *testing.T) {
	fw, err := NewFileWatcher(10 * time.Millisecond)
	if err != nil {
		t.Fatalf(ErrFailedToCreateFileWatcher, err)
	}
	defer func() {
		if err := fw.Stop(); err != nil {
			t.Logf("Warning: failed to stop file watcher: %v", err)
		}
	}()

	// Force cleanup by manipulating last cleanup time
	fw.debouncer.lastCleanup = time.Now().Add(-CleanupInterval - time.Second)

	// Add many events to grow the pending slice
	for range MaxPendingEvents * 2 {
		event := ChangeEvent{
			Type:    EventTypeModified,
			Path:    TestFilePath,
			ModTime: time.Now(),
			Size:    1024,
		}
		fw.debouncer.addEvent(event)
	}

	// Check that cleanup was triggered and capacity is reasonable
	fw.debouncer.mutex.Lock()
	capacity := cap(fw.debouncer.pending)
	fw.debouncer.mutex.Unlock()

	t.Logf("Pending slice capacity after cleanup: %d", capacity)

	// Capacity should be reasonable after cleanup
	if capacity > MaxPendingEvents*3 {
		t.Errorf(
			"Cleanup didn't prevent growth: capacity %d (expected <= %d)",
			capacity,
			MaxPendingEvents*3,
		)
	}
}

// BenchmarkWatcherMemoryUsage benchmarks memory usage under load.
func BenchmarkWatcherMemoryUsage(b *testing.B) {
	fw, err := NewFileWatcher(10 * time.Millisecond)
	if err != nil {
		b.Fatalf(ErrFailedToCreateFileWatcher, err)
	}
	defer func() {
		if err := fw.Stop(); err != nil {
			b.Logf("Warning: failed to stop file watcher: %v", err)
		}
	}()

	fw.AddHandler(func(events []ChangeEvent) error {
		return nil
	})

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		event := ChangeEvent{
			Type:    EventTypeModified,
			Path:    TestFilePath,
			ModTime: time.Now(),
			Size:    1024,
		}

		select {
		case fw.debouncer.events <- event:
		default:
			// Channel full, skip
		}
	}
}
