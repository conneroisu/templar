// Package apidocs provides OpenAPI specification generation from extracted API information.
//
// This file implements the main API documentation generator that orchestrates
// the extraction, analysis, and generation process to produce comprehensive
// OpenAPI specifications with interactive documentation capabilities.
package apidocs

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// APIDocumentationGenerator coordinates the complete API documentation generation process.
//
// Architecture: Combines extraction, analysis, and generation into a cohesive
// system that produces OpenAPI specifications in multiple formats (JSON, YAML)
// and provides interactive documentation through Swagger UI integration.
type APIDocumentationGenerator struct {
	// extractor analyzes source code to discover API endpoints
	extractor *APIExtractor
	// config controls generation behavior and output options
	config *GenerationConfig
	// outputDir specifies where to write generated documentation
	outputDir string
}

// NewAPIDocumentationGenerator creates a new documentation generator.
//
// Initialization strategy: Sets up all necessary components with sensible
// defaults while allowing full customization through the configuration.
func NewAPIDocumentationGenerator(config *GenerationConfig) (*APIDocumentationGenerator, error) {
	if config == nil {
		config = DefaultGenerationConfig()
	}

	// Validate configuration
	if err := validateGenerationConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	extractor := NewAPIExtractor(config)

	return &APIDocumentationGenerator{
		extractor: extractor,
		config:    config,
		outputDir: config.OutputDir,
	}, nil
}

// Generate performs the complete API documentation generation process.
//
// Generation workflow:
// 1. Extract API endpoints from source code
// 2. Analyze types and generate schemas
// 3. Build complete OpenAPI specification
// 4. Write output in requested formats
// 5. Generate interactive documentation if enabled.
func (g *APIDocumentationGenerator) Generate(serverPackagePath string) (*GenerationResult, error) {
	log.Printf("Starting API documentation generation for: %s", serverPackagePath)
	startTime := time.Now()

	// Extract API information from source code
	result, err := g.extractor.ExtractAPIs(serverPackagePath)
	if err != nil {
		return nil, fmt.Errorf("API extraction failed: %w", err)
	}

	log.Printf("Extraction completed in %v", time.Since(startTime))

	// Write output files in requested formats
	if err := g.writeOutput(result.Specification); err != nil {
		return nil, fmt.Errorf("failed to write output: %w", err)
	}

	// Generate interactive documentation if enabled
	if g.config.ServeEnabled {
		if err := g.generateInteractiveDocumentation(); err != nil {
			log.Printf("Warning: failed to generate interactive documentation: %v", err)
			result.Warnings = append(result.Warnings,
				"Interactive documentation generation failed: "+err.Error())
		}
	}

	result.GeneratedAt = time.Now()
	log.Printf("API documentation generation completed in %v", time.Since(startTime))

	return result, nil
}

// writeOutput writes the OpenAPI specification in all requested formats.
func (g *APIDocumentationGenerator) writeOutput(spec *APISpecification) error {
	for _, format := range g.config.Formats {
		if err := g.writeFormatOutput(spec, format); err != nil {
			return fmt.Errorf("failed to write %s format: %w", format, err)
		}
	}

	return nil
}

// writeFormatOutput writes the specification in a specific format.
func (g *APIDocumentationGenerator) writeFormatOutput(spec *APISpecification, format string) error {
	switch format {
	case "json":
		return g.writeJSONOutput(spec)
	case "yaml":
		return g.writeYAMLOutput(spec)
	case "html":
		return g.writeHTMLOutput(spec)
	case "markdown":
		return g.writeMarkdownOutput(spec)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// writeJSONOutput writes the OpenAPI specification as JSON.
func (g *APIDocumentationGenerator) writeJSONOutput(spec *APISpecification) error {
	filePath := filepath.Join(g.outputDir, "openapi.json")

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create JSON file: %w", err)
	}
	defer func() { _ = file.Close() }()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(spec); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	log.Printf("Generated OpenAPI JSON: %s", filePath)

	return nil
}

// writeYAMLOutput writes the OpenAPI specification as YAML.
func (g *APIDocumentationGenerator) writeYAMLOutput(spec *APISpecification) error {
	filePath := filepath.Join(g.outputDir, "openapi.yaml")

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create YAML file: %w", err)
	}
	defer func() { _ = file.Close() }()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)

	if err := encoder.Encode(spec); err != nil {
		return fmt.Errorf("failed to encode YAML: %w", err)
	}

	log.Printf("Generated OpenAPI YAML: %s", filePath)

	return nil
}

// writeHTMLOutput generates HTML documentation with Swagger UI.
func (g *APIDocumentationGenerator) writeHTMLOutput(spec *APISpecification) error {
	htmlContent := g.generateSwaggerUIHTML(spec)
	filePath := filepath.Join(g.outputDir, "index.html")

	if err := os.WriteFile(filePath, []byte(htmlContent), 0600); err != nil {
		return fmt.Errorf("failed to write HTML file: %w", err)
	}

	log.Printf("Generated HTML documentation: %s", filePath)

	return nil
}

// writeMarkdownOutput generates Markdown documentation.
func (g *APIDocumentationGenerator) writeMarkdownOutput(spec *APISpecification) error {
	markdownContent := g.generateMarkdownDocumentation(spec)
	filePath := filepath.Join(g.outputDir, "README.md")

	if err := os.WriteFile(filePath, []byte(markdownContent), 0600); err != nil {
		return fmt.Errorf("failed to write Markdown file: %w", err)
	}

	log.Printf("Generated Markdown documentation: %s", filePath)

	return nil
}

// generateSwaggerUIHTML creates an HTML page with embedded Swagger UI.
func (g *APIDocumentationGenerator) generateSwaggerUIHTML(spec *APISpecification) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s - API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui.css" />
    <style>
        html {
            box-sizing: border-box;
            overflow: -moz-scrollbars-vertical;
            overflow-y: scroll;
        }
        
        *, *:before, *:after {
            box-sizing: inherit;
        }
        
        body {
            margin: 0;
            background: #fafafa;
        }
        
        .swagger-ui .topbar {
            background-color: #007acc;
        }
        
        .swagger-ui .topbar .download-url-wrapper .select-label {
            color: #fff;
        }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    
    <script src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: './openapi.json',
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                validatorUrl: null,
                tryItOutEnabled: true,
                supportedSubmitMethods: ['get', 'post', 'put', 'delete', 'patch'],
                onComplete: function() {
                    console.log('Swagger UI loaded successfully');
                },
                onFailure: function(data) {
                    console.error('Failed to load Swagger UI:', data);
                }
            });
        };
    </script>
</body>
</html>`, spec.Info.Title)
}

// generateMarkdownDocumentation creates comprehensive Markdown documentation.
func (g *APIDocumentationGenerator) generateMarkdownDocumentation(spec *APISpecification) string {
	md := fmt.Sprintf(`# %s

%s

**Version:** %s

`, spec.Info.Title, spec.Info.Description, spec.Info.Version)

	// Add server information
	if len(spec.Servers) > 0 {
		md += "## Servers\n\n"
		for _, server := range spec.Servers {
			md += fmt.Sprintf("- **%s**: %s\n", server.URL, server.Description)
		}
		md += "\n"
	}

	// Add API endpoints
	md += "## API Endpoints\n\n"

	for path, methods := range spec.Paths {
		md += fmt.Sprintf("### %s\n\n", path)

		for method, endpoint := range methods {
			md += fmt.Sprintf("#### %s\n\n", strings.ToUpper(method))

			if endpoint.Summary != "" {
				md += fmt.Sprintf("**Summary:** %s\n\n", endpoint.Summary)
			}

			if endpoint.Description != "" {
				md += fmt.Sprintf("**Description:** %s\n\n", endpoint.Description)
			}

			// Add parameters
			if len(endpoint.Parameters) > 0 {
				md += "**Parameters:**\n\n"
				md += "| Name | Type | In | Required | Description |\n"
				md += "|------|------|----|---------|--------------|\n"

				for _, param := range endpoint.Parameters {
					required := "No"
					if param.Required {
						required = "Yes"
					}
					md += fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
						param.Name, param.Schema.Type, param.In, required, param.Description)
				}
				md += "\n"
			}

			// Add responses
			md += "**Responses:**\n\n"
			for status, response := range endpoint.Responses {
				md += fmt.Sprintf("- **%s**: %s\n", status, response.Description)
			}
			md += "\n"
		}
	}

	// Add schema definitions
	if spec.Components != nil && len(spec.Components.Schemas) > 0 {
		md += "## Data Models\n\n"

		for name, schema := range spec.Components.Schemas {
			md += fmt.Sprintf("### %s\n\n", name)

			if schema.Description != "" {
				md += schema.Description + "\n\n"
			}

			if len(schema.Properties) > 0 {
				md += "**Properties:**\n\n"
				md += "| Name | Type | Required | Description |\n"
				md += "|------|------|----------|--------------|\n"

				for propName, prop := range schema.Properties {
					required := "No"
					for _, req := range schema.Required {
						if req == propName {
							required = "Yes"

							break
						}
					}
					md += fmt.Sprintf("| %s | %s | %s | %s |\n",
						propName, prop.Type, required, prop.Description)
				}
				md += "\n"
			}
		}
	}

	return md
}

// generateInteractiveDocumentation sets up interactive documentation features.
func (g *APIDocumentationGenerator) generateInteractiveDocumentation() error {
	// This would set up additional interactive features like:
	// - API testing interface
	// - Live code examples
	// - Interactive schema browser
	// For now, we'll create placeholder files

	return g.createInteractiveAssets()
}

// createInteractiveAssets creates additional assets for interactive documentation.
func (g *APIDocumentationGenerator) createInteractiveAssets() error {
	// Create CSS customizations
	cssContent := `/* Custom styles for Templar API documentation */
.swagger-ui .topbar {
    background-color: #007acc;
}

.swagger-ui .info .title {
    color: #007acc;
}

.swagger-ui .btn.authorize {
    background-color: #007acc;
    border-color: #007acc;
}

.swagger-ui .btn.authorize:hover {
    background-color: #005a99;
    border-color: #005a99;
}
`

	cssPath := filepath.Join(g.outputDir, "custom.css")
	if err := os.WriteFile(cssPath, []byte(cssContent), 0600); err != nil {
		return fmt.Errorf("failed to write custom CSS: %w", err)
	}

	// Create JavaScript enhancements
	jsContent := `/* Custom JavaScript for Templar API documentation */
document.addEventListener('DOMContentLoaded', function() {
    // Add custom functionality
    console.log('Templar API documentation loaded');
    
    // Add keyboard shortcuts
    document.addEventListener('keydown', function(e) {
        if (e.ctrlKey && e.key === '/') {
            // Focus search box if available
            const searchBox = document.querySelector('.swagger-ui input[type="text"]');
            if (searchBox) {
                searchBox.focus();
                e.preventDefault();
            }
        }
    });
});
`

	jsPath := filepath.Join(g.outputDir, "custom.js")
	if err := os.WriteFile(jsPath, []byte(jsContent), 0644); err != nil {
		return fmt.Errorf("failed to write custom JavaScript: %w", err)
	}

	log.Printf("Generated interactive documentation assets")

	return nil
}

// DefaultGenerationConfig returns a default configuration for API generation.
func DefaultGenerationConfig() *GenerationConfig {
	return &GenerationConfig{
		OutputDir:       "./docs/api",
		Version:         "1.0.0",
		IncludeInternal: false,
		ServeEnabled:    true,
		ServePort:       8081,
		AutoReload:      true,
		Formats:         []string{"json", "yaml", "html", "markdown"},
	}
}

// validateGenerationConfig validates the generation configuration.
func validateGenerationConfig(config *GenerationConfig) error {
	if config.OutputDir == "" {
		return errors.New("output directory is required")
	}

	if config.Version == "" {
		return errors.New("API version is required")
	}

	if len(config.Formats) == 0 {
		return errors.New("at least one output format is required")
	}

	// Validate supported formats
	supportedFormats := map[string]bool{
		"json":     true,
		"yaml":     true,
		"html":     true,
		"markdown": true,
	}

	for _, format := range config.Formats {
		if !supportedFormats[format] {
			return fmt.Errorf("unsupported output format: %s", format)
		}
	}

	if config.ServePort <= 0 || config.ServePort > 65535 {
		return fmt.Errorf("invalid serve port: %d", config.ServePort)
	}

	return nil
}
