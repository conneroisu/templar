package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/di"
	"github.com/conneroisu/templar/internal/errors"
	"github.com/conneroisu/templar/internal/monitoring"
)

// ServeService handles development server business logic
type ServeService struct {
	config *config.Config
}

// NewServeService creates a new serve service
func NewServeService(cfg *config.Config) *ServeService {
	return &ServeService{
		config: cfg,
	}
}

// ServeOptions contains options for the serve process
type ServeOptions struct {
	TargetFiles []string
}

// ServeResult contains the result of a serve operation
type ServeResult struct {
	ServerURL  string
	MonitorURL string
	Success    bool
	Error      error
}

// Serve starts the development server with hot reload and monitoring
func (s *ServeService) Serve(ctx context.Context, opts ServeOptions) (*ServeResult, error) {
	result := &ServeResult{
		Success: true,
	}

	// Set target files if specified
	s.config.TargetFiles = opts.TargetFiles

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
		result.MonitorURL = "http://localhost:8081"
	}

	// Initialize dependency injection container
	container := di.NewServiceContainer(s.config)
	if err := container.Initialize(); err != nil {
		result.Success = false
		result.Error = errors.ServeServiceError(
			"INIT_CONTAINER",
			"service container initialization failed",
			err,
		)
		return result, result.Error
	}
	defer func() {
		if shutdownErr := container.Shutdown(context.Background()); shutdownErr != nil {
			log.Printf("Error during container shutdown: %v", shutdownErr)
		}
	}()

	srv, err := container.GetServer()
	if err != nil {
		result.Success = false
		// Check for server creation errors
		if strings.Contains(err.Error(), "address already in use") ||
			strings.Contains(err.Error(), "bind") {
			contextSuggestion := &errors.SuggestionContext{}
			suggestions := errors.ServerStartError(err, s.config.Server.Port, contextSuggestion)
			enhancedErr := errors.NewEnhancedError(
				fmt.Sprintf("Failed to start server on port %d", s.config.Server.Port),
				err,
				suggestions,
			)
			result.Error = enhancedErr
		} else {
			result.Error = errors.ServeServiceError("GET_SERVER", "failed to create server", err)
		}
		return result, result.Error
	}

	// Set server URL for result
	result.ServerURL = fmt.Sprintf("http://%s:%d", s.config.Server.Host, s.config.Server.Port)

	// Create context that cancels on interrupt
	serverCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down server...")

		// Shutdown server gracefully
		if shutdownErr := srv.Shutdown(serverCtx); shutdownErr != nil {
			log.Printf("Error during server shutdown: %v", shutdownErr)
		}

		cancel()
	}()

	// Start the server
	if err := srv.Start(serverCtx); err != nil {
		result.Success = false
		result.Error = errors.ServeServiceError("START_SERVER", "server startup failed", err)
		return result, result.Error
	}

	return result, nil
}

// GetServerInfo returns information about the server configuration
func (s *ServeService) GetServerInfo(targetFiles []string) *ServerInfo {
	info := &ServerInfo{
		Host:        s.config.Server.Host,
		Port:        s.config.Server.Port,
		ServerURL:   fmt.Sprintf("http://%s:%d", s.config.Server.Host, s.config.Server.Port),
		TargetFiles: targetFiles,
	}
	return info
}

// ServerInfo contains information about the server configuration
type ServerInfo struct {
	Host        string
	Port        int
	ServerURL   string
	MonitorURL  string
	TargetFiles []string
}
