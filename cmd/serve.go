package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/errors"
	"github.com/conneroisu/templar/internal/monitoring"
	"github.com/conneroisu/templar/internal/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serveCmd = &cobra.Command{
	Use:   "serve [file.templ]",
	Short: "Start the development server with hot reload",
	Long: `Start the development server with hot reload capability.
Automatically opens browser and watches for file changes.

Examples:
  templar serve                    # Serve all components
  templar serve example.templ      # Serve specific templ file
  templar serve components/*.templ # Serve multiple files`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().IntP("port", "p", 8080, "Port to serve on")
	serveCmd.Flags().String("host", "localhost", "Host to bind to")
	serveCmd.Flags().Bool("no-open", false, "Don't open browser automatically")
	serveCmd.Flags().StringP("watch", "w", "**/*.templ", "Watch pattern")

	viper.BindPFlag("server.port", serveCmd.Flags().Lookup("port"))
	viper.BindPFlag("server.host", serveCmd.Flags().Lookup("host"))
	viper.BindPFlag("server.no-open", serveCmd.Flags().Lookup("no-open"))
	viper.BindPFlag("build.watch", serveCmd.Flags().Lookup("watch"))
}

func runServe(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		// Enhanced error for configuration issues
		ctx := &errors.SuggestionContext{
			ConfigPath: ".templar.yml",
		}
		suggestions := errors.ConfigurationError(err.Error(), ".templar.yml", ctx)
		enhancedErr := errors.NewEnhancedError(
			"Failed to load configuration",
			err,
			suggestions,
		)
		return enhancedErr
	}

	// Set target files if specified
	cfg.TargetFiles = args

	// Initialize monitoring system
	monitor, err := monitoring.SetupTemplarMonitoring("")
	if err != nil {
		log.Printf("Warning: Failed to initialize monitoring: %v", err)
		// Continue without monitoring - non-fatal
	} else {
		log.Printf("Monitoring system initialized")
		defer func() {
			if shutdownErr := monitor.GracefulShutdown(context.Background()); shutdownErr != nil {
				log.Printf("Error during monitoring shutdown: %v", shutdownErr)
			}
		}()
	}

	srv, err := server.New(cfg)
	if err != nil {
		// Check for server creation errors
		if strings.Contains(err.Error(), "address already in use") || strings.Contains(err.Error(), "bind") {
			ctx := &errors.SuggestionContext{}
			suggestions := errors.ServerStartError(err, cfg.Server.Port, ctx)
			enhancedErr := errors.NewEnhancedError(
				fmt.Sprintf("Failed to start server on port %d", cfg.Server.Port),
				err,
				suggestions,
			)
			return enhancedErr
		}
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Create context that cancels on interrupt
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down server...")

		// Shutdown server gracefully
		if shutdownErr := srv.Shutdown(ctx); shutdownErr != nil {
			log.Printf("Error during server shutdown: %v", shutdownErr)
		}

		cancel()
	}()

	if len(args) > 0 {
		fmt.Printf("Starting Templar server for %v at http://%s:%d\n", args, cfg.Server.Host, cfg.Server.Port)
	} else {
		fmt.Printf("Starting Templar server at http://%s:%d\n", cfg.Server.Host, cfg.Server.Port)
	}

	// Add monitoring information if available
	if monitor != nil {
		fmt.Printf("Monitoring dashboard: http://localhost:8081\n")
	}

	if err := srv.Start(ctx); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
