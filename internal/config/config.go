// Package config provides configuration management for Templar applications
// using Viper for flexible configuration loading from files, environment
// variables, and command-line flags.
//
// The configuration system supports:
//   - YAML configuration files with customizable paths via TEMPLAR_CONFIG_FILE
//   - Environment variable overrides with TEMPLAR_ prefix (e.g., TEMPLAR_SERVER_PORT)
//   - Command-line flag overrides with highest precedence
//   - Comprehensive validation and security checks for all configuration values
//
// Configuration Loading Order:
//  1. Command-line flags (--config, --port, etc.)
//  2. TEMPLAR_CONFIG_FILE environment variable for custom config file paths
//  3. Individual environment variables (TEMPLAR_SERVER_PORT, TEMPLAR_SERVER_HOST, etc.)
//  4. Configuration file values (.templar.yml or custom path)
//  5. Built-in defaults
//
// The system manages server settings, component scanning paths, build pipeline
// configuration, development-specific options like hot reload and error overlays,
// plugin management, and monitoring configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server      ServerConfig      `yaml:"server"`
	Build       BuildConfig       `yaml:"build"`
	Preview     PreviewConfig     `yaml:"preview"`
	Components  ComponentsConfig  `yaml:"components"`
	Development DevelopmentConfig `yaml:"development"`
	Production  ProductionConfig  `yaml:"production"`
	Plugins     PluginsConfig     `yaml:"plugins"`
	CSS         *CSSConfig        `yaml:"css,omitempty"`
	Monitoring  MonitoringConfig  `yaml:"monitoring"`
	TargetFiles []string          `yaml:"-"` // CLI arguments, not from config file
}

type ServerConfig struct {
	Port           int        `yaml:"port"`
	Host           string     `yaml:"host"`
	Open           bool       `yaml:"open"`
	NoOpen         bool       `yaml:"no-open"`
	Middleware     []string   `yaml:"middleware"`
	AllowedOrigins []string   `yaml:"allowed_origins"`
	Environment    string     `yaml:"environment"`
	Auth           AuthConfig `yaml:"auth"`
}

type AuthConfig struct {
	Enabled         bool     `yaml:"enabled"`
	Mode            string   `yaml:"mode"`             // "token", "basic", "none"
	Token           string   `yaml:"token"`            // Simple token for token mode
	Username        string   `yaml:"username"`         // Username for basic auth
	Password        string   `yaml:"password"`         // Password for basic auth
	AllowedIPs      []string `yaml:"allowed_ips"`      // IP allowlist
	RequireAuth     bool     `yaml:"require_auth"`     // Require auth for non-localhost
	LocalhostBypass bool     `yaml:"localhost_bypass"` // Allow localhost without auth
}

type BuildConfig struct {
	Command  string   `yaml:"command"`
	Watch    []string `yaml:"watch"`
	Ignore   []string `yaml:"ignore"`
	CacheDir string   `yaml:"cache_dir"`
}

type PreviewConfig struct {
	MockData  string `yaml:"mock_data"`
	Wrapper   string `yaml:"wrapper"`
	AutoProps bool   `yaml:"auto_props"`
}

type ComponentsConfig struct {
	ScanPaths       []string `yaml:"scan_paths"`
	ExcludePatterns []string `yaml:"exclude_patterns"`
}

type DevelopmentConfig struct {
	HotReload         bool `yaml:"hot_reload"`
	CSSInjection      bool `yaml:"css_injection"`
	StatePreservation bool `yaml:"state_preservation"`
	ErrorOverlay      bool `yaml:"error_overlay"`
}

// ProductionConfig defines production-specific build and deployment settings
type ProductionConfig struct {
	// Output configuration
	OutputDir    string `yaml:"output_dir"`
	StaticDir    string `yaml:"static_dir"`
	AssetsDir    string `yaml:"assets_dir"`
	
	// Build optimization
	Minification      OptimizationSettings `yaml:"minification"`
	Compression       CompressionSettings  `yaml:"compression"`
	AssetOptimization AssetSettings        `yaml:"asset_optimization"`
	
	// Bundling and chunking
	Bundling      BundlingSettings `yaml:"bundling"`
	CodeSplitting SplittingSettings `yaml:"code_splitting"`
	
	// Deployment and CDN
	Deployment DeploymentSettings `yaml:"deployment"`
	CDN        CDNSettings        `yaml:"cdn"`
	
	// Performance and monitoring
	Performance PerformanceSettings `yaml:"performance"`
	Monitoring  ProductionMonitoring `yaml:"monitoring"`
	
	// Security and validation
	Security   SecuritySettings   `yaml:"security"`
	Validation ValidationSettings `yaml:"validation"`
	
	// Environment-specific overrides
	Environments map[string]EnvironmentConfig `yaml:"environments"`
}

// OptimizationSettings controls various optimization features
type OptimizationSettings struct {
	CSS        bool `yaml:"css"`
	JavaScript bool `yaml:"javascript"`
	HTML       bool `yaml:"html"`
	JSON       bool `yaml:"json"`
	RemoveComments bool `yaml:"remove_comments"`
	StripDebug     bool `yaml:"strip_debug"`
}

// CompressionSettings controls asset compression
type CompressionSettings struct {
	Enabled    bool     `yaml:"enabled"`
	Algorithms []string `yaml:"algorithms"` // gzip, brotli, deflate
	Level      int      `yaml:"level"`      // compression level 1-9
	Extensions []string `yaml:"extensions"` // file extensions to compress
}

// AssetSettings controls image and asset optimization
type AssetSettings struct {
	Images       ImageOptimization `yaml:"images"`
	Fonts        FontOptimization  `yaml:"fonts"`
	Icons        IconOptimization  `yaml:"icons"`
	CriticalCSS  bool             `yaml:"critical_css"`
	TreeShaking  bool             `yaml:"tree_shaking"`
	DeadCodeElimination bool      `yaml:"dead_code_elimination"`
}

// ImageOptimization controls image processing
type ImageOptimization struct {
	Enabled     bool     `yaml:"enabled"`
	Quality     int      `yaml:"quality"`
	Progressive bool     `yaml:"progressive"`
	Formats     []string `yaml:"formats"` // webp, avif, etc.
	Responsive  bool     `yaml:"responsive"`
}

// FontOptimization controls font processing
type FontOptimization struct {
	Enabled    bool     `yaml:"enabled"`
	Subsetting bool     `yaml:"subsetting"`
	Formats    []string `yaml:"formats"` // woff2, woff, etc.
	Preload    bool     `yaml:"preload"`
}

// IconOptimization controls icon processing
type IconOptimization struct {
	Enabled bool   `yaml:"enabled"`
	Sprite  bool   `yaml:"sprite"`
	SVG     bool   `yaml:"svg_optimization"`
	Format  string `yaml:"format"` // svg, sprite, font
}

// BundlingSettings controls asset bundling
type BundlingSettings struct {
	Enabled      bool     `yaml:"enabled"`
	Strategy     string   `yaml:"strategy"`     // single, multiple, adaptive
	ChunkSizeLimit int64  `yaml:"chunk_size_limit"`
	Externals    []string `yaml:"externals"`   // external dependencies to exclude
	Splitting    bool     `yaml:"splitting"`   // enable automatic code splitting
}

// SplittingSettings controls code splitting
type SplittingSettings struct {
	Enabled        bool     `yaml:"enabled"`
	VendorSplit    bool     `yaml:"vendor_split"`
	AsyncChunks    bool     `yaml:"async_chunks"`
	CommonChunks   bool     `yaml:"common_chunks"`
	ManualChunks   []string `yaml:"manual_chunks"`
}

// DeploymentSettings controls deployment configuration
type DeploymentSettings struct {
	Target        string            `yaml:"target"`         // static, docker, serverless
	Environment   string            `yaml:"environment"`    // production, staging, preview
	BaseURL       string            `yaml:"base_url"`       // deployment base URL
	AssetPrefix   string            `yaml:"asset_prefix"`   // prefix for asset URLs
	Headers       map[string]string `yaml:"headers"`        // custom HTTP headers
	Redirects     []RedirectRule    `yaml:"redirects"`      // URL redirects
	ErrorPages    map[string]string `yaml:"error_pages"`    // custom error pages
}

// RedirectRule defines URL redirect rules
type RedirectRule struct {
	From   string `yaml:"from"`
	To     string `yaml:"to"`
	Status int    `yaml:"status"`
}

// CDNSettings controls CDN integration
type CDNSettings struct {
	Enabled     bool              `yaml:"enabled"`
	Provider    string            `yaml:"provider"`    // cloudflare, aws, etc.
	BasePath    string            `yaml:"base_path"`   // CDN base path
	CacheTTL    int               `yaml:"cache_ttl"`   // cache time-to-live
	Invalidation bool             `yaml:"invalidation"` // auto-invalidate on deploy
	Headers     map[string]string `yaml:"headers"`     // CDN-specific headers
}

// PerformanceSettings controls performance optimizations
type PerformanceSettings struct {
	BudgetLimits    map[string]int64 `yaml:"budget_limits"`    // size budgets
	Preconnect      []string         `yaml:"preconnect"`       // domains to preconnect
	Prefetch        []string         `yaml:"prefetch"`         // resources to prefetch
	Preload         []string         `yaml:"preload"`          // resources to preload
	LazyLoading     bool             `yaml:"lazy_loading"`     // enable lazy loading
	ServiceWorker   bool             `yaml:"service_worker"`   // generate service worker
	ManifestFile    bool             `yaml:"manifest_file"`    // generate web manifest
}

// ProductionMonitoring extends base monitoring for production
type ProductionMonitoring struct {
	Analytics    AnalyticsSettings `yaml:"analytics"`
	ErrorTracking ErrorTrackingSettings `yaml:"error_tracking"`
	Performance  PerformanceMonitoring `yaml:"performance"`
	Uptime       UptimeSettings        `yaml:"uptime"`
}

// AnalyticsSettings controls analytics integration
type AnalyticsSettings struct {
	Enabled  bool   `yaml:"enabled"`
	Provider string `yaml:"provider"` // google, plausible, etc.
	ID       string `yaml:"id"`       // tracking ID
	Privacy  bool   `yaml:"privacy"`  // privacy-focused analytics
}

// ErrorTrackingSettings controls error tracking
type ErrorTrackingSettings struct {
	Enabled     bool   `yaml:"enabled"`
	Provider    string `yaml:"provider"` // sentry, bugsnag, etc.
	DSN         string `yaml:"dsn"`      // data source name
	Environment string `yaml:"environment"`
	SampleRate  float64 `yaml:"sample_rate"`
}

// PerformanceMonitoring controls performance tracking
type PerformanceMonitoring struct {
	Enabled      bool     `yaml:"enabled"`
	RealUserData bool     `yaml:"real_user_data"`
	Vitals       bool     `yaml:"vitals"`        // Core Web Vitals
	Metrics      []string `yaml:"metrics"`      // custom metrics to track
}

// UptimeSettings controls uptime monitoring
type UptimeSettings struct {
	Enabled   bool     `yaml:"enabled"`
	Endpoints []string `yaml:"endpoints"` // endpoints to monitor
	Interval  int      `yaml:"interval"`  // check interval in seconds
	Alerts    bool     `yaml:"alerts"`    // enable alerting
}

// SecuritySettings controls security features
type SecuritySettings struct {
	CSP                ContentSecurityPolicy `yaml:"csp"`
	HSTS               bool                  `yaml:"hsts"`
	XFrameOptions      string                `yaml:"x_frame_options"`
	XContentTypeOptions bool                 `yaml:"x_content_type_options"`
	Scan               SecurityScanSettings  `yaml:"scan"`
	Secrets            SecretsSettings       `yaml:"secrets"`
}

// ContentSecurityPolicy defines CSP configuration
type ContentSecurityPolicy struct {
	Enabled   bool              `yaml:"enabled"`
	Directives map[string]string `yaml:"directives"`
	ReportURI string             `yaml:"report_uri"`
}

// SecurityScanSettings controls security scanning
type SecurityScanSettings struct {
	Enabled       bool     `yaml:"enabled"`
	Dependencies  bool     `yaml:"dependencies"`  // scan dependencies for vulnerabilities
	Secrets       bool     `yaml:"secrets"`       // scan for exposed secrets
	StaticAnalysis bool    `yaml:"static_analysis"` // static code analysis
	AllowedRisks  []string `yaml:"allowed_risks"` // acceptable risk levels
}

// SecretsSettings controls secrets management
type SecretsSettings struct {
	Detection    bool     `yaml:"detection"`     // detect secrets in code
	Validation   bool     `yaml:"validation"`    // validate secret formats
	Patterns     []string `yaml:"patterns"`      // custom secret patterns
	Exclusions   []string `yaml:"exclusions"`    // files to exclude from scanning
}

// ValidationSettings controls build validation
type ValidationSettings struct {
	Enabled      bool                `yaml:"enabled"`
	Accessibility AccessibilityChecks `yaml:"accessibility"`
	Performance   PerformanceChecks   `yaml:"performance"`
	SEO          SEOChecks           `yaml:"seo"`
	Links        LinkChecks          `yaml:"links"`
	Standards    StandardsChecks     `yaml:"standards"`
}

// AccessibilityChecks controls accessibility validation
type AccessibilityChecks struct {
	Enabled    bool     `yaml:"enabled"`
	Level      string   `yaml:"level"`      // A, AA, AAA
	Rules      []string `yaml:"rules"`      // specific rules to check
	IgnoreRules []string `yaml:"ignore_rules"` // rules to ignore
}

// PerformanceChecks controls performance validation
type PerformanceChecks struct {
	Enabled       bool              `yaml:"enabled"`
	BundleSize    int64             `yaml:"bundle_size"`    // max bundle size
	LoadTime      int               `yaml:"load_time"`      // max load time (ms)
	Metrics       map[string]int64  `yaml:"metrics"`        // custom performance metrics
	Lighthouse    bool              `yaml:"lighthouse"`     // run Lighthouse audits
}

// SEOChecks controls SEO validation
type SEOChecks struct {
	Enabled     bool     `yaml:"enabled"`
	MetaTags    bool     `yaml:"meta_tags"`     // validate meta tags
	Sitemap     bool     `yaml:"sitemap"`       // generate and validate sitemap
	Robots      bool     `yaml:"robots"`        // generate robots.txt
	Schema      bool     `yaml:"schema"`        // validate structured data
	OpenGraph   bool     `yaml:"open_graph"`    // validate Open Graph tags
}

// LinkChecks controls link validation
type LinkChecks struct {
	Enabled    bool     `yaml:"enabled"`
	Internal   bool     `yaml:"internal"`   // check internal links
	External   bool     `yaml:"external"`   // check external links
	Images     bool     `yaml:"images"`     // check image links
	Timeout    int      `yaml:"timeout"`    // request timeout (seconds)
	IgnoreUrls []string `yaml:"ignore_urls"` // URLs to ignore
}

// StandardsChecks controls web standards validation
type StandardsChecks struct {
	Enabled   bool `yaml:"enabled"`
	HTML      bool `yaml:"html"`      // validate HTML
	CSS       bool `yaml:"css"`       // validate CSS
	JavaScript bool `yaml:"javascript"` // validate JavaScript
	W3C       bool `yaml:"w3c"`       // use W3C validators
}

// EnvironmentConfig allows per-environment configuration overrides
type EnvironmentConfig struct {
	Extends     string                 `yaml:"extends"`     // inherit from another environment
	Variables   map[string]string      `yaml:"variables"`   // environment variables
	Features    map[string]bool        `yaml:"features"`    // feature flags
	Overrides   map[string]interface{} `yaml:"overrides"`   // configuration overrides
	Deployment  DeploymentSettings     `yaml:"deployment"`  // environment-specific deployment
	CDN         CDNSettings            `yaml:"cdn"`         // environment-specific CDN
	Monitoring  ProductionMonitoring   `yaml:"monitoring"`  // environment-specific monitoring
}

type PluginsConfig struct {
	Enabled        []string                   `yaml:"enabled"`
	Disabled       []string                   `yaml:"disabled"`
	DiscoveryPaths []string                   `yaml:"discovery_paths"`
	Configurations map[string]PluginConfigMap `yaml:"configurations"`
}

type PluginConfigMap map[string]interface{}

type MonitoringConfig struct {
	Enabled       bool   `yaml:"enabled"`
	LogLevel      string `yaml:"log_level"`
	LogFormat     string `yaml:"log_format"`
	MetricsPath   string `yaml:"metrics_path"`
	HTTPPort      int    `yaml:"http_port"`
	AlertsEnabled bool   `yaml:"alerts_enabled"`
}

// CSSConfig defines CSS framework integration configuration
type CSSConfig struct {
	Framework    string               `yaml:"framework"`     // "tailwind", "bootstrap", "bulma", etc.
	OutputPath   string               `yaml:"output_path"`   // Path for generated CSS
	SourcePaths  []string             `yaml:"source_paths"`  // Paths to scan for CSS classes
	Optimization *OptimizationConfig  `yaml:"optimization,omitempty"`
	Theming      *ThemingConfig       `yaml:"theming,omitempty"`
	Variables    map[string]string    `yaml:"variables,omitempty"`
	Options      map[string]interface{} `yaml:"options,omitempty"`
}

// OptimizationConfig defines CSS optimization settings
type OptimizationConfig struct {
	Purge  bool `yaml:"purge"`
	Minify bool `yaml:"minify"`
}

// ThemingConfig defines CSS theming settings
type ThemingConfig struct {
	ExtractVariables bool `yaml:"extract_variables"`
	StyleGuide       bool `yaml:"style_guide"`
}

// loadDefaults applies default values to all configuration sections when not explicitly set.
// This function handles the application of sensible defaults across all configuration structs.
func loadDefaults(config *Config) {
	// Apply default values for BuildConfig if not set
	if config.Build.Command == "" {
		config.Build.Command = "templ generate"
	}
	if len(config.Build.Watch) == 0 {
		config.Build.Watch = []string{"**/*.templ"}
	}
	if len(config.Build.Ignore) == 0 {
		config.Build.Ignore = []string{"node_modules", ".git"}
	}
	if config.Build.CacheDir == "" {
		config.Build.CacheDir = ".templar/cache"
	}

	// Apply default values for AuthConfig if not set
	if config.Server.Auth.Mode == "" {
		config.Server.Auth.Mode = "none"
	}
	if !viper.IsSet("server.auth.enabled") {
		config.Server.Auth.Enabled = false
	}
	if !viper.IsSet("server.auth.localhost_bypass") {
		config.Server.Auth.LocalhostBypass = true // Default to allowing localhost without auth
	}
	if !viper.IsSet("server.auth.require_auth") {
		config.Server.Auth.RequireAuth = false // Default to not requiring auth
	}

	// Apply default values for PreviewConfig if not set
	if config.Preview.MockData == "" {
		config.Preview.MockData = "auto"
	}
	if config.Preview.Wrapper == "" {
		config.Preview.Wrapper = "layout.templ"
	}
	if !viper.IsSet("preview.auto_props") {
		config.Preview.AutoProps = true
	}

	// Apply default values for ComponentsConfig if not set
	if len(config.Components.ExcludePatterns) == 0 {
		config.Components.ExcludePatterns = []string{"*_test.templ", "*.bak"}
	}

	// Apply default values for DevelopmentConfig if not set
	if !viper.IsSet("development.hot_reload") {
		config.Development.HotReload = true
	}
	if !viper.IsSet("development.css_injection") {
		config.Development.CSSInjection = true
	}
	if !viper.IsSet("development.error_overlay") {
		config.Development.ErrorOverlay = true
	}

	// Apply default values for PluginsConfig if not set
	if len(config.Plugins.DiscoveryPaths) == 0 {
		config.Plugins.DiscoveryPaths = []string{"./plugins", "~/.templar/plugins"}
	}
	if config.Plugins.Configurations == nil {
		config.Plugins.Configurations = make(map[string]PluginConfigMap)
	}

	// Apply default values for MonitoringConfig if not set
	if !viper.IsSet("monitoring.enabled") {
		config.Monitoring.Enabled = true // Enable monitoring by default
	}
	if config.Monitoring.LogLevel == "" {
		config.Monitoring.LogLevel = "info"
	}
	if config.Monitoring.LogFormat == "" {
		config.Monitoring.LogFormat = "json"
	}
	if config.Monitoring.MetricsPath == "" {
		config.Monitoring.MetricsPath = "./logs/metrics.json"
	}
	if config.Monitoring.HTTPPort == 0 {
		config.Monitoring.HTTPPort = 8081
	}
	if !viper.IsSet("monitoring.alerts_enabled") {
		config.Monitoring.AlertsEnabled = false // Disable alerts by default
	}

	// Apply default values for ProductionConfig if not set
	loadProductionDefaults(&config.Production)
}

// loadProductionDefaults applies default values for production configuration
func loadProductionDefaults(prod *ProductionConfig) {
	// Output configuration defaults
	if prod.OutputDir == "" {
		prod.OutputDir = "dist"
	}
	if prod.StaticDir == "" {
		prod.StaticDir = "static"
	}
	if prod.AssetsDir == "" {
		prod.AssetsDir = "assets"
	}

	// Minification defaults
	if !viper.IsSet("production.minification.css") {
		prod.Minification.CSS = true
	}
	if !viper.IsSet("production.minification.javascript") {
		prod.Minification.JavaScript = true
	}
	if !viper.IsSet("production.minification.html") {
		prod.Minification.HTML = true
	}
	if !viper.IsSet("production.minification.remove_comments") {
		prod.Minification.RemoveComments = true
	}

	// Compression defaults
	if !viper.IsSet("production.compression.enabled") {
		prod.Compression.Enabled = true
	}
	if len(prod.Compression.Algorithms) == 0 {
		prod.Compression.Algorithms = []string{"gzip", "brotli"}
	}
	if prod.Compression.Level == 0 {
		prod.Compression.Level = 6 // Balanced compression level
	}
	if len(prod.Compression.Extensions) == 0 {
		prod.Compression.Extensions = []string{".html", ".css", ".js", ".json", ".xml", ".svg"}
	}

	// Asset optimization defaults
	if !viper.IsSet("production.asset_optimization.critical_css") {
		prod.AssetOptimization.CriticalCSS = true
	}
	if !viper.IsSet("production.asset_optimization.tree_shaking") {
		prod.AssetOptimization.TreeShaking = true
	}
	if !viper.IsSet("production.asset_optimization.images.enabled") {
		prod.AssetOptimization.Images.Enabled = true
	}
	if prod.AssetOptimization.Images.Quality == 0 {
		prod.AssetOptimization.Images.Quality = 85
	}
	if len(prod.AssetOptimization.Images.Formats) == 0 {
		prod.AssetOptimization.Images.Formats = []string{"webp", "avif"}
	}

	// Bundling defaults
	if !viper.IsSet("production.bundling.enabled") {
		prod.Bundling.Enabled = true
	}
	if prod.Bundling.Strategy == "" {
		prod.Bundling.Strategy = "adaptive"
	}
	if prod.Bundling.ChunkSizeLimit == 0 {
		prod.Bundling.ChunkSizeLimit = 250000 // 250KB chunks
	}

	// Code splitting defaults
	if !viper.IsSet("production.code_splitting.enabled") {
		prod.CodeSplitting.Enabled = true
	}
	if !viper.IsSet("production.code_splitting.vendor_split") {
		prod.CodeSplitting.VendorSplit = true
	}
	if !viper.IsSet("production.code_splitting.async_chunks") {
		prod.CodeSplitting.AsyncChunks = true
	}

	// Deployment defaults
	if prod.Deployment.Target == "" {
		prod.Deployment.Target = "static"
	}
	if prod.Deployment.Environment == "" {
		prod.Deployment.Environment = "production"
	}
	if prod.Deployment.Headers == nil {
		prod.Deployment.Headers = map[string]string{
			"X-Content-Type-Options": "nosniff",
			"X-Frame-Options":        "DENY",
			"X-XSS-Protection":       "1; mode=block",
		}
	}

	// Performance defaults
	if prod.Performance.BudgetLimits == nil {
		prod.Performance.BudgetLimits = map[string]int64{
			"bundle_size":    500000,  // 500KB
			"image_size":     1000000, // 1MB
			"css_size":       100000,  // 100KB
			"js_size":        300000,  // 300KB
		}
	}
	if !viper.IsSet("production.performance.lazy_loading") {
		prod.Performance.LazyLoading = true
	}

	// Security defaults
	if !viper.IsSet("production.security.hsts") {
		prod.Security.HSTS = true
	}
	if prod.Security.XFrameOptions == "" {
		prod.Security.XFrameOptions = "DENY"
	}
	if !viper.IsSet("production.security.x_content_type_options") {
		prod.Security.XContentTypeOptions = true
	}
	if !viper.IsSet("production.security.scan.enabled") {
		prod.Security.Scan.Enabled = true
	}
	if !viper.IsSet("production.security.scan.dependencies") {
		prod.Security.Scan.Dependencies = true
	}

	// Validation defaults
	if !viper.IsSet("production.validation.enabled") {
		prod.Validation.Enabled = true
	}
	if !viper.IsSet("production.validation.accessibility.enabled") {
		prod.Validation.Accessibility.Enabled = true
	}
	if prod.Validation.Accessibility.Level == "" {
		prod.Validation.Accessibility.Level = "AA"
	}
	if !viper.IsSet("production.validation.performance.enabled") {
		prod.Validation.Performance.Enabled = true
	}
	if prod.Validation.Performance.BundleSize == 0 {
		prod.Validation.Performance.BundleSize = 500000 // 500KB
	}
	if prod.Validation.Performance.LoadTime == 0 {
		prod.Validation.Performance.LoadTime = 3000 // 3 seconds
	}

	// Initialize environments if nil
	if prod.Environments == nil {
		prod.Environments = make(map[string]EnvironmentConfig)
	}

	// Add default environments if they don't exist
	if _, exists := prod.Environments["staging"]; !exists {
		prod.Environments["staging"] = EnvironmentConfig{
			Variables: map[string]string{
				"NODE_ENV": "staging",
			},
			Features: map[string]bool{
				"debug_mode": true,
				"analytics":  false,
			},
			Deployment: DeploymentSettings{
				Environment: "staging",
				BaseURL:     "https://staging.example.com",
			},
		}
	}

	if _, exists := prod.Environments["preview"]; !exists {
		prod.Environments["preview"] = EnvironmentConfig{
			Variables: map[string]string{
				"NODE_ENV": "development",
			},
			Features: map[string]bool{
				"debug_mode": true,
				"analytics":  false,
				"hot_reload": true,
			},
			Deployment: DeploymentSettings{
				Environment: "preview",
				BaseURL:     "https://preview.example.com",
			},
		}
	}

	if _, exists := prod.Environments["production"]; !exists {
		prod.Environments["production"] = EnvironmentConfig{
			Variables: map[string]string{
				"NODE_ENV": "production",
			},
			Features: map[string]bool{
				"debug_mode": false,
				"analytics":  true,
				"hot_reload": false,
			},
			Deployment: DeploymentSettings{
				Environment: "production",
				BaseURL:     "https://example.com",
			},
			Monitoring: ProductionMonitoring{
				Analytics: AnalyticsSettings{
					Enabled: true,
					Privacy: true,
				},
				Performance: PerformanceMonitoring{
					Enabled: true,
					Vitals:  true,
				},
			},
		}
	}
}

// applyOverrides handles Viper-specific workarounds and explicit overrides from environment variables and flags.
// This function addresses known Viper issues with slice and boolean value handling.
func applyOverrides(config *Config) {
	// Apply defaults for components scan paths only if not explicitly set
	if !viper.IsSet("components.scan_paths") && len(config.Components.ScanPaths) == 0 {
		config.Components.ScanPaths = []string{"./components", "./views", "./examples"}
	}

	// Handle scan_paths set via viper (workaround for viper slice handling)
	if viper.IsSet("components.scan_paths") && len(config.Components.ScanPaths) == 0 {
		scanPaths := viper.GetStringSlice("components.scan_paths")
		if len(scanPaths) > 0 {
			config.Components.ScanPaths = scanPaths
		}
	}

	// Handle development settings set via viper (workaround for viper bool handling)
	if viper.IsSet("development.hot_reload") {
		config.Development.HotReload = viper.GetBool("development.hot_reload")
	}
	if viper.IsSet("development.css_injection") {
		config.Development.CSSInjection = viper.GetBool("development.css_injection")
	}
	if viper.IsSet("development.state_preservation") {
		config.Development.StatePreservation = viper.GetBool("development.state_preservation")
	}
	if viper.IsSet("development.error_overlay") {
		config.Development.ErrorOverlay = viper.GetBool("development.error_overlay")
	}

	// Handle preview settings
	if viper.IsSet("preview.auto_props") {
		config.Preview.AutoProps = viper.GetBool("preview.auto_props")
	}

	// Handle exclude patterns set via viper (workaround for viper slice handling)
	if viper.IsSet("components.exclude_patterns") && len(config.Components.ExcludePatterns) == 0 {
		excludePatterns := viper.GetStringSlice("components.exclude_patterns")
		if len(excludePatterns) > 0 {
			config.Components.ExcludePatterns = excludePatterns
		}
	}

	// Override no-open if explicitly set via flag
	if viper.IsSet("server.no-open") && viper.GetBool("server.no-open") {
		config.Server.Open = false
	}

	// Handle plugin configuration set via viper
	if viper.IsSet("plugins.enabled") {
		config.Plugins.Enabled = viper.GetStringSlice("plugins.enabled")
	}
	if viper.IsSet("plugins.disabled") {
		config.Plugins.Disabled = viper.GetStringSlice("plugins.disabled")
	}
	if viper.IsSet("plugins.discovery_paths") {
		config.Plugins.DiscoveryPaths = viper.GetStringSlice("plugins.discovery_paths")
	}

	// Handle monitoring configuration set via viper
	if viper.IsSet("monitoring.enabled") {
		config.Monitoring.Enabled = viper.GetBool("monitoring.enabled")
	}
	if viper.IsSet("monitoring.log_level") {
		config.Monitoring.LogLevel = viper.GetString("monitoring.log_level")
	}
	if viper.IsSet("monitoring.log_format") {
		config.Monitoring.LogFormat = viper.GetString("monitoring.log_format")
	}
	if viper.IsSet("monitoring.metrics_path") {
		config.Monitoring.MetricsPath = viper.GetString("monitoring.metrics_path")
	}
	if viper.IsSet("monitoring.http_port") {
		config.Monitoring.HTTPPort = viper.GetInt("monitoring.http_port")
	}
	if viper.IsSet("monitoring.alerts_enabled") {
		config.Monitoring.AlertsEnabled = viper.GetBool("monitoring.alerts_enabled")
	}
}

// Load reads configuration from all available sources and returns a fully populated Config struct.
//
// This function expects that Viper has already been configured by cmd.initConfig() with:
//   - Config file path (from --config flag, TEMPLAR_CONFIG_FILE env var, or default .templar.yml)
//   - Environment variable binding with TEMPLAR_ prefix
//   - Automatic environment variable reading enabled
//
// The function applies intelligent defaults, handles Viper's quirks with slice/bool values,
// and performs comprehensive security validation on all configuration values.
func Load() (*Config, error) {
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	// Apply default values for all configuration sections
	loadDefaults(&config)
	
	// Apply overrides from Viper (environment variables, flags, etc.)
	applyOverrides(&config)

	// Validate configuration values
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// GetEnvironmentConfig returns environment-specific configuration
func (c *Config) GetEnvironmentConfig(env string) (*EnvironmentConfig, bool) {
	if c.Production.Environments == nil {
		return nil, false
	}
	envConfig, exists := c.Production.Environments[env]
	return &envConfig, exists
}

// ApplyEnvironmentOverrides applies environment-specific configuration overrides
func (c *Config) ApplyEnvironmentOverrides(env string) error {
	envConfig, exists := c.GetEnvironmentConfig(env)
	if !exists {
		return fmt.Errorf("environment '%s' not found in configuration", env)
	}

	// Apply environment variables
	for key, value := range envConfig.Variables {
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("failed to set environment variable %s: %w", key, err)
		}
	}

	// Apply deployment overrides
	if envConfig.Deployment.Environment != "" {
		c.Production.Deployment.Environment = envConfig.Deployment.Environment
	}
	if envConfig.Deployment.BaseURL != "" {
		c.Production.Deployment.BaseURL = envConfig.Deployment.BaseURL
	}
	if envConfig.Deployment.AssetPrefix != "" {
		c.Production.Deployment.AssetPrefix = envConfig.Deployment.AssetPrefix
	}

	// Apply CDN overrides
	if envConfig.CDN.BasePath != "" {
		c.Production.CDN.BasePath = envConfig.CDN.BasePath
	}
	if envConfig.CDN.Provider != "" {
		c.Production.CDN.Provider = envConfig.CDN.Provider
	}

	// Apply monitoring overrides
	if envConfig.Monitoring.Analytics.Enabled != c.Production.Monitoring.Analytics.Enabled {
		c.Production.Monitoring.Analytics.Enabled = envConfig.Monitoring.Analytics.Enabled
	}
	if envConfig.Monitoring.Performance.Enabled != c.Production.Monitoring.Performance.Enabled {
		c.Production.Monitoring.Performance.Enabled = envConfig.Monitoring.Performance.Enabled
	}

	return nil
}

// GetProductionConfig returns production configuration with environment overrides applied
func (c *Config) GetProductionConfig(env string) (*ProductionConfig, error) {
	// Clone production config to avoid modifying original
	prodConfig := c.Production

	// Apply environment overrides if environment is specified
	if env != "" {
		envConfig, exists := c.GetEnvironmentConfig(env)
		if !exists {
			return nil, fmt.Errorf("environment '%s' not found", env)
		}

		// Apply feature flags
		for feature, enabled := range envConfig.Features {
			switch feature {
			case "debug_mode":
				// Enable/disable debug features
				if !enabled {
					prodConfig.Security.Scan.Enabled = false
					prodConfig.Validation.Enabled = false
				}
			case "analytics":
				prodConfig.Monitoring.Analytics.Enabled = enabled
			case "hot_reload":
				// Hot reload is development feature, disabled in production
				if enabled && env != "production" {
					prodConfig.AssetOptimization.CriticalCSS = false
				}
			}
		}

		// Apply deployment settings
		prodConfig.Deployment = envConfig.Deployment
		prodConfig.CDN = envConfig.CDN
		prodConfig.Monitoring = envConfig.Monitoring
	}

	return &prodConfig, nil
}

// validateConfig validates configuration values for security and correctness
func validateConfig(config *Config) error {
	// Validate server configuration
	if err := validateServerConfig(&config.Server); err != nil {
		return fmt.Errorf("server config: %w", err)
	}

	// Validate build configuration
	if err := validateBuildConfig(&config.Build); err != nil {
		return fmt.Errorf("build config: %w", err)
	}

	// Validate components configuration
	if err := validateComponentsConfig(&config.Components); err != nil {
		return fmt.Errorf("components config: %w", err)
	}

	// Validate plugins configuration
	if err := validatePluginsConfig(&config.Plugins); err != nil {
		return fmt.Errorf("plugins config: %w", err)
	}

	// Validate monitoring configuration
	if err := validateMonitoringConfig(&config.Monitoring); err != nil {
		return fmt.Errorf("monitoring config: %w", err)
	}

	return nil
}

// validateServerConfig validates server configuration values
func validateServerConfig(config *ServerConfig) error {
	// Validate port range (allow 0 for system-assigned ports in testing)
	if config.Port < 0 || config.Port > 65535 {
		return fmt.Errorf("port %d is not in valid range 0-65535", config.Port)
	}

	// Validate host
	if config.Host != "" {
		// Basic validation - no dangerous characters
		dangerousChars := []string{";", "&", "|", "$", "`", "(", ")", "<", ">", "\"", "'", "\\"}
		for _, char := range dangerousChars {
			if strings.Contains(config.Host, char) {
				return fmt.Errorf("host contains dangerous character: %s", char)
			}
		}
	}

	return nil
}

// validateBuildConfig validates build configuration values
func validateBuildConfig(config *BuildConfig) error {
	// Validate cache directory if specified
	if config.CacheDir != "" {
		// Clean the path
		cleanPath := filepath.Clean(config.CacheDir)

		// Reject path traversal attempts
		if strings.Contains(cleanPath, "..") {
			return fmt.Errorf("cache_dir contains path traversal: %s", config.CacheDir)
		}

		// Should be relative path for security
		if filepath.IsAbs(cleanPath) {
			return fmt.Errorf("cache_dir should be relative path: %s", config.CacheDir)
		}
	}

	return nil
}

// validateComponentsConfig validates components configuration values
func validateComponentsConfig(config *ComponentsConfig) error {
	// Validate scan paths
	for _, path := range config.ScanPaths {
		if err := validatePath(path); err != nil {
			return fmt.Errorf("invalid scan path '%s': %w", path, err)
		}
	}

	return nil
}

// validatePath validates a file path for security
func validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}

	// Clean the path
	cleanPath := filepath.Clean(path)

	// Reject path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path contains traversal: %s", path)
	}

	// Reject dangerous characters
	dangerousChars := []string{";", "&", "|", "$", "`", "(", ")", "<", ">", "\"", "'"}
	for _, char := range dangerousChars {
		if strings.Contains(cleanPath, char) {
			return fmt.Errorf("path contains dangerous character: %s", char)
		}
	}

	return nil
}

// validateMonitoringConfig validates monitoring configuration values
func validateMonitoringConfig(config *MonitoringConfig) error {
	// Validate log level
	validLogLevels := []string{"debug", "info", "warn", "error", "fatal"}
	isValidLogLevel := false
	for _, level := range validLogLevels {
		if config.LogLevel == level {
			isValidLogLevel = true
			break
		}
	}
	if !isValidLogLevel {
		return fmt.Errorf("invalid log level '%s', must be one of: %v", config.LogLevel, validLogLevels)
	}

	// Validate log format
	validLogFormats := []string{"json", "text"}
	isValidLogFormat := false
	for _, format := range validLogFormats {
		if config.LogFormat == format {
			isValidLogFormat = true
			break
		}
	}
	if !isValidLogFormat {
		return fmt.Errorf("invalid log format '%s', must be one of: %v", config.LogFormat, validLogFormats)
	}

	// Validate HTTP port
	if config.HTTPPort < 0 || config.HTTPPort > 65535 {
		return fmt.Errorf("HTTP port %d is not in valid range 0-65535", config.HTTPPort)
	}

	// Validate metrics path
	if config.MetricsPath != "" {
		if err := validatePath(config.MetricsPath); err != nil {
			return fmt.Errorf("invalid metrics path '%s': %w", config.MetricsPath, err)
		}
	}

	return nil
}
