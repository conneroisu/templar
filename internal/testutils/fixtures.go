// Package testutils provides test fixtures and data generators for consistent testing.
// This follows the zero technical debt policy by providing reusable test data.
package testutils

import (
	"fmt"
	"strings"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/types"
)

// TestFixtures provides common test data and configurations.
type TestFixtures struct{}

// NewTestFixtures creates a new test fixtures instance.
func NewTestFixtures() *TestFixtures {
	return &TestFixtures{}
}

// BasicConfig returns a basic valid configuration for testing.
func (tf *TestFixtures) BasicConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Port:           8080,
			Host:           "localhost",
			Open:           false,
			NoOpen:         true,
			Middleware:     []string{"cors", "security"},
			AllowedOrigins: []string{"http://localhost:3000"},
			Environment:    "development",
		},
		Build: config.BuildConfig{
			Command:  "templ generate",
			Watch:    []string{"**/*.templ"},
			Ignore:   []string{"node_modules", ".git", "*.tmp"},
			CacheDir: ".templar/cache",
		},
		Preview: config.PreviewConfig{
			MockData:  "auto",
			Wrapper:   "layout.templ",
			AutoProps: true,
		},
		Components: config.ComponentsConfig{
			ScanPaths:       []string{"./components", "./views"},
			ExcludePatterns: []string{"*_test.templ", "*.bak"},
		},
		Development: config.DevelopmentConfig{
			HotReload:         true,
			CSSInjection:      true,
			StatePreservation: true,
			ErrorOverlay:      true,
		},
		Production: config.ProductionConfig{
			OutputDir: "./dist",
			StaticDir: "./static",
			AssetsDir: "./assets",
		},
		Plugins: config.PluginsConfig{
			DiscoveryPaths: []string{"./plugins"},
			Configurations: make(map[string]config.PluginConfigMap),
		},
		Monitoring: config.MonitoringConfig{
			Enabled:       true,
			MetricsPath:   "/metrics",
			HTTPPort:      9090,
			AlertsEnabled: true,
		},
		Timeouts: config.TimeoutConfig{
			Build:    30 * time.Second,
			External: 10 * time.Second,
		},
	}
}

// MinimalConfig returns a minimal valid configuration.
func (tf *TestFixtures) MinimalConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Host: "localhost",
		},
		Components: config.ComponentsConfig{
			ScanPaths: []string{"./components"},
		},
	}
}

// InvalidConfigs returns a set of invalid configurations for testing validation.
func (tf *TestFixtures) InvalidConfigs() map[string]*config.Config {
	return map[string]*config.Config{
		"invalid_port_negative": {
			Server: config.ServerConfig{Port: -1, Host: "localhost"},
		},
		"invalid_port_too_high": {
			Server: config.ServerConfig{Port: 70000, Host: "localhost"},
		},
		"empty_host": {
			Server: config.ServerConfig{Port: 8080, Host: ""},
		},
		"dangerous_host": {
			Server: config.ServerConfig{Port: 8080, Host: "host; rm -rf /"},
		},
		"empty_scan_paths": {
			Components: config.ComponentsConfig{ScanPaths: []string{}},
		},
		"path_traversal": {
			Components: config.ComponentsConfig{ScanPaths: []string{"../../../etc"}},
		},
	}
}

// SampleComponents returns sample templ components for testing.
func (tf *TestFixtures) SampleComponents() map[string]TestComponent {
	return map[string]TestComponent{
		"simple_button": {
			Name:     "Button",
			Package:  "components",
			FilePath: "button.templ",
			Function: "Button",
			Parameters: []TestParameter{
				{Name: "text", Type: "string"},
				{Name: "disabled", Type: "bool"},
			},
			Content: `package components

templ Button(text string, disabled bool) {
	<button class="btn" disabled?={ disabled }>
		{ text }
	</button>
}`,
		},
		"card_component": {
			Name:     "Card",
			Package:  "components",
			FilePath: "card.templ",
			Function: "Card",
			Parameters: []TestParameter{
				{Name: "title", Type: "string"},
				{Name: "content", Type: "string"},
				{Name: "bordered", Type: "bool"},
			},
			Content: `package components

templ Card(title string, content string, bordered bool) {
	<div class={ "card", templ.KV("card--bordered", bordered) }>
		<div class="card__header">
			<h3>{ title }</h3>
		</div>
		<div class="card__body">
			{ content }
		</div>
	</div>
}`,
		},
		"layout_component": {
			Name:     "Layout",
			Package:  "layouts",
			FilePath: "layout.templ",
			Function: "Layout",
			Parameters: []TestParameter{
				{Name: "title", Type: "string"},
			},
			Content: `package layouts

templ Layout(title string) {
	<!DOCTYPE html>
	<html>
		<head>
			<title>{ title }</title>
			<meta charset="utf-8"/>
		</head>
		<body>
			{ children... }
		</body>
	</html>
}`,
		},
		"list_component": {
			Name:     "List",
			Package:  "components",
			FilePath: "list.templ",
			Function: "List",
			Parameters: []TestParameter{
				{Name: "items", Type: "[]string"},
				{Name: "ordered", Type: "bool"},
			},
			Content: `package components

templ List(items []string, ordered bool) {
	if ordered {
		<ol class="list list--ordered">
			for _, item := range items {
				<li>{ item }</li>
			}
		</ol>
	} else {
		<ul class="list">
			for _, item := range items {
				<li>{ item }</li>
			}
		</ul>
	}
}`,
		},
	}
}

// ComponentWithErrors returns components with various types of errors for testing error handling.
func (tf *TestFixtures) ComponentsWithErrors() map[string]TestComponent {
	return map[string]TestComponent{
		"syntax_error": {
			Name:     "BrokenSyntax",
			Package:  "components",
			FilePath: "broken_syntax.templ",
			Function: "BrokenSyntax",
			Content: `package components

templ BrokenSyntax() {
	<div>
		<span>Unclosed tag
	</div>
`, // Missing closing span tag
		},
		"invalid_go_syntax": {
			Name:     "InvalidGo",
			Package:  "components",
			FilePath: "invalid_go.templ",
			Function: "InvalidGo",
			Content: `package components

templ InvalidGo() {
	{{ invalid go syntax here }}
	<div>Content</div>
}`,
		},
		"undefined_variable": {
			Name:     "UndefinedVar",
			Package:  "components",
			FilePath: "undefined_var.templ",
			Function: "UndefinedVar",
			Content: `package components

templ UndefinedVar() {
	<div>{ undefinedVariable }</div>
}`,
		},
	}
}

// MockData provides sample data for testing components.
type MockData struct {
	Strings    []string
	Numbers    []int
	Booleans   []bool
	Objects    []map[string]interface{}
	Users      []User
	Products   []Product
	TimeStamps []time.Time
}

// User represents a mock user for testing.
type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Active   bool   `json:"active"`
	Role     string `json:"role"`
	Created  time.Time `json:"created"`
}

// Product represents a mock product for testing.
type Product struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Category    string  `json:"category"`
	InStock     bool    `json:"in_stock"`
	Tags        []string `json:"tags"`
}

// GenerateMockData creates comprehensive mock data for testing.
func (tf *TestFixtures) GenerateMockData() *MockData {
	now := time.Now()
	
	return &MockData{
		Strings: []string{
			"",
			"simple text",
			"Text with spaces and punctuation!",
			"Unicode: 🚀 ñoño αβγ",
			"Very long text that spans multiple lines and contains various characters including numbers 123, symbols @#$%, and unicode characters ñoño αβγ δεζ",
			"<script>alert('xss')</script>", // XSS attempt
			"'; DROP TABLE users; --",        // SQL injection attempt
			"../../../etc/passwd",            // Path traversal attempt
		},
		Numbers: []int{
			0, 1, -1, 42, 100, 1000, 999999, -999999,
		},
		Booleans: []bool{true, false},
		Objects: []map[string]interface{}{
			{},
			{"key": "value"},
			{"nested": map[string]interface{}{"deep": "value"}},
			{"array": []string{"a", "b", "c"}},
		},
		Users: []User{
			{ID: 1, Name: "John Doe", Email: "john@example.com", Active: true, Role: "admin", Created: now},
			{ID: 2, Name: "Jane Smith", Email: "jane@example.com", Active: true, Role: "user", Created: now.Add(-24 * time.Hour)},
			{ID: 3, Name: "Bob Wilson", Email: "bob@example.com", Active: false, Role: "user", Created: now.Add(-7 * 24 * time.Hour)},
			{ID: 4, Name: "Alice Brown", Email: "alice@example.com", Active: true, Role: "moderator", Created: now.Add(-30 * 24 * time.Hour)},
		},
		Products: []Product{
			{
				ID: 1, Name: "Laptop", Description: "High-performance laptop", 
				Price: 999.99, Category: "Electronics", InStock: true,
				Tags: []string{"computer", "portable", "work"},
			},
			{
				ID: 2, Name: "Coffee Mug", Description: "Ceramic coffee mug", 
				Price: 12.99, Category: "Kitchen", InStock: true,
				Tags: []string{"drink", "ceramic", "kitchen"},
			},
			{
				ID: 3, Name: "Book", Description: "Programming guide", 
				Price: 29.99, Category: "Books", InStock: false,
				Tags: []string{"education", "programming", "guide"},
			},
		},
		TimeStamps: []time.Time{
			time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			now,
			now.Add(24 * time.Hour),
			now.Add(-24 * time.Hour),
			time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2030, 12, 31, 23, 59, 59, 0, time.UTC),
		},
	}
}

// ErrorScenarios provides various error scenarios for testing error handling.
func (tf *TestFixtures) ErrorScenarios() map[string]func() error {
	return map[string]func() error{
		"file_not_found": func() error {
			return fmt.Errorf("file not found: /nonexistent/path")
		},
		"permission_denied": func() error {
			return fmt.Errorf("permission denied: cannot read file")
		},
		"network_timeout": func() error {
			return fmt.Errorf("network timeout: connection timed out after 30s")
		},
		"invalid_syntax": func() error {
			return fmt.Errorf("syntax error at line 42: unexpected token '}'")
		},
		"out_of_memory": func() error {
			return fmt.Errorf("out of memory: allocation failed")
		},
		"context_cancelled": func() error {
			return fmt.Errorf("context cancelled: operation interrupted")
		},
	}
}

// TestServer configurations for testing server functionality.
func (tf *TestFixtures) ServerConfigs() map[string]config.ServerConfig {
	return map[string]config.ServerConfig{
		"development": {
			Port:           8080,
			Host:           "localhost",
			Open:           true,
			Middleware:     []string{"cors", "logging"},
			AllowedOrigins: []string{"*"},
			Environment:    "development",
		},
		"production": {
			Port:           80,
			Host:           "0.0.0.0",
			Open:           false,
			Middleware:     []string{"security", "ratelimit", "cors"},
			AllowedOrigins: []string{"https://example.com"},
			Environment:    "production",
		},
		"testing": {
			Port:           0, // Random port
			Host:           "127.0.0.1",
			Open:           false,
			Middleware:     []string{},
			AllowedOrigins: []string{},
			Environment:    "test",
		},
	}
}

// SecurityTestCases provides test cases for security validation.
func (tf *TestFixtures) SecurityTestCases() map[string]string {
	return map[string]string{
		"xss_script_tag":     "<script>alert('xss')</script>",
		"xss_img_onerror":    "<img src=x onerror=alert('xss')>",
		"xss_javascript_url": "<a href=\"javascript:alert('xss')\">click</a>",
		"sql_injection":      "'; DROP TABLE users; --",
		"path_traversal":     "../../../etc/passwd",
		"null_byte":          "file.txt\x00.jpg",
		"command_injection":  "file.txt; rm -rf /",
		"ldap_injection":     "*)(&(objectClass=*))",
		"xml_bomb":           "<?xml version=\"1.0\"?><!DOCTYPE bomb [<!ENTITY a \"1234567890\">]><bomb>&a;</bomb>",
		"unicode_bypass":     "java\u0000script:alert('xss')",
	}
}

// PerformanceTestData provides data sets of various sizes for performance testing.
func (tf *TestFixtures) PerformanceTestData() map[string][]string {
	small := make([]string, 10)
	medium := make([]string, 100)
	large := make([]string, 1000)
	xlarge := make([]string, 10000)
	
	// Fill with varied content
	for i := range small {
		small[i] = fmt.Sprintf("small_item_%d", i)
	}
	for i := range medium {
		medium[i] = fmt.Sprintf("medium_item_%d_with_longer_content", i)
	}
	for i := range large {
		large[i] = fmt.Sprintf("large_item_%d_with_much_longer_content_that_simulates_real_world_data", i)
	}
	for i := range xlarge {
		xlarge[i] = fmt.Sprintf("xlarge_item_%d_%s", i, strings.Repeat("x", 100))
	}
	
	return map[string][]string{
		"small":  small,
		"medium": medium,
		"large":  large,
		"xlarge": xlarge,
	}
}

// ComponentRegistry creates a mock component registry for testing.
func (tf *TestFixtures) ComponentRegistry() map[string]*types.ComponentInfo {
	components := tf.SampleComponents()
	registry := make(map[string]*types.ComponentInfo)
	
	for name, comp := range components {
		registry[name] = &types.ComponentInfo{
			Name:     comp.Name,
			Package:  comp.Package,
			FilePath: comp.FilePath,
			// Add other fields as needed
		}
	}
	
	return registry
}

// BuildConfigurations returns various build configurations for testing.
func (tf *TestFixtures) BuildConfigurations() map[string]config.BuildConfig {
	return map[string]config.BuildConfig{
		"basic": {
			Command:  "templ generate",
			Watch:    []string{"**/*.templ"},
			Ignore:   []string{"node_modules"},
			CacheDir: ".templar/cache",
		},
		"advanced": {
			Command:  "templ generate && go build",
			Watch:    []string{"**/*.templ", "**/*.go"},
			Ignore:   []string{"node_modules", ".git", "*.tmp", "dist/"},
			CacheDir: ".templar/advanced-cache",
		},
		"minimal": {
			Command: "templ generate",
		},
	}
}