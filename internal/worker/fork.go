package worker

import (
	"github.com/anonysec/koris/internal/safeexec"
	"fmt"
	"os"
	"strconv"
	"time"
)

// WorkerEnvKey is the environment variable name set on child worker processes
// to signal that the process should run in worker mode.
const WorkerEnvKey = "PANEL_WORKER_ID"

// forkWorker spawns a child worker process with the given ID.
// The child is the same binary re-executed with PANEL_WORKER_ID=N set in its
// environment. Each worker binds its own listener via SO_REUSEPORT on the
// configured address.
func (m *Manager) forkWorker(id int) (*WorkerProcess, error) {
	exe, err := os.Executable()
	if err != nil {
		return &WorkerProcess{
			ID:     id,
			Status: StatusDead,
		}, fmt.Errorf("failed to get executable path: %w", err)
	}

	cmd := safeexec.MustCommand(exe)

	// Inherit parent environment, adding/overriding the worker ID variable.
	env := buildWorkerEnv(id)
	cmd.Env = env

	// Inherit parent stdout/stderr for log collection.
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return &WorkerProcess{
			ID:     id,
			Status: StatusDead,
		}, fmt.Errorf("failed to start worker %d: %w", id, err)
	}

	wp := &WorkerProcess{
		ID:      id,
		PID:     cmd.Process.Pid,
		Cmd:     cmd,
		Status:  StatusStarting,
		StartAt: time.Now(),
	}

	return wp, nil
}

// buildWorkerEnv returns a copy of the current process environment with
// PANEL_WORKER_ID set to the given worker ID. If the variable already exists
// in the parent env, it is overridden.
func buildWorkerEnv(id int) []string {
	parentEnv := os.Environ()
	workerVar := WorkerEnvKey + "=" + strconv.Itoa(id)

	env := make([]string, 0, len(parentEnv)+1)
	found := false
	prefix := WorkerEnvKey + "="

	for _, e := range parentEnv {
		if len(e) >= len(prefix) && e[:len(prefix)] == prefix {
			env = append(env, workerVar)
			found = true
		} else {
			env = append(env, e)
		}
	}

	if !found {
		env = append(env, workerVar)
	}

	return env
}

// IsWorkerProcess checks whether the current process is a worker child by
// looking for the PANEL_WORKER_ID environment variable. It returns true and
// the worker ID if the variable is set and contains a valid integer.
func IsWorkerProcess() (bool, int) {
	val := os.Getenv(WorkerEnvKey)
	if val == "" {
		return false, 0
	}

	id, err := strconv.Atoi(val)
	if err != nil {
		return false, 0
	}

	return true, id
}
