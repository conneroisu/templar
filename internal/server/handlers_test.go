package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestServer(t *testing.T) *PreviewServer {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Components: config.ComponentsConfig{
			ScanPaths: []string{"./test"},
		},
	}

	reg := registry.NewComponentRegistry()
	
	// Add test components
	testComponent := &registry.ComponentInfo{
		Name:     "TestButton",
		FilePath: "/test/button.templ",
		Package:  "main",
		Parameters: []registry.ParameterInfo{
			{Name: "text", Type: "string"},
			{Name: "variant", Type: "string"},
		},
	}
	reg.Register(testComponent)

	server := &PreviewServer{
		config:   cfg,
		registry: reg,
		renderer: renderer.NewComponentRenderer(reg),
	}

	return server
}

func TestHandleIndex(t *testing.T) {
	server := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	server.handleIndex(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/html", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "Templar Component Preview")
	assert.Contains(t, w.Body.String(), "<!DOCTYPE html>")
}

func TestHandleComponents(t *testing.T) {
	server := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/components", nil)
	w := httptest.NewRecorder()

	server.handleComponents(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var components []*registry.ComponentInfo
	err := json.Unmarshal(w.Body.Bytes(), &components)
	require.NoError(t, err)
	
	assert.Len(t, components, 1)
	assert.Equal(t, "TestButton", components[0].Name)
	assert.Equal(t, "/test/button.templ", components[0].FilePath)
	assert.Len(t, components[0].Parameters, 2)
}

func TestHandleComponent(t *testing.T) {
	server := setupTestServer(t)

	t.Run("valid component", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/component/TestButton", nil)
		w := httptest.NewRecorder()

		server.handleComponent(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var component registry.ComponentInfo
		err := json.Unmarshal(w.Body.Bytes(), &component)
		require.NoError(t, err)
		
		assert.Equal(t, "TestButton", component.Name)
		assert.Equal(t, "/test/button.templ", component.FilePath)
	})

	t.Run("invalid component name", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/component/../etc/passwd", nil)
		w := httptest.NewRecorder()

		server.handleComponent(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid component name")
	})

	t.Run("nonexistent component", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/component/NonExistent", nil)
		w := httptest.NewRecorder()

		server.handleComponent(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("component name with dangerous characters", func(t *testing.T) {
		dangerousNames := []string{
			"test<script>",
			"test&lt;script&gt;",
			"test\"onclick=\"alert(1)\"",
			"test$(whoami)",
			"test`rm -rf /`",
			"test;malicious",
			"test|evil",
		}

		for _, name := range dangerousNames {
			t.Run(fmt.Sprintf("dangerous name: %s", name), func(t *testing.T) {
				// URL encode the dangerous name to prevent HTTP parsing issues
				encodedName := url.QueryEscape(name)
				req := httptest.NewRequest(http.MethodGet, "/component/"+encodedName, nil)
				w := httptest.NewRecorder()

				server.handleComponent(w, req)

				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.Contains(t, w.Body.String(), "Invalid component name")
			})
		}
	})
}

func TestHandleStatic(t *testing.T) {
	server := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/static/test.css", nil)
	w := httptest.NewRecorder()

	server.handleStatic(w, req)

	// Currently returns 404 as static handling is not implemented
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleRender(t *testing.T) {
	server := setupTestServer(t)

	t.Run("missing component name", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/render/", nil)
		w := httptest.NewRecorder()

		server.handleRender(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Component name required")
	})

	t.Run("nonexistent component", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/render/NonExistent", nil)
		w := httptest.NewRecorder()

		server.handleRender(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Error rendering component")
	})

	// Note: Testing successful rendering would require actual templ files and Go environment
	// which is complex for unit tests. Integration tests would be better suited for this.
}

func TestHandleTargetFiles(t *testing.T) {
	t.Run("single target file", func(t *testing.T) {
		cfg := &config.Config{
			Server: config.ServerConfig{
				Host: "localhost",
				Port: 8080,
			},
			TargetFiles: []string{"test.templ"},
		}

		server := &PreviewServer{
			config:   cfg,
			registry: registry.NewComponentRegistry(),
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		server.handleTargetFiles(w, req)

		// Since we don't have actual scanner setup, this will likely error
		// but we can verify the handler is called correctly
		assert.NotEqual(t, http.StatusOK, w.Code) // Expected to fail without proper setup
	})

	t.Run("multiple target files", func(t *testing.T) {
		cfg := &config.Config{
			Server: config.ServerConfig{
				Host: "localhost",
				Port: 8080,
			},
			TargetFiles: []string{"test1.templ", "test2.templ"},
		}

		server := &PreviewServer{
			config:   cfg,
			registry: registry.NewComponentRegistry(),
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		server.handleTargetFiles(w, req)

		// This should render file selection interface
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "text/html", w.Header().Get("Content-Type"))
		assert.Contains(t, w.Body.String(), "Select File to Preview")
		assert.Contains(t, w.Body.String(), "test1.templ")
		assert.Contains(t, w.Body.String(), "test2.templ")
	})
}

func TestValidateComponentName(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorText   string
	}{
		{
			name:        "valid component name",
			input:       "Button",
			expectError: false,
		},
		{
			name:        "valid component name with underscore",
			input:       "My_Button",
			expectError: false,
		},
		{
			name:        "empty name",
			input:       "",
			expectError: true,
			errorText:   "empty component name",
		},
		{
			name:        "path traversal attempt",
			input:       "../etc/passwd",
			expectError: true,
			errorText:   "path traversal attempt detected",
		},
		{
			name:        "absolute path",
			input:       "/etc/passwd",
			expectError: true,
			errorText:   "absolute path not allowed",
		},
		{
			name:        "path separator",
			input:       "components/Button",
			expectError: true,
			errorText:   "path separators not allowed",
		},
		{
			name:        "dangerous character - script tag",
			input:       "Button<script>",
			expectError: true,
			errorText:   "dangerous character not allowed",
		},
		{
			name:        "dangerous character - quote",
			input:       "Button\"onclick=\"alert(1)\"",
			expectError: true,
			errorText:   "dangerous character not allowed",
		},
		{
			name:        "dangerous character - semicolon",
			input:       "Button; rm -rf /",
			expectError: true,
			errorText:   "dangerous character not allowed",
		},
		{
			name:        "dangerous character - backtick",
			input:       "Button`whoami`",
			expectError: true,
			errorText:   "dangerous character not allowed",
		},
		{
			name:        "dangerous character - dollar",
			input:       "Button$(malicious)",
			expectError: true,
			errorText:   "dangerous character not allowed",
		},
		{
			name:        "name too long",
			input:       strings.Repeat("a", 101),
			expectError: true,
			errorText:   "component name too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateComponentName(tt.input)
			
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorText != "" {
					assert.Contains(t, err.Error(), tt.errorText)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRenderComponentSelection(t *testing.T) {
	server := setupTestServer(t)

	components := []*registry.ComponentInfo{
		{
			Name:       "Button",
			Parameters: []registry.ParameterInfo{{Name: "text", Type: "string"}},
		},
		{
			Name:       "Card",
			Parameters: []registry.ParameterInfo{{Name: "title", Type: "string"}, {Name: "content", Type: "string"}},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	server.renderComponentSelection(w, req, components, "test.templ")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/html", w.Header().Get("Content-Type"))
	
	body := w.Body.String()
	assert.Contains(t, body, "Select Component from test.templ")
	assert.Contains(t, body, "Button")
	assert.Contains(t, body, "Card")
	assert.Contains(t, body, "1 parameters")
	assert.Contains(t, body, "2 parameters")
	assert.Contains(t, body, "/render/Button")
	assert.Contains(t, body, "/render/Card")
}

func TestRenderFileSelection(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		TargetFiles: []string{"button.templ", "card.templ", "layout.templ"},
	}

	server := &PreviewServer{
		config:   cfg,
		registry: registry.NewComponentRegistry(),
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	server.renderFileSelection(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/html", w.Header().Get("Content-Type"))
	
	body := w.Body.String()
	assert.Contains(t, body, "Select File to Preview")
	assert.Contains(t, body, "button.templ")
	assert.Contains(t, body, "card.templ")
	assert.Contains(t, body, "layout.templ")
	assert.Contains(t, body, "?file=button.templ")
	assert.Contains(t, body, "?file=card.templ")
	assert.Contains(t, body, "?file=layout.templ")
}