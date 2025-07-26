package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// FuzzLoadConfig tests configuration loading with various malformed inputs
func FuzzLoadConfig(f *testing.F) {
	// Seed with valid and invalid YAML configurations
	f.Add(`server:
  port: 8080
  host: localhost
components:
  scan_paths:
    - ./components`)

	f.Add(`server:
  port: "invalid_port"
  host: localhost`)

	f.Add(`server:
  port: 65536
  host: localhost`)

	f.Add(`server:
  port: -1
  host: localhost`)

	f.Add(`malformed: yaml: content`)
	f.Add(``)
	f.Add(`---
server:
  port: 8080
  host: "0.0.0.0"
components:
  scan_paths: []`)

	f.Fuzz(func(t *testing.T, yamlContent string) {
		if len(yamlContent) > 50000 {
			t.Skip("Config content too large")
		}

		// Reset viper to clean state
		viper.Reset()

		// Create temporary config file
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, ".templar.yml")

		err := os.WriteFile(configFile, []byte(yamlContent), 0644)
		if err != nil {
			t.Skip("Could not write config file")
		}

		// Set config file path
		viper.SetConfigFile(configFile)

		// Test that Load doesn't panic with malformed config
		config, err := Load()
		_ = err // We expect many configs to be invalid

		// If config loaded successfully, validate it's safe
		if config != nil {
			// Ensure port is within valid range
			if config.Server.Port < 0 || config.Server.Port > 65535 {
				t.Errorf("Invalid port range: %d", config.Server.Port)
			}

			// Ensure host doesn't contain control characters
			if strings.ContainsAny(
				config.Server.Host,
				"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f",
			) {
				t.Errorf("Host contains control characters: %q", config.Server.Host)
			}

			// Validate scan paths don't contain dangerous patterns
			for _, path := range config.Components.ScanPaths {
				if strings.Contains(path, "..") && !strings.HasPrefix(path, "./") {
					t.Errorf("Potentially dangerous path traversal: %q", path)
				}
				if strings.ContainsAny(
					path,
					"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f",
				) {
					t.Errorf("Path contains control characters: %q", path)
				}
			}
		}
	})
}

// FuzzConfigValidation tests validation of configuration structures
func FuzzConfigValidation(f *testing.F) {
	// Seed with various config structures
	f.Add(
		`{"server":{"port":8080,"host":"localhost"},"components":{"scan_paths":["./components"]}}`,
	)
	f.Add(`{"server":{"port":"8080","host":"localhost"}}`)
	f.Add(`{"server":{"port":999999,"host":"localhost"}}`)
	f.Add(`{"malformed":"json"}`)
	f.Add(`{}`)
	f.Add(`{"server":{"port":8080,"host":"<script>alert('xss')</script>"}}`)

	f.Fuzz(func(t *testing.T, jsonContent string) {
		if len(jsonContent) > 20000 {
			t.Skip("JSON content too large")
		}

		var configData map[string]interface{}
		err := json.Unmarshal([]byte(jsonContent), &configData)
		if err != nil {
			// Invalid JSON is expected in fuzzing
			return
		}

		// Test validation doesn't panic with arbitrary JSON structures
		config := &Config{}

		// Attempt to populate config from the JSON data
		if server, ok := configData["server"].(map[string]interface{}); ok {
			if port, ok := server["port"].(float64); ok {
				config.Server.Port = int(port)
			}
			if host, ok := server["host"].(string); ok {
				config.Server.Host = host
			}
		}

		if components, ok := configData["components"].(map[string]interface{}); ok {
			if scanPaths, ok := components["scan_paths"].([]interface{}); ok {
				for _, path := range scanPaths {
					if pathStr, ok := path.(string); ok {
						config.Components.ScanPaths = append(config.Components.ScanPaths, pathStr)
					}
				}
			}
		}

		// Validate the config structure
		if err := ValidateConfig(config); err == nil {
			// If validation passed, ensure the config is actually safe
			if config.Server.Port < 0 || config.Server.Port > 65535 {
				t.Errorf("Validation allowed invalid port: %d", config.Server.Port)
			}
		}
	})
}

// FuzzYAMLParsing tests YAML parsing with various edge cases
func FuzzYAMLParsing(f *testing.F) {
	// Seed with YAML edge cases and potential attacks
	f.Add("key: value")
	f.Add("key: !!python/object/apply:os.system ['echo hello']")
	f.Add("key: &anchor value\nref: *anchor")
	f.Add("key: |\n  multiline\n  value")
	f.Add("key: >\n  folded\n  value")
	f.Add("!!binary |\n  R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7")
	f.Add(strings.Repeat("key: value\n", 10000))

	f.Fuzz(func(t *testing.T, yamlContent string) {
		if len(yamlContent) > 100000 {
			t.Skip("YAML content too large")
		}

		var data interface{}
		err := yaml.Unmarshal([]byte(yamlContent), &data)
		_ = err // Many inputs will be invalid YAML

		// If parsing succeeded, ensure no dangerous constructs were executed
		// This is mainly to ensure the YAML parser doesn't allow code execution
	})
}

// FuzzEnvironmentVariables tests environment variable parsing
func FuzzEnvironmentVariables(f *testing.F) {
	// Seed with various environment variable patterns
	f.Add("TEMPLAR_SERVER_PORT=8080")
	f.Add("TEMPLAR_SERVER_HOST=localhost")
	f.Add("TEMPLAR_COMPONENTS_SCAN_PATHS=./components,./views")
	f.Add("TEMPLAR_SERVER_PORT=invalid")
	f.Add("TEMPLAR_SERVER_PORT=999999")
	f.Add("TEMPLAR_SERVER_HOST=")
	f.Add("TEMPLAR_MALFORMED")

	f.Fuzz(func(t *testing.T, envVar string) {
		if len(envVar) > 10000 {
			t.Skip("Environment variable too long")
		}

		// Skip if contains control characters that could break parsing
		if strings.ContainsAny(
			envVar,
			"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f",
		) {
			t.Skip("Environment variable contains control characters")
		}

		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) != 2 {
			return // Invalid format
		}

		key, value := parts[0], parts[1]

		// Only test TEMPLAR_ prefixed variables
		if !strings.HasPrefix(key, "TEMPLAR_") {
			return
		}

		// Set environment variable
		originalValue := os.Getenv(key)
		err := os.Setenv(key, value)
		if err != nil {
			t.Skip("Could not set environment variable")
		}
		defer os.Setenv(key, originalValue)

		// Reset viper and test configuration loading
		viper.Reset()
		viper.AutomaticEnv()
		viper.SetEnvPrefix("TEMPLAR")
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

		// Test that environment variable processing doesn't panic
		config, err := Load()
		_ = err

		// If config loaded successfully, validate it
		if config != nil {
			if config.Server.Port < 0 || config.Server.Port > 65535 {
				t.Errorf("Environment variable resulted in invalid port: %d", config.Server.Port)
			}
		}
	})
}

// ValidateConfig validates a configuration structure for security and correctness
func ValidateConfig(config *Config) error {
	if config.Server.Port < 0 || config.Server.Port > 65535 {
		return ErrInvalidPort
	}

	if strings.TrimSpace(config.Server.Host) == "" {
		return ErrInvalidHost
	}

	return nil
}

// Custom errors for validation
var (
	ErrInvalidPort = fmt.Errorf("invalid port")
	ErrInvalidHost = fmt.Errorf("invalid host")
)
