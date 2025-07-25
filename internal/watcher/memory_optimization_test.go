package watcher

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestEnhancedMemoryPooling tests that the new memory pools work correctly
func TestEnhancedMemoryPooling(t *testing.T) {
	fw, err := NewFileWatcher(10 * time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to create file watcher: %v", err)
	}
	defer fw.Stop()

	// Test that object pools are being used
	initialBatch := eventBatchPool.Get().([]ChangeEvent)
	eventBatchPool.Put(initialBatch)

	// Add many events to trigger multiple pool operations
	for i := 0; i < 200; i++ {
		event := ChangeEvent{
			Type:    EventTypeModified,
			Path:    "/test/file" + string(rune(i%20)) + ".templ",
			ModTime: time.Now(),
			Size:    1024,
		}
		fw.debouncer.addEvent(event)

		// Force flush periodically to test pool reuse
		if i%25 == 0 {
			fw.debouncer.flush()
		}
	}

	// Verify pools are still functional
	testBatch := eventBatchPool.Get().([]ChangeEvent)
	if testBatch == nil {
		t.Error("Event batch pool returned nil")
	}
	eventBatchPool.Put(testBatch)
}

// TestBatchProcessing tests that events are processed in batches efficiently
func TestBatchProcessing(t *testing.T) {
	fw, err := NewFileWatcher(100 * time.Millisecond) // Longer delay to control batching
	if err != nil {
		t.Fatalf("Failed to create file watcher: %v", err)
	}
	defer fw.Stop()

	batchesReceived := 0
	totalEventsReceived := 0
	var mu sync.Mutex

	fw.AddHandler(func(events []ChangeEvent) error {
		mu.Lock()
		defer mu.Unlock()
		batchesReceived++
		totalEventsReceived += len(events)
		t.Logf("Received batch %d with %d events", batchesReceived, len(events))
		return nil
	})

	// Start the debouncer processing loop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go fw.debouncer.start(ctx)
	go fw.processEvents(ctx)

	time.Sleep(10 * time.Millisecond) // Let goroutines start

	// Add enough events to trigger immediate batch processing
	maxBatchSize := fw.debouncer.maxBatchSize
	eventsToAdd := maxBatchSize + 10

	for i := 0; i < eventsToAdd; i++ {
		event := ChangeEvent{
			Type:    EventTypeModified,
			Path:    "/test/batch_file" + string(rune(i)) + ".templ",
			ModTime: time.Now(),
			Size:    1024,
		}
		fw.debouncer.addEvent(event)
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	finalBatches := batchesReceived
	finalEvents := totalEventsReceived
	mu.Unlock()

	t.Logf("Final stats: %d batches, %d events total", finalBatches, finalEvents)

	// Should have received at least one batch due to maxBatchSize triggering
	if finalBatches == 0 {
		t.Error("No batches received - batch processing not working")
	}

	// Should have received some events (may be deduplicated)
	if finalEvents == 0 {
		t.Error("No events received")
	}
}

// TestLRUEviction tests that old events are evicted properly
func TestLRUEviction(t *testing.T) {
	fw, err := NewFileWatcher(1 * time.Second) // Long delay to prevent flushing
	if err != nil {
		t.Fatalf("Failed to create file watcher: %v", err)
	}
	defer fw.Stop()

	// Fill up the pending queue beyond max capacity
	eventsToAdd := MaxPendingEvents + 200

	for i := 0; i < eventsToAdd; i++ {
		event := ChangeEvent{
			Type:    EventTypeModified,
			Path:    "/test/eviction_file" + string(rune(i)) + ".templ",
			ModTime: time.Now(),
			Size:    1024,
		}
		fw.debouncer.addEvent(event)
	}

	// Check that eviction occurred
	stats := fw.GetStats()
	pendingCount := stats["pending_events"].(int)
	droppedCount := stats["dropped_events"].(int64)

	t.Logf("Pending events: %d, Dropped events: %d", pendingCount, droppedCount)

	// Should not exceed max pending events
	if pendingCount > MaxPendingEvents {
		t.Errorf("LRU eviction failed: %d pending events (max: %d)", pendingCount, MaxPendingEvents)
	}

	// Should have dropped some events
	if droppedCount == 0 {
		t.Error("Expected some events to be dropped due to LRU eviction")
	}
}

// TestBackpressureHandling tests that backpressure is handled gracefully
func TestBackpressureHandling(t *testing.T) {
	fw, err := NewFileWatcher(1 * time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to create file watcher: %v", err)
	}
	defer fw.Stop()

	// Add a slow handler to create backpressure
	processedCount := 0
	fw.AddHandler(func(events []ChangeEvent) error {
		processedCount += len(events)
		time.Sleep(20 * time.Millisecond) // Moderate processing delay
		return nil
	})

	// Start the processing loops
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go fw.debouncer.start(ctx)
	go fw.processEvents(ctx)

	time.Sleep(10 * time.Millisecond) // Let goroutines start

	// Flood the system with events
	eventsToAdd := 500
	for i := 0; i < eventsToAdd; i++ {
		event := ChangeEvent{
			Type:    EventTypeModified,
			Path:    "/test/backpressure_file" + string(rune(i%10)) + ".templ",
			ModTime: time.Now(),
			Size:    1024,
		}
		fw.debouncer.addEvent(event)

		// Small delay to allow some processing
		if i%50 == 0 {
			time.Sleep(1 * time.Millisecond)
		}
	}

	// Wait a bit for processing
	time.Sleep(500 * time.Millisecond)

	stats := fw.GetStats()
	droppedCount := stats["dropped_events"].(int64)

	t.Logf("Processed: %d events, Dropped: %d events", processedCount, droppedCount)

	// Under backpressure, some events should be dropped
	if droppedCount == 0 {
		t.Log("Warning: Expected some events to be dropped under backpressure")
	}

	// System should still be responsive (processed some events)
	if processedCount == 0 {
		t.Error("No events processed - system completely blocked")
	}
}

// TestMemoryGrowthPrevention tests that memory doesn't grow unbounded
func TestMemoryGrowthPrevention(t *testing.T) {
	fw, err := NewFileWatcher(1 * time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to create file watcher: %v", err)
	}
	defer fw.Stop()

	// Add handler
	fw.AddHandler(func(events []ChangeEvent) error {
		return nil
	})

	// Monitor stats over multiple cycles
	for cycle := 0; cycle < 10; cycle++ {
		// Add many events
		for i := 0; i < 100; i++ {
			event := ChangeEvent{
				Type:    EventTypeModified,
				Path:    "/test/growth_file" + string(rune(i%5)) + ".templ",
				ModTime: time.Now(),
				Size:    1024,
			}
			fw.debouncer.addEvent(event)
		}

		// Let some processing happen
		time.Sleep(10 * time.Millisecond)

		stats := fw.GetStats()
		pendingCount := stats["pending_events"].(int)
		pendingCapacity := stats["pending_capacity"].(int)

		t.Logf("Cycle %d: pending=%d, capacity=%d", cycle+1, pendingCount, pendingCapacity)

		// Capacity should not grow unbounded
		if pendingCapacity > MaxPendingEvents*3 {
			t.Errorf("Pending capacity grew too large: %d (cycle %d)", pendingCapacity, cycle+1)
		}
	}
}

// TestStatsAccuracy tests that statistics are accurate
func TestStatsAccuracy(t *testing.T) {
	fw, err := NewFileWatcher(10 * time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to create file watcher: %v", err)
	}
	defer fw.Stop()

	// Initial stats should be clean
	stats := fw.GetStats()
	if stats["pending_events"].(int) != 0 {
		t.Error("Expected 0 pending events initially")
	}
	if stats["dropped_events"].(int64) != 0 {
		t.Error("Expected 0 dropped events initially")
	}

	// Add some events
	eventsAdded := 10
	for i := 0; i < eventsAdded; i++ {
		event := ChangeEvent{
			Type:    EventTypeModified,
			Path:    "/test/stats_file" + string(rune(i)) + ".templ",
			ModTime: time.Now(),
			Size:    1024,
		}
		fw.debouncer.addEvent(event)
	}

	// Check updated stats
	stats = fw.GetStats()
	pendingCount := stats["pending_events"].(int)

	if pendingCount != eventsAdded {
		t.Errorf("Expected %d pending events, got %d", eventsAdded, pendingCount)
	}

	// Verify other stats are present and reasonable
	if stats["max_pending"].(int) != MaxPendingEvents {
		t.Error("Max pending events stat incorrect")
	}
	if stats["max_batch_size"].(int) <= 0 {
		t.Error("Max batch size should be positive")
	}
}

// BenchmarkEnhancedWatcher benchmarks the enhanced watcher implementation
func BenchmarkEnhancedWatcher(b *testing.B) {
	fw, err := NewFileWatcher(1 * time.Millisecond)
	if err != nil {
		b.Fatalf("Failed to create file watcher: %v", err)
	}
	defer fw.Stop()

	fw.AddHandler(func(events []ChangeEvent) error {
		return nil
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		event := ChangeEvent{
			Type:    EventTypeModified,
			Path:    "/test/bench_file" + string(rune(i%100)) + ".templ",
			ModTime: time.Now(),
			Size:    1024,
		}
		fw.debouncer.addEvent(event)
	}
}
