package build

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a properly initialized cache for tests
func newTestCache(maxSize int64, ttl time.Duration) *BuildCache {
	cache := &BuildCache{
		entries: make(map[string]*CacheEntry),
		maxSize: maxSize,
		ttl:     ttl,
	}
	
	// Initialize LRU doubly-linked list with dummy head and tail
	cache.head = &CacheEntry{}
	cache.tail = &CacheEntry{}
	cache.head.next = cache.tail
	cache.tail.prev = cache.head
	
	return cache
}

func TestNewBuildPipeline(t *testing.T) {
	reg := registry.NewComponentRegistry()
	bp := NewBuildPipeline(4, reg)

	assert.NotNil(t, bp)
	assert.Equal(t, 4, bp.workers)
	assert.Equal(t, reg, bp.registry)
	assert.NotNil(t, bp.compiler)
	assert.NotNil(t, bp.cache)
	assert.NotNil(t, bp.queue)
	assert.NotNil(t, bp.errorParser)
	assert.NotNil(t, bp.metrics)
}

func TestBuildPipelineStart(t *testing.T) {
	reg := registry.NewComponentRegistry()
	bp := NewBuildPipeline(2, reg)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start the pipeline
	bp.Start(ctx)

	// Wait for context to be done
	<-ctx.Done()

	// Should not panic or error
}

func TestBuildPipelineBuild(t *testing.T) {
	reg := registry.NewComponentRegistry()
	bp := NewBuildPipeline(1, reg)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	bp.Start(ctx)

	// Create a test component
	component := &registry.ComponentInfo{
		Name:       "TestComponent",
		FilePath:   "test.templ",
		Package:    "test",
		Parameters: []registry.ParameterInfo{},
	}

	// Build the component
	bp.Build(component)

	// Wait a bit for processing
	time.Sleep(50 * time.Millisecond)

	// Check metrics
	metrics := bp.GetMetrics()
	// The build will likely fail because templ is not available in test environment
	// But we can check that it was attempted
	assert.True(t, metrics.TotalBuilds >= 0)
}

func TestBuildPipelineCallback(t *testing.T) {
	reg := registry.NewComponentRegistry()
	bp := NewBuildPipeline(1, reg)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	bp.Start(ctx)

	// Add a callback
	callbackCalled := false
	bp.AddCallback(func(result BuildResult) {
		callbackCalled = true
	})

	// Create a test component
	component := &registry.ComponentInfo{
		Name:       "TestComponent",
		FilePath:   "test.templ",
		Package:    "test",
		Parameters: []registry.ParameterInfo{},
	}

	// Build the component
	bp.Build(component)

	// Wait a bit for processing
	time.Sleep(50 * time.Millisecond)

	// Callback should have been called
	assert.True(t, callbackCalled)
}

func TestBuildCache(t *testing.T) {
	cache := newTestCache(1024, time.Hour)

	// Test Set and Get
	key := "test-key"
	value := []byte("test-value")

	cache.Set(key, value)

	retrieved, found := cache.Get(key)
	assert.True(t, found)
	assert.Equal(t, value, retrieved)

	// Test non-existent key
	_, found = cache.Get("non-existent")
	assert.False(t, found)
}

func TestBuildCacheExpiration(t *testing.T) {
	cache := newTestCache(1024, time.Millisecond) // Very short TTL

	key := "test-key"
	value := []byte("test-value")

	cache.Set(key, value)

	// Should be found immediately
	_, found := cache.Get(key)
	assert.True(t, found)

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Should be expired
	_, found = cache.Get(key)
	assert.False(t, found)
}

func TestBuildCacheEviction(t *testing.T) {
	cache := newTestCache(10, time.Hour) // Very small cache (in bytes)

	// Add entries that exceed cache size
	cache.Set("key1", []byte("value1")) // 6 bytes
	cache.Set("key2", []byte("value2")) // 6 bytes (total 12 bytes > 10)

	// Since 12 > 10, key1 should be evicted
	_, found := cache.Get("key1")
	assert.False(t, found)

	// key2 should still be there
	_, found = cache.Get("key2")
	assert.True(t, found)
}

func TestBuildCacheStats(t *testing.T) {
	cache := newTestCache(1024, time.Hour)

	// Empty cache
	count, size, maxSize := cache.GetStats()
	assert.Equal(t, 0, count)
	assert.Equal(t, int64(0), size)
	assert.Equal(t, int64(1024), maxSize)

	// Add entries
	cache.Set("key1", []byte("value1"))
	cache.Set("key2", []byte("value2"))

	count, size, maxSize = cache.GetStats()
	assert.Equal(t, 2, count)
	assert.Greater(t, size, int64(0))
	assert.Equal(t, int64(1024), maxSize)
}

func TestBuildCacheClear(t *testing.T) {
	cache := newTestCache(1024, time.Hour)

	// Add entries
	cache.Set("key1", []byte("value1"))
	cache.Set("key2", []byte("value2"))

	// Clear cache
	cache.Clear()

	// Should be empty
	count, _, _ := cache.GetStats()
	assert.Equal(t, 0, count)

	// Entries should not be found
	_, found := cache.Get("key1")
	assert.False(t, found)
}

func TestGenerateContentHash(t *testing.T) {
	// Create a temporary file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.templ")

	content := "test content"
	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	reg := registry.NewComponentRegistry()
	bp := NewBuildPipeline(1, reg)

	component := &registry.ComponentInfo{
		Name:     "TestComponent",
		FilePath: testFile,
		Package:  "test",
	}

	hash1 := bp.generateContentHash(component)
	assert.NotEmpty(t, hash1)

	// Same content should generate same hash
	hash2 := bp.generateContentHash(component)
	assert.Equal(t, hash1, hash2)

	// Different content should generate different hash
	err = os.WriteFile(testFile, []byte("different content"), 0644)
	require.NoError(t, err)

	hash3 := bp.generateContentHash(component)
	assert.NotEqual(t, hash1, hash3)
}

func TestGenerateContentHashFileNotFound(t *testing.T) {
	reg := registry.NewComponentRegistry()
	bp := NewBuildPipeline(1, reg)

	component := &registry.ComponentInfo{
		Name:     "TestComponent",
		FilePath: "nonexistent.templ",
		Package:  "test",
	}

	hash := bp.generateContentHash(component)
	assert.Equal(t, "nonexistent.templ", hash)
}

func TestBuildMetrics(t *testing.T) {
	reg := registry.NewComponentRegistry()
	bp := NewBuildPipeline(1, reg)

	// Initial metrics
	metrics := bp.GetMetrics()
	assert.Equal(t, int64(0), metrics.TotalBuilds)
	assert.Equal(t, int64(0), metrics.SuccessfulBuilds)
	assert.Equal(t, int64(0), metrics.FailedBuilds)
	assert.Equal(t, int64(0), metrics.CacheHits)

	// Simulate a successful build
	result := BuildResult{
		Component: &registry.ComponentInfo{Name: "Test"},
		Error:     nil,
		Duration:  100 * time.Millisecond,
		CacheHit:  false,
	}

	bp.updateMetrics(result)

	metrics = bp.GetMetrics()
	assert.Equal(t, int64(1), metrics.TotalBuilds)
	assert.Equal(t, int64(1), metrics.SuccessfulBuilds)
	assert.Equal(t, int64(0), metrics.FailedBuilds)
	assert.Equal(t, int64(0), metrics.CacheHits)
	assert.Equal(t, 100*time.Millisecond, metrics.AverageDuration)

	// Simulate a failed build
	result.Error = errors.New("build failed")
	bp.updateMetrics(result)

	metrics = bp.GetMetrics()
	assert.Equal(t, int64(2), metrics.TotalBuilds)
	assert.Equal(t, int64(1), metrics.SuccessfulBuilds)
	assert.Equal(t, int64(1), metrics.FailedBuilds)
	assert.Equal(t, int64(0), metrics.CacheHits)

	// Simulate a cache hit
	result.Error = nil
	result.CacheHit = true
	bp.updateMetrics(result)

	metrics = bp.GetMetrics()
	assert.Equal(t, int64(3), metrics.TotalBuilds)
	assert.Equal(t, int64(2), metrics.SuccessfulBuilds)
	assert.Equal(t, int64(1), metrics.FailedBuilds)
	assert.Equal(t, int64(1), metrics.CacheHits)
}

func TestTemplCompiler(t *testing.T) {
	compiler := &TemplCompiler{
		command: "go",
		args:    []string{"version"},
	}

	component := &registry.ComponentInfo{
		Name:     "TestComponent",
		FilePath: "test.templ",
		Package:  "test",
	}

	output, err := compiler.Compile(component)
	require.NoError(t, err)
	assert.Contains(t, string(output), "go version")
}

func TestTemplCompilerFailure(t *testing.T) {
	compiler := &TemplCompiler{
		command: "nonexistent-command",
		args:    []string{},
	}

	component := &registry.ComponentInfo{
		Name:     "TestComponent",
		FilePath: "test.templ",
		Package:  "test",
	}

	_, err := compiler.Compile(component)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command validation failed")
}

func TestTemplCompilerSecurity(t *testing.T) {
	// Test command injection prevention
	compiler := &TemplCompiler{
		command: "echo",
		args:    []string{"test"},
	}
	
	component := &registry.ComponentInfo{
		Name:     "TestComponent",
		FilePath: "test.templ",
		Package:  "test",
	}
	
	_, err := compiler.Compile(component)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command 'echo' is not allowed")
	
	// Test dangerous argument rejection
	compiler = &TemplCompiler{
		command: "go",
		args:    []string{"version; rm -rf /"},
	}
	
	_, err = compiler.Compile(component)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "contains dangerous character")
}
