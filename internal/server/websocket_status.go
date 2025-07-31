// Package server provides WebSocket connection status management and indicators.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// StatusIndicator represents connection status information for clients.
type StatusIndicator struct {
	Status            string                 `json:"status"`
	Connected         bool                   `json:"connected"`
	LastConnected     *time.Time             `json:"last_connected,omitempty"`
	LastDisconnected  *time.Time             `json:"last_disconnected,omitempty"`
	ReconnectAttempts int64                  `json:"reconnect_attempts"`
	Uptime            string                 `json:"uptime,omitempty"`
	QueuedMessages    int                    `json:"queued_messages"`
	Metrics           map[string]interface{} `json:"metrics,omitempty"`
	NextRetry         *time.Time             `json:"next_retry,omitempty"`
}

// WebSocketStatusManager manages connection status and provides indicators.
type WebSocketStatusManager struct {
	// Reliability manager integration
	reliabilityManager *WebSocketReliabilityManager

	// Status tracking
	statusMutex      sync.RWMutex
	currentStatus    StatusIndicator
	lastStatusUpdate time.Time

	// Client notification
	statusSubscribers map[*websocket.Conn]bool
	subscribersMutex  sync.RWMutex

	// Offline mode management
	offlineModeEnabled bool
	offlineModeReason  string
	offlineModeMutex   sync.RWMutex
}

// NewWebSocketStatusManager creates a new status manager.
func NewWebSocketStatusManager(reliabilityManager *WebSocketReliabilityManager) *WebSocketStatusManager {
	wsm := &WebSocketStatusManager{
		reliabilityManager: reliabilityManager,
		statusSubscribers:  make(map[*websocket.Conn]bool),
		currentStatus: StatusIndicator{
			Status:    "disconnected",
			Connected: false,
		},
		lastStatusUpdate: time.Now(),
	}

	// Register for status updates from reliability manager
	if reliabilityManager != nil {
		reliabilityManager.AddStatusCallback(wsm.handleStatusChange)
	}

	return wsm
}

// GetStatusIndicator returns the current status indicator.
func (wsm *WebSocketStatusManager) GetStatusIndicator() StatusIndicator {
	wsm.statusMutex.RLock()
	defer wsm.statusMutex.RUnlock()

	// Create a copy to avoid race conditions
	indicator := wsm.currentStatus

	// Add real-time metrics if reliability manager is available
	if wsm.reliabilityManager != nil {
		metrics := wsm.reliabilityManager.GetMetrics()
		indicator.Metrics = map[string]interface{}{
			"total_attempts":       metrics.TotalAttempts,
			"successful_conns":     metrics.SuccessfulConns,
			"failed_attempts":      metrics.FailedAttempts,
			"reconnect_attempts":   metrics.ReconnectAttempts,
			"reconnect_success":    metrics.ReconnectSuccess,
			"ping_sent":            metrics.PingSent,
			"pong_received":        metrics.PongReceived,
			"consecutive_failures": metrics.ConsecutiveFailures,
		}

		// Calculate uptime if connected
		if indicator.Connected && metrics.CurrentConnStart > 0 {
			duration := time.Since(time.Unix(0, metrics.CurrentConnStart))
			indicator.Uptime = formatDuration(duration)
		}

		// Set queued message count
		// Note: We would need to expose this from the reliability manager
		// For now, using a placeholder
		indicator.QueuedMessages = 0 // TODO: Get from reliability manager
	}

	return indicator
}

// SubscribeToStatus adds a WebSocket connection to receive status updates.
func (wsm *WebSocketStatusManager) SubscribeToStatus(conn *websocket.Conn) {
	wsm.subscribersMutex.Lock()
	defer wsm.subscribersMutex.Unlock()

	wsm.statusSubscribers[conn] = true

	// Send current status immediately
	status := wsm.GetStatusIndicator()
	go wsm.sendStatusToConnection(conn, status)

	log.Printf("Client subscribed to status updates (total: %d)", len(wsm.statusSubscribers))
}

// UnsubscribeFromStatus removes a WebSocket connection from status updates.
func (wsm *WebSocketStatusManager) UnsubscribeFromStatus(conn *websocket.Conn) {
	wsm.subscribersMutex.Lock()
	defer wsm.subscribersMutex.Unlock()

	delete(wsm.statusSubscribers, conn)

	log.Printf("Client unsubscribed from status updates (total: %d)", len(wsm.statusSubscribers))
}

// EnableOfflineMode enables offline mode with a reason.
func (wsm *WebSocketStatusManager) EnableOfflineMode(reason string) {
	wsm.offlineModeMutex.Lock()
	wsm.offlineModeEnabled = true
	wsm.offlineModeReason = reason
	wsm.offlineModeMutex.Unlock()

	if wsm.reliabilityManager != nil {
		wsm.reliabilityManager.SetOfflineMode(true)
	}

	wsm.updateStatus(StatusIndicator{
		Status:    "offline",
		Connected: false,
	})

	log.Printf("Offline mode enabled: %s", reason)
}

// DisableOfflineMode disables offline mode and resumes normal operation.
func (wsm *WebSocketStatusManager) DisableOfflineMode() {
	wsm.offlineModeMutex.Lock()
	wasOffline := wsm.offlineModeEnabled
	wsm.offlineModeEnabled = false
	wsm.offlineModeReason = ""
	wsm.offlineModeMutex.Unlock()

	if wasOffline {
		if wsm.reliabilityManager != nil {
			wsm.reliabilityManager.SetOfflineMode(false)
		}

		wsm.updateStatus(StatusIndicator{
			Status:    "disconnected",
			Connected: false,
		})

		log.Printf("Offline mode disabled, resuming normal operation")
	}
}

// IsOfflineMode returns whether offline mode is currently enabled.
func (wsm *WebSocketStatusManager) IsOfflineMode() (bool, string) {
	wsm.offlineModeMutex.RLock()
	defer wsm.offlineModeMutex.RUnlock()

	return wsm.offlineModeEnabled, wsm.offlineModeReason
}

// handleStatusChange handles status changes from the reliability manager.
func (wsm *WebSocketStatusManager) handleStatusChange(status ConnectionStatus, err error) {
	var indicator StatusIndicator
	now := time.Now()

	switch status {
	case StatusDisconnected:
		indicator = StatusIndicator{
			Status:           "disconnected",
			Connected:        false,
			LastDisconnected: &now,
		}
	case StatusConnecting:
		indicator = StatusIndicator{
			Status:    "connecting",
			Connected: false,
		}
	case StatusConnected:
		indicator = StatusIndicator{
			Status:        "connected",
			Connected:     true,
			LastConnected: &now,
		}
	case StatusReconnecting:
		indicator = StatusIndicator{
			Status:    "reconnecting",
			Connected: false,
		}
	case StatusOffline:
		indicator = StatusIndicator{
			Status:    "offline",
			Connected: false,
		}
	default:
		indicator = StatusIndicator{
			Status:    "unknown",
			Connected: false,
		}
	}

	// Add reconnection attempts if available
	if wsm.reliabilityManager != nil {
		metrics := wsm.reliabilityManager.GetMetrics()
		indicator.ReconnectAttempts = metrics.ReconnectAttempts
	}

	wsm.updateStatus(indicator)
}

// updateStatus updates the current status and notifies subscribers.
func (wsm *WebSocketStatusManager) updateStatus(indicator StatusIndicator) {
	wsm.statusMutex.Lock()
	wsm.currentStatus = indicator
	wsm.lastStatusUpdate = time.Now()
	wsm.statusMutex.Unlock()

	// Notify all subscribers
	wsm.broadcastStatusUpdate(indicator)
}

// broadcastStatusUpdate sends status updates to all subscribed connections.
func (wsm *WebSocketStatusManager) broadcastStatusUpdate(indicator StatusIndicator) {
	wsm.subscribersMutex.RLock()
	subscribers := make([]*websocket.Conn, 0, len(wsm.statusSubscribers))
	for conn := range wsm.statusSubscribers {
		subscribers = append(subscribers, conn)
	}
	wsm.subscribersMutex.RUnlock()

	// Send to all subscribers concurrently
	for _, conn := range subscribers {
		go wsm.sendStatusToConnection(conn, indicator)
	}

	if len(subscribers) > 0 {
		log.Printf("Broadcasted status update '%s' to %d subscribers",
			indicator.Status, len(subscribers))
	}
}

// sendStatusToConnection sends a status update to a specific connection.
func (wsm *WebSocketStatusManager) sendStatusToConnection(conn *websocket.Conn, indicator StatusIndicator) {
	// Create status message
	message := map[string]interface{}{
		"type":   "status_update",
		"status": indicator,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal status message: %v", err)

		return
	}

	// Send with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		log.Printf("Failed to send status update: %v", err)

		// Remove failed connection from subscribers
		wsm.subscribersMutex.Lock()
		delete(wsm.statusSubscribers, conn)
		wsm.subscribersMutex.Unlock()
	}
}

// GetConnectionHealth returns detailed connection health information.
func (wsm *WebSocketStatusManager) GetConnectionHealth() map[string]interface{} {
	health := make(map[string]interface{})

	// Basic status
	indicator := wsm.GetStatusIndicator()
	health["status"] = indicator.Status
	health["connected"] = indicator.Connected
	health["last_update"] = wsm.lastStatusUpdate

	// Offline mode status
	offline, reason := wsm.IsOfflineMode()
	health["offline_mode"] = offline
	if offline {
		health["offline_reason"] = reason
	}

	// Subscriber count
	wsm.subscribersMutex.RLock()
	health["status_subscribers"] = len(wsm.statusSubscribers)
	wsm.subscribersMutex.RUnlock()

	// Reliability metrics if available
	if wsm.reliabilityManager != nil {
		metrics := wsm.reliabilityManager.GetMetrics()
		health["reliability_metrics"] = map[string]interface{}{
			"total_attempts":       metrics.TotalAttempts,
			"successful_conns":     metrics.SuccessfulConns,
			"failed_attempts":      metrics.FailedAttempts,
			"reconnect_attempts":   metrics.ReconnectAttempts,
			"reconnect_success":    metrics.ReconnectSuccess,
			"consecutive_failures": metrics.ConsecutiveFailures,
			"network_errors":       metrics.NetworkErrors,
			"protocol_errors":      metrics.ProtocolErrors,
			"unexpected_errors":    metrics.UnexpectedErrors,
		}

		// Connection duration stats
		if metrics.TotalConnectedTime > 0 {
			avgConnection := time.Duration(metrics.AverageConnection)
			longestConnection := time.Duration(metrics.LongestConnection)
			totalConnected := time.Duration(metrics.TotalConnectedTime)

			health["connection_stats"] = map[string]interface{}{
				"average_duration": formatDuration(avgConnection),
				"longest_duration": formatDuration(longestConnection),
				"total_connected":  formatDuration(totalConnected),
			}
		}

		// Health check stats
		health["health_checks"] = map[string]interface{}{
			"pings_sent":     metrics.PingSent,
			"pongs_received": metrics.PongReceived,
			"timeouts":       metrics.TimeoutCount,
		}

		// Calculate success rates
		if metrics.TotalAttempts > 0 {
			health["connection_success_rate"] = float64(metrics.SuccessfulConns) / float64(metrics.TotalAttempts) * 100
		}

		if metrics.ReconnectAttempts > 0 {
			health["reconnect_success_rate"] = float64(metrics.ReconnectSuccess) / float64(metrics.ReconnectAttempts) * 100
		}

		if metrics.PingSent > 0 {
			health["ping_success_rate"] = float64(metrics.PongReceived) / float64(metrics.PingSent) * 100
		}
	}

	return health
}

// formatDuration formats a duration for human readability.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	} else {
		days := int(d.Hours() / 24)
		hours := d - time.Duration(days)*24*time.Hour

		return fmt.Sprintf("%dd %.1fh", days, hours.Hours())
	}
}

// Integration methods for existing WebSocket system

// IntegrateWithPreviewServer integrates the status manager with PreviewServer.
func (wsm *WebSocketStatusManager) IntegrateWithPreviewServer(server *PreviewServer) {
	if server == nil {
		return
	}

	// Note: HTTP endpoint integration would need to be implemented
	// when PreviewServer's routing architecture is refactored to expose
	// the mux or provide a registration mechanism for endpoints.
	// For now, status can be accessed programmatically via GetStatusIndicator()

	log.Printf("WebSocket status manager integrated with PreviewServer")
}

// handleStatusEndpoint handles HTTP requests for WebSocket status.
// TODO: Integrate with HTTP router when status endpoints are needed
//
//nolint:unused
func (wsm *WebSocketStatusManager) handleStatusEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		indicator := wsm.GetStatusIndicator()
		if err := json.NewEncoder(w).Encode(indicator); err != nil {
			http.Error(w, "Failed to encode status", http.StatusInternalServerError)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleHealthEndpoint handles HTTP requests for connection health.
// TODO: Integrate with HTTP router when health endpoints are needed
//
//nolint:unused
func (wsm *WebSocketStatusManager) handleHealthEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		health := wsm.GetConnectionHealth()
		if err := json.NewEncoder(w).Encode(health); err != nil {
			http.Error(w, "Failed to encode health data", http.StatusInternalServerError)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleOfflineModeEndpoint handles offline mode management via HTTP.
// TODO: Integrate with HTTP router when offline mode endpoints are needed
//
//nolint:unused
func (wsm *WebSocketStatusManager) handleOfflineModeEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		offline, reason := wsm.IsOfflineMode()
		response := map[string]interface{}{
			"offline_mode": offline,
			"reason":       reason,
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode offline mode status", http.StatusInternalServerError)
		}

	case http.MethodPost:
		var request struct {
			Enable bool   `json:"enable"`
			Reason string `json:"reason,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)

			return
		}

		if request.Enable {
			reason := request.Reason
			if reason == "" {
				reason = "User requested"
			}
			wsm.EnableOfflineMode(reason)
		} else {
			wsm.DisableOfflineMode()
		}

		// Return updated status
		offline, currentReason := wsm.IsOfflineMode()
		response := map[string]interface{}{
			"offline_mode": offline,
			"reason":       currentReason,
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
