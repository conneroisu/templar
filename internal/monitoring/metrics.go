package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// MetricType represents different types of metrics.
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

// Metric represents a single metric measurement.
type Metric struct {
	Name      string                 `json:"name"`
	Type      MetricType             `json:"type"`
	Value     float64                `json:"value"`
	Labels    map[string]string      `json:"labels,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Help      string                 `json:"help,omitempty"`
	Unit      string                 `json:"unit,omitempty"`
	Tags      map[string]interface{} `json:"tags,omitempty"`
}

// MetricsCollector collects and manages application metrics.
type MetricsCollector struct {
	metrics     map[string]*Metric
	counters    map[string]*int64
	gauges      map[string]*float64
	histograms  map[string]*Histogram
	mutex       sync.RWMutex
	prefix      string
	enabled     bool
	collectors  []MetricCollector
	outputPath  string
	flushPeriod time.Duration
	stopChan    chan struct{}
	wg          sync.WaitGroup
}

// MetricCollector interface for custom metric collectors.
type MetricCollector interface {
	Collect() []Metric
	Name() string
}

// Histogram tracks distribution of values.
type Histogram struct {
	buckets map[float64]int64
	count   int64
	sum     float64
	mutex   sync.RWMutex
}

// HistogramBuckets defines histogram bucket boundaries.
var DefaultHistogramBuckets = []float64{
	0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
}

// NewMetricsCollector creates a new metrics collector.
func NewMetricsCollector(prefix string, outputPath string) *MetricsCollector {
	return &MetricsCollector{
		metrics:     make(map[string]*Metric),
		counters:    make(map[string]*int64),
		gauges:      make(map[string]*float64),
		histograms:  make(map[string]*Histogram),
		prefix:      prefix,
		enabled:     true,
		outputPath:  outputPath,
		flushPeriod: 30 * time.Second,
		stopChan:    make(chan struct{}),
	}
}

// Start begins background metric collection and flushing.
func (mc *MetricsCollector) Start() {
	if !mc.enabled {
		return
	}

	mc.wg.Add(1)
	go mc.flushLoop()
}

// Stop stops the metrics collector.
func (mc *MetricsCollector) Stop() {
	// Only close if not already closed
	select {
	case <-mc.stopChan:
		// Already closed
	default:
		close(mc.stopChan)
	}
	mc.wg.Wait()
}

// flushLoop periodically flushes metrics to disk.
func (mc *MetricsCollector) flushLoop() {
	defer mc.wg.Done()

	ticker := time.NewTicker(mc.flushPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = mc.FlushMetrics()
		case <-mc.stopChan:
			_ = mc.FlushMetrics() // Final flush

			return
		}
	}
}

// Counter increments a counter metric.
func (mc *MetricsCollector) Counter(name string, labels map[string]string) {
	if !mc.enabled {
		return
	}

	fullName := mc.getFullName(name)
	key := mc.getKey(fullName, labels)

	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if counter, exists := mc.counters[key]; exists {
		atomic.AddInt64(counter, 1)
	} else {
		var counter int64 = 1
		mc.counters[key] = &counter
		mc.metrics[key] = &Metric{
			Name:      fullName,
			Type:      MetricTypeCounter,
			Labels:    labels,
			Timestamp: time.Now(),
		}
	}
}

// CounterAdd adds a value to a counter metric.
func (mc *MetricsCollector) CounterAdd(name string, value float64, labels map[string]string) {
	if !mc.enabled {
		return
	}

	fullName := mc.getFullName(name)
	key := mc.getKey(fullName, labels)

	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if counter, exists := mc.counters[key]; exists {
		atomic.AddInt64(counter, int64(value))
	} else {
		counter := int64(value)
		mc.counters[key] = &counter
		mc.metrics[key] = &Metric{
			Name:      fullName,
			Type:      MetricTypeCounter,
			Labels:    labels,
			Timestamp: time.Now(),
		}
	}
}

// Gauge sets a gauge metric value.
func (mc *MetricsCollector) Gauge(name string, value float64, labels map[string]string) {
	if !mc.enabled {
		return
	}

	fullName := mc.getFullName(name)
	key := mc.getKey(fullName, labels)

	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if gauge, exists := mc.gauges[key]; exists {
		*gauge = value
	} else {
		gauge := value
		mc.gauges[key] = &gauge
		mc.metrics[key] = &Metric{
			Name:      fullName,
			Type:      MetricTypeGauge,
			Labels:    labels,
			Timestamp: time.Now(),
		}
	}
}

// Histogram observes a value in a histogram.
func (mc *MetricsCollector) Histogram(name string, value float64, labels map[string]string) {
	if !mc.enabled {
		return
	}

	fullName := mc.getFullName(name)
	key := mc.getKey(fullName, labels)

	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	hist, exists := mc.histograms[key]
	if !exists {
		hist = NewHistogram(DefaultHistogramBuckets)
		mc.histograms[key] = hist
		mc.metrics[key] = &Metric{
			Name:      fullName,
			Type:      MetricTypeHistogram,
			Labels:    labels,
			Timestamp: time.Now(),
		}
	}

	hist.Observe(value)
}

// Timer measures operation duration.
func (mc *MetricsCollector) Timer(name string, labels map[string]string) func() {
	start := time.Now()

	return func() {
		duration := time.Since(start).Seconds()
		mc.Histogram(name+"_duration_seconds", duration, labels)
	}
}

// TimerContext measures operation duration with context.
func (mc *MetricsCollector) TimerContext(
	ctx context.Context,
	name string,
	labels map[string]string,
) func() {
	start := time.Now()

	return func() {
		duration := time.Since(start).Seconds()
		mc.Histogram(name+"_duration_seconds", duration, labels)

		// Also track as gauge for current operations
		mc.Gauge(name+"_last_duration_seconds", duration, labels)
	}
}

// RegisterCollector adds a custom metric collector.
func (mc *MetricsCollector) RegisterCollector(collector MetricCollector) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.collectors = append(mc.collectors, collector)
}

// FlushMetrics writes current metrics to output.
func (mc *MetricsCollector) FlushMetrics() error {
	if !mc.enabled || mc.outputPath == "" {
		return nil
	}

	allMetrics := mc.GatherMetrics()

	// Create output directory if it doesn't exist
	dir := filepath.Dir(mc.outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create metrics directory: %w", err)
	}

	// Write metrics to file
	file, err := os.OpenFile(mc.outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open metrics file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	metricsData := map[string]interface{}{
		"timestamp": time.Now(),
		"metrics":   allMetrics,
		"system":    mc.getSystemMetrics(),
	}

	if err := encoder.Encode(metricsData); err != nil {
		return fmt.Errorf("failed to encode metrics: %w", err)
	}

	return nil
}

// GatherMetrics collects all current metrics.
func (mc *MetricsCollector) GatherMetrics() []Metric {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	var allMetrics []Metric

	// Collect counters
	for key, counter := range mc.counters {
		if metric, exists := mc.metrics[key]; exists {
			metricCopy := *metric
			metricCopy.Value = float64(atomic.LoadInt64(counter))
			metricCopy.Timestamp = time.Now()
			allMetrics = append(allMetrics, metricCopy)
		}
	}

	// Collect gauges
	for key, gauge := range mc.gauges {
		if metric, exists := mc.metrics[key]; exists {
			metricCopy := *metric
			metricCopy.Value = *gauge
			metricCopy.Timestamp = time.Now()
			allMetrics = append(allMetrics, metricCopy)
		}
	}

	// Collect histograms
	for key, hist := range mc.histograms {
		if metric, exists := mc.metrics[key]; exists {
			// Create multiple metrics for histogram buckets
			buckets := hist.GetBuckets()
			for bucket, count := range buckets {
				metricCopy := *metric
				metricCopy.Name = metricCopy.Name + "_bucket"
				metricCopy.Value = float64(count)
				metricCopy.Timestamp = time.Now()
				if metricCopy.Labels == nil {
					metricCopy.Labels = make(map[string]string)
				}
				metricCopy.Labels["le"] = fmt.Sprintf("%.3f", bucket)
				allMetrics = append(allMetrics, metricCopy)
			}

			// Add count and sum metrics
			metricCopy := *metric
			metricCopy.Name = metricCopy.Name + "_count"
			metricCopy.Value = float64(hist.GetCount())
			metricCopy.Timestamp = time.Now()
			allMetrics = append(allMetrics, metricCopy)

			metricCopy = *metric
			metricCopy.Name = metricCopy.Name + "_sum"
			metricCopy.Value = hist.GetSum()
			metricCopy.Timestamp = time.Now()
			allMetrics = append(allMetrics, metricCopy)
		}
	}

	// Collect from registered collectors
	for _, collector := range mc.collectors {
		collectorMetrics := collector.Collect()
		allMetrics = append(allMetrics, collectorMetrics...)
	}

	return allMetrics
}

// getSystemMetrics collects system-level metrics.
func (mc *MetricsCollector) getSystemMetrics() map[string]interface{} {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	return map[string]interface{}{
		"golang": map[string]interface{}{
			"goroutines":        runtime.NumGoroutine(),
			"memory_alloc":      mem.Alloc,
			"memory_total":      mem.TotalAlloc,
			"memory_sys":        mem.Sys,
			"memory_heap_alloc": mem.HeapAlloc,
			"memory_heap_sys":   mem.HeapSys,
			"gc_runs":           mem.NumGC,
			"gc_pause_ns":       mem.PauseNs[(mem.NumGC+255)%256],
		},
		"process": map[string]interface{}{
			"pid": os.Getpid(),
		},
	}
}

// getFullName returns the full metric name with prefix.
func (mc *MetricsCollector) getFullName(name string) string {
	if mc.prefix == "" {
		return name
	}

	return mc.prefix + "_" + name
}

// getKey generates a unique key for a metric with labels.
func (mc *MetricsCollector) getKey(name string, labels map[string]string) string {
	key := name
	if labels != nil {
		for k, v := range labels {
			key += fmt.Sprintf("_%s_%s", k, v)
		}
	}

	return key
}

// NewHistogram creates a new histogram with the given buckets.
func NewHistogram(buckets []float64) *Histogram {
	hist := &Histogram{
		buckets: make(map[float64]int64),
	}

	for _, bucket := range buckets {
		hist.buckets[bucket] = 0
	}

	return hist
}

// Observe adds an observation to the histogram.
func (h *Histogram) Observe(value float64) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.count++
	h.sum += value

	for bucket := range h.buckets {
		if value <= bucket {
			h.buckets[bucket]++
		}
	}
}

// GetBuckets returns the histogram buckets.
func (h *Histogram) GetBuckets() map[float64]int64 {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	buckets := make(map[float64]int64)
	for k, v := range h.buckets {
		buckets[k] = v
	}

	return buckets
}

// GetCount returns the total observation count.
func (h *Histogram) GetCount() int64 {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	return h.count
}

// GetSum returns the sum of all observations.
func (h *Histogram) GetSum() float64 {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	return h.sum
}

// ApplicationMetrics collects application-specific metrics.
type ApplicationMetrics struct {
	collector *MetricsCollector
}

// NewApplicationMetrics creates application metrics collector.
func NewApplicationMetrics(collector *MetricsCollector) *ApplicationMetrics {
	return &ApplicationMetrics{
		collector: collector,
	}
}

// ComponentScanned increments component scan counter.
func (am *ApplicationMetrics) ComponentScanned(componentType string) {
	am.collector.Counter("components_scanned_total", map[string]string{
		"type": componentType,
	})
}

// ComponentBuilt increments component build counter.
func (am *ApplicationMetrics) ComponentBuilt(componentName string, success bool) {
	status := "success"
	if !success {
		status = "error"
	}

	am.collector.Counter("components_built_total", map[string]string{
		"component": componentName,
		"status":    status,
	})
}

// BuildDuration records build duration.
func (am *ApplicationMetrics) BuildDuration(componentName string, duration time.Duration) {
	am.collector.Histogram("build_duration_seconds", duration.Seconds(), map[string]string{
		"component": componentName,
	})
}

// ServerRequest increments server request counter.
func (am *ApplicationMetrics) ServerRequest(method, path string, statusCode int) {
	am.collector.Counter("http_requests_total", map[string]string{
		"method": method,
		"path":   path,
		"status": strconv.Itoa(statusCode),
	})
}

// WebSocketConnection tracks WebSocket connections.
func (am *ApplicationMetrics) WebSocketConnection(action string) {
	am.collector.Counter("websocket_connections_total", map[string]string{
		"action": action, // "opened", "closed", "error"
	})
}

// WebSocketMessage tracks WebSocket messages.
func (am *ApplicationMetrics) WebSocketMessage(messageType string) {
	am.collector.Counter("websocket_messages_total", map[string]string{
		"type": messageType,
	})
}

// FileWatcherEvent tracks file watcher events.
func (am *ApplicationMetrics) FileWatcherEvent(eventType string) {
	am.collector.Counter("file_watcher_events_total", map[string]string{
		"type": eventType,
	})
}

// CacheOperation tracks cache operations.
func (am *ApplicationMetrics) CacheOperation(operation string, hit bool) {
	hitStr := "miss"
	if hit {
		hitStr = "hit"
	}

	am.collector.Counter("cache_operations_total", map[string]string{
		"operation": operation,
		"result":    hitStr,
	})
}

// ErrorOccurred tracks errors by category and component.
func (am *ApplicationMetrics) ErrorOccurred(category, component string) {
	am.collector.Counter("errors_total", map[string]string{
		"category":  category,
		"component": component,
	})
}

// SetGauge sets a gauge metric value.
func (am *ApplicationMetrics) SetGauge(name string, value float64, labels map[string]string) {
	am.collector.Gauge(name, value, labels)
}

// Collect implements MetricCollector interface.
func (am *ApplicationMetrics) Collect() []Metric {
	// This method can be used to export additional computed metrics
	return []Metric{
		{
			Name:      am.collector.getFullName("uptime_seconds"),
			Type:      MetricTypeGauge,
			Value:     time.Since(startTime).Seconds(),
			Timestamp: time.Now(),
			Help:      "Application uptime in seconds",
		},
	}
}

// Name returns the collector name.
func (am *ApplicationMetrics) Name() string {
	return "application_metrics"
}

// Global start time for uptime calculation
// startTime is defined in health.go
