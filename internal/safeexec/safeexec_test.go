package safeexec

import "testing"

func TestValidate_AllowsKnownBinary(t *testing.T) {
	if err := Validate("iptables"); err != nil {
		t.Fatalf("Validate rejected a known binary: %v", err)
	}
}

func TestValidate_RejectsShell(t *testing.T) {
	for _, sh := range []string{"sh", "bash", "/bin/sh", "/usr/bin/bash", "bash -c"} {
		if err := Validate(sh); err == nil {
			t.Fatalf("Validate allowed a shell: %q", sh)
		}
	}
}

func TestValidate_RejectsArbitrary(t *testing.T) {
	if err := Validate("rm -rf /"); err == nil {
		t.Fatal("Validate allowed an argument string as a command")
	}
	if err := Validate("/usr/bin/evil"); err == nil {
		t.Fatal("Validate allowed an untrusted absolute binary path")
	}
}

func TestCommand_BlockShell(t *testing.T) {
	if _, err := Command("bash", "-c", "touch /tmp/pwned"); err == nil {
		t.Fatal("Command allowed a shell invocation")
	}
}
