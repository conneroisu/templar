package testing

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// ErrorInjector provides controlled failure injection for testing.
type ErrorInjector struct {
	targets     map[string]*ErrorTarget
	mu          sync.RWMutex
	enabled     bool
	probability float64 // 0.0 to 1.0
}

// ErrorTarget represents an injection point with configuration.
type ErrorTarget struct {
	Name        string
	Error       error
	Probability float64
	Count       int64 // Number of times to inject
	Remaining   int64 // Remaining injections (-1 for unlimited)
	Delay       time.Duration
	Enabled     bool
}

// NewErrorInjector creates a new error injection framework.
func NewErrorInjector() *ErrorInjector {
	return &ErrorInjector{
		targets:     make(map[string]*ErrorTarget),
		enabled:     true,
		probability: 1.0, // Default to always inject when configured
	}
}

// InjectError configures error injection for a specific operation.
func (ei *ErrorInjector) InjectError(operation string, err error) *ErrorTarget {
	ei.mu.Lock()
	defer ei.mu.Unlock()

	target := &ErrorTarget{
		Name:        operation,
		Error:       err,
		Probability: ei.probability,
		Count:       0,
		Remaining:   -1, // Unlimited by default
		Enabled:     true,
	}

	ei.targets[operation] = target

	return target
}

// InjectErrorOnce configures error injection for a single occurrence.
func (ei *ErrorInjector) InjectErrorOnce(operation string, err error) *ErrorTarget {
	target := ei.InjectError(operation, err)
	target.Remaining = 1

	return target
}

// InjectErrorCount configures error injection for a specific count.
func (ei *ErrorInjector) InjectErrorCount(operation string, err error, count int64) *ErrorTarget {
	target := ei.InjectError(operation, err)
	target.Remaining = count

	return target
}

// InjectErrorWithDelay configures error injection with a delay.
func (ei *ErrorInjector) InjectErrorWithDelay(
	operation string,
	err error,
	delay time.Duration,
) *ErrorTarget {
	target := ei.InjectError(operation, err)
	target.Delay = delay

	return target
}

// ShouldFail checks if an operation should fail and returns the error.
func (ei *ErrorInjector) ShouldFail(operation string) error {
	ei.mu.RLock()
	defer ei.mu.RUnlock()

	if !ei.enabled {
		return nil
	}

	target, exists := ei.targets[operation]
	if !exists || !target.Enabled {
		return nil
	}

	// Check if we've exceeded the injection count
	if target.Remaining == 0 {
		return nil
	}

	// Apply probability
	if target.Probability < 1.0 {
		// For testing, we'll use a deterministic approach based on count
		// In real scenarios, you might use rand.Float64()
		if float64(target.Count%100)/100.0 >= target.Probability {
			return nil
		}
	}

	// Apply delay if configured
	if target.Delay > 0 {
		time.Sleep(target.Delay)
	}

	// Update counters
	target.Count++
	if target.Remaining > 0 {
		target.Remaining--
	}

	return target.Error
}

// Enable enables error injection globally.
func (ei *ErrorInjector) Enable() {
	ei.mu.Lock()
	defer ei.mu.Unlock()
	ei.enabled = true
}

// Disable disables error injection globally.
func (ei *ErrorInjector) Disable() {
	ei.mu.Lock()
	defer ei.mu.Unlock()
	ei.enabled = false
}

// SetGlobalProbability sets the default probability for new targets.
func (ei *ErrorInjector) SetGlobalProbability(prob float64) {
	ei.mu.Lock()
	defer ei.mu.Unlock()
	if prob >= 0.0 && prob <= 1.0 {
		ei.probability = prob
	}
}

// Clear removes all error injection targets.
func (ei *ErrorInjector) Clear() {
	ei.mu.Lock()
	defer ei.mu.Unlock()
	ei.targets = make(map[string]*ErrorTarget)
}

// RemoveTarget removes a specific error injection target.
func (ei *ErrorInjector) RemoveTarget(operation string) {
	ei.mu.Lock()
	defer ei.mu.Unlock()
	delete(ei.targets, operation)
}

// GetTarget retrieves information about a specific target.
func (ei *ErrorInjector) GetTarget(operation string) (*ErrorTarget, bool) {
	ei.mu.RLock()
	defer ei.mu.RUnlock()
	target, exists := ei.targets[operation]
	if exists {
		// Return a copy to avoid race conditions
		return &ErrorTarget{
			Name:        target.Name,
			Error:       target.Error,
			Probability: target.Probability,
			Count:       target.Count,
			Remaining:   target.Remaining,
			Delay:       target.Delay,
			Enabled:     target.Enabled,
		}, true
	}

	return nil, false
}

// ListTargets returns all configured targets.
func (ei *ErrorInjector) ListTargets() map[string]*ErrorTarget {
	ei.mu.RLock()
	defer ei.mu.RUnlock()

	result := make(map[string]*ErrorTarget)
	for name, target := range ei.targets {
		result[name] = &ErrorTarget{
			Name:        target.Name,
			Error:       target.Error,
			Probability: target.Probability,
			Count:       target.Count,
			Remaining:   target.Remaining,
			Delay:       target.Delay,
			Enabled:     target.Enabled,
		}
	}

	return result
}

// GetStats returns statistics about error injections.
func (ei *ErrorInjector) GetStats() ErrorInjectionStats {
	ei.mu.RLock()
	defer ei.mu.RUnlock()

	stats := ErrorInjectionStats{
		Enabled:         ei.enabled,
		TotalTargets:    len(ei.targets),
		ActiveTargets:   0,
		TotalInjections: 0,
	}

	for _, target := range ei.targets {
		if target.Enabled && target.Remaining != 0 {
			stats.ActiveTargets++
		}
		stats.TotalInjections += target.Count
	}

	return stats
}

// ErrorInjectionStats contains statistics about error injection.
type ErrorInjectionStats struct {
	Enabled         bool
	TotalTargets    int
	ActiveTargets   int
	TotalInjections int64
}

// ErrorTarget configuration methods for fluent interface

// WithProbability sets the injection probability.
func (et *ErrorTarget) WithProbability(prob float64) *ErrorTarget {
	if prob >= 0.0 && prob <= 1.0 {
		et.Probability = prob
	}

	return et
}

// WithDelay sets the injection delay.
func (et *ErrorTarget) WithDelay(delay time.Duration) *ErrorTarget {
	et.Delay = delay

	return et
}

// Enable enables this specific target.
func (et *ErrorTarget) Enable() *ErrorTarget {
	et.Enabled = true

	return et
}

// Disable disables this specific target.
func (et *ErrorTarget) Disable() *ErrorTarget {
	et.Enabled = false

	return et
}

// Common error types for testing.
var (
	ErrFileNotFound       = errors.New("file not found")
	ErrPermissionDenied   = errors.New("permission denied")
	ErrNetworkTimeout     = errors.New("network timeout")
	ErrOutOfMemory        = errors.New("out of memory")
	ErrDiskFull           = errors.New("disk full")
	ErrInvalidInput       = errors.New("invalid input")
	ErrConnectionLost     = errors.New("connection lost")
	ErrServiceUnavailable = errors.New("service unavailable")
)

// ErrorScenario represents a complex error injection scenario.
type ErrorScenario struct {
	Name        string
	Description string
	Steps       []ErrorStep
}

// ErrorStep represents a single step in an error scenario.
type ErrorStep struct {
	Operation   string
	Error       error
	Delay       time.Duration
	Count       int64
	Probability float64
}

// ScenarioManager manages complex error injection scenarios.
type ScenarioManager struct {
	injector  *ErrorInjector
	scenarios map[string]*ErrorScenario
	mu        sync.RWMutex
}

// NewScenarioManager creates a new scenario manager.
func NewScenarioManager(injector *ErrorInjector) *ScenarioManager {
	return &ScenarioManager{
		injector:  injector,
		scenarios: make(map[string]*ErrorScenario),
	}
}

// RegisterScenario registers a new error scenario.
func (sm *ScenarioManager) RegisterScenario(scenario *ErrorScenario) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.scenarios[scenario.Name] = scenario
}

// ExecuteScenario executes a specific error scenario.
func (sm *ScenarioManager) ExecuteScenario(name string) error {
	sm.mu.RLock()
	scenario, exists := sm.scenarios[name]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("scenario '%s' not found", name)
	}

	// Configure error injection for each step
	for _, step := range scenario.Steps {
		target := sm.injector.InjectErrorCount(step.Operation, step.Error, step.Count)
		if step.Probability > 0 {
			target.WithProbability(step.Probability)
		}
		if step.Delay > 0 {
			target.WithDelay(step.Delay)
		}
	}

	return nil
}

// StopScenario stops a running scenario by clearing its targets.
func (sm *ScenarioManager) StopScenario(name string) error {
	sm.mu.RLock()
	scenario, exists := sm.scenarios[name]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("scenario '%s' not found", name)
	}

	// Remove error injection targets for this scenario
	for _, step := range scenario.Steps {
		sm.injector.RemoveTarget(step.Operation)
	}

	return nil
}

// ListScenarios returns all registered scenarios.
func (sm *ScenarioManager) ListScenarios() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	scenarios := make([]string, 0, len(sm.scenarios))
	for name := range sm.scenarios {
		scenarios = append(scenarios, name)
	}

	return scenarios
}

// Predefined scenarios for common testing situations

// CreateBuildFailureScenario creates a scenario for build pipeline failures.
func CreateBuildFailureScenario() *ErrorScenario {
	return &ErrorScenario{
		Name:        "build_failure",
		Description: "Simulate build pipeline failures",
		Steps: []ErrorStep{
			{
				Operation:   "file.read",
				Error:       ErrPermissionDenied,
				Count:       3,
				Probability: 0.3,
			},
			{
				Operation: "exec.command",
				Error:     errors.New("command not found"),
				Count:     2,
				Delay:     100 * time.Millisecond,
			},
			{
				Operation:   "file.write",
				Error:       ErrDiskFull,
				Count:       1,
				Probability: 0.1,
			},
		},
	}
}

// CreateNetworkFailureScenario creates a scenario for network-related failures.
func CreateNetworkFailureScenario() *ErrorScenario {
	return &ErrorScenario{
		Name:        "network_failure",
		Description: "Simulate network connectivity issues",
		Steps: []ErrorStep{
			{
				Operation:   "websocket.connect",
				Error:       ErrConnectionLost,
				Count:       5,
				Probability: 0.2,
			},
			{
				Operation: "http.request",
				Error:     ErrNetworkTimeout,
				Count:     3,
				Delay:     500 * time.Millisecond,
			},
			{
				Operation:   "websocket.send",
				Error:       ErrServiceUnavailable,
				Count:       10,
				Probability: 0.1,
			},
		},
	}
}

// CreateResourceExhaustionScenario creates a scenario for resource exhaustion.
func CreateResourceExhaustionScenario() *ErrorScenario {
	return &ErrorScenario{
		Name:        "resource_exhaustion",
		Description: "Simulate resource exhaustion conditions",
		Steps: []ErrorStep{
			{
				Operation:   "memory.alloc",
				Error:       ErrOutOfMemory,
				Count:       2,
				Probability: 0.05,
			},
			{
				Operation:   "file.create",
				Error:       ErrDiskFull,
				Count:       1,
				Probability: 0.02,
			},
			{
				Operation:   "goroutine.start",
				Error:       errors.New("too many goroutines"),
				Count:       1,
				Probability: 0.01,
			},
		},
	}
}
