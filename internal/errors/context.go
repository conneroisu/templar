package errors

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/registry"
)

// ProjectContext provides intelligent context analysis for error suggestions.
type ProjectContext struct {
	WorkingDir   string
	ConfigPath   string
	Config       *config.Config
	Registry     *registry.ComponentRegistry
	ProjectFiles ProjectFiles
	RecentErrors []TimestampedError
	analysisTime time.Time
}

// ProjectFiles represents discovered project structure.
type ProjectFiles struct {
	ComponentFiles   []string
	ConfigFiles      []string
	TemplateFiles    []string
	StaticDirs       []string
	ExampleDirs      []string
	TestFiles        []string
	BuildArtifacts   []string
	PackageFiles     []string
	ScriptFiles      []string
	DocFiles         []string
	EmptyDirs        []string
	LargestDirs      []DirInfo
	RecentlyModified []FileInfo
}

// DirInfo contains directory analysis information.
type DirInfo struct {
	Path      string
	FileCount int
	Size      int64
}

// FileInfo contains file analysis information.
type FileInfo struct {
	Path         string
	Size         int64
	ModTime      time.Time
	Extension    string
	IsExecutable bool
}

// TimestampedError tracks errors with timestamps for pattern analysis.
type TimestampedError struct {
	Error     error
	Timestamp time.Time
	Context   map[string]interface{}
	Resolved  bool
}

// NewProjectContext creates a new project context analyzer.
func NewProjectContext(workingDir string, cfg *config.Config, reg *registry.ComponentRegistry) *ProjectContext {
	ctx := &ProjectContext{
		WorkingDir:   workingDir,
		Config:       cfg,
		Registry:     reg,
		RecentErrors: make([]TimestampedError, 0, 50), // Keep last 50 errors
		analysisTime: time.Now(),
	}

	ctx.analyzeProjectStructure()
	ctx.findConfigFile()

	return ctx
}

// analyzeProjectStructure scans the project to understand its structure.
func (pc *ProjectContext) analyzeProjectStructure() {
	pc.ProjectFiles = ProjectFiles{
		ComponentFiles:   make([]string, 0),
		ConfigFiles:      make([]string, 0),
		TemplateFiles:    make([]string, 0),
		StaticDirs:       make([]string, 0),
		ExampleDirs:      make([]string, 0),
		TestFiles:        make([]string, 0),
		BuildArtifacts:   make([]string, 0),
		PackageFiles:     make([]string, 0),
		ScriptFiles:      make([]string, 0),
		DocFiles:         make([]string, 0),
		EmptyDirs:        make([]string, 0),
		LargestDirs:      make([]DirInfo, 0),
		RecentlyModified: make([]FileInfo, 0),
	}

	// Walk the project directory to categorize files
	_ = filepath.Walk(pc.WorkingDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on errors
		}

		relPath, _ := filepath.Rel(pc.WorkingDir, path)

		// Skip hidden directories and common ignore patterns
		if pc.shouldSkipPath(relPath) {
			if info.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		if info.IsDir() {
			pc.analyzeDirectory(path, relPath, info)
		} else {
			pc.analyzeFile(path, relPath, info)
		}

		return nil
	})

	// Sort and limit collections
	pc.sortAndLimitCollections()
}

// shouldSkipPath determines if a path should be skipped during analysis.
func (pc *ProjectContext) shouldSkipPath(relPath string) bool {
	skipPatterns := []string{
		".git", ".svn", ".hg",
		"node_modules", ".next", ".nuxt",
		"vendor", ".vendor",
		".templar", ".cache",
		"target", "dist", "build",
		".vscode", ".idea",
		"coverage", "test-results",
		"logs", ".logs",
	}

	pathLower := strings.ToLower(relPath)
	for _, pattern := range skipPatterns {
		if strings.Contains(pathLower, pattern) {
			return true
		}
	}

	return false
}

// analyzeDirectory categorizes directories by their likely purpose.
func (pc *ProjectContext) analyzeDirectory(fullPath, relPath string, info os.FileInfo) {
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return
	}

	fileCount := len(entries)
	if fileCount == 0 {
		pc.ProjectFiles.EmptyDirs = append(pc.ProjectFiles.EmptyDirs, relPath)

		return
	}

	// Calculate directory size and type
	var totalSize int64
	hasComponents := false
	hasStatic := false
	hasExamples := false

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		entryInfo, _ := entry.Info()
		totalSize += entryInfo.Size()

		name := strings.ToLower(entry.Name())
		if strings.HasSuffix(name, ".templ") {
			hasComponents = true
		}
		if strings.Contains(name, "static") || strings.Contains(name, "asset") {
			hasStatic = true
		}
		if strings.Contains(name, "example") || strings.Contains(name, "demo") {
			hasExamples = true
		}
	}

	// Categorize directory
	dirLower := strings.ToLower(relPath)
	switch {
	case hasComponents || strings.Contains(dirLower, "component"):
		// Don't add to ComponentFiles here, individual files will be added
	case hasStatic || strings.Contains(dirLower, "static") || strings.Contains(dirLower, "asset"):
		pc.ProjectFiles.StaticDirs = append(pc.ProjectFiles.StaticDirs, relPath)
	case hasExamples || strings.Contains(dirLower, "example") || strings.Contains(dirLower, "demo"):
		pc.ProjectFiles.ExampleDirs = append(pc.ProjectFiles.ExampleDirs, relPath)
	}

	// Track largest directories
	pc.ProjectFiles.LargestDirs = append(pc.ProjectFiles.LargestDirs, DirInfo{
		Path:      relPath,
		FileCount: fileCount,
		Size:      totalSize,
	})
}

// analyzeFile categorizes files by their type and purpose.
func (pc *ProjectContext) analyzeFile(fullPath, relPath string, info os.FileInfo) {
	ext := strings.ToLower(filepath.Ext(relPath))
	nameLower := strings.ToLower(info.Name())

	fileInfo := FileInfo{
		Path:         relPath,
		Size:         info.Size(),
		ModTime:      info.ModTime(),
		Extension:    ext,
		IsExecutable: (info.Mode() & 0111) != 0,
	}

	// Track recently modified files (last 7 days)
	if time.Since(info.ModTime()) < 7*24*time.Hour {
		pc.ProjectFiles.RecentlyModified = append(pc.ProjectFiles.RecentlyModified, fileInfo)
	}

	// Categorize by file type
	switch {
	case ext == ".templ":
		pc.ProjectFiles.ComponentFiles = append(pc.ProjectFiles.ComponentFiles, relPath)
	case strings.HasSuffix(nameLower, ".templ.go"):
		pc.ProjectFiles.BuildArtifacts = append(pc.ProjectFiles.BuildArtifacts, relPath)
	case nameLower == ".templar.yml" || nameLower == "templar.yml":
		pc.ProjectFiles.ConfigFiles = append(pc.ProjectFiles.ConfigFiles, relPath)
	case strings.Contains(nameLower, "templar") && (ext == ".yml" || ext == ".yaml"):
		pc.ProjectFiles.ConfigFiles = append(pc.ProjectFiles.ConfigFiles, relPath)
	case ext == ".yml" || ext == ".yaml" || ext == ".json" || ext == ".toml":
		pc.ProjectFiles.ConfigFiles = append(pc.ProjectFiles.ConfigFiles, relPath)
	case ext == ".go":
		if strings.Contains(nameLower, "test") {
			pc.ProjectFiles.TestFiles = append(pc.ProjectFiles.TestFiles, relPath)
		} else {
			pc.ProjectFiles.PackageFiles = append(pc.ProjectFiles.PackageFiles, relPath)
		}
	case ext == ".html" || ext == ".htm":
		pc.ProjectFiles.TemplateFiles = append(pc.ProjectFiles.TemplateFiles, relPath)
	case ext == ".sh" || ext == ".bat" || ext == ".ps1" || info.Name() == "Makefile":
		pc.ProjectFiles.ScriptFiles = append(pc.ProjectFiles.ScriptFiles, relPath)
	case ext == ".md" || ext == ".txt" || ext == ".rst":
		pc.ProjectFiles.DocFiles = append(pc.ProjectFiles.DocFiles, relPath)
	case nameLower == "go.mod" || nameLower == "go.sum" || nameLower == "package.json":
		pc.ProjectFiles.PackageFiles = append(pc.ProjectFiles.PackageFiles, relPath)
		// Also add to config files for go.mod as it's a configuration file
		if nameLower == "go.mod" {
			pc.ProjectFiles.ConfigFiles = append(pc.ProjectFiles.ConfigFiles, relPath)
		}
	}
}

// sortAndLimitCollections sorts collections and limits them to reasonable sizes.
func (pc *ProjectContext) sortAndLimitCollections() {
	// Sort largest directories by size
	sort.Slice(pc.ProjectFiles.LargestDirs, func(i, j int) bool {
		return pc.ProjectFiles.LargestDirs[i].Size > pc.ProjectFiles.LargestDirs[j].Size
	})
	if len(pc.ProjectFiles.LargestDirs) > 10 {
		pc.ProjectFiles.LargestDirs = pc.ProjectFiles.LargestDirs[:10]
	}

	// Sort recently modified by time
	sort.Slice(pc.ProjectFiles.RecentlyModified, func(i, j int) bool {
		return pc.ProjectFiles.RecentlyModified[i].ModTime.After(pc.ProjectFiles.RecentlyModified[j].ModTime)
	})
	if len(pc.ProjectFiles.RecentlyModified) > 20 {
		pc.ProjectFiles.RecentlyModified = pc.ProjectFiles.RecentlyModified[:20]
	}
}

// findConfigFile locates the active configuration file.
func (pc *ProjectContext) findConfigFile() {
	// Check environment variable first
	if envPath := os.Getenv("TEMPLAR_CONFIG_FILE"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			pc.ConfigPath = envPath

			return
		}
	}

	// Check standard locations
	candidates := []string{
		".templar.yml",
		".templar.yaml",
		"templar.yml",
		"templar.yaml",
		filepath.Join(os.Getenv("HOME"), ".templar.yml"),
	}

	for _, candidate := range candidates {
		fullPath := candidate
		if !filepath.IsAbs(candidate) {
			fullPath = filepath.Join(pc.WorkingDir, candidate)
		}

		if _, err := os.Stat(fullPath); err == nil {
			pc.ConfigPath = candidate

			return
		}
	}

	pc.ConfigPath = ".templar.yml" // Default expected location
}

// AddRecentError tracks an error for pattern analysis.
func (pc *ProjectContext) AddRecentError(err error, context map[string]interface{}) {
	timestamped := TimestampedError{
		Error:     err,
		Timestamp: time.Now(),
		Context:   context,
		Resolved:  false,
	}

	pc.RecentErrors = append(pc.RecentErrors, timestamped)

	// Keep only last 50 errors
	if len(pc.RecentErrors) > 50 {
		pc.RecentErrors = pc.RecentErrors[1:]
	}
}

// MarkErrorResolved marks an error as resolved for learning.
func (pc *ProjectContext) MarkErrorResolved(err error) {
	for i := len(pc.RecentErrors) - 1; i >= 0; i-- {
		if pc.RecentErrors[i].Error.Error() == err.Error() {
			pc.RecentErrors[i].Resolved = true

			break
		}
	}
}

// GetRecentErrorPatterns analyzes recent errors for patterns.
func (pc *ProjectContext) GetRecentErrorPatterns() map[string]int {
	patterns := make(map[string]int)
	cutoff := time.Now().Add(-24 * time.Hour) // Last 24 hours

	for _, te := range pc.RecentErrors {
		if te.Timestamp.Before(cutoff) {
			continue
		}

		// Extract error patterns
		errStr := te.Error.Error()

		// Look for common patterns
		switch {
		case strings.Contains(errStr, "component not found"):
			patterns["component_not_found"]++
		case strings.Contains(errStr, "build failed"):
			patterns["build_failed"]++
		case strings.Contains(errStr, "port") && strings.Contains(errStr, "use"):
			patterns["port_in_use"]++
		case strings.Contains(errStr, "permission"):
			patterns["permission_denied"]++
		case strings.Contains(errStr, "config"):
			patterns["config_error"]++
		case strings.Contains(errStr, "path"):
			patterns["path_error"]++
		default:
			patterns["other"]++
		}
	}

	return patterns
}

// GetContextualSuggestions provides intelligent suggestions based on project analysis.
func (pc *ProjectContext) GetContextualSuggestions(errorType string, errorContext map[string]interface{}) []ErrorSuggestion {
	switch errorType {
	case ErrorContextComponentNotFound:
		return pc.getComponentNotFoundSuggestions(errorContext)
	case ErrorContextBuildFailed:
		return pc.getBuildFailedSuggestions(errorContext)
	case "config_error":
		return pc.getConfigErrorSuggestions(errorContext)
	case "server_start":
		return pc.getServerStartSuggestions(errorContext)
	case "path_error":
		return pc.getPathErrorSuggestions(errorContext)
	default:
		return pc.getGenericSuggestions(errorContext)
	}
}

// getComponentNotFoundSuggestions provides context-aware component suggestions.
func (pc *ProjectContext) getComponentNotFoundSuggestions(context map[string]interface{}) []ErrorSuggestion {
	suggestions := []ErrorSuggestion{}

	componentName, _ := context["component"].(string)
	if componentName == "" {
		componentName = "YourComponent"
	}

	// Check actual scan paths from config
	scanPaths := []string{"./components"}
	if pc.Config != nil && len(pc.Config.Components.ScanPaths) > 0 {
		scanPaths = pc.Config.Components.ScanPaths
	}

	// Use real scan paths in suggestions
	for _, scanPath := range scanPaths {
		fullPath := filepath.Join(pc.WorkingDir, scanPath)
		if _, err := os.Stat(fullPath); err == nil {
			suggestions = append(suggestions, ErrorSuggestion{
				Title:       "Check if component exists in " + scanPath,
				Description: "Verify the component file exists in the configured scan path",
				Command:     fmt.Sprintf("ls -la %s/ | grep -i %s", scanPath, strings.ToLower(componentName)),
			})
		} else {
			suggestions = append(suggestions, ErrorSuggestion{
				Title:       "Create missing directory " + scanPath,
				Description: "The configured scan path doesn't exist",
				Command:     "mkdir -p " + scanPath,
			})
		}
	}

	// List actual discovered components
	if len(pc.ProjectFiles.ComponentFiles) > 0 {
		var componentNames []string
		for _, compFile := range pc.ProjectFiles.ComponentFiles {
			name := strings.TrimSuffix(filepath.Base(compFile), ".templ")
			componentNames = append(componentNames, name)
		}

		suggestions = append(suggestions, ErrorSuggestion{
			Title: "Available components in your project",
			Description: fmt.Sprintf("Found %d components: %s",
				len(componentNames), strings.Join(componentNames[:min(5, len(componentNames))], ", ")),
			Command: "templar list --format json",
		})

		// Suggest similar components
		for _, compFile := range pc.ProjectFiles.ComponentFiles {
			name := strings.TrimSuffix(filepath.Base(compFile), ".templ")
			if pc.isSimilarName(componentName, name) {
				suggestions = append(suggestions, ErrorSuggestion{
					Title:       fmt.Sprintf("Did you mean '%s'?", name),
					Description: "Found similar component at " + compFile,
					Command:     "templar preview " + name,
				})

				break // Only suggest one similar component
			}
		}
	} else {
		suggestions = append(suggestions, ErrorSuggestion{
			Title:       "No components found in project",
			Description: "No .templ files were discovered in your scan paths",
			Command:     fmt.Sprintf("find %s -name '*.templ' -type f", strings.Join(scanPaths, " ")),
		})
	}

	// Check configuration
	suggestions = append(suggestions, ErrorSuggestion{
		Title:       "Verify scan paths configuration",
		Description: "Check your configuration file: " + pc.ConfigPath,
		Command:     "cat " + pc.ConfigPath,
	})

	return suggestions
}

// getBuildFailedSuggestions provides context-aware build failure suggestions.
func (pc *ProjectContext) getBuildFailedSuggestions(context map[string]interface{}) []ErrorSuggestion {
	suggestions := []ErrorSuggestion{}

	// Check for recent build artifacts
	if len(pc.ProjectFiles.BuildArtifacts) > 0 {
		suggestions = append(suggestions, ErrorSuggestion{
			Title:       "Clean build artifacts",
			Description: fmt.Sprintf("Found %d build artifacts that might be stale", len(pc.ProjectFiles.BuildArtifacts)),
			Command:     "find . -name '*_templ.go' -delete",
		})
	}

	// Check for recently modified files
	if len(pc.ProjectFiles.RecentlyModified) > 0 {
		recentFile := pc.ProjectFiles.RecentlyModified[0]
		suggestions = append(suggestions, ErrorSuggestion{
			Title: "Check recently modified file",
			Description: fmt.Sprintf("Recently modified: %s (%s ago)",
				recentFile.Path, time.Since(recentFile.ModTime).Round(time.Minute)),
			Command: "templ generate " + recentFile.Path,
		})
	}

	// Suggest checking syntax
	suggestions = append(suggestions, ErrorSuggestion{
		Title:       "Validate templ syntax",
		Description: "Check all component files for syntax errors",
		Command:     "templ generate",
	})

	return suggestions
}

// getConfigErrorSuggestions provides configuration-specific suggestions.
func (pc *ProjectContext) getConfigErrorSuggestions(context map[string]interface{}) []ErrorSuggestion {
	suggestions := []ErrorSuggestion{}

	// Check if config file exists
	if _, err := os.Stat(pc.ConfigPath); os.IsNotExist(err) {
		suggestions = append(suggestions, ErrorSuggestion{
			Title:       "Create configuration file",
			Description: fmt.Sprintf("Configuration file %s doesn't exist", pc.ConfigPath),
			Command:     "templar init --config " + pc.ConfigPath,
		})
	} else {
		suggestions = append(suggestions, ErrorSuggestion{
			Title:       "Validate configuration syntax",
			Description: "Check for YAML syntax errors in configuration",
			Command:     "cat " + pc.ConfigPath,
		})
	}

	// List alternative config files found
	if len(pc.ProjectFiles.ConfigFiles) > 1 {
		suggestions = append(suggestions, ErrorSuggestion{
			Title: "Other configuration files found",
			Description: fmt.Sprintf("Found %d config files: %s",
				len(pc.ProjectFiles.ConfigFiles),
				strings.Join(pc.ProjectFiles.ConfigFiles[:min(3, len(pc.ProjectFiles.ConfigFiles))], ", ")),
		})
	}

	return suggestions
}

// getServerStartSuggestions provides server startup suggestions.
func (pc *ProjectContext) getServerStartSuggestions(context map[string]interface{}) []ErrorSuggestion {
	suggestions := []ErrorSuggestion{}

	port, _ := context["port"].(int)
	if port == 0 {
		port = 8080 // Default
	}

	suggestions = append(suggestions, ErrorSuggestion{
		Title:       "Check port availability",
		Description: fmt.Sprintf("Check what's using port %d", port),
		Command:     fmt.Sprintf("lsof -i :%d", port),
	})

	suggestions = append(suggestions, ErrorSuggestion{
		Title:       "Use alternative port",
		Description: "Try starting server on a different port",
		Command:     fmt.Sprintf("templar serve --port %d", port+1000),
	})

	return suggestions
}

// getPathErrorSuggestions provides path-related suggestions.
func (pc *ProjectContext) getPathErrorSuggestions(context map[string]interface{}) []ErrorSuggestion {
	suggestions := []ErrorSuggestion{}

	path, _ := context["path"].(string)
	if path == "" {
		return suggestions
	}

	// Check if it's a relative path issue
	if !filepath.IsAbs(path) {
		suggestions = append(suggestions, ErrorSuggestion{
			Title:       "Check current directory",
			Description: "Verify you're in the correct working directory",
			Command:     "pwd",
		})

		suggestions = append(suggestions, ErrorSuggestion{
			Title:       "List current directory contents",
			Description: "See what files and directories are available",
			Command:     "ls -la",
		})
	}

	return suggestions
}

// getGenericSuggestions provides fallback suggestions.
func (pc *ProjectContext) getGenericSuggestions(context map[string]interface{}) []ErrorSuggestion {
	suggestions := []ErrorSuggestion{
		{
			Title:       "Check project structure",
			Description: "Verify your project is properly initialized",
			Command:     "templar list",
		},
		{
			Title:       "Validate configuration",
			Description: "Check for configuration issues",
			Command:     "cat " + pc.ConfigPath,
		},
	}

	// Add suggestions based on project analysis
	if len(pc.ProjectFiles.ComponentFiles) == 0 {
		suggestions = append(suggestions, ErrorSuggestion{
			Title:       "No components found",
			Description: "Initialize your project with example components",
			Command:     "templar init",
		})
	}

	return suggestions
}

// isSimilarName checks if two component names are similar.
func (pc *ProjectContext) isSimilarName(name1, name2 string) bool {
	name1 = strings.ToLower(name1)
	name2 = strings.ToLower(name2)

	// Simple similarity check
	return strings.Contains(name1, name2) || strings.Contains(name2, name1) ||
		levenshteinDistance(name1, name2) <= 2
}

// levenshteinDistance calculates the edit distance between two strings.
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}
			matrix[i][j] = min3(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

// min3 returns the minimum of three integers.
func min3(a, b, c int) int {
	if a <= b && a <= c {
		return a
	}
	if b <= c {
		return b
	}

	return c
}

// GetProjectSummary returns a summary of the project structure.
func (pc *ProjectContext) GetProjectSummary() string {
	summary := fmt.Sprintf("Project Analysis (analyzed %s ago):\n",
		time.Since(pc.analysisTime).Round(time.Second))

	summary += fmt.Sprintf("• Components: %d files\n", len(pc.ProjectFiles.ComponentFiles))
	summary += fmt.Sprintf("• Configuration: %s\n", pc.ConfigPath)
	summary += fmt.Sprintf("• Recent activity: %d files modified in last 7 days\n",
		len(pc.ProjectFiles.RecentlyModified))

	if pc.Config != nil {
		summary += fmt.Sprintf("• Scan paths: %s\n", strings.Join(pc.Config.Components.ScanPaths, ", "))
	}

	return summary
}
