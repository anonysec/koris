package worker

import (
	"os/exec"
	"time"
)

// WorkerProcess represents a single worker child process managed by the Manager.
type WorkerProcess struct {
	// ID is the unique worker identifier (0-based index).
	ID int

	// PID is the OS process ID of the running worker. Zero if not started.
	PID int

	// Cmd is the exec.Cmd handle for the worker process.
	Cmd *exec.Cmd

	// Status is the current lifecycle state of this worker.
	Status WorkerStatus

	// StartAt is when this worker process was last started.
	StartAt time.Time

	// Restarts is the number of times this worker has been restarted.
	Restarts int
}
