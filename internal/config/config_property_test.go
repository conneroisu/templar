//go:build property
// +build property

package config

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestConfigurationProperties tests configuration loading and validation properties
func TestConfigurationProperties(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: Valid configurations should always parse without error
	properties.Property("valid config parsing", prop.ForAll(
		func(port int, host string, scanPaths []string) bool {
			if port < 1 || port > 65535 {
				return true // Skip invalid ports
			}
			if host == "" || strings.ContainsAny(host, " \t\n\r") {
				return true // Skip invalid hosts
			}

			// Filter scan paths to be valid
			validPaths := make([]string, 0, len(scanPaths))
			for _, path := range scanPaths {
				if path != "" && !strings.ContainsAny(path, "\x00\n\r") {
					validPaths = append(validPaths, path)
				}
			}

			if len(validPaths) == 0 {
				validPaths = []string{"./components"} // Default path
			}

			// Create config
			cfg := &Config{
				Server: ServerConfig{
					Port: port,
					Host: host,
				},
				Components: ComponentsConfig{
					ScanPaths: validPaths,
				},
			}

			// Validate config
			err := validateConfig(cfg)

			// Should be valid if inputs are reasonable
			return err == nil
		},
		gen.IntRange(1000, 9999),                             // Valid port range
		gen.RegexMatch(`^[a-zA-Z0-9.-]+$`),                   // Valid hostname
		gen.SliceOfN(5, gen.RegexMatch(`^[a-zA-Z0-9_./]+$`)), // Valid paths
	))

	// Property: Path validation should be consistent
	properties.Property("path validation consistency", prop.ForAll(
		func(path string) bool {
			if path == "" {
				return true
			}

			// Validate path multiple times
			valid1 := isValidScanPath(path)
			valid2 := isValidScanPath(path)
			valid3 := isValidScanPath(path)

			// Should return same result every time
			return valid1 == valid2 && valid2 == valid3
		},
		gen.OneConstOf("./components", "../components", "/etc/passwd", "components", ".", ""),
	))

	// Property: Default config should always be valid
	properties.Property("default config validity", prop.ForAll(
		func() bool {
			defaultCfg := getDefaultConfig()
			err := validateConfig(defaultCfg)
			return err == nil
		},
	))

	properties.TestingRun(t)
}

// TestServerConfigProperties tests server configuration properties
func TestServerConfigProperties(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: Port validation should reject invalid ranges
	properties.Property("port validation", prop.ForAll(
		func(port int) bool {
			cfg := &Config{
				Server: ServerConfig{
					Port: port,
					Host: "localhost",
				},
			}

			err := validateServerConfig(&cfg.Server)

			// Should be valid only for valid port range
			if port >= 1 && port <= 65535 {
				return err == nil
			} else {
				return err != nil
			}
		},
		gen.IntRange(-1000, 70000), // Include invalid ranges
	))

	// Property: Host validation should handle edge cases
	properties.Property("host validation", prop.ForAll(
		func(host string) bool {
			cfg := &Config{
				Server: ServerConfig{
					Port: 8080,
					Host: host,
				},
			}

			err := validateServerConfig(&cfg.Server)

			// Empty or whitespace-only hosts should be invalid
			if strings.TrimSpace(host) == "" || strings.ContainsAny(host, " \t\n\r") {
				return err != nil
			}

			// Hosts with dangerous characters should be invalid
			if strings.ContainsAny(host, ";|&`$()") {
				return err != nil
			}

			return err == nil
		},
		gen.OneConstOf("localhost", "127.0.0.1", "0.0.0.0", "", " ", "host;rm -rf /", "host\n"),
	))

	properties.TestingRun(t)
}

// TestComponentsConfigProperties tests components configuration properties
func TestComponentsConfigProperties(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: Scan paths should be normalized consistently
	properties.Property("scan path normalization", prop.ForAll(
		func(paths []string) bool {
			if len(paths) == 0 {
				return true
			}

			// Filter valid paths
			validPaths := make([]string, 0, len(paths))
			for _, path := range paths {
				if path != "" && !strings.ContainsAny(path, "\x00") {
					validPaths = append(validPaths, path)
				}
			}

			if len(validPaths) == 0 {
				return true
			}

			cfg := &Config{
				Components: ComponentsConfig{
					ScanPaths: validPaths,
				},
			}

			// Normalize paths
			normalizedPaths := normalizeScanPaths(cfg.Components.ScanPaths)

			// All normalized paths should be clean
			for _, path := range normalizedPaths {
				if filepath.Clean(path) != path {
					return false
				}
			}

			return true
		},
		gen.SliceOfN(5, gen.OneConstOf("./components", "../components", "components/", "./components/../other")),
	))

	// Property: Exclude patterns should be valid regex
	properties.Property("exclude pattern validation", prop.ForAll(
		func(patterns []string) bool {
			cfg := &Config{
				Components: ComponentsConfig{
					ExcludePatterns: patterns,
				},
			}

			err := validateComponentsConfig(&cfg.Components)

			// Should validate all patterns
			for _, pattern := range patterns {
				if !isValidGlobPattern(pattern) {
					return err != nil
				}
			}

			return err == nil
		},
		gen.SliceOfN(3, gen.OneConstOf("*.templ", "*_test.templ", "**/*.bak", "[invalid", "*.{templ,go}")),
	))

	properties.TestingRun(t)
}

// TestBuildConfigProperties tests build configuration properties
func TestBuildConfigProperties(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: Build commands should be validated for security
	properties.Property("build command validation", prop.ForAll(
		func(command string) bool {
			cfg := &Config{
				Build: BuildConfig{
					Command: command,
				},
			}

			err := validateBuildConfig(&cfg.Build)

			// Commands with dangerous characters should be rejected
			if strings.ContainsAny(command, ";|&`$()") {
				return err != nil
			}

			// Empty commands should be rejected
			if strings.TrimSpace(command) == "" {
				return err != nil
			}

			return err == nil
		},
		gen.OneConstOf("templ generate", "go build", "make build", "rm -rf /", "cmd; rm -rf /", "", "  "),
	))

	// Property: Watch patterns should be valid
	properties.Property("watch pattern validation", prop.ForAll(
		func(patterns []string) bool {
			cfg := &Config{
				Build: BuildConfig{
					WatchPatterns: patterns,
				},
			}

			err := validateBuildConfig(&cfg.Build)

			// All patterns should be valid globs
			for _, pattern := range patterns {
				if pattern != "" && !isValidGlobPattern(pattern) {
					return err != nil
				}
			}

			return err == nil
		},
		gen.SliceOfN(3, gen.OneConstOf("**/*.templ", "*.go", "**/*.{templ,go}", "[invalid", "components/**")),
	))

	properties.TestingRun(t)
}

// TestConfigMergingProperties tests configuration merging behavior
func TestConfigMergingProperties(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: Merging with default should preserve non-zero values
	properties.Property("config merging preservation", prop.ForAll(
		func(port int, host string) bool {
			if port <= 0 || port > 65535 || host == "" {
				return true // Skip invalid values
			}

			defaultCfg := getDefaultConfig()
			userCfg := &Config{
				Server: ServerConfig{
					Port: port,
					Host: host,
				},
			}

			merged := mergeConfigs(defaultCfg, userCfg)

			// User values should be preserved
			return merged.Server.Port == port && merged.Server.Host == host
		},
		gen.IntRange(1000, 9999),
		gen.RegexMatch(`^[a-zA-Z0-9.-]+$`),
	))

	// Property: Merging should be commutative for non-conflicting fields
	properties.Property("config merging commutativity", prop.ForAll(
		func(port1, port2 int) bool {
			if port1 <= 0 || port1 > 65535 || port2 <= 0 || port2 > 65535 {
				return true
			}

			cfg1 := &Config{
				Server: ServerConfig{Port: port1},
			}
			cfg2 := &Config{
				Build: BuildConfig{Command: "templ generate"},
			}

			merged1 := mergeConfigs(cfg1, cfg2)
			merged2 := mergeConfigs(cfg2, cfg1)

			// Non-conflicting fields should be the same
			return merged1.Build.Command == merged2.Build.Command
		},
		gen.IntRange(1000, 9999),
		gen.IntRange(1000, 9999),
	))

	properties.TestingRun(t)
}

// Helper functions for property testing

func isValidScanPath(path string) bool {
	if path == "" {
		return false
	}
	if strings.ContainsAny(path, "\x00\n\r") {
		return false
	}
	if strings.Contains(path, "..") {
		return false
	}
	return true
}

func isValidGlobPattern(pattern string) bool {
	if pattern == "" {
		return true
	}
	// Simple glob pattern validation
	// In real implementation, would use filepath.Match or similar
	if strings.Contains(pattern, "[") && !strings.Contains(pattern, "]") {
		return false
	}
	return !strings.ContainsAny(pattern, "\x00\n\r")
}

func validateConfig(cfg *Config) error {
	if err := validateServerConfig(&cfg.Server); err != nil {
		return err
	}
	if err := validateComponentsConfig(&cfg.Components); err != nil {
		return err
	}
	if err := validateBuildConfig(&cfg.Build); err != nil {
		return err
	}
	return nil
}

func validateServerConfig(cfg *ServerConfig) error {
	if cfg.Port < 1 || cfg.Port > 65535 {
		return fmt.Errorf("invalid port: %d", cfg.Port)
	}
	if strings.TrimSpace(cfg.Host) == "" {
		return fmt.Errorf("empty host")
	}
	if strings.ContainsAny(cfg.Host, " \t\n\r;|&`$()") {
		return fmt.Errorf("invalid host: %s", cfg.Host)
	}
	return nil
}

func validateComponentsConfig(cfg *ComponentsConfig) error {
	if len(cfg.ScanPaths) == 0 {
		return fmt.Errorf("no scan paths specified")
	}
	for _, pattern := range cfg.ExcludePatterns {
		if !isValidGlobPattern(pattern) {
			return fmt.Errorf("invalid exclude pattern: %s", pattern)
		}
	}
	return nil
}

func validateBuildConfig(cfg *BuildConfig) error {
	if strings.TrimSpace(cfg.Command) == "" {
		return fmt.Errorf("empty build command")
	}
	if strings.ContainsAny(cfg.Command, ";|&`$()") {
		return fmt.Errorf("dangerous build command: %s", cfg.Command)
	}
	for _, pattern := range cfg.WatchPatterns {
		if pattern != "" && !isValidGlobPattern(pattern) {
			return fmt.Errorf("invalid watch pattern: %s", pattern)
		}
	}
	return nil
}

func getDefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port: 8080,
			Host: "localhost",
		},
		Components: ComponentsConfig{
			ScanPaths: []string{"./components"},
		},
		Build: BuildConfig{
			Command: "templ generate",
		},
	}
}

func normalizeScanPaths(paths []string) []string {
	normalized := make([]string, len(paths))
	for i, path := range paths {
		normalized[i] = filepath.Clean(path)
	}
	return normalized
}

func mergeConfigs(base, override *Config) *Config {
	result := *base

	if override.Server.Port != 0 {
		result.Server.Port = override.Server.Port
	}
	if override.Server.Host != "" {
		result.Server.Host = override.Server.Host
	}
	if len(override.Components.ScanPaths) > 0 {
		result.Components.ScanPaths = override.Components.ScanPaths
	}
	if override.Build.Command != "" {
		result.Build.Command = override.Build.Command
	}

	return &result
}
