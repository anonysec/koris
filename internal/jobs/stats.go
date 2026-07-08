package jobs

import (
	"sync/atomic"
	"time"
)

// Stats holds queue statistics
type Stats struct {
	Processed     int64
	Queued        int64
	Failed        int64
	ActiveWorkers int32
	Uptime        time.Duration
	startedAt     time.Time
}

// QueueStats returns queue statistics
func (q *Queue) Stats() *Stats {
	return &Stats{
		Processed:     atomic.LoadInt64(&q.processed),
		Queued:        int64(len(q.jobs)),
		Failed:        atomic.LoadInt64(&q.failed),
		ActiveWorkers: atomic.LoadInt32(&q.activeWorkers),
		Uptime:        time.Since(q.startedAt),
	}
}

// Processed increments the processed counter
func (q *Queue) Processed() {
	atomic.AddInt64(&q.processed, 1)
}

// Failed increments the failed counter
func (q *Queue) Failed() {
	atomic.AddInt64(&q.failed, 1)
}

// ActiveWorker increments active workers
func (q *Queue) ActiveWorker() {
	atomic.AddInt32(&q.activeWorkers, 1)
}

// InactiveWorker decrements active workers
func (q *Queue) InactiveWorker() {
	atomic.AddInt32(&q.activeWorkers, -1)
}
