package build

import (
	"sync"

	"github.com/conneroisu/templar/internal/types"
)

// ObjectPools provides memory pools for large objects only
// Small struct allocations are faster without pooling
type ObjectPools struct {
	outputBuffers sync.Pool // Only pool large buffers
}

// NewObjectPools creates a new set of object pools
func NewObjectPools() *ObjectPools {
	return &ObjectPools{
		outputBuffers: sync.Pool{
			New: func() interface{} {
				// Pre-allocate 64KB buffer for typical templ output
				return make([]byte, 0, 64*1024)
			},
		},
	}
}

// GetBuildResult creates a new BuildResult (no pooling for small structs)
func (p *ObjectPools) GetBuildResult() *BuildResult {
	return &BuildResult{}
}

// PutBuildResult is a no-op (no pooling for small structs)
func (p *ObjectPools) PutBuildResult(result *BuildResult) {
	// No-op: Small struct allocation is faster than pooling overhead
}

// GetBuildTask creates a new BuildTask (no pooling for small structs)
func (p *ObjectPools) GetBuildTask() *BuildTask {
	return &BuildTask{}
}

// PutBuildTask is a no-op (no pooling for small structs)
func (p *ObjectPools) PutBuildTask(task *BuildTask) {
	// No-op: Small struct allocation is faster than pooling overhead
}

// GetOutputBuffer gets a byte slice from the pool
func (p *ObjectPools) GetOutputBuffer() []byte {
	buffer := p.outputBuffers.Get().([]byte)
	return buffer[:0] // Reset length but keep capacity
}

// PutOutputBuffer returns a byte slice to the pool
func (p *ObjectPools) PutOutputBuffer(buffer []byte) {
	if buffer != nil {
		// Only return to pool if capacity is reasonable (avoid memory bloat)
		if cap(buffer) <= 1024*1024 { // 1MB max
			p.outputBuffers.Put(buffer[:0])
		}
	}
}

// GetStringBuilder creates a new string builder buffer (no pooling)
func (p *ObjectPools) GetStringBuilder() *[]byte {
	buffer := make([]byte, 0, 4*1024) // 4KB initial capacity
	return &buffer
}

// PutStringBuilder is a no-op (no pooling for small buffers)
func (p *ObjectPools) PutStringBuilder(buffer *[]byte) {
	// No-op: Small buffer allocation is faster than pooling overhead
}

// Note: Reset methods removed since we're no longer pooling small structs
// Direct allocation is faster than pooling overhead for small objects

// WorkerPool manages a pool of build workers with their contexts
type WorkerPool struct {
	workers  sync.Pool
	contexts sync.Pool
}

// NewWorkerPool creates a new worker pool
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

// BuildWorker represents a reusable build worker
type BuildWorker struct {
	ID      int
	State   WorkerState
	Context *WorkerContext
}

// WorkerState represents the state of a build worker
type WorkerState int

const (
	WorkerIdle WorkerState = iota
	WorkerBusy
	WorkerStopped
)

// WorkerContext holds the working context for a build worker
type WorkerContext struct {
	TempDir      string
	OutputBuffer []byte
	ErrorBuffer  []byte
	Environment  map[string]string
}

// GetWorker gets a build worker from the pool
func (wp *WorkerPool) GetWorker() *BuildWorker {
	worker := wp.workers.Get().(*BuildWorker)
	worker.Reset()
	worker.Context = wp.GetWorkerContext()
	return worker
}

// PutWorker returns a build worker to the pool
func (wp *WorkerPool) PutWorker(worker *BuildWorker) {
	if worker != nil {
		if worker.Context != nil {
			wp.PutWorkerContext(worker.Context)
			worker.Context = nil
		}
		worker.Reset()
		wp.workers.Put(worker)
	}
}

// GetWorkerContext gets a worker context from the pool
func (wp *WorkerPool) GetWorkerContext() *WorkerContext {
	ctx := wp.contexts.Get().(*WorkerContext)
	ctx.Reset()
	return ctx
}

// PutWorkerContext returns a worker context to the pool
func (wp *WorkerPool) PutWorkerContext(ctx *WorkerContext) {
	if ctx != nil {
		ctx.Reset()
		wp.contexts.Put(ctx)
	}
}

// Reset clears a BuildWorker for reuse
func (bw *BuildWorker) Reset() {
	bw.ID = 0
	bw.State = WorkerIdle
	bw.Context = nil
}

// Reset clears a WorkerContext for reuse
func (wc *WorkerContext) Reset() {
	wc.TempDir = ""
	if cap(wc.OutputBuffer) <= 1024*1024 { // 1MB max
		wc.OutputBuffer = wc.OutputBuffer[:0]
	} else {
		wc.OutputBuffer = nil
	}
	if cap(wc.ErrorBuffer) <= 64*1024 { // 64KB max
		wc.ErrorBuffer = wc.ErrorBuffer[:0]
	} else {
		wc.ErrorBuffer = nil
	}
	// Clear environment map but reuse it
	for k := range wc.Environment {
		delete(wc.Environment, k)
	}
	if wc.Environment == nil {
		wc.Environment = make(map[string]string, 8) // Pre-size for common env vars
	}
}

// Pre-sized slice pools for common patterns
type SlicePools struct {
	componentInfoSlices sync.Pool
	stringSlices        sync.Pool
	errorSlices         sync.Pool
}

// NewSlicePools creates pools for commonly used slices
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

// GetComponentInfoSlice gets a slice of ComponentInfo pointers from the pool
func (sp *SlicePools) GetComponentInfoSlice() []*types.ComponentInfo {
	slice := sp.componentInfoSlices.Get().([]*types.ComponentInfo)
	return slice[:0] // Reset length but keep capacity
}

// PutComponentInfoSlice returns a slice to the pool
func (sp *SlicePools) PutComponentInfoSlice(slice []*types.ComponentInfo) {
	if slice != nil && cap(slice) <= 1000 { // Reasonable limit
		sp.componentInfoSlices.Put(slice[:0])
	}
}

// GetStringSlice gets a string slice from the pool
func (sp *SlicePools) GetStringSlice() []string {
	slice := sp.stringSlices.Get().([]string)
	return slice[:0] // Reset length but keep capacity
}

// PutStringSlice returns a string slice to the pool
func (sp *SlicePools) PutStringSlice(slice []string) {
	if slice != nil && cap(slice) <= 200 { // Reasonable limit
		sp.stringSlices.Put(slice[:0])
	}
}

// GetErrorSlice gets an error slice from the pool
func (sp *SlicePools) GetErrorSlice() []error {
	slice := sp.errorSlices.Get().([]error)
	return slice[:0] // Reset length but keep capacity
}

// PutErrorSlice returns an error slice to the pool
func (sp *SlicePools) PutErrorSlice(slice []error) {
	if slice != nil && cap(slice) <= 100 { // Reasonable limit
		sp.errorSlices.Put(slice[:0])
	}
}
