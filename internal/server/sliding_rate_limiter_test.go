package server

import (
	"sync"
	"testing"
	"time"
)

func TestSlidingWindowRateLimiter_BasicFunctionality(t *testing.T) {
	limiter := NewSlidingWindowRateLimiter(5, time.Minute)

	// Should allow first 5 requests
	for i := range 5 {
		if !limiter.IsAllowed() {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Should reject the 6th request
	if limiter.IsAllowed() {
		t.Error("6th request should be rejected")
	}

	// Current count should be 5
	if count := limiter.GetCurrentCount(); count != 5 {
		t.Errorf("Expected count 5, got %d", count)
	}
}

func TestSlidingWindowRateLimiter_WindowSliding(t *testing.T) {
	limiter := NewSlidingWindowRateLimiter(3, 100*time.Millisecond)

	// Use all 3 requests
	for i := range 3 {
		if !limiter.IsAllowed() {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Should be rejected
	if limiter.IsAllowed() {
		t.Error("4th request should be rejected")
	}

	// Wait for window to slide AND backoff to expire (1 second base backoff + window slide)
	time.Sleep(1200 * time.Millisecond)

	// Should be allowed again after both window slide and backoff expiry
	for i := range 3 {
		if !limiter.IsAllowed() {
			t.Errorf("Request %d after window slide and backoff expiry should be allowed", i+1)
		}
	}
}

func TestSlidingWindowRateLimiter_PreventsBypassAttack(t *testing.T) {
	limiter := NewSlidingWindowRateLimiter(5, 100*time.Millisecond)

	// Fill the bucket
	for i := range 5 {
		if !limiter.IsAllowed() {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Wait almost until window expiry (but not quite)
	time.Sleep(90 * time.Millisecond)

	// Try to send burst right before expiry - should still be rejected
	if limiter.IsAllowed() {
		t.Error("Burst attack right before window expiry should be prevented")
	}

	// Wait for backoff to expire (1 second) plus full window expiry
	time.Sleep(1200 * time.Millisecond)

	// Now should be allowed after both backoff and window expiry
	if !limiter.IsAllowed() {
		t.Error("Request after full window and backoff expiry should be allowed")
	}
}

func TestSlidingWindowRateLimiter_ConcurrentAccess(t *testing.T) {
	limiter := NewSlidingWindowRateLimiter(100, time.Minute)
	var wg sync.WaitGroup
	var allowedCount int64
	var rejectedCount int64
	var mutex sync.Mutex

	// Simulate 200 concurrent requests
	for range 200 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			if limiter.IsAllowed() {
				mutex.Lock()
				allowedCount++
				mutex.Unlock()
			} else {
				mutex.Lock()
				rejectedCount++
				mutex.Unlock()
			}
		}()
	}

	wg.Wait()

	// Should allow exactly 100 requests
	if allowedCount != 100 {
		t.Errorf("Expected 100 allowed requests, got %d", allowedCount)
	}

	// Should reject exactly 100 requests
	if rejectedCount != 100 {
		t.Errorf("Expected 100 rejected requests, got %d", rejectedCount)
	}
}

func TestSlidingWindowRateLimiter_Reset(t *testing.T) {
	limiter := NewSlidingWindowRateLimiter(3, time.Minute)

	// Use all requests
	for range 3 {
		limiter.IsAllowed()
	}

	// Should be rejected
	if limiter.IsAllowed() {
		t.Error("Request should be rejected before reset")
	}

	// Reset
	limiter.Reset()

	// Should be allowed again
	if !limiter.IsAllowed() {
		t.Error("Request should be allowed after reset")
	}

	// Count should be 1
	if count := limiter.GetCurrentCount(); count != 1 {
		t.Errorf("Expected count 1 after reset, got %d", count)
	}
}

func TestSlidingWindowRateLimiter_TimeUntilReset(t *testing.T) {
	limiter := NewSlidingWindowRateLimiter(1, 100*time.Millisecond)

	// No requests yet - should be 0
	if duration := limiter.GetTimeUntilReset(); duration != 0 {
		t.Errorf("Expected 0 time until reset with no requests, got %v", duration)
	}

	// Make a request
	limiter.IsAllowed()

	// Should have some time until reset
	duration := limiter.GetTimeUntilReset()
	if duration <= 0 || duration > 100*time.Millisecond {
		t.Errorf("Expected reasonable time until reset, got %v", duration)
	}

	// Wait for expiry
	time.Sleep(110 * time.Millisecond)

	// Should be 0 again
	if duration := limiter.GetTimeUntilReset(); duration != 0 {
		t.Errorf("Expected 0 time until reset after expiry, got %v", duration)
	}
}

// Benchmark tests to ensure performance.
func BenchmarkSlidingWindowRateLimiter_IsAllowed(b *testing.B) {
	limiter := NewSlidingWindowRateLimiter(1000, time.Minute)

	b.ResetTimer()
	for range b.N {
		limiter.IsAllowed()
	}
}

func BenchmarkSlidingWindowRateLimiter_Concurrent(b *testing.B) {
	limiter := NewSlidingWindowRateLimiter(1000, time.Minute)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			limiter.IsAllowed()
		}
	})
}

// TestWebSocketRateLimiterInterface ensures our sliding window rate limiter implements the interface.
func TestWebSocketRateLimiterInterface(t *testing.T) {
	var _ WebSocketRateLimiter = (*SlidingWindowRateLimiter)(nil)
}
