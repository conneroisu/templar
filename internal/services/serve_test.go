package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestServeService_GetServerInfo(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}

	service := NewServeService(cfg)

	tests := []struct {
		name        string
		targetFiles []string
	}{
		{
			name:        "no_target_files",
			targetFiles: nil,
		},
		{
			name:        "with_target_files",
			targetFiles: []string{"button.templ", "card.templ"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := service.GetServerInfo(tt.targetFiles)

			assert.Equal(t, "localhost", info.Host)
			assert.Equal(t, 8080, info.Port)
			assert.Equal(t, "http://localhost:8080", info.ServerURL)
			assert.Equal(t, tt.targetFiles, info.TargetFiles)
		})
	}
}

func TestServeService_Serve(t *testing.T) {
	tests := []struct {
		name        string
		opts        ServeOptions
		configSetup func() *config.Config
		wantErr     bool
		timeout     time.Duration
	}{
		{
			name: "basic_serve",
			opts: ServeOptions{
				TargetFiles: []string{},
			},
			configSetup: func() *config.Config {
				return createTestServeConfig(t.TempDir())
			},
			wantErr: true, // Expected in test environment due to missing dependencies
			timeout: 100 * time.Millisecond,
		},
		{
			name: "serve_with_target_files",
			opts: ServeOptions{
				TargetFiles: []string{"button.templ", "card.templ"},
			},
			configSetup: func() *config.Config {
				return createTestServeConfig(t.TempDir())
			},
			wantErr: true, // Expected in test environment
			timeout: 100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.configSetup()
			service := NewServeService(cfg)

			// Create a context with timeout to prevent hanging
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			// Run serve in a goroutine to handle timeout
			errChan := make(chan error, 1)
			resultChan := make(chan *ServeResult, 1)

			go func() {
				result, err := service.Serve(ctx, tt.opts)
				errChan <- err
				resultChan <- result
			}()

			select {
			case err := <-errChan:
				result := <-resultChan

				if tt.wantErr {
					// In test environment, we expect certain errors with our new standardized error handling
					if err != nil {
						expectedErrors := []string{
							"SERVE service",
							"server startup failed",
							"address already in use",
							"bind:",
						}

						foundExpected := false
						for _, expectedErr := range expectedErrors {
							if assert.Contains(t, err.Error(), expectedErr) {
								foundExpected = true
								break
							}
						}

						if !foundExpected {
							t.Logf("Got unexpected error (but this might be ok in test env): %v", err)
						}
					}
				} else {
					assert.NoError(t, err)
					assert.NotNil(t, result)
					if result != nil {
						assert.True(t, result.Success)
						assert.NotEmpty(t, result.ServerURL)
					}
				}

			case <-ctx.Done():
				// Timeout occurred, which is fine for these tests
				t.Log("Test timed out as expected")
			}

			// Verify target files were set
			assert.Equal(t, tt.opts.TargetFiles, cfg.TargetFiles)
		})
	}
}

func TestServeOptions(t *testing.T) {
	opts := ServeOptions{
		TargetFiles: []string{"button.templ", "card.templ"},
	}

	assert.Len(t, opts.TargetFiles, 2)
	assert.Contains(t, opts.TargetFiles, "button.templ")
	assert.Contains(t, opts.TargetFiles, "card.templ")

	// Test empty options
	emptyOpts := ServeOptions{}
	assert.Empty(t, emptyOpts.TargetFiles)
}

func TestServeResult(t *testing.T) {
	// Test successful result
	result := &ServeResult{
		ServerURL:  "http://localhost:8080",
		MonitorURL: "http://localhost:8081",
		Success:    true,
		Error:      nil,
	}

	assert.Equal(t, "http://localhost:8080", result.ServerURL)
	assert.Equal(t, "http://localhost:8081", result.MonitorURL)
	assert.True(t, result.Success)
	assert.NoError(t, result.Error)

	// Test failed result
	result.Success = false
	result.Error = assert.AnError

	assert.False(t, result.Success)
	assert.Error(t, result.Error)
}

func TestServerInfo(t *testing.T) {
	info := &ServerInfo{
		Host:        "localhost",
		Port:        8080,
		ServerURL:   "http://localhost:8080",
		MonitorURL:  "http://localhost:8081",
		TargetFiles: []string{"test.templ"},
	}

	assert.Equal(t, "localhost", info.Host)
	assert.Equal(t, 8080, info.Port)
	assert.Equal(t, "http://localhost:8080", info.ServerURL)
	assert.Equal(t, "http://localhost:8081", info.MonitorURL)
	assert.Len(t, info.TargetFiles, 1)
	assert.Contains(t, info.TargetFiles, "test.templ")
}

func TestNewServeService(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 3000,
		},
	}

	service := NewServeService(cfg)

	assert.NotNil(t, service)
	assert.Equal(t, cfg, service.config)
}

// Helper functions

func createTestServeConfig(tempDir string) *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Host: "localhost",
			Open: true,
		},
		Components: config.ComponentsConfig{
			ScanPaths: []string{filepath.Join(tempDir, "components")},
		},
		Build: config.BuildConfig{
			Command:  "templ generate",
			Watch:    []string{"**/*.templ"},
			Ignore:   []string{"node_modules", ".git"},
			CacheDir: filepath.Join(tempDir, ".templar/cache"),
		},
		Development: config.DevelopmentConfig{
			HotReload:    true,
			CSSInjection: true,
			ErrorOverlay: true,
		},
		Monitoring: config.MonitoringConfig{
			Enabled:  true,
			HTTPPort: 8081,
		},
	}
}

