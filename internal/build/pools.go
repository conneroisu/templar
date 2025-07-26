package build

import (
	"sync"
	"time"

	"github.com/conneroisu/templar/internal/types"
)

// ObjectPools provides memory pools optimized for actual usage patterns
// Focus on objects that genuinely benefit from pooling.
type ObjectPools struct {
	buildResults  sync.Pool // Pool BuildResults - they're reused frequently
	outputBuffers sync.Pool // Pool for output buffers with right-sized capacity
	cacheEntries  sync.Pool // Pool for cache entries to reduce GC pressure
	astResults    sync.Pool // Pool for AST parsing results
}

// NewObjectPools creates optimized object pools.
func NewObjectPools() *ObjectPools {
	return &ObjectPools{
		buildResults: sync.Pool{
			New: func() interface{} {
				return &BuildResult{}
			},
		},
		outputBuffers: sync.Pool{
			New: func() interface{} {
				// Right-size for typical templ output (2KB typical, with room to grow)
				return make([]byte, 0, 4*1024)
			},
		},
		cacheEntries: sync.Pool{
			New: func() interface{} {
				return &CacheEntry{}
			},
		},
		astResults: sync.Pool{
			New: func() interface{} {
				return &ASTParseResult{}
			},
		},
	}
}

// GetBuildResult gets a BuildResult from the pool with minimal reset overhead.
func (p *ObjectPools) GetBuildResult() *BuildResult {
	result := p.buildResults.Get().(*BuildResult)
	// Reset all fields to match test expectations
	result.Component = nil
	result.Output = nil
	result.Error = nil
	result.Duration = 0
	result.Hash = ""
	result.CacheHit = false
	result.ParsedErrors = result.ParsedErrors[:0] // Keep capacity for efficiency

	return result
}

// PutBuildResult returns a BuildResult to the pool.
func (p *ObjectPools) PutBuildResult(result *BuildResult) {
	if result != nil {
		p.buildResults.Put(result)
	}
}

// GetBuildTask creates a new BuildTask (no pooling for small structs).
func (p *ObjectPools) GetBuildTask() *BuildTask {
	return &BuildTask{}
}

// PutBuildTask is a no-op (no pooling for small structs).
func (p *ObjectPools) PutBuildTask(task *BuildTask) {
	// No-op: Small struct allocation is faster than pooling overhead
}

// GetOutputBuffer gets a right-sized byte slice from the pool.
func (p *ObjectPools) GetOutputBuffer() []byte {
	buffer := p.outputBuffers.Get().([]byte)

	return buffer[:0] // Reset length but keep capacity
}

// PutOutputBuffer returns a byte slice to the pool with size limits.
func (p *ObjectPools) PutOutputBuffer(buffer []byte) {
	if buffer != nil {
		// Only pool buffers with reasonable capacity to prevent memory bloat
		if cap(buffer) >= 1*1024 && cap(buffer) <= 64*1024 { // 1KB-64KB sweet spot
			//nolint:staticcheck // SA6002: intentional slice value for sync.Pool performance
			p.outputBuffers.Put(buffer[:0])
		}
		// Buffers outside this range are just discarded (too small or too large)
	}
}

// GetStringBuilder creates a new string builder buffer (no pooling).
func (p *ObjectPools) GetStringBuilder() *[]byte {
	buffer := make([]byte, 0, 4*1024) // 4KB initial capacity

	return &buffer
}

// PutStringBuilder is a no-op (no pooling for small buffers).
func (p *ObjectPools) PutStringBuilder(buffer *[]byte) {
	// No-op: Small buffer allocation is faster than pooling overhead
}

// GetCacheEntry gets a CacheEntry from the pool.
func (p *ObjectPools) GetCacheEntry() *CacheEntry {
	entry := p.cacheEntries.Get().(*CacheEntry)
	// Reset all fields
	entry.Key = ""
	entry.Value = nil
	entry.Hash = ""
	entry.CreatedAt = time.Time{}
	entry.AccessedAt = time.Time{}
	entry.Size = 0
	entry.ASTData = nil
	entry.Metadata = nil
	entry.prev = nil
	entry.next = nil

	return entry
}

// PutCacheEntry returns a CacheEntry to the pool.
func (p *ObjectPools) PutCacheEntry(entry *CacheEntry) {
	if entry != nil {
		p.cacheEntries.Put(entry)
	}
}

// GetASTResult gets an ASTParseResult from the pool.
func (p *ObjectPools) GetASTResult() *ASTParseResult {
	result := p.astResults.Get().(*ASTParseResult)
	// Reset all fields
	result.Component = nil
	result.Parameters = result.Parameters[:0]     // Keep capacity
	result.Dependencies = result.Dependencies[:0] // Keep capacity
	result.ParseTime = 0
	result.CachedAt = time.Time{}

	return result
}

// PutASTResult returns an ASTParseResult to the pool.
func (p *ObjectPools) PutASTResult(result *ASTParseResult) {
	if result != nil {
		p.astResults.Put(result)
	}
}

// Note: Reset methods removed since we're no longer pooling small structs
// Direct allocation is faster than pooling overhead for small objects

// WorkerPool manages a pool of build workers with their contexts.
type WorkerPool struct {
	workers  sync.Pool
	contexts sync.Pool
}

// NewWorkerPool creates a new worker pool.
func NewWorkerPool() *WorkerPool {
	return &WorkerPool{
		workers: sync.Pool{
			New: func() interface{} {
				return &BuildWorker{}
			},
		},
		contexts: sync.Pool{
			New: func() interface{} {
				return &WorkerContext{}
			},
		},
	}
}

// BuildWorker represents a reusable build worker.
type BuildWorker struct {
	ID      int
	State   WorkerState
	Context *WorkerContext
}

// WorkerState represents the state of a build worker.
type WorkerState int

const (
	WorkerIdle WorkerState = iota
	WorkerBusy
	WorkerStopped
)

// WorkerContext holds the working context for a build worker.
type WorkerContext struct {
	TempDir      string
	OutputBuffer []byte
	ErrorBuffer  []byte
	Environment  map[string]string
}

// GetWorker gets a build worker from the pool with lazy context allocation.
func (wp *WorkerPool) GetWorker() *BuildWorker {
	worker := wp.workers.Get().(*BuildWorker)
	// Minimal reset - avoid allocating context unless needed
	worker.ID = 0
	worker.State = WorkerIdle
	// Context allocated lazily when actually needed
	if worker.Context == nil {
		worker.Context = wp.GetWorkerContext()
	}

	return worker
}

// PutWorker returns a build worker to the pool with minimal cleanup.
func (wp *WorkerPool) PutWorker(worker *BuildWorker) {
	if worker != nil {
		// Don't return context to pool - keep it attached for reuse
		// This avoids the overhead of constantly getting/putting contexts
		if worker.Context != nil {
			worker.Context.Reset()
		}
		wp.workers.Put(worker)
	}
}

// GetWorkerContext gets a worker context from the pool.
func (wp *WorkerPool) GetWorkerContext() *WorkerContext {
	ctx := wp.contexts.Get().(*WorkerContext)
	ctx.Reset()

	return ctx
}

// PutWorkerContext returns a worker context to the pool.
func (wp *WorkerPool) PutWorkerContext(ctx *WorkerContext) {
	if ctx != nil {
		ctx.Reset()
		wp.contexts.Put(ctx)
	}
}

// Reset clears a BuildWorker for reuse.
func (bw *BuildWorker) Reset() {
	bw.ID = 0
	bw.State = WorkerIdle
	bw.Context = nil
}

// Reset clears a WorkerContext for reuse with minimal overhead while maintaining test contract.
func (wc *WorkerContext) Reset() {
	wc.TempDir = ""

	// Reset buffers efficiently - maintain test behavior expectations
	if wc.OutputBuffer != nil && cap(wc.OutputBuffer) <= 1024*1024 { // 1MB max per original test
		wc.OutputBuffer = wc.OutputBuffer[:0] // Keep capacity
	} else {
		wc.OutputBuffer = nil // Test expects nil for oversized buffers
	}

	if wc.ErrorBuffer != nil && cap(wc.ErrorBuffer) <= 64*1024 { // 64KB max per original test
		wc.ErrorBuffer = wc.ErrorBuffer[:0] // Keep capacity
	} else {
		wc.ErrorBuffer = nil // Test expects nil for oversized buffers
	}

	// Fast map clear for small maps, recreate for large maps
	if wc.Environment != nil && len(wc.Environment) <= 10 {
		for k := range wc.Environment {
			delete(wc.Environment, k)
		}
	} else {
		wc.Environment = make(map[string]string, 8)
	}
}

// Pre-sized slice pools for common patterns.
type SlicePools struct {
	componentInfoSlices sync.Pool
	stringSlices        sync.Pool
	errorSlices         sync.Pool
}

// NewSlicePools creates pools for commonly used slices.
func NewSlicePools() *SlicePools {
	return &SlicePools{
		componentInfoSlices: sync.Pool{
			New: func() interface{} {
				// Pre-allocate slice for typical component count
				return make([]*types.ComponentInfo, 0, 50)
			},
		},
		stringSlices: sync.Pool{
			New: func() interface{} {
				// Pre-allocate slice for typical string collections
				return make([]string, 0, 20)
			},
		},
		errorSlices: sync.Pool{
			New: func() interface{} {
				// Pre-allocate slice for error collections
				return make([]error, 0, 10)
			},
		},
	}
}

// GetComponentInfoSlice gets a slice of ComponentInfo pointers from the pool.
func (sp *SlicePools) GetComponentInfoSlice() []*types.ComponentInfo {
	slice := sp.componentInfoSlices.Get().([]*types.ComponentInfo)

	return slice[:0] // Reset length but keep capacity
}

// PutComponentInfoSlice returns a slice to the pool.
func (sp *SlicePools) PutComponentInfoSlice(slice []*types.ComponentInfo) {
	if slice != nil && cap(slice) <= 1000 { // Reasonable limit
		//nolint:staticcheck // SA6002: intentional slice value for sync.Pool performance
		sp.componentInfoSlices.Put(slice[:0])
	}
}

// GetStringSlice gets a string slice from the pool.
func (sp *SlicePools) GetStringSlice() []string {
	slice := sp.stringSlices.Get().([]string)

	return slice[:0] // Reset length but keep capacity
}

// PutStringSlice returns a string slice to the pool.
func (sp *SlicePools) PutStringSlice(slice []string) {
	if slice != nil && cap(slice) <= 200 { // Reasonable limit
		//nolint:staticcheck // SA6002: intentional slice value for sync.Pool performance
		sp.stringSlices.Put(slice[:0])
	}
}

// GetErrorSlice gets an error slice from the pool.
func (sp *SlicePools) GetErrorSlice() []error {
	slice := sp.errorSlices.Get().([]error)

	return slice[:0] // Reset length but keep capacity
}

// PutErrorSlice returns an error slice to the pool.
func (sp *SlicePools) PutErrorSlice(slice []error) {
	if slice != nil && cap(slice) <= 100 { // Reasonable limit
		//nolint:staticcheck // SA6002: intentional slice value for sync.Pool performance
		sp.errorSlices.Put(slice[:0])
	}
}
