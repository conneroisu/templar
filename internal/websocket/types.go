package websocket

import (
	"time"

	"github.com/coder/websocket"
)

// WebSocketRateLimiter defines rate limiting for WebSocket connections
type WebSocketRateLimiter interface {
	IsAllowed() bool
	Reset()
}

// Client represents a WebSocket client connection
type Client struct {
	conn         *websocket.Conn
	send         chan []byte
	lastActivity time.Time
	rateLimiter  RateLimiter
}

// UpdateMessage represents a message sent to the browser
type UpdateMessage struct {
	Type      string    `json:"type"`
	Target    string    `json:"target,omitempty"`
	Content   string    `json:"content,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// RateLimiter interface for WebSocket rate limiting
type RateLimiter interface {
	Allow() bool
	Reset()
}

// WebSocketEnhancements provides additional WebSocket features and metrics
type WebSocketEnhancements struct {
	// Placeholder for enhanced WebSocket functionality
	// This will be implemented as part of performance optimizations
}
