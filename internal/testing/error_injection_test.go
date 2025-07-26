package testing

import (
	"errors"
	"testing"
	"time"
)

func TestErrorInjector_BasicInjection(t *testing.T) {
	injector := NewErrorInjector()

	// Test basic error injection
	testErr := errors.New("test error")
	injector.InjectError("test.operation", testErr)

	// Should fail
	if err := injector.ShouldFail("test.operation"); err == nil {
		t.Error("Expected error injection to trigger")
	} else if err.Error() != "test error" {
		t.Errorf("Expected 'test error', got '%s'", err.Error())
	}

	// Non-configured operation should not fail
	if err := injector.ShouldFail("other.operation"); err != nil {
		t.Errorf("Expected no error for unconfigured operation, got: %v", err)
	}
}

func TestErrorInjector_SingleInjection(t *testing.T) {
	injector := NewErrorInjector()

	// Configure single injection
	testErr := errors.New("single error")
	injector.InjectErrorOnce("single.operation", testErr)

	// First call should fail
	if err := injector.ShouldFail("single.operation"); err == nil {
		t.Error("Expected first call to fail")
	}

	// Second call should not fail
	if err := injector.ShouldFail("single.operation"); err != nil {
		t.Errorf("Expected second call to succeed, got: %v", err)
	}
}

func TestErrorInjector_CountedInjection(t *testing.T) {
	injector := NewErrorInjector()

	// Configure 3 injections
	testErr := errors.New("counted error")
	injector.InjectErrorCount("counted.operation", testErr, 3)

	// First 3 calls should fail
	for i := 0; i < 3; i++ {
		if err := injector.ShouldFail("counted.operation"); err == nil {
			t.Errorf("Expected call %d to fail", i+1)
		}
	}

	// Fourth call should not fail
	if err := injector.ShouldFail("counted.operation"); err != nil {
		t.Errorf("Expected call 4 to succeed, got: %v", err)
	}
}

func TestErrorInjector_WithDelay(t *testing.T) {
	injector := NewErrorInjector()

	// Configure injection with delay
	testErr := errors.New("delayed error")
	delay := 50 * time.Millisecond
	injector.InjectErrorWithDelay("delayed.operation", testErr, delay)

	start := time.Now()
	err := injector.ShouldFail("delayed.operation")
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Expected delayed error injection to trigger")
	}

	if elapsed < delay {
		t.Errorf("Expected delay of at least %v, got %v", delay, elapsed)
	}
}

func TestErrorInjector_Probability(t *testing.T) {
	injector := NewErrorInjector()

	// Configure low probability injection (deterministic for testing)
	testErr := errors.New("probabilistic error")
	target := injector.InjectError("prob.operation", testErr)
	target.WithProbability(0.1) // 10% chance

	failures := 0
	attempts := 100

	for i := 0; i < attempts; i++ {
		if err := injector.ShouldFail("prob.operation"); err != nil {
			failures++
		}
	}

	// With deterministic probability based on count, we should get ~10% failures
	expectedFailures := 10 // 10% of 100
	tolerance := 3

	if failures < expectedFailures-tolerance || failures > expectedFailures+tolerance {
		t.Errorf("Expected ~%d failures (±%d), got %d", expectedFailures, tolerance, failures)
	}
}

func TestErrorInjector_EnableDisable(t *testing.T) {
	injector := NewErrorInjector()

	testErr := errors.New("test error")
	injector.InjectError("test.operation", testErr)

	// Should fail when enabled
	if err := injector.ShouldFail("test.operation"); err == nil {
		t.Error("Expected error when injection enabled")
	}

	// Disable injection
	injector.Disable()
	if err := injector.ShouldFail("test.operation"); err != nil {
		t.Errorf("Expected no error when injection disabled, got: %v", err)
	}

	// Re-enable injection
	injector.Enable()
	if err := injector.ShouldFail("test.operation"); err == nil {
		t.Error("Expected error when injection re-enabled")
	}
}

func TestErrorInjector_TargetManagement(t *testing.T) {
	injector := NewErrorInjector()

	testErr := errors.New("test error")
	injector.InjectError("test.operation", testErr)

	// Check target exists
	target, exists := injector.GetTarget("test.operation")
	if !exists {
		t.Error("Expected target to exist")
	}
	if target.Error.Error() != "test error" {
		t.Errorf("Expected 'test error', got '%s'", target.Error.Error())
	}

	// Remove target
	injector.RemoveTarget("test.operation")

	// Should not fail after removal
	if err := injector.ShouldFail("test.operation"); err != nil {
		t.Errorf("Expected no error after target removal, got: %v", err)
	}

	// Target should not exist
	_, exists = injector.GetTarget("test.operation")
	if exists {
		t.Error("Expected target to not exist after removal")
	}
}

func TestErrorInjector_Stats(t *testing.T) {
	injector := NewErrorInjector()

	// Add some targets
	injector.InjectError("op1", errors.New("error1"))
	injector.InjectError("op2", errors.New("error2"))
	injector.InjectErrorOnce("op3", errors.New("error3"))

	// Trigger some injections
	_ = injector.ShouldFail("op1")
	_ = injector.ShouldFail("op1")
	_ = injector.ShouldFail("op3") // This should exhaust the single injection

	stats := injector.GetStats()

	if stats.TotalTargets != 3 {
		t.Errorf("Expected 3 total targets, got %d", stats.TotalTargets)
	}

	if stats.ActiveTargets != 2 { // op1 and op2 still active, op3 exhausted
		t.Errorf("Expected 2 active targets, got %d", stats.ActiveTargets)
	}

	if stats.TotalInjections != 3 {
		t.Errorf("Expected 3 total injections, got %d", stats.TotalInjections)
	}
}

func TestScenarioManager_BasicScenario(t *testing.T) {
	injector := NewErrorInjector()
	manager := NewScenarioManager(injector)

	// Register a test scenario
	scenario := &ErrorScenario{
		Name:        "test_scenario",
		Description: "Test scenario for unit testing",
		Steps: []ErrorStep{
			{
				Operation: "step1",
				Error:     errors.New("step1 error"),
				Count:     2,
			},
			{
				Operation: "step2",
				Error:     errors.New("step2 error"),
				Count:     1,
			},
		},
	}

	manager.RegisterScenario(scenario)

	// Execute scenario
	err := manager.ExecuteScenario("test_scenario")
	if err != nil {
		t.Fatalf("Failed to execute scenario: %v", err)
	}

	// Verify step1 fails twice
	for i := 0; i < 2; i++ {
		if err := injector.ShouldFail("step1"); err == nil {
			t.Errorf("Expected step1 to fail on attempt %d", i+1)
		}
	}

	// Third attempt should succeed
	if err := injector.ShouldFail("step1"); err != nil {
		t.Errorf("Expected step1 to succeed on attempt 3, got: %v", err)
	}

	// Step2 should fail once
	if err := injector.ShouldFail("step2"); err == nil {
		t.Error("Expected step2 to fail")
	}

	// Second attempt should succeed
	if err := injector.ShouldFail("step2"); err != nil {
		t.Errorf("Expected step2 to succeed on second attempt, got: %v", err)
	}
}

func TestScenarioManager_PredefinedScenarios(t *testing.T) {
	injector := NewErrorInjector()
	manager := NewScenarioManager(injector)

	// Test build failure scenario
	buildScenario := CreateBuildFailureScenario()
	manager.RegisterScenario(buildScenario)

	err := manager.ExecuteScenario("build_failure")
	if err != nil {
		t.Fatalf("Failed to execute build failure scenario: %v", err)
	}

	// Verify some injections work
	if err := injector.ShouldFail("file.read"); err == nil {
		// With probability 0.3, this might not fail, which is okay
		t.Logf("file.read did not fail (probabilistic)")
	}

	if err := injector.ShouldFail("exec.command"); err == nil {
		t.Error("exec.command should fail (deterministic)")
	}

	// Test network failure scenario
	networkScenario := CreateNetworkFailureScenario()
	manager.RegisterScenario(networkScenario)

	// Stop build scenario first
	_ = manager.StopScenario("build_failure")

	err = manager.ExecuteScenario("network_failure")
	if err != nil {
		t.Fatalf("Failed to execute network failure scenario: %v", err)
	}

	// Verify network injections
	if err := injector.ShouldFail("http.request"); err == nil {
		t.Error("http.request should fail")
	}
}

func TestScenarioManager_ListScenarios(t *testing.T) {
	injector := NewErrorInjector()
	manager := NewScenarioManager(injector)

	// Register scenarios
	manager.RegisterScenario(CreateBuildFailureScenario())
	manager.RegisterScenario(CreateNetworkFailureScenario())
	manager.RegisterScenario(CreateResourceExhaustionScenario())

	scenarios := manager.ListScenarios()

	if len(scenarios) != 3 {
		t.Errorf("Expected 3 scenarios, got %d", len(scenarios))
	}

	expectedScenarios := map[string]bool{
		"build_failure":       false,
		"network_failure":     false,
		"resource_exhaustion": false,
	}

	for _, scenario := range scenarios {
		if _, exists := expectedScenarios[scenario]; exists {
			expectedScenarios[scenario] = true
		} else {
			t.Errorf("Unexpected scenario: %s", scenario)
		}
	}

	for scenario, found := range expectedScenarios {
		if !found {
			t.Errorf("Expected scenario not found: %s", scenario)
		}
	}
}

// Benchmark tests for error injection performance
func BenchmarkErrorInjector_ShouldFail(b *testing.B) {
	injector := NewErrorInjector()
	injector.InjectError("bench.operation", errors.New("bench error"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		injector.ShouldFail("bench.operation")
	}
}

func BenchmarkErrorInjector_NoInjection(b *testing.B) {
	injector := NewErrorInjector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		injector.ShouldFail("bench.operation")
	}
}

func BenchmarkErrorInjector_ConcurrentAccess(b *testing.B) {
	injector := NewErrorInjector()
	injector.InjectError("concurrent.operation", errors.New("concurrent error"))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			injector.ShouldFail("concurrent.operation")
		}
	})
}
