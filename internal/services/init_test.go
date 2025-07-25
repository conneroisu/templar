package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitService_InitProject(t *testing.T) {
	service := NewInitService()

	tests := []struct {
		name    string
		opts    InitOptions
		wantErr bool
	}{
		{
			name: "default_initialization",
			opts: InitOptions{
				ProjectDir: "test-project",
				Minimal:    false,
				Example:    true,
				Template:   "",
				Wizard:     false,
			},
			wantErr: false,
		},
		{
			name: "minimal_initialization",
			opts: InitOptions{
				ProjectDir: "test-minimal",
				Minimal:    true,
				Example:    false,
				Template:   "",
				Wizard:     false,
			},
			wantErr: false,
		},
		{
			name: "template_initialization",
			opts: InitOptions{
				ProjectDir: "test-template",
				Minimal:    false,
				Example:    false,
				Template:   "blog",
				Wizard:     false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			tt.opts.ProjectDir = filepath.Join(tempDir, tt.opts.ProjectDir)

			err := service.InitProject(tt.opts)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify basic directory structure
			expectedDirs := []string{
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

			for _, dir := range expectedDirs {
				assert.DirExists(t, filepath.Join(tt.opts.ProjectDir, dir))
			}

			// Verify config file
			assert.FileExists(t, filepath.Join(tt.opts.ProjectDir, ".templar.yml"))

			// Verify go.mod
			assert.FileExists(t, filepath.Join(tt.opts.ProjectDir, "go.mod"))

			// Check example components based on options
			if tt.opts.Example || (!tt.opts.Minimal && tt.opts.Template == "") {
				assert.FileExists(t, filepath.Join(tt.opts.ProjectDir, "components", "button.templ"))
				assert.FileExists(t, filepath.Join(tt.opts.ProjectDir, "components", "card.templ"))
				assert.FileExists(t, filepath.Join(tt.opts.ProjectDir, "static", "css", "styles.css"))
			}

			if tt.opts.Minimal {
				assert.NoFileExists(t, filepath.Join(tt.opts.ProjectDir, "components", "button.templ"))
				assert.NoFileExists(t, filepath.Join(tt.opts.ProjectDir, "components", "card.templ"))
			}
		})
	}
}

func TestInitService_validateProjectDirectory(t *testing.T) {
	service := NewInitService()

	tests := []struct {
		name         string
		projectDir   string
		wantErr      bool
		setupDir     bool
		makeReadOnly bool
	}{
		{
			name:       "valid_new_directory",
			projectDir: "new-project",
			wantErr:    false,
			setupDir:   false,
		},
		{
			name:       "existing_directory",
			projectDir: "existing-project",
			wantErr:    false,
			setupDir:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			projectPath := filepath.Join(tempDir, tt.projectDir)

			if tt.setupDir {
				err := os.MkdirAll(projectPath, 0755)
				require.NoError(t, err)
			}

			if tt.makeReadOnly {
				defer func() {
					os.Chmod(projectPath, 0755) // Restore permissions for cleanup
				}()
				os.Chmod(filepath.Dir(projectPath), 0444)
			}

			err := service.validateProjectDirectory(projectPath)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.DirExists(t, projectPath)
			}
		})
	}
}

func TestInitService_createDirectoryStructure(t *testing.T) {
	service := NewInitService()

	tempDir := t.TempDir()

	err := service.createDirectoryStructure(tempDir)
	require.NoError(t, err)

	expectedDirs := []string{
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

	for _, dir := range expectedDirs {
		assert.DirExists(t, filepath.Join(tempDir, dir))
	}
}

func TestInitService_createConfigFile(t *testing.T) {
	service := NewInitService()

	tempDir := t.TempDir()

	err := service.createConfigFile(tempDir)
	require.NoError(t, err)

	configPath := filepath.Join(tempDir, ".templar.yml")
	assert.FileExists(t, configPath)

	// Check content
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	configStr := string(content)
	assert.Contains(t, configStr, "server:")
	assert.Contains(t, configStr, "port: 8080")
	assert.Contains(t, configStr, "host: localhost")
	assert.Contains(t, configStr, "components:")
	assert.Contains(t, configStr, "scan_paths:")
	assert.Contains(t, configStr, "build:")
	assert.Contains(t, configStr, "command: templ generate")
	assert.Contains(t, configStr, "development:")
	assert.Contains(t, configStr, "hot_reload: true")
}

func TestInitService_createGoModule(t *testing.T) {
	service := NewInitService()

	tests := []struct {
		name        string
		existingMod bool
		wantErr     bool
	}{
		{
			name:        "new_module",
			existingMod: false,
			wantErr:     false,
		},
		{
			name:        "existing_module",
			existingMod: true,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			if tt.existingMod {
				// Create existing go.mod
				existingContent := `module existing-project

go 1.21
`
				err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(existingContent), 0644)
				require.NoError(t, err)
			}

			err := service.createGoModule(tempDir)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			goModPath := filepath.Join(tempDir, "go.mod")
			assert.FileExists(t, goModPath)

			// Check content
			content, err := os.ReadFile(goModPath)
			require.NoError(t, err)

			contentStr := string(content)
			assert.Contains(t, contentStr, "module")
			assert.Contains(t, contentStr, "go 1.21")

			if !tt.existingMod {
				// Only check for templ dependency if we created the module
				assert.Contains(t, contentStr, "github.com/a-h/templ")
			}
		})
	}
}

func TestInitService_createExampleComponents(t *testing.T) {
	service := NewInitService()

	tempDir := t.TempDir()

	// Create required directory structure
	err := service.createDirectoryStructure(tempDir)
	require.NoError(t, err)

	err = service.createExampleComponents(tempDir)
	require.NoError(t, err)

	// Check example component files
	buttonPath := filepath.Join(tempDir, "components", "button.templ")
	assert.FileExists(t, buttonPath)

	buttonContent, err := os.ReadFile(buttonPath)
	require.NoError(t, err)
	assert.Contains(t, string(buttonContent), "templ Button")
	assert.Contains(t, string(buttonContent), "class={ \"btn\"")

	cardPath := filepath.Join(tempDir, "components", "card.templ")
	assert.FileExists(t, cardPath)

	cardContent, err := os.ReadFile(cardPath)
	require.NoError(t, err)
	assert.Contains(t, string(cardContent), "templ Card")
	assert.Contains(t, string(cardContent), "class=\"card\"")

	// Check CSS file
	cssPath := filepath.Join(tempDir, "static", "css", "styles.css")
	assert.FileExists(t, cssPath)

	cssContent, err := os.ReadFile(cssPath)
	require.NoError(t, err)
	assert.Contains(t, string(cssContent), ".btn {")
	assert.Contains(t, string(cssContent), ".card {")
}

func TestInitService_createFromTemplate(t *testing.T) {
	service := NewInitService()

	tests := []struct {
		name     string
		template string
		wantErr  bool
	}{
		{
			name:     "minimal_template",
			template: "minimal",
			wantErr:  false,
		},
		{
			name:     "blog_template",
			template: "blog",
			wantErr:  false,
		},
		{
			name:     "unknown_template",
			template: "unknown",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			// Create required directory structure
			err := service.createDirectoryStructure(tempDir)
			require.NoError(t, err)

			err = service.createFromTemplate(tempDir, tt.template)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
