// Package build provides task queue management for the build pipeline.
//
// TaskQueueManager implements a priority-based task queue with separate
// channels for regular and high-priority tasks, ensuring optimal resource
// utilization and proper task ordering.
package build

import (
	"context"
	"sync"

	"github.com/conneroisu/templar/internal/interfaces"
)

// TaskQueueManager manages build task queues with priority support.
// It provides separate channels for regular and priority tasks,
// ensuring high-priority builds are processed first while maintaining
// proper backpressure and resource management.
type TaskQueueManager struct {
	// tasks channel for regular priority build tasks
	tasks chan BuildTask
	// results channel for publishing build results
	results chan BuildResult
	// priority channel for high-priority build tasks
	priority chan BuildTask
	// metrics tracks queue health and performance
	metrics *BuildMetrics
	// mu protects concurrent access to queue state
	mu sync.RWMutex
	// closed indicates if the queue has been shut down
	closed bool
}

// NewTaskQueueManager creates a new task queue manager with the specified
// buffer sizes and metrics tracking.
func NewTaskQueueManager(taskBufferSize, resultBufferSize, priorityBufferSize int, metrics *BuildMetrics) *TaskQueueManager {
	return &TaskQueueManager{
		tasks:    make(chan BuildTask, taskBufferSize),
		results:  make(chan BuildResult, resultBufferSize),
		priority: make(chan BuildTask, priorityBufferSize),
		metrics:  metrics,
		closed:   false,
	}
}

// Enqueue adds a regular priority task to the queue.
// Returns an error if the queue is closed or if the task cannot be enqueued.
func (tqm *TaskQueueManager) Enqueue(task interface{}) error {
	tqm.mu.RLock()
	if tqm.closed {
		tqm.mu.RUnlock()
		return ErrQueueClosed
	}
	tqm.mu.RUnlock()

	buildTask, ok := task.(BuildTask)
	if !ok {
		return ErrInvalidTaskType
	}

	select {
	case tqm.tasks <- buildTask:
		// Note: Task enqueue tracking could be added to metrics if needed
		return nil
	default:
		// Note: Task drop tracking could be added to metrics if needed
		return ErrQueueFull
	}
}

// EnqueuePriority adds a high priority task to the queue.
// Priority tasks are processed before regular tasks.
func (tqm *TaskQueueManager) EnqueuePriority(task interface{}) error {
	tqm.mu.RLock()
	if tqm.closed {
		tqm.mu.RUnlock()
		return ErrQueueClosed
	}
	tqm.mu.RUnlock()

	buildTask, ok := task.(BuildTask)
	if !ok {
		return ErrInvalidTaskType
	}

	select {
	case tqm.priority <- buildTask:
		// Note: Priority task tracking could be added to metrics if needed
		return nil
	default:
		// Note: Task drop tracking could be added to metrics if needed
		return ErrQueueFull
	}
}

// GetNextTask returns a channel for receiving the next available task.
// Priority tasks are delivered before regular tasks when both are available.
func (tqm *TaskQueueManager) GetNextTask() <-chan interface{} {
	// Create a merged channel that prioritizes priority tasks
	merged := make(chan interface{}, 1)
	
	go func() {
		defer close(merged)
		for {
			tqm.mu.RLock()
			closed := tqm.closed
			tqm.mu.RUnlock()
			
			if closed && len(tqm.priority) == 0 && len(tqm.tasks) == 0 {
				return
			}

			select {
			case task, ok := <-tqm.priority:
				if !ok {
					return
				}
				merged <- task
			default:
				select {
				case task, ok := <-tqm.priority:
					if !ok {
						return
					}
					merged <- task
				case task, ok := <-tqm.tasks:
					if !ok {
						return
					}
					merged <- task
				case <-context.Background().Done():
					return
				}
			}
		}
	}()
	
	return merged
}

// PublishResult publishes a build result to the results channel.
func (tqm *TaskQueueManager) PublishResult(result interface{}) error {
	tqm.mu.RLock()
	if tqm.closed {
		tqm.mu.RUnlock()
		return ErrQueueClosed
	}
	tqm.mu.RUnlock()

	buildResult, ok := result.(BuildResult)
	if !ok {
		return ErrInvalidResultType
	}

	select {
	case tqm.results <- buildResult:
		// Note: Result publish tracking could be added to metrics if needed
		return nil
	default:
		// Note: Results dropped tracking could be added to metrics if needed
		return ErrQueueFull
	}
}

// GetResults returns a channel for receiving build results.
func (tqm *TaskQueueManager) GetResults() <-chan interface{} {
	resultChan := make(chan interface{})
	
	go func() {
		defer close(resultChan)
		for result := range tqm.results {
			select {
			case resultChan <- result:
			case <-context.Background().Done():
				return
			}
		}
	}()
	
	return resultChan
}

// Close gracefully shuts down the task queue, preventing new tasks
// from being enqueued while allowing existing tasks to be processed.
func (tqm *TaskQueueManager) Close() {
	tqm.mu.Lock()
	defer tqm.mu.Unlock()
	
	if !tqm.closed {
		tqm.closed = true
		close(tqm.tasks)
		close(tqm.priority)
		close(tqm.results)
	}
}

// GetQueueStats returns current queue statistics for monitoring.
func (tqm *TaskQueueManager) GetQueueStats() QueueStats {
	tqm.mu.RLock()
	defer tqm.mu.RUnlock()
	
	return QueueStats{
		TasksQueued:    len(tqm.tasks),
		PriorityQueued: len(tqm.priority),
		ResultsQueued:  len(tqm.results),
		Closed:         tqm.closed,
	}
}

// QueueStats provides queue health and capacity information.
type QueueStats struct {
	TasksQueued    int
	PriorityQueued int
	ResultsQueued  int
	Closed         bool
}

// Queue error definitions
var (
	ErrQueueClosed       = &QueueError{Code: "QUEUE_CLOSED", Message: "task queue has been closed"}
	ErrQueueFull         = &QueueError{Code: "QUEUE_FULL", Message: "task queue is full"}
	ErrInvalidTaskType   = &QueueError{Code: "INVALID_TASK_TYPE", Message: "invalid task type provided"}
	ErrInvalidResultType = &QueueError{Code: "INVALID_RESULT_TYPE", Message: "invalid result type provided"}
)

// QueueError represents an error in queue operations.
type QueueError struct {
	Code    string
	Message string
}

func (qe *QueueError) Error() string {
	return qe.Message
}

// Verify that TaskQueueManager implements the TaskQueue interface
var _ interfaces.TaskQueue = (*TaskQueueManager)(nil)