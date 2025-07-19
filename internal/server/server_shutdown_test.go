package server

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShutdownRaceCondition(t *testing.T) {
	// Create a test server
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 0, // Use system-assigned port
		},
	}

	server, err := New(cfg)
	require.NoError(t, err)

	// Start the server in a goroutine
	go func() {
		ctx := context.Background()
		server.Start(ctx)
	}()

	// Give the server time to start
	time.Sleep(100 * time.Millisecond)

	// Test concurrent shutdown calls
	var wg sync.WaitGroup
	shutdownResults := make(chan error, 10)

	// Launch multiple concurrent shutdown calls
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			
			err := server.Shutdown(ctx)
			shutdownResults <- err
		}()
	}

	// Wait for all shutdowns to complete
	wg.Wait()
	close(shutdownResults)

	// Verify no panics occurred and all shutdowns completed
	shutdownCount := 0
	for err := range shutdownResults {
		shutdownCount++
		// All shutdown calls should complete without error
		assert.NoError(t, err, "Shutdown should not return error on concurrent calls")
	}

	assert.Equal(t, 10, shutdownCount, "All shutdown calls should complete")
}

func TestShutdownOnce(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 0,
		},
	}

	server, err := New(cfg)
	require.NoError(t, err)

	// First shutdown should work
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	err1 := server.Shutdown(ctx)
	assert.NoError(t, err1)

	// Second shutdown should be safe (no panic)
	err2 := server.Shutdown(ctx)
	assert.NoError(t, err2)

	// Verify server is marked as shutdown
	server.shutdownMutex.RLock()
	isShutdown := server.isShutdown
	server.shutdownMutex.RUnlock()
	
	assert.True(t, isShutdown, "Server should be marked as shutdown")
}

func TestChannelSafeClose(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost", 
			Port: 0,
		},
	}

	server, err := New(cfg)
	require.NoError(t, err)

	// Manually close one of the channels to simulate race condition
	close(server.broadcast)

	// Shutdown should handle already-closed channels gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	err = server.Shutdown(ctx)
	assert.NoError(t, err, "Shutdown should handle pre-closed channels gracefully")
}

func TestBuildPipelineShutdown(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 0,
		},
	}

	server, err := New(cfg)
	require.NoError(t, err)

	// Verify build pipeline exists
	require.NotNil(t, server.buildPipeline, "Build pipeline should be initialized")

	// Start build pipeline
	ctx := context.Background()
	server.buildPipeline.Start(ctx)

	// Shutdown should stop build pipeline
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	err = server.Shutdown(shutdownCtx)
	assert.NoError(t, err)

	// Build pipeline should be stopped (this is verified by the Stop() method not hanging)
}

func TestFileWatcherShutdown(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 0,
		},
	}

	server, err := New(cfg)
	require.NoError(t, err)

	// Verify file watcher exists
	require.NotNil(t, server.watcher, "File watcher should be initialized")

	// Shutdown should stop file watcher
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	err = server.Shutdown(ctx)
	assert.NoError(t, err)

	// File watcher should be stopped (this is verified by the Stop() method not hanging)
}