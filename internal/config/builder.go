// Package config provides a builder pattern for creating configurations
// with progressive complexity and clear separation of concerns.
package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// ConfigBuilder provides a fluent interface for building configurations
// with progressive complexity tiers and clear separation of concerns.
//
// Usage:
//
//	config, err := NewConfigBuilder().
//	    WithBasicSettings().
//	    WithDevelopmentMode().
//	    WithProductionOptimizations().
//	    Build()
type ConfigBuilder struct {
	config     *Config
	validators []ValidatorFunc
	tier       ConfigTier
}

// ConfigTier represents the complexity level of configuration
type ConfigTier int

const (
	TierBasic ConfigTier = iota
	TierDevelopment
	TierProduction
	TierEnterprise
)

// ValidatorFunc represents a configuration validation function
type ValidatorFunc func(*Config) error

// NewConfigBuilder creates a new configuration builder with sensible defaults
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config:     &Config{},
		validators: []ValidatorFunc{},
		tier:       TierBasic,
	}
}

// WithBasicSettings applies basic configuration suitable for simple projects
func (cb *ConfigBuilder) WithBasicSettings() *ConfigBuilder {
	cb.tier = TierBasic
	cb.config.Server = ServerConfig{
		Port:        8080,
		Host:        "localhost",
		Open:        true,
		Environment: "development",
	}
	cb.config.Components = ComponentsConfig{
		ScanPaths:       []string{"./components"},
		ExcludePatterns: []string{"*_test.templ", "*.bak"},
	}
	cb.config.Build = BuildConfig{
		Command:  "templ generate",
		Watch:    []string{"**/*.templ"},
		Ignore:   []string{"node_modules", ".git"},
		CacheDir: ".templar/cache",
	}
	return cb
}

// WithDevelopmentMode adds development-specific settings
func (cb *ConfigBuilder) WithDevelopmentMode() *ConfigBuilder {
	if cb.tier < TierDevelopment {
		cb.tier = TierDevelopment
	}
	cb.config.Development = DevelopmentConfig{
		HotReload:         true,
		CSSInjection:      true,
		ErrorOverlay:      true,
		StatePreservation: false,
	}
	cb.config.Preview = PreviewConfig{
		MockData:  "auto",
		Wrapper:   "layout.templ",
		AutoProps: true,
	}
	cb.config.Monitoring = MonitoringConfig{
		Enabled:       true,
		LogLevel:      "info",
		LogFormat:     "json",
		MetricsPath:   "./logs/metrics.json",
		HTTPPort:      8081,
		AlertsEnabled: false,
	}
	return cb
}

// WithProductionOptimizations adds production-ready settings
func (cb *ConfigBuilder) WithProductionOptimizations() *ConfigBuilder {
	if cb.tier < TierProduction {
		cb.tier = TierProduction
	}
	cb.config.Production = ProductionConfig{
		OutputDir: "dist",
		StaticDir: "static",
		AssetsDir: "assets",
		Minification: OptimizationSettings{
			CSS:            true,
			JavaScript:     true,
			HTML:           true,
			RemoveComments: true,
		},
		Compression: CompressionSettings{
			Enabled:    true,
			Algorithms: []string{"gzip", "brotli"},
			Level:      6,
			Extensions: []string{".html", ".css", ".js", ".json"},
		},
		Bundling: BundlingSettings{
			Enabled:        true,
			Strategy:       "adaptive",
			ChunkSizeLimit: 250000,
			Splitting:      true,
		},
		AssetOptimization: AssetSettings{
			CriticalCSS: true,
			TreeShaking: true,
			Images: ImageOptimization{
				Enabled: true,
				Quality: 85,
				Formats: []string{"webp", "avif"},
			},
		},
		Security: SecuritySettings{
			HSTS:                true,
			XFrameOptions:       "DENY",
			XContentTypeOptions: true,
			Scan: SecurityScanSettings{
				Enabled:      true,
				Dependencies: true,
				Secrets:      true,
			},
		},
	}
	return cb
}

// WithEnterpriseFeatures adds enterprise-level configuration
func (cb *ConfigBuilder) WithEnterpriseFeatures() *ConfigBuilder {
	cb.tier = TierEnterprise

	// Add authentication
	cb.config.Server.Auth = AuthConfig{
		Enabled:         true,
		Mode:            "token",
		RequireAuth:     true,
		LocalhostBypass: false,
		AllowedIPs:      []string{},
	}

	// Enhanced monitoring
	cb.config.Monitoring.AlertsEnabled = true

	// Production monitoring
	cb.config.Production.Monitoring = ProductionMonitoring{
		Analytics: AnalyticsSettings{
			Enabled: true,
			Privacy: true,
		},
		Performance: PerformanceMonitoring{
			Enabled: true,
			Vitals:  true,
		},
		ErrorTracking: ErrorTrackingSettings{
			Enabled:    true,
			SampleRate: 0.1,
		},
	}

	return cb
}

// WithCustomServer allows customization of server settings
func (cb *ConfigBuilder) WithCustomServer(port int, host string) *ConfigBuilder {
	cb.config.Server.Port = port
	cb.config.Server.Host = host
	cb.addValidator(validateServerConfig(&cb.config.Server))
	return cb
}

// WithComponentPaths sets custom component scan paths
func (cb *ConfigBuilder) WithComponentPaths(paths ...string) *ConfigBuilder {
	cb.config.Components.ScanPaths = paths
	cb.addValidator(validateComponentsConfig(&cb.config.Components))
	return cb
}

// WithCSS adds CSS framework configuration
func (cb *ConfigBuilder) WithCSS(framework string, config *CSSConfig) *ConfigBuilder {
	if config == nil {
		config = &CSSConfig{
			Framework:   framework,
			OutputPath:  "./static/css/style.css",
			SourcePaths: []string{"./components", "./views"},
		}

		// Add framework-specific defaults
		switch framework {
		case "tailwind":
			config.Options = map[string]interface{}{
				"purge":  true,
				"minify": true,
			}
		case "bootstrap":
			config.Options = map[string]interface{}{
				"theme":     "default",
				"customize": true,
			}
		}
	}
	cb.config.CSS = config
	return cb
}

// WithPlugins adds plugin configuration
func (cb *ConfigBuilder) WithPlugins(enabled []string, discoveryPaths []string) *ConfigBuilder {
	cb.config.Plugins = PluginsConfig{
		Enabled:        enabled,
		DiscoveryPaths: discoveryPaths,
		Configurations: make(map[string]PluginConfigMap),
	}
	cb.addValidator(validatePluginsConfig(&cb.config.Plugins))
	return cb
}

// WithEnvironment applies environment-specific overrides
func (cb *ConfigBuilder) WithEnvironment(env string) *ConfigBuilder {
	switch env {
	case "development":
		cb.WithDevelopmentMode()
		cb.config.Server.Environment = "development"
		cb.config.Monitoring.LogLevel = "debug"
	case "staging":
		cb.WithDevelopmentMode()
		cb.WithProductionOptimizations()
		cb.config.Server.Environment = "staging"
		cb.config.Production.Security.Scan.Enabled = true
	case "production":
		cb.WithProductionOptimizations()
		cb.config.Server.Environment = "production"
		cb.config.Development.HotReload = false
		cb.config.Development.ErrorOverlay = false
		cb.config.Monitoring.LogLevel = "warn"
	case "enterprise":
		cb.WithEnterpriseFeatures()
		cb.config.Server.Environment = "production"
	}
	return cb
}

// FromViper loads settings from viper configuration
func (cb *ConfigBuilder) FromViper() *ConfigBuilder {
	var viperConfig Config
	if err := viper.Unmarshal(&viperConfig); err == nil {
		cb.mergeViperConfig(&viperConfig)
	}
	return cb
}

// AddValidator adds a custom validation function
func (cb *ConfigBuilder) AddValidator(validator ValidatorFunc) *ConfigBuilder {
	cb.validators = append(cb.validators, validator)
	return cb
}

// Build creates the final configuration after applying all settings and validations
func (cb *ConfigBuilder) Build() (*Config, error) {
	// Apply viper overrides for known issues
	cb.applyViperWorkarounds()

	// Run all validators
	for _, validator := range cb.validators {
		if err := validator(cb.config); err != nil {
			return nil, fmt.Errorf("configuration validation failed: %w", err)
		}
	}

	// Final validation
	if err := validateConfig(cb.config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cb.config, nil
}

// GetTier returns the current configuration tier
func (cb *ConfigBuilder) GetTier() ConfigTier {
	return cb.tier
}

// addValidator is a helper to add validator functions
func (cb *ConfigBuilder) addValidator(err error) {
	if err != nil {
		cb.validators = append(cb.validators, func(*Config) error {
			return err
		})
	}
}

// mergeViperConfig merges settings from viper into the current config
func (cb *ConfigBuilder) mergeViperConfig(viperConfig *Config) {
	// Only merge non-zero values to avoid overriding builder settings
	if viperConfig.Server.Port != 0 {
		cb.config.Server.Port = viperConfig.Server.Port
	}
	if viperConfig.Server.Host != "" {
		cb.config.Server.Host = viperConfig.Server.Host
	}
	if len(viperConfig.Components.ScanPaths) > 0 {
		cb.config.Components.ScanPaths = viperConfig.Components.ScanPaths
	}
	// Add more merge logic as needed
}

// applyViperWorkarounds handles known viper issues with slice and boolean handling
func (cb *ConfigBuilder) applyViperWorkarounds() {
	// Handle scan_paths set via viper
	if viper.IsSet("components.scan_paths") {
		if scanPaths := viper.GetStringSlice("components.scan_paths"); len(scanPaths) > 0 {
			cb.config.Components.ScanPaths = scanPaths
		}
	}

	// Handle development settings
	if viper.IsSet("development.hot_reload") {
		cb.config.Development.HotReload = viper.GetBool("development.hot_reload")
	}
	if viper.IsSet("development.css_injection") {
		cb.config.Development.CSSInjection = viper.GetBool("development.css_injection")
	}

	// Handle monitoring settings
	if viper.IsSet("monitoring.enabled") {
		cb.config.Monitoring.Enabled = viper.GetBool("monitoring.enabled")
	}

	// Override no-open if explicitly set
	if viper.IsSet("server.no-open") && viper.GetBool("server.no-open") {
		cb.config.Server.Open = false
	}
}
