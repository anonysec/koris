package worker

import (
	"log"
	"sync"
	"time"
)

// gracefulShutdown sends a termination signal to all workers, waits up to
// GracefulWait for them to exit cleanly, then forcefully kills any survivors.
func (m *Manager) gracefulShutdown() {
	m.mu.RLock()
	workers := make([]*WorkerProcess, len(m.workers))
	copy(workers, m.workers)
	m.mu.RUnlock()

	if len(workers) == 0 {
		log.Printf("[worker] no workers to shut down")
		return
	}

	// 1. Signal all running workers to stop gracefully.
	for _, wp := range workers {
		if wp == nil || wp.Cmd == nil || wp.Cmd.Process == nil {
			continue
		}
		if wp.Status == StatusDead {
			continue
		}

		m.mu.Lock()
		wp.Status = StatusStopping
		m.mu.Unlock()

		if err := signalTerminate(wp.Cmd.Process); err != nil {
			log.Printf("[worker] failed to signal worker %d (pid %d): %v", wp.ID, wp.PID, err)
		} else {
			log.Printf("[worker] sent terminate signal to worker %d (pid %d)", wp.ID, wp.PID)
		}
	}

	// 2. Wait for all workers to exit, with a deadline.
	deadline := time.After(m.cfg.GracefulWait)
	done := make(chan struct{})

	go func() {
		var wg sync.WaitGroup
		for _, wp := range workers {
			if wp == nil || wp.Cmd == nil || wp.Cmd.Process == nil {
				continue
			}
			if wp.Status == StatusDead {
				continue
			}
			wg.Add(1)
			go func(w *WorkerProcess) {
				defer wg.Done()
				// Cmd.Wait() may already have been called by waitForExit;
				// Process.Wait() is idempotent-safe on some platforms but
				// we just need to block until the process is gone.
				_ = w.Cmd.Wait()
			}(wp)
		}
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("[worker] all workers exited gracefully")
	case <-deadline:
		log.Printf("[worker] graceful wait expired (%s), killing survivors", m.cfg.GracefulWait)
		m.killSurvivors(workers)
	}
}

// killSurvivors sends SIGKILL (or platform equivalent) to any workers that
// have not yet exited.
func (m *Manager) killSurvivors(workers []*WorkerProcess) {
	for _, wp := range workers {
		if wp == nil || wp.Cmd == nil || wp.Cmd.Process == nil {
			continue
		}

		m.mu.RLock()
		status := wp.Status
		m.mu.RUnlock()

		if status == StatusDead {
			continue
		}

		log.Printf("[worker] killing worker %d (pid %d)", wp.ID, wp.PID)
		if err := wp.Cmd.Process.Kill(); err != nil {
			log.Printf("[worker] failed to kill worker %d (pid %d): %v", wp.ID, wp.PID, err)
		}
	}
}
