package testing

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"
)

// ResourceTracker tracks resource usage to detect leaks
type ResourceTracker struct {
	initialGoroutines int
	initialFiles      int
	initialMemory     uint64
	initialObjects    int64
	name              string
	startTime         time.Time
	samples           []ResourceSample
	mu                sync.Mutex
}

// ResourceSample represents a point-in-time resource measurement
type ResourceSample struct {
	Timestamp  time.Time
	Goroutines int
	Files      int
	Memory     uint64
	Objects    int64
	HeapAlloc  uint64
	HeapSys    uint64
	GCCycles   uint32
}

// ResourceLimits defines acceptable resource usage limits
type ResourceLimits struct {
	MaxGoroutineIncrease int     // Maximum allowed goroutine increase
	MaxFileIncrease      int     // Maximum allowed file handle increase
	MaxMemoryIncrease    uint64  // Maximum allowed memory increase (bytes)
	MaxObjectIncrease    int64   // Maximum allowed object increase
	TolerancePercent     float64 // Tolerance for resource increases (0.0 to 1.0)
}

// DefaultResourceLimits returns sensible default limits
func DefaultResourceLimits() ResourceLimits {
	return ResourceLimits{
		MaxGoroutineIncrease: 5,
		MaxFileIncrease:      10,
		MaxMemoryIncrease:    10 * 1024 * 1024, // 10MB
		MaxObjectIncrease:    1000,
		TolerancePercent:     0.1, // 10% tolerance
	}
}

// NewResourceTracker creates a new resource tracker
func NewResourceTracker(name string) *ResourceTracker {
	rt := &ResourceTracker{
		name:      name,
		startTime: time.Now(),
		samples:   make([]ResourceSample, 0, 100),
	}

	rt.captureBaseline()
	return rt
}

// captureBaseline captures initial resource state
func (rt *ResourceTracker) captureBaseline() {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	// Force GC to get accurate baseline
	runtime.GC()
	runtime.GC() // Run twice to ensure cleanup
	time.Sleep(10 * time.Millisecond)

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	rt.initialGoroutines = runtime.NumGoroutine()
	rt.initialFiles = getOpenFileCount()
	rt.initialMemory = memStats.Alloc
	rt.initialObjects = int64(memStats.Mallocs - memStats.Frees)

	// Take initial sample
	rt.samples = append(rt.samples, ResourceSample{
		Timestamp:  time.Now(),
		Goroutines: rt.initialGoroutines,
		Files:      rt.initialFiles,
		Memory:     rt.initialMemory,
		Objects:    rt.initialObjects,
		HeapAlloc:  memStats.Alloc,
		HeapSys:    memStats.Sys,
		GCCycles:   memStats.NumGC,
	})
}

// TakeSample captures current resource usage
func (rt *ResourceTracker) TakeSample() ResourceSample {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	sample := ResourceSample{
		Timestamp:  time.Now(),
		Goroutines: runtime.NumGoroutine(),
		Files:      getOpenFileCount(),
		Memory:     memStats.Alloc,
		Objects:    int64(memStats.Mallocs - memStats.Frees),
		HeapAlloc:  memStats.Alloc,
		HeapSys:    memStats.Sys,
		GCCycles:   memStats.NumGC,
	}

	rt.samples = append(rt.samples, sample)
	return sample
}

// CheckLeaks checks for resource leaks against default limits
func (rt *ResourceTracker) CheckLeaks(t *testing.T) {
	rt.CheckLeaksWithLimits(t, DefaultResourceLimits())
}

// TestingInterface defines the interface needed for leak checking
type TestingInterface interface {
	Errorf(format string, args ...interface{})
	Logf(format string, args ...interface{})
}

// CheckLeaksWithLimits checks for resource leaks against specified limits
func (rt *ResourceTracker) CheckLeaksWithLimits(t TestingInterface, limits ResourceLimits) {
	// Force cleanup before checking
	runtime.GC()
	runtime.GC()
	time.Sleep(50 * time.Millisecond)

	currentSample := rt.TakeSample()

	// Check goroutine leaks
	goroutineDiff := currentSample.Goroutines - rt.initialGoroutines
	if goroutineDiff > limits.MaxGoroutineIncrease {
		t.Errorf(
			"%s: Goroutine leak detected: %d initial, %d current (+%d, limit: +%d)",
			rt.name,
			rt.initialGoroutines,
			currentSample.Goroutines,
			goroutineDiff,
			limits.MaxGoroutineIncrease,
		)

		// Log goroutine stack trace for debugging
		if goroutineDiff > 0 {
			t.Logf("Goroutine stack trace:\n%s", getGoroutineStackTrace())
		}
	}

	// Check file handle leaks
	fileDiff := currentSample.Files - rt.initialFiles
	if fileDiff > limits.MaxFileIncrease {
		t.Errorf("%s: File handle leak detected: %d initial, %d current (+%d, limit: +%d)",
			rt.name, rt.initialFiles, currentSample.Files, fileDiff, limits.MaxFileIncrease)
	}

	// Check memory leaks (with tolerance)
	memoryDiff := int64(currentSample.Memory) - int64(rt.initialMemory)
	toleranceBytes := int64(float64(rt.initialMemory) * limits.TolerancePercent)
	if memoryDiff > int64(limits.MaxMemoryIncrease)+toleranceBytes {
		t.Errorf(
			"%s: Memory leak detected: %d initial, %d current (+%d bytes, limit: +%d bytes + tolerance)",
			rt.name,
			rt.initialMemory,
			currentSample.Memory,
			memoryDiff,
			limits.MaxMemoryIncrease,
		)
	}

	// Check object leaks
	objectDiff := currentSample.Objects - rt.initialObjects
	toleranceObjects := int64(float64(rt.initialObjects) * limits.TolerancePercent)
	if objectDiff > limits.MaxObjectIncrease+toleranceObjects {
		t.Errorf(
			"%s: Object leak detected: %d initial, %d current (+%d objects, limit: +%d objects + tolerance)",
			rt.name,
			rt.initialObjects,
			currentSample.Objects,
			objectDiff,
			limits.MaxObjectIncrease,
		)
	}
}

// GetResourceUsage returns current resource usage compared to baseline
func (rt *ResourceTracker) GetResourceUsage() ResourceUsage {
	currentSample := rt.TakeSample()

	return ResourceUsage{
		Name:          rt.name,
		Duration:      time.Since(rt.startTime),
		GoroutineDiff: currentSample.Goroutines - rt.initialGoroutines,
		FileDiff:      currentSample.Files - rt.initialFiles,
		MemoryDiff:    int64(currentSample.Memory) - int64(rt.initialMemory),
		ObjectDiff:    currentSample.Objects - rt.initialObjects,
		Initial:       rt.getInitialSample(),
		Current:       currentSample,
		SampleCount:   len(rt.samples),
	}
}

// ResourceUsage represents resource usage comparison
type ResourceUsage struct {
	Name          string
	Duration      time.Duration
	GoroutineDiff int
	FileDiff      int
	MemoryDiff    int64
	ObjectDiff    int64
	Initial       ResourceSample
	Current       ResourceSample
	SampleCount   int
}

// getInitialSample returns the first (baseline) sample
func (rt *ResourceTracker) getInitialSample() ResourceSample {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if len(rt.samples) > 0 {
		return rt.samples[0]
	}
	return ResourceSample{}
}

// GetSamples returns all resource samples
func (rt *ResourceTracker) GetSamples() []ResourceSample {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	samples := make([]ResourceSample, len(rt.samples))
	copy(samples, rt.samples)
	return samples
}

// GenerateReport generates a detailed resource usage report
func (rt *ResourceTracker) GenerateReport() string {
	usage := rt.GetResourceUsage()

	return fmt.Sprintf(`Resource Usage Report: %s
Duration: %v
Goroutines: %d → %d (%+d)
File Handles: %d → %d (%+d)  
Memory: %d → %d (%+d bytes)
Objects: %d → %d (%+d)
Samples Taken: %d
`,
		usage.Name,
		usage.Duration,
		usage.Initial.Goroutines, usage.Current.Goroutines, usage.GoroutineDiff,
		usage.Initial.Files, usage.Current.Files, usage.FileDiff,
		usage.Initial.Memory, usage.Current.Memory, usage.MemoryDiff,
		usage.Initial.Objects, usage.Current.Objects, usage.ObjectDiff,
		usage.SampleCount,
	)
}

// Utility functions

// getOpenFileCount returns the number of open file descriptors
func getOpenFileCount() int {
	// On Unix-like systems, count files in /proc/self/fd
	if entries, err := os.ReadDir("/proc/self/fd"); err == nil {
		return len(entries)
	}

	// Fallback: return a reasonable default
	return 10
}

// getGoroutineStackTrace returns stack traces of all goroutines
func getGoroutineStackTrace() string {
	buf := make([]byte, 64*1024)
	n := runtime.Stack(buf, true)
	return string(buf[:n])
}

// ResourceMonitor provides continuous resource monitoring
type ResourceMonitor struct {
	tracker  *ResourceTracker
	interval time.Duration
	stopCh   chan struct{}
	mu       sync.Mutex
	running  bool
}

// NewResourceMonitor creates a continuous resource monitor
func NewResourceMonitor(name string, interval time.Duration) *ResourceMonitor {
	return &ResourceMonitor{
		tracker:  NewResourceTracker(name),
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins continuous monitoring
func (rm *ResourceMonitor) Start() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.running {
		return
	}

	rm.running = true
	go rm.monitor()
}

// Stop ends continuous monitoring
func (rm *ResourceMonitor) Stop() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if !rm.running {
		return
	}

	rm.running = false
	close(rm.stopCh)
}

// monitor runs the monitoring loop
func (rm *ResourceMonitor) monitor() {
	ticker := time.NewTicker(rm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rm.tracker.TakeSample()
		case <-rm.stopCh:
			return
		}
	}
}

// GetTracker returns the underlying resource tracker
func (rm *ResourceMonitor) GetTracker() *ResourceTracker {
	return rm.tracker
}

// MemoryPressureTest applies memory pressure and checks for leaks
type MemoryPressureTest struct {
	tracker     *ResourceTracker
	allocations [][]byte
	mu          sync.Mutex
}

// NewMemoryPressureTest creates a new memory pressure test
func NewMemoryPressureTest(name string) *MemoryPressureTest {
	return &MemoryPressureTest{
		tracker:     NewResourceTracker(name + "_memory_pressure"),
		allocations: make([][]byte, 0),
	}
}

// ApplyPressure allocates memory to create pressure
func (mpt *MemoryPressureTest) ApplyPressure(totalMB int, chunkSizeMB int) {
	mpt.mu.Lock()
	defer mpt.mu.Unlock()

	chunkSize := chunkSizeMB * 1024 * 1024
	numChunks := totalMB / chunkSizeMB

	for i := 0; i < numChunks; i++ {
		chunk := make([]byte, chunkSize)
		// Write to the memory to ensure it's allocated
		for j := 0; j < len(chunk); j += 4096 {
			chunk[j] = byte(i)
		}
		mpt.allocations = append(mpt.allocations, chunk)
		mpt.tracker.TakeSample()
	}
}

// ReleasePressure frees all allocated memory
func (mpt *MemoryPressureTest) ReleasePressure() {
	mpt.mu.Lock()
	defer mpt.mu.Unlock()

	mpt.allocations = nil
	runtime.GC()
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	mpt.tracker.TakeSample()
}

// CheckMemoryRecovery verifies memory was properly released
func (mpt *MemoryPressureTest) CheckMemoryRecovery(t *testing.T) {
	mpt.tracker.CheckLeaksWithLimits(t, ResourceLimits{
		MaxGoroutineIncrease: 2,
		MaxFileIncrease:      5,
		MaxMemoryIncrease:    50 * 1024 * 1024, // 50MB tolerance for memory pressure test
		MaxObjectIncrease:    10000,
		TolerancePercent:     0.2, // 20% tolerance
	})
}

// GetTracker returns the underlying resource tracker
func (mpt *MemoryPressureTest) GetTracker() *ResourceTracker {
	return mpt.tracker
}
