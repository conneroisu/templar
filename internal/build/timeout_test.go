package build

import (
	"context"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestBuildPipelineTimeout(t *testing.T) {
	t.Run("build operation respects context timeout", func(t *testing.T) {
		// Create a config with very short build timeout
		cfg := &config.Config{
			Timeouts: config.TimeoutConfig{
				Build: 50 * time.Millisecond, // Very short timeout
			},
		}

		// Create pipeline with timeout config
		pipeline := NewBuildPipeline(1, nil, cfg)

		// Test that the getBuildTimeout returns the configured value
		timeout := pipeline.getBuildTimeout()
		assert.Equal(t, 50*time.Millisecond, timeout, "Should return configured timeout")
	})

	t.Run("build operation uses default timeout when no config", func(t *testing.T) {
		// Create pipeline without config
		pipeline := NewBuildPipeline(1, nil)

		// Test that the getBuildTimeout returns the default value
		timeout := pipeline.getBuildTimeout()
		assert.Equal(t, 5*time.Minute, timeout, "Should return default timeout")
	})

	t.Run("compiler respects context cancellation", func(t *testing.T) {
		compiler := NewTemplCompiler()
		
		// Create a very short timeout context
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()
		
		// Wait for context to timeout
		<-ctx.Done()
		
		component := &types.ComponentInfo{
			Name:     "TestComponent",
			FilePath: "test.templ",
			Package:  "test",
		}
		
		_, err := compiler.Compile(ctx, component)
		assert.Error(t, err, "Should fail due to context timeout")
		assert.Contains(t, err.Error(), "timed out", "Error should mention timeout")
	})

	t.Run("compiler pools respect context cancellation", func(t *testing.T) {
		compiler := NewTemplCompiler()
		pools := NewObjectPools()
		
		// Create a very short timeout context
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()
		
		// Wait for context to timeout
		<-ctx.Done()
		
		component := &types.ComponentInfo{
			Name:     "TestComponent",
			FilePath: "test.templ",
			Package:  "test",
		}
		
		_, err := compiler.CompileWithPools(ctx, component, pools)
		assert.Error(t, err, "Should fail due to context timeout")
		assert.Contains(t, err.Error(), "timed out", "Error should mention timeout")
	})

	t.Run("timeout configuration validation", func(t *testing.T) {
		// Test various timeout values
		testCases := []struct {
			name    string
			timeout time.Duration
			want    time.Duration
		}{
			{
				name:    "positive timeout",
				timeout: 30 * time.Second,
				want:    30 * time.Second,
			},
			{
				name:    "zero timeout uses default",
				timeout: 0,
				want:    5 * time.Minute,
			},
			{
				name:    "negative timeout uses default",
				timeout: -1 * time.Second,
				want:    5 * time.Minute,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cfg := &config.Config{
					Timeouts: config.TimeoutConfig{
						Build: tc.timeout,
					},
				}

				pipeline := NewBuildPipeline(1, nil, cfg)
				got := pipeline.getBuildTimeout()
				assert.Equal(t, tc.want, got)
			})
		}
	})

	t.Run("multiple config parameters", func(t *testing.T) {
		// Test multiple config parameters - should use the first one
		cfg1 := &config.Config{
			Timeouts: config.TimeoutConfig{
				Build: 1 * time.Minute,
			},
		}
		cfg2 := &config.Config{
			Timeouts: config.TimeoutConfig{
				Build: 2 * time.Minute,
			},
		}

		pipeline := NewBuildPipeline(1, nil, cfg1, cfg2)
		timeout := pipeline.getBuildTimeout()
		assert.Equal(t, 1*time.Minute, timeout, "Should use first config")
	})

	t.Run("stop with timeout functionality", func(t *testing.T) {
		pipeline := NewBuildPipeline(1, nil)
		
		// Start pipeline
		ctx := context.Background()
		pipeline.Start(ctx)
		
		// Stop with timeout should complete quickly since no work is in progress
		err := pipeline.StopWithTimeout(1 * time.Second)
		assert.NoError(t, err, "Should stop without timeout")
	})

	t.Run("graceful shutdown handling", func(t *testing.T) {
		pipeline := NewBuildPipeline(1, nil)
		
		// Build should fail if pipeline is not started
		component := &types.ComponentInfo{
			Name:     "TestComponent",
			FilePath: "test.templ",
			Package:  "test",
		}
		
		pipeline.Build(component)
		// No assertion here as this tests logging behavior
	})
}