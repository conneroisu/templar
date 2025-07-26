package scanner

import (
	"context"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/registry"
	"github.com/stretchr/testify/assert"
)

func TestScannerTimeout(t *testing.T) {
	t.Run("scanner respects configured file scan timeout", func(t *testing.T) {
		// Create a config with very short file scan timeout
		cfg := &config.Config{
			Timeouts: config.TimeoutConfig{
				FileScan: 100 * time.Millisecond, // Very short timeout
			},
		}

		// Create registry
		reg := registry.NewComponentRegistry()

		// Create scanner with timeout config
		scanner := NewComponentScanner(reg, cfg)

		// Test that the getFileScanTimeout returns the configured value
		timeout := scanner.getFileScanTimeout()
		assert.Equal(t, 100*time.Millisecond, timeout, "Should return configured timeout")
	})

	t.Run("scanner uses default timeout when no config", func(t *testing.T) {
		// Create registry
		reg := registry.NewComponentRegistry()

		// Create scanner without config
		scanner := NewComponentScanner(reg)

		// Test that the getFileScanTimeout returns the default value
		timeout := scanner.getFileScanTimeout()
		assert.Equal(t, 30*time.Second, timeout, "Should return default timeout")
	})

	t.Run("scanner respects context cancellation in directory scan", func(t *testing.T) {
		// Create registry
		reg := registry.NewComponentRegistry()

		// Create scanner
		scanner := NewComponentScanner(reg)

		// Create a very short timeout context
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		// Wait for context to timeout
		<-ctx.Done()

		// Test scanning with cancelled context
		err := scanner.ScanDirectoryWithContext(ctx, ".")
		assert.Error(t, err, "Should fail due to context timeout")
		assert.Contains(
			t,
			err.Error(),
			"context deadline exceeded",
			"Error should mention context deadline",
		)
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
				timeout: 45 * time.Second,
				want:    45 * time.Second,
			},
			{
				name:    "zero timeout uses default",
				timeout: 0,
				want:    30 * time.Second,
			},
			{
				name:    "negative timeout uses default",
				timeout: -1 * time.Second,
				want:    30 * time.Second,
			},
		}

		reg := registry.NewComponentRegistry()

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cfg := &config.Config{
					Timeouts: config.TimeoutConfig{
						FileScan: tc.timeout,
					},
				}

				scanner := NewComponentScanner(reg, cfg)
				got := scanner.getFileScanTimeout()
				assert.Equal(t, tc.want, got)
			})
		}
	})

	t.Run("scanner with concurrency uses timeout config", func(t *testing.T) {
		// Create a config with specific file scan timeout
		cfg := &config.Config{
			Timeouts: config.TimeoutConfig{
				FileScan: 2 * time.Minute,
			},
		}

		// Create registry
		reg := registry.NewComponentRegistry()

		// Create scanner with concurrency and timeout config
		scanner := NewComponentScannerWithConcurrency(reg, 4, cfg)

		// Test that the getFileScanTimeout returns the configured value
		timeout := scanner.getFileScanTimeout()
		assert.Equal(t, 2*time.Minute, timeout, "Should return configured timeout")
	})

	t.Run("multiple config parameters", func(t *testing.T) {
		// Test multiple config parameters - should use the first one
		cfg1 := &config.Config{
			Timeouts: config.TimeoutConfig{
				FileScan: 1 * time.Minute,
			},
		}
		cfg2 := &config.Config{
			Timeouts: config.TimeoutConfig{
				FileScan: 2 * time.Minute,
			},
		}

		reg := registry.NewComponentRegistry()
		scanner := NewComponentScanner(reg, cfg1, cfg2)
		timeout := scanner.getFileScanTimeout()
		assert.Equal(t, 1*time.Minute, timeout, "Should use first config")
	})
}
