// Package testdata provides utilities for generating test data and fixtures
// across the test suite. This ensures consistent and realistic test data
// for all testing scenarios.
package testdata

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/conneroisu/templar/internal/registry"
)

// ComponentGenerator provides methods for generating test components
type ComponentGenerator struct {
	baseDir string
}

// NewComponentGenerator creates a new component generator
func NewComponentGenerator(baseDir string) *ComponentGenerator {
	return &ComponentGenerator{
		baseDir: baseDir,
	}
}

// GenerateSimpleComponents creates a set of simple test components
func (g *ComponentGenerator) GenerateSimpleComponents(count int) (string, error) {
	testDir := filepath.Join(g.baseDir, fmt.Sprintf("simple_test_%d", time.Now().UnixNano()))
	if err := os.MkdirAll(testDir, 0755); err != nil {
		return "", err
	}

	templates := []string{
		`package components

templ Button%d(text string) {
	<button class="btn">{text}</button>
}`,
		`package components

templ Card%d(title string, content string) {
	<div class="card">
		<h3>{title}</h3>
		<p>{content}</p>
	</div>
}`,
		`package components

templ Alert%d(message string, type string) {
	<div class={"alert", "alert-" + type}>
		{message}
	</div>
}`,
		`package components

templ Modal%d(title string, visible bool) {
	if visible {
		<div class="modal">
			<div class="modal-header">
				<h2>{title}</h2>
			</div>
		</div>
	}
}`,
	}

	for i := 0; i < count; i++ {
		template := templates[i%len(templates)]
		content := fmt.Sprintf(template, i)
		filename := filepath.Join(testDir, fmt.Sprintf("component_%d.templ", i))

		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			return "", err
		}
	}

	return testDir, nil
}

// GenerateComplexComponents creates components with various complexities
func (g *ComponentGenerator) GenerateComplexComponents() (string, error) {
	testDir := filepath.Join(g.baseDir, fmt.Sprintf("complex_test_%d", time.Now().UnixNano()))
	if err := os.MkdirAll(testDir, 0755); err != nil {
		return "", err
	}

	components := map[string]string{
		"data_table.templ": `package components

import "fmt"

type User struct {
	ID   int
	Name string
	Role string
}

templ DataTable(users []User, sortBy string, ascending bool) {
	<table class="data-table">
		<thead>
			<tr>
				<th class={ "sortable", templ.KV("active", sortBy == "id") }>ID</th>
				<th class={ "sortable", templ.KV("active", sortBy == "name") }>Name</th>
				<th class={ "sortable", templ.KV("active", sortBy == "role") }>Role</th>
			</tr>
		</thead>
		<tbody>
			for _, user := range users {
				<tr>
					<td>{fmt.Sprintf("%d", user.ID)}</td>
					<td>{user.Name}</td>
					<td>{user.Role}</td>
				</tr>
			}
		</tbody>
	</table>
}`,
		"form_builder.templ": `package components

type FormField struct {
	Name        string
	Type        string
	Label       string
	Required    bool
	Placeholder string
	Options     []string
}

templ FormBuilder(fields []FormField, values map[string]string, errors map[string]string) {
	<form class="dynamic-form">
		for _, field := range fields {
			<div class="form-group">
				<label for={field.Name}>
					{field.Label}
					if field.Required {
						<span class="required">*</span>
					}
				</label>
				switch field.Type {
					case "select":
						<select name={field.Name} id={field.Name}>
							for _, option := range field.Options {
								<option 
									value={option}
									selected?={values[field.Name] == option}
								>
									{option}
								</option>
							}
						</select>
					case "textarea":
						<textarea 
							name={field.Name} 
							id={field.Name}
							placeholder={field.Placeholder}
						>{values[field.Name]}</textarea>
					default:
						<input 
							type={field.Type}
							name={field.Name}
							id={field.Name}
							value={values[field.Name]}
							placeholder={field.Placeholder}
							required?={field.Required}
						/>
				}
				if error, exists := errors[field.Name]; exists {
					<div class="error">{error}</div>
				}
			</div>
		}
		<button type="submit">Submit</button>
	</form>
}`,
		"layout.templ": `package components

templ Layout(title string, sidebar bool) {
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<title>{title}</title>
			<meta charset="utf-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1"/>
		</head>
		<body>
			@Header(title)
			<main class="main">
				if sidebar {
					<div class="layout-with-sidebar">
						<aside class="sidebar">
							@Sidebar()
						</aside>
						<div class="content">
							{ children... }
						</div>
					</div>
				} else {
					<div class="content-full">
						{ children... }
					</div>
				}
			</main>
			@Footer()
		</body>
	</html>
}

templ Header(title string) {
	<header class="header">
		<nav class="navbar">
			<div class="navbar-brand">{title}</div>
		</nav>
	</header>
}

templ Sidebar() {
	<nav class="sidebar-nav">
		<ul>
			<li><a href="/dashboard">Dashboard</a></li>
			<li><a href="/components">Components</a></li>
			<li><a href="/settings">Settings</a></li>
		</ul>
	</nav>
}

templ Footer() {
	<footer class="footer">
		<p>&copy; 2024 Templar Framework</p>
	</footer>
}`,
	}

	for filename, content := range components {
		filepath := filepath.Join(testDir, filename)
		if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
			return "", err
		}
	}

	return testDir, nil
}

// GenerateSecurityTestComponents creates components with potential security issues
func (g *ComponentGenerator) GenerateSecurityTestComponents() (string, error) {
	testDir := filepath.Join(g.baseDir, fmt.Sprintf("security_test_%d", time.Now().UnixNano()))
	if err := os.MkdirAll(testDir, 0755); err != nil {
		return "", err
	}

	// Components with various security considerations for testing
	components := map[string]string{
		"safe_component.templ": `package components

templ SafeComponent(userInput string) {
	<div class="safe">
		{userInput}
	</div>
}`,
		"file_input.templ": `package components

templ FileUpload(allowedTypes []string, maxSize int) {
	<form enctype="multipart/form-data">
		<input 
			type="file" 
			name="upload"
			accept={strings.Join(allowedTypes, ",")}
		/>
		<input type="hidden" name="max_size" value={fmt.Sprintf("%d", maxSize)}/>
		<button type="submit">Upload</button>
	</form>
}`,
		"user_profile.templ": `package components

import "strings"

type UserProfile struct {
	ID       int
	Username string
	Email    string
	Bio      string
	Avatar   string
}

templ UserProfileCard(profile UserProfile, isOwner bool) {
	<div class="user-profile">
		<div class="avatar">
			<img src={profile.Avatar} alt={profile.Username + " avatar"}/>
		</div>
		<div class="info">
			<h3>{profile.Username}</h3>
			<p class="email">{profile.Email}</p>
			<div class="bio">
				{profile.Bio}
			</div>
			if isOwner {
				<div class="actions">
					<a href={"/users/" + fmt.Sprintf("%d", profile.ID) + "/edit"}>
						Edit Profile
					</a>
				</div>
			}
		</div>
	</div>
}`,
	}

	for filename, content := range components {
		filepath := filepath.Join(testDir, filename)
		if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
			return "", err
		}
	}

	return testDir, nil
}

// MockData provides consistent mock data for tests
type MockData struct{}

// NewMockData creates a new mock data provider
func NewMockData() *MockData {
	return &MockData{}
}

// SampleComponents returns a set of sample component info for testing
func (m *MockData) SampleComponents() []*registry.ComponentInfo {
	return []*registry.ComponentInfo{
		{
			Name:     "Button",
			Package:  "components",
			FilePath: "button.templ",
			Parameters: []registry.ParameterInfo{
				{Name: "text", Type: "string"},
				{Name: "disabled", Type: "bool"},
			},
			LastMod: time.Now().Add(-1 * time.Hour),
		},
		{
			Name:     "Card",
			Package:  "components",
			FilePath: "card.templ",
			Parameters: []registry.ParameterInfo{
				{Name: "title", Type: "string"},
				{Name: "content", Type: "string"},
				{Name: "footer", Type: "string"},
			},
			LastMod: time.Now().Add(-30 * time.Minute),
		},
		{
			Name:     "DataTable",
			Package:  "components",
			FilePath: "data_table.templ",
			Parameters: []registry.ParameterInfo{
				{Name: "data", Type: "[]interface{}"},
				{Name: "columns", Type: "[]string"},
				{Name: "sortable", Type: "bool"},
				{Name: "pageSize", Type: "int"},
			},
			LastMod: time.Now().Add(-5 * time.Minute),
		},
	}
}

// SampleFormFields returns sample form field data
func (m *MockData) SampleFormFields() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":        "username",
			"type":        "text",
			"label":       "Username",
			"required":    true,
			"placeholder": "Enter your username",
		},
		{
			"name":        "email",
			"type":        "email",
			"label":       "Email Address",
			"required":    true,
			"placeholder": "user@example.com",
		},
		{
			"name":     "role",
			"type":     "select",
			"label":    "Role",
			"required": true,
			"options":  []string{"user", "admin", "moderator"},
		},
		{
			"name":        "bio",
			"type":        "textarea",
			"label":       "Biography",
			"required":    false,
			"placeholder": "Tell us about yourself...",
		},
	}
}

// SecurityTestCases returns test cases for security testing
func (m *MockData) SecurityTestCases() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":        "XSS Script Tag",
			"input":       "<script>alert('xss')</script>",
			"expectSafe":  true,
			"description": "Should escape script tags",
		},
		{
			"name":        "SQL Injection Attempt",
			"input":       "'; DROP TABLE users; --",
			"expectSafe":  true,
			"description": "Should handle SQL injection patterns safely",
		},
		{
			"name":        "Path Traversal",
			"input":       "../../../etc/passwd",
			"expectSafe":  true,
			"description": "Should prevent path traversal",
		},
		{
			"name":        "Command Injection",
			"input":       "test; rm -rf /",
			"expectSafe":  true,
			"description": "Should prevent command injection",
		},
		{
			"name":        "Normal User Input",
			"input":       "Hello, World!",
			"expectSafe":  true,
			"description": "Should handle normal input correctly",
		},
	}
}

// CleanupTestData removes test directories and files
func CleanupTestData(baseDir string) error {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() && (contains(entry.Name(), "test_") ||
			contains(entry.Name(), "simple_test_") ||
			contains(entry.Name(), "complex_test_") ||
			contains(entry.Name(), "security_test_")) {

			fullPath := filepath.Join(baseDir, entry.Name())
			if err := os.RemoveAll(fullPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}
