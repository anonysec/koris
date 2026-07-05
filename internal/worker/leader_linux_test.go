//go:build linux

package worker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLeaderLock_CreatesFile_Linux(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "leader.lock")

	ll := NewLeaderLock(path)
	defer ll.Release()

	if !ll.TryAcquire() {
		t.Fatal("expected TryAcquire to succeed")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected lock file to be created on Linux")
	}
}

func TestLeaderLock_InvalidPath_Linux(t *testing.T) {
	// On Linux the real flock implementation requires the file to be openable.
	ll := NewLeaderLock("/nonexistent/dir/leader.lock")

	if ll.TryAcquire() {
		t.Error("expected TryAcquire to fail with invalid path on Linux")
		ll.Release()
	}
	if ll.IsLeader() {
		t.Error("should not be leader when acquire fails")
	}
}

func TestLeaderLock_Contention_Linux(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "leader.lock")

	// First lock acquires successfully.
	ll1 := NewLeaderLock(path)
	defer ll1.Release()

	if !ll1.TryAcquire() {
		t.Fatal("first lock should acquire")
	}

	// Second lock on the same file should fail (non-blocking flock).
	ll2 := NewLeaderLock(path)
	defer ll2.Release()

	if ll2.TryAcquire() {
		t.Error("second lock should fail while first is held")
	}
	if ll2.IsLeader() {
		t.Error("second lock should not be leader")
	}
}

func TestLeaderLock_ContentionRelease_Linux(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "leader.lock")

	ll1 := NewLeaderLock(path)
	if !ll1.TryAcquire() {
		t.Fatal("first lock should acquire")
	}

	// Release first lock.
	ll1.Release()

	// Now second lock should succeed.
	ll2 := NewLeaderLock(path)
	defer ll2.Release()

	if !ll2.TryAcquire() {
		t.Error("second lock should succeed after first is released")
	}
	if !ll2.IsLeader() {
		t.Error("second lock should be leader after acquiring")
	}
}
