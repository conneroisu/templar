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
	Short:   "Start the development server with hot reload and live preview",
	Long: `Start the development server with hot reload capability and live preview.

The server automatically watches for file changes and rebuilds components
as needed. WebSocket connections provide instant browser updates without
manual page refreshes.

Examples:
  templar serve                    # Serve all components on localhost:8080
  templar serve -p 3000            # Use custom port
  templar serve --host 0.0.0.0     # Bind to all interfaces (external access)
  templar serve --no-open          # Don't automatically open browser  
  templar serve --watch "**/*.go"  # Custom file watch pattern
  templar serve -v                 # Enable verbose logging
  templar serve example.templ      # Serve specific templ file
  templar serve components/*.templ # Serve multiple files

Security Note:
  Using --host 0.0.0.0 exposes the server to external connections.
  Only use this in secure environments or for intentional network access.`,
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

func runServe(cmd *cobra.Command, args []string) error {
	// Validate enhanced flags
	if err := serveFlags.ValidateEnhancedFlags(); err != nil {
		return fmt.Errorf("flag validation failed: %w", err)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		// Enhanced error for configuration issues
		ctx := &errors.SuggestionContext{
			ConfigPath: ".templar.yml",
		}
		suggestions := errors.ConfigurationErrorSuggestions(err.Error(), ".templar.yml", ctx)
		enhancedErr := errors.NewEnhancedError(
			"Failed to load configuration",
			err,
			suggestions,
		)

		return enhancedErr
	}

	// Create serve service
	serveService := services.NewServeService(cfg)

	// Get server info for display
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
