package monitoring

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// MonitoringConfiguration represents the complete monitoring configuration
type MonitoringConfiguration struct {
	// Core monitoring settings
	Enabled bool `yaml:"enabled" json:"enabled" env:"TEMPLAR_MONITORING_ENABLED"`

	// Logging configuration
	Logging LoggingConfig `yaml:"logging" json:"logging"`

	// Metrics configuration
	Metrics MetricsConfig `yaml:"metrics" json:"metrics"`

	// Health monitoring configuration
	Health HealthConfig `yaml:"health" json:"health"`

	// Performance monitoring configuration
	Performance PerformanceConfig `yaml:"performance" json:"performance"`

	// Alerting configuration
	Alerting AlertingConfig `yaml:"alerting" json:"alerting"`

	// HTTP server configuration
	HTTP HTTPConfig `yaml:"http" json:"http"`

	// Templar-specific configuration
	Templar TemplarConfig `yaml:"templar" json:"templar"`
}

// LoggingConfig configures the logging system
type LoggingConfig struct {
	Level           string        `yaml:"level" json:"level" env:"TEMPLAR_LOG_LEVEL"`
	Format          string        `yaml:"format" json:"format" env:"TEMPLAR_LOG_FORMAT"`
	Output          string        `yaml:"output" json:"output" env:"TEMPLAR_LOG_OUTPUT"`
	Structured      bool          `yaml:"structured" json:"structured" env:"TEMPLAR_LOG_STRUCTURED"`
	SanitizeSecrets bool          `yaml:"sanitize_secrets" json:"sanitize_secrets"`
	MaxFieldLength  int           `yaml:"max_field_length" json:"max_field_length"`
	RotationSize    string        `yaml:"rotation_size" json:"rotation_size"`
	RotationAge     time.Duration `yaml:"rotation_age" json:"rotation_age"`
	MaxBackups      int           `yaml:"max_backups" json:"max_backups"`
	CompressBackups bool          `yaml:"compress_backups" json:"compress_backups"`
}

// MetricsConfig configures the metrics collection system
type MetricsConfig struct {
	Enabled        bool          `yaml:"enabled" json:"enabled" env:"TEMPLAR_METRICS_ENABLED"`
	OutputPath     string        `yaml:"output_path" json:"output_path" env:"TEMPLAR_METRICS_OUTPUT_PATH"`
	FlushInterval  time.Duration `yaml:"flush_interval" json:"flush_interval"`
	Prefix         string        `yaml:"prefix" json:"prefix" env:"TEMPLAR_METRICS_PREFIX"`
	MaxSeries      int           `yaml:"max_series" json:"max_series"`
	RetentionHours int           `yaml:"retention_hours" json:"retention_hours"`

	// Histogram configuration
	HistogramBuckets []float64 `yaml:"histogram_buckets" json:"histogram_buckets"`

	// System metrics
	SystemMetrics SystemMetricsConfig `yaml:"system_metrics" json:"system_metrics"`
}

// SystemMetricsConfig configures system metric collection
type SystemMetricsConfig struct {
	Enabled            bool          `yaml:"enabled" json:"enabled"`
	CollectionInterval time.Duration `yaml:"collection_interval" json:"collection_interval"`
	IncludeMemory      bool          `yaml:"include_memory" json:"include_memory"`
	IncludeGC          bool          `yaml:"include_gc" json:"include_gc"`
	IncludeGoroutines  bool          `yaml:"include_goroutines" json:"include_goroutines"`
	IncludeCPU         bool          `yaml:"include_cpu" json:"include_cpu"`
	IncludeNetwork     bool          `yaml:"include_network" json:"include_network"`
}

// HealthConfig configures health monitoring
type HealthConfig struct {
	Enabled          bool          `yaml:"enabled" json:"enabled" env:"TEMPLAR_HEALTH_ENABLED"`
	CheckInterval    time.Duration `yaml:"check_interval" json:"check_interval"`
	CheckTimeout     time.Duration `yaml:"check_timeout" json:"check_timeout"`
	FailureThreshold int           `yaml:"failure_threshold" json:"failure_threshold"`

	// Built-in health checks
	Checks HealthChecksConfig `yaml:"checks" json:"checks"`
}

// HealthChecksConfig configures individual health checks
type HealthChecksConfig struct {
	Filesystem        HealthCheckConfig `yaml:"filesystem" json:"filesystem"`
	Memory            HealthCheckConfig `yaml:"memory" json:"memory"`
	Goroutines        HealthCheckConfig `yaml:"goroutines" json:"goroutines"`
	ComponentRegistry HealthCheckConfig `yaml:"component_registry" json:"component_registry"`
	TemplBinary       HealthCheckConfig `yaml:"templ_binary" json:"templ_binary"`
	CacheDirectory    HealthCheckConfig `yaml:"cache_directory" json:"cache_directory"`
}

// HealthCheckConfig configures a specific health check
type HealthCheckConfig struct {
	Enabled  bool          `yaml:"enabled" json:"enabled"`
	Critical bool          `yaml:"critical" json:"critical"`
	Timeout  time.Duration `yaml:"timeout" json:"timeout"`
	Interval time.Duration `yaml:"interval" json:"interval"`

	// Check-specific parameters
	Parameters map[string]interface{} `yaml:"parameters" json:"parameters"`
}

// PerformanceConfig configures performance monitoring
type PerformanceConfig struct {
	Enabled          bool          `yaml:"enabled" json:"enabled" env:"TEMPLAR_PERFORMANCE_ENABLED"`
	SampleRate       float64       `yaml:"sample_rate" json:"sample_rate"`
	MaxOperations    int           `yaml:"max_operations" json:"max_operations"`
	TrackDurations   bool          `yaml:"track_durations" json:"track_durations"`
	TrackPercentiles bool          `yaml:"track_percentiles" json:"track_percentiles"`
	UpdateInterval   time.Duration `yaml:"update_interval" json:"update_interval"`

	// Operation-specific configuration
	Operations map[string]OperationConfig `yaml:"operations" json:"operations"`
}

// OperationConfig configures monitoring for specific operations
type OperationConfig struct {
	Enabled    bool          `yaml:"enabled" json:"enabled"`
	SampleRate float64       `yaml:"sample_rate" json:"sample_rate"`
	Timeout    time.Duration `yaml:"timeout" json:"timeout"`
}

// AlertingConfig configures the alerting system
type AlertingConfig struct {
	Enabled  bool          `yaml:"enabled" json:"enabled" env:"TEMPLAR_ALERTING_ENABLED"`
	Cooldown time.Duration `yaml:"cooldown" json:"cooldown"`

	// Alert rules
	Rules []AlertRuleConfig `yaml:"rules" json:"rules"`

	// Alert channels
	Channels AlertChannelsConfig `yaml:"channels" json:"channels"`

	// Default thresholds
	Thresholds AlertThresholdsConfig `yaml:"thresholds" json:"thresholds"`
}

// AlertRuleConfig configures an alert rule
type AlertRuleConfig struct {
	Name      string            `yaml:"name" json:"name"`
	Component string            `yaml:"component" json:"component"`
	Metric    string            `yaml:"metric" json:"metric"`
	Condition string            `yaml:"condition" json:"condition"`
	Threshold float64           `yaml:"threshold" json:"threshold"`
	Duration  time.Duration     `yaml:"duration" json:"duration"`
	Level     string            `yaml:"level" json:"level"`
	Message   string            `yaml:"message" json:"message"`
	Labels    map[string]string `yaml:"labels" json:"labels"`
	Enabled   bool              `yaml:"enabled" json:"enabled"`
	Cooldown  time.Duration     `yaml:"cooldown" json:"cooldown"`
}

// AlertChannelsConfig configures alert delivery channels
type AlertChannelsConfig struct {
	Log     LogChannelConfig     `yaml:"log" json:"log"`
	Webhook WebhookChannelConfig `yaml:"webhook" json:"webhook"`
	Email   EmailChannelConfig   `yaml:"email" json:"email"`
	Slack   SlackChannelConfig   `yaml:"slack" json:"slack"`
}

// LogChannelConfig configures log-based alerting
type LogChannelConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Level   string `yaml:"level" json:"level"`
}

// WebhookChannelConfig configures webhook alerting
type WebhookChannelConfig struct {
	Enabled bool              `yaml:"enabled" json:"enabled"`
	URL     string            `yaml:"url" json:"url" env:"TEMPLAR_WEBHOOK_URL"`
	Timeout time.Duration     `yaml:"timeout" json:"timeout"`
	Headers map[string]string `yaml:"headers" json:"headers"`
}

// EmailChannelConfig configures email alerting
type EmailChannelConfig struct {
	Enabled  bool     `yaml:"enabled" json:"enabled"`
	SMTPHost string   `yaml:"smtp_host" json:"smtp_host" env:"TEMPLAR_SMTP_HOST"`
	SMTPPort int      `yaml:"smtp_port" json:"smtp_port" env:"TEMPLAR_SMTP_PORT"`
	Username string   `yaml:"username" json:"username" env:"TEMPLAR_SMTP_USERNAME"`
	Password string   `yaml:"password" json:"password" env:"TEMPLAR_SMTP_PASSWORD"`
	From     string   `yaml:"from" json:"from" env:"TEMPLAR_EMAIL_FROM"`
	To       []string `yaml:"to" json:"to"`
	Subject  string   `yaml:"subject" json:"subject"`
}

// SlackChannelConfig configures Slack alerting
type SlackChannelConfig struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	WebhookURL string `yaml:"webhook_url" json:"webhook_url" env:"TEMPLAR_SLACK_WEBHOOK_URL"`
	Channel    string `yaml:"channel" json:"channel" env:"TEMPLAR_SLACK_CHANNEL"`
	Username   string `yaml:"username" json:"username"`
	IconEmoji  string `yaml:"icon_emoji" json:"icon_emoji"`
}

// AlertThresholdsConfig defines default alert thresholds
type AlertThresholdsConfig struct {
	ErrorRate           float64       `yaml:"error_rate" json:"error_rate"`
	ResponseTime        time.Duration `yaml:"response_time" json:"response_time"`
	MemoryUsage         int64         `yaml:"memory_usage" json:"memory_usage"`
	GoroutineCount      int           `yaml:"goroutine_count" json:"goroutine_count"`
	DiskUsage           float64       `yaml:"disk_usage" json:"disk_usage"`
	UnhealthyComponents int           `yaml:"unhealthy_components" json:"unhealthy_components"`
	BuildFailures       int           `yaml:"build_failures" json:"build_failures"`
	GCPauseTime         time.Duration `yaml:"gc_pause_time" json:"gc_pause_time"`
}

// HTTPConfig configures the monitoring HTTP server
type HTTPConfig struct {
	Enabled      bool          `yaml:"enabled" json:"enabled" env:"TEMPLAR_HTTP_MONITORING_ENABLED"`
	Host         string        `yaml:"host" json:"host" env:"TEMPLAR_HTTP_HOST"`
	Port         int           `yaml:"port" json:"port" env:"TEMPLAR_HTTP_PORT"`
	ReadTimeout  time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout" json:"idle_timeout"`

	// TLS configuration
	TLS HTTPTLSConfig `yaml:"tls" json:"tls"`

	// Authentication
	Auth HTTPAuthConfig `yaml:"auth" json:"auth"`

	// CORS configuration
	CORS HTTPCORSConfig `yaml:"cors" json:"cors"`
}

// HTTPTLSConfig configures TLS for the HTTP server
type HTTPTLSConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	CertFile string `yaml:"cert_file" json:"cert_file" env:"TEMPLAR_TLS_CERT_FILE"`
	KeyFile  string `yaml:"key_file" json:"key_file" env:"TEMPLAR_TLS_KEY_FILE"`
}

// HTTPAuthConfig configures authentication for the HTTP server
type HTTPAuthConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	Type     string `yaml:"type" json:"type"` // "basic", "bearer", "api_key"
	Username string `yaml:"username" json:"username" env:"TEMPLAR_AUTH_USERNAME"`
	Password string `yaml:"password" json:"password" env:"TEMPLAR_AUTH_PASSWORD"`
	APIKey   string `yaml:"api_key" json:"api_key" env:"TEMPLAR_AUTH_API_KEY"`
}

// HTTPCORSConfig configures CORS for the HTTP server
type HTTPCORSConfig struct {
	Enabled          bool     `yaml:"enabled" json:"enabled"`
	AllowedOrigins   []string `yaml:"allowed_origins" json:"allowed_origins"`
	AllowedMethods   []string `yaml:"allowed_methods" json:"allowed_methods"`
	AllowedHeaders   []string `yaml:"allowed_headers" json:"allowed_headers"`
	AllowCredentials bool     `yaml:"allow_credentials" json:"allow_credentials"`
}

// DefaultMonitoringConfiguration returns default monitoring configuration
func DefaultMonitoringConfiguration() *MonitoringConfiguration {
	return &MonitoringConfiguration{
		Enabled: true,

		Logging: LoggingConfig{
			Level:           "info",
			Format:          "json",
			Output:          "stdout",
			Structured:      true,
			SanitizeSecrets: true,
			MaxFieldLength:  1000,
			RotationSize:    "100MB",
			RotationAge:     24 * time.Hour,
			MaxBackups:      7,
			CompressBackups: true,
		},

		Metrics: MetricsConfig{
			Enabled:        true,
			OutputPath:     "./logs/metrics.json",
			FlushInterval:  30 * time.Second,
			Prefix:         "templar",
			MaxSeries:      10000,
			RetentionHours: 24,
			HistogramBuckets: []float64{
				0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
			},
			SystemMetrics: SystemMetricsConfig{
				Enabled:            true,
				CollectionInterval: 10 * time.Second,
				IncludeMemory:      true,
				IncludeGC:          true,
				IncludeGoroutines:  true,
				IncludeCPU:         true,
				IncludeNetwork:     false,
			},
		},

		Health: HealthConfig{
			Enabled:          true,
			CheckInterval:    30 * time.Second,
			CheckTimeout:     10 * time.Second,
			FailureThreshold: 3,
			Checks: HealthChecksConfig{
				Filesystem: HealthCheckConfig{
					Enabled:  true,
					Critical: true,
					Timeout:  5 * time.Second,
				},
				Memory: HealthCheckConfig{
					Enabled:  true,
					Critical: true,
					Timeout:  1 * time.Second,
				},
				Goroutines: HealthCheckConfig{
					Enabled:  true,
					Critical: false,
					Timeout:  1 * time.Second,
				},
				ComponentRegistry: HealthCheckConfig{
					Enabled:  true,
					Critical: true,
					Timeout:  2 * time.Second,
				},
				TemplBinary: HealthCheckConfig{
					Enabled:  true,
					Critical: true,
					Timeout:  2 * time.Second,
				},
				CacheDirectory: HealthCheckConfig{
					Enabled:  true,
					Critical: false,
					Timeout:  2 * time.Second,
				},
			},
		},

		Performance: PerformanceConfig{
			Enabled:          true,
			SampleRate:       1.0,
			MaxOperations:    1000,
			TrackDurations:   true,
			TrackPercentiles: true,
			UpdateInterval:   10 * time.Second,
			Operations:       make(map[string]OperationConfig),
		},

		Alerting: AlertingConfig{
			Enabled:  false,
			Cooldown: 5 * time.Minute,
			Rules:    make([]AlertRuleConfig, 0),
			Channels: AlertChannelsConfig{
				Log: LogChannelConfig{
					Enabled: true,
					Level:   "error",
				},
			},
			Thresholds: AlertThresholdsConfig{
				ErrorRate:           0.1,
				ResponseTime:        5 * time.Second,
				MemoryUsage:         1024 * 1024 * 1024, // 1GB
				GoroutineCount:      1000,
				DiskUsage:           0.9, // 90%
				UnhealthyComponents: 1,
				BuildFailures:       5,
				GCPauseTime:         100 * time.Millisecond,
			},
		},

		HTTP: HTTPConfig{
			Enabled:      true,
			Host:         "localhost",
			Port:         8081,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
			TLS: HTTPTLSConfig{
				Enabled: false,
			},
			Auth: HTTPAuthConfig{
				Enabled: false,
			},
			CORS: HTTPCORSConfig{
				Enabled:        false,
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowedHeaders: []string{"Content-Type", "Authorization"},
			},
		},

		Templar: TemplarConfig{
			ComponentPaths: []string{
				"./components",
				"./views",
				"./layouts",
			},
			CacheDirectory:   ".templar/cache",
			BuildCommand:     "templ generate",
			WatchPatterns:    []string{"**/*.templ", "**/*.go"},
			PreviewPort:      8080,
			WebSocketEnabled: true,
		},
	}
}

// LoadConfiguration loads monitoring configuration from file and environment
func LoadConfiguration(configPath string) (*MonitoringConfiguration, error) {
	config := DefaultMonitoringConfiguration()

	// Load from file if provided
	if configPath != "" {
		if err := loadConfigFromFile(config, configPath); err != nil {
			return nil, fmt.Errorf("failed to load config from file: %w", err)
		}
	}

	// Override with environment variables
	if err := loadConfigFromEnv(config); err != nil {
		return nil, fmt.Errorf("failed to load config from environment: %w", err)
	}

	// Validate configuration
	if err := validateConfiguration(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// loadConfigFromFile loads configuration from YAML file
func loadConfigFromFile(config *MonitoringConfiguration, path string) error {
	// Clean and validate path
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("invalid config path (contains path traversal): %s", path)
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return nil
}

// loadConfigFromEnv loads configuration from environment variables
func loadConfigFromEnv(config *MonitoringConfiguration) error {
	// Use reflection or manual mapping to load environment variables
	// This is a simplified implementation

	if val := os.Getenv("TEMPLAR_MONITORING_ENABLED"); val != "" {
		config.Enabled = val == "true"
	}

	if val := os.Getenv("TEMPLAR_LOG_LEVEL"); val != "" {
		config.Logging.Level = val
	}

	if val := os.Getenv("TEMPLAR_LOG_FORMAT"); val != "" {
		config.Logging.Format = val
	}

	if val := os.Getenv("TEMPLAR_METRICS_ENABLED"); val != "" {
		config.Metrics.Enabled = val == "true"
	}

	if val := os.Getenv("TEMPLAR_METRICS_OUTPUT_PATH"); val != "" {
		config.Metrics.OutputPath = val
	}

	if val := os.Getenv("TEMPLAR_HTTP_PORT"); val != "" {
		if port := parseInt(val); port > 0 {
			config.HTTP.Port = port
		}
	}

	return nil
}

// validateConfiguration validates the configuration
func validateConfiguration(config *MonitoringConfiguration) error {
	// Validate log level
	validLogLevels := []string{"debug", "info", "warn", "error", "fatal"}
	if !contains(validLogLevels, config.Logging.Level) {
		return fmt.Errorf("invalid log level: %s", config.Logging.Level)
	}

	// Validate log format
	validLogFormats := []string{"json", "text"}
	if !contains(validLogFormats, config.Logging.Format) {
		return fmt.Errorf("invalid log format: %s", config.Logging.Format)
	}

	// Validate HTTP port
	if config.HTTP.Port < 1 || config.HTTP.Port > 65535 {
		return fmt.Errorf("invalid HTTP port: %d", config.HTTP.Port)
	}

	// Validate metrics output path
	if config.Metrics.Enabled && config.Metrics.OutputPath != "" {
		dir := filepath.Dir(config.Metrics.OutputPath)
		if strings.Contains(dir, "..") {
			return fmt.Errorf(
				"invalid metrics output path (contains path traversal): %s",
				config.Metrics.OutputPath,
			)
		}
	}

	// Validate performance sample rate
	if config.Performance.SampleRate < 0 || config.Performance.SampleRate > 1 {
		return fmt.Errorf("invalid performance sample rate: %f", config.Performance.SampleRate)
	}

	// Validate alert thresholds
	if config.Alerting.Thresholds.ErrorRate < 0 || config.Alerting.Thresholds.ErrorRate > 1 {
		return fmt.Errorf("invalid error rate threshold: %f", config.Alerting.Thresholds.ErrorRate)
	}

	return nil
}

// SaveConfiguration saves configuration to YAML file
func SaveConfiguration(config *MonitoringConfiguration, path string) error {
	// Clean and validate path
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("invalid config path (contains path traversal): %s", path)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(cleanPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	// Write to file
	if err := os.WriteFile(cleanPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Utility functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func parseInt(s string) int {
	var result int
	for _, r := range s {
		if r >= '0' && r <= '9' {
			result = result*10 + int(r-'0')
		} else {
			return 0
		}
	}
	return result
}
