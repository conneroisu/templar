package monitoring

import (
	"net/http"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/interfaces"
	"github.com/conneroisu/templar/internal/logging"
)

// SetupComprehensiveHealthSystem creates a complete health monitoring and self-healing system
func SetupComprehensiveHealthSystem(deps *ComprehensiveDependencies) (*ComprehensiveHealthSystem, error) {
	// Create health monitor
	healthMonitor := NewHealthMonitor(deps.Logger)

	// Register predefined health checks
	healthMonitor.RegisterCheck(FileSystemHealthChecker("."))
	healthMonitor.RegisterCheck(MemoryHealthChecker())
	healthMonitor.RegisterCheck(GoroutineHealthChecker())

	// Register component-specific health checks if available
	if deps.BuildPipeline != nil {
		healthMonitor.RegisterCheck(CreateBuildPipelineHealthChecker(deps.BuildPipeline))
	}
	if deps.Registry != nil {
		healthMonitor.RegisterCheck(CreateComponentRegistryHealthChecker(deps.Registry))
	}
	if deps.FileWatcher != nil {
		healthMonitor.RegisterCheck(CreateFileWatcherHealthChecker(deps.FileWatcher))
	}

	// Create self-healing system
	selfHealingDeps := &SelfHealingDependencies{
		Logger:        deps.Logger,
		BuildPipeline: deps.BuildPipeline,
		Registry:      deps.Registry,
		Scanner:       deps.Scanner,
		FileWatcher:   deps.FileWatcher,
	}
	selfHealingSystem := SetupSelfHealingSystem(healthMonitor, selfHealingDeps)

	// Create dashboard
	dashboard := NewHealthDashboard(healthMonitor, selfHealingSystem, deps.Logger)

	return &ComprehensiveHealthSystem{
		HealthMonitor:     healthMonitor,
		SelfHealingSystem: selfHealingSystem,
		Dashboard:         dashboard,
		Config:            deps.Config,
	}, nil
}

// ComprehensiveDependencies contains all dependencies needed for the complete health system
type ComprehensiveDependencies struct {
	Config        *config.Config
	Logger        logging.Logger
	BuildPipeline interfaces.BuildPipeline
	Registry      interfaces.ComponentRegistry
	Scanner       interfaces.ComponentScanner
	FileWatcher   interfaces.FileWatcher
}

// ComprehensiveHealthSystem combines all health monitoring components
type ComprehensiveHealthSystem struct {
	HealthMonitor     *HealthMonitor
	SelfHealingSystem *SelfHealingSystem
	Dashboard         *HealthDashboard
	Config            *config.Config
}

// Start starts all components of the health system
func (chs *ComprehensiveHealthSystem) Start() {
	chs.HealthMonitor.Start()
	chs.SelfHealingSystem.Start()
}

// Stop stops all components of the health system
func (chs *ComprehensiveHealthSystem) Stop() {
	chs.SelfHealingSystem.Stop()
	chs.HealthMonitor.Stop()
}

// HTTPHandler returns an HTTP handler for health-related endpoints
func (chs *ComprehensiveHealthSystem) HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Route health-related requests
		switch {
		case r.URL.Path == "/health":
			// Standard health endpoint
			chs.HealthMonitor.HTTPHandler()(w, r)
		case r.URL.Path == "/health-dashboard" ||
			r.URL.Path == "/health-dashboard/api/data" ||
			r.URL.Path == "/health-dashboard/api/recovery":
			// Dashboard endpoints
			chs.Dashboard.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	}
}

// AddToServerMux adds health endpoints to an existing HTTP server mux
func (chs *ComprehensiveHealthSystem) AddToServerMux(mux *http.ServeMux) {
	// Standard health endpoint (existing endpoint, enhanced)
	mux.HandleFunc("/health", chs.HealthMonitor.HTTPHandler())

	// Dashboard endpoints (new)
	mux.HandleFunc("/health-dashboard", chs.Dashboard.ServeHTTP)
	mux.HandleFunc("/health-dashboard/", chs.Dashboard.ServeHTTP)

	// API endpoints (new)
	mux.HandleFunc("/api/health/status", chs.HealthMonitor.HTTPHandler())
	mux.HandleFunc("/api/health/recovery", chs.Dashboard.ServeHTTP)
}

// GetHealthStatus returns the current health status
func (chs *ComprehensiveHealthSystem) GetHealthStatus() HealthResponse {
	return chs.HealthMonitor.GetHealth()
}

// GetRecoveryHistory returns the current recovery history
func (chs *ComprehensiveHealthSystem) GetRecoveryHistory() map[string]*RecoveryHistory {
	return chs.SelfHealingSystem.GetRecoveryHistory()
}

// IsHealthy returns true if all critical health checks are passing
func (chs *ComprehensiveHealthSystem) IsHealthy() bool {
	health := chs.HealthMonitor.GetHealth()
	return health.Status == HealthStatusHealthy
}

// IsDegraded returns true if the system is in a degraded state
func (chs *ComprehensiveHealthSystem) IsDegraded() bool {
	health := chs.HealthMonitor.GetHealth()
	return health.Status == HealthStatusDegraded
}

// GetSystemSummary returns a human-readable summary of system health
func (chs *ComprehensiveHealthSystem) GetSystemSummary() string {
	health := chs.HealthMonitor.GetHealth()

	switch health.Status {
	case HealthStatusHealthy:
		return "✅ All systems operational"
	case HealthStatusDegraded:
		return "⚠️ System degraded - some non-critical components have issues"
	case HealthStatusUnhealthy:
		return "❌ System unhealthy - critical components are failing"
	default:
		return "❓ System status unknown"
	}
}

// ForceHealthCheck triggers an immediate health check of all registered checks
func (chs *ComprehensiveHealthSystem) ForceHealthCheck() {
	// The health monitor doesn't expose a public method to force checks
	// This would need to be added to the HealthMonitor interface
	// For now, we'll just return the current status
}

// RegisterCustomHealthCheck allows adding application-specific health checks
func (chs *ComprehensiveHealthSystem) RegisterCustomHealthCheck(checker HealthChecker) {
	chs.HealthMonitor.RegisterCheck(checker)
}

// RegisterCustomRecoveryRule allows adding application-specific recovery rules
func (chs *ComprehensiveHealthSystem) RegisterCustomRecoveryRule(rule *RecoveryRule) {
	chs.SelfHealingSystem.RegisterRecoveryRule(rule)
}
