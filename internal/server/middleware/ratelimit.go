package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// RateLimit represents a rate limiter configuration.
type RateLimit struct {
	RequestsPerMinute int
	BurstLimit        int
}

// RateLimiter implements a token bucket rate limiter per IP address.
type RateLimiter struct {
	config        RateLimit
	buckets       map[string]*tokenBucket
	mutex         sync.RWMutex
	cleanupTicker *time.Ticker
}

// tokenBucket represents a token bucket for rate limiting.
type tokenBucket struct {
	tokens     int
	maxTokens  int
	refillRate time.Duration
	lastRefill time.Time
	mutex      sync.Mutex
}

// NewRateLimiter creates a new rate limiter with the given configuration.
func NewRateLimiter(config RateLimit) *RateLimiter {
	rl := &RateLimiter{
		config:  config,
		buckets: make(map[string]*tokenBucket),
	}

	// Start cleanup goroutine to remove old buckets
	rl.cleanupTicker = time.NewTicker(5 * time.Minute)
	go rl.cleanup()

	return rl
}

// RateLimit returns a middleware that implements rate limiting per IP.
func (rl *RateLimiter) RateLimit() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract client IP
			ip := getClientIP(r)

			// Check if request is allowed
			if !rl.allow(ip) {
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// allow checks if a request from the given IP is allowed.
func (rl *RateLimiter) allow(ip string) bool {
	rl.mutex.RLock()
	bucket, exists := rl.buckets[ip]
	rl.mutex.RUnlock()

	if !exists {
		// Create new bucket for this IP
		bucket = &tokenBucket{
			tokens:     rl.config.BurstLimit,
			maxTokens:  rl.config.BurstLimit,
			refillRate: time.Minute / time.Duration(rl.config.RequestsPerMinute),
			lastRefill: time.Now(),
		}

		rl.mutex.Lock()
		rl.buckets[ip] = bucket
		rl.mutex.Unlock()
	}

	return bucket.consume()
}

// consume attempts to consume a token from the bucket.
func (tb *tokenBucket) consume() bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)
	tokensToAdd := int(elapsed / tb.refillRate)

	if tokensToAdd > 0 {
		tb.tokens = min(tb.maxTokens, tb.tokens+tokensToAdd)
		tb.lastRefill = now
	}

	// Check if we have tokens available
	if tb.tokens > 0 {
		tb.tokens--

		return true
	}

	return false
}

// cleanup removes old buckets to prevent memory leaks.
func (rl *RateLimiter) cleanup() {
	for range rl.cleanupTicker.C {
		rl.mutex.Lock()
		cutoff := time.Now().Add(-10 * time.Minute)

		for ip, bucket := range rl.buckets {
			bucket.mutex.Lock()
			if bucket.lastRefill.Before(cutoff) {
				delete(rl.buckets, ip)
			}
			bucket.mutex.Unlock()
		}
		rl.mutex.Unlock()
	}
}

// Stop stops the rate limiter and cleanup goroutine.
func (rl *RateLimiter) Stop() {
	if rl.cleanupTicker != nil {
		rl.cleanupTicker.Stop()
	}
}

// getClientIP extracts the client IP address from the request.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP in the chain
		if firstComma := len(xff); firstComma > 0 {
			if commaIndex := 0; commaIndex < len(xff) {
				for i, char := range xff {
					if char == ',' {
						commaIndex = i

						break
					}
				}
				if commaIndex > 0 {
					xff = xff[:commaIndex]
				}
			}
		}
		if ip := net.ParseIP(xff); ip != nil {
			return xff
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		if ip := net.ParseIP(xri); ip != nil {
			return xri
		}
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}
