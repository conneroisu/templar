package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/conneroisu/templar/internal/errors"
	"github.com/conneroisu/templar/internal/logging"
)

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	buckets     map[string]*TokenBucket
	bucketMutex sync.RWMutex
	config      *RateLimitConfig
	logger      logging.Logger
	cleaner     *time.Ticker
	stopCleaner chan struct{}
}

// TokenBucket represents a token bucket for rate limiting
type TokenBucket struct {
	tokens       int
	capacity     int
	refillRate   int           // tokens per minute
	lastRefill   time.Time
	mutex        sync.Mutex
	lastAccess   time.Time
}

// RateLimitResult represents the result of a rate limit check
type RateLimitResult struct {
	Allowed       bool
	Remaining     int
	RetryAfter    time.Duration
	ResetTime     time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *RateLimitConfig, logger logging.Logger) *RateLimiter {
	if config == nil {
		config = &RateLimitConfig{
			RequestsPerMinute: 1000,
			BurstSize:         50,
			WindowSize:        time.Minute,
			Enabled:           true,
		}
	}

	rl := &RateLimiter{
		buckets:     make(map[string]*TokenBucket),
		config:      config,
		logger:      logger,
		stopCleaner: make(chan struct{}),
	}

	// Start cleanup goroutine to remove expired buckets
	rl.cleaner = time.NewTicker(5 * time.Minute)
	go rl.cleanupExpiredBuckets()

	return rl
}

// Check checks if a request is allowed for the given key (usually IP address)
func (rl *RateLimiter) Check(key string) RateLimitResult {
	if !rl.config.Enabled {
		return RateLimitResult{
			Allowed:   true,
			Remaining: rl.config.BurstSize,
		}
	}

	bucket := rl.getBucket(key)
	return bucket.consume()
}

// getBucket gets or creates a token bucket for the given key
func (rl *RateLimiter) getBucket(key string) *TokenBucket {
	rl.bucketMutex.RLock()
	bucket, exists := rl.buckets[key]
	rl.bucketMutex.RUnlock()

	if exists {
		bucket.mutex.Lock()
		bucket.lastAccess = time.Now()
		bucket.mutex.Unlock()
		return bucket
	}

	// Create new bucket
	rl.bucketMutex.Lock()
	defer rl.bucketMutex.Unlock()

	// Double-check after acquiring write lock
	if bucket, exists := rl.buckets[key]; exists {
		bucket.mutex.Lock()
		bucket.lastAccess = time.Now()
		bucket.mutex.Unlock()
		return bucket
	}

	bucket = &TokenBucket{
		tokens:     rl.config.BurstSize,
		capacity:   rl.config.BurstSize,
		refillRate: rl.config.RequestsPerMinute,
		lastRefill: time.Now(),
		lastAccess: time.Now(),
	}

	rl.buckets[key] = bucket
	return bucket
}

// consume attempts to consume a token from the bucket
func (tb *TokenBucket) consume() RateLimitResult {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	now := time.Now()
	tb.refill(now)

	if tb.tokens > 0 {
		tb.tokens--
		return RateLimitResult{
			Allowed:   true,
			Remaining: tb.tokens,
			ResetTime: now.Add(time.Minute),
		}
	}

	// Calculate retry after
	tokensNeeded := 1
	timePerToken := time.Minute / time.Duration(tb.refillRate)
	retryAfter := timePerToken * time.Duration(tokensNeeded)

	return RateLimitResult{
		Allowed:    false,
		Remaining:  0,
		RetryAfter: retryAfter,
		ResetTime:  now.Add(retryAfter),
	}
}

// refill adds tokens to the bucket based on elapsed time
func (tb *TokenBucket) refill(now time.Time) {
	elapsed := now.Sub(tb.lastRefill)
	if elapsed < time.Second {
		return // Don't refill too frequently
	}

	// Calculate tokens to add based on elapsed time
	tokensToAdd := int(elapsed.Minutes() * float64(tb.refillRate))
	if tokensToAdd > 0 {
		tb.tokens += tokensToAdd
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}
		tb.lastRefill = now
	}
}

// cleanupExpiredBuckets removes buckets that haven't been accessed recently
func (rl *RateLimiter) cleanupExpiredBuckets() {
	defer close(rl.stopCleaner)

	for {
		select {
		case <-rl.cleaner.C:
			rl.performCleanup()
		case <-rl.stopCleaner:
			rl.cleaner.Stop()
			return
		}
	}
}

// performCleanup removes expired buckets
func (rl *RateLimiter) performCleanup() {
	rl.bucketMutex.Lock()
	defer rl.bucketMutex.Unlock()

	now := time.Now()
	expiry := 10 * time.Minute // Remove buckets not accessed for 10 minutes

	for key, bucket := range rl.buckets {
		bucket.mutex.Lock()
		if now.Sub(bucket.lastAccess) > expiry {
			delete(rl.buckets, key)
		}
		bucket.mutex.Unlock()
	}
}

// Stop stops the rate limiter and cleanup goroutine
func (rl *RateLimiter) Stop() {
	select {
	case rl.stopCleaner <- struct{}{}:
	default:
	}
}

// GetStats returns rate limiter statistics
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.bucketMutex.RLock()
	defer rl.bucketMutex.RUnlock()

	stats := map[string]interface{}{
		"enabled":          rl.config.Enabled,
		"requests_per_min": rl.config.RequestsPerMinute,
		"burst_size":       rl.config.BurstSize,
		"active_buckets":   len(rl.buckets),
	}

	return stats
}

// RateLimitMiddleware creates HTTP middleware for rate limiting
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract client IP
			clientIP := getClientIP(r)
			
			// Check rate limit
			result := limiter.Check(clientIP)
			
			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.config.RequestsPerMinute))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", result.Remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", result.ResetTime.Unix()))
			
			if !result.Allowed {
				// Set retry after header
				w.Header().Set("Retry-After", fmt.Sprintf("%.0f", result.RetryAfter.Seconds()))
				
				// Log rate limit exceeded
				if limiter.logger != nil {
					limiter.logger.Warn(r.Context(),
						errors.NewSecurityError("RATE_LIMIT_EXCEEDED", "Rate limit exceeded"),
						"Rate limit exceeded",
						"client_ip", clientIP,
						"user_agent", r.UserAgent(),
						"path", r.URL.Path,
						"method", r.Method)
				}
				
				// Return 429 Too Many Requests
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// AdaptiveRateLimiter implements adaptive rate limiting based on system load
type AdaptiveRateLimiter struct {
	*RateLimiter
	loadThreshold    float64
	reductionFactor  float64
	checkInterval    time.Duration
	lastCheck        time.Time
	currentLimit     int
	baseLimit        int
	adjustmentMutex  sync.RWMutex
}

// NewAdaptiveRateLimiter creates a new adaptive rate limiter
func NewAdaptiveRateLimiter(config *RateLimitConfig, logger logging.Logger) *AdaptiveRateLimiter {
	baseLimiter := NewRateLimiter(config, logger)
	
	return &AdaptiveRateLimiter{
		RateLimiter:     baseLimiter,
		loadThreshold:   0.8, // 80% load threshold
		reductionFactor: 0.5, // Reduce to 50% when overloaded
		checkInterval:   30 * time.Second,
		baseLimit:       config.RequestsPerMinute,
		currentLimit:    config.RequestsPerMinute,
	}
}

// Check checks rate limit with adaptive adjustment
func (arl *AdaptiveRateLimiter) Check(key string) RateLimitResult {
	arl.adjustLimitsIfNeeded()
	return arl.RateLimiter.Check(key)
}

// adjustLimitsIfNeeded adjusts rate limits based on system load
func (arl *AdaptiveRateLimiter) adjustLimitsIfNeeded() {
	now := time.Now()
	if now.Sub(arl.lastCheck) < arl.checkInterval {
		return
	}

	arl.adjustmentMutex.Lock()
	defer arl.adjustmentMutex.Unlock()

	// Simple load check - in a real implementation, you'd check CPU, memory, etc.
	activeConnections := len(arl.buckets)
	loadFactor := float64(activeConnections) / 100.0 // Assume 100 is high load

	if loadFactor > arl.loadThreshold {
		// Reduce limits
		newLimit := int(float64(arl.baseLimit) * arl.reductionFactor)
		if newLimit != arl.currentLimit {
			arl.currentLimit = newLimit
			arl.config.RequestsPerMinute = newLimit
			
			if arl.logger != nil {
				arl.logger.Warn(context.Background(), nil,
					"Rate limit reduced due to high load",
					"load_factor", loadFactor,
					"new_limit", newLimit,
					"base_limit", arl.baseLimit)
			}
		}
	} else if loadFactor < arl.loadThreshold*0.5 {
		// Restore normal limits
		if arl.currentLimit != arl.baseLimit {
			arl.currentLimit = arl.baseLimit
			arl.config.RequestsPerMinute = arl.baseLimit
			
			if arl.logger != nil {
				arl.logger.Info(context.Background(),
					"Rate limit restored to normal",
					"load_factor", loadFactor,
					"restored_limit", arl.baseLimit)
			}
		}
	}

	arl.lastCheck = now
}

// GetCurrentLimit returns the current effective rate limit
func (arl *AdaptiveRateLimiter) GetCurrentLimit() int {
	arl.adjustmentMutex.RLock()
	defer arl.adjustmentMutex.RUnlock()
	return arl.currentLimit
}

// IPWhitelist manages IP addresses that bypass rate limiting
type IPWhitelist struct {
	whitelist map[string]bool
	mutex     sync.RWMutex
}

// NewIPWhitelist creates a new IP whitelist
func NewIPWhitelist(ips []string) *IPWhitelist {
	whitelist := make(map[string]bool)
	for _, ip := range ips {
		whitelist[ip] = true
	}
	
	return &IPWhitelist{
		whitelist: whitelist,
	}
}

// IsWhitelisted checks if an IP is whitelisted
func (w *IPWhitelist) IsWhitelisted(ip string) bool {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	return w.whitelist[ip]
}

// Add adds an IP to the whitelist
func (w *IPWhitelist) Add(ip string) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.whitelist[ip] = true
}

// Remove removes an IP from the whitelist
func (w *IPWhitelist) Remove(ip string) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	delete(w.whitelist, ip)
}

// WhitelistMiddleware creates middleware that bypasses rate limiting for whitelisted IPs
func WhitelistMiddleware(whitelist *IPWhitelist, rateLimitHandler http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := getClientIP(r)
			
			if whitelist.IsWhitelisted(clientIP) {
				// Bypass rate limiting
				next.ServeHTTP(w, r)
				return
			}
			
			// Apply rate limiting
			rateLimitHandler.ServeHTTP(w, r)
		})
	}
}

// DDoSProtection implements basic DDoS protection
type DDoSProtection struct {
	rateLimiter     *RateLimiter
	suspiciousIPs   map[string]*SuspiciousIP
	mutex           sync.RWMutex
	blockDuration   time.Duration
	requestThreshold int
	timeWindow      time.Duration
	logger          logging.Logger
}

// SuspiciousIP tracks suspicious IP activity
type SuspiciousIP struct {
	requestCount int
	firstSeen    time.Time
	lastSeen     time.Time
	blocked      bool
	blockUntil   time.Time
}

// NewDDoSProtection creates a new DDoS protection system
func NewDDoSProtection(rateLimiter *RateLimiter, logger logging.Logger) *DDoSProtection {
	return &DDoSProtection{
		rateLimiter:      rateLimiter,
		suspiciousIPs:    make(map[string]*SuspiciousIP),
		blockDuration:    10 * time.Minute,
		requestThreshold: 1000, // 1000 requests in time window is suspicious
		timeWindow:       time.Minute,
		logger:           logger,
	}
}

// CheckRequest checks if a request should be blocked
func (ddos *DDoSProtection) CheckRequest(ip string) bool {
	ddos.mutex.Lock()
	defer ddos.mutex.Unlock()

	now := time.Now()
	
	// Get or create suspicious IP entry
	suspIP, exists := ddos.suspiciousIPs[ip]
	if !exists {
		suspIP = &SuspiciousIP{
			requestCount: 1,
			firstSeen:    now,
			lastSeen:     now,
		}
		ddos.suspiciousIPs[ip] = suspIP
		return true // Allow first request
	}

	// Check if currently blocked
	if suspIP.blocked && now.Before(suspIP.blockUntil) {
		return false // Still blocked
	}

	// Reset block if expired
	if suspIP.blocked && now.After(suspIP.blockUntil) {
		suspIP.blocked = false
		suspIP.requestCount = 1
		suspIP.firstSeen = now
	}

	// Reset counter if outside time window
	if now.Sub(suspIP.firstSeen) > ddos.timeWindow {
		suspIP.requestCount = 1
		suspIP.firstSeen = now
	} else {
		suspIP.requestCount++
	}

	suspIP.lastSeen = now

	// Check if threshold exceeded
	if suspIP.requestCount > ddos.requestThreshold {
		suspIP.blocked = true
		suspIP.blockUntil = now.Add(ddos.blockDuration)
		
		if ddos.logger != nil {
			ddos.logger.Error(context.Background(),
				errors.NewSecurityError("DDOS_PROTECTION_TRIGGERED", "DDoS protection triggered"),
				"DDoS protection: IP blocked",
				"ip", ip,
				"request_count", suspIP.requestCount,
				"time_window", ddos.timeWindow.String(),
				"block_duration", ddos.blockDuration.String())
		}
		
		return false
	}

	return true
}

// GetBlockedIPs returns currently blocked IPs
func (ddos *DDoSProtection) GetBlockedIPs() []string {
	ddos.mutex.RLock()
	defer ddos.mutex.RUnlock()

	var blockedIPs []string
	now := time.Now()

	for ip, suspIP := range ddos.suspiciousIPs {
		if suspIP.blocked && now.Before(suspIP.blockUntil) {
			blockedIPs = append(blockedIPs, ip)
		}
	}

	return blockedIPs
}

// DDoSMiddleware creates middleware for DDoS protection
func DDoSMiddleware(ddos *DDoSProtection) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := getClientIP(r)
			
			if !ddos.CheckRequest(clientIP) {
				http.Error(w, "Too Many Requests - Blocked", http.StatusTooManyRequests)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}