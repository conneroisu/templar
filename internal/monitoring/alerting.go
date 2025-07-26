package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/conneroisu/templar/internal/logging"
)

// AlertLevel represents the severity of an alert.
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelCritical AlertLevel = "critical"
)

// Alert represents a monitoring alert.
type Alert struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Level     AlertLevel             `json:"level"`
	Message   string                 `json:"message"`
	Component string                 `json:"component"`
	Metric    string                 `json:"metric,omitempty"`
	Value     float64                `json:"value,omitempty"`
	Threshold float64                `json:"threshold,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Active    bool                   `json:"active"`
	Count     int                    `json:"count"`
	FirstSeen time.Time              `json:"first_seen"`
	LastSeen  time.Time              `json:"last_seen"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Labels    map[string]string      `json:"labels,omitempty"`
}

// AlertRule defines conditions for triggering alerts.
type AlertRule struct {
	Name      string            `json:"name"`
	Component string            `json:"component"`
	Metric    string            `json:"metric"`
	Condition string            `json:"condition"` // "gt", "lt", "eq", "ne"
	Threshold float64           `json:"threshold"`
	Duration  time.Duration     `json:"duration"`
	Level     AlertLevel        `json:"level"`
	Message   string            `json:"message"`
	Labels    map[string]string `json:"labels,omitempty"`
	Enabled   bool              `json:"enabled"`
	Cooldown  time.Duration     `json:"cooldown"`
}

// AlertChannel defines how alerts are delivered.
type AlertChannel interface {
	Send(ctx context.Context, alert Alert) error
	Name() string
}

// AlertManager manages alert rules, state, and delivery.
type AlertManager struct {
	rules        map[string]*AlertRule
	activeAlerts map[string]*Alert
	channels     []AlertChannel
	logger       logging.Logger
	mutex        sync.RWMutex

	// Alert state tracking
	cooldowns   map[string]time.Time
	metricCache map[string]float64
	lastCheck   time.Time
}

// NewAlertManager creates a new alert manager.
func NewAlertManager(logger logging.Logger) *AlertManager {
	return &AlertManager{
		rules:        make(map[string]*AlertRule),
		activeAlerts: make(map[string]*Alert),
		channels:     make([]AlertChannel, 0),
		logger:       logger.WithComponent("alert_manager"),
		cooldowns:    make(map[string]time.Time),
		metricCache:  make(map[string]float64),
		lastCheck:    time.Now(),
	}
}

// AddRule adds an alert rule.
func (am *AlertManager) AddRule(rule *AlertRule) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	am.rules[rule.Name] = rule
	am.logger.Info(context.Background(), "Alert rule added",
		"rule_name", rule.Name,
		"component", rule.Component,
		"metric", rule.Metric,
		"threshold", rule.Threshold)
}

// RemoveRule removes an alert rule.
func (am *AlertManager) RemoveRule(name string) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	delete(am.rules, name)
	// Also remove any active alerts for this rule
	delete(am.activeAlerts, name)
	am.logger.Info(context.Background(), "Alert rule removed", "rule_name", name)
}

// AddChannel adds an alert delivery channel.
func (am *AlertManager) AddChannel(channel AlertChannel) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	am.channels = append(am.channels, channel)
	am.logger.Info(context.Background(), "Alert channel added", "channel", channel.Name())
}

// EvaluateMetrics evaluates current metrics against alert rules.
func (am *AlertManager) EvaluateMetrics(ctx context.Context, metrics []Metric) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Update metric cache
	for _, metric := range metrics {
		key := am.getMetricKey(metric.Name, metric.Labels)
		am.metricCache[key] = metric.Value
	}

	// Evaluate rules
	for _, rule := range am.rules {
		if !rule.Enabled {
			continue
		}

		am.evaluateRule(ctx, rule)
	}

	am.lastCheck = time.Now()
}

// evaluateRule evaluates a single alert rule.
func (am *AlertManager) evaluateRule(ctx context.Context, rule *AlertRule) {
	metricKey := am.getMetricKey(rule.Metric, rule.Labels)
	value, exists := am.metricCache[metricKey]

	if !exists {
		// Metric not found - this might be an alert condition itself
		if rule.Condition == "exists" {
			am.triggerAlert(ctx, rule, 0, 0)
		}

		return
	}

	// Check if condition is met
	conditionMet := am.evaluateCondition(rule.Condition, value, rule.Threshold)

	alertID := rule.Name
	existingAlert, exists := am.activeAlerts[alertID]
	isActive := exists && existingAlert.Active

	if conditionMet {
		if !isActive {
			// New alert (or reactivate resolved alert)
			am.triggerAlert(ctx, rule, value, rule.Threshold)
		} else {
			// Update existing alert
			existingAlert.Count++
			existingAlert.LastSeen = time.Now()
			existingAlert.Value = value
		}
	} else if isActive {
		// Condition no longer met - resolve alert
		am.resolveAlert(ctx, existingAlert)
	}
}

// evaluateCondition checks if a condition is met.
func (am *AlertManager) evaluateCondition(condition string, value, threshold float64) bool {
	switch condition {
	case "gt", ">":
		return value > threshold
	case "gte", ">=":
		return value >= threshold
	case "lt", "<":
		return value < threshold
	case "lte", "<=":
		return value <= threshold
	case "eq", "==":
		return value == threshold
	case "ne", "!=":
		return value != threshold
	default:
		return false
	}
}

// triggerAlert creates and sends a new alert.
func (am *AlertManager) triggerAlert(
	ctx context.Context,
	rule *AlertRule,
	value, threshold float64,
) {
	// Check cooldown
	if cooldownTime, exists := am.cooldowns[rule.Name]; exists {
		if time.Since(cooldownTime) < rule.Cooldown {
			return // Still in cooldown period
		}
	}

	alert := &Alert{
		ID:        generateAlertID(rule.Name),
		Name:      rule.Name,
		Level:     rule.Level,
		Message:   rule.Message,
		Component: rule.Component,
		Metric:    rule.Metric,
		Value:     value,
		Threshold: threshold,
		Timestamp: time.Now(),
		Active:    true,
		Count:     1,
		FirstSeen: time.Now(),
		LastSeen:  time.Now(),
		Labels:    copyStringMap(rule.Labels),
		Metadata: map[string]interface{}{
			"condition": rule.Condition,
			"duration":  rule.Duration.String(),
		},
	}

	am.activeAlerts[rule.Name] = alert

	// Log alert
	am.logger.Error(ctx, nil, "Alert triggered",
		"alert_name", alert.Name,
		"level", string(alert.Level),
		"component", alert.Component,
		"metric", alert.Metric,
		"value", alert.Value,
		"threshold", alert.Threshold)

	// Send to channels
	am.sendAlert(ctx, *alert)
}

// resolveAlert marks an alert as resolved.
func (am *AlertManager) resolveAlert(ctx context.Context, alert *Alert) {
	alert.Active = false
	alert.LastSeen = time.Now()

	// Set cooldown
	am.cooldowns[alert.Name] = time.Now()

	am.logger.Info(ctx, "Alert resolved",
		"alert_name", alert.Name,
		"duration", alert.LastSeen.Sub(alert.FirstSeen),
		"count", alert.Count)

	// Send resolution notification
	resolvedAlert := *alert
	resolvedAlert.Message = "RESOLVED: " + alert.Message
	am.sendAlert(ctx, resolvedAlert)

	// Remove from active alerts after some time
	go func() {
		time.Sleep(5 * time.Minute)
		am.mutex.Lock()
		delete(am.activeAlerts, alert.Name)
		am.mutex.Unlock()
	}()
}

// sendAlert sends an alert to all configured channels.
func (am *AlertManager) sendAlert(ctx context.Context, alert Alert) {
	for _, channel := range am.channels {
		go func(ch AlertChannel) {
			if err := ch.Send(ctx, alert); err != nil {
				am.logger.Error(ctx, err, "Failed to send alert",
					"channel", ch.Name(),
					"alert_name", alert.Name)
			}
		}(channel)
	}
}

// GetActiveAlerts returns all currently active alerts.
func (am *AlertManager) GetActiveAlerts() []Alert {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	alerts := make([]Alert, 0, len(am.activeAlerts))
	for _, alert := range am.activeAlerts {
		if alert.Active {
			alerts = append(alerts, *alert)
		}
	}

	return alerts
}

// GetAlertHistory returns alert history (simplified implementation).
func (am *AlertManager) GetAlertHistory(hours int) []Alert {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)
	alerts := make([]Alert, 0)

	for _, alert := range am.activeAlerts {
		if alert.FirstSeen.After(cutoff) {
			alerts = append(alerts, *alert)
		}
	}

	return alerts
}

// HTTPHandler returns an HTTP handler for alerts API.
func (am *AlertManager) HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/alerts":
			am.handleAlertsAPI(w, r)
		case "/alerts/active":
			am.handleActiveAlerts(w, r)
		case "/alerts/history":
			am.handleAlertHistory(w, r)
		case "/alerts/rules":
			am.handleAlertRules(w, r)
		default:
			http.NotFound(w, r)
		}
	}
}

// handleAlertsAPI handles the main alerts API.
func (am *AlertManager) handleAlertsAPI(w http.ResponseWriter, r *http.Request) {
	activeAlerts := am.GetActiveAlerts()

	response := map[string]interface{}{
		"active_count": len(activeAlerts),
		"alerts":       activeAlerts,
		"status":       "ok",
		"timestamp":    time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// handleActiveAlerts handles active alerts endpoint.
func (am *AlertManager) handleActiveAlerts(w http.ResponseWriter, r *http.Request) {
	alerts := am.GetActiveAlerts()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(alerts)
}

// handleAlertHistory handles alert history endpoint.
func (am *AlertManager) handleAlertHistory(w http.ResponseWriter, r *http.Request) {
	hours := 24 // Default to 24 hours
	if h := r.URL.Query().Get("hours"); h != "" {
		if parsed, err := time.ParseDuration(h + "h"); err == nil {
			hours = int(parsed.Hours())
		}
	}

	alerts := am.GetAlertHistory(hours)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(alerts)
}

// handleAlertRules handles alert rules endpoint.
func (am *AlertManager) handleAlertRules(w http.ResponseWriter, r *http.Request) {
	am.mutex.RLock()
	rules := make([]*AlertRule, 0, len(am.rules))
	for _, rule := range am.rules {
		rules = append(rules, rule)
	}
	am.mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rules)
}

// Utility functions

func (am *AlertManager) getMetricKey(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}

	parts := []string{name}
	for k, v := range labels {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}

	return strings.Join(parts, ",")
}

func generateAlertID(ruleName string) string {
	return fmt.Sprintf("%s_%d", ruleName, time.Now().UnixNano())
}

func copyStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}

	copy := make(map[string]string, len(m))
	for k, v := range m {
		copy[k] = v
	}

	return copy
}

// Built-in Alert Channels

// LogChannel sends alerts to the logging system.
type LogChannel struct {
	logger logging.Logger
}

// NewLogChannel creates a log-based alert channel.
func NewLogChannel(logger logging.Logger) *LogChannel {
	return &LogChannel{
		logger: logger.WithComponent("alert_channel"),
	}
}

// Send implements AlertChannel.
func (lc *LogChannel) Send(ctx context.Context, alert Alert) error {
	switch alert.Level {
	case AlertLevelCritical:
		lc.logger.Error(ctx, nil, alert.Message,
			"alert_id", alert.ID,
			"component", alert.Component,
			"metric", alert.Metric,
			"value", alert.Value,
			"threshold", alert.Threshold)
	case AlertLevelWarning:
		lc.logger.Warn(ctx, nil, alert.Message,
			"alert_id", alert.ID,
			"component", alert.Component,
			"metric", alert.Metric,
			"value", alert.Value)
	default:
		lc.logger.Info(ctx, alert.Message,
			"alert_id", alert.ID,
			"component", alert.Component)
	}

	return nil
}

// Name implements AlertChannel.
func (lc *LogChannel) Name() string {
	return "log"
}

// WebhookChannel sends alerts to HTTP webhooks.
type WebhookChannel struct {
	url    string
	client *http.Client
	logger logging.Logger
}

// NewWebhookChannel creates a webhook-based alert channel.
func NewWebhookChannel(url string, logger logging.Logger) *WebhookChannel {
	return &WebhookChannel{
		url: url,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger.WithComponent("webhook_channel"),
	}
}

// Send implements AlertChannel.
func (wc *WebhookChannel) Send(ctx context.Context, alert Alert) error {
	payload := map[string]interface{}{
		"alert":     alert,
		"timestamp": time.Now(),
		"source":    "templar-monitoring",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal alert: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		wc.url,
		strings.NewReader(string(data)),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "templar-monitoring/1.0")

	resp, err := wc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	wc.logger.Info(ctx, "Alert sent via webhook",
		"alert_id", alert.ID,
		"url", wc.url,
		"status", resp.StatusCode)

	return nil
}

// Name implements AlertChannel.
func (wc *WebhookChannel) Name() string {
	return "webhook"
}

// Default Alert Rules

// CreateDefaultAlertRules creates a set of default alert rules for Templar.
func CreateDefaultAlertRules() []*AlertRule {
	return []*AlertRule{
		{
			Name:      "high_error_rate",
			Component: "application",
			Metric:    "templar_errors_total",
			Condition: "gt",
			Threshold: 10,
			Duration:  5 * time.Minute,
			Level:     AlertLevelWarning,
			Message:   "High error rate detected",
			Enabled:   true,
			Cooldown:  10 * time.Minute,
		},
		{
			Name:      "memory_usage_high",
			Component: "system",
			Metric:    "memory_heap_alloc",
			Condition: "gt",
			Threshold: 1024 * 1024 * 1024, // 1GB
			Duration:  2 * time.Minute,
			Level:     AlertLevelWarning,
			Message:   "High memory usage detected",
			Enabled:   true,
			Cooldown:  15 * time.Minute,
		},
		{
			Name:      "goroutine_leak",
			Component: "system",
			Metric:    "goroutines",
			Condition: "gt",
			Threshold: 1000,
			Duration:  5 * time.Minute,
			Level:     AlertLevelCritical,
			Message:   "Potential goroutine leak detected",
			Enabled:   true,
			Cooldown:  30 * time.Minute,
		},
		{
			Name:      "build_failures",
			Component: "build",
			Metric:    "templar_components_built_total",
			Condition: "gt",
			Threshold: 5,
			Duration:  1 * time.Minute,
			Level:     AlertLevelWarning,
			Message:   "Multiple build failures detected",
			Labels:    map[string]string{"status": "error"},
			Enabled:   true,
			Cooldown:  5 * time.Minute,
		},
		{
			Name:      "health_check_failure",
			Component: "health",
			Metric:    "unhealthy_components",
			Condition: "gt",
			Threshold: 0,
			Duration:  30 * time.Second,
			Level:     AlertLevelCritical,
			Message:   "Critical health check failure",
			Enabled:   true,
			Cooldown:  2 * time.Minute,
		},
	}
}
