package server

import (
	"sync"
	"testing"
	"time"
)

// TestSlidingWindowRateLimiterComprehensiveSecurity validates all aspects of the
// sliding window rate limiter implementation for task-145.
func TestSlidingWindowRateLimiterComprehensiveSecurity(t *testing.T) {
	t.Run("BurstAttackPrevention", func(t *testing.T) {
		limiter := NewSlidingWindowRateLimiter(5, 100*time.Millisecond)

		// Fill the window completely
		for i := 0; i < 5; i++ {
			if !limiter.IsAllowed() {
				t.Fatalf("Request %d should be allowed", i+1)
			}
		}

		// Attempt burst attack - should be rejected
		if limiter.IsAllowed() {
			t.Error("Burst attack should be rejected")
		}

		// Multiple rapid attempts should trigger exponential backoff
		for i := 0; i < 5; i++ {
			if limiter.IsAllowed() {
				t.Error("Rapid attack attempts should be rejected due to backoff")
			}
		}

		count, _, backoffUntil := limiter.GetViolationInfo()
		if count < 5 {
			t.Errorf("Expected multiple violations, got %d", count)
		}
		if time.Until(backoffUntil) <= 0 {
			t.Error("Should be in active backoff period")
		}
	})

	t.Run("WindowBoundaryBypassPrevention", func(t *testing.T) {
		limiter := NewSlidingWindowRateLimiter(3, 50*time.Millisecond)

		// Fill window
		for i := 0; i < 3; i++ {
			if !limiter.IsAllowed() {
				t.Fatalf("Request %d should be allowed", i+1)
			}
		}

		// Trigger violation and backoff
		if limiter.IsAllowed() {
			t.Error("Should trigger violation")
		}

		// Wait for window to expire but not backoff
		time.Sleep(60 * time.Millisecond) // Window expired

		// Should still be rejected due to backoff
		if limiter.IsAllowed() {
			t.Error("Should still be rejected due to backoff even after window expires")
		}

		// Verify we're still in backoff
		if !limiter.IsInBackoff() {
			t.Error("Should still be in backoff period")
		}
	})

	t.Run("ExponentialBackoffProgression", func(t *testing.T) {
		limiter := NewSlidingWindowRateLimiter(1, 50*time.Millisecond)

		// Trigger first violation
		limiter.IsAllowed() // allowed
		limiter.IsAllowed() // violation 1

		_, _, backoff1 := limiter.GetViolationInfo()

		// Trigger second violation
		limiter.IsAllowed() // violation 2

		count, _, backoff2 := limiter.GetViolationInfo()

		// Verify progression
		if count != 2 {
			t.Errorf("Expected 2 violations, got %d", count)
		}
		if !backoff2.After(backoff1) {
			t.Error("Backoff should increase exponentially")
		}

		// Verify backoff duration increases (approximately doubles)
		duration1 := time.Until(backoff1)
		duration2 := time.Until(backoff2)

		if duration2 < duration1 {
			t.Error("Second backoff should be longer than first")
		}
	})

	t.Run("MemoryExhaustionResistance", func(t *testing.T) {
		limiter := NewSlidingWindowRateLimiter(10, 100*time.Millisecond)

		// Attempt to exhaust memory with many timestamp entries
		for i := 0; i < 1000; i++ {
			limiter.IsAllowed()
			if i%100 == 0 {
				time.Sleep(1 * time.Millisecond)
			}
		}

		// Limiter should still be functional
		currentCount := limiter.GetCurrentCount()
		if currentCount > 10 {
			t.Errorf("Timestamp array should be bounded, got %d entries", currentCount)
		}

		violationCount, _, _ := limiter.GetViolationInfo()
		if violationCount < 100 {
			t.Errorf("Expected many violations from attack, got %d", violationCount)
		}
	})

	t.Run("ConcurrentSafetyUnderAttack", func(t *testing.T) {
		limiter := NewSlidingWindowRateLimiter(5, 100*time.Millisecond)

		var wg sync.WaitGroup
		var allowedCount, rejectedCount int64
		var mutex sync.Mutex

		// Simulate 20 concurrent attackers
		numAttackers := 20
		requestsPerAttacker := 50

		for i := 0; i < numAttackers; i++ {
			wg.Add(1)
			go func(attackerId int) {
				defer wg.Done()

				localAllowed := 0
				localRejected := 0

				for j := 0; j < requestsPerAttacker; j++ {
					if limiter.IsAllowed() {
						localAllowed++
					} else {
						localRejected++
					}
					time.Sleep(1 * time.Millisecond)
				}

				mutex.Lock()
				allowedCount += int64(localAllowed)
				rejectedCount += int64(localRejected)
				mutex.Unlock()
			}(i)
		}

		wg.Wait()

		totalRequests := int64(numAttackers * requestsPerAttacker)

		// Verify all requests were processed
		if allowedCount+rejectedCount != totalRequests {
			t.Errorf("Request count mismatch: %d + %d != %d",
				allowedCount, rejectedCount, totalRequests)
		}

		// With exponential backoff, vast majority should be rejected
		allowedPercent := float64(allowedCount) / float64(totalRequests) * 100
		if allowedPercent > 10 {
			t.Errorf("Too many requests allowed during concurrent attack: %.1f%%", allowedPercent)
		}

		// Verify violations were tracked
		violationCount, _, _ := limiter.GetViolationInfo()
		if violationCount < int(totalRequests/2) {
			t.Errorf("Expected significant violations from concurrent attack, got %d", violationCount)
		}

		t.Logf("Concurrent attack results: %d allowed (%.1f%%), %d rejected, %d violations",
			allowedCount, allowedPercent, rejectedCount, violationCount)
	})

	t.Run("BackoffMaximumCapEnforcement", func(t *testing.T) {
		limiter := NewSlidingWindowRateLimiter(1, 10*time.Millisecond)

		// Trigger many violations to reach maximum backoff
		for i := 0; i < 15; i++ {
			limiter.IsAllowed()
		}

		count, lastViolation, backoffUntil := limiter.GetViolationInfo()
		maxAllowedBackoff := lastViolation.Add(5 * time.Minute) // Max is 5 minutes

		if backoffUntil.After(maxAllowedBackoff) {
			t.Error("Backoff should not exceed 5 minute maximum")
		}

		if count < 10 {
			t.Errorf("Expected many violations, got %d", count)
		}

		t.Logf("After %d violations, backoff duration: %v",
			count, backoffUntil.Sub(lastViolation))
	})

	t.Run("ViolationResetAfterGoodBehavior", func(t *testing.T) {
		limiter := NewSlidingWindowRateLimiter(2, 50*time.Millisecond)

		// Trigger violations
		limiter.IsAllowed() // 1
		limiter.IsAllowed() // 2
		limiter.IsAllowed() // violation

		count, _, _ := limiter.GetViolationInfo()
		if count != 1 {
			t.Errorf("Expected 1 violation, got %d", count)
		}

		// Wait for backoff to expire (1 second base)
		time.Sleep(1100 * time.Millisecond)

		// Good behavior - make allowed request
		if !limiter.IsAllowed() {
			t.Error("Should be allowed after backoff expires")
		}

		// Wait for violation reset (2x window = 100ms)
		time.Sleep(150 * time.Millisecond)

		// Trigger reset check
		limiter.IsAllowed()

		// Violations should be reset
		count, _, _ = limiter.GetViolationInfo()
		if count != 0 {
			t.Errorf("Violations should reset after good behavior, got %d", count)
		}
	})

	t.Run("PerformanceUnderLoad", func(t *testing.T) {
		limiter := NewSlidingWindowRateLimiter(100, 1*time.Second)

		// Performance test - should handle high throughput efficiently
		start := time.Now()
		iterations := 10000

		for i := 0; i < iterations; i++ {
			limiter.IsAllowed()
		}

		duration := time.Since(start)
		opsPerSecond := float64(iterations) / duration.Seconds()

		// Should maintain high performance even under load
		if opsPerSecond < 100000 { // 100k ops/sec minimum
			t.Errorf("Performance degradation: only %.0f ops/sec", opsPerSecond)
		}

		t.Logf("Performance: %.0f operations/second", opsPerSecond)
	})
}

// TestSlidingWindowVulnerabilityMitigation specifically tests the vulnerability
// mentioned in task-145 - burst attacks at window boundaries.
func TestSlidingWindowVulnerabilityMitigation(t *testing.T) {
	t.Run("WindowBoundaryBurstAttackBlocked", func(t *testing.T) {
		// Test the specific vulnerability: attacker sends max messages,
		// waits for window reset, then repeats
		limiter := NewSlidingWindowRateLimiter(10, 100*time.Millisecond)

		// Phase 1: Fill the window
		allowedInPhase1 := 0
		for i := 0; i < 10; i++ {
			if limiter.IsAllowed() {
				allowedInPhase1++
			}
		}

		if allowedInPhase1 != 10 {
			t.Fatalf("Expected 10 allowed in phase 1, got %d", allowedInPhase1)
		}

		// Phase 2: Try to burst (should be rejected)
		burstAttempts := 0
		burstAllowed := 0
		for i := 0; i < 5; i++ {
			burstAttempts++
			if limiter.IsAllowed() {
				burstAllowed++
			}
		}

		if burstAllowed > 0 {
			t.Errorf("Burst attack should be completely blocked, but %d/%d succeeded",
				burstAllowed, burstAttempts)
		}

		// Phase 3: Wait for window to reset and try again (should still be blocked by backoff)
		time.Sleep(120 * time.Millisecond) // Window expired

		postWindowAllowed := 0
		for i := 0; i < 5; i++ {
			if limiter.IsAllowed() {
				postWindowAllowed++
			}
		}

		// Due to exponential backoff from previous violations,
		// requests should still be blocked
		if postWindowAllowed > 1 {
			t.Errorf("Window boundary attack should be mitigated by backoff, but %d requests allowed",
				postWindowAllowed)
		}

		violationCount, _, _ := limiter.GetViolationInfo()
		if violationCount < 5 {
			t.Errorf("Expected multiple violations from attack, got %d", violationCount)
		}

		t.Logf("Vulnerability test: %d violations recorded, backoff active", violationCount)
	})

	t.Run("RepeatedWindowBoundaryAttackIneffective", func(t *testing.T) {
		// Simulate an attacker trying the window boundary attack multiple times
		limiter := NewSlidingWindowRateLimiter(5, 50*time.Millisecond)

		totalAllowed := 0
		totalAttempts := 0

		// Simulate 10 attack cycles
		for cycle := 0; cycle < 10; cycle++ {
			// Fill window
			for i := 0; i < 5; i++ {
				totalAttempts++
				if limiter.IsAllowed() {
					totalAllowed++
				}
			}

			// Try burst attack
			for i := 0; i < 5; i++ {
				totalAttempts++
				if limiter.IsAllowed() {
					totalAllowed++
				}
			}

			// Wait for window reset
			time.Sleep(60 * time.Millisecond)

			// Attack after window reset
			for i := 0; i < 5; i++ {
				totalAttempts++
				if limiter.IsAllowed() {
					totalAllowed++
				}
			}

			// Brief pause before next cycle
			time.Sleep(10 * time.Millisecond)
		}

		successRate := float64(totalAllowed) / float64(totalAttempts) * 100

		// With exponential backoff, success rate should be very low
		if successRate > 20 {
			t.Errorf("Repeated boundary attacks too successful: %.1f%% success rate", successRate)
		}

		violationCount, _, backoffUntil := limiter.GetViolationInfo()
		if violationCount < 50 {
			t.Errorf("Expected many violations from repeated attacks, got %d", violationCount)
		}

		if time.Until(backoffUntil) <= 0 {
			t.Error("Should still be in backoff after sustained attack")
		}

		t.Logf("Repeated attack results: %.1f%% success rate, %d violations, backoff until %v",
			successRate, violationCount, backoffUntil.Format("15:04:05"))
	})
}
