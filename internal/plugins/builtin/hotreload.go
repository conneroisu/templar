package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/conneroisu/templar/internal/plugins"
)

// HotReloadPlugin provides hot reload functionality for development
type HotReloadPlugin struct {
	config      plugins.PluginConfig
	connections map[string]plugins.WebSocketConnection
	connMutex   sync.RWMutex
	enabled     bool
	reloadQueue chan ReloadEvent
}

// ReloadEvent represents a hot reload event
type ReloadEvent struct {
	Type      string            `json:"type"`
	File      string            `json:"file"`
	Component string            `json:"component,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// NewHotReloadPlugin creates a new hot reload plugin
func NewHotReloadPlugin() *HotReloadPlugin {
	return &HotReloadPlugin{
		connections: make(map[string]plugins.WebSocketConnection),
		enabled:     true,
		reloadQueue: make(chan ReloadEvent, 100),
	}
}

// Name returns the plugin name
func (hrp *HotReloadPlugin) Name() string {
	return "hotreload"
}

// Version returns the plugin version
func (hrp *HotReloadPlugin) Version() string {
	return "1.0.0"
}

// Description returns the plugin description
func (hrp *HotReloadPlugin) Description() string {
	return "Hot reload functionality for real-time development feedback"
}

// Initialize initializes the hot reload plugin
func (hrp *HotReloadPlugin) Initialize(ctx context.Context, config plugins.PluginConfig) error {
	hrp.config = config
	
	// Start event processor
	go hrp.processReloadEvents(ctx)
	
	return nil
}

// Shutdown shuts down the plugin
func (hrp *HotReloadPlugin) Shutdown(ctx context.Context) error {
	hrp.enabled = false
	
	// Close all connections
	hrp.connMutex.Lock()
	for id, conn := range hrp.connections {
		conn.Close()
		delete(hrp.connections, id)
	}
	hrp.connMutex.Unlock()
	
	close(hrp.reloadQueue)
	return nil
}

// Health returns the plugin health status
func (hrp *HotReloadPlugin) Health() plugins.PluginHealth {
	hrp.connMutex.RLock()
	connectionCount := len(hrp.connections)
	hrp.connMutex.RUnlock()
	
	status := plugins.HealthStatusHealthy
	if !hrp.enabled {
		status = plugins.HealthStatusUnhealthy
	}
	
	return plugins.PluginHealth{
		Status:    status,
		LastCheck: time.Now(),
		Metrics: map[string]interface{}{
			"active_connections": connectionCount,
			"queue_length":       len(hrp.reloadQueue),
		},
	}
}

// RegisterRoutes registers HTTP routes for hot reload functionality
func (hrp *HotReloadPlugin) RegisterRoutes(router plugins.Router) error {
	// Register hot reload WebSocket endpoint
	router.GET("/ws/hotreload", hrp.handleWebSocket)
	
	// Register hot reload status endpoint
	router.GET("/api/hotreload/status", hrp.handleStatus)
	
	// Register manual reload trigger
	router.POST("/api/hotreload/trigger", hrp.handleTrigger)
	
	return nil
}

// Middleware returns middleware functions
func (hrp *HotReloadPlugin) Middleware() []plugins.MiddlewareFunc {
	return []plugins.MiddlewareFunc{
		hrp.injectReloadScript,
	}
}

// WebSocketHandler handles WebSocket connections for hot reload
func (hrp *HotReloadPlugin) WebSocketHandler(ctx context.Context, conn plugins.WebSocketConnection) error {
	if !hrp.enabled {
		return fmt.Errorf("hot reload plugin is disabled")
	}
	
	// Generate connection ID
	connID := fmt.Sprintf("conn_%d", time.Now().UnixNano())
	
	// Store connection
	hrp.connMutex.Lock()
	hrp.connections[connID] = conn
	hrp.connMutex.Unlock()
	
	// Clean up on exit
	defer func() {
		hrp.connMutex.Lock()
		delete(hrp.connections, connID)
		hrp.connMutex.Unlock()
		conn.Close()
	}()
	
	// Send initial connection message
	welcomeMsg := ReloadEvent{
		Type:      "connected",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"id":      connID,
			"version": hrp.Version(),
		},
	}
	
	if err := hrp.sendEvent(conn, welcomeMsg); err != nil {
		return fmt.Errorf("failed to send welcome message: %w", err)
	}
	
	// Keep connection alive and handle incoming messages
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Set read timeout
			// Note: This would require a more sophisticated WebSocket implementation
			data, err := conn.Receive()
			if err != nil {
				return fmt.Errorf("failed to receive data: %w", err)
			}
			
			// Handle ping/pong or other client messages
			if string(data) == "ping" {
				if err := conn.Send([]byte("pong")); err != nil {
					return fmt.Errorf("failed to send pong: %w", err)
				}
			}
		}
	}
}

// WatchPatterns returns file patterns to watch for hot reload
func (hrp *HotReloadPlugin) WatchPatterns() []string {
	return []string{
		"**/*.templ",
		"**/*.html",
		"**/*.css",
		"**/*.js",
		"**/*.ts",
		"**/*.tsx",
		"**/*.jsx",
		"**/*.go",
	}
}

// HandleFileChange handles file change events for hot reload
func (hrp *HotReloadPlugin) HandleFileChange(ctx context.Context, event plugins.FileChangeEvent) error {
	if !hrp.enabled {
		return nil
	}
	
	// Create reload event
	reloadEvent := ReloadEvent{
		Type:      "file_changed",
		File:      event.Path,
		Timestamp: event.Timestamp,
		Data: map[string]interface{}{
			"change_type": string(event.Type),
		},
	}
	
	// Determine component name from file path
	if component := hrp.extractComponentName(event.Path); component != "" {
		reloadEvent.Component = component
		reloadEvent.Type = "component_changed"
	}
	
	// Queue the event
	select {
	case hrp.reloadQueue <- reloadEvent:
	default:
		// Queue is full, skip this event
	}
	
	return nil
}

// ShouldIgnore determines if a file change should be ignored
func (hrp *HotReloadPlugin) ShouldIgnore(filePath string) bool {
	// Ignore certain file patterns
	ignorePatterns := []string{
		".git/",
		"node_modules/",
		".DS_Store",
		"*.tmp",
		"*.log",
		"*~",
	}
	
	for _, pattern := range ignorePatterns {
		if match, _ := filepath.Match(pattern, filePath); match {
			return true
		}
	}
	
	return false
}

// processReloadEvents processes queued reload events
func (hrp *HotReloadPlugin) processReloadEvents(ctx context.Context) {
	debouncer := make(map[string]time.Time)
	debounceInterval := 250 * time.Millisecond
	
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-hrp.reloadQueue:
			if !ok {
				return
			}
			
			// Debounce events for the same file
			key := event.File
			if lastTime, exists := debouncer[key]; exists {
				if time.Since(lastTime) < debounceInterval {
					continue // Skip this event, too soon
				}
			}
			debouncer[key] = time.Now()
			
			// Broadcast to all connections
			hrp.broadcastEvent(event)
			
			// Clean up old debouncer entries
			for file, timestamp := range debouncer {
				if time.Since(timestamp) > time.Minute {
					delete(debouncer, file)
				}
			}
		}
	}
}

// broadcastEvent broadcasts an event to all connected clients
func (hrp *HotReloadPlugin) broadcastEvent(event ReloadEvent) {
	hrp.connMutex.RLock()
	connections := make([]plugins.WebSocketConnection, 0, len(hrp.connections))
	for _, conn := range hrp.connections {
		connections = append(connections, conn)
	}
	hrp.connMutex.RUnlock()
	
	for _, conn := range connections {
		if err := hrp.sendEvent(conn, event); err != nil {
			// Log error but continue with other connections
			fmt.Printf("Failed to send event to client: %v\n", err)
		}
	}
}

// sendEvent sends an event to a WebSocket connection
func (hrp *HotReloadPlugin) sendEvent(conn plugins.WebSocketConnection, event ReloadEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	
	return conn.Send(data)
}

// extractComponentName extracts component name from file path
func (hrp *HotReloadPlugin) extractComponentName(filePath string) string {
	// Simple extraction - get filename without extension
	base := filepath.Base(filePath)
	ext := filepath.Ext(base)
	if ext != "" {
		base = base[:len(base)-len(ext)]
	}
	
	// Capitalize first letter to match Go conventions
	if len(base) > 0 {
		base = strings.ToUpper(string(base[0])) + base[1:]
	}
	
	return base
}

// HTTP handlers
func (hrp *HotReloadPlugin) handleWebSocket(ctx plugins.Context) error {
	// This would be implemented based on the specific HTTP framework
	// For now, return a placeholder
	return ctx.String(200, "WebSocket endpoint for hot reload")
}

func (hrp *HotReloadPlugin) handleStatus(ctx plugins.Context) error {
	status := hrp.Health()
	return ctx.JSON(200, map[string]interface{}{
		"plugin":  hrp.Name(),
		"version": hrp.Version(),
		"health":  status,
	})
}

func (hrp *HotReloadPlugin) handleTrigger(ctx plugins.Context) error {
	// Manual reload trigger
	event := ReloadEvent{
		Type:      "manual_reload",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"triggered_by": "api",
		},
	}
	
	hrp.broadcastEvent(event)
	
	return ctx.JSON(200, map[string]interface{}{
		"status": "reload triggered",
		"event":  event,
	})
}

// injectReloadScript middleware injects hot reload client script
func (hrp *HotReloadPlugin) injectReloadScript(next plugins.HandlerFunc) plugins.HandlerFunc {
	return func(ctx plugins.Context) error {
		// Call the next handler
		err := next(ctx)
		
		// If this is an HTML response, inject the reload script
		if ctx.Header("Content-Type") == "text/html" {
			// This is a simplified implementation
			// In practice, you'd need to modify the response body
			ctx.Set("hotreload_script_injected", true)
		}
		
		return err
	}
}

// Ensure HotReloadPlugin implements the required interfaces
var _ plugins.ServerPlugin = (*HotReloadPlugin)(nil)
var _ plugins.WatcherPlugin = (*HotReloadPlugin)(nil)