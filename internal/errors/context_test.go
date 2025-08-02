package errors

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/registry"
)

// TestProjectContext tests the project context analysis functionality.
func TestProjectContext(t *testing.T) {
	// Create a temporary test directory
	tempDir := t.TempDir()

	// Create test project structure
	setupTestProject(t, tempDir)

	// Create test config
	cfg := &config.Config{
		Components: config.ComponentsConfig{
			ScanPaths: []string{"./components", "./views"},
		},
		Server: config.ServerConfig{
			Port: 8080,
			Host: "localhost",
		},
	}

	// Create test registry
	reg := registry.NewComponentRegistry()

	// Create project context
	ctx := NewProjectContext(tempDir, cfg, reg)

	// Test project structure analysis
	t.Run("AnalyzesProjectStructure", func(t *testing.T) {
		if len(ctx.ProjectFiles.ComponentFiles) == 0 {
			t.Error("Expected to find component files")
		}

		// Debug: list all files that were found
		t.Logf("Found component files: %v", ctx.ProjectFiles.ComponentFiles)
		t.Logf("Found config files: %v", ctx.ProjectFiles.ConfigFiles)
		t.Logf("Config path: %s", ctx.ConfigPath)

		// Check if .templar.yml exists in the temp directory
		templConfigPath := filepath.Join(tempDir, ".templar.yml")
		if _, err := os.Stat(templConfigPath); err != nil {
			t.Errorf("Test setup error: .templar.yml not created at %s: %v", templConfigPath, err)
		}

		// Check that at least some config files were found (we create go.mod and .templar.yml)
		if len(ctx.ProjectFiles.ConfigFiles) == 0 {
			t.Error("Expected to find at least some config files (go.mod, .templar.yml)")
		}

		if ctx.ConfigPath == "" {
			t.Error("Expected to find config path")
		}
	})

	// Test error tracking
	t.Run("TracksRecentErrors", func(t *testing.T) {
		testErr := NewValidationError("TEST_ERROR", "test error message")
		ctx.AddRecentError(testErr, map[string]interface{}{"test": "context"})

		if len(ctx.RecentErrors) != 1 {
			t.Errorf("Expected 1 recent error, got %d", len(ctx.RecentErrors))
		}

		// Test error pattern analysis
		patterns := ctx.GetRecentErrorPatterns()
		if len(patterns) == 0 {
			t.Error("Expected to find error patterns")
		}
	})

	// Test contextual suggestions
	t.Run("ProvidesContextualSuggestions", func(t *testing.T) {
		suggestions := ctx.GetContextualSuggestions("component_not_found", map[string]interface{}{
			"component": "TestComponent",
		})

		if len(suggestions) == 0 {
			t.Error("Expected to receive suggestions")
		}

		// Verify suggestions contain real paths
		hasRealPath := false
		for _, suggestion := range suggestions {
			if strings.Contains(suggestion.Command, tempDir) ||
				strings.Contains(suggestion.Command, "components") {
				hasRealPath = true

				break
			}
		}

		if !hasRealPath {
			t.Error("Expected suggestions to contain real project paths")
		}
	})

	// Test project summary
	t.Run("GeneratesProjectSummary", func(t *testing.T) {
		summary := ctx.GetProjectSummary()

		if summary == "" {
			t.Error("Expected non-empty project summary")
		}

		if !strings.Contains(summary, "Components:") {
			t.Error("Expected summary to contain component count")
		}
	})
}

// TestContextualSuggestionProvider tests the suggestion provider functionality.
func TestContextualSuggestionProvider(t *testing.T) {
	tempDir := t.TempDir()
	setupTestProject(t, tempDir)

	cfg := &config.Config{
		Components: config.ComponentsConfig{
			ScanPaths: []string{"./components"},
		},
	}

	projectCtx := NewProjectContext(tempDir, cfg, nil)
	provider := NewContextualSuggestionProvider(projectCtx)

	t.Run("EnhancesTemplarErrors", func(t *testing.T) {
		testErr := NewValidationError("ERR_COMPONENT_NOT_FOUND", "component 'Button' not found")
		suggestions := provider.GetEnhancedSuggestions(testErr)

		if len(suggestions) == 0 {
			t.Error("Expected to receive enhanced suggestions")
		}

		// Check for learning resources
		hasLearningResource := false
		for _, suggestion := range suggestions {
			if strings.Contains(suggestion.Title, "📚") {
				hasLearningResource = true

				break
			}
		}

		if !hasLearningResource {
			t.Error("Expected suggestions to include learning resources")
		}
	})

	t.Run("ProvidesComponentNotFoundSuggestions", func(t *testing.T) {
		suggestions := provider.GetComponentNotFoundSuggestions("MissingComponent")

		if len(suggestions) == 0 {
			t.Error("Expected component not found suggestions")
		}

		// Verify suggestions are actionable
		hasCommand := false
		for _, suggestion := range suggestions {
			if suggestion.Command != "" {
				hasCommand = true

				break
			}
		}

		if !hasCommand {
			t.Error("Expected at least one suggestion with a command")
		}
	})

	t.Run("ProvidesBuildFailureSuggestions", func(t *testing.T) {
		suggestions := provider.GetBuildFailureSuggestions("syntax error at line 42", "TestComponent")

		if len(suggestions) == 0 {
			t.Error("Expected build failure suggestions")
		}

		// Check for templ-specific guidance
		hasTemplGuidance := false
		for _, suggestion := range suggestions {
			if strings.Contains(strings.ToLower(suggestion.Description), "templ") ||
				strings.Contains(strings.ToLower(suggestion.Title), "templ") {
				hasTemplGuidance = true

				break
			}
		}

		if !hasTemplGuidance {
			t.Error("Expected templ-specific guidance in suggestions")
		}
	})

	t.Run("ProvidesConfigurationSuggestions", func(t *testing.T) {
		suggestions := provider.GetConfigurationSuggestions("YAML syntax error")

		if len(suggestions) == 0 {
			t.Error("Expected configuration suggestions")
		}

		// Verify configuration-specific help
		hasConfigHelp := false
		for _, suggestion := range suggestions {
			if strings.Contains(strings.ToLower(suggestion.Description), "config") {
				hasConfigHelp = true

				break
			}
		}

		if !hasConfigHelp {
			t.Error("Expected configuration-specific help")
		}
	})
}

// TestCLIErrorHandler tests the CLI integration functionality.
func TestCLIErrorHandler(t *testing.T) {
	tempDir := t.TempDir()
	setupTestProject(t, tempDir)

	cfg := &config.Config{
		Components: config.ComponentsConfig{
			ScanPaths: []string{"./components"},
		},
		Server: config.ServerConfig{
			Port: 8080,
			Host: "localhost",
		},
	}

	handler := NewCLIErrorHandler(tempDir, cfg, nil)

	t.Run("FormatsCLIErrors", func(t *testing.T) {
		testErr := NewValidationError("ERR_TEST", "test validation error")
		formatted := handler.FormatCLIError(testErr)

		if formatted == "" {
			t.Error("Expected non-empty formatted error")
		}

		if !strings.Contains(formatted, "❌ Error:") {
			t.Error("Expected formatted error to contain error marker")
		}

		if !strings.Contains(formatted, "💡 Suggestions:") {
			t.Error("Expected formatted error to contain suggestions")
		}
	})

	t.Run("FormatsCLIErrorsWithHelp", func(t *testing.T) {
		testErr := NewNetworkError("ERR_PORT_IN_USE", "port 8080 already in use", nil)
		formatted := handler.FormatCLIErrorWithHelp(testErr, "serve")

		if !strings.Contains(formatted, "🔧 Command Help for 'serve':") {
			t.Error("Expected command-specific help in formatted output")
		}
	})

	t.Run("MakesCommandsCopyPasteable", func(t *testing.T) {
		originalCommand := "ls -la ./components"
		copyPasteable := handler.makeCommandCopyPasteable(originalCommand)

		// Should convert relative path to absolute
		if !strings.Contains(copyPasteable, tempDir) {
			t.Error("Expected command to contain absolute path")
		}
	})

	t.Run("ProvidesProjectInsights", func(t *testing.T) {
		insights := handler.GetProjectInsights()

		if insights.PrimaryComponentsDir == "" {
			t.Error("Expected primary components directory")
		}

		if insights.ProjectSummary == "" {
			t.Error("Expected project summary")
		}

		// Test health check
		isHealthy := insights.IsHealthy()
		if insights.ComponentCount == 0 && isHealthy {
			t.Error("Project with no components should not be healthy")
		}
	})

	t.Run("CreatesEnhancedErrors", func(t *testing.T) {
		enhancedErr := handler.CreateEnhancedError(
			ErrorTypeValidation,
			"ERR_TEST_ENHANCED",
			"test enhanced error",
			nil,
		)

		if enhancedErr == nil {
			t.Error("Expected enhanced error to be created")

			return // Early return to avoid nil pointer dereference
		}

		if len(enhancedErr.Suggestions) == 0 {
			t.Error("Expected enhanced error to have suggestions")
		}
	})
}

// TestSuggestionQuality tests the quality and usefulness of suggestions.
func TestSuggestionQuality(t *testing.T) {
	tempDir := t.TempDir()
	setupTestProject(t, tempDir)

	cfg := &config.Config{
		Components: config.ComponentsConfig{
			ScanPaths: []string{"./components"},
		},
	}

	projectCtx := NewProjectContext(tempDir, cfg, nil)
	provider := NewContextualSuggestionProvider(projectCtx)

	t.Run("SuggestionsHaveRequiredFields", func(t *testing.T) {
		testErr := NewValidationError("ERR_TEST", "test error")
		suggestions := provider.GetEnhancedSuggestions(testErr)

		for i, suggestion := range suggestions {
			if suggestion.Title == "" {
				t.Errorf("Suggestion %d missing title", i)
			}

			if suggestion.Description == "" && suggestion.Command == "" && suggestion.Example == "" {
				t.Errorf("Suggestion %d has no actionable content", i)
			}
		}
	})

	t.Run("CommandsAreExecutable", func(t *testing.T) {
		testErr := NewBuildError("ERR_BUILD_FAILED", "build failed", nil)
		suggestions := provider.GetEnhancedSuggestions(testErr)

		for _, suggestion := range suggestions {
			if suggestion.Command == "" {
				continue
			}

			// Commands should not contain placeholder text
			problematicPatterns := []string{
				"<component>", "<path>", "<name>", "YOUR_", "PLACEHOLDER",
			}

			for _, pattern := range problematicPatterns {
				if strings.Contains(suggestion.Command, pattern) {
					t.Errorf("Command contains placeholder text: %s", suggestion.Command)
				}
			}
		}
	})

	t.Run("SuggestionsArePrioritized", func(t *testing.T) {
		testErr := NewValidationError("ERR_TEST", "test error")
		suggestions := provider.GetEnhancedSuggestions(testErr)

		// Immediate fixes should come before learning resources
		immediateFixes := 0
		learningResources := 0
		learningStarted := false

		for _, suggestion := range suggestions {
			if strings.Contains(suggestion.Title, "📚") {
				learningStarted = true
				learningResources++
			} else if !learningStarted {
				immediateFixes++
			}
		}

		if learningStarted && immediateFixes == 0 {
			t.Error("Expected immediate fixes before learning resources")
		}

		// Should have reasonable limits
		if len(suggestions) > 8 {
			t.Errorf("Too many suggestions (%d), should be limited for usability", len(suggestions))
		}
	})
}

// TestErrorPatternAnalysis tests the error pattern analysis functionality.
func TestErrorPatternAnalysis(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{}
	projectCtx := NewProjectContext(tempDir, cfg, nil)

	t.Run("TracksErrorPatterns", func(t *testing.T) {
		// Add multiple similar errors
		for range 5 {
			err := NewValidationError("ERR_COMPONENT_NOT_FOUND", "component not found")
			projectCtx.AddRecentError(err, map[string]interface{}{})
		}

		patterns := projectCtx.GetRecentErrorPatterns()

		if patterns["component_not_found"] != 5 {
			t.Errorf("Expected 5 component_not_found patterns, got %d", patterns["component_not_found"])
		}
	})

	t.Run("ProvidesPatternBasedSuggestions", func(t *testing.T) {
		provider := NewContextualSuggestionProvider(projectCtx)

		// Add recurring build failures
		for range 4 {
			err := NewBuildError("ERR_BUILD_FAILED", "build failed", nil)
			projectCtx.AddRecentError(err, map[string]interface{}{})
		}

		suggestions := provider.getPatternBasedSuggestions()

		hasRecurringIssue := false
		for _, suggestion := range suggestions {
			if strings.Contains(suggestion.Title, "🔄 Recurring Issue") {
				hasRecurringIssue = true

				break
			}
		}

		if !hasRecurringIssue {
			t.Error("Expected recurring issue suggestion for repeated errors")
		}
	})
}

// TestRealPathGeneration tests that suggestions contain real, usable paths.
func TestRealPathGeneration(t *testing.T) {
	tempDir := t.TempDir()
	setupTestProject(t, tempDir)

	cfg := &config.Config{
		Components: config.ComponentsConfig{
			ScanPaths: []string{"./components"},
		},
	}

	handler := NewCLIErrorHandler(tempDir, cfg, nil)

	t.Run("GeneratesRealPaths", func(t *testing.T) {
		paths := handler.getRealProjectPaths()

		if paths.ProjectRoot != tempDir {
			t.Errorf("Expected project root to be %s, got %s", tempDir, paths.ProjectRoot)
		}

		if !filepath.IsAbs(paths.ComponentsDir) {
			t.Error("Expected components directory to be absolute path")
		}

		if !strings.HasPrefix(paths.ComponentsDir, tempDir) {
			t.Error("Expected components directory to be within project root")
		}
	})

	t.Run("ExpandsRelativePaths", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"ls ./components", fmt.Sprintf("ls %s/components", tempDir)},
			{"cat .templar.yml", fmt.Sprintf("cat %s/.templar.yml", tempDir)},
		}

		for _, tc := range testCases {
			result := handler.expandRelativePaths(tc.input)
			if !strings.Contains(result, tempDir) {
				t.Errorf("Expected %s to contain %s, got %s", tc.input, tempDir, result)
			}
		}
	})
}

// setupTestProject creates a realistic test project structure.
func setupTestProject(t *testing.T, dir string) {
	t.Helper()

	// Create directories
	dirs := []string{
		"components",
		"views",
		"static",
		"examples",
		"docs",
	}

	for _, d := range dirs {
		err := os.MkdirAll(filepath.Join(dir, d), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory %s: %v", d, err)
		}
	}

	// Create test files
	files := map[string]string{
		"components/button.templ": `package components

templ Button(text string) {
	<button>{text}</button>
}`,
		"components/card.templ": `package components

templ Card(title string) {
	<div class="card">
		<h3>{title}</h3>
	</div>
}`,
		"views/layout.templ": `package views

templ Layout() {
	<html>
		<head><title>Test</title></head>
		<body>Content</body>
	</html>
}`,
		".templar.yml": `server:
  port: 8080
  host: localhost
components:
  scan_paths: ["./components", "./views"]
development:
  hot_reload: true`,
		"go.mod": `module test-project

go 1.21`,
		"README.md":        "# Test Project",
		"static/style.css": "body { margin: 0; }",
		"examples/demo.templ": `package examples

templ Demo() {
	<div>Demo component</div>
}`,
	}

	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		err := os.WriteFile(fullPath, []byte(content), 0600)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	// Set some files as recently modified
	recentTime := time.Now().Add(-1 * time.Hour)
	recentFiles := []string{
		"components/button.templ",
		"static/style.css",
	}

	for _, path := range recentFiles {
		fullPath := filepath.Join(dir, path)
		err := os.Chtimes(fullPath, recentTime, recentTime)
		if err != nil {
			t.Fatalf("Failed to set time on file %s: %v", path, err)
		}
	}
}

// TestSuggestionIntegration tests end-to-end suggestion integration.
func TestSuggestionIntegration(t *testing.T) {
	tempDir := t.TempDir()
	setupTestProject(t, tempDir)

	cfg := &config.Config{
		Components: config.ComponentsConfig{
			ScanPaths: []string{"./components"},
		},
	}

	handler := NewCLIErrorHandler(tempDir, cfg, nil)

	t.Run("ComponentNotFoundIntegration", func(t *testing.T) {
		// Simulate component not found error
		err := ErrComponentNotFound("NonExistentComponent")
		formatted := handler.FormatCLIError(err)

		// Should contain actionable suggestions
		if !strings.Contains(formatted, "templar list") {
			t.Error("Expected suggestion to run 'templar list'")
		}

		// Should contain real paths
		if !strings.Contains(formatted, tempDir) {
			t.Error("Expected suggestions to contain real project paths")
		}

		// Should contain learning resources
		if !strings.Contains(formatted, "📚") {
			t.Error("Expected learning resources in suggestions")
		}
	})

	t.Run("ServerStartIntegration", func(t *testing.T) {
		// Simulate server start error
		err := NewNetworkError("ERR_PORT_IN_USE", "port 8080 already in use", nil)
		formatted := handler.FormatCLIErrorWithHelp(err, "serve")

		// Should contain port-specific suggestions
		if !strings.Contains(formatted, "8080") {
			t.Error("Expected port-specific suggestions")
		}

		// Should contain command help
		if !strings.Contains(formatted, "serve command") {
			t.Error("Expected serve command help")
		}
	})
}
