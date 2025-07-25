// Package build provides result processing and callback management for build operations.
//
// ResultProcessor implements efficient result handling with callback management,
// error processing, and metrics tracking. It provides a clean separation between
// build execution and result handling concerns.
package build

import (
	"context"
	"fmt"
	"sync"

	"github.com/conneroisu/templar/internal/errors"
	"github.com/conneroisu/templar/internal/interfaces"
)

// ResultProcessor handles build result processing and callback management.
// It provides thread-safe callback registration, result distribution,
// and error handling with comprehensive logging and metrics.
type ResultProcessor struct {
	// callbacks receive build status updates for UI integration
	callbacks []BuildCallback
	// metrics tracks result processing performance
	metrics *BuildMetrics
	// errorParser processes build errors for enhanced reporting
	errorParser *errors.ErrorParser
	// mu protects concurrent access to callbacks
	mu sync.RWMutex
	// resultWg synchronizes result processing
	resultWg sync.WaitGroup
	// cancel terminates result processing gracefully
	cancel context.CancelFunc
	// stopped indicates if the processor has been shut down
	stopped bool
}

// NewResultProcessor creates a new result processor with the specified components.
func NewResultProcessor(metrics *BuildMetrics, errorParser *errors.ErrorParser) *ResultProcessor {
	return &ResultProcessor{
		callbacks:   make([]BuildCallback, 0),
		metrics:     metrics,
		errorParser: errorParser,
		stopped:     false,
	}
}

// ProcessResults processes results from the given channel until the context is cancelled.
// This method should be called in a goroutine to handle results asynchronously.
func (rp *ResultProcessor) ProcessResults(ctx context.Context, results <-chan interface{}) {
	// Create cancellable context for result processing
	ctx, rp.cancel = context.WithCancel(ctx)
	
	rp.resultWg.Add(1)
	go func() {
		defer rp.resultWg.Done()
		
		for {
			select {
			case <-ctx.Done():
				return
			case result, ok := <-results:
				if !ok {
					return // Results channel closed
				}
				
				buildResult, ok := result.(BuildResult)
				if !ok {
					// Log invalid result type - could add custom metric tracking if needed
					// Note: IncrementInvalidResults would need to be added to BuildMetrics
					continue
				}
				
				rp.handleBuildResult(buildResult)
			}
		}
	}()
}

// AddCallback registers a callback for build completion events.
// Callbacks are called synchronously for each build result.
func (rp *ResultProcessor) AddCallback(callback interfaces.BuildCallbackFunc) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	
	// Wrap the interface callback in our concrete type
	buildCallback := BuildCallback(func(result BuildResult) {
		callback(result)
	})
	rp.callbacks = append(rp.callbacks, buildCallback)
}

// Stop gracefully shuts down result processing and waits for completion.
func (rp *ResultProcessor) Stop() {
	rp.mu.Lock()
	if rp.stopped {
		rp.mu.Unlock()
		return
	}
	rp.stopped = true
	rp.mu.Unlock()
	
	if rp.cancel != nil {
		rp.cancel()
	}
	
	// Wait for result processing to complete
	rp.resultWg.Wait()
}

// handleBuildResult processes a single build result and invokes callbacks.
func (rp *ResultProcessor) handleBuildResult(result BuildResult) {
	// Update metrics
	if rp.metrics != nil {
		rp.metrics.RecordBuild(result)
	}
	
	// Enhanced error reporting
	if result.Error != nil && len(result.ParsedErrors) > 0 {
		rp.handleBuildErrors(result)
	}
	
	// Invoke all registered callbacks
	rp.invokeCallbacks(result)
}

// handleBuildErrors processes build errors for enhanced reporting and debugging.
func (rp *ResultProcessor) handleBuildErrors(result BuildResult) {
	// Log parsed errors for debugging
	if len(result.ParsedErrors) > 0 {
		fmt.Println("Parsed errors:")
		for _, err := range result.ParsedErrors {
			fmt.Print(err.FormatError())
		}
	}
	
	// Update error metrics
	// Note: Error type tracking could be added to BuildMetrics if needed
}

// invokeCallbacks calls all registered callbacks with the build result.
func (rp *ResultProcessor) invokeCallbacks(result BuildResult) {
	rp.mu.RLock()
	callbacks := make([]BuildCallback, len(rp.callbacks))
	copy(callbacks, rp.callbacks)
	rp.mu.RUnlock()
	
	// Call callbacks without holding the lock to avoid deadlocks
	for _, callback := range callbacks {
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Handle callback panics gracefully
					// Note: Callback error tracking could be added to BuildMetrics if needed
					fmt.Printf("Callback panic recovered: %v\n", r)
				}
			}()
			
			callback(result)
		}()
	}
}

// RemoveCallback removes a specific callback from the processor.
// This is useful for cleanup when callbacks are no longer needed.
func (rp *ResultProcessor) RemoveCallback(targetCallback BuildCallback) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	
	for i, callback := range rp.callbacks {
		// Note: Function comparison in Go is limited, this is a simplified approach
		// In practice, you might want to use callback IDs or a different mechanism
		if fmt.Sprintf("%p", callback) == fmt.Sprintf("%p", targetCallback) {
			rp.callbacks = append(rp.callbacks[:i], rp.callbacks[i+1:]...)
			break
		}
	}
}

// ClearCallbacks removes all registered callbacks.
func (rp *ResultProcessor) ClearCallbacks() {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	
	rp.callbacks = rp.callbacks[:0]
}

// GetCallbackCount returns the number of registered callbacks.
func (rp *ResultProcessor) GetCallbackCount() int {
	rp.mu.RLock()
	defer rp.mu.RUnlock()
	
	return len(rp.callbacks)
}

// GetProcessorStats returns current result processor statistics.
func (rp *ResultProcessor) GetProcessorStats() ProcessorStats {
	rp.mu.RLock()
	defer rp.mu.RUnlock()
	
	return ProcessorStats{
		CallbackCount: len(rp.callbacks),
		Stopped:       rp.stopped,
	}
}

// ProcessorStats provides result processor performance metrics.
type ProcessorStats struct {
	CallbackCount int
	Stopped       bool
}

// Verify that ResultProcessor implements the ResultProcessor interface
var _ interfaces.ResultProcessor = (*ResultProcessor)(nil)