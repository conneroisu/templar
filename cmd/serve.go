package cmd

import (
	"context"
	"fmt"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/errors"
	"github.com/conneroisu/templar/internal/services"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serveCmd = &cobra.Command{
	Use:     "serve [file.templ]",
	Aliases: []string{"s"},
	Short:   "Start development server with hot reload and live preview",
	Long: `Start the development server with hot reload capability and live preview.

The server automatically watches for file changes, rebuilds components as needed,
and provides instant browser updates through WebSocket connections without manual
page refreshes. Ideal for rapid component development and testing.

Examples:
  templar serve                    # Serve all components on localhost:8080
  templar serve --port 3000        # Use custom port number
  templar serve --host 0.0.0.0     # Allow external network connections
  templar serve --no-open          # Don't automatically open browser
  templar serve --verbose          # Enable detailed logging output
  templar serve example.templ      # Serve specific component file
  templar serve "components/*.templ" # Serve multiple component files
  templar serve --config prod.yml  # Use custom configuration file

Server Features:
  • Hot reload with file watching
  • WebSocket-based live updates
  • Component isolation and preview
  • Mock data integration
  • Build error overlay
  • Accessibility testing integration

Pro Tips:
  • Server automatically opens browser unless --no-open is used
  • Use Ctrl+C to stop the server gracefully
  • WebSocket connections show real-time build status
  • Visit /components for component browser interface

Security Note:
  Using --host 0.0.0.0 exposes the server to external connections.
  Only use this in secure environments or for intentional network access.

See also: templar build, templar preview, templar watch`,
	RunE: runServe,
}

var serveFlags *EnhancedStandardFlags

func init() {
	rootCmd.AddCommand(serveCmd)

	// Use enhanced standard flags for consistency
	serveFlags = AddEnhancedFlags(serveCmd, "server", "build", "output")

	// Bind flags to viper for configuration integration
	_ = viper.BindPFlag("server.port", serveCmd.Flags().Lookup("port"))
	_ = viper.BindPFlag("server.host", serveCmd.Flags().Lookup("host"))
	_ = viper.BindPFlag("server.no-open", serveCmd.Flags().Lookup("no-open"))
	_ = viper.BindPFlag("build.watch", serveCmd.Flags().Lookup("watch"))
}

// runServe implements the core development server orchestration with enhanced error handling
// and service integration. This function coordinates multiple subsystems to provide
// a comprehensive development experience with hot reload, live preview, and build monitoring.
//
// Architecture overview:
// - Flag validation ensures consistent CLI behavior across all commands
// - Configuration loading with error suggestion provides user-friendly diagnostics
// - Service orchestration integrates component scanning, building, and serving
// - Enhanced error handling offers actionable suggestions for common issues
//
// Integration points:
// - Component scanner: Discovers and monitors .templ files for changes
// - Build pipeline: Compiles components with caching and error collection
// - WebSocket server: Provides real-time updates to connected browsers
// - File watcher: Triggers rebuilds on file system changes
//
// Performance considerations:
// - Service initialization is lazy to reduce startup time
// - Configuration validation occurs early to fail fast on misconfigurations
// - Resource cleanup is handled through context cancellation patterns
//
// Error handling strategy:
// - Structured errors with categorization for better debugging
// - Context-aware suggestions based on common configuration issues
// - Graceful degradation when non-critical subsystems fail.
func runServe(cmd *cobra.Command, args []string) error {
	// Validate CLI flags using enhanced validation framework
	// This provides comprehensive validation with user-friendly error messages
	// and suggestions for common flag usage patterns
	if err := serveFlags.ValidateEnhancedFlags(); err != nil {
		return fmt.Errorf("flag validation failed: %w", err)
	}

	// Load and validate configuration with enhanced error reporting
	// Configuration errors are common during initial setup, so we provide
	// detailed suggestions and context to help users resolve issues quickly
	cfg, err := config.Load()
	if err != nil {
		// Create enhanced error with actionable suggestions
		// This improves the developer experience by providing specific guidance
		// rather than generic error messages
		ctx := &errors.SuggestionContext{
			ConfigPath: TemplarConfigFile,
		}
		suggestions := errors.ConfigurationErrorSuggestions(err.Error(), TemplarConfigFile, ctx)
		enhancedErr := errors.NewEnhancedError(
			"Failed to load configuration",
			err,
			suggestions,
		)

		return enhancedErr
	}

	// Initialize serve service with dependency injection pattern
	// The service encapsulates all server-related functionality and manages
	// the lifecycle of component scanning, building, and HTTP serving
	serveService := services.NewServeService(cfg)

	// Extract server information for user feedback
	// Provides visibility into server configuration before starting
	serverInfo := serveService.GetServerInfo(args)

	// Display startup information
	if len(args) > 0 {
		fmt.Printf("Starting Templar server for %v at %s\n", args, serverInfo.ServerURL)
	} else {
		fmt.Printf("Starting Templar server at %s\n", serverInfo.ServerURL)
	}

	// Configure serve options
	opts := services.ServeOptions{
		TargetFiles: args,
	}

	// Start the server
	ctx := context.Background()
	result, err := serveService.Serve(ctx, opts)
	if err != nil {
		return err
	}

	// Display additional information
	if result.MonitorURL != "" {
		fmt.Printf("Monitoring dashboard: %s\n", result.MonitorURL)
	}

	if !result.Success {
		return result.Error
	}

	return nil
}
