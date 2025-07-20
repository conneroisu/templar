package testing

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestResourceTracker_BasicTracking(t *testing.T) {
	tracker := NewResourceTracker("basic_test")

	// Get initial usage
	initialUsage := tracker.GetResourceUsage()
	if initialUsage.GoroutineDiff != 0 {
		t.Errorf("Expected 0 goroutine diff initially, got %d", initialUsage.GoroutineDiff)
	}

	// Take a sample
	sample := tracker.TakeSample()
	if sample.Goroutines <= 0 {
		t.Error("Expected positive goroutine count")
	}

	// Generate report
	report := tracker.GenerateReport()
	if len(report) == 0 {
		t.Error("Expected non-empty report")
	}
	t.Logf("Report:\n%s", report)
}

func TestResourceTracker_GoroutineLeak(t *testing.T) {
	tracker := NewResourceTracker("goroutine_leak_test")

	// Create goroutine leaks
	var wg sync.WaitGroup
	stopCh := make(chan struct{})

	// Start 5 goroutines that will leak
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-stopCh // Block until we close the channel
		}()
	}

	// Wait a bit for goroutines to start
	time.Sleep(10 * time.Millisecond)

	// This should detect the leak (we expect failure here for testing)
	// We'll use a test helper to capture the error
	testHelper := &TestHelper{}
	tracker.CheckLeaksWithLimits(testHelper, ResourceLimits{
		MaxGoroutineIncrease: 2, // Limit to 2, but we created 5
		MaxFileIncrease:      10,
		MaxMemoryIncrease:    10 * 1024 * 1024,
		MaxObjectIncrease:    1000,
		TolerancePercent:     0.1,
	})

	if !testHelper.HasError() {
		t.Error("Expected goroutine leak to be detected")
	}

	// Clean up
	close(stopCh)
	wg.Wait()
}

func TestResourceTracker_NoLeakDetection(t *testing.T) {
	tracker := NewResourceTracker("no_leak_test")

	// Do some work that shouldn't leak
	for i := 0; i < 100; i++ {
		go func() {
			// Do nothing and exit immediately
		}()
	}

	// Wait for goroutines to finish
	time.Sleep(50 * time.Millisecond)
	runtime.GC()
	runtime.GC()
	time.Sleep(10 * time.Millisecond)

	// This should not detect any leaks
	tracker.CheckLeaks(t)
}

func TestResourceTracker_MemoryTracking(t *testing.T) {
	tracker := NewResourceTracker("memory_test")

	// Allocate some memory
	var allocations [][]byte
	for i := 0; i < 100; i++ {
		allocation := make([]byte, 1024) // 1KB each
		allocations = append(allocations, allocation)
	}

	// Take sample after allocation
	usage := tracker.GetResourceUsage()

	if usage.MemoryDiff <= 0 {
		t.Logf("Memory diff: %d (might be optimized away)", usage.MemoryDiff)
	}

	// Clean up
	allocations = nil
	runtime.GC()
	runtime.GC()
	time.Sleep(10 * time.Millisecond)

	// Check that memory was cleaned up
	tracker.CheckLeaks(t)
}

func TestResourceTracker_SamplingHistory(t *testing.T) {
	tracker := NewResourceTracker("sampling_test")

	// Take several samples
	for i := 0; i < 5; i++ {
		tracker.TakeSample()
		time.Sleep(1 * time.Millisecond)
	}

	samples := tracker.GetSamples()
	if len(samples) < 6 { // Initial + 5 manual samples
		t.Errorf("Expected at least 6 samples, got %d", len(samples))
	}

	// Verify samples are in chronological order
	for i := 1; i < len(samples); i++ {
		if samples[i].Timestamp.Before(samples[i-1].Timestamp) {
			t.Error("Samples are not in chronological order")
		}
	}
}

func TestResourceMonitor_ContinuousMonitoring(t *testing.T) {
	monitor := NewResourceMonitor("continuous_test", 10*time.Millisecond)

	// Start monitoring
	monitor.Start()

	// Let it run for a bit
	time.Sleep(50 * time.Millisecond)

	// Stop monitoring
	monitor.Stop()

	// Check that samples were collected
	tracker := monitor.GetTracker()
	samples := tracker.GetSamples()

	if len(samples) < 3 { // Should have multiple samples
		t.Errorf("Expected multiple samples from continuous monitoring, got %d", len(samples))
	}
}

func TestMemoryPressureTest_BasicPressure(t *testing.T) {
	test := NewMemoryPressureTest("pressure_test")

	// Apply memory pressure (10MB in 1MB chunks)
	test.ApplyPressure(10, 1)

	// Check that memory usage increased
	usage := test.GetTracker().GetResourceUsage()
	if usage.MemoryDiff <= 0 {
		t.Logf("Memory diff: %d (allocation might be optimized)", usage.MemoryDiff)
	}

	// Release pressure
	test.ReleasePressure()

	// Check memory recovery
	test.CheckMemoryRecovery(t)
}

func TestMemoryPressureTest_LargePressure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large memory pressure test in short mode")
	}

	test := NewMemoryPressureTest("large_pressure_test")

	// Apply significant memory pressure (100MB in 10MB chunks)
	test.ApplyPressure(100, 10)

	// Verify memory usage
	usage := test.GetTracker().GetResourceUsage()
	t.Logf("Memory usage after pressure: %d bytes", usage.MemoryDiff)

	// Release and check recovery
	test.ReleasePressure()
	test.CheckMemoryRecovery(t)
}

func TestResourceLimits_CustomLimits(t *testing.T) {
	tracker := NewResourceTracker("custom_limits_test")

	// Create a small goroutine leak
	done := make(chan struct{})
	go func() {
		<-done
	}()

	// Use very strict limits
	strictLimits := ResourceLimits{
		MaxGoroutineIncrease: 0, // No goroutine increase allowed
		MaxFileIncrease:      0,
		MaxMemoryIncrease:    1024, // 1KB
		MaxObjectIncrease:    10,
		TolerancePercent:     0.01, // 1% tolerance
	}

	testHelper := &TestHelper{}
	tracker.CheckLeaksWithLimits(testHelper, strictLimits)

	if !testHelper.HasError() {
		t.Error("Expected strict limits to detect the goroutine")
	}

	// Clean up
	close(done)
}

func TestResourceTracker_ConcurrentAccess(t *testing.T) {
	tracker := NewResourceTracker("concurrent_test")

	// Multiple goroutines taking samples concurrently
	var wg sync.WaitGroup
	const numGoroutines = 10
	const samplesPerGoroutine = 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < samplesPerGoroutine; j++ {
				tracker.TakeSample()
				time.Sleep(time.Millisecond)
			}
		}()
	}

	wg.Wait()

	// Should have many samples without crashing
	samples := tracker.GetSamples()
	expectedMinSamples := numGoroutines * samplesPerGoroutine
	if len(samples) < expectedMinSamples {
		t.Errorf("Expected at least %d samples, got %d", expectedMinSamples, len(samples))
	}
}

// Test helper that captures testing.T calls for verification
type TestHelper struct {
	errors []string
	logs   []string
	mu     sync.Mutex
}

func (th *TestHelper) Errorf(format string, args ...interface{}) {
	th.mu.Lock()
	defer th.mu.Unlock()
	th.errors = append(th.errors, fmt.Sprintf(format, args...))
}

func (th *TestHelper) Logf(format string, args ...interface{}) {
	th.mu.Lock()
	defer th.mu.Unlock()
	th.logs = append(th.logs, fmt.Sprintf(format, args...))
}

func (th *TestHelper) HasError() bool {
	th.mu.Lock()
	defer th.mu.Unlock()
	return len(th.errors) > 0
}

func (th *TestHelper) GetErrors() []string {
	th.mu.Lock()
	defer th.mu.Unlock()
	result := make([]string, len(th.errors))
	copy(result, th.errors)
	return result
}

// Benchmark tests
func BenchmarkResourceTracker_TakeSample(b *testing.B) {
	tracker := NewResourceTracker("bench_test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.TakeSample()
	}
}

func BenchmarkResourceTracker_GetResourceUsage(b *testing.B) {
	tracker := NewResourceTracker("bench_usage_test")

	// Take some samples first
	for i := 0; i < 10; i++ {
		tracker.TakeSample()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.GetResourceUsage()
	}
}

func BenchmarkResourceTracker_ConcurrentSampling(b *testing.B) {
	tracker := NewResourceTracker("bench_concurrent_test")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tracker.TakeSample()
		}
	})
}

func BenchmarkMemoryPressure_SmallAllocations(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		test := NewMemoryPressureTest("bench_pressure")
		test.ApplyPressure(1, 1) // 1MB
		test.ReleasePressure()
	}
}

// Integration test combining error injection and resource tracking
func TestIntegration_ErrorInjectionWithResourceTracking(t *testing.T) {
	tracker := NewResourceTracker("integration_test")
	injector := NewErrorInjector()

	// Configure error injection for memory allocation
	injector.InjectError("memory.alloc", ErrOutOfMemory)

	// Simulate a function that allocates memory
	simulateMemoryAllocation := func() error {
		if err := injector.ShouldFail("memory.alloc"); err != nil {
			return err
		}

		// Allocate memory if no error injection
		allocation := make([]byte, 1024*1024) // 1MB
		_ = allocation
		return nil
	}

	// Try allocation (should fail due to injection)
	if err := simulateMemoryAllocation(); err == nil {
		t.Error("Expected memory allocation to fail due to error injection")
	}

	// Disable injection and try again
	injector.RemoveTarget("memory.alloc")
	if err := simulateMemoryAllocation(); err != nil {
		t.Errorf("Expected memory allocation to succeed after disabling injection: %v", err)
	}

	// Check for resource leaks
	tracker.CheckLeaks(t)
}
