package monitoring

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/conneroisu/templar/internal/logging"
)

// RecoveryAction represents an action to take when a health check fails
type RecoveryAction interface {
	Execute(ctx context.Context, check HealthCheck) error
	Name() string
	Description() string
}

// RecoveryActionFunc implements RecoveryAction as a function
type RecoveryActionFunc struct {
	name        string
	description string
	actionFn    func(ctx context.Context, check HealthCheck) error
}

// Execute runs the recovery action
func (r *RecoveryActionFunc) Execute(ctx context.Context, check HealthCheck) error {
	return r.actionFn(ctx, check)
}

// Name returns the action name
func (r *RecoveryActionFunc) Name() string {
	return r.name
}

// Description returns the action description
func (r *RecoveryActionFunc) Description() string {
	return r.description
}

// NewRecoveryActionFunc creates a new recovery action function
func NewRecoveryActionFunc(name, description string, actionFn func(ctx context.Context, check HealthCheck) error) *RecoveryActionFunc {
	return &RecoveryActionFunc{
		name:        name,
		description: description,
		actionFn:    actionFn,
	}
}

// RecoveryRule defines when and how to recover from health check failures
type RecoveryRule struct {
	CheckName           string           // Name of health check this applies to
	MinFailureCount     int              // Minimum consecutive failures before triggering recovery
	RecoveryTimeout     time.Duration    // Maximum time to wait for recovery
	CooldownPeriod      time.Duration    // Time to wait before attempting recovery again
	MaxRecoveryAttempts int              // Maximum recovery attempts before giving up
	Actions             []RecoveryAction // Recovery actions to execute in order
}

// SelfHealingSystem manages automated recovery from component failures
type SelfHealingSystem struct {
	healthMonitor   *HealthMonitor
	rules           map[string]*RecoveryRule
	recoveryHistory map[string]*RecoveryHistory
	mutex           sync.RWMutex
	logger          logging.Logger
	stopChan        chan struct{}
	wg              sync.WaitGroup
}

// RecoveryHistory tracks recovery attempts for a specific check
type RecoveryHistory struct {
	CheckName           string
	ConsecutiveFailures int
	LastFailureTime     time.Time
	LastRecoveryTime    time.Time
	RecoveryAttempts    int
	RecoverySuccessful  bool
	LastError           error
}

// RecoveryEvent represents a recovery attempt
type RecoveryEvent struct {
	CheckName     string
	Action        string
	Success       bool
	Error         error
	Duration      time.Duration
	Timestamp     time.Time
	AttemptNumber int
}

// NewSelfHealingSystem creates a new self-healing system
func NewSelfHealingSystem(healthMonitor *HealthMonitor, logger logging.Logger) *SelfHealingSystem {
	return &SelfHealingSystem{
		healthMonitor:   healthMonitor,
		rules:           make(map[string]*RecoveryRule),
		recoveryHistory: make(map[string]*RecoveryHistory),
		logger:          logger.WithComponent("self_healing"),
		stopChan:        make(chan struct{}),
	}
}

// RegisterRecoveryRule registers a recovery rule for a specific health check
func (shs *SelfHealingSystem) RegisterRecoveryRule(rule *RecoveryRule) {
	shs.mutex.Lock()
	defer shs.mutex.Unlock()

	shs.rules[rule.CheckName] = rule
	shs.logger.Info(context.Background(), "Registered recovery rule",
		"check_name", rule.CheckName,
		"min_failures", rule.MinFailureCount,
		"max_attempts", rule.MaxRecoveryAttempts)
}

// Start begins monitoring and automated recovery
func (shs *SelfHealingSystem) Start() {
	shs.wg.Add(1)
	go shs.monitorLoop()
	shs.logger.Info(context.Background(), "Self-healing system started")
}

// Stop stops the self-healing system
func (shs *SelfHealingSystem) Stop() {
	// Only close if not already closed
	select {
	case <-shs.stopChan:
		// Already closed
	default:
		close(shs.stopChan)
	}
	shs.wg.Wait()
	shs.logger.Info(context.Background(), "Self-healing system stopped")
}

// monitorLoop runs the self-healing monitoring loop
func (shs *SelfHealingSystem) monitorLoop() {
	defer shs.wg.Done()

	ticker := time.NewTicker(10 * time.Second) // Check every 10 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			shs.checkAndRecover()
		case <-shs.stopChan:
			return
		}
	}
}

// checkAndRecover examines health status and triggers recovery actions
func (shs *SelfHealingSystem) checkAndRecover() {
	health := shs.healthMonitor.GetHealth()

	shs.mutex.Lock()
	defer shs.mutex.Unlock()

	for checkName, check := range health.Checks {
		rule, hasRule := shs.rules[checkName]
		if !hasRule {
			continue
		}

		history := shs.getOrCreateHistory(checkName)

		if check.Status == HealthStatusHealthy {
			// Reset failure count on successful check
			if history.ConsecutiveFailures > 0 {
				shs.logger.Info(context.Background(), "Health check recovered",
					"check_name", checkName,
					"previous_failures", history.ConsecutiveFailures)
				history.ConsecutiveFailures = 0
				history.RecoverySuccessful = true
			}
			continue
		}

		// Health check is failing
		history.ConsecutiveFailures++
		history.LastFailureTime = time.Now()

		// Check if we should attempt recovery
		if shs.shouldAttemptRecovery(history, rule) {
			shs.logger.Warn(context.Background(), nil, "Triggering recovery for failed health check",
				"check_name", checkName,
				"consecutive_failures", history.ConsecutiveFailures,
				"recovery_attempt", history.RecoveryAttempts+1)

			shs.attemptRecovery(checkName, check, rule, history)
		}
	}
}

// shouldAttemptRecovery determines if recovery should be attempted
func (shs *SelfHealingSystem) shouldAttemptRecovery(history *RecoveryHistory, rule *RecoveryRule) bool {
	// Check minimum failure count
	if history.ConsecutiveFailures < rule.MinFailureCount {
		return false
	}

	// Check max recovery attempts
	if history.RecoveryAttempts >= rule.MaxRecoveryAttempts {
		return false
	}

	// Check cooldown period (handle zero time case)
	if !history.LastRecoveryTime.IsZero() && time.Since(history.LastRecoveryTime) < rule.CooldownPeriod {
		return false
	}

	return true
}

// attemptRecovery executes recovery actions for a failed health check
func (shs *SelfHealingSystem) attemptRecovery(checkName string, check HealthCheck, rule *RecoveryRule, history *RecoveryHistory) {
	history.RecoveryAttempts++
	history.LastRecoveryTime = time.Now()
	history.RecoverySuccessful = false

	ctx, cancel := context.WithTimeout(context.Background(), rule.RecoveryTimeout)
	defer cancel()

	for i, action := range rule.Actions {
		start := time.Now()
		err := action.Execute(ctx, check)
		duration := time.Since(start)

		if err != nil {
			shs.logger.Error(ctx, err, "Recovery action failed",
				"check_name", checkName,
				"action", action.Name(),
				"attempt", history.RecoveryAttempts,
				"duration", duration)
			history.LastError = err

			// Continue to next action on failure
			continue
		}

		shs.logger.Info(ctx, "Recovery action succeeded",
			"check_name", checkName,
			"action", action.Name(),
			"attempt", history.RecoveryAttempts,
			"duration", duration)

		// If any action succeeds, consider recovery successful
		history.RecoverySuccessful = true
		history.LastError = nil

		// Wait a moment for the system to stabilize before checking if more actions are needed
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}

		// Re-check health status to see if recovery was effective
		currentHealth := shs.healthMonitor.GetHealth()
		if currentCheck, exists := currentHealth.Checks[checkName]; exists && currentCheck.Status == HealthStatusHealthy {
			shs.logger.Info(ctx, "Recovery successful - health check now passing",
				"check_name", checkName,
				"action", action.Name(),
				"actions_executed", i+1)
			return
		}
	}

	if history.RecoveryAttempts >= rule.MaxRecoveryAttempts {
		shs.logger.Error(ctx, history.LastError, "Recovery failed - maximum attempts reached",
			"check_name", checkName,
			"max_attempts", rule.MaxRecoveryAttempts,
			"giving_up", true)
	}
}

// getOrCreateHistory gets or creates recovery history for a check
func (shs *SelfHealingSystem) getOrCreateHistory(checkName string) *RecoveryHistory {
	if history, exists := shs.recoveryHistory[checkName]; exists {
		return history
	}

	history := &RecoveryHistory{
		CheckName: checkName,
	}
	shs.recoveryHistory[checkName] = history
	return history
}

// GetRecoveryHistory returns the recovery history for all checks
func (shs *SelfHealingSystem) GetRecoveryHistory() map[string]*RecoveryHistory {
	shs.mutex.RLock()
	defer shs.mutex.RUnlock()

	history := make(map[string]*RecoveryHistory)
	for name, h := range shs.recoveryHistory {
		// Create a copy to avoid race conditions
		history[name] = &RecoveryHistory{
			CheckName:           h.CheckName,
			ConsecutiveFailures: h.ConsecutiveFailures,
			LastFailureTime:     h.LastFailureTime,
			LastRecoveryTime:    h.LastRecoveryTime,
			RecoveryAttempts:    h.RecoveryAttempts,
			RecoverySuccessful:  h.RecoverySuccessful,
			LastError:           h.LastError,
		}
	}
	return history
}

// Predefined recovery actions

// RestartServiceAction creates an action to restart a service
func RestartServiceAction(serviceName string, restartFn func() error) RecoveryAction {
	return NewRecoveryActionFunc(
		fmt.Sprintf("restart_%s", serviceName),
		fmt.Sprintf("Restart %s service", serviceName),
		func(ctx context.Context, check HealthCheck) error {
			return restartFn()
		},
	)
}

// ClearCacheAction creates an action to clear caches
func ClearCacheAction(cacheName string, clearFn func() error) RecoveryAction {
	return NewRecoveryActionFunc(
		fmt.Sprintf("clear_%s_cache", cacheName),
		fmt.Sprintf("Clear %s cache", cacheName),
		func(ctx context.Context, check HealthCheck) error {
			return clearFn()
		},
	)
}

// GarbageCollectAction creates an action to trigger garbage collection
func GarbageCollectAction() RecoveryAction {
	return NewRecoveryActionFunc(
		"garbage_collect",
		"Trigger garbage collection to free memory",
		func(ctx context.Context, check HealthCheck) error {
			// Force garbage collection
			runtime.GC()
			runtime.GC() // Call twice to ensure cleanup
			return nil
		},
	)
}

// WaitAction creates an action that waits for a specified duration
func WaitAction(duration time.Duration) RecoveryAction {
	return NewRecoveryActionFunc(
		fmt.Sprintf("wait_%s", duration),
		fmt.Sprintf("Wait for %s to allow system stabilization", duration),
		func(ctx context.Context, check HealthCheck) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(duration):
				return nil
			}
		},
	)
}

// LoggingAction creates an action that logs the health check failure
func LoggingAction(logger logging.Logger) RecoveryAction {
	return NewRecoveryActionFunc(
		"log_failure",
		"Log detailed information about the health check failure",
		func(ctx context.Context, check HealthCheck) error {
			logger.Error(ctx, nil, "Health check failure details",
				"check_name", check.Name,
				"status", string(check.Status),
				"message", check.Message,
				"metadata", check.Metadata,
				"last_checked", check.LastChecked,
				"duration", check.Duration)
			return nil
		},
	)
}
