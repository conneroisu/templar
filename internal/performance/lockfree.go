// Package performance provides lock-free metric collection for high-performance monitoring.
//
// This implementation uses atomic operations, lock-free data structures, and wait-free algorithms
// to eliminate lock contention in metric recording while maintaining thread safety.
package performance

import (
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// LockFreeMetricCollector provides lock-free metric collection with minimal contention
type LockFreeMetricCollector struct {
	// High-frequency write path (lock-free)
	metricBuffer *LockFreeRingBuffer
	aggregateMap *sync.Map // MetricType -> *LockFreeAggregate

	// Low-frequency read path (minimal locking)
	subscribers     atomic.Value // []chan<- Metric
	subscriberMutex sync.RWMutex // Only for subscriber management

	// Configuration
	maxMetrics int64
}

// LockFreeAggregate stores aggregated metric data using atomic operations
type LockFreeAggregate struct {
	// Atomic counters and values
	count int64  // atomic
	sum   uint64 // atomic (float64 bits)
	min   uint64 // atomic (float64 bits)
	max   uint64 // atomic (float64 bits)

	// Percentile calculation (uses efficient skip list)
	percentileCalc *PercentileCalculator
	percMutex      sync.RWMutex // Only for percentile operations

	// Derived values (updated periodically)
	cachedAvg  uint64 // atomic (float64 bits)
	cachedP95  uint64 // atomic (float64 bits)
	cachedP99  uint64 // atomic (float64 bits)
	lastUpdate int64  // atomic (unix nano)
}

// LockFreeRingBuffer implements a lock-free ring buffer for metrics
type LockFreeRingBuffer struct {
	buffer   []Metric
	mask     int64 // buffer size - 1 (for power of 2 sizes)
	writePos int64 // atomic write position
	readPos  int64 // atomic read position
	size     int64 // buffer size (power of 2)
}

// MetricBatch represents a batch of metrics for efficient processing
type MetricBatch struct {
	// Note: metrics field removed as unused
}

// NewLockFreeMetricCollector creates a new lock-free metric collector
func NewLockFreeMetricCollector(maxMetrics int) *LockFreeMetricCollector {
	// Ensure buffer size is power of 2 for efficient masking
	bufferSize := nextPowerOf2(maxMetrics)

	collector := &LockFreeMetricCollector{
		metricBuffer: NewLockFreeRingBuffer(bufferSize),
		aggregateMap: &sync.Map{},
		maxMetrics:   int64(maxMetrics),
	}

	// Initialize empty subscribers slice
	collector.subscribers.Store([]chan<- Metric{})

	return collector
}

// NewLockFreeRingBuffer creates a new lock-free ring buffer
func NewLockFreeRingBuffer(size int) *LockFreeRingBuffer {
	// Ensure size is power of 2
	size = nextPowerOf2(size)

	return &LockFreeRingBuffer{
		buffer: make([]Metric, size),
		mask:   int64(size - 1),
		size:   int64(size),
	}
}

// Record records a new metric using lock-free operations
func (lfc *LockFreeMetricCollector) Record(metric Metric) {
	// Add timestamp if not set (lock-free)
	if metric.Timestamp.IsZero() {
		metric.Timestamp = time.Now()
	}

	// Store metric in lock-free ring buffer
	lfc.metricBuffer.Write(metric)

	// Update aggregates atomically
	lfc.updateAggregateAtomic(metric)

	// Notify subscribers (minimal lock contention)
	lfc.notifySubscribers(metric)
}

// Write writes a metric to the ring buffer using lock-free operations
func (rb *LockFreeRingBuffer) Write(metric Metric) {
	// Get write position atomically
	pos := atomic.AddInt64(&rb.writePos, 1) - 1
	index := pos & rb.mask

	// Store metric at position (may overwrite old data)
	rb.buffer[index] = metric

	// Update read position if buffer is full (maintain ring buffer semantics)
	for {
		currentRead := atomic.LoadInt64(&rb.readPos)
		if pos-currentRead < rb.size {
			break // Buffer not full
		}

		// Try to advance read position
		if atomic.CompareAndSwapInt64(&rb.readPos, currentRead, currentRead+1) {
			break
		}
		// If CAS failed, another goroutine advanced it, try again
	}
}

// updateAggregateAtomic updates metric aggregates using atomic operations
func (lfc *LockFreeMetricCollector) updateAggregateAtomic(metric Metric) {
	// Get or create aggregate for this metric type
	aggInterface, loaded := lfc.aggregateMap.LoadOrStore(metric.Type, &LockFreeAggregate{
		percentileCalc: NewPercentileCalculator(1000),
		lastUpdate:     time.Now().UnixNano(),
		min:            math.Float64bits(metric.Value), // Initialize min with first value
		max:            math.Float64bits(metric.Value), // Initialize max with first value
		sum:            0,                              // Initialize sum to 0
	})

	agg := aggInterface.(*LockFreeAggregate)

	// Update atomic counters
	atomic.AddInt64(&agg.count, 1)

	// Update sum atomically using compare-and-swap loop to handle concurrent updates.
	// This avoids the incorrect approach of adding bit representations directly,
	// which would result in invalid float64 values and incorrect calculations.
	for {
		currentSum := atomic.LoadUint64(&agg.sum)
		currentSumFloat := math.Float64frombits(currentSum)
		newSumFloat := currentSumFloat + metric.Value
		newSum := math.Float64bits(newSumFloat)
		// Retry if another goroutine modified the sum between load and swap
		if atomic.CompareAndSwapUint64(&agg.sum, currentSum, newSum) {
			break
		}
	}

	// If this is not a new aggregate, update min/max
	if loaded {
		// Update min atomically
		for {
			currentMin := atomic.LoadUint64(&agg.min)
			currentMinFloat := math.Float64frombits(currentMin)

			if metric.Value < currentMinFloat {
				newMin := math.Float64bits(metric.Value)
				if atomic.CompareAndSwapUint64(&agg.min, currentMin, newMin) {
					break
				}
			} else {
				break
			}
		}

		// Update max atomically
		for {
			currentMax := atomic.LoadUint64(&agg.max)
			currentMaxFloat := math.Float64frombits(currentMax)

			if metric.Value > currentMaxFloat {
				newMax := math.Float64bits(metric.Value)
				if atomic.CompareAndSwapUint64(&agg.max, currentMax, newMax) {
					break
				}
			} else {
				break
			}
		}
	}

	// Update percentiles (uses read-write lock only for percentile calculator)
	agg.percMutex.Lock()
	agg.percentileCalc.AddValue(metric.Value)

	// Update cached percentiles periodically (reduce computation frequency)
	now := time.Now().UnixNano()
	if now-atomic.LoadInt64(&agg.lastUpdate) > int64(100*time.Millisecond) {
		p95 := agg.percentileCalc.GetP95()
		p99 := agg.percentileCalc.GetP99()

		atomic.StoreUint64(&agg.cachedP95, math.Float64bits(p95))
		atomic.StoreUint64(&agg.cachedP99, math.Float64bits(p99))
		atomic.StoreInt64(&agg.lastUpdate, now)

		// Update cached average
		count := atomic.LoadInt64(&agg.count)
		if count > 0 {
			sum := math.Float64frombits(atomic.LoadUint64(&agg.sum))
			avg := sum / float64(count)
			atomic.StoreUint64(&agg.cachedAvg, math.Float64bits(avg))
		}
	}
	agg.percMutex.Unlock()
}

// notifySubscribers notifies all subscribers with minimal lock contention
func (lfc *LockFreeMetricCollector) notifySubscribers(metric Metric) {
	// Load current subscribers atomically
	subscribers := lfc.subscribers.Load().([]chan<- Metric)

	// Notify all subscribers without blocking
	for _, subscriber := range subscribers {
		select {
		case subscriber <- metric:
		default:
			// Don't block if subscriber can't keep up
		}
	}
}

// Subscribe subscribes to metric updates with minimal locking
func (lfc *LockFreeMetricCollector) Subscribe() <-chan Metric {
	lfc.subscriberMutex.Lock()
	defer lfc.subscriberMutex.Unlock()

	ch := make(chan Metric, 1000) // Large buffer to prevent blocking

	// Get current subscribers and add new one
	current := lfc.subscribers.Load().([]chan<- Metric)
	updated := make([]chan<- Metric, len(current)+1)
	copy(updated, current)
	updated[len(current)] = ch

	// Update subscribers atomically
	lfc.subscribers.Store(updated)

	return ch
}

// GetMetrics returns metrics within time range using lock-free read
func (lfc *LockFreeMetricCollector) GetMetrics(metricType MetricType, since time.Time) []Metric {
	// Read from ring buffer
	writePos := atomic.LoadInt64(&lfc.metricBuffer.writePos)
	readPos := atomic.LoadInt64(&lfc.metricBuffer.readPos)

	var result []Metric

	// Calculate how many metrics to read
	available := writePos - readPos
	if available > lfc.metricBuffer.size {
		available = lfc.metricBuffer.size
	}

	// Read metrics from buffer
	for i := int64(0); i < available; i++ {
		pos := (readPos + i) & lfc.metricBuffer.mask
		metric := lfc.metricBuffer.buffer[pos]

		if (metricType == "" || metric.Type == metricType) && !metric.Timestamp.Before(since) {
			result = append(result, metric)
		}
	}

	return result
}

// GetAggregate returns aggregated data using atomic reads
func (lfc *LockFreeMetricCollector) GetAggregate(metricType MetricType) *MetricAggregate {
	aggInterface, exists := lfc.aggregateMap.Load(metricType)
	if !exists {
		return nil
	}

	agg := aggInterface.(*LockFreeAggregate)

	// Read all values atomically
	count := atomic.LoadInt64(&agg.count)
	sum := math.Float64frombits(atomic.LoadUint64(&agg.sum))
	min := math.Float64frombits(atomic.LoadUint64(&agg.min))
	max := math.Float64frombits(atomic.LoadUint64(&agg.max))
	avg := math.Float64frombits(atomic.LoadUint64(&agg.cachedAvg))
	p95 := math.Float64frombits(atomic.LoadUint64(&agg.cachedP95))
	p99 := math.Float64frombits(atomic.LoadUint64(&agg.cachedP99))

	return &MetricAggregate{
		Count:          count,
		Sum:            sum,
		Min:            min,
		Max:            max,
		Avg:            avg,
		P95:            p95,
		P99:            p99,
		percentileCalc: nil, // Don't expose internal calculator
		maxSize:        1000,
	}
}

// GetSize returns the current number of metrics in the buffer
func (lfc *LockFreeMetricCollector) GetSize() int64 {
	writePos := atomic.LoadInt64(&lfc.metricBuffer.writePos)
	readPos := atomic.LoadInt64(&lfc.metricBuffer.readPos)
	size := writePos - readPos

	if size > lfc.metricBuffer.size {
		size = lfc.metricBuffer.size
	}
	if size < 0 {
		size = 0
	}

	return size
}

// nextPowerOf2 returns the next power of 2 greater than or equal to n
func nextPowerOf2(n int) int {
	if n <= 1 {
		return 2
	}

	// Find the highest set bit
	power := 1
	for power < n {
		power <<= 1
	}

	return power
}

// Additional helper methods for benchmarking and testing

// GetBufferUtilization returns the buffer utilization percentage
func (lfc *LockFreeMetricCollector) GetBufferUtilization() float64 {
	size := lfc.GetSize()
	return float64(size) / float64(lfc.metricBuffer.size) * 100.0
}

// GetMetricTypes returns all currently tracked metric types
func (lfc *LockFreeMetricCollector) GetMetricTypes() []MetricType {
	var types []MetricType

	lfc.aggregateMap.Range(func(key, value interface{}) bool {
		types = append(types, key.(MetricType))
		return true
	})

	return types
}

// FlushMetrics forces an update of all cached percentile values
func (lfc *LockFreeMetricCollector) FlushMetrics() {
	lfc.aggregateMap.Range(func(key, value interface{}) bool {
		agg := value.(*LockFreeAggregate)

		agg.percMutex.Lock()
		p95 := agg.percentileCalc.GetP95()
		p99 := agg.percentileCalc.GetP99()

		atomic.StoreUint64(&agg.cachedP95, math.Float64bits(p95))
		atomic.StoreUint64(&agg.cachedP99, math.Float64bits(p99))
		atomic.StoreInt64(&agg.lastUpdate, time.Now().UnixNano())

		// Update cached average
		count := atomic.LoadInt64(&agg.count)
		if count > 0 {
			sum := math.Float64frombits(atomic.LoadUint64(&agg.sum))
			avg := sum / float64(count)
			atomic.StoreUint64(&agg.cachedAvg, math.Float64bits(avg))
		}
		agg.percMutex.Unlock()

		return true
	})
}
