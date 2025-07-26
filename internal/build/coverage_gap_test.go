package build

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/errors"
	"github.com/conneroisu/templar/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildCache_GetHash_TTL_Coverage tests the TTL expiration logic specifically for GetHash method
func TestBuildCache_GetHash_TTL_Coverage(t *testing.T) {
	t.Run("GetHash TTL expiration with cleanup", func(t *testing.T) {
		cache := NewBuildCache(1000, 50*time.Millisecond) // 50ms TTL

		// Set a hash value
		cache.SetHash("metadata-key", "test-hash-value")

		// Should be found immediately
		hash, found := cache.GetHash("metadata-key")
		assert.True(t, found, "Hash should be found immediately")
		assert.Equal(t, "test-hash-value", hash, "Hash value should match")

		// Wait for TTL expiration
		time.Sleep(60 * time.Millisecond)

		// Should be expired and removed during GetHash call
		hash, found = cache.GetHash("metadata-key")
		assert.False(t, found, "Hash should be expired and not found")
		assert.Empty(t, hash, "Hash should be empty when not found")

		// Cache should be cleaned up automatically
		count, size, _ := cache.GetStats()
		assert.Equal(t, 0, count, "Expired hash entry should be removed from cache")
		assert.Equal(t, int64(0), size, "Cache size should be 0 after cleanup")
	})

	t.Run("GetHash TTL edge case - exact expiration time", func(t *testing.T) {
		cache := NewBuildCache(1000, 100*time.Millisecond)

		cache.SetHash("edge-case-key", "edge-hash")

		// Wait exactly the TTL duration
		time.Sleep(100 * time.Millisecond)

		// Should be expired (time.Since() should be >= TTL)
		hash, found := cache.GetHash("edge-case-key")
		assert.False(t, found, "Hash should be expired at exact TTL duration")
		assert.Empty(t, hash, "Hash should be empty when expired")
	})

	t.Run("GetHash TTL with concurrent access", func(t *testing.T) {
		cache := NewBuildCache(1000, 30*time.Millisecond)

		cache.SetHash("concurrent-key", "concurrent-hash")

		var wg sync.WaitGroup
		var results []bool
		var mutex sync.Mutex

		// Start multiple goroutines that will try to access after TTL expiration
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				time.Sleep(40 * time.Millisecond) // Wait for expiration
				_, found := cache.GetHash("concurrent-key")
				mutex.Lock()
				results = append(results, found)
				mutex.Unlock()
			}()
		}

		wg.Wait()

		// All should find the entry expired
		for _, found := range results {
			assert.False(t, found, "All concurrent access should find entry expired")
		}
	})
}

// TestHandleBuildResult_Coverage tests the missing coverage areas in handleBuildResult
func TestHandleBuildResult_Coverage(t *testing.T) {
	t.Run("handleBuildResult with parsed errors", func(t *testing.T) {
		// Create a mock registry for pipeline
		reg := NewMockComponentRegistry()
		bp := NewBuildPipeline(1, reg)

		// Create a component
		component := &types.ComponentInfo{
			Name:     "TestComponent",
			FilePath: "/test/component.templ",
			Package:  "test",
		}

		// Create a mock ParsedError for testing
		mockError := &errors.ParsedError{
			Type:      errors.BuildErrorTypeTemplSyntax,
			Severity:  errors.ErrorSeverityError,
			Component: "TestComponent",
			File:      "/test/component.templ",
			Line:      10,
			Column:    5,
			Message:   "test error",
			RawError:  "raw error output",
		}

		// Create a BuildResult with error and parsed errors
		result := BuildResult{
			Component: component,
			Error:     fmt.Errorf("build failed"),
			ParsedErrors: []*errors.ParsedError{
				mockError,
			},
			Duration: 100 * time.Millisecond,
			CacheHit: false,
		}

		// Capture callback executions
		var callbackResults []BuildResult
		var callbackMutex sync.Mutex
		bp.AddCallback(func(r BuildResult) {
			callbackMutex.Lock()
			callbackResults = append(callbackResults, r)
			callbackMutex.Unlock()
		})

		// Handle the result - this should cover the error printing path with parsed errors
		bp.handleBuildResult(result)

		// Verify callback was called
		callbackMutex.Lock()
		assert.Equal(t, 1, len(callbackResults), "Callback should be called")
		assert.Equal(
			t,
			component.Name,
			callbackResults[0].Component.Name,
			"Callback should receive correct component",
		)
		callbackMutex.Unlock()
	})

	t.Run("handleBuildResult with cache hit", func(t *testing.T) {
		reg := NewMockComponentRegistry()
		bp := NewBuildPipeline(1, reg)

		component := &types.ComponentInfo{
			Name:     "CachedComponent",
			FilePath: "/test/cached.templ",
			Package:  "test",
		}

		// Create a successful result with cache hit
		result := BuildResult{
			Component: component,
			Error:     nil,
			Duration:  50 * time.Millisecond,
			CacheHit:  true, // This path is not well covered
		}

		var callbackCalled bool
		var callbackMutex sync.Mutex
		bp.AddCallback(func(r BuildResult) {
			callbackMutex.Lock()
			callbackCalled = true
			callbackMutex.Unlock()
		})

		// Handle the result - this should cover the cache hit success path
		bp.handleBuildResult(result)

		// Verify callback was called
		callbackMutex.Lock()
		assert.True(t, callbackCalled, "Callback should be called for cache hit")
		callbackMutex.Unlock()
	})

	t.Run("handleBuildResult with multiple callbacks", func(t *testing.T) {
		reg := NewMockComponentRegistry()
		bp := NewBuildPipeline(1, reg)

		component := &types.ComponentInfo{
			Name:     "MultiCallbackComponent",
			FilePath: "/test/multi.templ",
			Package:  "test",
		}

		result := BuildResult{
			Component: component,
			Error:     nil,
			Duration:  75 * time.Millisecond,
			CacheHit:  false,
		}

		// Add multiple callbacks to test the callback loop
		var callback1Called, callback2Called, callback3Called bool
		var callbackMutex sync.Mutex

		bp.AddCallback(func(r BuildResult) {
			callbackMutex.Lock()
			callback1Called = true
			callbackMutex.Unlock()
		})

		bp.AddCallback(func(r BuildResult) {
			callbackMutex.Lock()
			callback2Called = true
			callbackMutex.Unlock()
		})

		bp.AddCallback(func(r BuildResult) {
			callbackMutex.Lock()
			callback3Called = true
			callbackMutex.Unlock()
		})

		// Handle the result - this should test the callback iteration
		bp.handleBuildResult(result)

		// Verify all callbacks were called
		callbackMutex.Lock()
		assert.True(t, callback1Called, "First callback should be called")
		assert.True(t, callback2Called, "Second callback should be called")
		assert.True(t, callback3Called, "Third callback should be called")
		callbackMutex.Unlock()
	})
}

// TestUntestedMetricsFunctions tests the 0% coverage functions in metrics
func TestUntestedMetricsFunctions(t *testing.T) {
	t.Run("GetCacheHitRate calculation", func(t *testing.T) {
		metrics := NewBuildMetrics()

		// Initially should return 0 (no builds)
		rate := metrics.GetCacheHitRate()
		assert.Equal(t, 0.0, rate, "Initial cache hit rate should be 0")

		// Record some builds with mix of cache hits and misses
		component := &types.ComponentInfo{Name: "TestComponent"}

		// Cache miss
		metrics.RecordBuild(BuildResult{
			Component: component,
			CacheHit:  false,
			Duration:  100 * time.Millisecond,
		})

		// Cache hit
		metrics.RecordBuild(BuildResult{
			Component: component,
			CacheHit:  true,
			Duration:  10 * time.Millisecond,
		})

		// Another cache hit
		metrics.RecordBuild(BuildResult{
			Component: component,
			CacheHit:  true,
			Duration:  15 * time.Millisecond,
		})

		// Should be 2/3 = 66.67% (as percentage)
		rate = metrics.GetCacheHitRate()
		expected := (2.0 / 3.0) * 100.0
		assert.InDelta(t, expected, rate, 0.1, "Cache hit rate should be calculated correctly")
	})

	t.Run("GetSuccessRate calculation", func(t *testing.T) {
		metrics := NewBuildMetrics()

		// Initially should return 0% (no builds)
		rate := metrics.GetSuccessRate()
		assert.Equal(t, 0.0, rate, "Initial success rate should be 0% (no builds)")

		// Record some builds with mix of success and failure
		component := &types.ComponentInfo{Name: "TestComponent"}

		// Success
		metrics.RecordBuild(BuildResult{
			Component: component,
			Error:     nil,
			Duration:  100 * time.Millisecond,
		})

		// Failure
		metrics.RecordBuild(BuildResult{
			Component: component,
			Error:     fmt.Errorf("build failed"),
			Duration:  50 * time.Millisecond,
		})

		// Success
		metrics.RecordBuild(BuildResult{
			Component: component,
			Error:     nil,
			Duration:  80 * time.Millisecond,
		})

		// Should be 2/3 = 66.67%
		rate = metrics.GetSuccessRate()
		expected := (2.0 / 3.0) * 100.0
		assert.InDelta(t, expected, rate, 0.1, "Success rate should be calculated correctly")
	})
}

// TestUntestedPoolFunctions tests the 0% coverage pool functions
func TestUntestedPoolFunctions(t *testing.T) {
	t.Run("PutBuildTask operation", func(t *testing.T) {
		pools := NewObjectPools()

		// Get a task to put back
		task := pools.GetBuildTask()
		require.NotNil(t, task, "Should get a valid build task")

		// Test putting it back
		pools.PutBuildTask(task)

		// Get another task - it might be the same one (pooled)
		task2 := pools.GetBuildTask()
		require.NotNil(t, task2, "Should get a valid build task after put")
	})

	t.Run("StringBuilder pool operations", func(t *testing.T) {
		pools := NewObjectPools()

		// Get a string builder (returns *[]byte)
		sb := pools.GetStringBuilder()
		require.NotNil(t, sb, "Should get a valid string builder")

		// Append some content
		*sb = append(*sb, []byte("test content")...)
		assert.Equal(t, "test content", string(*sb), "String builder should work correctly")

		// Put it back
		pools.PutStringBuilder(sb)

		// Get another - should be reset
		sb2 := pools.GetStringBuilder()
		require.NotNil(t, sb2, "Should get a valid string builder after put")
		assert.Equal(t, 0, len(*sb2), "String builder should be reset when retrieved from pool")
	})

	t.Run("ErrorSlice pool operations", func(t *testing.T) {
		slicePools := NewSlicePools()

		// Get error slice
		errorSlice := slicePools.GetErrorSlice()
		require.NotNil(t, errorSlice, "Should get a valid error slice")
		assert.Equal(t, 0, len(errorSlice), "Error slice should be empty initially")

		// Use the slice
		errorSlice = append(errorSlice, fmt.Errorf("test error"))
		assert.Equal(t, 1, len(errorSlice), "Should be able to append to error slice")

		// Put it back
		slicePools.PutErrorSlice(errorSlice)

		// Get another - should be reset
		errorSlice2 := slicePools.GetErrorSlice()
		require.NotNil(t, errorSlice2, "Should get a valid error slice after put")
		assert.Equal(t, 0, len(errorSlice2), "Error slice should be reset when retrieved from pool")
	})
}
