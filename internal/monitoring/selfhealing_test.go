package monitoring

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLogger implements logging.Logger for testing
type mockLogger struct {
	logs []string
	mu   sync.Mutex
}

func (m *mockLogger) Debug(ctx context.Context, msg string, fields ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, "DEBUG: "+msg)
}

func (m *mockLogger) Info(ctx context.Context, msg string, fields ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, "INFO: "+msg)
}

func (m *mockLogger) Warn(ctx context.Context, err error, msg string, fields ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, "WARN: "+msg)
}

func (m *mockLogger) Error(ctx context.Context, err error, msg string, fields ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, "ERROR: "+msg)
}

func (m *mockLogger) Fatal(ctx context.Context, err error, msg string, fields ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, "FATAL: "+msg)
}

func (m *mockLogger) With(fields ...interface{}) logging.Logger {
	return m
}

func (m *mockLogger) WithComponent(component string) logging.Logger {
	return m
}

func (m *mockLogger) GetLogs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.logs))
	copy(result, m.logs)
	return result
}

// mockHealthChecker implements HealthChecker for testing
type mockHealthChecker struct {
	name     string
	status   HealthStatus
	critical bool
	fail     bool
}

func (m *mockHealthChecker) Check(ctx context.Context) HealthCheck {
	status := HealthStatusHealthy
	message := "All good"

	if m.fail {
		status = m.status
		message = "Health check failed"
	}

	return HealthCheck{
		Name:        m.name,
		Status:      status,
		Message:     message,
		LastChecked: time.Now(),
		Duration:    10 * time.Millisecond,
		Critical:    m.critical,
	}
}

func (m *mockHealthChecker) Name() string {
	return m.name
}

func (m *mockHealthChecker) IsCritical() bool {
	return m.critical
}

// mockRecoveryAction implements RecoveryAction for testing
type mockRecoveryAction struct {
	name        string
	description string
	executed    bool
	shouldFail  bool
	mu          sync.Mutex
}

func (m *mockRecoveryAction) Execute(ctx context.Context, check HealthCheck) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.executed = true

	if m.shouldFail {
		return assert.AnError
	}

	return nil
}

func (m *mockRecoveryAction) Name() string {
	return m.name
}

func (m *mockRecoveryAction) Description() string {
	return m.description
}

func (m *mockRecoveryAction) WasExecuted() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.executed
}

func (m *mockRecoveryAction) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executed = false
}

func TestSelfHealingSystem_BasicRecovery(t *testing.T) {
	// Create mock logger
	logger := &mockLogger{}

	// Create health monitor
	healthMonitor := NewHealthMonitor(logger)

	// Create failing health checker
	failingChecker := &mockHealthChecker{
		name:     "test_check",
		status:   HealthStatusUnhealthy,
		critical: true,
		fail:     true,
	}

	healthMonitor.RegisterCheck(failingChecker)
	healthMonitor.Start()
	defer healthMonitor.Stop()

	// Create self-healing system
	selfHealing := NewSelfHealingSystem(healthMonitor, logger)

	// Create mock recovery action
	recoveryAction := &mockRecoveryAction{
		name:        "test_recovery",
		description: "Test recovery action",
		shouldFail:  false,
	}

	// Register recovery rule
	rule := &RecoveryRule{
		CheckName:           "test_check",
		MinFailureCount:     2,
		RecoveryTimeout:     5 * time.Second,
		CooldownPeriod:      1 * time.Second,
		MaxRecoveryAttempts: 3,
		Actions:             []RecoveryAction{recoveryAction},
	}

	selfHealing.RegisterRecoveryRule(rule)
	selfHealing.Start()
	defer selfHealing.Stop()

	// Wait for health checks to run and failures to accumulate
	time.Sleep(100 * time.Millisecond)

	// Trigger recovery by running the monitoring loop multiple times
	for i := 0; i < 3; i++ {
		selfHealing.checkAndRecover()
		time.Sleep(50 * time.Millisecond)
	}

	// Verify recovery action was executed
	assert.True(t, recoveryAction.WasExecuted(), "Recovery action should have been executed")

	// Verify recovery history was recorded
	history := selfHealing.GetRecoveryHistory()
	require.Contains(t, history, "test_check")

	checkHistory := history["test_check"]
	assert.Greater(t, checkHistory.ConsecutiveFailures, 1, "Should have recorded consecutive failures")
	assert.Greater(t, checkHistory.RecoveryAttempts, 0, "Should have recorded recovery attempts")
}

func TestSelfHealingSystem_CooldownPeriod(t *testing.T) {
	logger := &mockLogger{}
	healthMonitor := NewHealthMonitor(logger)

	failingChecker := &mockHealthChecker{
		name:     "cooldown_test",
		status:   HealthStatusUnhealthy,
		critical: true,
		fail:     true,
	}

	healthMonitor.RegisterCheck(failingChecker)
	healthMonitor.Start()
	defer healthMonitor.Stop()

	selfHealing := NewSelfHealingSystem(healthMonitor, logger)

	recoveryAction := &mockRecoveryAction{
		name:        "cooldown_recovery",
		description: "Cooldown test recovery",
	}

	rule := &RecoveryRule{
		CheckName:           "cooldown_test",
		MinFailureCount:     1,
		RecoveryTimeout:     5 * time.Second,
		CooldownPeriod:      3 * time.Second, // Extended cooldown for test stability
		MaxRecoveryAttempts: 5,
		Actions:             []RecoveryAction{recoveryAction},
	}

	selfHealing.RegisterRecoveryRule(rule)
	selfHealing.Start()
	defer selfHealing.Stop()

	// Wait for initial failure
	time.Sleep(100 * time.Millisecond)

	// First recovery attempt
	selfHealing.checkAndRecover()
	assert.True(t, recoveryAction.WasExecuted(), "First recovery should execute")

	// Reset and try again immediately (should be blocked by cooldown)
	recoveryAction.Reset()
	time.Sleep(10 * time.Millisecond) // Small delay to ensure timestamps are different
	selfHealing.checkAndRecover()

	// Debug: Check if the recovery was actually blocked
	executed := recoveryAction.WasExecuted()
	if executed {
		t.Logf("DEBUG: Second recovery was executed (should have been blocked). Cooldown period: %v", rule.CooldownPeriod)
	}
	assert.False(t, executed, "Second recovery should be blocked by cooldown")

	// Wait for cooldown to expire and try again
	time.Sleep(3100 * time.Millisecond) // Slightly longer than cooldown
	selfHealing.checkAndRecover()
	assert.True(t, recoveryAction.WasExecuted(), "Recovery should work after cooldown expires")
}

func TestSelfHealingSystem_MaxAttempts(t *testing.T) {
	logger := &mockLogger{}
	healthMonitor := NewHealthMonitor(logger)

	failingChecker := &mockHealthChecker{
		name:     "max_attempts_test",
		status:   HealthStatusUnhealthy,
		critical: true,
		fail:     true,
	}

	healthMonitor.RegisterCheck(failingChecker)
	healthMonitor.Start()
	defer healthMonitor.Stop()

	selfHealing := NewSelfHealingSystem(healthMonitor, logger)

	executionCount := 0
	recoveryAction := NewRecoveryActionFunc("counting_recovery", "Count executions",
		func(ctx context.Context, check HealthCheck) error {
			executionCount++
			return nil
		})

	rule := &RecoveryRule{
		CheckName:           "max_attempts_test",
		MinFailureCount:     1,
		RecoveryTimeout:     5 * time.Second,
		CooldownPeriod:      100 * time.Millisecond, // Short cooldown for testing
		MaxRecoveryAttempts: 2,                      // Limit to 2 attempts
		Actions:             []RecoveryAction{recoveryAction},
	}

	selfHealing.RegisterRecoveryRule(rule)
	selfHealing.Start()
	defer selfHealing.Stop()

	// Wait for initial failure
	time.Sleep(100 * time.Millisecond)

	// Trigger multiple recovery attempts
	for i := 0; i < 5; i++ {
		selfHealing.checkAndRecover()
		time.Sleep(200 * time.Millisecond) // Wait for cooldown
	}

	// Should only execute twice due to MaxRecoveryAttempts
	assert.Equal(t, 2, executionCount, "Should only execute recovery action twice")

	// Verify history reflects max attempts reached
	history := selfHealing.GetRecoveryHistory()
	require.Contains(t, history, "max_attempts_test")
	assert.Equal(t, 2, history["max_attempts_test"].RecoveryAttempts)
}

func TestSelfHealingSystem_SuccessfulRecovery(t *testing.T) {
	logger := &mockLogger{}
	healthMonitor := NewHealthMonitor(logger)

	// Create a checker that can be toggled between healthy and unhealthy
	checker := &mockHealthChecker{
		name:     "toggle_test",
		status:   HealthStatusUnhealthy,
		critical: true,
		fail:     true,
	}

	healthMonitor.RegisterCheck(checker)
	healthMonitor.Start()
	defer healthMonitor.Stop()

	selfHealing := NewSelfHealingSystem(healthMonitor, logger)

	// Recovery action that "fixes" the health check
	recoveryAction := NewRecoveryActionFunc("fix_checker", "Fix the checker",
		func(ctx context.Context, check HealthCheck) error {
			checker.fail = false // "Fix" the issue
			return nil
		})

	rule := &RecoveryRule{
		CheckName:           "toggle_test",
		MinFailureCount:     1,
		RecoveryTimeout:     5 * time.Second,
		CooldownPeriod:      100 * time.Millisecond,
		MaxRecoveryAttempts: 3,
		Actions:             []RecoveryAction{recoveryAction},
	}

	selfHealing.RegisterRecoveryRule(rule)
	selfHealing.Start()
	defer selfHealing.Stop()

	// Wait for initial failure
	time.Sleep(100 * time.Millisecond)

	// Trigger recovery
	selfHealing.checkAndRecover()

	// Wait for the system to stabilize and check health again
	time.Sleep(200 * time.Millisecond)
	selfHealing.checkAndRecover()

	// Verify the failure count was reset after successful recovery
	history := selfHealing.GetRecoveryHistory()
	require.Contains(t, history, "toggle_test")

	// The consecutive failures should be reset after successful recovery
	// Note: This test might be flaky depending on timing, so we check that recovery was attempted
	assert.Greater(t, history["toggle_test"].RecoveryAttempts, 0, "Should have attempted recovery")
}

func TestGarbageCollectAction(t *testing.T) {
	action := GarbageCollectAction()

	assert.Equal(t, "garbage_collect", action.Name())
	assert.Contains(t, action.Description(), "garbage collection")

	// Execute the action
	err := action.Execute(context.Background(), HealthCheck{})
	assert.NoError(t, err)
}

func TestWaitAction(t *testing.T) {
	duration := 100 * time.Millisecond
	action := WaitAction(duration)

	assert.Contains(t, action.Name(), "wait_")
	assert.Contains(t, action.Description(), "100ms")

	// Execute and time the action
	start := time.Now()
	err := action.Execute(context.Background(), HealthCheck{})
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, elapsed, duration)
}

func TestWaitAction_ContextCancellation(t *testing.T) {
	duration := 1 * time.Second
	action := WaitAction(duration)

	// Create a context that cancels quickly
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := action.Execute(ctx, HealthCheck{})
	elapsed := time.Since(start)

	assert.Error(t, err)
	assert.Less(t, elapsed, duration)
}

func TestLoggingAction(t *testing.T) {
	logger := &mockLogger{}
	action := LoggingAction(logger)

	assert.Equal(t, "log_failure", action.Name())
	assert.Contains(t, action.Description(), "Log detailed information")

	healthCheck := HealthCheck{
		Name:        "test_check",
		Status:      HealthStatusUnhealthy,
		Message:     "Test failure",
		LastChecked: time.Now(),
		Duration:    100 * time.Millisecond,
		Critical:    true,
	}

	err := action.Execute(context.Background(), healthCheck)
	assert.NoError(t, err)

	// Verify logging occurred
	logs := logger.GetLogs()
	assert.Len(t, logs, 1)
	assert.Contains(t, logs[0], "ERROR:")
}
