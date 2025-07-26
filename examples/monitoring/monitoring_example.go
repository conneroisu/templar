package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/conneroisu/templar/internal/monitoring"
)

// Example demonstrating comprehensive monitoring integration.
func main() {
	// Setup monitoring
	config := monitoring.MonitoringConfig{
		EnableHTTPMiddleware: true,
		EnableHealthChecks:   true,
		EnableMetrics:        true,
		LogLevel:             "info",
	}

	monitor, err := monitoring.SetupMonitoring(config)
	if err != nil {
		log.Fatalf("Failed to setup monitoring: %v", err)
	}

	// Start monitoring
	if err := monitor.Start(); err != nil {
		log.Fatalf("Failed to start monitoring: %v", err)
	}
	defer monitor.Stop()

	// Register custom health checks
	registerCustomHealthChecks(monitor)

	// Setup HTTP server with monitoring middleware
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleHome)
	mux.HandleFunc("/api/component", handleComponent)
	mux.HandleFunc("/api/build", handleBuild)

	// Apply monitoring middleware
	handler := monitoring.GetMiddleware()(mux)

	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	// Start HTTP server
	go func() {
		fmt.Println("Starting server on :8080")
		fmt.Println("Monitoring endpoints:")
		fmt.Println("  - Health: http://localhost:8081/health")
		fmt.Println("  - Metrics: http://localhost:8081/metrics")
		fmt.Println("  - Info: http://localhost:8081/info")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Simulate some application activity
	go simulateActivity()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	fmt.Println("Server exited")
}

// registerCustomHealthChecks demonstrates registering custom health checks.
func registerCustomHealthChecks(monitor *monitoring.Monitor) {
	// Component health check
	componentChecker := monitoring.ComponentHealthChecker("example_component", func() error {
		// Simulate component check
		return nil
	})
	monitor.RegisterHealthCheck(componentChecker)

	// Build pipeline health check
	buildChecker := monitoring.BuildPipelineHealthChecker(func() error {
		// Simulate build check
		return nil
	})
	monitor.RegisterHealthCheck(buildChecker)

	// File watcher health check
	fileWatcherChecker := monitoring.FileWatcherHealthChecker(func() bool {
		// Simulate file watcher status
		return true
	})
	monitor.RegisterHealthCheck(fileWatcherChecker)

	// WebSocket health check
	wsChecker := monitoring.WebSocketHealthChecker(func() int {
		// Simulate WebSocket connection count
		return 5
	})
	monitor.RegisterHealthCheck(wsChecker)
}

// handleHome demonstrates basic request handling.
func handleHome(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Track operation
	err := monitoring.TrackOperation(ctx, "http_handler", "home", func(ctx context.Context) error {
		// Simulate some work
		time.Sleep(10 * time.Millisecond)

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		html := `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Templar Monitoring Example</title>
		</head>
		<body>
			<h1>Templar Monitoring Example</h1>
			<p>This is a demonstration of the comprehensive monitoring system.</p>
			<h2>Monitoring Endpoints</h2>
			<ul>
				<li><a href="http://localhost:8081/health">Health Check</a></li>
				<li><a href="http://localhost:8081/metrics">Metrics</a></li>
				<li><a href="http://localhost:8081/info">Application Info</a></li>
			</ul>
			<h2>API Endpoints</h2>
			<ul>
				<li><a href="/api/component">Component API</a></li>
				<li><a href="/api/build">Build API</a></li>
			</ul>
		</body>
		</html>
		`

		w.Write([]byte(html))

		return nil
	})

	if err != nil {
		monitoring.LogError(ctx, "http_handler", "home", err, "Failed to handle home request")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleComponent demonstrates component operation tracking.
func handleComponent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	err := monitoring.TrackOperation(
		ctx,
		"component_api",
		"list_components",
		func(ctx context.Context) error {
			// Simulate component discovery
			components := []string{"Button", "Card", "Modal", "Form"}

			monitoring.LogInfo(
				ctx,
				"component_api",
				"list_components",
				"Listing components",
				"count",
				len(components),
			)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			response := fmt.Sprintf(`{
			"components": %q,
			"count": %d,
			"timestamp": "%s"
		}`, components, len(components), time.Now().Format(time.RFC3339))

			w.Write([]byte(response))

			return nil
		},
	)

	if err != nil {
		monitoring.LogError(
			ctx,
			"component_api",
			"list_components",
			err,
			"Failed to list components",
		)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleBuild demonstrates build operation tracking with batch processing.
func handleBuild(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	err := monitoring.TrackOperation(
		ctx,
		"build_api",
		"build_components",
		func(ctx context.Context) error {
			// Simulate batch build process
			components := []string{"Button", "Card", "Modal", "Form", "Layout"}

			// Create batch tracker
			monitor := monitoring.GetGlobalMonitor()
			if monitor == nil {
				return errors.New("monitor not available")
			}

			batchTracker := monitoring.NewBatchTracker(
				monitor,
				monitor.GetLogger(),
				"build_system",
				len(components),
			)

			successCount := 0
			for _, component := range components {
				err := batchTracker.TrackItem(ctx, component, func() error {
					// Simulate component build
					time.Sleep(20 * time.Millisecond)

					// Simulate occasional build failure
					if component == "Modal" {
						return fmt.Errorf("build failed for %s", component)
					}

					successCount++

					return nil
				})

				// Continue processing even if individual item fails
				if err != nil {
					monitoring.LogError(
						ctx,
						"build_system",
						"build_component",
						err,
						"Component build failed",
						"component",
						component,
					)
				}
			}

			batchTracker.Complete(ctx)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			response := fmt.Sprintf(`{
			"total": %d,
			"successful": %d,
			"failed": %d,
			"timestamp": "%s"
		}`, len(components), successCount, len(components)-successCount, time.Now().Format(time.RFC3339))

			w.Write([]byte(response))

			return nil
		},
	)

	if err != nil {
		monitoring.LogError(ctx, "build_api", "build_components", err, "Failed to build components")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// simulateActivity demonstrates background monitoring.
func simulateActivity() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		simulateBackgroundOperations()
	}
}

// simulateBackgroundOperations demonstrates various monitoring scenarios.
func simulateBackgroundOperations() {
	ctx := context.Background()

	// Simulate file scanning
	monitoring.TrackOperation(ctx, "scanner", "scan_files", func(ctx context.Context) error {
		monitoring.LogInfo(ctx, "scanner", "scan_files", "Scanning component files")
		time.Sleep(50 * time.Millisecond)

		return nil
	})

	// Simulate cache operations
	monitor := monitoring.GetGlobalMonitor()
	if monitor != nil && monitor.GetMetrics() != nil {
		// Simulate cache hits and misses
		monitor.GetMetrics().CacheOperation("get", true)  // hit
		monitor.GetMetrics().CacheOperation("get", false) // miss
		monitor.GetMetrics().CacheOperation("set", true)  // success

		// Simulate file watcher events
		monitor.GetMetrics().FileWatcherEvent("created")
		monitor.GetMetrics().FileWatcherEvent("modified")

		// Simulate WebSocket events
		monitor.GetMetrics().WebSocketConnection("opened")
		monitor.GetMetrics().WebSocketMessage("reload")
	}

	// Simulate occasional errors
	if time.Now().Unix()%10 == 0 { // Every 10th cycle
		monitoring.LogError(ctx, "background_worker", "cleanup",
			errors.New("temporary cleanup error"),
			"Cleanup operation failed",
			"retry_after", "30s")
	}
}
