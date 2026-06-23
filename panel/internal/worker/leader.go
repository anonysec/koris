package worker

import (
	"os"
	"sync"
)

// LeaderLockPath is the default file path used for leader election.
// Only the process that acquires the exclusive lock on this file runs
// the background ticker (billing, cleanup, monitoring).
const LeaderLockPath = "/var/run/panel-leader.lock"

// LeaderLock implements leader election via an exclusive file lock (flock).
// In a multi-worker deployment, each worker process attempts to acquire
// the lock at startup. Only one succeeds and becomes the leader responsible
// for running the background ticker. When a leader process dies, the OS
// releases the flock automatically, allowing another worker to acquire it.
type LeaderLock struct {
	path   string
	file   *os.File
	locked bool
	mu     sync.Mutex
}

// NewLeaderLock creates a LeaderLock targeting the given file path.
// If path is empty, LeaderLockPath is used.
func NewLeaderLock(path string) *LeaderLock {
	if path == "" {
		path = LeaderLockPath
	}
	return &LeaderLock{
		path: path,
	}
}

// IsLeader reports whether this process currently holds the leader lock.
func (l *LeaderLock) IsLeader() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.locked
}

// Path returns the lock file path.
func (l *LeaderLock) Path() string {
	return l.path
}
