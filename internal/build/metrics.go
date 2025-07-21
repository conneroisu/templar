// Package build provides build metrics tracking and performance monitoring.
package build

import (
	"sync"
	"time"
)

// BuildMetrics tracks build performance
type BuildMetrics struct {
	TotalBuilds      int64
	SuccessfulBuilds int64
	FailedBuilds     int64
	CacheHits        int64
	AverageDuration  time.Duration
	TotalDuration    time.Duration
	mutex            sync.RWMutex
}

// NewBuildMetrics creates a new build metrics tracker
func NewBuildMetrics() *BuildMetrics {
	return &BuildMetrics{}
}

// RecordBuild records a build result in the metrics
func (bm *BuildMetrics) RecordBuild(result BuildResult) {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	bm.TotalBuilds++
	bm.TotalDuration += result.Duration

	if result.CacheHit {
		bm.CacheHits++
	}

	if result.Error != nil {
		bm.FailedBuilds++
	} else {
		bm.SuccessfulBuilds++
	}

	// Update average duration
	if bm.TotalBuilds > 0 {
		bm.AverageDuration = bm.TotalDuration / time.Duration(bm.TotalBuilds)
	}
}

// GetSnapshot returns a snapshot of current metrics
func (bm *BuildMetrics) GetSnapshot() BuildMetrics {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()
	// Return a copy without the mutex to avoid lock copying issues
	return BuildMetrics{
		TotalBuilds:      bm.TotalBuilds,
		SuccessfulBuilds: bm.SuccessfulBuilds,
		FailedBuilds:     bm.FailedBuilds,
		CacheHits:        bm.CacheHits,
		AverageDuration:  bm.AverageDuration,
		TotalDuration:    bm.TotalDuration,
		// mutex is intentionally omitted to prevent lock copying
	}
}

// Reset resets all metrics
func (bm *BuildMetrics) Reset() {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	bm.TotalBuilds = 0
	bm.SuccessfulBuilds = 0
	bm.FailedBuilds = 0
	bm.CacheHits = 0
	bm.AverageDuration = 0
	bm.TotalDuration = 0
}

// GetCacheHitRate returns the cache hit rate as a percentage
func (bm *BuildMetrics) GetCacheHitRate() float64 {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	if bm.TotalBuilds == 0 {
		return 0.0
	}

	return float64(bm.CacheHits) / float64(bm.TotalBuilds) * 100.0
}

// GetSuccessRate returns the success rate as a percentage
func (bm *BuildMetrics) GetSuccessRate() float64 {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	if bm.TotalBuilds == 0 {
		return 0.0
	}

	return float64(bm.SuccessfulBuilds) / float64(bm.TotalBuilds) * 100.0
}
