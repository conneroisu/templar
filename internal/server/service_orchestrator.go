package server

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"sync"

	"github.com/conneroisu/templar/internal/build"
	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/errors"
	"github.com/conneroisu/templar/internal/interfaces"
	"github.com/conneroisu/templar/internal/monitoring"
	"github.com/conneroisu/templar/internal/renderer"
	"github.com/conneroisu/templar/internal/watcher"
)

// ServiceOrchestrator coordinates business logic and service interactions
// Following Single Responsibility Principle: orchestrates service coordination only
//
// Design Principles:
// - Single Responsibility: Only orchestrates coordination between services
// - Dependency Injection: All services injected through ServiceDependencies
// - Event-Driven Architecture: Responds to file changes and build events
// - Graceful Lifecycle: Proper startup/shutdown coordination across services
// - Thread Safety: All shared state protected by appropriate mutexes
//
// Architecture:
// - Mediator Pattern: Coordinates interactions between independent services
// - Observer Pattern: Listens to file changes and build events
// - Command Pattern: Processes events and dispatches actions
// - State Management: Maintains build error state with thread-safe access
//
// Invariants:
// - config must never be nil after construction
// - ctx and cancel are never nil after construction
// - lastBuildErrors access always protected by buildMutex
// - shutdown happens exactly once via shutdownOnce
type ServiceOrchestrator struct {
	// Configuration - immutable reference to application configuration
	config *config.Config

	// Core services (injected dependencies) - may be nil if not needed
	registry      interfaces.ComponentRegistry // Component discovery and management
	fileWatcher   interfaces.FileWatcher       // File system change detection
	scanner       interfaces.ComponentScanner  // Template component analysis
	buildPipeline interfaces.BuildPipeline     // Component build orchestration
	renderer      *renderer.ComponentRenderer  // Component rendering engine
	monitor       *monitoring.TemplarMonitor   // Monitoring and metrics collection

	// WebSocket manager for real-time client communication
	wsManager *WebSocketManager // WebSocket connection management

	// Build state management - thread-safe build error tracking
	lastBuildErrors []*errors.ParsedError // Latest build errors for clients
	buildMutex      sync.RWMutex          // Protects lastBuildErrors access

	// Lifecycle management - coordinates shutdown across all services
	ctx          context.Context    // Cancellation context for all operations
	cancel       context.CancelFunc // Function to cancel all operations
	shutdownOnce sync.Once          // Ensures shutdown happens exactly once
}

// ServiceDependencies contains all services needed by the orchestrator
type ServiceDependencies struct {
	Config        *config.Config
	Registry      interfaces.ComponentRegistry
	FileWatcher   interfaces.FileWatcher
	Scanner       interfaces.ComponentScanner
	BuildPipeline interfaces.BuildPipeline
	Renderer      *renderer.ComponentRenderer
	Monitor       *monitoring.TemplarMonitor
	WSManager     *WebSocketManager
}

// NewServiceOrchestrator creates a new service orchestrator with dependency injection
//
// This constructor initializes a service orchestrator that coordinates all business logic:
// - Validates required dependencies for safe operation
// - Sets up cancellation context for coordinated shutdown
// - Initializes thread-safe state management
// - Creates the foundation for event-driven service coordination
//
// The orchestrator follows the mediator pattern to coordinate between:
// - File watching and change detection
// - Component scanning and analysis
// - Build pipeline and error handling
// - WebSocket broadcasting and client updates
// - Monitoring and metrics collection
//
// Parameters:
// - deps: Struct containing all service dependencies (some may be nil)
//
// Returns:
// - Fully initialized ServiceOrchestrator ready for Start()
//
// Panics:
// - If required dependencies are nil or invalid
func NewServiceOrchestrator(deps ServiceDependencies) *ServiceOrchestrator {
	// Critical dependency validation - config is always required
	if deps.Config == nil {
		panic("ServiceOrchestrator: config cannot be nil")
	}

	// Validate essential configuration fields
	if len(deps.Config.Components.ScanPaths) == 0 {
		panic("ServiceOrchestrator: config.Components.ScanPaths cannot be empty")
	}

	// Create cancellable context for coordinated lifecycle management
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize orchestrator with validated dependencies
	orchestrator := &ServiceOrchestrator{
		config:          deps.Config,        // Required configuration
		registry:        deps.Registry,      // Component registry (optional but recommended)
		fileWatcher:     deps.FileWatcher,   // File system watching (optional)
		scanner:         deps.Scanner,       // Component scanning (optional but recommended)
		buildPipeline:   deps.BuildPipeline, // Build orchestration (optional)
		renderer:        deps.Renderer,      // Component rendering (optional)
		monitor:         deps.Monitor,       // Monitoring system (optional)
		wsManager:       deps.WSManager,     // WebSocket management (optional)
		ctx:             ctx,                // Cancellation context
		cancel:          cancel,             // Cancellation function
		lastBuildErrors: nil,                // No build errors initially
	}

	// Post-construction invariant validation
	if orchestrator.ctx == nil || orchestrator.cancel == nil {
		panic("ServiceOrchestrator: context initialization failed")
	}
	if orchestrator.config == nil {
		panic("ServiceOrchestrator: config storage failed")
	}

	log.Printf(
		"ServiceOrchestrator initialized with %d scan paths",
		len(deps.Config.Components.ScanPaths),
	)
	return orchestrator
}

// Start initializes and starts all coordinated services
func (so *ServiceOrchestrator) Start(ctx context.Context) error {
	// Start build pipeline
	if so.buildPipeline != nil {
		so.buildPipeline.Start(ctx)

		// Add build result callback
		so.buildPipeline.AddCallback(func(result interface{}) {
			if buildResult, ok := result.(build.BuildResult); ok {
				so.handleBuildResult(buildResult)
			}
		})
	}

	// Setup file watcher
	so.setupFileWatcher(ctx)

	// Perform initial component scan
	if err := so.initialScan(); err != nil {
		return fmt.Errorf("initial component scan failed: %w", err)
	}

	// Start file watching
	if so.fileWatcher != nil {
		if err := so.fileWatcher.Start(ctx); err != nil {
			log.Printf("Warning: Failed to start file watcher: %v", err)
		}
	}

	log.Printf("Service orchestrator started successfully")
	return nil
}

// setupFileWatcher configures the file watcher with appropriate filters and handlers
func (so *ServiceOrchestrator) setupFileWatcher(ctx context.Context) {
	if so.fileWatcher == nil {
		return
	}

	// Add filters for relevant file types
	so.fileWatcher.AddFilter(interfaces.FileFilterFunc(watcher.TemplFilter))
	so.fileWatcher.AddFilter(interfaces.FileFilterFunc(watcher.GoFilter))
	so.fileWatcher.AddFilter(interfaces.FileFilterFunc(watcher.NoTestFilter))
	so.fileWatcher.AddFilter(interfaces.FileFilterFunc(watcher.NoVendorFilter))
	so.fileWatcher.AddFilter(interfaces.FileFilterFunc(watcher.NoGitFilter))

	// Add change handler
	so.fileWatcher.AddHandler(func(events []interfaces.ChangeEvent) error {
		changeEvents := make([]watcher.ChangeEvent, len(events))
		for i, event := range events {
			// Convert interfaces.ChangeEvent to watcher.ChangeEvent
			changeEvents[i] = watcher.ChangeEvent(event)
		}
		return so.handleFileChange(changeEvents)
	})

	// Add watch paths from configuration
	for _, path := range so.config.Components.ScanPaths {
		if err := so.fileWatcher.AddRecursive(path); err != nil {
			log.Printf("Warning: Failed to watch directory %s: %v", path, err)
		}
	}
}

// initialScan performs the initial component scanning
func (so *ServiceOrchestrator) initialScan() error {
	if so.scanner == nil {
		return fmt.Errorf("component scanner not available")
	}

	// Scan all configured paths
	for _, path := range so.config.Components.ScanPaths {
		if err := so.scanner.ScanDirectory(path); err != nil {
			log.Printf("Warning: Failed to scan directory %s: %v", path, err)
		}
	}

	log.Printf("Initial component scan completed")
	return nil
}

// handleFileChange processes file change events and coordinates appropriate actions
func (so *ServiceOrchestrator) handleFileChange(events []watcher.ChangeEvent) error {
	if len(events) == 0 {
		return nil
	}

	log.Printf("Processing %d file changes", len(events))

	// Group events by type for efficient processing
	var templFiles []string
	var otherFiles []string

	for _, event := range events {
		if watcher.TemplFilter(event.Path) {
			templFiles = append(templFiles, event.Path)
		} else if watcher.GoFilter(event.Path) {
			otherFiles = append(otherFiles, event.Path)
		}
	}

	// Process template file changes
	for _, filePath := range templFiles {
		if err := so.processTemplateFileChange(filePath); err != nil {
			log.Printf("Error processing template file %s: %v", filePath, err)
		}
	}

	// Process other file changes
	for _, filePath := range otherFiles {
		if err := so.processGeneralFileChange(filePath); err != nil {
			log.Printf("Error processing file %s: %v", filePath, err)
		}
	}

	// Broadcast change notification
	so.broadcastFileChangeNotification(len(events))

	return nil
}

// processTemplateFileChange handles changes to template files
func (so *ServiceOrchestrator) processTemplateFileChange(filePath string) error {
	// Re-scan the specific file
	if so.scanner != nil {
		if err := so.scanner.ScanFile(filePath); err != nil {
			return fmt.Errorf("failed to rescan template file: %w", err)
		}
	}

	// Get the component from the registry
	if so.registry != nil {
		// Find component by file path
		components := so.registry.GetAll()
		for _, component := range components {
			if component.FilePath == filePath {
				// Trigger rebuild for this specific component
				if so.buildPipeline != nil {
					so.buildPipeline.Build(component)
				}
				break
			}
		}
	}

	return nil
}

// processGeneralFileChange handles changes to non-template files
func (so *ServiceOrchestrator) processGeneralFileChange(filePath string) error {
	log.Printf("Processing general file change: %s", filePath)

	// For Go files, trigger a full rebuild since they might affect templates
	if watcher.GoFilter(filePath) {
		so.triggerFullRebuild()
	}

	return nil
}

// handleBuildResult processes build results and updates system state
func (so *ServiceOrchestrator) handleBuildResult(result build.BuildResult) {
	so.buildMutex.Lock()
	defer so.buildMutex.Unlock()

	// Update last build errors
	if len(result.ParsedErrors) > 0 {
		so.lastBuildErrors = result.ParsedErrors
		log.Printf("Build completed with %d errors", len(result.ParsedErrors))
	} else {
		so.lastBuildErrors = nil
		log.Printf("Build completed successfully")
	}

	// Broadcast build result
	so.broadcastBuildResult(result)

	// Track metrics if monitoring is enabled
	if so.monitor != nil {
		if len(result.ParsedErrors) > 0 {
			so.monitor.RecordWebSocketEvent("build_error", len(result.ParsedErrors))
		} else {
			so.monitor.RecordWebSocketEvent("build_success", 1)
		}
	}
}

// triggerFullRebuild initiates a complete rebuild of all components
func (so *ServiceOrchestrator) triggerFullRebuild() {
	log.Printf("Triggering full component rebuild")

	if so.registry == nil {
		return
	}

	// Get all components and trigger rebuild
	components := so.registry.GetAll()
	for _, component := range components {
		if so.buildPipeline != nil {
			so.buildPipeline.Build(component)
		}
	}
}

// broadcastFileChangeNotification sends file change notifications to WebSocket clients
func (so *ServiceOrchestrator) broadcastFileChangeNotification(eventCount int) {
	if so.wsManager != nil {
		message := UpdateMessage{
			Type:      "file_change",
			Content:   fmt.Sprintf("%d files changed", eventCount),
			Timestamp: GetCurrentTime(),
		}
		so.wsManager.BroadcastMessage(message)
	}
}

// broadcastBuildResult sends build results to WebSocket clients
func (so *ServiceOrchestrator) broadcastBuildResult(result build.BuildResult) {
	if so.wsManager == nil {
		return
	}

	messageType := "build_success"
	content := "Build completed successfully"

	if len(result.ParsedErrors) > 0 {
		messageType = "build_error"
		content = fmt.Sprintf("Build failed with %d errors", len(result.ParsedErrors))
	}

	message := UpdateMessage{
		Type:      messageType,
		Content:   content,
		Timestamp: GetCurrentTime(),
	}

	so.wsManager.BroadcastMessage(message)
}

// OpenBrowser opens the default browser to the application URL
func (so *ServiceOrchestrator) OpenBrowser(url string) {
	if !so.config.Server.Open {
		return
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		log.Printf("Cannot open browser on %s", runtime.GOOS)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to open browser: %v", err)
		return
	}

	log.Printf("Successfully opened browser for URL: %s", url)
}

// GetBuildMetrics returns current build metrics
func (so *ServiceOrchestrator) GetBuildMetrics() interfaces.BuildMetrics {
	if so.buildPipeline != nil {
		return so.buildPipeline.GetMetrics()
	}

	// Return empty metrics interface implementation
	return &build.BuildMetrics{}
}

// GetLastBuildErrors returns the errors from the last build
func (so *ServiceOrchestrator) GetLastBuildErrors() []*errors.ParsedError {
	so.buildMutex.RLock()
	defer so.buildMutex.RUnlock()

	// Return a copy to avoid race conditions
	if so.lastBuildErrors == nil {
		return nil
	}

	errorsCopy := make([]*errors.ParsedError, len(so.lastBuildErrors))
	copy(errorsCopy, so.lastBuildErrors)
	return errorsCopy
}

// Shutdown gracefully shuts down all coordinated services
func (so *ServiceOrchestrator) Shutdown(ctx context.Context) error {
	var shutdownErr error

	so.shutdownOnce.Do(func() {
		log.Printf("Shutting down service orchestrator...")

		// Cancel context to stop all operations
		so.cancel()

		// Stop file watcher
		if so.fileWatcher != nil {
			so.fileWatcher.Stop()
		}

		// Stop build pipeline
		if so.buildPipeline != nil {
			so.buildPipeline.Stop()
		}

		log.Printf("Service orchestrator shut down successfully")
	})

	return shutdownErr
}

// GetComponentCount returns the number of registered components
func (so *ServiceOrchestrator) GetComponentCount() int {
	if so.registry != nil {
		return so.registry.Count()
	}
	return 0
}

// GetConnectedWebSocketClients returns the number of connected WebSocket clients
func (so *ServiceOrchestrator) GetConnectedWebSocketClients() int {
	if so.wsManager != nil {
		return so.wsManager.GetConnectedClients()
	}
	return 0
}

// IsHealthy performs a health check on all coordinated services
func (so *ServiceOrchestrator) IsHealthy() bool {
	// Check if essential services are available
	if so.registry == nil || so.scanner == nil {
		return false
	}

	// Additional health checks can be added here
	return true
}

// GetServiceStatus returns the status of all coordinated services
func (so *ServiceOrchestrator) GetServiceStatus() map[string]interface{} {
	status := make(map[string]interface{})

	status["registry_available"] = so.registry != nil
	status["scanner_available"] = so.scanner != nil
	status["build_pipeline_available"] = so.buildPipeline != nil
	status["file_watcher_available"] = so.fileWatcher != nil
	status["websocket_manager_available"] = so.wsManager != nil
	status["renderer_available"] = so.renderer != nil
	status["monitor_available"] = so.monitor != nil

	status["component_count"] = so.GetComponentCount()
	status["websocket_clients"] = so.GetConnectedWebSocketClients()

	return status
}
