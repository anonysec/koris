package worker

import (
	"os"
	"strconv"
	"testing"
)

func TestBuildWorkerEnv_AddsWorkerID(t *testing.T) {
	env := buildWorkerEnv(3)

	found := false
	expected := WorkerEnvKey + "=3"
	for _, e := range env {
		if e == expected {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected env to contain %q, got: %v", expected, env)
	}
}

func TestBuildWorkerEnv_OverridesExisting(t *testing.T) {
	// Set the env var so it exists in the parent env
	os.Setenv(WorkerEnvKey, "99")
	defer os.Unsetenv(WorkerEnvKey)

	env := buildWorkerEnv(5)

	expected := WorkerEnvKey + "=5"
	count := 0
	for _, e := range env {
		if len(e) >= len(WorkerEnvKey+"=") && e[:len(WorkerEnvKey+"=")] == WorkerEnvKey+"=" {
			count++
			if e != expected {
				t.Errorf("expected %q, got %q", expected, e)
			}
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 occurrence of %s, found %d", WorkerEnvKey, count)
	}
}

func TestBuildWorkerEnv_PreservesParentEnv(t *testing.T) {
	// Ensure there's at least one known var in the parent env
	os.Setenv("TEST_FORK_PRESERVE", "hello")
	defer os.Unsetenv("TEST_FORK_PRESERVE")

	env := buildWorkerEnv(1)

	found := false
	for _, e := range env {
		if e == "TEST_FORK_PRESERVE=hello" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected parent env var TEST_FORK_PRESERVE=hello to be preserved")
	}
}

func TestIsWorkerProcess_NotSet(t *testing.T) {
	os.Unsetenv(WorkerEnvKey)

	isWorker, id := IsWorkerProcess()
	if isWorker {
		t.Error("expected IsWorkerProcess to return false when env var is not set")
	}
	if id != 0 {
		t.Errorf("expected id=0, got %d", id)
	}
}

func TestIsWorkerProcess_ValidID(t *testing.T) {
	os.Setenv(WorkerEnvKey, "7")
	defer os.Unsetenv(WorkerEnvKey)

	isWorker, id := IsWorkerProcess()
	if !isWorker {
		t.Error("expected IsWorkerProcess to return true")
	}
	if id != 7 {
		t.Errorf("expected id=7, got %d", id)
	}
}

func TestIsWorkerProcess_InvalidValue(t *testing.T) {
	os.Setenv(WorkerEnvKey, "notanumber")
	defer os.Unsetenv(WorkerEnvKey)

	isWorker, id := IsWorkerProcess()
	if isWorker {
		t.Error("expected IsWorkerProcess to return false for invalid value")
	}
	if id != 0 {
		t.Errorf("expected id=0, got %d", id)
	}
}

func TestIsWorkerProcess_ZeroID(t *testing.T) {
	os.Setenv(WorkerEnvKey, "0")
	defer os.Unsetenv(WorkerEnvKey)

	isWorker, id := IsWorkerProcess()
	if !isWorker {
		t.Error("expected IsWorkerProcess to return true for id=0")
	}
	if id != 0 {
		t.Errorf("expected id=0, got %d", id)
	}
}

func TestWorkerEnvKey_Constant(t *testing.T) {
	if WorkerEnvKey != "PANEL_WORKER_ID" {
		t.Errorf("expected WorkerEnvKey=%q, got %q", "PANEL_WORKER_ID", WorkerEnvKey)
	}
}

func TestBuildWorkerEnv_LargeID(t *testing.T) {
	id := 999
	env := buildWorkerEnv(id)

	expected := WorkerEnvKey + "=" + strconv.Itoa(id)
	found := false
	for _, e := range env {
		if e == expected {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected env to contain %q for large id", expected)
	}
}
