// Package build provides build progress reporting with real-time updates.
package build

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/conneroisu/templar/internal/types"
)

// BuildPhase represents different phases of the build process.
type BuildPhase int32

const (
	// PhaseQueued indicates the build task is queued but not started.
	PhaseQueued BuildPhase = iota
	// PhaseStarted indicates the build process has begun.
	PhaseStarted
	// PhaseScanning indicates component scanning phase.
	PhaseScanning
	// PhaseCompiling indicates template compilation phase.
	PhaseCompiling
	// PhaseValidating indicates build output validation phase.
	PhaseValidating
	// PhaseCompleted indicates successful build completion.
	PhaseCompleted
	// PhaseFailed indicates build failure.
	PhaseFailed
	// PhaseCancelled indicates build was cancelled.
	PhaseCancelled
)

// String returns a string representation of the build phase.
func (p BuildPhase) String() string {
	switch p {
	case PhaseQueued:
		return "queued"
	case PhaseStarted:
		return "started"
	case PhaseScanning:
		return "scanning"
	case PhaseCompiling:
		return "compiling"
	case PhaseValidating:
		return "validating"
	case PhaseCompleted:
		return "completed"
	case PhaseFailed:
		return "failed"
	case PhaseCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// BuildProgressInfo contains detailed progress information for a build.
type BuildProgressInfo struct {
	// Build identification
	BuildID       string `json:"build_id"`
	ComponentName string `json:"component_name"`
	ComponentPath string `json:"component_path"`

	// Progress tracking
	Phase         string    `json:"phase"`
	Progress      float64   `json:"progress"` // 0.0 to 1.0
	StartTime     time.Time `json:"start_time"`
	LastUpdate    time.Time `json:"last_update"`
	Duration      string    `json:"duration"`
	EstimatedTime string    `json:"estimated_time,omitempty"`

	// Build details
	CacheHit     bool   `json:"cache_hit"`
	OutputSize   int64  `json:"output_size,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`

	// Performance metrics
	CompileTime    string `json:"compile_time,omitempty"`
	ValidationTime string `json:"validation_time,omitempty"`
	QueueTime      string `json:"queue_time,omitempty"`

	// Context information
	WorkerID   int `json:"worker_id,omitempty"`
	Priority   int `json:"priority"`
	RetryCount int `json:"retry_count,omitempty"`

	// Build artifacts
	Warnings       []string `json:"warnings,omitempty"`
	GeneratedFiles []string `json:"generated_files,omitempty"`
}

// BuildProgressCallback is called when build progress is updated.
type BuildProgressCallback func(progress BuildProgressInfo)

// ProgressUpdate represents a single progress update event.
type ProgressUpdate struct {
	BuildID   string     `json:"build_id"`
	Phase     BuildPhase `json:"phase"`
	Progress  float64    `json:"progress"`
	Message   string     `json:"message,omitempty"`
	Error     error      `json:"-"`
	Timestamp time.Time  `json:"timestamp"`
	WorkerID  int        `json:"worker_id,omitempty"`
}

// BuildProgressReporter manages build progress tracking and reporting.
type BuildProgressReporter struct {
	// Active build tracking
	activeBuilds map[string]*BuildProgressInfo
	activeMutex  sync.RWMutex

	// Progress callbacks
	callbacks     []BuildProgressCallback
	callbackMutex sync.RWMutex

	// Statistics
	totalBuilds     int64
	completedBuilds int64
	failedBuilds    int64
	cancelledBuilds int64

	// Performance tracking
	averageBuildTime time.Duration
	totalBuildTime   time.Duration
	buildTimes       []time.Duration
	timesMutex       sync.Mutex
	maxHistorySize   int

	// Queue monitoring
	queueDepth     int64
	maxQueueDepth  int64
	queueWaitTimes []time.Duration

	// Worker utilization
	workerStats map[int]*WorkerStats
	workerMutex sync.RWMutex
}

// WorkerStats tracks individual worker performance.
type WorkerStats struct {
	WorkerID         int           `json:"worker_id"`
	TotalBuilds      int64         `json:"total_builds"`
	CompletedBuilds  int64         `json:"completed_builds"`
	FailedBuilds     int64         `json:"failed_builds"`
	AverageBuildTime time.Duration `json:"average_build_time"`
	TotalBuildTime   time.Duration `json:"total_build_time"`
	LastActive       time.Time     `json:"last_active"`
	IsActive         bool          `json:"is_active"`
}

// NewBuildProgressReporter creates a new build progress reporter.
func NewBuildProgressReporter() *BuildProgressReporter {
	return &BuildProgressReporter{
		activeBuilds:   make(map[string]*BuildProgressInfo),
		callbacks:      make([]BuildProgressCallback, 0),
		workerStats:    make(map[int]*WorkerStats),
		maxHistorySize: 100,
		buildTimes:     make([]time.Duration, 0, 100),
		queueWaitTimes: make([]time.Duration, 0, 100),
	}
}

// StartBuild initiates progress tracking for a new build.
func (bpr *BuildProgressReporter) StartBuild(component *types.ComponentInfo, priority int, workerID int) string {
	buildID := fmt.Sprintf("build_%s_%d", component.Name, time.Now().UnixNano())

	progress := &BuildProgressInfo{
		BuildID:       buildID,
		ComponentName: component.Name,
		ComponentPath: component.FilePath,
		Phase:         PhaseQueued.String(),
		Progress:      0.0,
		StartTime:     time.Now(),
		LastUpdate:    time.Now(),
		Priority:      priority,
		WorkerID:      workerID,
	}

	bpr.activeMutex.Lock()
	bpr.activeBuilds[buildID] = progress
	bpr.activeMutex.Unlock()

	atomic.AddInt64(&bpr.totalBuilds, 1)

	// Update queue depth
	atomic.AddInt64(&bpr.queueDepth, 1)
	currentDepth := atomic.LoadInt64(&bpr.queueDepth)
	for {
		maxDepth := atomic.LoadInt64(&bpr.maxQueueDepth)
		if currentDepth <= maxDepth {
			break
		}
		if atomic.CompareAndSwapInt64(&bpr.maxQueueDepth, maxDepth, currentDepth) {
			break
		}
	}

	bpr.notifyCallbacks(*progress)

	log.Printf("Started build tracking for %s (ID: %s, Worker: %d)",
		component.Name, buildID, workerID)

	return buildID
}

// UpdateProgress updates the progress of an active build.
func (bpr *BuildProgressReporter) UpdateProgress(buildID string, phase BuildPhase, progress float64, message string) {
	bpr.activeMutex.Lock()
	info, exists := bpr.activeBuilds[buildID]
	if !exists {
		bpr.activeMutex.Unlock()

		return
	}

	// Update progress information
	now := time.Now()
	info.Phase = phase.String()
	info.Progress = progress
	info.LastUpdate = now
	info.Duration = formatDuration(now.Sub(info.StartTime))

	// Calculate estimated completion time
	if progress > 0.0 && progress < 1.0 {
		elapsed := now.Sub(info.StartTime)
		estimated := time.Duration(float64(elapsed) / progress)
		remaining := estimated - elapsed
		if remaining > 0 {
			info.EstimatedTime = formatDuration(remaining)
		}
	}

	// Update worker stats
	if info.WorkerID > 0 {
		bpr.updateWorkerStats(info.WorkerID, phase, now)
	}

	// Create copy for callback
	progressCopy := *info
	bpr.activeMutex.Unlock()

	// Notify callbacks
	bpr.notifyCallbacks(progressCopy)

	if message != "" {
		log.Printf("Build %s: %s (%.1f%%) - %s",
			buildID, phase.String(), progress*100, message)
	}
}

// FinishBuild marks a build as completed and removes it from active tracking.
func (bpr *BuildProgressReporter) FinishBuild(buildID string, result BuildResult) {
	bpr.activeMutex.Lock()
	info, exists := bpr.activeBuilds[buildID]
	if !exists {
		bpr.activeMutex.Unlock()

		return
	}

	now := time.Now()
	duration := now.Sub(info.StartTime)

	// Update final progress information
	if result.Error != nil {
		info.Phase = PhaseFailed.String()
		info.ErrorMessage = result.Error.Error()
		atomic.AddInt64(&bpr.failedBuilds, 1)
	} else {
		info.Phase = PhaseCompleted.String()
		info.Progress = 1.0
		atomic.AddInt64(&bpr.completedBuilds, 1)
	}

	info.LastUpdate = now
	info.Duration = formatDuration(duration)
	info.CacheHit = result.CacheHit
	info.OutputSize = int64(len(result.Output))

	// Update build time statistics
	bpr.updateBuildTimeStats(duration)

	// Update worker completion stats
	if info.WorkerID > 0 {
		bpr.finalizeWorkerStats(info.WorkerID, result.Error == nil, duration)
	}

	// Update queue depth
	atomic.AddInt64(&bpr.queueDepth, -1)

	// Create copy for callback
	progressCopy := *info

	// Remove from active builds
	delete(bpr.activeBuilds, buildID)
	bpr.activeMutex.Unlock()

	// Final progress notification
	bpr.notifyCallbacks(progressCopy)

	status := "completed"
	if result.Error != nil {
		status = "failed"
	}

	log.Printf("Build %s %s in %s (Cache: %v, Size: %d bytes)",
		buildID, status, formatDuration(duration), result.CacheHit, len(result.Output))
}

// CancelBuild marks a build as cancelled.
func (bpr *BuildProgressReporter) CancelBuild(buildID string) {
	bpr.activeMutex.Lock()
	info, exists := bpr.activeBuilds[buildID]
	if !exists {
		bpr.activeMutex.Unlock()

		return
	}

	now := time.Now()
	info.Phase = PhaseCancelled.String()
	info.LastUpdate = now
	info.Duration = formatDuration(now.Sub(info.StartTime))

	progressCopy := *info
	delete(bpr.activeBuilds, buildID)
	bpr.activeMutex.Unlock()

	atomic.AddInt64(&bpr.cancelledBuilds, 1)
	atomic.AddInt64(&bpr.queueDepth, -1)

	bpr.notifyCallbacks(progressCopy)

	log.Printf("Build %s cancelled after %s", buildID, progressCopy.Duration)
}

// GetActiveBuilds returns a snapshot of all active builds.
func (bpr *BuildProgressReporter) GetActiveBuilds() []BuildProgressInfo {
	bpr.activeMutex.RLock()
	defer bpr.activeMutex.RUnlock()

	builds := make([]BuildProgressInfo, 0, len(bpr.activeBuilds))
	for _, info := range bpr.activeBuilds {
		builds = append(builds, *info)
	}

	return builds
}

// GetBuildStats returns comprehensive build statistics.
func (bpr *BuildProgressReporter) GetBuildStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// Basic counters
	stats["total_builds"] = atomic.LoadInt64(&bpr.totalBuilds)
	stats["completed_builds"] = atomic.LoadInt64(&bpr.completedBuilds)
	stats["failed_builds"] = atomic.LoadInt64(&bpr.failedBuilds)
	stats["cancelled_builds"] = atomic.LoadInt64(&bpr.cancelledBuilds)

	// Active builds
	bpr.activeMutex.RLock()
	stats["active_builds"] = len(bpr.activeBuilds)
	bpr.activeMutex.RUnlock()

	// Queue statistics
	stats["current_queue_depth"] = atomic.LoadInt64(&bpr.queueDepth)
	stats["max_queue_depth"] = atomic.LoadInt64(&bpr.maxQueueDepth)

	// Performance statistics
	bpr.timesMutex.Lock()
	if len(bpr.buildTimes) > 0 {
		stats["average_build_time"] = formatDuration(bpr.averageBuildTime)
		stats["total_build_time"] = formatDuration(bpr.totalBuildTime)

		// Calculate percentiles
		if len(bpr.buildTimes) >= 5 {
			sortedTimes := make([]time.Duration, len(bpr.buildTimes))
			copy(sortedTimes, bpr.buildTimes)
			// Simple sort for percentiles (not optimal but sufficient for small sizes)
			for i := range len(sortedTimes) - 1 {
				for j := i + 1; j < len(sortedTimes); j++ {
					if sortedTimes[i] > sortedTimes[j] {
						sortedTimes[i], sortedTimes[j] = sortedTimes[j], sortedTimes[i]
					}
				}
			}

			p50 := sortedTimes[len(sortedTimes)/2]
			p95 := sortedTimes[int(float64(len(sortedTimes))*0.95)]

			stats["build_time_p50"] = formatDuration(p50)
			stats["build_time_p95"] = formatDuration(p95)
		}
	}
	bpr.timesMutex.Unlock()

	// Success rates
	total := atomic.LoadInt64(&bpr.totalBuilds)
	if total > 0 {
		completed := atomic.LoadInt64(&bpr.completedBuilds)
		failed := atomic.LoadInt64(&bpr.failedBuilds)

		stats["success_rate"] = float64(completed) / float64(total) * 100
		stats["failure_rate"] = float64(failed) / float64(total) * 100
	}

	// Worker statistics
	bpr.workerMutex.RLock()
	workerStats := make(map[string]interface{})
	for workerID, worker := range bpr.workerStats {
		workerStats[fmt.Sprintf("worker_%d", workerID)] = map[string]interface{}{
			"total_builds":       worker.TotalBuilds,
			"completed_builds":   worker.CompletedBuilds,
			"failed_builds":      worker.FailedBuilds,
			"average_build_time": formatDuration(worker.AverageBuildTime),
			"is_active":          worker.IsActive,
			"last_active":        worker.LastActive,
		}
	}
	stats["worker_stats"] = workerStats
	bpr.workerMutex.RUnlock()

	return stats
}

// AddProgressCallback adds a callback for progress updates.
func (bpr *BuildProgressReporter) AddProgressCallback(callback BuildProgressCallback) {
	bpr.callbackMutex.Lock()
	defer bpr.callbackMutex.Unlock()
	bpr.callbacks = append(bpr.callbacks, callback)
}

// notifyCallbacks sends progress updates to all registered callbacks.
func (bpr *BuildProgressReporter) notifyCallbacks(progress BuildProgressInfo) {
	bpr.callbackMutex.RLock()
	callbacks := make([]BuildProgressCallback, len(bpr.callbacks))
	copy(callbacks, bpr.callbacks)
	bpr.callbackMutex.RUnlock()

	for _, callback := range callbacks {
		go func(cb BuildProgressCallback, p BuildProgressInfo) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Progress callback panic: %v", r)
				}
			}()
			cb(p)
		}(callback, progress)
	}
}

// updateBuildTimeStats updates build time statistics.
func (bpr *BuildProgressReporter) updateBuildTimeStats(duration time.Duration) {
	bpr.timesMutex.Lock()
	defer bpr.timesMutex.Unlock()

	// Add new duration
	bpr.buildTimes = append(bpr.buildTimes, duration)
	bpr.totalBuildTime += duration

	// Keep history within limits
	if len(bpr.buildTimes) > bpr.maxHistorySize {
		// Remove oldest entries
		removed := bpr.buildTimes[:len(bpr.buildTimes)-bpr.maxHistorySize]
		for _, d := range removed {
			bpr.totalBuildTime -= d
		}
		bpr.buildTimes = bpr.buildTimes[len(bpr.buildTimes)-bpr.maxHistorySize:]
	}

	// Update average
	if len(bpr.buildTimes) > 0 {
		bpr.averageBuildTime = bpr.totalBuildTime / time.Duration(len(bpr.buildTimes))
	}
}

// updateWorkerStats updates individual worker statistics.
func (bpr *BuildProgressReporter) updateWorkerStats(workerID int, phase BuildPhase, timestamp time.Time) {
	bpr.workerMutex.Lock()
	defer bpr.workerMutex.Unlock()

	worker, exists := bpr.workerStats[workerID]
	if !exists {
		worker = &WorkerStats{
			WorkerID: workerID,
		}
		bpr.workerStats[workerID] = worker
	}

	worker.LastActive = timestamp
	worker.IsActive = (phase != PhaseCompleted && phase != PhaseFailed && phase != PhaseCancelled)
}

// finalizeWorkerStats updates worker statistics when a build completes.
func (bpr *BuildProgressReporter) finalizeWorkerStats(workerID int, success bool, duration time.Duration) {
	bpr.workerMutex.Lock()
	defer bpr.workerMutex.Unlock()

	worker, exists := bpr.workerStats[workerID]
	if !exists {
		return
	}

	worker.TotalBuilds++
	if success {
		worker.CompletedBuilds++
	} else {
		worker.FailedBuilds++
	}

	// Update average build time
	worker.TotalBuildTime += duration
	worker.AverageBuildTime = worker.TotalBuildTime / time.Duration(worker.TotalBuilds)
	worker.IsActive = false
}

// CreateWebSocketMessage creates a WebSocket message for progress updates.
func (bpr *BuildProgressReporter) CreateWebSocketMessage(progress BuildProgressInfo) ([]byte, error) {
	message := map[string]interface{}{
		"type":     "build_progress",
		"progress": progress,
	}

	return json.Marshal(message)
}

// GetProgressSummary returns a summary of current build progress.
func (bpr *BuildProgressReporter) GetProgressSummary() map[string]interface{} {
	summary := make(map[string]interface{})

	// Active builds summary
	activeBuilds := bpr.GetActiveBuilds()
	summary["active_builds_count"] = len(activeBuilds)

	if len(activeBuilds) > 0 {
		phases := make(map[string]int)
		totalProgress := 0.0

		for _, build := range activeBuilds {
			phases[build.Phase]++
			totalProgress += build.Progress
		}

		summary["phase_breakdown"] = phases
		summary["average_progress"] = totalProgress / float64(len(activeBuilds))

		// Find longest running build
		oldestStart := time.Now()
		oldestBuild := ""
		for _, build := range activeBuilds {
			if build.StartTime.Before(oldestStart) {
				oldestStart = build.StartTime
				oldestBuild = build.ComponentName
			}
		}

		summary["longest_running_build"] = oldestBuild
		summary["longest_running_duration"] = formatDuration(time.Since(oldestStart))
	}

	// Overall statistics
	stats := bpr.GetBuildStats()
	summary["total_builds"] = stats["total_builds"]
	summary["success_rate"] = stats["success_rate"]
	summary["average_build_time"] = stats["average_build_time"]
	summary["queue_depth"] = stats["current_queue_depth"]

	return summary
}

// formatDuration formats a duration for human readability.
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%.0fμs", float64(d.Nanoseconds())/1000)
	} else if d < time.Second {
		return fmt.Sprintf("%.0fms", float64(d.Nanoseconds())/1000000)
	} else if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
}
