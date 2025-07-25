package server

import (
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestWebSocketManagerTimeout(t *testing.T) {
	t.Run("websocket manager respects configured websocket timeout", func(t *testing.T) {
		// Create a config with specific WebSocket timeout
		cfg := &config.Config{
			Timeouts: config.TimeoutConfig{
				WebSocket: 90 * time.Second, // Custom timeout
			},
		}

		// Create WebSocket manager with timeout config
		originValidator := &MockOriginValidator{}
		manager := NewWebSocketManager(originValidator, nil, cfg)
		defer manager.Shutdown(nil)

		// Test that the getWebSocketTimeout returns the configured value
		timeout := manager.getWebSocketTimeout()
		assert.Equal(t, 90*time.Second, timeout, "Should return configured WebSocket timeout")
	})

	t.Run("websocket manager uses default websocket timeout when no config", func(t *testing.T) {
		// Create WebSocket manager without config
		originValidator := &MockOriginValidator{}
		manager := NewWebSocketManager(originValidator, nil)
		defer manager.Shutdown(nil)

		// Test that the getWebSocketTimeout returns the default value
		timeout := manager.getWebSocketTimeout()
		assert.Equal(t, 60*time.Second, timeout, "Should return default WebSocket timeout")
	})

	t.Run("websocket manager respects configured network timeout", func(t *testing.T) {
		// Create a config with specific network timeout
		cfg := &config.Config{
			Timeouts: config.TimeoutConfig{
				Network: 15 * time.Second, // Custom network timeout
			},
		}

		// Create WebSocket manager with timeout config
		originValidator := &MockOriginValidator{}
		manager := NewWebSocketManager(originValidator, nil, cfg)
		defer manager.Shutdown(nil)

		// Test that the getNetworkTimeout returns the configured value
		timeout := manager.getNetworkTimeout()
		assert.Equal(t, 15*time.Second, timeout, "Should return configured network timeout")
	})

	t.Run("websocket manager uses default network timeout when no config", func(t *testing.T) {
		// Create WebSocket manager without config
		originValidator := &MockOriginValidator{}
		manager := NewWebSocketManager(originValidator, nil)
		defer manager.Shutdown(nil)

		// Test that the getNetworkTimeout returns the default value
		timeout := manager.getNetworkTimeout()
		assert.Equal(t, 10*time.Second, timeout, "Should return default network timeout")
	})

	t.Run("timeout configuration validation", func(t *testing.T) {
		// Test various timeout values
		testCases := []struct {
			name              string
			websocketTimeout  time.Duration
			networkTimeout    time.Duration
			expectedWS        time.Duration
			expectedNetwork   time.Duration
		}{
			{
				name:              "positive timeouts",
				websocketTimeout:  45 * time.Second,
				networkTimeout:    20 * time.Second,
				expectedWS:        45 * time.Second,
				expectedNetwork:   20 * time.Second,
			},
			{
				name:              "zero timeouts use defaults",
				websocketTimeout:  0,
				networkTimeout:    0,
				expectedWS:        60 * time.Second,
				expectedNetwork:   10 * time.Second,
			},
			{
				name:              "negative timeouts use defaults",
				websocketTimeout:  -1 * time.Second,
				networkTimeout:    -1 * time.Second,
				expectedWS:        60 * time.Second,
				expectedNetwork:   10 * time.Second,
			},
		}

		originValidator := &MockOriginValidator{}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cfg := &config.Config{
					Timeouts: config.TimeoutConfig{
						WebSocket: tc.websocketTimeout,
						Network:   tc.networkTimeout,
					},
				}

				manager := NewWebSocketManager(originValidator, nil, cfg)
				defer manager.Shutdown(nil)

				wsTimeout := manager.getWebSocketTimeout()
				networkTimeout := manager.getNetworkTimeout()
				
				assert.Equal(t, tc.expectedWS, wsTimeout, "WebSocket timeout mismatch")
				assert.Equal(t, tc.expectedNetwork, networkTimeout, "Network timeout mismatch")
			})
		}
	})

	t.Run("multiple config parameters", func(t *testing.T) {
		// Test multiple config parameters - should use the first one
		cfg1 := &config.Config{
			Timeouts: config.TimeoutConfig{
				WebSocket: 30 * time.Second,
				Network:   5 * time.Second,
			},
		}
		cfg2 := &config.Config{
			Timeouts: config.TimeoutConfig{
				WebSocket: 120 * time.Second,
				Network:   25 * time.Second,
			},
		}

		originValidator := &MockOriginValidator{}
		manager := NewWebSocketManager(originValidator, nil, cfg1, cfg2)
		defer manager.Shutdown(nil)

		wsTimeout := manager.getWebSocketTimeout()
		networkTimeout := manager.getNetworkTimeout()
		
		assert.Equal(t, 30*time.Second, wsTimeout, "Should use first config for WebSocket timeout")
		assert.Equal(t, 5*time.Second, networkTimeout, "Should use first config for network timeout")
	})
}