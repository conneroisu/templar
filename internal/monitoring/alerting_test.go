package monitoring

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAlertManager(t *testing.T) {
	t.Run("add and remove rules", func(t *testing.T) {
		logger := logging.NewLogger(logging.DefaultConfig())
		am := NewAlertManager(logger)
		rule := &AlertRule{
			Name:      "test_rule",
			Component: "test",
			Metric:    "test_metric",
			Condition: "gt",
			Threshold: 10.0,
			Level:     AlertLevelWarning,
			Message:   "Test alert",
			Enabled:   true,
			Cooldown:  1 * time.Minute,
		}

		am.AddRule(rule)
		assert.Contains(t, am.rules, "test_rule")

		am.RemoveRule("test_rule")
		assert.NotContains(t, am.rules, "test_rule")
	})

	t.Run("evaluate condition", func(t *testing.T) {
		logger := logging.NewLogger(logging.DefaultConfig())
		am := NewAlertManager(logger)
		tests := []struct {
			condition string
			value     float64
			threshold float64
			expected  bool
		}{
			{"gt", 15.0, 10.0, true},
			{"gt", 5.0, 10.0, false},
			{"lt", 5.0, 10.0, true},
			{"lt", 15.0, 10.0, false},
			{"eq", 10.0, 10.0, true},
			{"eq", 15.0, 10.0, false},
			{"gte", 10.0, 10.0, true},
			{"gte", 15.0, 10.0, true},
			{"gte", 5.0, 10.0, false},
		}

		for _, tt := range tests {
			result := am.evaluateCondition(tt.condition, tt.value, tt.threshold)
			assert.Equal(t, tt.expected, result,
				"condition %s with value %f and threshold %f",
				tt.condition, tt.value, tt.threshold)
		}
	})

	t.Run("trigger and resolve alerts", func(t *testing.T) {
		logger := logging.NewLogger(logging.DefaultConfig())
		am := NewAlertManager(logger)
		// Create test rule
		rule := &AlertRule{
			Name:      "cpu_high",
			Component: "system",
			Metric:    "cpu_usage",
			Condition: "gt",
			Threshold: 80.0,
			Level:     AlertLevelWarning,
			Message:   "High CPU usage",
			Enabled:   true,
			Cooldown:  1 * time.Second, // Short cooldown for testing
		}
		am.AddRule(rule)

		// Create test channel
		testChannel := &TestChannel{alerts: make([]Alert, 0)}
		am.AddChannel(testChannel)

		// Simulate high CPU
		metrics := []Metric{
			{
				Name:  "cpu_usage",
				Value: 90.0,
			},
		}

		ctx := context.Background()
		am.EvaluateMetrics(ctx, metrics)

		// Check alert was triggered
		activeAlerts := am.GetActiveAlerts()
		assert.Len(t, activeAlerts, 1)
		assert.Equal(t, "cpu_high", activeAlerts[0].Name)
		assert.Equal(t, 90.0, activeAlerts[0].Value)
		assert.True(t, activeAlerts[0].Active)

		// Check alert was sent to channel (wait for async delivery)
		assert.Eventually(t, func() bool {
			testChannel.mutex.Lock()
			defer testChannel.mutex.Unlock()

			return len(testChannel.alerts) == 1
		}, 100*time.Millisecond, 10*time.Millisecond, "Alert should be delivered to channel")

		testChannel.mutex.Lock()
		assert.Equal(t, "cpu_high", testChannel.alerts[0].Name)
		testChannel.mutex.Unlock()

		// Simulate CPU back to normal
		metrics[0].Value = 50.0
		am.EvaluateMetrics(ctx, metrics)

		// Check alert was resolved
		activeAlerts = am.GetActiveAlerts()
		assert.Len(t, activeAlerts, 0) // Should be removed from active alerts

		// Check resolution was sent to channel (wait for async delivery)
		assert.Eventually(t, func() bool {
			testChannel.mutex.Lock()
			defer testChannel.mutex.Unlock()

			return len(testChannel.alerts) == 2
		}, 100*time.Millisecond, 10*time.Millisecond, "Resolution alert should be delivered to channel")

		testChannel.mutex.Lock()
		assert.Contains(t, testChannel.alerts[1].Message, "RESOLVED")
		testChannel.mutex.Unlock()
	})

	t.Run("cooldown mechanism", func(t *testing.T) {
		logger := logging.NewLogger(logging.DefaultConfig())
		am := NewAlertManager(logger)
		rule := &AlertRule{
			Name:      "memory_test",
			Component: "system",
			Metric:    "memory_usage",
			Condition: "gt",
			Threshold: 100.0,
			Level:     AlertLevelCritical,
			Message:   "High memory",
			Enabled:   true,
			Cooldown:  100 * time.Millisecond,
		}
		am.AddRule(rule)

		testChannel := &TestChannel{alerts: make([]Alert, 0)}
		am.AddChannel(testChannel)

		ctx := context.Background()
		metrics := []Metric{{Name: "memory_usage", Value: 150.0}}

		// First alert should trigger
		am.EvaluateMetrics(ctx, metrics)
		assert.Eventually(t, func() bool {
			testChannel.mutex.Lock()
			defer testChannel.mutex.Unlock()

			return len(testChannel.alerts) == 1
		}, 100*time.Millisecond, 10*time.Millisecond, "First alert should be delivered")

		// Resolve and immediately trigger again
		metrics[0].Value = 50.0
		am.EvaluateMetrics(ctx, metrics)
		metrics[0].Value = 150.0
		am.EvaluateMetrics(ctx, metrics)

		// Should not trigger new alert due to cooldown
		// Should have: initial alert + resolution = 2 alerts
		assert.Eventually(t, func() bool {
			testChannel.mutex.Lock()
			defer testChannel.mutex.Unlock()

			return len(testChannel.alerts) == 2
		}, 100*time.Millisecond, 10*time.Millisecond, "Should have initial alert + resolution")

		// Wait for cooldown to expire
		time.Sleep(150 * time.Millisecond)
		am.EvaluateMetrics(ctx, metrics)

		// Now should trigger new alert
		assert.Eventually(t, func() bool {
			testChannel.mutex.Lock()
			defer testChannel.mutex.Unlock()

			return len(testChannel.alerts) == 3
		}, 100*time.Millisecond, 10*time.Millisecond, "Should trigger new alert after cooldown")
	})
}

func TestAlertRules(t *testing.T) {
	t.Run("default rules", func(t *testing.T) {
		rules := CreateDefaultAlertRules()
		assert.Greater(t, len(rules), 0)

		// Check specific rules exist
		ruleNames := make(map[string]bool)
		for _, rule := range rules {
			ruleNames[rule.Name] = true
			assert.NotEmpty(t, rule.Component)
			assert.NotEmpty(t, rule.Metric)
			assert.NotEmpty(t, rule.Condition)
			assert.NotEmpty(t, rule.Message)
			assert.True(t, rule.Enabled)
		}

		assert.True(t, ruleNames["high_error_rate"])
		assert.True(t, ruleNames["memory_usage_high"])
		assert.True(t, ruleNames["goroutine_leak"])
	})
}

func TestLogChannel(t *testing.T) {
	logger := logging.NewLogger(logging.DefaultConfig())
	channel := NewLogChannel(logger)

	t.Run("send alerts", func(t *testing.T) {
		alert := Alert{
			ID:        "test_alert",
			Name:      "test",
			Level:     AlertLevelWarning,
			Message:   "Test alert message",
			Component: "test_component",
			Metric:    "test_metric",
			Value:     100.0,
			Threshold: 80.0,
		}

		err := channel.Send(context.Background(), alert)
		assert.NoError(t, err)
		assert.Equal(t, "log", channel.Name())
	})
}

func TestWebhookChannel(t *testing.T) {
	// Create test server
	var receivedAlert Alert
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		json.NewDecoder(r.Body).Decode(&payload)

		alertData, _ := json.Marshal(payload["alert"])
		json.Unmarshal(alertData, &receivedAlert)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := logging.NewLogger(logging.DefaultConfig())
	channel := NewWebhookChannel(server.URL, logger)

	t.Run("send webhook", func(t *testing.T) {
		alert := Alert{
			ID:        "webhook_test",
			Name:      "webhook_test",
			Level:     AlertLevelCritical,
			Message:   "Webhook test alert",
			Component: "webhook",
			Value:     200.0,
		}

		err := channel.Send(context.Background(), alert)
		assert.NoError(t, err)
		assert.Equal(t, "webhook", channel.Name())
		assert.Equal(t, "webhook_test", receivedAlert.ID)
	})

	t.Run("webhook error handling", func(t *testing.T) {
		// Test with invalid URL
		badChannel := NewWebhookChannel("http://invalid-url-that-does-not-exist", logger)

		alert := Alert{ID: "test", Name: "test", Message: "test"}
		err := badChannel.Send(context.Background(), alert)
		assert.Error(t, err)
	})
}

func TestAlertManagerHTTPHandlers(t *testing.T) {
	logger := logging.NewLogger(logging.DefaultConfig())
	am := NewAlertManager(logger)

	// Add test rule and trigger alert
	rule := &AlertRule{
		Name:      "http_test",
		Component: "test",
		Metric:    "test_metric",
		Condition: "gt",
		Threshold: 50.0,
		Level:     AlertLevelWarning,
		Message:   "HTTP test alert",
		Enabled:   true,
		Cooldown:  1 * time.Minute,
	}
	am.AddRule(rule)

	// Trigger alert
	metrics := []Metric{{Name: "test_metric", Value: 75.0}}
	am.EvaluateMetrics(context.Background(), metrics)

	handler := am.HTTPHandler()

	t.Run("alerts endpoint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/alerts", nil)
		recorder := httptest.NewRecorder()

		handler(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

		var response map[string]interface{}
		err := json.NewDecoder(recorder.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, float64(1), response["active_count"])
		assert.Contains(t, response, "alerts")
		assert.Equal(t, "ok", response["status"])
	})

	t.Run("active alerts endpoint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/alerts/active", nil)
		recorder := httptest.NewRecorder()

		handler(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)

		var alerts []Alert
		err := json.NewDecoder(recorder.Body).Decode(&alerts)
		require.NoError(t, err)

		assert.Len(t, alerts, 1)
		assert.Equal(t, "http_test", alerts[0].Name)
		assert.True(t, alerts[0].Active)
	})

	t.Run("alert history endpoint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/alerts/history?hours=1", nil)
		recorder := httptest.NewRecorder()

		handler(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)

		var alerts []Alert
		err := json.NewDecoder(recorder.Body).Decode(&alerts)
		require.NoError(t, err)

		assert.Len(t, alerts, 1)
	})

	t.Run("alert rules endpoint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/alerts/rules", nil)
		recorder := httptest.NewRecorder()

		handler(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)

		var rules []*AlertRule
		err := json.NewDecoder(recorder.Body).Decode(&rules)
		require.NoError(t, err)

		assert.Len(t, rules, 1)
		assert.Equal(t, "http_test", rules[0].Name)
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/alerts/invalid", nil)
		recorder := httptest.NewRecorder()

		handler(recorder, req)

		assert.Equal(t, http.StatusNotFound, recorder.Code)
	})
}

func TestAlertUtilities(t *testing.T) {
	am := NewAlertManager(logging.NewLogger(logging.DefaultConfig()))

	t.Run("metric key generation", func(t *testing.T) {
		// Test without labels
		key1 := am.getMetricKey("test_metric", nil)
		assert.Equal(t, "test_metric", key1)

		// Test with labels
		labels := map[string]string{
			"component": "scanner",
			"status":    "success",
		}
		key2 := am.getMetricKey("test_metric", labels)
		assert.Contains(t, key2, "test_metric")
		assert.Contains(t, key2, "component=scanner")
		assert.Contains(t, key2, "status=success")
	})

	t.Run("alert ID generation", func(t *testing.T) {
		id1 := generateAlertID("test_rule")
		id2 := generateAlertID("test_rule")

		assert.Contains(t, id1, "test_rule")
		assert.Contains(t, id2, "test_rule")
		assert.NotEqual(t, id1, id2) // Should be unique
	})

	t.Run("copy string map", func(t *testing.T) {
		original := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}

		copied := copyStringMap(original)
		assert.Equal(t, original, copied)

		// Modify original
		original["key3"] = "value3"
		assert.NotEqual(t, original, copied)
		assert.NotContains(t, copied, "key3")
	})
}

func TestAlertIntegration(t *testing.T) {
	t.Run("alert manager with monitor", func(t *testing.T) {
		// Create monitor with alerting
		config := DefaultMonitorConfig()
		config.AlertingEnabled = true
		config.HTTPEnabled = false

		logger := logging.NewLogger(logging.DefaultConfig())
		monitor, err := NewMonitor(config, logger)
		require.NoError(t, err)
		defer monitor.Stop()

		// Create alert manager
		alertManager := NewAlertManager(logger)

		// Add default rules
		for _, rule := range CreateDefaultAlertRules() {
			alertManager.AddRule(rule)
		}

		// Add test channel
		testChannel := &TestChannel{alerts: make([]Alert, 0)}
		alertManager.AddChannel(testChannel)

		// Generate metrics that should trigger alerts
		metrics := []Metric{
			{Name: "templar_errors_total", Value: 15.0}, // Should trigger high_error_rate
			{Name: "goroutines", Value: 1500.0},         // Should trigger goroutine_leak
		}

		alertManager.EvaluateMetrics(context.Background(), metrics)

		// Check alerts were triggered
		activeAlerts := alertManager.GetActiveAlerts()
		assert.Greater(t, len(activeAlerts), 0)

		// Wait for async alert delivery
		assert.Eventually(t, func() bool {
			testChannel.mutex.Lock()
			defer testChannel.mutex.Unlock()

			return len(testChannel.alerts) > 0
		}, 100*time.Millisecond, 10*time.Millisecond, "Alerts should be delivered to channel")
	})
}

// TestChannel is a test implementation of AlertChannel.
type TestChannel struct {
	alerts []Alert
	mutex  sync.Mutex
}

func (tc *TestChannel) Send(ctx context.Context, alert Alert) error {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	tc.alerts = append(tc.alerts, alert)

	return nil
}

func (tc *TestChannel) Name() string {
	return "test"
}

// Benchmarks

func BenchmarkAlertEvaluation(b *testing.B) {
	logger := logging.NewLogger(logging.DefaultConfig())
	am := NewAlertManager(logger)

	// Add rules
	for _, rule := range CreateDefaultAlertRules() {
		am.AddRule(rule)
	}

	// Create test metrics
	metrics := []Metric{
		{Name: "templar_errors_total", Value: 5.0},
		{Name: "memory_heap_alloc", Value: 500000000},
		{Name: "goroutines", Value: 500.0},
		{
			Name:   "templar_components_built_total",
			Value:  2.0,
			Labels: map[string]string{"status": "error"},
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	for range b.N {
		am.EvaluateMetrics(ctx, metrics)
	}
}

func BenchmarkAlertChannelSend(b *testing.B) {
	logger := logging.NewLogger(logging.DefaultConfig())
	channel := NewLogChannel(logger)

	alert := Alert{
		ID:        "bench_alert",
		Name:      "benchmark",
		Level:     AlertLevelWarning,
		Message:   "Benchmark alert",
		Component: "test",
	}

	ctx := context.Background()

	b.ResetTimer()
	for range b.N {
		channel.Send(ctx, alert)
	}
}
