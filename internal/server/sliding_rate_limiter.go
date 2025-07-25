package server

import (
	"sync"
	"time"
)

// ViolationInfo tracks rate limit violations for exponential backoff.
type ViolationInfo struct {
	Count         int       // Number of consecutive violations
	LastViolation time.Time // Timestamp of last violation
	BackoffUntil  time.Time // Time when backoff expires
}

// SlidingWindowRateLimiter implements a sliding window rate limiting algorithm
// with exponential backoff to prevent burst attacks at window boundaries.
type SlidingWindowRateLimiter struct {
	maxRequests    int            // Maximum requests allowed in the window
	windowDuration time.Duration  // Duration of the sliding window
	timestamps     []time.Time    // Timestamps of recent requests
	violations     *ViolationInfo // Tracks rate limit violations for exponential backoff
	mutex          sync.Mutex     // Protects all fields
	// Exponential backoff configuration
	baseBackoff       time.Duration // Base backoff duration (default: 1 second)
	maxBackoff        time.Duration // Maximum backoff duration (default: 5 minutes)
	backoffMultiplier float64       // Multiplier for exponential backoff (default: 2.0)
}

// NewSlidingWindowRateLimiter creates a new sliding window rate limiter with exponential backoff.
func NewSlidingWindowRateLimiter(maxRequests int, windowDuration time.Duration) *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		maxRequests:       maxRequests,
		windowDuration:    windowDuration,
		timestamps:        make([]time.Time, 0, maxRequests+10), // Pre-allocate with some buffer
		violations:        &ViolationInfo{},
		baseBackoff:       1 * time.Second, // Start with 1 second backoff
		maxBackoff:        5 * time.Minute, // Max 5 minutes backoff
		backoffMultiplier: 2.0,             // Double backoff each time
	}
}

// IsAllowed checks if a request is allowed under the current rate limit
// with exponential backoff for repeated violations.
// Returns true if the request is allowed, false if it should be rejected.
func (rl *SlidingWindowRateLimiter) IsAllowed() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()

	// Check if we're currently in a backoff period
	if now.Before(rl.violations.BackoffUntil) {
		// Still in backoff - record additional violation and extend backoff
		rl.recordViolation(now)

		return false
	}

	// Remove timestamps outside the current window
	rl.cleanOldTimestamps(now)

	// Check if we're under the limit
	if len(rl.timestamps) >= rl.maxRequests {
		// Rate limit exceeded - record violation and apply exponential backoff
		rl.recordViolation(now)

		return false
	}

	// Request allowed - reset violations if enough time has passed
	rl.resetViolationsIfExpired(now)

	// Add the current timestamp
	rl.timestamps = append(rl.timestamps, now)

	return true
}

// GetCurrentCount returns the current number of requests in the sliding window.
func (rl *SlidingWindowRateLimiter) GetCurrentCount() int {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	rl.cleanOldTimestamps(now)

	return len(rl.timestamps)
}

// GetTimeUntilReset returns the time until the oldest request expires from the window.
func (rl *SlidingWindowRateLimiter) GetTimeUntilReset() time.Duration {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	if len(rl.timestamps) == 0 {
		return 0
	}

	now := time.Now()
	rl.cleanOldTimestamps(now)

	if len(rl.timestamps) == 0 {
		return 0
	}

	oldestTimestamp := rl.timestamps[0]
	expireTime := oldestTimestamp.Add(rl.windowDuration)

	if expireTime.After(now) {
		return expireTime.Sub(now)
	}

	return 0
}

// Reset clears all timestamps and violations (useful for testing or manual reset).
func (rl *SlidingWindowRateLimiter) Reset() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	rl.timestamps = rl.timestamps[:0] // Clear slice but keep capacity
	rl.violations.Count = 0
	rl.violations.LastViolation = time.Time{}
	rl.violations.BackoffUntil = time.Time{}
}

// GetViolationInfo returns current violation information for monitoring.
func (rl *SlidingWindowRateLimiter) GetViolationInfo() (int, time.Time, time.Time) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	return rl.violations.Count, rl.violations.LastViolation, rl.violations.BackoffUntil
}

// IsInBackoff returns true if the rate limiter is currently in a backoff period.
func (rl *SlidingWindowRateLimiter) IsInBackoff() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	return time.Now().Before(rl.violations.BackoffUntil)
}

// recordViolation records a rate limit violation and calculates exponential backoff.
// This method must be called with the mutex already held.
func (rl *SlidingWindowRateLimiter) recordViolation(now time.Time) {
	// Increment violation count
	rl.violations.Count++
	rl.violations.LastViolation = now

	// Calculate exponential backoff duration
	backoffDuration := rl.baseBackoff
	for i := 1; i < rl.violations.Count; i++ {
		backoffDuration = time.Duration(float64(backoffDuration) * rl.backoffMultiplier)
		if backoffDuration > rl.maxBackoff {
			backoffDuration = rl.maxBackoff

			break
		}
	}

	// Set backoff expiry time
	rl.violations.BackoffUntil = now.Add(backoffDuration)
}

// resetViolationsIfExpired resets violation tracking if enough time has passed.
// This method must be called with the mutex already held.
func (rl *SlidingWindowRateLimiter) resetViolationsIfExpired(now time.Time) {
	// Reset violations if no violations for 2x the window duration
	// This provides forgiveness for well-behaved clients
	resetThreshold := rl.windowDuration * 2
	if rl.violations.Count > 0 && now.Sub(rl.violations.LastViolation) > resetThreshold {
		rl.violations.Count = 0
		rl.violations.LastViolation = time.Time{}
		rl.violations.BackoffUntil = time.Time{}
	}
}

// cleanOldTimestamps removes timestamps that fall outside the current window.
// This method must be called with the mutex already held.
func (rl *SlidingWindowRateLimiter) cleanOldTimestamps(now time.Time) {
	cutoff := now.Add(-rl.windowDuration)

	// Find the first timestamp that's still within the window
	validIndex := 0
	for i, timestamp := range rl.timestamps {
		if timestamp.After(cutoff) {
			validIndex = i

			break
		}
		validIndex = i + 1
	}

	// Remove expired timestamps by slicing
	if validIndex > 0 {
		// Move valid timestamps to the beginning
		copy(rl.timestamps, rl.timestamps[validIndex:])
		rl.timestamps = rl.timestamps[:len(rl.timestamps)-validIndex]
	}
}

// WebSocketRateLimiter is the interface for WebSocket-specific rate limiting.
type WebSocketRateLimiter interface {
	IsAllowed() bool
	Reset()
}
