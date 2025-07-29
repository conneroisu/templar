package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/conneroisu/templar/internal/apidocs"
	"github.com/conneroisu/templar/internal/config"
	"github.com/spf13/cobra"
)

var docsCmd = &cobra.Command{
	Use:     "docs",
	Aliases: []string{"doc", "documentation", "api-docs"},
	Short:   "Generate comprehensive API documentation from source code",
	Long: `Generate comprehensive API documentation from source code using automated analysis.

Analyzes HTTP handlers and Go types to produce OpenAPI 3.0 specifications,
interactive Swagger UI documentation, and multiple output formats. The system
uses static code analysis to ensure documentation stays synchronized with
the actual implementation without requiring runtime inspection.

Examples:
  templar docs                           # Generate docs with default settings
  templar docs --output ./docs/api       # Custom output directory
  templar docs --format json,yaml,html   # Multiple output formats
  templar docs --serve                   # Generate and serve interactive docs
  templar docs --serve-port 8081         # Custom documentation server port
  templar docs --version 2.0.0           # Specify API version
  templar docs --include-internal        # Include internal endpoints
  templar docs --no-auto-reload          # Disable auto-reload

Output Formats:
  json       OpenAPI 3.0 specification in JSON format
  yaml       OpenAPI 3.0 specification in YAML format  
  html       Interactive Swagger UI documentation
  markdown   Human-readable markdown documentation

Pro Tips:
  • Use --serve for interactive development and testing
  • Generate docs in CI/CD pipelines for automated updates
  • Multiple formats can be generated simultaneously
  • HTML output includes embedded Swagger UI for standalone use

See also: templar serve, templar build, templar list`,
	RunE: runDocs,
}

var (
	docsOutputDir    string
	docsFormats      []string
	docsServe        bool
	docsServePort    int
	docsIncludeInternal bool
	docsAutoReload   bool
	docsVersion      string
)

func init() {
	rootCmd.AddCommand(docsCmd)

	// Output configuration
	docsCmd.Flags().StringVarP(&docsOutputDir, "output", "o", "./docs/api", 
		"Output directory for generated documentation")
	docsCmd.Flags().StringSliceVarP(&docsFormats, "format", "f", 
		[]string{"json", "yaml", "html", "markdown"}, 
		"Output formats (json,yaml,html,markdown)")

	// Server configuration  
	docsCmd.Flags().BoolVar(&docsServe, "serve", false, 
		"Start documentation server after generation")
	docsCmd.Flags().IntVar(&docsServePort, "serve-port", 8081, 
		"Port for documentation server")
	docsCmd.Flags().BoolVar(&docsAutoReload, "auto-reload", true, 
		"Enable automatic regeneration on code changes")

	// API configuration
	docsCmd.Flags().StringVarP(&docsVersion, "version", "v", "1.0.0", 
		"API version to document")
	docsCmd.Flags().BoolVar(&docsIncludeInternal, "include-internal", false, 
		"Include internal/private endpoints in documentation")

	// Add format validation
	AddFlagValidation(docsCmd, "format", func(formats string) error {
		validFormats := []string{"json", "yaml", "html", "markdown"}
		formatList := strings.Split(formats, ",")
		for _, format := range formatList {
			format = strings.TrimSpace(format)  
			valid := false
			for _, validFormat := range validFormats {
				if format == validFormat {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("unsupported format '%s'. Valid formats: %s", 
					format, strings.Join(validFormats, ", "))
			}
		}
		return nil
	})
}

func runDocs(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	ctx := context.Background()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	_ = cfg // Suppress unused variable warning - config might be used for documentation customization in the future

	fmt.Println("📚 Starting API documentation generation...")

	// Determine server package path
	serverPackagePath := filepath.Join(".", "internal", "server")
	if _, err := os.Stat(serverPackagePath); os.IsNotExist(err) {
		return fmt.Errorf("server package not found at %s", serverPackagePath)
	}

	// Convert to absolute path
	absServerPath, err := filepath.Abs(serverPackagePath)
	if err != nil {
		return fmt.Errorf("failed to resolve server path: %w", err)
	}

	// Create documentation generation configuration
	genConfig := &apidocs.GenerationConfig{
		OutputDir:       docsOutputDir,
		Version:         docsVersion,
		IncludeInternal: docsIncludeInternal,
		ServeEnabled:    docsServe,
		ServePort:       docsServePort,
		AutoReload:      docsAutoReload,
		Formats:         docsFormats,
	}

	// Create API documentation generator
	generator, err := apidocs.NewAPIDocumentationGenerator(genConfig)
	if err != nil {
		return fmt.Errorf("failed to create documentation generator: %w", err)
	}

	// Generate documentation
	result, err := generator.Generate(absServerPath)
	if err != nil {
		return fmt.Errorf("documentation generation failed: %w", err)
	}

	// Display results
	fmt.Printf("✅ Documentation generation completed in %v\n", time.Since(startTime))
	fmt.Printf("📊 Found %d API endpoints\n", result.EndpointsFound)
	fmt.Printf("📋 Generated %d schemas\n", result.SchemasGenerated)
	fmt.Printf("📁 Output directory: %s\n", docsOutputDir)

	// List generated files
	fmt.Println("\n📄 Generated files:")
	for _, format := range docsFormats {
		var filename string
		switch format {
		case "json":
			filename = "openapi.json"
		case "yaml":
			filename = "openapi.yaml"
		case "html":
			filename = "index.html"
		case "markdown":
			filename = "README.md"
		}
		fmt.Printf("   • %s\n", filepath.Join(docsOutputDir, filename))
	}

	// Display warnings if any
	if len(result.Warnings) > 0 {
		fmt.Println("\n⚠️  Warnings:")
		for _, warning := range result.Warnings {
			fmt.Printf("   • %s\n", warning)
		}
	}

	// Start documentation server if requested
	if docsServe {
		fmt.Printf("\n🌐 Starting documentation server on http://localhost:%d\n", docsServePort)
		fmt.Println("📖 Interactive API documentation available at:")
		fmt.Printf("   • Swagger UI: http://localhost:%d\n", docsServePort)
		fmt.Printf("   • OpenAPI JSON: http://localhost:%d/openapi.json\n", docsServePort)
		fmt.Printf("   • OpenAPI YAML: http://localhost:%d/openapi.yaml\n", docsServePort)
		
		if docsAutoReload {
			fmt.Println("👀 Watching for code changes...")
		}
		
		fmt.Println("\nPress Ctrl+C to stop the server")
		return startDocumentationServer(ctx, genConfig, result)
	}

	fmt.Printf("\n⏱️  Total generation time: %v\n", time.Since(startTime))
	fmt.Println("🎉 API documentation generation complete!")

	return nil
}

// startDocumentationServer starts the documentation server (placeholder implementation)
func startDocumentationServer(ctx context.Context, config *apidocs.GenerationConfig, result *apidocs.GenerationResult) error {
	// This is a placeholder implementation
	// In a complete implementation, this would:
	// 1. Start an HTTP server serving the generated documentation
	// 2. Set up file watchers if auto-reload is enabled
	// 3. Handle graceful shutdown on context cancellation
	// 4. Serve Swagger UI, JSON/YAML specs, and static assets
	
	fmt.Println("📡 Documentation server would start here...")
	fmt.Println("   (Server implementation coming in next phase)")
	
	// For demo purposes, just indicate where files were generated
	fmt.Printf("💡 You can manually serve the documentation with:\n")
	fmt.Printf("   cd %s && python -m http.server %d\n", config.OutputDir, config.ServePort)
	
	return nil
}