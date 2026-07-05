package worker

// WorkerStatus represents the lifecycle state of a worker process.
type WorkerStatus int

const (
	// StatusStarting indicates the worker process is being spawned.
	StatusStarting WorkerStatus = iota
	// StatusRunning indicates the worker process is alive and accepting connections.
	StatusRunning
	// StatusStopping indicates the worker has been signaled to shut down gracefully.
	StatusStopping
	// StatusDead indicates the worker process has exited or failed to start.
	StatusDead
)

// String returns a human-readable label for the worker status.
func (s WorkerStatus) String() string {
	switch s {
	case StatusStarting:
		return "starting"
	case StatusRunning:
		return "running"
	case StatusStopping:
		return "stopping"
	case StatusDead:
		return "dead"
	default:
		return "unknown"
	}
}
