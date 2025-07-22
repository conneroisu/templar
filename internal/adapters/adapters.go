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
	bp *build.RefactoredBuildPipeline
}

// NewBuildPipelineAdapter creates a new adapter for BuildPipeline
func NewBuildPipelineAdapter(bp *build.RefactoredBuildPipeline) interfaces.BuildPipeline {
	return &BuildPipelineAdapter{bp: bp}
}

func (a *BuildPipelineAdapter) Build(component *types.ComponentInfo) error {
	return a.bp.Build(component)
}

func (a *BuildPipelineAdapter) Start(ctx context.Context) error {
	return a.bp.Start(ctx)
}

func (a *BuildPipelineAdapter) Stop() error {
	return a.bp.Stop()
}

func (a *BuildPipelineAdapter) AddCallback(callback interfaces.BuildCallbackFunc) {
	a.bp.AddCallback(callback)
}

func (a *BuildPipelineAdapter) BuildWithPriority(component *types.ComponentInfo) {
	a.bp.BuildWithPriority(component)
}

func (a *BuildPipelineAdapter) GetMetrics() interfaces.BuildMetrics {
	return a.bp.GetMetrics()
}

func (a *BuildPipelineAdapter) GetCache() interfaces.CacheStats {
	return a.bp.GetCache()
}

func (a *BuildPipelineAdapter) ClearCache() {
	// RefactoredBuildPipeline doesn't have ClearCache, use cache directly
	cache := a.bp.GetCache()
	cache.Clear()
}
