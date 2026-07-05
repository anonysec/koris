package worker

import (
	"context"
	"log"
	"time"
)

// monitorLoop runs a 2-second ticker that checks worker liveness. Dead workers
// are restarted up to MaxRestarts times. The loop exits when ctx is cancelled.
func (m *Manager) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkWorkers()
		}
	}
}

// checkWorkers iterates through all workers and restarts any that are dead,
// up to the configured MaxRestarts limit.
func (m *Manager) checkWorkers() {
	m.mu.RLock()
	workers := make([]*WorkerProcess, len(m.workers))
	copy(workers, m.workers)
	m.mu.RUnlock()

	for _, wp := range workers {
		if wp == nil {
			continue
		}

		if wp.Status != StatusDead {
			continue
		}

		if wp.Restarts >= m.cfg.MaxRestarts {
			// Already logged when it first exceeded, skip.
			continue
		}

		// Attempt restart.
		m.restartWorker(wp)
	}
}

// restartWorker attempts to restart a dead worker. If the worker has reached
// MaxRestarts, it is marked as permanently dead and a warning is logged.
func (m *Manager) restartWorker(wp *WorkerProcess) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check under write lock.
	if wp.Status != StatusDead {
		return
	}

	wp.Restarts++

	if wp.Restarts > m.cfg.MaxRestarts {
		log.Printf("[worker] worker %d exceeded max restarts (%d), marking permanently dead", wp.ID, m.cfg.MaxRestarts)
		return
	}

	log.Printf("[worker] restarting worker %d (attempt %d/%d)", wp.ID, wp.Restarts, m.cfg.MaxRestarts)

	newWP, err := m.forkWorker(wp.ID)
	if err != nil {
		log.Printf("[worker] failed to restart worker %d: %v", wp.ID, err)
		return
	}

	// Preserve the restart count on the new process.
	newWP.Restarts = wp.Restarts
	newWP.Status = StatusRunning

	// Replace in the workers slice.
	for i, w := range m.workers {
		if w != nil && w.ID == wp.ID {
			m.workers[i] = newWP
			break
		}
	}

	// Start a goroutine to wait for the new worker to exit.
	go m.waitForExit(newWP)
}

// waitForExit blocks until the worker process exits, then marks it as dead.
// This should be started as a goroutine for each forked worker.
func (m *Manager) waitForExit(wp *WorkerProcess) {
	if wp.Cmd == nil || wp.Cmd.Process == nil {
		m.mu.Lock()
		wp.Status = StatusDead
		m.mu.Unlock()
		return
	}

	// Block until the child process exits.
	_ = wp.Cmd.Wait()

	m.mu.Lock()
	wp.Status = StatusDead
	m.mu.Unlock()

	log.Printf("[worker] worker %d (pid %d) exited", wp.ID, wp.PID)
}
