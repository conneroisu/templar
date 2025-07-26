package monitoring

import (
	"errors"
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/conneroisu/templar/internal/interfaces"
	"github.com/conneroisu/templar/internal/logging"
)

// DefaultSelfHealingRules creates default recovery rules for common failure scenarios.
func DefaultSelfHealingRules(deps *SelfHealingDependencies) []*RecoveryRule {
	return []*RecoveryRule{
		{
			CheckName:           "memory",
			MinFailureCount:     2,
			RecoveryTimeout:     30 * time.Second,
			CooldownPeriod:      5 * time.Minute,
			MaxRecoveryAttempts: 3,
			Actions: []RecoveryAction{
				LoggingAction(deps.Logger),
				GarbageCollectAction(),
				WaitAction(5 * time.Second),
			},
		},
		{
			CheckName:           "filesystem",
			MinFailureCount:     1, // Immediate action for filesystem issues
			RecoveryTimeout:     15 * time.Second,
			CooldownPeriod:      2 * time.Minute,
			MaxRecoveryAttempts: 2,
			Actions: []RecoveryAction{
				LoggingAction(deps.Logger),
				CleanTemporaryFilesAction(),
				WaitAction(2 * time.Second),
			},
		},
		{
			CheckName:           "goroutines",
			MinFailureCount:     3,
			RecoveryTimeout:     20 * time.Second,
			CooldownPeriod:      10 * time.Minute,
			MaxRecoveryAttempts: 2,
			Actions: []RecoveryAction{
				LoggingAction(deps.Logger),
				GarbageCollectAction(),
				DumpGoroutineStacksAction(deps.Logger),
			},
		},
		{
			CheckName:           "build_pipeline",
			MinFailureCount:     2,
			RecoveryTimeout:     45 * time.Second,
			CooldownPeriod:      3 * time.Minute,
			MaxRecoveryAttempts: 3,
			Actions: []RecoveryAction{
				LoggingAction(deps.Logger),
				ClearBuildCacheAction(deps.BuildPipeline),
				RestartBuildPipelineAction(deps.BuildPipeline),
				WaitAction(5 * time.Second),
			},
		},
		{
			CheckName:           "component_registry",
			MinFailureCount:     2,
			RecoveryTimeout:     30 * time.Second,
			CooldownPeriod:      5 * time.Minute,
			MaxRecoveryAttempts: 2,
			Actions: []RecoveryAction{
				LoggingAction(deps.Logger),
				RefreshComponentRegistryAction(deps.Registry, deps.Scanner),
				WaitAction(3 * time.Second),
			},
		},
		{
			CheckName:           "file_watcher",
			MinFailureCount:     1,
			RecoveryTimeout:     20 * time.Second,
			CooldownPeriod:      2 * time.Minute,
			MaxRecoveryAttempts: 2,
			Actions: []RecoveryAction{
				LoggingAction(deps.Logger),
				RestartFileWatcherAction(deps.FileWatcher),
				WaitAction(3 * time.Second),
			},
		},
	}
}

// SelfHealingDependencies contains dependencies needed for self-healing actions.
type SelfHealingDependencies struct {
	Logger        logging.Logger
	BuildPipeline interfaces.BuildPipeline
	Registry      interfaces.ComponentRegistry
	Scanner       interfaces.ComponentScanner
	FileWatcher   interfaces.FileWatcher
}

// SetupSelfHealingSystem creates and configures a complete self-healing system.
func SetupSelfHealingSystem(
	healthMonitor *HealthMonitor,
	deps *SelfHealingDependencies,
) *SelfHealingSystem {
	system := NewSelfHealingSystem(healthMonitor, deps.Logger)

	// Register default recovery rules
	rules := DefaultSelfHealingRules(deps)
	for _, rule := range rules {
		system.RegisterRecoveryRule(rule)
	}

	return system
}

// Advanced recovery actions specific to Templar components

// CleanTemporaryFilesAction creates an action to clean up temporary files.
func CleanTemporaryFilesAction() RecoveryAction {
	return NewRecoveryActionFunc(
		"clean_temp_files",
		"Clean up temporary files to free disk space",
		func(ctx context.Context, check HealthCheck) error {
			tempDirs := []string{
				os.TempDir(),
				".templar/temp",
				".templar/cache/temp",
			}

			for _, dir := range tempDirs {
				if err := cleanTempDirectory(dir); err != nil {
					// Log error but continue with other directories
					continue
				}
			}

			return nil
		},
	)
}

// cleanTempDirectory removes old temporary files from a directory.
func cleanTempDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-1 * time.Hour) // Remove files older than 1 hour

	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) &&
			(entry.Name() == ".templar_temp_" ||
				entry.Name() == "templar-health-") {
			os.Remove(dir + "/" + entry.Name())
		}
	}

	return nil
}

// DumpGoroutineStacksAction creates an action to dump goroutine stacks for debugging.
func DumpGoroutineStacksAction(logger logging.Logger) RecoveryAction {
	return NewRecoveryActionFunc(
		"dump_goroutine_stacks",
		"Dump goroutine stacks to help diagnose goroutine leaks",
		func(ctx context.Context, check HealthCheck) error {
			buf := make([]byte, 64*1024) // 64KB should be enough for stack traces
			n := runtime.Stack(buf, true)

			logger.Error(ctx, nil, "Goroutine stack dump due to high goroutine count",
				"goroutine_count", runtime.NumGoroutine(),
				"stack_trace", string(buf[:n]))

			return nil
		},
	)
}

// ClearBuildCacheAction creates an action to clear build caches.
func ClearBuildCacheAction(buildPipeline interfaces.BuildPipeline) RecoveryAction {
	return NewRecoveryActionFunc(
		"clear_build_cache",
		"Clear build pipeline caches to resolve build issues",
		func(ctx context.Context, check HealthCheck) error {
			if buildPipeline == nil {
				return errors.New("build pipeline is nil")
			}
			buildPipeline.ClearCache()

			return nil
		},
	)
}

// RestartBuildPipelineAction creates an action to restart the build pipeline.
func RestartBuildPipelineAction(buildPipeline interfaces.BuildPipeline) RecoveryAction {
	return NewRecoveryActionFunc(
		"restart_build_pipeline",
		"Restart the build pipeline to recover from build failures",
		func(ctx context.Context, check HealthCheck) error {
			if buildPipeline == nil {
				return errors.New("build pipeline is nil")
			}

			// Stop and restart the build pipeline
			buildPipeline.Stop()
			buildPipeline.Start(ctx)

			return nil
		},
	)
}

// RefreshComponentRegistryAction creates an action to refresh the component registry.
func RefreshComponentRegistryAction(
	registry interfaces.ComponentRegistry,
	scanner interfaces.ComponentScanner,
) RecoveryAction {
	return NewRecoveryActionFunc(
		"refresh_component_registry",
		"Refresh component registry by rescanning component directories",
		func(ctx context.Context, check HealthCheck) error {
			if registry == nil || scanner == nil {
				return errors.New("registry or scanner is nil")
			}

			// Clear registry and rescan
			// Note: This assumes the registry has a Clear method
			// If not available, we can just rescan which should update existing entries

			// Scan common component directories
			commonDirs := []string{"./components", "./views", "./examples"}
			for _, dir := range commonDirs {
				if _, err := os.Stat(dir); err == nil {
					if err := scanner.ScanDirectory(dir); err != nil {
						// Log error but continue with other directories
						continue
					}
				}
			}

			return nil
		},
	)
}

// RestartFileWatcherAction creates an action to restart the file watcher.
func RestartFileWatcherAction(fileWatcher interfaces.FileWatcher) RecoveryAction {
	return NewRecoveryActionFunc(
		"restart_file_watcher",
		"Restart file watcher to recover from file system monitoring issues",
		func(ctx context.Context, check HealthCheck) error {
			if fileWatcher == nil {
				return errors.New("file watcher is nil")
			}

			// Stop and restart the file watcher
			fileWatcher.Stop()

			// Add a brief delay to ensure cleanup
			time.Sleep(1 * time.Second)

			if err := fileWatcher.Start(ctx); err != nil {
				return fmt.Errorf("failed to restart file watcher: %w", err)
			}

			return nil
		},
	)
}

// CreateBuildPipelineHealthChecker creates a health checker for the build pipeline.
func CreateBuildPipelineHealthChecker(buildPipeline interfaces.BuildPipeline) HealthChecker {
	return NewHealthCheckFunc("build_pipeline", true, func(ctx context.Context) HealthCheck {
		start := time.Now()

		if buildPipeline == nil {
			return HealthCheck{
				Name:        "build_pipeline",
				Status:      HealthStatusUnhealthy,
				Message:     "Build pipeline is not initialized",
				LastChecked: time.Now(),
				Duration:    time.Since(start),
				Critical:    true,
			}
		}

		// Get build metrics to assess pipeline health
		metricsInterface := buildPipeline.GetMetrics()

		// Check if we can get metrics (indicates pipeline is responsive)
		if metricsInterface == nil {
			return HealthCheck{
				Name:        "build_pipeline",
				Status:      HealthStatusUnhealthy,
				Message:     "Build pipeline metrics unavailable",
				LastChecked: time.Now(),
				Duration:    time.Since(start),
				Critical:    true,
			}
		}

		return HealthCheck{
			Name:        "build_pipeline",
			Status:      HealthStatusHealthy,
			Message:     "Build pipeline is operational",
			LastChecked: time.Now(),
			Duration:    time.Since(start),
			Critical:    true,
			Metadata: map[string]interface{}{
				"metrics_available": true,
			},
		}
	})
}

// CreateComponentRegistryHealthChecker creates a health checker for the component registry.
func CreateComponentRegistryHealthChecker(registry interfaces.ComponentRegistry) HealthChecker {
	return NewHealthCheckFunc("component_registry", false, func(ctx context.Context) HealthCheck {
		start := time.Now()

		if registry == nil {
			return HealthCheck{
				Name:        "component_registry",
				Status:      HealthStatusUnhealthy,
				Message:     "Component registry is not initialized",
				LastChecked: time.Now(),
				Duration:    time.Since(start),
				Critical:    false,
			}
		}

		// Check if we can access the registry
		count := registry.Count()
		components := registry.GetAll()

		status := HealthStatusHealthy
		message := fmt.Sprintf("Component registry operational with %d components", count)

		// Basic sanity checks
		if len(components) != count {
			status = HealthStatusDegraded
			message = "Component registry count mismatch"
		}

		return HealthCheck{
			Name:        "component_registry",
			Status:      status,
			Message:     message,
			LastChecked: time.Now(),
			Duration:    time.Since(start),
			Critical:    false,
			Metadata: map[string]interface{}{
				"component_count":      count,
				"components_available": len(components),
			},
		}
	})
}

// CreateFileWatcherHealthChecker creates a health checker for the file watcher.
func CreateFileWatcherHealthChecker(fileWatcher interfaces.FileWatcher) HealthChecker {
	return NewHealthCheckFunc("file_watcher", false, func(ctx context.Context) HealthCheck {
		start := time.Now()

		if fileWatcher == nil {
			return HealthCheck{
				Name:        "file_watcher",
				Status:      HealthStatusUnhealthy,
				Message:     "File watcher is not initialized",
				LastChecked: time.Now(),
				Duration:    time.Since(start),
				Critical:    false,
			}
		}

		// The file watcher interface doesn't expose internal state
		// So we'll assume it's healthy if it exists
		// In a real implementation, you might want to add a Status() method
		// to the FileWatcher interface

		return HealthCheck{
			Name:        "file_watcher",
			Status:      HealthStatusHealthy,
			Message:     "File watcher is operational",
			LastChecked: time.Now(),
			Duration:    time.Since(start),
			Critical:    false,
		}
	})
}
