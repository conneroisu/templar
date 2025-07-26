package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTokenBucket_Consume(t *testing.T) {
	bucket := &TokenBucket{
		tokens:     5,
		capacity:   10,
		refillRate: 60, // 60 tokens per minute = 1 per second
		lastRefill: time.Now(),
	}

	// Should allow consuming tokens up to capacity
	for i := range 5 {
		result := bucket.consume()
		assert.True(t, result.Allowed)
		assert.Equal(t, 4-i, result.Remaining)
	}

	// Should deny when no tokens left
	result := bucket.consume()
	assert.False(t, result.Allowed)
	assert.Equal(t, 0, result.Remaining)
	assert.Greater(t, result.RetryAfter, time.Duration(0))
}

func TestTokenBucket_Refill(t *testing.T) {
	bucket := &TokenBucket{
		tokens:     0,
		capacity:   10,
		refillRate: 60,                               // 60 tokens per minute
		lastRefill: time.Now().Add(-2 * time.Minute), // 2 minutes ago
	}

	result := bucket.consume()

	// Should have refilled and allowed request
	assert.True(t, result.Allowed)
	assert.Greater(t, result.Remaining, 0)
}

func TestTokenBucket_RefillCap(t *testing.T) {
	bucket := &TokenBucket{
		tokens:     5,
		capacity:   10,
		refillRate: 60,
		lastRefill: time.Now().Add(-10 * time.Minute), // 10 minutes ago
	}

	bucket.refill(time.Now())

	// Should not exceed capacity
	assert.Equal(t, 10, bucket.tokens)
}

func TestRateLimiter_Check(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerMinute: 60,
		BurstSize:         10,
		WindowSize:        time.Minute,
		Enabled:           true,
	}

	limiter := NewRateLimiter(config, nil)
	defer limiter.Stop()

	// Test multiple requests from same IP
	for i := range 10 {
		result := limiter.Check("192.168.1.1")
		assert.True(t, result.Allowed)
		assert.Equal(t, 9-i, result.Remaining)
	}

	// 11th request should be denied
	result := limiter.Check("192.168.1.1")
	assert.False(t, result.Allowed)
	assert.Equal(t, 0, result.Remaining)
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerMinute: 60,
		BurstSize:         5,
		WindowSize:        time.Minute,
		Enabled:           true,
	}

	limiter := NewRateLimiter(config, nil)
	defer limiter.Stop()

	// Different IPs should have separate buckets
	result1 := limiter.Check("192.168.1.1")
	result2 := limiter.Check("192.168.1.2")

	assert.True(t, result1.Allowed)
	assert.True(t, result2.Allowed)
	assert.Equal(t, 4, result1.Remaining)
	assert.Equal(t, 4, result2.Remaining)
}

func TestRateLimiter_Disabled(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerMinute: 1,
		BurstSize:         1,
		WindowSize:        time.Minute,
		Enabled:           false,
	}

	limiter := NewRateLimiter(config, nil)
	defer limiter.Stop()

	// Should allow all requests when disabled
	for range 100 {
		result := limiter.Check("192.168.1.1")
		assert.True(t, result.Allowed)
	}
}

func TestRateLimiter_Concurrent(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerMinute: 1000,
		BurstSize:         50,
		WindowSize:        time.Minute,
		Enabled:           true,
	}

	limiter := NewRateLimiter(config, nil)
	defer limiter.Stop()

	// Test concurrent access
	var wg sync.WaitGroup
	var allowedCount, deniedCount int32
	var mu sync.Mutex

	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := limiter.Check("192.168.1.1")

			mu.Lock()
			if result.Allowed {
				allowedCount++
			} else {
				deniedCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Should allow exactly burst size requests
	assert.Equal(t, int32(50), allowedCount)
	assert.Equal(t, int32(50), deniedCount)
}

func TestRateLimitMiddleware(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerMinute: 60,
		BurstSize:         3,
		WindowSize:        time.Minute,
		Enabled:           true,
	}

	limiter := NewRateLimiter(config, nil)
	defer limiter.Stop()

	middleware := RateLimitMiddleware(limiter)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First 3 requests should pass
	for i := range 3 {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:8080"
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "60", w.Header().Get("X-RateLimit-Limit"))

		// Remaining should decrease with each request
		expectedRemaining := strconv.Itoa(2 - i)
		assert.Equal(t, expectedRemaining, w.Header().Get("X-RateLimit-Remaining"))
	}

	// 4th request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "0", w.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
}

func TestRateLimitMiddleware_Headers(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerMinute: 100,
		BurstSize:         10,
		WindowSize:        time.Minute,
		Enabled:           true,
	}

	limiter := NewRateLimiter(config, nil)
	defer limiter.Stop()

	middleware := RateLimitMiddleware(limiter)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Check rate limit headers
	assert.Equal(t, "100", w.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "9", w.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
}

func TestIPWhitelist(t *testing.T) {
	whitelist := NewIPWhitelist([]string{"192.168.1.1", "10.0.0.1"})

	assert.True(t, whitelist.IsWhitelisted("192.168.1.1"))
	assert.True(t, whitelist.IsWhitelisted("10.0.0.1"))
	assert.False(t, whitelist.IsWhitelisted("192.168.1.2"))

	// Test add/remove
	whitelist.Add("192.168.1.3")
	assert.True(t, whitelist.IsWhitelisted("192.168.1.3"))

	whitelist.Remove("192.168.1.1")
	assert.False(t, whitelist.IsWhitelisted("192.168.1.1"))
}

func TestWhitelistMiddleware(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerMinute: 1,
		BurstSize:         1,
		WindowSize:        time.Minute,
		Enabled:           true,
	}

	limiter := NewRateLimiter(config, nil)
	defer limiter.Stop()

	whitelist := NewIPWhitelist([]string{"192.168.1.100"})

	rateLimitHandler := RateLimitMiddleware(
		limiter,
	)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	middleware := WhitelistMiddleware(whitelist, rateLimitHandler)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Whitelisted IP should bypass rate limit
	for range 10 {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.100:8080"
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Non-whitelisted IP should be rate limited
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "192.168.1.1:8080"
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code) // First request allowed

	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.1:8080"
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusTooManyRequests, w2.Code) // Second request blocked
}

func TestAdaptiveRateLimiter(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerMinute: 100,
		BurstSize:         10,
		WindowSize:        time.Minute,
		Enabled:           true,
	}

	limiter := NewAdaptiveRateLimiter(config, nil)
	defer limiter.Stop()

	// Test normal operation
	result := limiter.Check("192.168.1.1")
	assert.True(t, result.Allowed)
	assert.Equal(t, 100, limiter.GetCurrentLimit())

	// Simulate high load by creating many buckets
	for i := range 200 {
		limiter.Check(fmt.Sprintf("192.168.1.%d", i))
	}

	// Force adjustment by manipulating lastCheck to bypass time interval check
	limiter.adjustmentMutex.Lock()
	limiter.lastCheck = time.Now().
		Add(-time.Hour)
		// Make it seem like we haven't checked in an hour
	limiter.adjustmentMutex.Unlock()

	// Force adjustment check
	limiter.adjustLimitsIfNeeded()

	// Should have reduced limit due to high load
	assert.Less(t, limiter.GetCurrentLimit(), 100)
}

func TestDDoSProtection(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerMinute: 1000,
		BurstSize:         100,
		WindowSize:        time.Minute,
		Enabled:           true,
	}

	limiter := NewRateLimiter(config, nil)
	defer limiter.Stop()

	ddos := NewDDoSProtection(limiter, nil)

	// Test normal requests
	for range 100 {
		allowed := ddos.CheckRequest("192.168.1.1")
		assert.True(t, allowed)
	}

	// Test excessive requests that should trigger protection
	ip := "192.168.1.2"
	var blockedCount int

	for range 1500 {
		if !ddos.CheckRequest(ip) {
			blockedCount++
		}
	}

	assert.Greater(t, blockedCount, 0)

	// Check that IP is in blocked list
	blockedIPs := ddos.GetBlockedIPs()
	assert.Contains(t, blockedIPs, ip)
}

func TestDDoSMiddleware(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerMinute: 1000,
		BurstSize:         100,
		WindowSize:        time.Minute,
		Enabled:           true,
	}

	limiter := NewRateLimiter(config, nil)
	defer limiter.Stop()

	ddos := NewDDoSProtection(limiter, nil)
	ddos.requestThreshold = 5 // Lower threshold for testing

	middleware := DDoSMiddleware(ddos)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Send many requests quickly to trigger DDoS protection
	for i := range 10 {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:8080"
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if i < 5 {
			assert.Equal(t, http.StatusOK, w.Code)
		} else {
			assert.Equal(t, http.StatusTooManyRequests, w.Code)
		}
	}
}

func TestRateLimiter_CleanupExpiredBuckets(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerMinute: 60,
		BurstSize:         10,
		WindowSize:        time.Minute,
		Enabled:           true,
	}

	limiter := NewRateLimiter(config, nil)

	// Create some buckets
	limiter.Check("192.168.1.1")
	limiter.Check("192.168.1.2")
	limiter.Check("192.168.1.3")

	assert.Equal(t, 3, len(limiter.buckets))

	// Manually set old access times
	for _, bucket := range limiter.buckets {
		bucket.mutex.Lock()
		bucket.lastAccess = time.Now().Add(-15 * time.Minute)
		bucket.mutex.Unlock()
	}

	// Trigger cleanup
	limiter.performCleanup()

	// All buckets should be cleaned up
	assert.Equal(t, 0, len(limiter.buckets))

	limiter.Stop()
}

func TestRateLimiter_GetStats(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerMinute: 100,
		BurstSize:         20,
		WindowSize:        time.Minute,
		Enabled:           true,
	}

	limiter := NewRateLimiter(config, nil)
	defer limiter.Stop()

	// Create some activity
	limiter.Check("192.168.1.1")
	limiter.Check("192.168.1.2")

	stats := limiter.GetStats()

	assert.Equal(t, true, stats["enabled"])
	assert.Equal(t, 100, stats["requests_per_min"])
	assert.Equal(t, 20, stats["burst_size"])
	assert.Equal(t, 2, stats["active_buckets"])
}

func BenchmarkRateLimiter_Check(b *testing.B) {
	config := &RateLimitConfig{
		RequestsPerMinute: 1000,
		BurstSize:         100,
		WindowSize:        time.Minute,
		Enabled:           true,
	}

	limiter := NewRateLimiter(config, nil)
	defer limiter.Stop()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			ip := fmt.Sprintf("192.168.1.%d", i%255)
			limiter.Check(ip)
			i++
		}
	})
}

func BenchmarkTokenBucket_Consume(b *testing.B) {
	bucket := &TokenBucket{
		tokens:     1000,
		capacity:   1000,
		refillRate: 1000,
		lastRefill: time.Now(),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		bucket.consume()
	}
}

func BenchmarkRateLimitMiddleware(b *testing.B) {
	config := &RateLimitConfig{
		RequestsPerMinute: 10000,
		BurstSize:         1000,
		WindowSize:        time.Minute,
		Enabled:           true,
	}

	limiter := NewRateLimiter(config, nil)
	defer limiter.Stop()

	middleware := RateLimitMiddleware(limiter)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:8080"

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}
