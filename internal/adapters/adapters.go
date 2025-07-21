// Package adapters provides wrapper types to adapt concrete implementations
// to the interface contracts, handling type conversions where necessary.
package adapters

import (
	"context"

	"github.com/conneroisu/templar/internal/build"
	"github.com/conneroisu/templar/internal/interfaces"
	"github.com/conneroisu/templar/internal/scanner"
	"github.com/conneroisu/templar/internal/types"
	"github.com/conneroisu/templar/internal/watcher"
)

// FileWatcherAdapter wraps a concrete FileWatcher to implement the interface
type FileWatcherAdapter struct {
	fw *watcher.FileWatcher
}

// NewFileWatcherAdapter creates a new adapter for FileWatcher
func NewFileWatcherAdapter(fw *watcher.FileWatcher) interfaces.FileWatcher {
	return &FileWatcherAdapter{fw: fw}
}

func (a *FileWatcherAdapter) AddPath(path string) error {
	return a.fw.AddPath(path)
}

func (a *FileWatcherAdapter) Start(ctx context.Context) error {
	return a.fw.Start(ctx)
}

func (a *FileWatcherAdapter) Stop() error {
	return a.fw.Stop()
}

func (a *FileWatcherAdapter) AddFilter(filter interfaces.FileFilter) {
	// Convert interface filter to concrete filter
	a.fw.AddFilter(watcher.FileFilter(func(path string) bool {
		return filter.ShouldInclude(path)
	}))
}

func (a *FileWatcherAdapter) AddHandler(handler interfaces.ChangeHandlerFunc) {
	// Convert interface handler to concrete handler
	a.fw.AddHandler(func(events []watcher.ChangeEvent) error {
		// Convert concrete events to interface
		interfaceEvents := make([]interface{}, len(events))
		for i, event := range events {
			interfaceEvents[i] = event
		}
		return handler(interfaceEvents)
	})
}

func (a *FileWatcherAdapter) AddRecursive(root string) error {
	return a.fw.AddRecursive(root)
}

// ComponentScannerAdapter wraps a concrete ComponentScanner to implement the interface
type ComponentScannerAdapter struct {
	cs *scanner.ComponentScanner
}

// NewComponentScannerAdapter creates a new adapter for ComponentScanner
func NewComponentScannerAdapter(cs *scanner.ComponentScanner) interfaces.ComponentScanner {
	return &ComponentScannerAdapter{cs: cs}
}

func (a *ComponentScannerAdapter) ScanDirectory(dir string) error {
	return a.cs.ScanDirectory(dir)
}

func (a *ComponentScannerAdapter) ScanDirectoryParallel(dir string, workers int) error {
	return a.cs.ScanDirectoryParallel(dir, workers)
}

func (a *ComponentScannerAdapter) ScanFile(path string) error {
	return a.cs.ScanFile(path)
}

func (a *ComponentScannerAdapter) GetRegistry() interfaces.ComponentRegistry {
	// Return the concrete registry directly - it already implements the interface
	return a.cs.GetRegistry()
}

// BuildPipelineAdapter wraps a concrete BuildPipeline to implement the interface
type BuildPipelineAdapter struct {
	bp *build.BuildPipeline
}

// NewBuildPipelineAdapter creates a new adapter for BuildPipeline
func NewBuildPipelineAdapter(bp *build.BuildPipeline) interfaces.BuildPipeline {
	return &BuildPipelineAdapter{bp: bp}
}

func (a *BuildPipelineAdapter) Build(component *types.ComponentInfo) error {
	// Convert to the concrete method call
	a.bp.Build(component)
	return nil // The concrete method doesn't return an error
}

func (a *BuildPipelineAdapter) Start(ctx context.Context) error {
	a.bp.Start(ctx)
	return nil // The concrete method doesn't return an error
}

func (a *BuildPipelineAdapter) Stop() error {
	a.bp.Stop()
	return nil // The concrete method doesn't return an error
}

func (a *BuildPipelineAdapter) AddCallback(callback interfaces.BuildCallbackFunc) {
	// Convert interface callback to concrete callback
	a.bp.AddCallback(func(result build.BuildResult) {
		callback(result)
	})
}

func (a *BuildPipelineAdapter) BuildWithPriority(component *types.ComponentInfo) {
	a.bp.BuildWithPriority(component)
}

func (a *BuildPipelineAdapter) GetMetrics() interface{} {
	return a.bp.GetMetrics()
}

func (a *BuildPipelineAdapter) GetCache() interface{} {
	// Return cache information from the pipeline
	hits, size, entries := a.bp.GetCacheStats()
	return map[string]interface{}{
		"hits":    hits,
		"size":    size,
		"entries": entries,
	}
}

func (a *BuildPipelineAdapter) ClearCache() {
	a.bp.ClearCache()
}
