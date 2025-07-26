package monitoring

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsCollector(t *testing.T) {
	t.Run("create new metrics collector", func(t *testing.T) {
		collector := NewMetricsCollector("test", "")
		assert.NotNil(t, collector)
		assert.Equal(t, "test", collector.prefix)
		assert.True(t, collector.enabled)
	})

	t.Run("counter operations", func(t *testing.T) {
		collector := NewMetricsCollector("test", "")

		// Test counter increment
		collector.Counter("requests", map[string]string{"method": "GET"})
		collector.Counter("requests", map[string]string{"method": "GET"})
		collector.Counter("requests", map[string]string{"method": "POST"})

		metrics := collector.GatherMetrics()

		// Should have 2 different counter metrics
		getCounter := 0
		postCounter := 0
		for _, metric := range metrics {
			if metric.Name == "test_requests" {
				if metric.Labels["method"] == "GET" {
					getCounter = int(metric.Value)
				} else if metric.Labels["method"] == "POST" {
					postCounter = int(metric.Value)
				}
			}
		}

		assert.Equal(t, 2, getCounter)
		assert.Equal(t, 1, postCounter)
	})

	t.Run("gauge operations", func(t *testing.T) {
		collector := NewMetricsCollector("test", "")

		collector.Gauge("memory_usage", 100.5, map[string]string{"type": "heap"})
		collector.Gauge(
			"memory_usage",
			150.7,
			map[string]string{"type": "heap"},
		) // Should overwrite

		metrics := collector.GatherMetrics()

		found := false
		for _, metric := range metrics {
			if metric.Name == "test_memory_usage" && metric.Labels["type"] == "heap" {
				assert.Equal(t, 150.7, metric.Value)
				found = true
				break
			}
		}
		assert.True(t, found, "Should find gauge metric")
	})

	t.Run("histogram operations", func(t *testing.T) {
		collector := NewMetricsCollector("test", "")

		collector.Histogram("request_duration", 0.1, map[string]string{"method": "GET"})
		collector.Histogram("request_duration", 0.5, map[string]string{"method": "GET"})
		collector.Histogram("request_duration", 2.0, map[string]string{"method": "GET"})

		metrics := collector.GatherMetrics()

		// Should have bucket, count, and sum metrics
		bucketMetrics := 0
		countMetrics := 0
		sumMetrics := 0

		for _, metric := range metrics {
			if metric.Name == "test_request_duration_bucket" {
				bucketMetrics++
			} else if metric.Name == "test_request_duration_count" {
				countMetrics++
				assert.Equal(t, 3.0, metric.Value) // 3 observations
			} else if metric.Name == "test_request_duration_sum" {
				sumMetrics++
				assert.Equal(t, 2.6, metric.Value) // 0.1 + 0.5 + 2.0
			}
		}

		assert.Greater(t, bucketMetrics, 0, "Should have bucket metrics")
		assert.Equal(t, 1, countMetrics, "Should have one count metric")
		assert.Equal(t, 1, sumMetrics, "Should have one sum metric")
	})

	t.Run("timer functionality", func(t *testing.T) {
		collector := NewMetricsCollector("test", "")

		timer := collector.Timer("operation", map[string]string{"type": "test"})
		time.Sleep(10 * time.Millisecond)
		timer()

		metrics := collector.GatherMetrics()

		// Should have histogram metrics for duration
		found := false
		for _, metric := range metrics {
			if metric.Name == "test_operation_duration_seconds_count" {
				assert.Equal(t, 1.0, metric.Value)
				found = true
				break
			}
		}
		assert.True(t, found, "Should find timer metric")
	})

	t.Run("timer with context", func(t *testing.T) {
		collector := NewMetricsCollector("test", "")

		ctx := context.Background()
		timer := collector.TimerContext(ctx, "context_operation", map[string]string{"type": "test"})
		time.Sleep(5 * time.Millisecond)
		timer()

		metrics := collector.GatherMetrics()

		// Should have both histogram and gauge metrics
		histogramFound := false
		gaugeFound := false

		for _, metric := range metrics {
			if metric.Name == "test_context_operation_duration_seconds_count" {
				histogramFound = true
			} else if metric.Name == "test_context_operation_last_duration_seconds" {
				gaugeFound = true
			}
		}

		assert.True(t, histogramFound, "Should find histogram metric")
		assert.True(t, gaugeFound, "Should find gauge metric")
	})

	t.Run("flush metrics to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := tmpDir + "/metrics.json"

		collector := NewMetricsCollector("test", outputPath)
		collector.Counter("test_metric", nil)

		err := collector.FlushMetrics()
		require.NoError(t, err)

		// Verify file was created
		_, err = os.Stat(outputPath)
		assert.NoError(t, err)

		// Verify file content
		data, err := os.ReadFile(outputPath)
		require.NoError(t, err)

		var metricsData map[string]interface{}
		err = json.Unmarshal(data, &metricsData)
		require.NoError(t, err)

		assert.Contains(t, metricsData, "timestamp")
		assert.Contains(t, metricsData, "metrics")
		assert.Contains(t, metricsData, "system")
	})
}

func TestHistogram(t *testing.T) {
	t.Run("histogram observations", func(t *testing.T) {
		hist := NewHistogram([]float64{0.1, 0.5, 1.0, 5.0})

		hist.Observe(0.05) // Below 0.1
		hist.Observe(0.2)  // Between 0.1 and 0.5
		hist.Observe(0.8)  // Between 0.5 and 1.0
		hist.Observe(2.0)  // Between 1.0 and 5.0
		hist.Observe(10.0) // Above 5.0

		buckets := hist.GetBuckets()
		assert.Equal(t, int64(1), buckets[0.1]) // 0.05 <= 0.1
		assert.Equal(t, int64(2), buckets[0.5]) // 0.05, 0.2 <= 0.5
		assert.Equal(t, int64(3), buckets[1.0]) // 0.05, 0.2, 0.8 <= 1.0
		assert.Equal(t, int64(4), buckets[5.0]) // 0.05, 0.2, 0.8, 2.0 <= 5.0

		assert.Equal(t, int64(5), hist.GetCount())
		assert.Equal(t, 13.05, hist.GetSum()) // 0.05 + 0.2 + 0.8 + 2.0 + 10.0
	})

	t.Run("concurrent histogram access", func(t *testing.T) {
		hist := NewHistogram(DefaultHistogramBuckets)

		// Simulate concurrent access
		done := make(chan bool)

		for i := 0; i < 10; i++ {
			go func(val float64) {
				hist.Observe(val)
				done <- true
			}(float64(i) * 0.1)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		assert.Equal(t, int64(10), hist.GetCount())
		assert.Equal(t, 4.5, hist.GetSum()) // 0 + 0.1 + 0.2 + ... + 0.9
	})
}

func TestApplicationMetrics(t *testing.T) {
	t.Run("component metrics", func(t *testing.T) {
		collector := NewMetricsCollector("test", "")
		appMetrics := NewApplicationMetrics(collector)

		appMetrics.ComponentScanned("button")
		appMetrics.ComponentScanned("card")
		appMetrics.ComponentBuilt("button", true)
		appMetrics.ComponentBuilt("card", false)

		metrics := collector.GatherMetrics()

		// Verify component scanned metrics
		scannedCount := 0
		builtSuccessCount := 0
		builtErrorCount := 0

		for _, metric := range metrics {
			if metric.Name == "test_components_scanned_total" {
				scannedCount += int(metric.Value)
			} else if metric.Name == "test_components_built_total" {
				if metric.Labels["status"] == "success" {
					builtSuccessCount += int(metric.Value)
				} else if metric.Labels["status"] == "error" {
					builtErrorCount += int(metric.Value)
				}
			}
		}

		assert.Equal(t, 2, scannedCount)
		assert.Equal(t, 1, builtSuccessCount)
		assert.Equal(t, 1, builtErrorCount)
	})

	t.Run("build duration tracking", func(t *testing.T) {
		collector := NewMetricsCollector("test", "")
		appMetrics := NewApplicationMetrics(collector)

		appMetrics.BuildDuration("test_component", 150*time.Millisecond)
		appMetrics.BuildDuration("test_component", 250*time.Millisecond)

		metrics := collector.GatherMetrics()

		// Should have histogram metrics
		found := false
		for _, metric := range metrics {
			if metric.Name == "test_build_duration_seconds_count" {
				assert.Equal(t, 2.0, metric.Value)
				found = true
				break
			}
		}
		assert.True(t, found, "Should find build duration metric")
	})

	t.Run("server request tracking", func(t *testing.T) {
		collector := NewMetricsCollector("test", "")
		appMetrics := NewApplicationMetrics(collector)

		appMetrics.ServerRequest("GET", "/api/components", 200)
		appMetrics.ServerRequest("POST", "/api/build", 500)

		metrics := collector.GatherMetrics()

		// Verify HTTP request metrics
		getCount := 0
		postCount := 0

		for _, metric := range metrics {
			if metric.Name == "test_http_requests_total" {
				if metric.Labels["method"] == "GET" && metric.Labels["status"] == "200" {
					getCount = int(metric.Value)
				} else if metric.Labels["method"] == "POST" && metric.Labels["status"] == "500" {
					postCount = int(metric.Value)
				}
			}
		}

		assert.Equal(t, 1, getCount)
		assert.Equal(t, 1, postCount)
	})

	t.Run("websocket event tracking", func(t *testing.T) {
		collector := NewMetricsCollector("test", "")
		appMetrics := NewApplicationMetrics(collector)

		appMetrics.WebSocketConnection("opened")
		appMetrics.WebSocketConnection("closed")
		appMetrics.WebSocketMessage("reload")

		metrics := collector.GatherMetrics()

		// Verify WebSocket metrics
		connectionCount := 0
		messageCount := 0

		for _, metric := range metrics {
			if metric.Name == "test_websocket_connections_total" {
				connectionCount += int(metric.Value)
			} else if metric.Name == "test_websocket_messages_total" {
				messageCount += int(metric.Value)
			}
		}

		assert.Equal(t, 2, connectionCount) // opened + closed
		assert.Equal(t, 1, messageCount)
	})

	t.Run("cache operation tracking", func(t *testing.T) {
		collector := NewMetricsCollector("test", "")
		appMetrics := NewApplicationMetrics(collector)

		appMetrics.CacheOperation("get", true)  // hit
		appMetrics.CacheOperation("get", false) // miss
		appMetrics.CacheOperation("set", true)  // hit (success)

		metrics := collector.GatherMetrics()

		// Verify cache metrics
		hits := 0
		misses := 0

		for _, metric := range metrics {
			if metric.Name == "test_cache_operations_total" {
				if metric.Labels["result"] == "hit" {
					hits += int(metric.Value)
				} else if metric.Labels["result"] == "miss" {
					misses += int(metric.Value)
				}
			}
		}

		assert.Equal(t, 2, hits)   // get hit + set success
		assert.Equal(t, 1, misses) // get miss
	})

	t.Run("error tracking", func(t *testing.T) {
		collector := NewMetricsCollector("test", "")
		appMetrics := NewApplicationMetrics(collector)

		appMetrics.ErrorOccurred("build", "scanner")
		appMetrics.ErrorOccurred("network", "server")
		appMetrics.ErrorOccurred("build", "scanner") // Same error again

		metrics := collector.GatherMetrics()

		// Verify error metrics
		buildErrors := 0
		networkErrors := 0

		for _, metric := range metrics {
			if metric.Name == "test_errors_total" {
				if metric.Labels["category"] == "build" {
					buildErrors += int(metric.Value)
				} else if metric.Labels["category"] == "network" {
					networkErrors += int(metric.Value)
				}
			}
		}

		assert.Equal(t, 2, buildErrors)
		assert.Equal(t, 1, networkErrors)
	})

	t.Run("custom gauge setting", func(t *testing.T) {
		collector := NewMetricsCollector("test", "")
		appMetrics := NewApplicationMetrics(collector)

		appMetrics.SetGauge("custom_metric", 42.5, map[string]string{"type": "custom"})

		metrics := collector.GatherMetrics()

		found := false
		for _, metric := range metrics {
			if metric.Name == "test_custom_metric" && metric.Labels["type"] == "custom" {
				assert.Equal(t, 42.5, metric.Value)
				found = true
				break
			}
		}
		assert.True(t, found, "Should find custom gauge metric")
	})

	t.Run("uptime metric from collector interface", func(t *testing.T) {
		collector := NewMetricsCollector("test", "")
		appMetrics := NewApplicationMetrics(collector)

		metrics := appMetrics.Collect()

		found := false
		for _, metric := range metrics {
			if metric.Name == "test_uptime_seconds" {
				assert.Greater(t, metric.Value, 0.0)
				assert.Equal(t, MetricTypeGauge, metric.Type)
				found = true
				break
			}
		}
		assert.True(t, found, "Should find uptime metric")
		assert.Equal(t, "application_metrics", appMetrics.Name())
	})
}

func TestMetricsCollectorStartStop(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := tmpDir + "/metrics.json"

	collector := NewMetricsCollector("test", outputPath)
	collector.flushPeriod = 50 * time.Millisecond // Fast flush for testing

	collector.Start()

	// Add some metrics
	collector.Counter("test_counter", nil)

	// Wait for flush
	time.Sleep(100 * time.Millisecond)

	collector.Stop()

	// Verify metrics file was created
	_, err := os.Stat(outputPath)
	assert.NoError(t, err)
}

func TestMetricsCollectorDisabled(t *testing.T) {
	collector := NewMetricsCollector("test", "")
	collector.enabled = false

	// Operations should not panic when disabled
	collector.Counter("test", nil)
	collector.Gauge("test", 1.0, nil)
	collector.Histogram("test", 1.0, nil)

	metrics := collector.GatherMetrics()
	assert.Empty(t, metrics, "Should have no metrics when disabled")
}

func TestMetricsKeyGeneration(t *testing.T) {
	collector := NewMetricsCollector("test", "")

	t.Run("key without labels", func(t *testing.T) {
		key := collector.getKey("metric_name", nil)
		assert.Equal(t, "metric_name", key)
	})

	t.Run("key with labels", func(t *testing.T) {
		labels := map[string]string{
			"method": "GET",
			"status": "200",
		}
		key := collector.getKey("metric_name", labels)
		// Key should contain all label pairs (order may vary)
		assert.Contains(t, key, "metric_name")
		assert.Contains(t, key, "method_GET")
		assert.Contains(t, key, "status_200")
	})
}

func TestMetricsSystemInfo(t *testing.T) {
	collector := NewMetricsCollector("test", "")
	systemInfo := collector.getSystemMetrics()

	assert.Contains(t, systemInfo, "golang")
	assert.Contains(t, systemInfo, "process")

	golangInfo := systemInfo["golang"].(map[string]interface{})
	assert.Contains(t, golangInfo, "goroutines")
	assert.Contains(t, golangInfo, "memory_alloc")
	assert.Contains(t, golangInfo, "gc_runs")

	processInfo := systemInfo["process"].(map[string]interface{})
	assert.Contains(t, processInfo, "pid")
}
