package worker

import (
	"context"
	"os/exec"
	"runtime"
	"sync"
	"testing"
	"time"
)

// helperSleepCmd returns an exec.Cmd that sleeps for the given duration.
// On Windows it uses "timeout /t N /nobreak", on Unix it uses "sleep N".
func helperSleepCmd(seconds int) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("ping", "-n", "60", "127.0.0.1")
	}
	return exec.Command("sleep", "60")
}

// helperShortCmd returns a command that exits immediately.
func helperShortCmd() *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/C", "echo done")
	}
	return exec.Command("true")
}

func TestGracefulShutdown_AllWorkersExitGracefully(t *testing.T) {
	// Use a command that responds to interrupt/kill by exiting.
	mgr := &Manager{
		cfg: Config{
			GracefulWait: 5 * time.Second,
		},
		workers: make([]*WorkerProcess, 0),
	}

	// Start a real short-lived process that we can signal.
	cmd := helperSleepCmd(60)
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start helper process: %v", err)
	}

	wp := &WorkerProcess{
		ID:     0,
		PID:    cmd.Process.Pid,
		Cmd:    cmd,
		Status: StatusRunning,
	}

	mgr.workers = append(mgr.workers, wp)

	// Run shutdown — should signal and the process should die.
	done := make(chan struct{})
	go func() {
		mgr.gracefulShutdown()
		close(done)
	}()

	select {
	case <-done:
		// Success - shutdown completed.
	case <-time.After(10 * time.Second):
		t.Fatal("gracefulShutdown did not complete within timeout")
	}

	// Worker should be marked stopping or dead by waitForExit goroutines.
	if wp.Status != StatusStopping && wp.Status != StatusDead {
		t.Errorf("expected worker status stopping or dead, got %s", wp.Status)
	}
}

func TestGracefulShutdown_NoWorkers(t *testing.T) {
	mgr := &Manager{
		cfg: Config{
			GracefulWait: 1 * time.Second,
		},
		workers: make([]*WorkerProcess, 0),
	}

	// Should not panic and return quickly.
	done := make(chan struct{})
	go func() {
		mgr.gracefulShutdown()
		close(done)
	}()

	select {
	case <-done:
		// Success.
	case <-time.After(2 * time.Second):
		t.Fatal("gracefulShutdown blocked on empty worker list")
	}
}

func TestGracefulShutdown_SkipsDeadWorkers(t *testing.T) {
	mgr := &Manager{
		cfg: Config{
			GracefulWait: 1 * time.Second,
		},
		workers: make([]*WorkerProcess, 0),
	}

	// A dead worker with no process handle.
	wp := &WorkerProcess{
		ID:     0,
		Status: StatusDead,
		Cmd:    nil,
	}
	mgr.workers = append(mgr.workers, wp)

	done := make(chan struct{})
	go func() {
		mgr.gracefulShutdown()
		close(done)
	}()

	select {
	case <-done:
		// Success — dead workers are skipped.
	case <-time.After(2 * time.Second):
		t.Fatal("gracefulShutdown blocked on dead worker")
	}
}

func TestGracefulShutdown_SetsStatusToStopping(t *testing.T) {
	mgr := &Manager{
		cfg: Config{
			GracefulWait: 5 * time.Second,
		},
		workers: make([]*WorkerProcess, 0),
	}

	cmd := helperSleepCmd(60)
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start helper process: %v", err)
	}

	wp := &WorkerProcess{
		ID:     0,
		PID:    cmd.Process.Pid,
		Cmd:    cmd,
		Status: StatusRunning,
	}
	mgr.workers = append(mgr.workers, wp)

	// Capture the status after signal but before completion.
	var statusAfterSignal WorkerStatus
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		mgr.gracefulShutdown()
	}()

	// Give time for the signal to be sent.
	time.Sleep(100 * time.Millisecond)
	mgr.mu.RLock()
	statusAfterSignal = wp.Status
	mgr.mu.RUnlock()

	wg.Wait()

	if statusAfterSignal != StatusStopping {
		t.Errorf("expected status=stopping after signal, got %s", statusAfterSignal)
	}
}

func TestStop_CancelsContextAndShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mgr := &Manager{
		cfg: Config{
			GracefulWait: 1 * time.Second,
		},
		workers: make([]*WorkerProcess, 0),
		ctx:     ctx,
		cancel:  cancel,
	}

	err := mgr.Stop()
	if err != nil {
		t.Fatalf("Stop() returned error: %v", err)
	}

	// Context should be cancelled.
	select {
	case <-mgr.ctx.Done():
		// Good - context was cancelled.
	default:
		t.Error("expected context to be cancelled after Stop()")
	}
}

func TestKillSurvivors_KillsRunningProcesses(t *testing.T) {
	mgr := &Manager{
		cfg: Config{
			GracefulWait: 1 * time.Second,
		},
	}

	cmd := helperSleepCmd(60)
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start helper process: %v", err)
	}

	wp := &WorkerProcess{
		ID:     0,
		PID:    cmd.Process.Pid,
		Cmd:    cmd,
		Status: StatusStopping,
	}

	mgr.killSurvivors([]*WorkerProcess{wp})

	// Wait for the process to actually exit after being killed.
	_ = cmd.Wait()

	// If we got here without hanging, the kill worked.
}

func TestKillSurvivors_SkipsDeadWorkers(t *testing.T) {
	mgr := &Manager{
		cfg: Config{
			GracefulWait: 1 * time.Second,
		},
	}

	wp := &WorkerProcess{
		ID:     0,
		Status: StatusDead,
		Cmd:    nil,
	}

	// Should not panic.
	mgr.killSurvivors([]*WorkerProcess{wp})
}
