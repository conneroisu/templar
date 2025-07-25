package server

import (
	"sync"
	"testing"
	"time"
)

// TestExponentialBackoffSecurity tests the security properties of exponential backoff
func TestExponentialBackoffSecurity(t *testing.T) {
	t.Run("PreventsBurstAttackWithExponentialBackoff", func(t *testing.T) {
		limiter := NewSlidingWindowRateLimiter(3, 100*time.Millisecond)

		// First burst - fill the window
		for i := 0; i < 3; i++ {
			if !limiter.IsAllowed() {
				t.Fatalf("Request %d should be allowed", i+1)
			}
		}

		// Try to exceed limit - should trigger violation and backoff
		if limiter.IsAllowed() {
			t.Error("4th request should be rejected and trigger backoff")
		}

		// Verify we're in backoff
		if !limiter.IsInBackoff() {
			t.Error("Should be in backoff after violation")
		}

		// Try another request immediately - should still be rejected due to backoff
		if limiter.IsAllowed() {
			t.Error("Request during backoff should be rejected")
		}

		// Second violation should increase backoff duration
		violationCount, _, backoffUntil1 := limiter.GetViolationInfo()
		if violationCount != 2 {
			t.Errorf("Expected 2 violations, got %d", violationCount)
		}

		// Wait a bit and try again - should still be in backoff
		time.Sleep(500 * time.Millisecond)
		if limiter.IsAllowed() {
			t.Error("Request should still be rejected due to longer backoff")
		}

		// Third violation should have even longer backoff
		violationCount, _, backoffUntil2 := limiter.GetViolationInfo()
		if violationCount != 3 {
			t.Errorf("Expected 3 violations, got %d", violationCount)
		}

		// Backoff should increase exponentially
		if !backoffUntil2.After(backoffUntil1) {
			t.Error("Backoff duration should increase with more violations")
		}
	})

	t.Run("BackoffResetsAfterGoodBehavior", func(t *testing.T) {
		limiter := NewSlidingWindowRateLimiter(2, 100*time.Millisecond)

		// Trigger violation
		limiter.IsAllowed() // 1
		limiter.IsAllowed() // 2
		limiter.IsAllowed() // violation

		violationCount, _, _ := limiter.GetViolationInfo()
		if violationCount != 1 {
			t.Errorf("Expected 1 violation, got %d", violationCount)
		}

		// First, wait for the backoff period to expire (1 second base backoff)
		time.Sleep(1100 * time.Millisecond) // Wait for backoff to expire

		// Should allow requests now that backoff expired
		if !limiter.IsAllowed() {
			t.Error("Request should be allowed after backoff expires")
		}

		// Now wait for violations to reset (2x window duration from last violation)
		time.Sleep(250 * time.Millisecond) // 2.5x the window duration

		// Make another request to trigger violation reset check
		limiter.IsAllowed()

		// Check if violations were reset
		violationCount, _, _ = limiter.GetViolationInfo()
		if violationCount != 0 {
			t.Errorf("Violations should be reset after good behavior, got %d", violationCount)
		}
	})

	t.Run("MaxBackoffCapIsEnforced", func(t *testing.T) {
		limiter := NewSlidingWindowRateLimiter(1, 50*time.Millisecond)

		// Trigger many violations to test max backoff
		for i := 0; i < 10; i++ {
			limiter.IsAllowed() // Should be rejected after first one
		}

		_, _, backoffUntil := limiter.GetViolationInfo()
		maxBackoffTime := time.Now().Add(5 * time.Minute) // Max backoff is 5 minutes

		if backoffUntil.After(maxBackoffTime) {
			t.Error("Backoff should not exceed maximum duration")
		}
	})
}

// TestWindowBoundaryAttackPrevention tests that the enhanced rate limiter prevents
// sophisticated attacks that try to exploit window boundaries
func TestWindowBoundaryAttackPrevention(t *testing.T) {
	t.Run("PreventsBurstAtWindowReset", func(t *testing.T) {
		limiter := NewSlidingWindowRateLimiter(5, 100*time.Millisecond)

		// First burst - use up the window
		for i := 0; i < 5; i++ {
			if !limiter.IsAllowed() {
				t.Fatalf("Request %d should be allowed", i+1)
			}
		}

		// Try to burst immediately - should be rejected
		if limiter.IsAllowed() {
			t.Error("Burst attack should be rejected")
		}

		// Wait for window to partially reset (but not full backoff)
		time.Sleep(60 * time.Millisecond)

		// Should still be in backoff from the violation
		if limiter.IsAllowed() {
			t.Error("Request should still be rejected during backoff")
		}

		// Wait for full window reset but still in backoff
		time.Sleep(50 * time.Millisecond) // Total 110ms, window expired but backoff may still be active

		// Try again - might still be in backoff depending on timing
		allowed := limiter.IsAllowed()
		inBackoff := limiter.IsInBackoff()

		if inBackoff && allowed {
			t.Error("If in backoff, request should be rejected")
		}
		if !inBackoff && !allowed {
			t.Error("If not in backoff and window reset, request should be allowed")
		}
	})

	t.Run("PreventsSustainedAttack", func(t *testing.T) {
		limiter := NewSlidingWindowRateLimiter(3, 100*time.Millisecond)

		attackAttempts := 0
		successfulRequests := 0

		// Simulate sustained attack over 2 seconds
		start := time.Now()
		for time.Since(start) < 2*time.Second {
			attackAttempts++
			if limiter.IsAllowed() {
				successfulRequests++
			}
			time.Sleep(10 * time.Millisecond) // Attack every 10ms
		}

		// With exponential backoff, the attacker should get very few successful requests
		// Even in 2 seconds, with backoff increasing exponentially, they should get << 30 requests
		// (without backoff, they could get ~60 requests in 2 seconds with 100ms window and 3 req/window)
		if successfulRequests > 20 {
			t.Errorf("Sustained attack succeeded too much: %d successful out of %d attempts",
				successfulRequests, attackAttempts)
		}

		violationCount, _, _ := limiter.GetViolationInfo()
		if violationCount < 5 {
			t.Errorf("Expected multiple violations from sustained attack, got %d", violationCount)
		}

		t.Logf("Attack results: %d successful requests out of %d attempts (%d violations)",
			successfulRequests, attackAttempts, violationCount)
	})
}

// TestConcurrentBackoffSafety tests that the backoff mechanism is thread-safe
func TestConcurrentBackoffSafety(t *testing.T) {
	limiter := NewSlidingWindowRateLimiter(5, 100*time.Millisecond)
	var wg sync.WaitGroup
	var allowedCount int64
	var rejectedCount int64
	var mutex sync.Mutex

	// Simulate multiple concurrent clients trying to overwhelm the rate limiter
	numGoroutines := 10
	requestsPerGoroutine := 20

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < requestsPerGoroutine; j++ {
				if limiter.IsAllowed() {
					mutex.Lock()
					allowedCount++
					mutex.Unlock()
				} else {
					mutex.Lock()
					rejectedCount++
					mutex.Unlock()
				}
				time.Sleep(5 * time.Millisecond) // Small delay between requests
			}
		}(i)
	}

	wg.Wait()

	totalRequests := int64(numGoroutines * requestsPerGoroutine)
	if allowedCount+rejectedCount != totalRequests {
		t.Errorf("Total requests mismatch: %d allowed + %d rejected != %d total",
			allowedCount, rejectedCount, totalRequests)
	}

	// With exponential backoff, most requests should be rejected
	if allowedCount > totalRequests/2 {
		t.Errorf("Too many requests allowed during concurrent attack: %d/%d",
			allowedCount, totalRequests)
	}

	// Verify rate limiter is still functional after concurrent access
	violationCount, _, _ := limiter.GetViolationInfo()
	if violationCount < 1 {
		t.Error("Expected violations from concurrent attack")
	}

	t.Logf("Concurrent attack results: %d allowed, %d rejected, %d violations",
		allowedCount, rejectedCount, violationCount)
}

// TestBackoffTimingAccuracy tests that backoff timing is accurate
func TestBackoffTimingAccuracy(t *testing.T) {
	limiter := NewSlidingWindowRateLimiter(1, 50*time.Millisecond)

	// Trigger first violation
	limiter.IsAllowed() // allowed
	limiter.IsAllowed() // violation, should trigger 1 second backoff

	violationCount, lastViolation, backoffUntil := limiter.GetViolationInfo()
	if violationCount != 1 {
		t.Errorf("Expected 1 violation, got %d", violationCount)
	}

	// Check backoff duration is approximately 1 second (base backoff)
	expectedBackoff := lastViolation.Add(1 * time.Second)
	timeDiff := backoffUntil.Sub(expectedBackoff)
	if timeDiff < -10*time.Millisecond || timeDiff > 10*time.Millisecond {
		t.Errorf("Backoff timing inaccurate: expected ~1s, got %v", backoffUntil.Sub(lastViolation))
	}

	// Wait for backoff to expire
	sleepDuration := time.Until(backoffUntil) + 10*time.Millisecond
	if sleepDuration > 0 {
		time.Sleep(sleepDuration)
	}

	// Should be allowed after backoff expires
	if !limiter.IsAllowed() {
		t.Error("Request should be allowed after backoff expires")
	}
}

// TestViolationInfoConsistency tests that violation tracking is consistent
func TestViolationInfoConsistency(t *testing.T) {
	limiter := NewSlidingWindowRateLimiter(2, 100*time.Millisecond)

	// No violations initially
	count, lastViolation, backoffUntil := limiter.GetViolationInfo()
	if count != 0 || !lastViolation.IsZero() || !backoffUntil.IsZero() {
		t.Error("Initial violation info should be zero values")
	}

	// Trigger first violation
	limiter.IsAllowed() // 1
	limiter.IsAllowed() // 2
	violationTime := time.Now()
	limiter.IsAllowed() // violation

	count, lastViolation, backoffUntil = limiter.GetViolationInfo()
	if count != 1 {
		t.Errorf("Expected 1 violation, got %d", count)
	}
	if lastViolation.Before(violationTime) || lastViolation.After(time.Now()) {
		t.Error("Last violation time should be recent")
	}
	if backoffUntil.Before(time.Now()) {
		t.Error("Backoff should be in the future")
	}

	// Reset should clear violation info
	limiter.Reset()
	count, lastViolation, backoffUntil = limiter.GetViolationInfo()
	if count != 0 || !lastViolation.IsZero() || !backoffUntil.IsZero() {
		t.Error("Reset should clear all violation info")
	}
}
