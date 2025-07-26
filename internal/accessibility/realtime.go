package accessibility

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/conneroisu/templar/internal/logging"
)

// RealtimeAccessibilityMonitor provides real-time accessibility warnings during preview.
type RealtimeAccessibilityMonitor struct {
	tester        AccessibilityTester
	logger        logging.Logger
	subscribers   map[string]chan AccessibilityUpdate
	subscribersMu sync.RWMutex
	config        RealtimeConfig
}

// RealtimeConfig configures real-time accessibility monitoring.
type RealtimeConfig struct {
	EnableRealTimeWarnings  bool              `json:"enable_real_time_warnings"`
	WarningSeverityLevel    ViolationSeverity `json:"warning_severity_level"`
	CheckInterval           time.Duration     `json:"check_interval"`
	MaxWarningsPerComponent int               `json:"max_warnings_per_component"`
	EnableAutoFixes         bool              `json:"enable_auto_fixes"`
	ShowSuccessMessages     bool              `json:"show_success_messages"`
}

// AccessibilityUpdate represents a real-time accessibility update.
type AccessibilityUpdate struct {
	Type           UpdateType                `json:"type"`
	ComponentName  string                    `json:"component_name"`
	Timestamp      time.Time                 `json:"timestamp"`
	Violations     []AccessibilityViolation  `json:"violations,omitempty"`
	FixedIssues    []string                  `json:"fixed_issues,omitempty"`
	OverallScore   float64                   `json:"overall_score"`
	Message        string                    `json:"message"`
	Suggestions    []AccessibilitySuggestion `json:"suggestions,omitempty"`
	AutoFixApplied bool                      `json:"auto_fix_applied,omitempty"`
}

// UpdateType represents different types of accessibility updates.
type UpdateType string

const (
	UpdateTypeWarning UpdateType = "warning"
	UpdateTypeError   UpdateType = "error"
	UpdateTypeSuccess UpdateType = "success"
	UpdateTypeAutoFix UpdateType = "auto_fix"
	UpdateTypeInfo    UpdateType = "info"
)

// NewRealtimeAccessibilityMonitor creates a new real-time accessibility monitor.
func NewRealtimeAccessibilityMonitor(
	tester AccessibilityTester,
	logger logging.Logger,
	config RealtimeConfig,
) *RealtimeAccessibilityMonitor {
	return &RealtimeAccessibilityMonitor{
		tester:      tester,
		logger:      logger.WithComponent("realtime_accessibility"),
		subscribers: make(map[string]chan AccessibilityUpdate),
		config:      config,
	}
}

// Subscribe subscribes to real-time accessibility updates.
func (monitor *RealtimeAccessibilityMonitor) Subscribe(
	subscriberID string,
) <-chan AccessibilityUpdate {
	monitor.subscribersMu.Lock()
	defer monitor.subscribersMu.Unlock()

	ch := make(chan AccessibilityUpdate, 100) // Buffered channel
	monitor.subscribers[subscriberID] = ch

	monitor.logger.Info(
		context.Background(),
		"New accessibility monitor subscriber",
		"subscriber_id",
		subscriberID,
	)

	return ch
}

// Unsubscribe removes a subscriber from real-time updates.
func (monitor *RealtimeAccessibilityMonitor) Unsubscribe(subscriberID string) {
	monitor.subscribersMu.Lock()
	defer monitor.subscribersMu.Unlock()

	if ch, exists := monitor.subscribers[subscriberID]; exists {
		close(ch)
		delete(monitor.subscribers, subscriberID)
		monitor.logger.Info(
			context.Background(),
			"Accessibility monitor subscriber removed",
			"subscriber_id",
			subscriberID,
		)
	}
}

// CheckComponent performs real-time accessibility check on a component.
func (monitor *RealtimeAccessibilityMonitor) CheckComponent(
	ctx context.Context,
	componentName string,
	props map[string]interface{},
) {
	if !monitor.config.EnableRealTimeWarnings {
		return
	}

	go monitor.performCheck(ctx, componentName, props)
}

// performCheck performs the actual accessibility check in a goroutine.
func (monitor *RealtimeAccessibilityMonitor) performCheck(
	ctx context.Context,
	componentName string,
	props map[string]interface{},
) {
	start := time.Now()

	// Run accessibility test
	report, err := monitor.tester.TestComponent(ctx, componentName, props)
	if err != nil {
		monitor.logger.Warn(
			ctx,
			err,
			"Failed to run real-time accessibility check",
			"component",
			componentName,
		)

		return
	}

	// Filter violations by severity level
	relevantViolations := monitor.filterViolationsBySeverity(report.Violations)

	// Limit number of warnings per component
	if len(relevantViolations) > monitor.config.MaxWarningsPerComponent {
		relevantViolations = relevantViolations[:monitor.config.MaxWarningsPerComponent]
	}

	// Create update based on results
	var update AccessibilityUpdate

	if len(relevantViolations) == 0 {
		if monitor.config.ShowSuccessMessages {
			update = AccessibilityUpdate{
				Type:          UpdateTypeSuccess,
				ComponentName: componentName,
				Timestamp:     time.Now(),
				OverallScore:  report.Summary.OverallScore,
				Message:       "✅ No accessibility issues found in " + componentName,
			}
		} else {
			return // Don't send success updates if disabled
		}
	} else {
		// Determine update type based on most severe violation
		updateType := monitor.getUpdateTypeFromViolations(relevantViolations)

		// Generate combined suggestions
		suggestions := monitor.generateCombinedSuggestions(relevantViolations)

		update = AccessibilityUpdate{
			Type:          updateType,
			ComponentName: componentName,
			Timestamp:     time.Now(),
			Violations:    relevantViolations,
			OverallScore:  report.Summary.OverallScore,
			Message:       monitor.generateUpdateMessage(componentName, relevantViolations),
			Suggestions:   suggestions,
		}

		// Apply auto-fixes if enabled
		if monitor.config.EnableAutoFixes {
			fixedIssues := monitor.attemptAutoFixes(ctx, report.HTMLSnapshot, relevantViolations)
			if len(fixedIssues) > 0 {
				update.AutoFixApplied = true
				update.FixedIssues = fixedIssues
				update.Message += fmt.Sprintf(" (%d issues auto-fixed)", len(fixedIssues))
			}
		}
	}

	// Broadcast update to all subscribers
	monitor.broadcastUpdate(update)

	monitor.logger.Debug(ctx, "Real-time accessibility check completed",
		"component", componentName,
		"violations", len(relevantViolations),
		"score", report.Summary.OverallScore,
		"duration", time.Since(start))
}

// filterViolationsBySeverity filters violations based on configured severity level.
func (monitor *RealtimeAccessibilityMonitor) filterViolationsBySeverity(
	violations []AccessibilityViolation,
) []AccessibilityViolation {
	if monitor.config.WarningSeverityLevel == SeverityInfo {
		return violations // Include all
	}

	filtered := []AccessibilityViolation{}
	for _, violation := range violations {
		switch monitor.config.WarningSeverityLevel {
		case SeverityError:
			if violation.Severity == SeverityError {
				filtered = append(filtered, violation)
			}
		case SeverityWarning:
			if violation.Severity == SeverityError || violation.Severity == SeverityWarning {
				filtered = append(filtered, violation)
			}
		}
	}

	return filtered
}

// getUpdateTypeFromViolations determines the update type based on violation severity.
func (monitor *RealtimeAccessibilityMonitor) getUpdateTypeFromViolations(
	violations []AccessibilityViolation,
) UpdateType {
	hasError := false
	hasWarning := false

	for _, violation := range violations {
		switch violation.Severity {
		case SeverityError:
			hasError = true
		case SeverityWarning:
			hasWarning = true
		}
	}

	if hasError {
		return UpdateTypeError
	}
	if hasWarning {
		return UpdateTypeWarning
	}

	return UpdateTypeInfo
}

// generateUpdateMessage creates a user-friendly message for the update.
func (monitor *RealtimeAccessibilityMonitor) generateUpdateMessage(
	componentName string,
	violations []AccessibilityViolation,
) string {
	if len(violations) == 0 {
		return fmt.Sprintf("✅ %s passes accessibility checks", componentName)
	}

	criticalCount := 0
	seriousCount := 0

	for _, violation := range violations {
		switch violation.Impact {
		case ImpactCritical:
			criticalCount++
		case ImpactSerious:
			seriousCount++
		}
	}

	if criticalCount > 0 {
		return fmt.Sprintf(
			"🚨 %s has %d critical accessibility issue(s)",
			componentName,
			criticalCount,
		)
	}

	if seriousCount > 0 {
		return fmt.Sprintf(
			"⚠️ %s has %d serious accessibility issue(s)",
			componentName,
			seriousCount,
		)
	}

	return fmt.Sprintf("ℹ️ %s has %d accessibility issue(s)", componentName, len(violations))
}

// generateCombinedSuggestions creates combined suggestions from multiple violations.
func (monitor *RealtimeAccessibilityMonitor) generateCombinedSuggestions(
	violations []AccessibilityViolation,
) []AccessibilitySuggestion {
	suggestionMap := make(map[string]*AccessibilitySuggestion)

	// Collect all suggestions and merge similar ones
	for _, violation := range violations {
		for _, suggestion := range violation.Suggestions {
			key := fmt.Sprintf("%s_%s", suggestion.Type, suggestion.Title)

			if existing, exists := suggestionMap[key]; exists {
				// Merge with existing suggestion (lower priority = higher importance)
				if suggestion.Priority < existing.Priority {
					existing.Priority = suggestion.Priority
				}
			} else {
				suggestionCopy := suggestion
				suggestionMap[key] = &suggestionCopy
			}
		}
	}

	// Convert map back to slice
	suggestions := []AccessibilitySuggestion{}
	for _, suggestion := range suggestionMap {
		suggestions = append(suggestions, *suggestion)
	}

	// Sort by priority (lower number = higher priority)
	for i := range len(suggestions) - 1 {
		for j := i + 1; j < len(suggestions); j++ {
			if suggestions[i].Priority > suggestions[j].Priority {
				suggestions[i], suggestions[j] = suggestions[j], suggestions[i]
			}
		}
	}

	// Limit to top 5 suggestions for real-time display
	maxSuggestions := 5
	if len(suggestions) > maxSuggestions {
		suggestions = suggestions[:maxSuggestions]
	}

	return suggestions
}

// attemptAutoFixes tries to automatically fix accessibility issues.
func (monitor *RealtimeAccessibilityMonitor) attemptAutoFixes(
	ctx context.Context,
	html string,
	violations []AccessibilityViolation,
) []string {
	fixedIssues := []string{}

	// Only attempt fixes for violations that can be auto-fixed
	autoFixableViolations := []AccessibilityViolation{}
	for _, violation := range violations {
		if violation.CanAutoFix {
			autoFixableViolations = append(autoFixableViolations, violation)
		}
	}

	if len(autoFixableViolations) == 0 {
		return fixedIssues
	}

	// Apply auto-fixes
	if engine, ok := monitor.tester.(*ComponentAccessibilityTester); ok {
		fixedHTML, err := engine.engine.AutoFix(ctx, html, autoFixableViolations)
		if err != nil {
			monitor.logger.Warn(ctx, err, "Failed to apply auto-fixes")

			return fixedIssues
		}

		if fixedHTML != html {
			for _, violation := range autoFixableViolations {
				fixedIssues = append(fixedIssues, violation.Rule)
			}
		}
	}

	return fixedIssues
}

// broadcastUpdate sends an update to all subscribers.
func (monitor *RealtimeAccessibilityMonitor) broadcastUpdate(update AccessibilityUpdate) {
	monitor.subscribersMu.RLock()
	defer monitor.subscribersMu.RUnlock()

	if len(monitor.subscribers) == 0 {
		return
	}

	// Convert update to JSON for WebSocket transmission
	updateJSON, err := json.Marshal(update)
	if err != nil {
		monitor.logger.Error(context.Background(), err, "Failed to marshal accessibility update")

		return
	}

	monitor.logger.Debug(context.Background(), "Broadcasting accessibility update",
		"subscribers", len(monitor.subscribers),
		"update_type", string(update.Type),
		"component", update.ComponentName)

	// Send to all subscribers (non-blocking)
	for subscriberID, ch := range monitor.subscribers {
		select {
		case ch <- update:
			// Success
		default:
			// Channel full, skip this subscriber
			monitor.logger.Warn(context.Background(), nil,
				"Accessibility update channel full, skipping subscriber",
				"subscriber_id", subscriberID)
		}
	}

	_ = updateJSON // Use the JSON for WebSocket transmission in integration
}

// StartPeriodicChecks starts periodic accessibility checks for active components.
func (monitor *RealtimeAccessibilityMonitor) StartPeriodicChecks(
	ctx context.Context,
	activeComponents map[string]map[string]interface{},
) {
	if !monitor.config.EnableRealTimeWarnings || monitor.config.CheckInterval <= 0 {
		return
	}

	ticker := time.NewTicker(monitor.config.CheckInterval)
	defer ticker.Stop()

	monitor.logger.Info(ctx, "Started periodic accessibility checks",
		"interval", monitor.config.CheckInterval,
		"components", len(activeComponents))

	for {
		select {
		case <-ctx.Done():
			monitor.logger.Info(ctx, "Stopped periodic accessibility checks")

			return
		case <-ticker.C:
			for componentName, props := range activeComponents {
				// Check each component (in separate goroutines)
				go monitor.performCheck(ctx, componentName, props)
			}
		}
	}
}

// GetAccessibilityStatus returns the current accessibility status for all monitored components.
func (monitor *RealtimeAccessibilityMonitor) GetAccessibilityStatus(
	ctx context.Context,
) (*AccessibilityStatus, error) {
	// This would typically cache recent results
	// For now, return basic status

	status := &AccessibilityStatus{
		MonitoringEnabled: monitor.config.EnableRealTimeWarnings,
		ActiveSubscribers: len(monitor.subscribers),
		CheckInterval:     monitor.config.CheckInterval,
		LastCheckTime:     time.Now(),
		ComponentStatuses: make(map[string]ComponentAccessibilityStatus),
	}

	return status, nil
}

// AccessibilityStatus represents the overall accessibility monitoring status.
type AccessibilityStatus struct {
	MonitoringEnabled bool                                    `json:"monitoring_enabled"`
	ActiveSubscribers int                                     `json:"active_subscribers"`
	CheckInterval     time.Duration                           `json:"check_interval"`
	LastCheckTime     time.Time                               `json:"last_check_time"`
	ComponentStatuses map[string]ComponentAccessibilityStatus `json:"component_statuses"`
}

// ComponentAccessibilityStatus represents the accessibility status of a single component.
type ComponentAccessibilityStatus struct {
	ComponentName      string    `json:"component_name"`
	LastChecked        time.Time `json:"last_checked"`
	OverallScore       float64   `json:"overall_score"`
	ViolationCount     int       `json:"violation_count"`
	CriticalViolations int       `json:"critical_violations"`
	HighestWCAGLevel   WCAGLevel `json:"highest_wcag_level"`
	Status             string    `json:"status"` // "healthy", "warning", "error"
}

// Shutdown gracefully shuts down the real-time monitor.
func (monitor *RealtimeAccessibilityMonitor) Shutdown() {
	monitor.subscribersMu.Lock()
	defer monitor.subscribersMu.Unlock()

	// Close all subscriber channels
	for subscriberID, ch := range monitor.subscribers {
		close(ch)
		monitor.logger.Info(context.Background(), "Closed accessibility monitor subscriber channel",
			"subscriber_id", subscriberID)
	}

	monitor.subscribers = make(map[string]chan AccessibilityUpdate)
	monitor.logger.Info(context.Background(), "Real-time accessibility monitor shutdown complete")
}
