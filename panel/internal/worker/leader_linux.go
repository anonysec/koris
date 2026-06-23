//go:build linux

package worker

import (
	"os"
	"syscall"
)

// TryAcquire attempts a non-blocking exclusive file lock (flock) on the
// lock file. Returns true if the lock was successfully acquired, making
// this process the leader. Returns false if another process already holds
// the lock.
//
// The lock file is created if it does not exist.
func (l *LeaderLock) TryAcquire() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.locked {
		return true
	}

	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return false
	}

	// LOCK_EX = exclusive, LOCK_NB = non-blocking.
	// If another process holds the lock, this returns EWOULDBLOCK immediately.
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		f.Close()
		return false
	}

	l.file = f
	l.locked = true
	return true
}

// Release releases the exclusive file lock and closes the lock file.
// After Release, IsLeader returns false.
func (l *LeaderLock) Release() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.locked || l.file == nil {
		return
	}

	// Unlock then close. Closing also releases the flock, but we unlock
	// explicitly for clarity.
	syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	l.file.Close()
	l.file = nil
	l.locked = false
}
