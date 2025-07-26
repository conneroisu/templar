package testutils

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/types"
	"github.com/stretchr/testify/require"
)

// CreateTempProject creates a temporary project structure for testing
func CreateTempProject(t *testing.T) string {
	tempDir := t.TempDir()

	// Create standard directory structure
	dirs := []string{
		"components",
		"examples",
		"static",
		".templar/cache",
		".templar/render",
	}

	for _, dir := range dirs {
		err := os.MkdirAll(filepath.Join(tempDir, dir), 0755)
		require.NoError(t, err)
	}

	return tempDir
}

// CreateTestComponent creates a test component file
func CreateTestComponent(t *testing.T, dir, name, content string) string {
	componentPath := filepath.Join(dir, name+".templ")
	err := os.WriteFile(componentPath, []byte(content), 0644)
	require.NoError(t, err)
	return componentPath
}

// CreateTestConfig creates a test configuration
func CreateTestConfig(projectDir string) *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
			Open: false,
		},
		Components: config.ComponentsConfig{
			ScanPaths: []string{
				filepath.Join(projectDir, "components"),
				filepath.Join(projectDir, "examples"),
			},
			ExcludePatterns: []string{"*_test.templ"},
		},
		Build: config.BuildConfig{
			Command:  "echo 'test build'",
			Watch:    []string{"**/*.templ"},
			Ignore:   []string{"node_modules", ".git"},
			CacheDir: filepath.Join(projectDir, ".templar", "cache"),
		},
		Development: config.DevelopmentConfig{
			HotReload:    true,
			CSSInjection: true,
			ErrorOverlay: true,
		},
	}
}

// CreateTestRegistry creates a registry with sample components
func CreateTestRegistry() *registry.ComponentRegistry {
	reg := registry.NewComponentRegistry()

	components := []*types.ComponentInfo{
		{
			Name:     "Button",
			Package:  "components",
			FilePath: "/test/button.templ",
			Parameters: []types.ParameterInfo{
				{Name: "text", Type: "string", Optional: false},
				{Name: "variant", Type: "string", Optional: true},
			},
			Imports:      []string{"context"},
			LastMod:      time.Now(),
			Hash:         "buttonhash123",
			Dependencies: []string{},
		},
		{
			Name:     "Card",
			Package:  "components",
			FilePath: "/test/card.templ",
			Parameters: []types.ParameterInfo{
				{Name: "title", Type: "string", Optional: false},
				{Name: "content", Type: "string", Optional: false},
				{Name: "image", Type: "string", Optional: true},
			},
			Imports:      []string{"context"},
			LastMod:      time.Now(),
			Hash:         "cardhash456",
			Dependencies: []string{},
		},
		{
			Name:     "Nav",
			Package:  "components",
			FilePath: "/test/nav.templ",
			Parameters: []types.ParameterInfo{
				{Name: "items", Type: "[]string", Optional: false},
				{Name: "active", Type: "string", Optional: true},
			},
			Imports:      []string{"context"},
			LastMod:      time.Now(),
			Hash:         "navhash789",
			Dependencies: []string{},
		},
	}

	for _, comp := range components {
		reg.Register(comp)
	}

	return reg
}

// StandardTemplContent provides standard templ component templates for testing
var StandardTemplContent = map[string]string{
	"Button": `package components

templ Button(text string, variant string) {
	<button class={"btn", "btn-" + variant}>
		{text}
	</button>
}`,
	"Card": `package components

templ Card(title string, content string) {
	<div class="card">
		<div class="card-header">
			<h3>{title}</h3>
		</div>
		<div class="card-body">
			<p>{content}</p>
		</div>
	</div>
}`,
	"Nav": `package components

templ Nav(items []string, active string) {
	<nav class="navbar">
		for _, item := range items {
			<a href="#" class={item == active ? "active" : ""}>
				{item}
			</a>
		}
	</nav>
}`,
	"Layout": `package components

templ Layout(title string, content templ.Component) {
	<!DOCTYPE html>
	<html>
		<head>
			<title>{title}</title>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
		</head>
		<body>
			@content
		</body>
	</html>
}`,
}

// SecurityTestCases provides common security test vectors
var SecurityTestCases = struct {
	PathTraversal    []string
	CommandInjection []string
	ScriptInjection  []string
	SQLInjection     []string
}{
	PathTraversal: []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32\\config\\sam",
		"....//....//....//etc/passwd",
		"..%2F..%2F..%2Fetc%2Fpasswd",
		"..%252F..%252F..%252Fetc%252Fpasswd",
		"/%2e%2e/%2e%2e/%2e%2e/etc/passwd",
		"/./../../etc/passwd",
		"../../../../../etc/passwd",
	},
	CommandInjection: []string{
		"component; rm -rf /",
		"component && rm -rf /",
		"component | rm -rf /",
		"component`rm -rf /`",
		"component$(rm -rf /)",
		"component & del /s /q C:\\",
		"component; cat /etc/passwd",
		"component\nrm -rf /",
	},
	ScriptInjection: []string{
		"<script>alert('xss')</script>",
		"<img src=x onerror=alert('xss')>",
		"javascript:alert('xss')",
		"<svg onload=alert('xss')>",
		"<iframe src=javascript:alert('xss')>",
		"<body onload=alert('xss')>",
		"<div onclick=alert('xss')>",
		"<script src=//evil.com/malicious.js></script>",
	},
	SQLInjection: []string{
		"'; DROP TABLE users; --",
		"' OR '1'='1",
		"' UNION SELECT * FROM users --",
		"'; DELETE FROM components; --",
		"' OR 1=1 --",
		"admin'--",
		"' OR 'a'='a",
		"'; INSERT INTO users VALUES ('hacker', 'password'); --",
	},
}

// CreateSecureTestEnvironment sets up a test environment with security considerations
func CreateSecureTestEnvironment(t *testing.T) (string, *config.Config) {
	projectDir := CreateTempProject(t)

	// Create restrictive configuration
	cfg := CreateTestConfig(projectDir)

	// Ensure secure permissions on cache directory
	cacheDir := cfg.Build.CacheDir
	err := os.Chmod(cacheDir, 0700)
	require.NoError(t, err)

	return projectDir, cfg
}

// AssertFilePermissions checks that files have secure permissions
func AssertFilePermissions(t *testing.T, path string, expectedMode os.FileMode) {
	info, err := os.Stat(path)
	require.NoError(t, err)

	actualMode := info.Mode()
	require.Equal(t, expectedMode, actualMode&os.FileMode(0777),
		"File %s has incorrect permissions: got %o, want %o",
		path, actualMode&os.FileMode(0777), expectedMode)
}

// AssertDirectoryPermissions checks that directories have secure permissions
func AssertDirectoryPermissions(t *testing.T, path string, expectedMode os.FileMode) {
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.True(t, info.IsDir(), "Path %s is not a directory", path)

	actualMode := info.Mode()
	require.Equal(t, expectedMode, actualMode&os.FileMode(0777),
		"Directory %s has incorrect permissions: got %o, want %o",
		path, actualMode&os.FileMode(0777), expectedMode)
}

// CleanupTestEnvironment removes test files and directories
func CleanupTestEnvironment(projectDir string) error {
	return os.RemoveAll(projectDir)
}

// WaitForFileChange waits for a file to be modified (useful for testing file watchers)
func WaitForFileChange(
	t *testing.T,
	filePath string,
	originalModTime time.Time,
	timeout time.Duration,
) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		info, err := os.Stat(filePath)
		if err == nil && info.ModTime().After(originalModTime) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("File %s was not modified within %v", filePath, timeout)
}
