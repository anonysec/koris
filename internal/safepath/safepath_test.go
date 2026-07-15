package safepath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadFile_BlocksDotDot(t *testing.T) {
	if _, err := ReadFile("../etc/passwd"); err == nil {
		t.Fatal("ReadFile allowed a '..' traversal")
	}
}

func TestReadFile_BlocksAbsoluteEscape(t *testing.T) {
	// An absolute path under cwd is allowed (trusted own-dir model),
	// but a '..' escape is not.
	if _, err := ReadFile(filepath.Join("a", "..", "..", "etc", "passwd")); err == nil {
		t.Fatal("ReadFile allowed a multi-level '..' traversal")
	}
}

func TestReadFile_AllowsSafeAbsolute(t *testing.T) {
	// Trusted absolute path inside its own directory must still work,
	// so existing internal callers (db, backup, templates, ...) don't break.
	dir := t.TempDir()
	f := filepath.Join(dir, "ok.txt")
	if err := os.WriteFile(f, []byte("hi"), 0600); err != nil {
		t.Fatal(err)
	}
	data, err := ReadFile(f)
	if err != nil {
		t.Fatalf("ReadFile rejected a trusted absolute path: %v", err)
	}
	if string(data) != "hi" {
		t.Fatalf("unexpected content: %q", data)
	}
}

func TestReadFileInDir_ConfinesToRoot(t *testing.T) {
	base := t.TempDir()
	if _, err := ReadFileInDir(base, "../../etc/passwd"); err == nil {
		t.Fatal("ReadFileInDir allowed escaping the base directory")
	}
	safe := filepath.Join(base, "x.txt")
	if err := os.WriteFile(safe, []byte("y"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadFileInDir(base, "x.txt"); err != nil {
		t.Fatalf("ReadFileInDir rejected a safe in-dir path: %v", err)
	}
}

func TestValidatePath_SymlinkEscape(t *testing.T) {
	base := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skip("symlinks unsupported on this FS")
	}
	if _, err := validatePath(filepath.Join("link", "secret.txt"), base); err == nil {
		t.Fatal("validatePath allowed a symlink escape out of the allowed root")
	}
}
