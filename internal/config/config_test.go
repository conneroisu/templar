package config

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name          string
		setup         func()
		expectError   bool
		expectedPaths []string
	}{
		{
			name: "successful load with defaults",
			setup: func() {
				viper.Reset()
				viper.Set("server.port", 8080)
				viper.Set("server.host", "localhost")
			},
			expectError:   false,
			expectedPaths: []string{"./components", "./views", "./examples"},
		},
		{
			name: "successful load with custom scan paths",
			setup: func() {
				viper.Reset()
				viper.Set("server.port", 3000)
				viper.Set("server.host", "0.0.0.0")
				viper.Set("components.scan_paths", []string{"./custom", "./paths"})
			},
			expectError:   false,
			expectedPaths: []string{"./custom", "./paths"},
		},
		{
			name: "no-open flag override",
			setup: func() {
				viper.Reset()
				viper.Set("server.port", 8080)
				viper.Set("server.host", "localhost")
				viper.Set("server.open", true)
				viper.Set("server.no-open", true)
			},
			expectError:   false,
			expectedPaths: []string{"./components", "./views", "./examples"},
		},
		{
			name: "invalid viper config",
			setup: func() {
				viper.Reset()
				// Set invalid configuration that would cause unmarshal to fail
				viper.Set("server.port", "invalid_port")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			config, err := Load()

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, config)
			} else {
				require.NoError(t, err)
				require.NotNil(t, config)
				assert.Equal(t, tt.expectedPaths, config.Components.ScanPaths)

				// Test no-open flag override
				if tt.name == "no-open flag override" {
					assert.False(t, config.Server.Open)
				}
			}
		})
	}
}

func TestConfigStructure(t *testing.T) {
	viper.Reset()
	viper.Set("server.port", 9090)
	viper.Set("server.host", "127.0.0.1")
	viper.Set("server.open", true)
	viper.Set("server.no-open", false)
	viper.Set("server.middleware", []string{"cors", "logging"})

	viper.Set("build.command", "templ generate")
	viper.Set("build.watch", []string{"**/*.templ"})
	viper.Set("build.ignore", []string{"node_modules", ".git"})
	viper.Set("build.cache_dir", ".templar/cache")

	viper.Set("preview.mock_data", "auto")
	viper.Set("preview.wrapper", "layout.templ")
	viper.Set("preview.auto_props", true)

	viper.Set("components.scan_paths", []string{"./components", "./ui"})
	viper.Set("components.exclude_patterns", []string{"*_test.templ", "*.bak"})

	viper.Set("development.hot_reload", true)
	viper.Set("development.css_injection", true)
	viper.Set("development.state_preservation", false)
	viper.Set("development.error_overlay", true)

	config, err := Load()

	require.NoError(t, err)
	require.NotNil(t, config)

	// Test ServerConfig
	assert.Equal(t, 9090, config.Server.Port)
	assert.Equal(t, "127.0.0.1", config.Server.Host)
	assert.True(t, config.Server.Open)
	assert.False(t, config.Server.NoOpen)
	assert.Equal(t, []string{"cors", "logging"}, config.Server.Middleware)

	// Test BuildConfig
	assert.Equal(t, "templ generate", config.Build.Command)
	assert.Equal(t, []string{"**/*.templ"}, config.Build.Watch)
	assert.Equal(t, []string{"node_modules", ".git"}, config.Build.Ignore)
	assert.Equal(t, ".templar/cache", config.Build.CacheDir)

	// Test PreviewConfig
	assert.Equal(t, "auto", config.Preview.MockData)
	assert.Equal(t, "layout.templ", config.Preview.Wrapper)
	assert.True(t, config.Preview.AutoProps)

	// Test ComponentsConfig
	assert.Equal(t, []string{"./components", "./ui"}, config.Components.ScanPaths)
	assert.Equal(t, []string{"*_test.templ", "*.bak"}, config.Components.ExcludePatterns)

	// Test DevelopmentConfig
	assert.True(t, config.Development.HotReload)
	assert.True(t, config.Development.CSSInjection)
	assert.False(t, config.Development.StatePreservation)
	assert.True(t, config.Development.ErrorOverlay)
}

func TestConfigDefaults(t *testing.T) {
	viper.Reset()
	// Set minimal required config
	viper.Set("server.port", 8080)
	viper.Set("server.host", "localhost")

	config, err := Load()

	require.NoError(t, err)
	require.NotNil(t, config)

	// Test that defaults are applied
	assert.Equal(t, []string{"./components", "./views", "./examples"}, config.Components.ScanPaths)
	assert.Empty(t, config.TargetFiles) // Should be empty initially
}

func TestTargetFiles(t *testing.T) {
	viper.Reset()
	viper.Set("server.port", 8080)
	viper.Set("server.host", "localhost")

	config, err := Load()
	require.NoError(t, err)

	// Test that TargetFiles can be set
	testFiles := []string{"component1.templ", "component2.templ"}
	config.TargetFiles = testFiles

	assert.Equal(t, testFiles, config.TargetFiles)
}

// TestLoadWithEnvironment tests loading config with environment variables
func TestLoadWithEnvironment(t *testing.T) {
	// Save original environment
	originalPort := os.Getenv("TEMPLAR_SERVER_PORT")
	originalHost := os.Getenv("TEMPLAR_SERVER_HOST")

	defer func() {
		// Restore original environment
		if originalPort != "" {
			os.Setenv("TEMPLAR_SERVER_PORT", originalPort)
		} else {
			os.Unsetenv("TEMPLAR_SERVER_PORT")
		}
		if originalHost != "" {
			os.Setenv("TEMPLAR_SERVER_HOST", originalHost)
		} else {
			os.Unsetenv("TEMPLAR_SERVER_HOST")
		}
	}()

	// Set environment variables
	os.Setenv("TEMPLAR_SERVER_PORT", "9999")
	os.Setenv("TEMPLAR_SERVER_HOST", "0.0.0.0")

	viper.Reset()
	viper.AutomaticEnv()
	viper.SetEnvPrefix("TEMPLAR")
	viper.BindEnv("server.port")
	viper.BindEnv("server.host")

	config, err := Load()
	require.NoError(t, err)

	// Note: This test might need adjustment based on how viper is configured in the actual app
	// For now, we'll just verify the config loads successfully
	assert.NotNil(t, config)
}

// TestLoadDefaults tests the loadDefaults function
func TestLoadDefaults(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected Config
	}{
		{
			name:   "empty config gets defaults",
			config: Config{},
			expected: Config{
				Build: BuildConfig{
					Command:  "templ generate",
					Watch:    []string{"**/*.templ"},
					Ignore:   []string{"node_modules", ".git"},
					CacheDir: ".templar/cache",
				},
				Server: ServerConfig{
					Auth: AuthConfig{
						Mode:            "none",
						Enabled:         false,
						LocalhostBypass: true,
						RequireAuth:     false,
					},
				},
				Preview: PreviewConfig{
					MockData:  "auto",
					Wrapper:   "layout.templ",
					AutoProps: true,
				},
				Components: ComponentsConfig{
					ExcludePatterns: []string{"*_test.templ", "*.bak"},
				},
				Development: DevelopmentConfig{
					HotReload:    true,
					CSSInjection: true,
					ErrorOverlay: true,
				},
				Plugins: PluginsConfig{
					DiscoveryPaths:   []string{"./plugins", "~/.templar/plugins"},
					Configurations:   make(map[string]PluginConfigMap),
				},
				Monitoring: MonitoringConfig{
					Enabled:       true,
					LogLevel:      "info",
					LogFormat:     "json",
					MetricsPath:   "./logs/metrics.json",
					HTTPPort:      8081,
					AlertsEnabled: false,
				},
			},
		},
		{
			name: "partially filled config preserves existing values",
			config: Config{
				Build: BuildConfig{
					Command: "custom build command",
				},
				Preview: PreviewConfig{
					MockData: "custom",
				},
			},
			expected: Config{
				Build: BuildConfig{
					Command:  "custom build command", // Preserved
					Watch:    []string{"**/*.templ"},
					Ignore:   []string{"node_modules", ".git"},
					CacheDir: ".templar/cache",
				},
				Server: ServerConfig{
					Auth: AuthConfig{
						Mode:            "none",
						Enabled:         false,
						LocalhostBypass: true,
						RequireAuth:     false,
					},
				},
				Preview: PreviewConfig{
					MockData:  "custom", // Preserved
					Wrapper:   "layout.templ",
					AutoProps: true,
				},
				Components: ComponentsConfig{
					ExcludePatterns: []string{"*_test.templ", "*.bak"},
				},
				Development: DevelopmentConfig{
					HotReload:    true,
					CSSInjection: true,
					ErrorOverlay: true,
				},
				Plugins: PluginsConfig{
					DiscoveryPaths:   []string{"./plugins", "~/.templar/plugins"},
					Configurations:   make(map[string]PluginConfigMap),
				},
				Monitoring: MonitoringConfig{
					Enabled:       true,
					LogLevel:      "info",
					LogFormat:     "json",
					MetricsPath:   "./logs/metrics.json",
					HTTPPort:      8081,
					AlertsEnabled: false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper state
			viper.Reset()
			
			loadDefaults(&tt.config)
			
			assert.Equal(t, tt.expected.Build, tt.config.Build)
			assert.Equal(t, tt.expected.Server.Auth, tt.config.Server.Auth)
			assert.Equal(t, tt.expected.Preview, tt.config.Preview)
			assert.Equal(t, tt.expected.Components, tt.config.Components)
			assert.Equal(t, tt.expected.Development, tt.config.Development)
			assert.Equal(t, tt.expected.Plugins.DiscoveryPaths, tt.config.Plugins.DiscoveryPaths)
			assert.NotNil(t, tt.config.Plugins.Configurations)
			assert.Equal(t, tt.expected.Monitoring, tt.config.Monitoring)
		})
	}
}

// TestApplyOverrides tests the applyOverrides function
func TestApplyOverrides(t *testing.T) {
	tests := []struct {
		name        string
		viperSetup  func()
		inputConfig Config
		expected    func(*Config)
	}{
		{
			name: "scan paths override via viper",
			viperSetup: func() {
				viper.Reset()
				viper.Set("components.scan_paths", []string{"./custom", "./override"})
			},
			inputConfig: Config{},
			expected: func(c *Config) {
				assert.Equal(t, []string{"./custom", "./override"}, c.Components.ScanPaths)
			},
		},
		{
			name: "development settings override via viper",
			viperSetup: func() {
				viper.Reset()
				viper.Set("development.hot_reload", false)
				viper.Set("development.css_injection", false)
				viper.Set("development.error_overlay", false)
			},
			inputConfig: Config{},
			expected: func(c *Config) {
				assert.False(t, c.Development.HotReload)
				assert.False(t, c.Development.CSSInjection)
				assert.False(t, c.Development.ErrorOverlay)
			},
		},
		{
			name: "no-open flag override",
			viperSetup: func() {
				viper.Reset()
				viper.Set("server.no-open", true)
			},
			inputConfig: Config{
				Server: ServerConfig{Open: true},
			},
			expected: func(c *Config) {
				assert.False(t, c.Server.Open)
			},
		},
		{
			name: "monitoring config override via viper",
			viperSetup: func() {
				viper.Reset()
				viper.Set("monitoring.enabled", false)
				viper.Set("monitoring.log_level", "debug")
				viper.Set("monitoring.http_port", 9000)
			},
			inputConfig: Config{},
			expected: func(c *Config) {
				assert.False(t, c.Monitoring.Enabled)
				assert.Equal(t, "debug", c.Monitoring.LogLevel)
				assert.Equal(t, 9000, c.Monitoring.HTTPPort)
			},
		},
		{
			name: "plugin config override via viper",
			viperSetup: func() {
				viper.Reset()
				viper.Set("plugins.enabled", []string{"plugin1", "plugin2"})
				viper.Set("plugins.disabled", []string{"plugin3"})
			},
			inputConfig: Config{},
			expected: func(c *Config) {
				assert.Equal(t, []string{"plugin1", "plugin2"}, c.Plugins.Enabled)
				assert.Equal(t, []string{"plugin3"}, c.Plugins.Disabled)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.viperSetup()
			
			config := tt.inputConfig
			applyOverrides(&config)
			
			tt.expected(&config)
		})
	}
}
