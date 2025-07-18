package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Server      ServerConfig      `yaml:"server"`
	Build       BuildConfig       `yaml:"build"`
	Preview     PreviewConfig     `yaml:"preview"`
	Components  ComponentsConfig  `yaml:"components"`
	Development DevelopmentConfig `yaml:"development"`
	TargetFiles []string          `yaml:"-"` // CLI arguments, not from config file
}

type ServerConfig struct {
	Port       int      `yaml:"port"`
	Host       string   `yaml:"host"`
	Open       bool     `yaml:"open"`
	NoOpen     bool     `yaml:"no-open"`
	Middleware []string `yaml:"middleware"`
}

type BuildConfig struct {
	Command   string   `yaml:"command"`
	Watch     []string `yaml:"watch"`
	Ignore    []string `yaml:"ignore"`
	CacheDir  string   `yaml:"cache_dir"`
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

func Load() (*Config, error) {
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}
	
	// If scan paths are empty, use defaults
	if len(config.Components.ScanPaths) == 0 {
		config.Components.ScanPaths = []string{"./components", "./views", "./examples"}
	}
	
	// Override no-open if explicitly set via flag
	if viper.IsSet("server.no-open") && viper.GetBool("server.no-open") {
		config.Server.Open = false
	}
	
	return &config, nil
}