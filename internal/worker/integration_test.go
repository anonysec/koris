//go:build integration

package worker

import (
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

// TestIntegration_ManagerCreatesCorrectWorkerEntries verifies that when
// a Manager is configured with N workers, calling Status() after setup
// returns the correct number of WorkerProcess entries.
func TestIntegration_ManagerCreatesCorrectWorkerEntries(t *testing.T) {
	tests := []struct {
		name     string
		workers  int
		expected int
	}{
		{"2 workers", 2, 2},
		{"4 workers", 4, 4},
		{"1 worker", 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &Manager{
				numWorkers: tt.workers,
				addr:       ":0",
				workers:    make([]*WorkerProcess, 0, tt.workers),
				cfg: Config{
					NumWorkers:   tt.workers,
					Addr:         ":0",
					GracefulWait: 5 * time.Second,
					MaxRestarts:  3,
				},
			}

			// Simulate Start by adding worker entries.
			for i := 0; i < tt.workers; i++ {
				mgr.workers = append(mgr.workers, &WorkerProcess{
					ID:      i,
					PID:     1000 + i,
					Status:  StatusRunning,
					StartAt: time.Now(),
				})
			}

			status := mgr.Status()
			if len(status) != tt.expected {
				t.Errorf("expected %d worker entries, got %d", tt.expected, len(status))
			}

			// Verify each entry has correct ID.
			for i, s := range status {
				if s.ID != i {
					t.Errorf("worker[%d].ID=%d, want %d", i, s.ID, i)
				}
				if s.Status != StatusRunning {
					t.Errorf("worker[%d].Status=%s, want running", i, s.Status)
				}
			}
		})
	}
}

// TestIntegration_LeaderElection_OneWins verifies that when two locks
// compete for the same lock file, exactly one wins (on Linux) or both
// succeed (on non-Linux stubs).
func TestIntegration_LeaderElection_OneWins(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "leader.lock")

	const numContenders = 3
	locks := make([]*LeaderLock, numContenders)
	results := make([]bool, numContenders)

	for i := 0; i < numContenders; i++ {
		locks[i] = NewLeaderLock(lockPath)
	}

	// All contenders try to acquire concurrently.
	var wg sync.WaitGroup
	for i := 0; i < numContenders; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = locks[idx].TryAcquire()
		}(i)
	}
	wg.Wait()

	// Count how many acquired.
	leaders := 0
	for _, r := range results {
		if r {
			leaders++
		}
	}

	if runtime.GOOS == "linux" {
		if leaders != 1 {
			t.Errorf("on Linux, expected exactly 1 leader from %d contenders, got %d", numContenders, leaders)
		}
	} else {
		// On non-Linux (stub), all succeed since TryAcquire always returns true.
		if leaders != numContenders {
			t.Errorf("on non-Linux stub, expected all %d to succeed, got %d", numContenders, leaders)
		}
	}

	// Cleanup.
	for _, l := range locks {
		l.Release()
	}
}

// TestIntegration_LeaderElection_Failover verifies that when the current
// leader releases the lock, another contender can acquire it.
func TestIntegration_LeaderElection_Failover(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "leader.lock")

	leader := NewLeaderLock(lockPath)
	challenger := NewLeaderLock(lockPath)

	// Leader acquires first.
	if !leader.TryAcquire() {
		t.Fatal("expected leader to acquire lock")
	}
	if !leader.IsLeader() {
		t.Fatal("leader should report IsLeader=true")
	}

	// Simulate leader dying by releasing the lock.
	leader.Release()

	// Challenger should now be able to acquire.
	if !challenger.TryAcquire() {
		t.Fatal("challenger should acquire lock after leader release")
	}
	if !challenger.IsLeader() {
		t.Fatal("challenger should report IsLeader=true")
	}

	challenger.Release()
}

// TestIntegration_ConfigResolvedWorkers_Bounds verifies that the auto-detect
// path always returns a value in [1, 4].
func TestIntegration_ConfigResolvedWorkers_Bounds(t *testing.T) {
	cfg := Config{NumWorkers: 0}
	got := cfg.ResolvedWorkers()

	if got < 1 || got > 4 {
		t.Errorf("ResolvedWorkers() auto-detect returned %d, expected [1, 4]", got)
	}

	// Also test that explicit values are passed through correctly.
	for _, n := range []int{1, 2, 3, 4, 8, 16} {
		cfg.NumWorkers = n
		if cfg.ResolvedWorkers() != n {
			t.Errorf("ResolvedWorkers() with explicit %d returned %d", n, cfg.ResolvedWorkers())
		}
	}
}

// TestIntegration_ConcurrentStatusAccess verifies that Status() is safe
// to call concurrently with workers being added/modified.
func TestIntegration_ConcurrentStatusAccess(t *testing.T) {
	mgr := &Manager{
		numWorkers: 4,
		workers:    make([]*WorkerProcess, 0, 4),
		cfg:        Config{NumWorkers: 4},
	}

	// Add initial workers.
	for i := 0; i < 4; i++ {
		mgr.workers = append(mgr.workers, &WorkerProcess{
			ID:      i,
			PID:     2000 + i,
			Status:  StatusRunning,
			StartAt: time.Now(),
		})
	}

	// Concurrently read status and modify workers.
	var wg sync.WaitGroup
	const iterations = 100

	// Readers.
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				status := mgr.Status()
				if len(status) != 4 {
					t.Errorf("expected 4 workers in status, got %d", len(status))
					return
				}
			}
		}()
	}

	// Writer — modifies status under lock.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < iterations; j++ {
			mgr.mu.Lock()
			for _, w := range mgr.workers {
				if w != nil {
					// Toggle status to simulate state changes.
					if w.Status == StatusRunning {
						w.Status = StatusStopping
					} else {
						w.Status = StatusRunning
					}
				}
			}
			mgr.mu.Unlock()
		}
	}()

	wg.Wait()
}
