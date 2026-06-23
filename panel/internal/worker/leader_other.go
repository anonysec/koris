//go:build !linux

package worker

// TryAcquire on non-Linux platforms always succeeds. This is a stub for
// development on Windows/macOS where flock semantics differ. In single-process
// mode (the typical dev scenario), the process is always the leader.
func (l *LeaderLock) TryAcquire() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.locked = true
	return true
}

// Release on non-Linux platforms resets the leader state.
func (l *LeaderLock) Release() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.locked = false
	l.file = nil
}
