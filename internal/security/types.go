package security

// TokenBucketManager manages rate limiting using token bucket algorithm
type TokenBucketManager struct {
	// Implementation will be moved here from sliding_rate_limiter.go
}

// OriginValidator validates WebSocket connection origins
type OriginValidator interface {
	ValidateOrigin(origin string) bool
}
