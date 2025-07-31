package server

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/conneroisu/templar/internal/build"
	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/interfaces"
	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/scanner"
	"github.com/conneroisu/templar/internal/types"
	"github.com/conneroisu/templar/internal/watcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHotReloadRaceConditions tests the hot reload flow for race conditions.
func TestHotReloadRaceConditions(t *testing.T) {
	// Create test components
	reg := registry.NewComponentRegistry()
	componentScanner := scanner.NewComponentScanner(reg)
	fileWatcher, err := watcher.NewFileWatcher(50 * time.Millisecond)
	require.NoError(t, err)
	defer func() { _ = fileWatcher.Stop() }()

	buildPipeline := build.NewRefactoredBuildPipeline(2, reg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	require.NoError(t, buildPipeline.Start(ctx))
	defer func() { _ = buildPipeline.Stop() }()

	// Create service orchestrator
	cfg := &config.Config{
		Components: config.ComponentsConfig{
			ScanPaths: []string{"./test_components"},
		},
		Server: config.ServerConfig{
			Environment: "test",
		},
	}

	orchestrator := &ServiceOrchestrator{
		config:        cfg,
		registry:      reg,
		scanner:       componentScanner,
		fileWatcher:   fileWatcher,
		buildPipeline: buildPipeline,
		ctx:           ctx,
		cancel:        cancel,
	}

	// Test concurrent file changes
	t.Run("concurrent_file_changes", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 10
		numEventsPerGoroutine := 5

		// Add test component to registry
		testComponent := &types.ComponentInfo{
			Name:     "TestComponent",
			FilePath: "test.templ",
			Package:  "test",
		}
		reg.Register(testComponent)

		for i := range numGoroutines {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				for range numEventsPerGoroutine {
					events := []watcher.ChangeEvent{
						{
							Path: "test.templ",
							Type: interfaces.EventTypeModified,
						},
					}
					// This should not cause race conditions
					err := orchestrator.handleFileChange(events)
					assert.NoError(t, err)
				}
			}(i)
		}

		wg.Wait()
		t.Log("Concurrent file change handling completed without race conditions")
	})

	// Test concurrent build result handling
	t.Run("concurrent_build_results", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 10

		for i := range numGoroutines {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				testComponent := &types.ComponentInfo{
					Name:     "TestComponent",
					FilePath: "test.templ",
					Package:  "test",
				}

				result := build.BuildResult{
					Component: testComponent,
					Error:     nil,
				}

				// This should not cause race conditions
				orchestrator.handleBuildResult(result)
			}(i)
		}

		wg.Wait()
		t.Log("Concurrent build result handling completed without race conditions")
	})

	// Test concurrent registry access during hot reload
	t.Run("concurrent_registry_access", func(t *testing.T) {
		var wg sync.WaitGroup
		numReaders := 5
		numWriters := 3

		// Readers
		for i := range numReaders {
			wg.Add(1)
			go func(readerID int) {
				defer wg.Done()
				for range 10 {
					components := reg.GetAll()
					_ = components // Use the result to prevent optimization
					time.Sleep(1 * time.Millisecond)
				}
			}(i)
		}

		// Writers
		for i := range numWriters {
			wg.Add(1)
			go func(writerID int) {
				defer wg.Done()
				for range 10 {
					component := &types.ComponentInfo{
						Name:     "ConcurrentComponent",
						FilePath: "concurrent.templ",
						Package:  "test",
					}
					reg.Register(component)
					time.Sleep(1 * time.Millisecond)
				}
			}(i)
		}

		wg.Wait()
		t.Log("Concurrent registry access completed without race conditions")
	})
}

// TestWebSocketBroadcastRaceConditions tests WebSocket broadcasting for race conditions.
func TestWebSocketBroadcastRaceConditions(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:        "localhost",
			Port:        8080,
			Environment: "development",
		},
	}

	server := &PreviewServer{
		config:    cfg,
		clients:   make(map[*websocket.Conn]*Client),
		broadcast: make(chan []byte, 100), // Buffered to prevent blocking
		registry:  registry.NewComponentRegistry(),
	}

	// Test concurrent broadcast attempts
	t.Run("concurrent_broadcasts", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 10
		numBroadcasts := 5

		for i := range numGoroutines {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				for range numBroadcasts {
					// Use select with default to prevent blocking on full channel
					select {
					case server.broadcast <- []byte("test"):
					default:
						// Channel full, skip this broadcast
					}
				}
			}(i)
		}

		// Drain the broadcast channel concurrently
		done := make(chan bool)
		go func() {
			defer close(done)
			for {
				select {
				case <-server.broadcast:
					// Consume broadcast messages
				case <-time.After(100 * time.Millisecond):
					return
				}
			}
		}()

		wg.Wait()
		<-done
		t.Log("Concurrent WebSocket broadcasts completed without race conditions")
	})
}

// TestBuildPipelineRaceConditions tests build pipeline operations for race conditions.
func TestBuildPipelineRaceConditions(t *testing.T) {
	reg := registry.NewComponentRegistry()
	buildPipeline := build.NewRefactoredBuildPipeline(4, reg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	require.NoError(t, buildPipeline.Start(ctx))
	defer func() { _ = buildPipeline.Stop() }()

	// Test concurrent build requests
	t.Run("concurrent_builds", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 8

		for i := range numGoroutines {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				component := &types.ComponentInfo{
					Name:     "ConcurrentBuildComponent",
					FilePath: "concurrent_build.templ",
					Package:  "test",
				}

				// Register component first
				reg.Register(component)

				// Attempt concurrent builds
				for range 3 {
					err := buildPipeline.Build(component)
					// Build may fail due to missing actual file, but shouldn't race
					_ = err // Ignore error for race condition testing
				}
			}(i)
		}

		wg.Wait()
		t.Log("Concurrent builds completed without race conditions")
	})
}
