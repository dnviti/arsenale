package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestFileHistoryCommandWiring(t *testing.T) {
	if !strings.Contains(strings.ToLower(fileCmd.Short), "sandbox") {
		t.Fatalf("fileCmd.Short = %q; want sandbox wording", fileCmd.Short)
	}
	if !strings.Contains(strings.ToLower(fileHistoryCmd.Short), "history") {
		t.Fatalf("fileHistoryCmd.Short = %q; want history wording", fileHistoryCmd.Short)
	}

	if fileHistoryCmd.Parent() != fileCmd {
		t.Fatalf("fileHistoryCmd parent = %v; want fileCmd", fileHistoryCmd.Parent())
	}
	if got := len(fileHistoryCmd.Commands()); got != 4 {
		t.Fatalf("fileHistoryCmd subcommands = %d; want 4", got)
	}

	checks := map[string]*cobra.Command{
		"list":     fileHistoryListCmd,
		"download": fileHistoryDownloadCmd,
		"restore":  fileHistoryRestoreCmd,
		"delete":   fileHistoryDeleteCmd,
	}
	for name, cmd := range checks {
		if cmd.Parent() != fileHistoryCmd {
			t.Fatalf("%s parent = %v; want fileHistoryCmd", name, cmd.Parent())
		}
		flag := cmd.Flags().Lookup("connection")
		if flag == nil {
			t.Fatalf("%s missing connection flag", name)
		}
		if !strings.Contains(strings.ToLower(flag.Usage), "file history") {
			t.Fatalf("%s connection flag usage = %q; want file history wording", name, flag.Usage)
		}
	}

	if fileHistoryDownloadCmd.Flags().Lookup("dest") == nil {
		t.Fatal("history download missing dest flag")
	}
	if got := fileHistoryDownloadCmd.Flags().Lookup("dest").DefValue; got != "." {
		t.Fatalf("history download --dest default = %q; want .", got)
	}
}

func TestResolveSSHDownloadDestination(t *testing.T) {
	tests := []struct {
		name        string
		remotePath  string
		destination string
		want        string
	}{
		{
			name:        "empty destination defaults to current dir and joins filename",
			remotePath:  "/remote/path/file.txt",
			destination: "",
			want:        "file.txt",
		},
		{
			name:        "dot destination joins with filename",
			remotePath:  "/remote/path/file.txt",
			destination: ".",
			want:        "file.txt",
		},
		{
			name:        "non-existent path returned as-is",
			remotePath:  "/remote/path/file.txt",
			destination: "/non/existent/path",
			want:        "/non/existent/path",
		},
		{
			name:        "explicit file destination is returned as-is",
			remotePath:  "/remote/path/file.txt",
			destination: "/tmp/myfile.txt",
			want:        "/tmp/myfile.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveSSHDownloadDestination(tt.remotePath, tt.destination)
			if got != tt.want {
				t.Errorf("resolveSSHDownloadDestination(%q, %q) = %q; want %q",
					tt.remotePath, tt.destination, got, tt.want)
			}
		})
	}
}

func TestResolveSSHDownloadDestinationWithRealFilesystem(t *testing.T) {
	// Create a temp directory to test directory detection
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	got := resolveSSHDownloadDestination("/remote/file.txt", subDir)
	want := filepath.Join(subDir, "file.txt")
	if got != want {
		t.Errorf("resolveSSHDownloadDestination with dir = %q; want %q", got, want)
	}
}

func TestResolveSSHDownloadDestinationTrailingSlashJoins(t *testing.T) {
	// Path with trailing separator joins with filename
	got := resolveSSHDownloadDestination("/remote/file.txt", "/tmp/downloads/")
	want := "/tmp/downloads/file.txt"
	if got != want {
		t.Errorf("resolveSSHDownloadDestination with trailing slash = %q; want %q", got, want)
	}
}
