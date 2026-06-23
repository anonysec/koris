package worker

import (
	"path/filepath"
	"testing"
)

func TestNewLeaderLock_DefaultPath(t *testing.T) {
	ll := NewLeaderLock("")
	if ll.Path() != LeaderLockPath {
		t.Errorf("expected default path %q, got %q", LeaderLockPath, ll.Path())
	}
}

func TestNewLeaderLock_CustomPath(t *testing.T) {
	custom := "/tmp/test-leader.lock"
	ll := NewLeaderLock(custom)
	if ll.Path() != custom {
		t.Errorf("expected path %q, got %q", custom, ll.Path())
	}
}

func TestLeaderLock_InitialState(t *testing.T) {
	ll := NewLeaderLock("/tmp/test.lock")
	if ll.IsLeader() {
		t.Error("new LeaderLock should not be leader before TryAcquire")
	}
}

func TestLeaderLock_TryAcquire(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "leader.lock")

	ll := NewLeaderLock(path)
	defer ll.Release()

	if !ll.TryAcquire() {
		t.Fatal("expected TryAcquire to succeed")
	}
	if !ll.IsLeader() {
		t.Error("expected IsLeader to return true after successful acquire")
	}
}

func TestLeaderLock_TryAcquireIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "leader.lock")

	ll := NewLeaderLock(path)
	defer ll.Release()

	if !ll.TryAcquire() {
		t.Fatal("first acquire should succeed")
	}
	// Calling TryAcquire again on the same lock should still return true.
	if !ll.TryAcquire() {
		t.Error("second acquire on same lock should return true (idempotent)")
	}
}

func TestLeaderLock_Release(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "leader.lock")

	ll := NewLeaderLock(path)

	if !ll.TryAcquire() {
		t.Fatal("expected TryAcquire to succeed")
	}
	ll.Release()

	if ll.IsLeader() {
		t.Error("expected IsLeader to return false after Release")
	}
}

func TestLeaderLock_ReleaseWithoutAcquire(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "leader.lock")

	ll := NewLeaderLock(path)
	// Should not panic
	ll.Release()

	if ll.IsLeader() {
		t.Error("should not be leader after Release without Acquire")
	}
}

func TestLeaderLock_ReacquireAfterRelease(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "leader.lock")

	ll := NewLeaderLock(path)

	if !ll.TryAcquire() {
		t.Fatal("first acquire should succeed")
	}
	ll.Release()

	if !ll.TryAcquire() {
		t.Fatal("re-acquire after release should succeed")
	}
	defer ll.Release()

	if !ll.IsLeader() {
		t.Error("should be leader after re-acquire")
	}
}

func TestLeaderLock_AcquireSucceeds(t *testing.T) {
	// On all platforms, acquiring a valid temp path should succeed.
	dir := t.TempDir()
	path := filepath.Join(dir, "leader.lock")

	ll := NewLeaderLock(path)
	defer ll.Release()

	if !ll.TryAcquire() {
		t.Fatal("expected TryAcquire to succeed on valid path")
	}
	if !ll.IsLeader() {
		t.Error("expected IsLeader to be true")
	}
}
