package sessionrecording

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAsciicastHeaderMakesFileSharedWritable(t *testing.T) {
	t.Parallel()

	filePath := filepath.Join(t.TempDir(), "session.cast")
	if err := writeAsciicastHeader(filePath, map[string]any{
		"version":   2,
		"width":     80,
		"height":    24,
		"timestamp": 1,
	}); err != nil {
		t.Fatalf("writeAsciicastHeader() error = %v", err)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o666); got != want {
		t.Fatalf("file mode = %o, want %o", got, want)
	}
}
