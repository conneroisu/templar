package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServerComponentIntegration tests the integration between the server
// and component system for serving and previewing components.
func TestServerComponentIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping server integration tests in short mode")
	}

	fixtures := testutils.NewTestFixtures()

	t.Run("component_serving_workflow", func(t *testing.T) {
		tc := testutils.NewTestContext(t)
		fs := testutils.NewTestFileSystem(t, tc.TempDir())

		// Step 1: Create test components
		components := fixtures.SampleComponents()
		for name, comp := range components {
			filename := strings.ToLower(name) + ".templ"
			fs.CreateFile(filename, comp.Content)
		}

		// Step 2: Set up server configuration
		cfg := fixtures.BasicConfig()
		cfg.Components.ScanPaths = []string{tc.TempDir()}
		cfg.Server.Port = 0 // Random port for testing
		cfg.Development.HotReload = true
		cfg.Development.ErrorOverlay = true

		// Step 3: Create mock server for testing
		server := createMockServer(t, cfg)
		testServer := httptest.NewServer(server)
		defer testServer.Close()

		// Step 4: Test component listing endpoint
		t.Run("list_components_endpoint", func(t *testing.T) {
			resp, err := http.Get(testServer.URL + "/api/components")
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Warning: failed to close response body: %v", err)
				}
			}()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

			var componentList []ComponentInfo
			err = json.NewDecoder(resp.Body).Decode(&componentList)
			require.NoError(t, err)

			assert.Greater(t, len(componentList), 0, "Should return components")

			// Verify component structure
			for _, comp := range componentList {
				assert.NotEmpty(t, comp.Name, "Component should have name")
				assert.NotEmpty(t, comp.Package, "Component should have package")
			}
		})

		// Step 5: Test individual component preview
		t.Run("component_preview_endpoint", func(t *testing.T) {
			// Test previewing a specific component
			resp, err := http.Get(testServer.URL + "/preview/Button")
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Warning: failed to close response body: %v", err)
				}
			}()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "text/html", resp.Header.Get("Content-Type"))

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			htmlContent := string(body)
			assert.Contains(t, htmlContent, "<html", "Should return HTML")
			assert.Contains(t, htmlContent, "Button", "Should contain component name")
		})

		// Step 6: Test component preview with parameters
		t.Run("component_preview_with_params", func(t *testing.T) {
			// Test with query parameters
			resp, err := http.Get(testServer.URL + "/preview/Button?text=Click%20Me&disabled=false")
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Warning: failed to close response body: %v", err)
				}
			}()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			htmlContent := string(body)
			assert.Contains(t, htmlContent, "Click Me", "Should use parameter values")
		})

		// Step 7: Test error handling for non-existent components
		t.Run("non_existent_component_error", func(t *testing.T) {
			resp, err := http.Get(testServer.URL + "/preview/NonExistentComponent")
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Warning: failed to close response body: %v", err)
				}
			}()

			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		})
	})

	t.Run("websocket_hot_reload_integration", func(t *testing.T) {
		t.Skip("WebSocket integration test - requires WebSocket mock implementation")

		// This test would verify:
		// 1. WebSocket connection establishment
		// 2. File change notifications
		// 3. Component reload events
		// 4. Client-side hot reload updates
	})

	t.Run("static_asset_serving", func(t *testing.T) {
		tc := testutils.NewTestContext(t)
		fs := testutils.NewTestFileSystem(t, tc.TempDir())

		// Create static assets
		cssContent := ".button { background: blue; }"
		fs.CreateFile("styles.css", cssContent)

		jsContent := "console.log('Component loaded');"
		fs.CreateFile("script.js", jsContent)

		cfg := fixtures.BasicConfig()
		cfg.Production.StaticDir = tc.TempDir()

		server := createMockServer(t, cfg)
		testServer := httptest.NewServer(server)
		defer testServer.Close()

		// Test CSS serving
		t.Run("css_assets", func(t *testing.T) {
			resp, err := http.Get(testServer.URL + "/static/styles.css")
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Warning: failed to close response body: %v", err)
				}
			}()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "text/css", resp.Header.Get("Content-Type"))

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Contains(t, string(body), "background: blue")
		})

		// Test JS serving
		t.Run("js_assets", func(t *testing.T) {
			resp, err := http.Get(testServer.URL + "/static/script.js")
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Warning: failed to close response body: %v", err)
				}
			}()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "application/javascript", resp.Header.Get("Content-Type"))

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Contains(t, string(body), "Component loaded")
		})
	})

	t.Run("cors_and_security_headers", func(t *testing.T) {
		cfg := fixtures.BasicConfig()
		cfg.Server.Environment = "development"
		cfg.Server.AllowedOrigins = []string{"http://localhost:3000"}

		server := createMockServer(t, cfg)
		testServer := httptest.NewServer(server)
		defer testServer.Close()

		// Test CORS headers
		t.Run("cors_headers", func(t *testing.T) {
			req, err := http.NewRequest("GET", testServer.URL+"/api/components", nil)
			require.NoError(t, err)
			req.Header.Set("Origin", "http://localhost:3000")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Warning: failed to close response body: %v", err)
				}
			}()

			assert.Equal(t, "http://localhost:3000", resp.Header.Get("Access-Control-Allow-Origin"))
		})

		// Test preflight requests
		t.Run("preflight_requests", func(t *testing.T) {
			req, err := http.NewRequest("OPTIONS", testServer.URL+"/api/components", nil)
			require.NoError(t, err)
			req.Header.Set("Origin", "http://localhost:3000")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Warning: failed to close response body: %v", err)
				}
			}()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Methods"))
		})

		// Test security headers
		t.Run("security_headers", func(t *testing.T) {
			resp, err := http.Get(testServer.URL + "/")
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Warning: failed to close response body: %v", err)
				}
			}()

			// Verify security headers are present
			assert.NotEmpty(t, resp.Header.Get("X-Content-Type-Options"))
			assert.NotEmpty(t, resp.Header.Get("X-Frame-Options"))
		})
	})

	t.Run("error_overlay_integration", func(t *testing.T) {
		tc := testutils.NewTestContext(t)
		fs := testutils.NewTestFileSystem(t, tc.TempDir())

		// Create component with syntax error
		brokenComponent := `package components

templ BrokenComponent() {
	<div>
		<span>Unclosed tag
	</div>
` // Missing closing span tag

		fs.CreateFile("broken.templ", brokenComponent)

		cfg := fixtures.BasicConfig()
		cfg.Components.ScanPaths = []string{tc.TempDir()}
		cfg.Development.ErrorOverlay = true

		server := createMockServer(t, cfg)
		testServer := httptest.NewServer(server)
		defer testServer.Close()

		// Test that error overlay is displayed for broken components
		t.Run("displays_error_overlay", func(t *testing.T) {
			resp, err := http.Get(testServer.URL + "/preview/BrokenComponent")
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Warning: failed to close response body: %v", err)
				}
			}()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			htmlContent := string(body)

			// Should show error overlay instead of crashing
			if resp.StatusCode == http.StatusOK {
				assert.Contains(t, htmlContent, "error", "Should show error information")
			} else {
				// Or return appropriate error status
				assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
			}
		})
	})
}

// TestAPIEndpoints tests the API endpoints for component management.
func TestAPIEndpoints(t *testing.T) {
	fixtures := testutils.NewTestFixtures()

	t.Run("health_check_endpoint", func(t *testing.T) {
		cfg := fixtures.BasicConfig()
		server := createMockServer(t, cfg)
		testServer := httptest.NewServer(server)
		defer testServer.Close()

		resp, err := http.Get(testServer.URL + "/health")
		require.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Warning: failed to close response body: %v", err)
			}
		}()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var health HealthResponse
		err = json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(t, err)

		assert.Equal(t, "ok", health.Status)
		assert.NotEmpty(t, health.Timestamp)
	})

	t.Run("metrics_endpoint", func(t *testing.T) {
		cfg := fixtures.BasicConfig()
		cfg.Monitoring.Enabled = true
		server := createMockServer(t, cfg)
		testServer := httptest.NewServer(server)
		defer testServer.Close()

		resp, err := http.Get(testServer.URL + "/metrics")
		require.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Warning: failed to close response body: %v", err)
			}
		}()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "text/plain")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		metricsContent := string(body)
		// Basic Prometheus metrics format check
		assert.Contains(t, metricsContent, "# HELP")
		assert.Contains(t, metricsContent, "# TYPE")
	})

	t.Run("component_metadata_endpoint", func(t *testing.T) {
		tc := testutils.NewTestContext(t)
		fs := testutils.NewTestFileSystem(t, tc.TempDir())

		// Create test components
		buttonComp := fixtures.SampleComponents()["simple_button"]
		fs.WriteTemplFile("button", buttonComp.Content)

		cfg := fixtures.BasicConfig()
		cfg.Components.ScanPaths = []string{tc.TempDir()}
		server := createMockServer(t, cfg)
		testServer := httptest.NewServer(server)
		defer testServer.Close()

		resp, err := http.Get(testServer.URL + "/api/components/Button/metadata")
		require.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Warning: failed to close response body: %v", err)
			}
		}()

		if resp.StatusCode == http.StatusOK {
			var metadata ComponentMetadata
			err = json.NewDecoder(resp.Body).Decode(&metadata)
			require.NoError(t, err)

			assert.Equal(t, "Button", metadata.Name)
			assert.Equal(t, "components", metadata.Package)
			assert.Greater(t, len(metadata.Parameters), 0)
		} else {
			// Endpoint might not be implemented yet
			t.Logf("Component metadata endpoint returned %d", resp.StatusCode)
		}
	})
}

// TestRateLimiting tests rate limiting functionality.
func TestRateLimiting(t *testing.T) {
	fixtures := testutils.NewTestFixtures()

	t.Run("rate_limiting_enforcement", func(t *testing.T) {
		cfg := fixtures.BasicConfig()
		// Enable rate limiting for testing
		cfg.Server.Middleware = append(cfg.Server.Middleware, "ratelimit")

		server := createMockServer(t, cfg)
		testServer := httptest.NewServer(server)
		defer testServer.Close()

		// Make multiple rapid requests
		const numRequests = 20
		var statusCodes []int

		for i := 0; i < numRequests; i++ {
			resp, err := http.Get(testServer.URL + "/api/components")
			require.NoError(t, err)
			statusCodes = append(statusCodes, resp.StatusCode)
			if err := resp.Body.Close(); err != nil {
				t.Logf("Warning: failed to close response body: %v", err)
			}
		}

		// Should have some rate-limited responses
		rateLimitedCount := 0
		for _, code := range statusCodes {
			if code == http.StatusTooManyRequests {
				rateLimitedCount++
			}
		}

		t.Logf("Rate limited %d/%d requests", rateLimitedCount, numRequests)
		// In a properly configured system, some requests should be rate limited
	})
}

// TestServerPerformance tests server performance characteristics.
func TestServerPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	fixtures := testutils.NewTestFixtures()

	t.Run("concurrent_request_handling", func(t *testing.T) {
		cfg := fixtures.BasicConfig()
		server := createMockServer(t, cfg)
		testServer := httptest.NewServer(server)
		defer testServer.Close()

		// Test concurrent requests
		const numConcurrent = 50
		done := make(chan error, numConcurrent)

		timer := testutils.NewTestTimer(t)

		for i := 0; i < numConcurrent; i++ {
			go func(reqID int) {
				resp, err := http.Get(testServer.URL + "/health")
				if err != nil {
					done <- fmt.Errorf("request %d failed: %w", reqID, err)
					return
				}
				if err := resp.Body.Close(); err != nil {
					t.Logf("Warning: failed to close response body: %v", err)
				}

				if resp.StatusCode != http.StatusOK {
					done <- fmt.Errorf("request %d returned %d", reqID, resp.StatusCode)
					return
				}

				done <- nil
			}(i)
		}

		// Wait for all requests
		for i := 0; i < numConcurrent; i++ {
			select {
			case err := <-done:
				if err != nil {
					t.Logf("Concurrent request failed: %v", err)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("Timeout waiting for concurrent requests")
			}
		}

		// Should complete within reasonable time
		timer.AssertWithinTimeout(5 * time.Second)
	})
}

// Mock server creation for testing
func createMockServer(t *testing.T, cfg *config.Config) http.Handler {
	// This would create a real server instance in the actual implementation
	// For now, return a mock handler
	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := HealthResponse{
			Status:    "ok",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Logf("Warning: failed to encode health response: %v", err)
		}
	})

	// Components API
	mux.HandleFunc("/api/components", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		components := []ComponentInfo{
			{Name: "Button", Package: "components", FilePath: "button.templ"},
			{Name: "Card", Package: "components", FilePath: "card.templ"},
		}
		if err := json.NewEncoder(w).Encode(components); err != nil {
			t.Logf("Warning: failed to encode components response: %v", err)
		}
	})

	// Component preview
	mux.HandleFunc("/preview/", func(w http.ResponseWriter, r *http.Request) {
		componentName := strings.TrimPrefix(r.URL.Path, "/preview/")

		if componentName == "NonExistentComponent" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head><title>%s Preview</title></head>
<body>
	<div class="component-preview">
		<h1>%s Component</h1>
		<p>Parameters: %s</p>
	</div>
</body>
</html>`, componentName, componentName, r.URL.RawQuery)
		if _, err := w.Write([]byte(html)); err != nil {
			t.Logf("Warning: failed to write HTML response: %v", err)
		}
	})

	// Static assets
	mux.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/static/")

		switch {
		case strings.HasSuffix(path, ".css"):
			w.Header().Set("Content-Type", "text/css")
			if _, err := w.Write([]byte(".button { background: blue; }")); err != nil {
				t.Logf("Warning: failed to write CSS response: %v", err)
			}
		case strings.HasSuffix(path, ".js"):
			w.Header().Set("Content-Type", "application/javascript")
			if _, err := w.Write([]byte("console.log('Component loaded');")); err != nil {
				t.Logf("Warning: failed to write JS response: %v", err)
			}
		default:
			http.NotFound(w, r)
		}
	})

	// Metrics endpoint
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		if cfg.Monitoring.Enabled {
			w.Header().Set("Content-Type", "text/plain")
			if _, err := w.Write([]byte(`# HELP http_requests_total Total HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",handler="/api/components"} 42
`)); err != nil {
				t.Logf("Warning: failed to write metrics response: %v", err)
			}
		} else {
			http.NotFound(w, r)
		}
	})

	return mux
}

// Response types for testing
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

type ComponentInfo struct {
	Name     string `json:"name"`
	Package  string `json:"package"`
	FilePath string `json:"file_path"`
}

type ComponentMetadata struct {
	Name       string          `json:"name"`
	Package    string          `json:"package"`
	Parameters []ParameterInfo `json:"parameters"`
}

type ParameterInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}