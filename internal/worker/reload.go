package worker

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// Reload performs a zero-downtime restart of all worker processes.
// It spawns a fresh set of workers, waits for them to stabilize (bind their
// SO_REUSEPORT sockets), then gracefully terminates the old workers.
// If old workers don't exit within GracefulWait, they are forcefully killed.
func (m *Manager) Reload() error {
	log.Printf("[worker] starting reload — spawning %d new workers", m.numWorkers)

	// 1. Fork new workers.
	newWorkers := make([]*WorkerProcess, 0, m.numWorkers)
	for i := 0; i < m.numWorkers; i++ {
		wp, err := m.forkWorker(i)
		if err != nil {
			// Rollback: kill any new workers already started.
			for _, nw := range newWorkers {
				if nw != nil && nw.Cmd != nil && nw.Cmd.Process != nil {
					_ = nw.Cmd.Process.Kill()
				}
			}
			return fmt.Errorf("reload failed while forking worker %d: %w", i, err)
		}
		wp.Status = StatusRunning
		newWorkers = append(newWorkers, wp)
	}

	// 2. Wait for new workers to stabilize (bind SO_REUSEPORT sockets).
	time.Sleep(500 * time.Millisecond)

	// 3. Swap workers: save old set and replace with new.
	m.mu.Lock()
	oldWorkers := m.workers
	m.workers = newWorkers
	m.mu.Unlock()

	// 4. Start waitForExit goroutines for the new workers.
	for _, wp := range newWorkers {
		go m.waitForExit(wp)
	}

	// 5. Signal old workers to terminate gracefully.
	for _, wp := range oldWorkers {
		if wp == nil || wp.Cmd == nil || wp.Cmd.Process == nil {
			continue
		}

		m.mu.Lock()
		wp.Status = StatusStopping
		m.mu.Unlock()

		if err := signalTerminate(wp.Cmd.Process); err != nil {
			log.Printf("[worker] reload: failed to signal old worker %d (pid %d): %v", wp.ID, wp.PID, err)
		}
	}

	// 6. Wait for old workers to exit with a timeout.
	deadline := time.After(m.cfg.GracefulWait)
	done := make(chan struct{})

	go func() {
		var wg sync.WaitGroup
		for _, wp := range oldWorkers {
			if wp == nil || wp.Cmd == nil || wp.Cmd.Process == nil {
				continue
			}
			if wp.Status == StatusDead {
				continue
			}
			wg.Add(1)
			go func(w *WorkerProcess) {
				defer wg.Done()
				_ = w.Cmd.Wait()
			}(wp)
		}
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("[worker] reload complete — old workers exited gracefully")
	case <-deadline:
		// 7. Force-kill old workers that didn't exit in time.
		log.Printf("[worker] reload: graceful wait expired (%s), killing old workers", m.cfg.GracefulWait)
		for _, wp := range oldWorkers {
			if wp == nil || wp.Cmd == nil || wp.Cmd.Process == nil {
				continue
			}

			m.mu.RLock()
			status := wp.Status
			m.mu.RUnlock()

			if status == StatusDead {
				continue
			}

			log.Printf("[worker] reload: killing old worker %d (pid %d)", wp.ID, wp.PID)
			if err := wp.Cmd.Process.Kill(); err != nil {
				log.Printf("[worker] reload: failed to kill old worker %d (pid %d): %v", wp.ID, wp.PID, err)
			}
		}
		log.Printf("[worker] reload complete — old workers force-killed")
	}

	return nil
}
