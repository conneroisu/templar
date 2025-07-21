package server

import (
	"sync"
	"time"
)

// SlidingWindowRateLimiter implements a sliding window rate limiting algorithm
// that prevents the bypass vulnerabilities of fixed window rate limiting.
type SlidingWindowRateLimiter struct {
	maxRequests    int           // Maximum requests allowed in the window
	windowDuration time.Duration // Duration of the sliding window
	timestamps     []time.Time   // Timestamps of recent requests
	mutex          sync.Mutex    // Protects timestamps slice
}

// NewSlidingWindowRateLimiter creates a new sliding window rate limiter
func NewSlidingWindowRateLimiter(maxRequests int, windowDuration time.Duration) *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		maxRequests:    maxRequests,
		windowDuration: windowDuration,
		timestamps:     make([]time.Time, 0, maxRequests+10), // Pre-allocate with some buffer
	}
}

// IsAllowed checks if a request is allowed under the current rate limit
// Returns true if the request is allowed, false if it should be rejected
func (rl *SlidingWindowRateLimiter) IsAllowed() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()

	// Remove timestamps outside the current window
	rl.cleanOldTimestamps(now)

	// Check if we're under the limit
	if len(rl.timestamps) >= rl.maxRequests {
		return false
	}

	// Add the current timestamp
	rl.timestamps = append(rl.timestamps, now)
	return true
}

// GetCurrentCount returns the current number of requests in the sliding window
func (rl *SlidingWindowRateLimiter) GetCurrentCount() int {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	rl.cleanOldTimestamps(now)
	return len(rl.timestamps)
}

// GetTimeUntilReset returns the time until the oldest request expires from the window
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

// Reset clears all timestamps (useful for testing or manual reset)
func (rl *SlidingWindowRateLimiter) Reset() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	rl.timestamps = rl.timestamps[:0] // Clear slice but keep capacity
}

// cleanOldTimestamps removes timestamps that fall outside the current window
// This method must be called with the mutex already held
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

// WebSocketRateLimiter is the interface for WebSocket-specific rate limiting
type WebSocketRateLimiter interface {
	IsAllowed() bool
	Reset()
}
