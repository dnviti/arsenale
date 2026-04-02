package desktopbroker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureRecordingReadableAddsOtherReadBit(t *testing.T) {
	t.Parallel()

	filePath := filepath.Join(t.TempDir(), "session.guac")
	if err := os.WriteFile(filePath, []byte("test"), 0o640); err != nil {
		t.Fatalf("write file: %v", err)
	}

	stat, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if err := ensureRecordingReadable(filePath, stat); err != nil {
		t.Fatalf("ensureRecordingReadable returned error: %v", err)
	}

	updated, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("restat file: %v", err)
	}
	if got, want := updated.Mode().Perm(), os.FileMode(0o644); got != want {
		t.Fatalf("file mode = %o, want %o", got, want)
	}
}

func TestEnsureRecordingReadableLeavesReadableFileUnchanged(t *testing.T) {
	t.Parallel()

	filePath := filepath.Join(t.TempDir(), "session.guac")
	if err := os.WriteFile(filePath, []byte("test"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	stat, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if err := ensureRecordingReadable(filePath, stat); err != nil {
		t.Fatalf("ensureRecordingReadable returned error: %v", err)
	}

	updated, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("restat file: %v", err)
	}
	if got, want := updated.Mode().Perm(), os.FileMode(0o644); got != want {
		t.Fatalf("file mode = %o, want %o", got, want)
	}
}
