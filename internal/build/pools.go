package build

import (
	"sync"
	
	"github.com/conneroisu/templar/internal/registry"
)

// ObjectPools provides memory pools for frequently allocated objects
type ObjectPools struct {
	buildResults   sync.Pool
	buildTasks     sync.Pool
	outputBuffers  sync.Pool
	stringBuilders sync.Pool
}

// NewObjectPools creates a new set of object pools
func NewObjectPools() *ObjectPools {
	return &ObjectPools{
		buildResults: sync.Pool{
			New: func() interface{} {
				return &BuildResult{}
			},
		},
		buildTasks: sync.Pool{
			New: func() interface{} {
				return &BuildTask{}
			},
		},
		outputBuffers: sync.Pool{
			New: func() interface{} {
				// Pre-allocate 64KB buffer for typical templ output
				return make([]byte, 0, 64*1024)
			},
		},
		stringBuilders: sync.Pool{
			New: func() interface{} {
				// Pre-allocate 4KB for string building operations
				buffer := make([]byte, 0, 4*1024)
				return &buffer
			},
		},
	}
}

// GetBuildResult gets a BuildResult from the pool
func (p *ObjectPools) GetBuildResult() *BuildResult {
	result := p.buildResults.Get().(*BuildResult)
	result.Reset() // Ensure clean state
	return result
}

// PutBuildResult returns a BuildResult to the pool
func (p *ObjectPools) PutBuildResult(result *BuildResult) {
	if result != nil {
		result.Reset() // Clean state before returning to pool
		p.buildResults.Put(result)
	}
}

// GetBuildTask gets a BuildTask from the pool
func (p *ObjectPools) GetBuildTask() *BuildTask {
	task := p.buildTasks.Get().(*BuildTask)
	task.Reset() // Ensure clean state
	return task
}

// PutBuildTask returns a BuildTask to the pool
func (p *ObjectPools) PutBuildTask(task *BuildTask) {
	if task != nil {
		task.Reset() // Clean state before returning to pool
		p.buildTasks.Put(task)
	}
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

// GetStringBuilder gets a string builder buffer from the pool
func (p *ObjectPools) GetStringBuilder() *[]byte {
	buffer := p.stringBuilders.Get().(*[]byte)
	*buffer = (*buffer)[:0] // Reset length but keep capacity
	return buffer
}

// PutStringBuilder returns a string builder buffer to the pool
func (p *ObjectPools) PutStringBuilder(buffer *[]byte) {
	if buffer != nil && cap(*buffer) <= 64*1024 { // 64KB max
		p.stringBuilders.Put(buffer)
	}
}

// Reset methods for pooled objects

// Reset clears a BuildResult for reuse
func (br *BuildResult) Reset() {
	br.Component = nil
	br.Output = nil
	br.Error = nil
	br.ParsedErrors = nil
	br.Duration = 0
	br.CacheHit = false
	br.Hash = ""
}

// Reset clears a BuildTask for reuse
func (bt *BuildTask) Reset() {
	bt.Component = nil
	bt.Priority = 0
	bt.Timestamp = bt.Timestamp.Truncate(0) // Zero time
}

// WorkerPool manages a pool of build workers with their contexts
type WorkerPool struct {
	workers sync.Pool
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
	TempDir    string
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
				return make([]*registry.ComponentInfo, 0, 50)
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
func (sp *SlicePools) GetComponentInfoSlice() []*registry.ComponentInfo {
	slice := sp.componentInfoSlices.Get().([]*registry.ComponentInfo)
	return slice[:0] // Reset length but keep capacity
}

// PutComponentInfoSlice returns a slice to the pool
func (sp *SlicePools) PutComponentInfoSlice(slice []*registry.ComponentInfo) {
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