package testutils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

// TestOptimizer provides intelligent test execution optimization.
type TestOptimizer struct {
	maxWorkers    int
	testTimeout   time.Duration
	cacheEnabled  bool
	parallelTests map[string]bool
	mu            sync.RWMutex
}

// OptimizerConfig configures the test optimizer.
type OptimizerConfig struct {
	MaxWorkers   int
	TestTimeout  time.Duration
	CacheEnabled bool
}

// TestMetrics tracks test execution metrics.
type TestMetrics struct {
	StartTime      time.Time
	EndTime        time.Time
	Duration       time.Duration
	MemoryUsage    uint64
	GoroutineCount int
	CacheHit       bool
	TestName       string
	PackageName    string
}

// NewTestOptimizer creates a new test optimizer with optimal settings.
func NewTestOptimizer(config OptimizerConfig) *TestOptimizer {
	maxWorkers := config.MaxWorkers
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
		if maxWorkers > 8 {
			maxWorkers = 8 // Avoid overwhelming system
		}
	}

	timeout := config.TestTimeout
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}

	return &TestOptimizer{
		maxWorkers:    maxWorkers,
		testTimeout:   timeout,
		cacheEnabled:  config.CacheEnabled,
		parallelTests: make(map[string]bool),
	}
}

// OptimizeTest applies optimizations to test execution.
func (to *TestOptimizer) OptimizeTest(t *testing.T, testFunc func(*testing.T)) {
	// Enable parallel execution for suitable tests
	if to.canRunInParallel(t.Name()) {
		t.Parallel()
	}

	// Set up test context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), to.testTimeout)
	defer cancel()

	// Track test metrics
	metrics := to.startMetrics(t.Name())
	defer to.finishMetrics(metrics)

	// Create optimized test context
	tc := &OptimizedTestContext{
		T:       t,
		Context: ctx,
		tempDir: t.TempDir(),
		cleanup: make([]func(), 0),
		metrics: metrics,
	}

	// Run test with resource monitoring
	done := make(chan struct{})
	go func() {
		defer close(done)
		testFunc(tc.T)
	}()

	// Monitor test execution
	select {
	case <-done:
		// Test completed normally
	case <-ctx.Done():
		t.Fatalf("Test timed out after %v", to.testTimeout)
	}
}

// OptimizeParallelTests runs a group of tests with optimal parallelization.
func (to *TestOptimizer) OptimizeParallelTests(t *testing.T, tests map[string]func(*testing.T)) {
	workerChan := make(chan struct{}, to.maxWorkers)
	var wg sync.WaitGroup

	for name, testFunc := range tests {
		wg.Add(1)
		go func(testName string, fn func(*testing.T)) {
			defer wg.Done()

			// Acquire worker slot
			workerChan <- struct{}{}
			defer func() { <-workerChan }()

			// Run individual test
			t.Run(testName, func(subT *testing.T) {
				to.OptimizeTest(subT, fn)
			})
		}(name, testFunc)
	}

	wg.Wait()
}

// canRunInParallel determines if a test can run in parallel safely.
func (to *TestOptimizer) canRunInParallel(testName string) bool {
	to.mu.RLock()
	defer to.mu.RUnlock()

	// Check cache first
	if canParallel, exists := to.parallelTests[testName]; exists {
		return canParallel
	}

	// Apply heuristics to determine parallelization safety
	canParallel := to.analyzeTestParallelSafety(testName)

	to.mu.Lock()
	to.parallelTests[testName] = canParallel
	to.mu.Unlock()

	return canParallel
}

// analyzeTestParallelSafety uses heuristics to determine if test is parallel-safe.
func (to *TestOptimizer) analyzeTestParallelSafety(testName string) bool {
	// Tests that typically shouldn't run in parallel
	unsafePatterns := []string{
		"Integration",
		"E2E",
		"Server",
		"Database",
		"FileSystem",
		"Port",
		"Network",
		"Global",
		"Singleton",
	}

	for _, pattern := range unsafePatterns {
		if containsIgnoreCase(testName, pattern) {
			return false
		}
	}

	// Tests that are typically safe for parallel execution
	safePatterns := []string{
		"Unit",
		"Parser",
		"Validator",
		"Utility",
		"Helper",
		"Pure",
		"Stateless",
	}

	for _, pattern := range safePatterns {
		if containsIgnoreCase(testName, pattern) {
			return true
		}
	}

	// Default to safe for parallel execution
	return true
}

// startMetrics begins tracking test metrics.
func (to *TestOptimizer) startMetrics(testName string) *TestMetrics {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return &TestMetrics{
		StartTime:      time.Now(),
		TestName:       testName,
		MemoryUsage:    memStats.Alloc,
		GoroutineCount: runtime.NumGoroutine(),
	}
}

// finishMetrics completes test metrics tracking.
func (to *TestOptimizer) finishMetrics(metrics *TestMetrics) {
	metrics.EndTime = time.Now()
	metrics.Duration = metrics.EndTime.Sub(metrics.StartTime)

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Calculate memory delta
	if memStats.Alloc > metrics.MemoryUsage {
		metrics.MemoryUsage = memStats.Alloc - metrics.MemoryUsage
	} else {
		metrics.MemoryUsage = 0
	}

	// Log metrics if test took more than 1 second
	if metrics.Duration > time.Second {
		fmt.Printf("⏱️  Test %s: %v (mem: %d bytes, goroutines: %d)\n",
			metrics.TestName, metrics.Duration, metrics.MemoryUsage, metrics.GoroutineCount)
	}
}

// OptimizedTestContext provides an optimized context for test execution.
type OptimizedTestContext struct {
	*testing.T
	context.Context
	tempDir string
	cleanup []func()
	metrics *TestMetrics
}

// TempDir returns the temporary directory for the test.
func (otc *OptimizedTestContext) TempDir() string {
	return otc.tempDir
}

// AddCleanup adds a cleanup function to be called after test completion.
func (otc *OptimizedTestContext) AddCleanup(cleanup func()) {
	otc.cleanup = append(otc.cleanup, cleanup)
}

// Cleanup runs all registered cleanup functions.
func (otc *OptimizedTestContext) Cleanup() {
	for i := len(otc.cleanup) - 1; i >= 0; i-- {
		otc.cleanup[i]()
	}
}

// SmartTestRunner provides intelligent test selection and execution.
type SmartTestRunner struct {
	optimizer     *TestOptimizer
	changedFiles  []string
	testCache     map[string]time.Time
	affectedTests map[string]bool
}

// NewSmartTestRunner creates a new smart test runner.
func NewSmartTestRunner() *SmartTestRunner {
	return &SmartTestRunner{
		optimizer:     NewTestOptimizer(OptimizerConfig{CacheEnabled: true}),
		testCache:     make(map[string]time.Time),
		affectedTests: make(map[string]bool),
	}
}

// ShouldRunTest determines if a test should be executed based on file changes.
func (str *SmartTestRunner) ShouldRunTest(testName, packagePath string) bool {
	// Always run if no change tracking
	if len(str.changedFiles) == 0 {
		return true
	}

	// Check if test or related files have changed
	testFile := filepath.Join(packagePath, testName+"_test.go")
	if str.fileHasChanged(testFile) {
		return true
	}

	// Check if source files in the same package have changed
	sourceFiles, _ := filepath.Glob(filepath.Join(packagePath, "*.go"))
	for _, sourceFile := range sourceFiles {
		if str.fileHasChanged(sourceFile) {
			return true
		}
	}

	return false
}

// fileHasChanged checks if a file has been modified recently.
func (str *SmartTestRunner) fileHasChanged(filePath string) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return true // Run test if we can't determine file status
	}

	if lastRun, exists := str.testCache[filePath]; exists {
		return info.ModTime().After(lastRun)
	}

	return true // Run test if no cache entry exists
}

// UpdateFileCache updates the file modification cache.
func (str *SmartTestRunner) UpdateFileCache(filePath string) {
	str.testCache[filePath] = time.Now()
}

// TestProfiler provides performance profiling for tests.
type TestProfiler struct {
	profiles map[string]*TestMetrics
	mu       sync.RWMutex
}

// NewTestProfiler creates a new test profiler.
func NewTestProfiler() *TestProfiler {
	return &TestProfiler{
		profiles: make(map[string]*TestMetrics),
	}
}

// ProfileTest runs a test with performance profiling.
func (tp *TestProfiler) ProfileTest(t *testing.T, testFunc func(*testing.T)) {
	startTime := time.Now()

	var startMem, endMem runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	// Run the test
	testFunc(t)

	runtime.GC()
	runtime.ReadMemStats(&endMem)
	endTime := time.Now()

	// Calculate metrics
	metrics := &TestMetrics{
		StartTime:      startTime,
		EndTime:        endTime,
		Duration:       endTime.Sub(startTime),
		MemoryUsage:    endMem.Alloc - startMem.Alloc,
		GoroutineCount: runtime.NumGoroutine(),
		TestName:       t.Name(),
	}

	tp.mu.Lock()
	tp.profiles[t.Name()] = metrics
	tp.mu.Unlock()

	// Log slow tests
	if metrics.Duration > 5*time.Second {
		t.Logf("⚠️  Slow test detected: %s took %v", t.Name(), metrics.Duration)
	}

	// Log memory-intensive tests
	if metrics.MemoryUsage > 10*1024*1024 { // 10MB
		t.Logf("🧠 Memory-intensive test: %s used %d bytes", t.Name(), metrics.MemoryUsage)
	}
}

// GetProfile returns the performance profile for a test.
func (tp *TestProfiler) GetProfile(testName string) (*TestMetrics, bool) {
	tp.mu.RLock()
	defer tp.mu.RUnlock()
	profile, exists := tp.profiles[testName]

	return profile, exists
}

// GetSlowTests returns tests that exceed the duration threshold.
func (tp *TestProfiler) GetSlowTests(threshold time.Duration) []*TestMetrics {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	var slowTests []*TestMetrics
	for _, profile := range tp.profiles {
		if profile.Duration > threshold {
			slowTests = append(slowTests, profile)
		}
	}

	return slowTests
}

// GetMemoryIntensiveTests returns tests that exceed the memory threshold.
func (tp *TestProfiler) GetMemoryIntensiveTests(threshold uint64) []*TestMetrics {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	var memoryIntensiveTests []*TestMetrics
	for _, profile := range tp.profiles {
		if profile.MemoryUsage > threshold {
			memoryIntensiveTests = append(memoryIntensiveTests, profile)
		}
	}

	return memoryIntensiveTests
}

// Helper functions

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr || containsIgnoreCaseRec(s[1:], substr))))
}

func containsIgnoreCaseRec(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	if s[:len(substr)] == substr {
		return true
	}

	return containsIgnoreCaseRec(s[1:], substr)
}

// Global optimizer instance for easy access.
var globalOptimizer = NewTestOptimizer(OptimizerConfig{
	CacheEnabled: true,
})

var globalProfiler = NewTestProfiler()

// OptimizedTest is a helper function for optimized test execution.
func OptimizedTest(t *testing.T, testFunc func(*testing.T)) {
	globalOptimizer.OptimizeTest(t, testFunc)
}

// ProfiledTest is a helper function for profiled test execution.
func ProfiledTest(t *testing.T, testFunc func(*testing.T)) {
	globalProfiler.ProfileTest(t, testFunc)
}

// OptimizedAndProfiledTest combines optimization and profiling.
func OptimizedAndProfiledTest(t *testing.T, testFunc func(*testing.T)) {
	globalOptimizer.OptimizeTest(t, func(innerT *testing.T) {
		globalProfiler.ProfileTest(innerT, testFunc)
	})
}
