package cmd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/conneroisu/templar/internal/monitoring"
	"github.com/spf13/cobra"
)

// serveWithMonitoringCmd demonstrates a complete integration of the monitoring system.
var serveWithMonitoringCmd = &cobra.Command{
	Use:   "serve-monitored",
	Short: "Start development server with comprehensive monitoring",
	Long: `Start the Templar development server with full monitoring capabilities including:
- HTTP request tracking and metrics
- Component operation monitoring  
- Health checks for all services
- WebSocket connection tracking
- File watcher monitoring
- Performance metrics and alerting`,
	RunE: runServeWithMonitoring,
}

var (
	monitoringConfigPath string
	enableProfiling      bool
	metricsPort          int
)

func init() {
	serveWithMonitoringCmd.Flags().
		StringVar(&monitoringConfigPath, "monitoring-config", "", "Path to monitoring configuration file")
	serveWithMonitoringCmd.Flags().
		BoolVar(&enableProfiling, "enable-profiling", false, "Enable Go profiling endpoints")
	serveWithMonitoringCmd.Flags().
		IntVar(&metricsPort, "metrics-port", 8081, "Port for monitoring endpoints")
}

func runServeWithMonitoring(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Setup monitoring system
	monitor, err := setupMonitoringSystem(ctx)
	if err != nil {
		return fmt.Errorf("failed to setup monitoring: %w", err)
	}
	defer monitor.GracefulShutdown(ctx)

	// Create the application server with monitoring
	server, err := createMonitoredServer(monitor)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Setup graceful shutdown
	return runServerWithGracefulShutdown(ctx, server, monitor)
}

func setupMonitoringSystem(ctx context.Context) (*monitoring.TemplarMonitor, error) {
	// Create monitoring configuration
	config := monitoring.DefaultTemplarConfig()
	if metricsPort > 0 {
		config.HTTPPort = metricsPort
	}

	// Load configuration from file if provided
	if monitoringConfigPath != "" {
		// In a real implementation, you would load from YAML file
		fmt.Printf("Loading monitoring config from: %s\n", monitoringConfigPath)
	}

	// Setup Templar-specific monitoring
	monitor, err := monitoring.NewTemplarMonitor(monitoringConfigPath)
	if err != nil {
		return nil, err
	}

	// Start monitoring system
	if err := monitor.Start(); err != nil {
		return nil, fmt.Errorf("failed to start monitoring: %w", err)
	}

	// Set as global monitor for easy access
	monitoring.SetGlobalMonitor(monitor.Monitor)

	// Register additional health checks specific to serve command
	registerServeHealthChecks(monitor)

	fmt.Printf("Monitoring system started:\n")
	fmt.Printf("  - Health: http://localhost:%d/health\n", config.HTTPPort)
	fmt.Printf("  - Metrics: http://localhost:%d/metrics\n", config.HTTPPort)
	fmt.Printf("  - Info: http://localhost:%d/info\n", config.HTTPPort)

	return monitor, nil
}

func registerServeHealthChecks(monitor *monitoring.TemplarMonitor) {
	// Port availability check
	portCheck := monitoring.NewHealthCheckFunc(
		"port_availability",
		true,
		func(ctx context.Context) monitoring.HealthCheck {
			start := time.Now()
			port := 8080 // Default Templar port

			// Try to bind to the port temporarily
			listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
			if err != nil {
				return monitoring.HealthCheck{
					Name:        "port_availability",
					Status:      monitoring.HealthStatusUnhealthy,
					Message:     fmt.Sprintf("Port %d is not available: %v", port, err),
					LastChecked: time.Now(),
					Duration:    time.Since(start),
					Critical:    true,
					Metadata: map[string]interface{}{
						"port":  port,
						"error": err.Error(),
					},
				}
			}
			listener.Close()

			return monitoring.HealthCheck{
				Name:        "port_availability",
				Status:      monitoring.HealthStatusHealthy,
				Message:     fmt.Sprintf("Port %d is available", port),
				LastChecked: time.Now(),
				Duration:    time.Since(start),
				Critical:    true,
				Metadata: map[string]interface{}{
					"port": port,
				},
			}
		},
	)
	monitor.RegisterHealthCheck(portCheck)

	// Template directory check
	templateCheck := monitoring.NewHealthCheckFunc(
		"template_directory",
		false,
		func(ctx context.Context) monitoring.HealthCheck {
			start := time.Now()
			templateDirs := []string{"./components", "./views", "./layouts"}

			for _, dir := range templateDirs {
				if _, err := os.Stat(dir); err != nil {
					return monitoring.HealthCheck{
						Name:        "template_directory",
						Status:      monitoring.HealthStatusDegraded,
						Message:     "Template directory not found: " + dir,
						LastChecked: time.Now(),
						Duration:    time.Since(start),
						Critical:    false,
						Metadata: map[string]interface{}{
							"missing_directory": dir,
						},
					}
				}
			}

			return monitoring.HealthCheck{
				Name:        "template_directory",
				Status:      monitoring.HealthStatusHealthy,
				Message:     "All template directories are accessible",
				LastChecked: time.Now(),
				Duration:    time.Since(start),
				Critical:    false,
				Metadata: map[string]interface{}{
					"directories": templateDirs,
				},
			}
		},
	)
	monitor.RegisterHealthCheck(templateCheck)
}

func createMonitoredServer(monitor *monitoring.TemplarMonitor) (*http.Server, error) {
	// Create HTTP multiplexer
	mux := http.NewServeMux()

	// Add application routes with monitoring
	addMonitoredRoutes(mux, monitor)

	// Apply monitoring middleware
	handler := monitor.CreateTemplarMiddleware()(mux)

	// Create server
	server := &http.Server{
		Addr:         ":8080",
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server, nil
}

func addMonitoredRoutes(mux *http.ServeMux, monitor *monitoring.TemplarMonitor) {
	// Home page
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		err := monitor.TrackServerOperation(ctx, "serve_homepage", func(ctx context.Context) error {
			// Simulate component discovery
			components := []string{"Button", "Card", "Modal", "Form"}

			monitoring.LogInfo(ctx, "server", "homepage", "Serving homepage",
				"components_available", len(components),
				"user_agent", r.Header.Get("User-Agent"))

			w.Header().Set("Content-Type", "text/html")
			html := generateHomePage(components)
			w.Write([]byte(html))

			return nil
		})

		if err != nil {
			monitoring.LogComponentError(ctx, "server", "homepage", err, map[string]interface{}{
				"path":        r.URL.Path,
				"method":      r.Method,
				"remote_addr": r.RemoteAddr,
			})
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})

	// API: List components
	mux.HandleFunc("/api/components", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		err := monitor.TrackScanOperation(ctx, "list_components", func(ctx context.Context) error {
			// Simulate component scanning
			time.Sleep(50 * time.Millisecond) // Simulate work

			components := scanComponents(ctx, monitor)

			w.Header().Set("Content-Type", "application/json")
			response := fmt.Sprintf(`{
				"components": %q,
				"count": %d,
				"timestamp": "%s"
			}`, components, len(components), time.Now().Format(time.RFC3339))

			w.Write([]byte(response))

			return nil
		})

		if err != nil {
			monitoring.LogComponentError(
				ctx,
				"scanner",
				"list_components",
				err,
				map[string]interface{}{
					"endpoint": "/api/components",
				},
			)
			http.Error(w, "Failed to scan components", http.StatusInternalServerError)
		}
	})

	// API: Build components
	mux.HandleFunc("/api/build", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

			return
		}

		ctx := r.Context()

		err := monitor.TrackBuildOperation(ctx, "build_all", func(ctx context.Context) error {
			return buildAllComponents(ctx, monitor)
		})

		if err != nil {
			monitoring.LogComponentError(ctx, "build", "build_all", err, map[string]interface{}{
				"endpoint": "/api/build",
			})
			http.Error(w, "Build failed", http.StatusInternalServerError)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "success", "message": "All components built successfully"}`))
	})

	// WebSocket endpoint for live reload
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		err := monitor.TrackServerOperation(
			ctx,
			"websocket_connection",
			func(ctx context.Context) error {
				return handleWebSocketConnection(ctx, w, r, monitor)
			},
		)

		if err != nil {
			monitoring.LogComponentError(
				ctx,
				"websocket",
				"connection",
				err,
				map[string]interface{}{
					"remote_addr": r.RemoteAddr,
				},
			)
		}
	})

	// File watcher status
	mux.HandleFunc("/api/watcher", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		err := monitor.TrackWatcherOperation(ctx, "get_status", func(ctx context.Context) error {
			status := getWatcherStatus(ctx, monitor)

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(status))

			return nil
		})

		if err != nil {
			http.Error(w, "Failed to get watcher status", http.StatusInternalServerError)
		}
	})
}

func scanComponents(ctx context.Context, monitor *monitoring.TemplarMonitor) []string {
	components := []string{"Button", "Card", "Modal", "Form", "Layout"}

	for _, component := range components {
		// Record each discovered component
		monitor.RecordComponentScanned("template", component)

		monitoring.LogInfo(ctx, "scanner", "component_discovered", "Found component",
			"component", component,
			"type", "template")
	}

	return components
}

func buildAllComponents(ctx context.Context, monitor *monitoring.TemplarMonitor) error {
	components := []string{"Button", "Card", "Modal", "Form", "Layout"}

	// Create batch tracker for build operation
	batchTracker := monitoring.NewBatchTracker(
		monitor.Monitor,
		monitor.GetLogger(),
		"build_system",
		len(components),
	)

	successCount := 0
	for _, component := range components {
		err := batchTracker.TrackItem(ctx, component, func() error {
			start := time.Now()

			// Simulate component build
			time.Sleep(time.Duration(50+component[0]) * time.Millisecond) // Variable duration

			// Simulate occasional failure
			if component == "Modal" {
				return errors.New("modal component failed to compile")
			}

			duration := time.Since(start)
			monitor.RecordComponentBuilt(component, true, duration)
			successCount++

			monitoring.LogInfo(ctx, "build", "component_built", "Component built successfully",
				"component", component,
				"duration", duration)

			return nil
		})

		if err != nil {
			duration := 100 * time.Millisecond // Estimated failure time
			monitor.RecordComponentBuilt(component, false, duration)

			monitoring.LogComponentError(
				ctx,
				"build",
				"component_build",
				err,
				map[string]interface{}{
					"component": component,
					"duration":  duration,
				},
			)
			// Continue with other components
		}
	}

	batchTracker.Complete(ctx)

	monitoring.LogInfo(ctx, "build", "build_complete", "Build operation completed",
		"total_components", len(components),
		"successful", successCount,
		"failed", len(components)-successCount)

	return nil
}

func handleWebSocketConnection(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	monitor *monitoring.TemplarMonitor,
) error {
	// Simulate WebSocket upgrade (in real implementation, use gorilla/websocket)
	monitor.RecordWebSocketEvent("connection_attempt", 1)

	// Simulate connection success
	monitor.RecordWebSocketEvent("client_connected", 1)

	monitoring.LogInfo(ctx, "websocket", "connection", "WebSocket connection established",
		"remote_addr", r.RemoteAddr,
		"user_agent", r.Header.Get("User-Agent"))

	// Simulate connection handling
	time.Sleep(100 * time.Millisecond)

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("WebSocket connection simulated"))

	return nil
}

func getWatcherStatus(ctx context.Context, monitor *monitoring.TemplarMonitor) string {
	// Simulate file watcher events
	events := []string{"created", "modified", "deleted"}
	for _, event := range events {
		monitor.RecordFileWatchEvent(event, "./components/example.templ")
	}

	monitoring.LogInfo(ctx, "watcher", "status_check", "File watcher status requested")

	return `{
		"active": true,
		"watched_paths": ["./components", "./views", "./layouts"],
		"events_today": 42,
		"last_event": "2024-01-15T10:30:00Z"
	}`
}

func generateHomePage(components []string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<title>Templar Development Server</title>
	<style>
		body { font-family: Arial, sans-serif; margin: 40px; }
		.components { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; }
		.component { border: 1px solid #ddd; padding: 20px; border-radius: 8px; }
		.monitoring { background: #f0f8ff; padding: 20px; border-radius: 8px; margin: 20px 0; }
		a { color: #0066cc; text-decoration: none; }
		a:hover { text-decoration: underline; }
	</style>
</head>
<body>
	<h1>Templar Development Server</h1>
	
	<div class="monitoring">
		<h2>🔍 Monitoring & Health</h2>
		<p>This server is running with comprehensive monitoring enabled:</p>
		<ul>
			<li><a href="http://localhost:8081/health">Health Status</a> - Component health checks</li>
			<li><a href="http://localhost:8081/metrics">Metrics</a> - Performance and usage metrics</li>
			<li><a href="http://localhost:8081/info">System Info</a> - Application information</li>
		</ul>
	</div>
	
	<h2>Available Components (%d)</h2>
	<div class="components">
		%s
	</div>
	
	<h2>API Endpoints</h2>
	<ul>
		<li><a href="/api/components">GET /api/components</a> - List all components</li>
		<li>POST /api/build - Build all components</li>
		<li><a href="/api/watcher">GET /api/watcher</a> - File watcher status</li>
		<li>GET /ws - WebSocket for live reload</li>
	</ul>
	
	<script>
		// Simulate some user interaction tracking
		document.addEventListener('click', function(e) {
			if (e.target.tagName === 'A') {
				console.log('Link clicked:', e.target.href);
				// In a real app, this could send analytics events
			}
		});
	</script>
</body>
</html>`, len(components), generateComponentCards(components))
}

func generateComponentCards(components []string) string {
	cards := ""
	for _, component := range components {
		cards += fmt.Sprintf(`
		<div class="component">
			<h3>%s</h3>
			<p>A reusable %s component for Templar applications.</p>
			<a href="/preview?component=%s">Preview</a>
		</div>`, component, component, component)
	}

	return cards
}

func runServerWithGracefulShutdown(
	ctx context.Context,
	server *http.Server,
	monitor *monitoring.TemplarMonitor,
) error {
	// Start server in background
	serverErr := make(chan error, 1)
	go func() {
		fmt.Printf("Starting Templar development server on %s\n", server.Addr)

		err := monitor.TrackServerOperation(ctx, "start_server", func(ctx context.Context) error {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				return err
			}

			return nil
		})

		serverErr <- err
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return err
	case <-quit:
		fmt.Println("\nReceived shutdown signal, stopping server...")

		// Create shutdown context with timeout
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Graceful server shutdown
		err := monitor.TrackServerOperation(
			shutdownCtx,
			"shutdown_server",
			func(ctx context.Context) error {
				return server.Shutdown(ctx)
			},
		)

		if err != nil {
			monitoring.LogComponentError(
				shutdownCtx,
				"server",
				"shutdown",
				err,
				map[string]interface{}{
					"timeout": "30s",
				},
			)

			return fmt.Errorf("server shutdown failed: %w", err)
		}

		fmt.Println("Server stopped successfully")

		return nil
	}
}
