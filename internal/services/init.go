package services

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/errors"
)

// InitService handles project initialization business logic
type InitService struct {
	// Dependencies can be added here in the future
}

// NewInitService creates a new initialization service
func NewInitService() *InitService {
	return &InitService{}
}

// InitOptions contains options for project initialization
type InitOptions struct {
	ProjectDir string
	Minimal    bool
	Example    bool
	Template   string
	Wizard     bool
}

// InitProject initializes a new Templar project with the specified options
func (s *InitService) InitProject(opts InitOptions) error {
	// Validate project directory
	if err := s.validateProjectDirectory(opts.ProjectDir); err != nil {
		return errors.InitError("VALIDATE_DIR", "project directory validation failed", err)
	}

	// Create directory structure
	if err := s.createDirectoryStructure(opts.ProjectDir); err != nil {
		return errors.InitError("CREATE_DIRS", "directory structure creation failed", err)
	}

	// Create configuration file
	if opts.Wizard {
		if err := s.createConfigWithWizard(opts.ProjectDir); err != nil {
			return errors.InitError("CREATE_CONFIG_WIZARD", "configuration creation with wizard failed", err)
		}
	} else {
		if err := s.createConfigFile(opts.ProjectDir); err != nil {
			return errors.InitError("CREATE_CONFIG", "configuration file creation failed", err)
		}
	}

	// Create Go module if it doesn't exist
	if err := s.createGoModule(opts.ProjectDir); err != nil {
		return errors.InitError("CREATE_MODULE", "Go module creation failed", err)
	}

	// Create example components if requested
	if opts.Example || (!opts.Minimal && opts.Template == "") {
		if err := s.createExampleComponents(opts.ProjectDir); err != nil {
			return errors.InitError("CREATE_EXAMPLES", "example components creation failed", err)
		}
	}

	// Create template files if template is specified
	if opts.Template != "" {
		if err := s.createFromTemplate(opts.ProjectDir, opts.Template); err != nil {
			return errors.InitError("CREATE_TEMPLATE", fmt.Sprintf("template '%s' creation failed", opts.Template), err)
		}
	}

	return nil
}

// validateProjectDirectory ensures the project directory is valid
func (s *InitService) validateProjectDirectory(projectDir string) error {
	// Check if directory exists and is writable
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return errors.FileOperationError("CREATE_DIR", projectDir, "cannot create project directory", err)
	}
	return nil
}

// createDirectoryStructure creates the standard Templar project directory structure
func (s *InitService) createDirectoryStructure(projectDir string) error {
	dirs := []string{
		"components",
		"views", 
		"examples",
		"static",
		"static/css",
		"static/js",
		"static/images",
		"mocks",
		"preview",
		".templar",
		".templar/cache",
	}

	for _, dir := range dirs {
		dirPath := filepath.Join(projectDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return errors.FileOperationError("CREATE_DIR", dirPath, fmt.Sprintf("failed to create directory %s", dir), err)
		}
	}

	return nil
}

// createConfigFile creates a default configuration file
func (s *InitService) createConfigFile(projectDir string) error {
	cfg := &config.Config{}
	// Apply defaults manually since we don't have DefaultConfig
	cfg.Server.Port = 8080
	cfg.Server.Host = "localhost"
	cfg.Server.Open = true
	cfg.Components.ScanPaths = []string{"components", "views", "examples"}
	cfg.Build.Command = "templ generate"
	cfg.Build.Watch = []string{"**/*.templ"}
	cfg.Build.Ignore = []string{"node_modules", ".git"}
	cfg.Build.CacheDir = ".templar/cache"
	cfg.Development.HotReload = true
	cfg.Development.CSSInjection = true
	cfg.Development.ErrorOverlay = true

	// Write config file
	configPath := filepath.Join(projectDir, ".templar.yml")
	return s.writeConfigToFile(cfg, configPath)
}

// writeConfigToFile writes config to YAML file
func (s *InitService) writeConfigToFile(cfg *config.Config, path string) error {
	content := fmt.Sprintf(`server:
  port: %d
  host: %s
  open: %t

components:
  scan_paths:
    - %s
    - %s
    - %s

build:
  command: %s
  watch:
    - %s
  ignore:
    - %s
    - %s
  cache_dir: %s

development:
  hot_reload: %t
  css_injection: %t
  error_overlay: %t
`,
		cfg.Server.Port, cfg.Server.Host, cfg.Server.Open,
		cfg.Components.ScanPaths[0], cfg.Components.ScanPaths[1], cfg.Components.ScanPaths[2],
		cfg.Build.Command, cfg.Build.Watch[0], cfg.Build.Ignore[0], cfg.Build.Ignore[1],
		cfg.Build.CacheDir, cfg.Development.HotReload, cfg.Development.CSSInjection, cfg.Development.ErrorOverlay)

	return os.WriteFile(path, []byte(content), 0644)
}

// createConfigWithWizard creates configuration using interactive wizard
func (s *InitService) createConfigWithWizard(projectDir string) error {
	// For now, use default config - wizard implementation would go here
	return s.createConfigFile(projectDir)
}

// createGoModule creates a Go module if it doesn't exist
func (s *InitService) createGoModule(projectDir string) error {
	modFile := filepath.Join(projectDir, "go.mod")
	if _, err := os.Stat(modFile); err == nil {
		return nil // go.mod already exists
	}

	// Create a basic go.mod file
	content := fmt.Sprintf(`module %s

go 1.21

require (
	github.com/a-h/templ v0.2.680
)
`, filepath.Base(projectDir))

	return os.WriteFile(modFile, []byte(content), 0644)
}

// createExampleComponents creates example component files
func (s *InitService) createExampleComponents(projectDir string) error {
	// Example button component
	buttonContent := `package components

templ Button(text string, variant string) {
	<button class={ "btn", "btn-" + variant } type="button">
		{ text }
	</button>
}
`

	buttonPath := filepath.Join(projectDir, "components", "button.templ")
	if err := os.WriteFile(buttonPath, []byte(buttonContent), 0644); err != nil {
		return errors.FileOperationError("CREATE_COMPONENT", buttonPath, "failed to create button component", err)
	}

	// Example card component  
	cardContent := `package components

templ Card(title string, content string) {
	<div class="card">
		<div class="card-header">
			<h3>{ title }</h3>
		</div>
		<div class="card-body">
			<p>{ content }</p>
		</div>
	</div>
}
`

	cardPath := filepath.Join(projectDir, "components", "card.templ")
	if err := os.WriteFile(cardPath, []byte(cardContent), 0644); err != nil {
		return errors.FileOperationError("CREATE_COMPONENT", cardPath, "failed to create card component", err)
	}

	// Create basic CSS file
	cssContent := `.btn {
  padding: 0.5rem 1rem;
  border: 1px solid #ccc;
  border-radius: 0.25rem;
  cursor: pointer;
}

.btn-primary {
  background-color: #007bff;
  color: white;
  border-color: #007bff;
}

.btn-secondary {
  background-color: #6c757d;
  color: white;
  border-color: #6c757d;
}

.card {
  border: 1px solid #dee2e6;
  border-radius: 0.375rem;
  margin-bottom: 1rem;
}

.card-header {
  padding: 0.75rem 1.25rem;
  background-color: #f8f9fa;
  border-bottom: 1px solid #dee2e6;
}

.card-body {
  padding: 1.25rem;
}
`

	cssPath := filepath.Join(projectDir, "static", "css", "styles.css")
	return os.WriteFile(cssPath, []byte(cssContent), 0644)
}

// createFromTemplate creates files from the specified template
func (s *InitService) createFromTemplate(projectDir, template string) error {
	switch template {
	case "minimal":
		return nil // Already handled by not creating examples
	case "blog", "dashboard", "landing", "ecommerce", "documentation":
		// For now, just create basic structure - full templates would be implemented here
		return s.createExampleComponents(projectDir)
	default:
		return errors.ValidationFailure("template", "unknown template specified", template, "Use one of: minimal, blog, dashboard, landing, ecommerce, documentation")
	}
}