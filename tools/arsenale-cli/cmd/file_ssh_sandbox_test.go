package cmd

import (
	"strings"
	"testing"
)

func TestSSHFileHelpShowsSandboxRelativePaths(t *testing.T) {
	if fileSSHListCmd.Flags().Lookup("path").DefValue != "." {
		t.Fatalf("list --path default = %q; want .", fileSSHListCmd.Flags().Lookup("path").DefValue)
	}
	if !strings.Contains(strings.ToLower(fileSSHCmd.Short), "sandbox") {
		t.Fatalf("fileSSHCmd.Short = %q; want sandbox wording", fileSSHCmd.Short)
	}

	checks := []struct {
		name  string
		usage string
	}{
		{name: "list", usage: fileSSHListCmd.Flags().Lookup("path").Usage},
		{name: "mkdir", usage: fileSSHMkdirCmd.Flags().Lookup("path").Usage},
		{name: "delete", usage: fileSSHDeleteCmd.Flags().Lookup("path").Usage},
		{name: "download", usage: fileSSHDownloadCmd.Flags().Lookup("path").Usage},
		{name: "upload", usage: fileSSHUploadCmd.Flags().Lookup("to").Usage},
		{name: "rename from", usage: fileSSHRenameCmd.Flags().Lookup("from").Usage},
		{name: "rename to", usage: fileSSHRenameCmd.Flags().Lookup("to").Usage},
	}

	for _, tc := range checks {
		if !strings.Contains(tc.usage, "sandbox-relative") {
			t.Fatalf("%s usage = %q; want sandbox-relative wording", tc.name, tc.usage)
		}
		if !strings.Contains(tc.usage, "workspace/current/") {
			t.Fatalf("%s usage = %q; want workspace/current/ wording", tc.name, tc.usage)
		}
		if strings.Contains(strings.ToLower(tc.usage), "remote ssh target") {
			t.Fatalf("%s usage = %q; should not mention remote SSH target", tc.name, tc.usage)
		}
	}
}

func TestSSHFileRejectsAbsolutePaths(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		allowRoot bool
	}{
		{name: "root slash", input: "/", allowRoot: true},
		{name: "absolute unix", input: "/tmp/report.txt"},
		{name: "drive root", input: "C:/Users/test/report.txt"},
		{name: "drive root backslash", input: `C:\Users\test\report.txt`},
		{name: "uri", input: "file:///tmp/report.txt"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := normalizeSSHSandboxCLIPath(tc.input, tc.allowRoot)
			if err == nil {
				t.Fatal("expected error")
			}
			if err.Error() != sshSandboxRelativePathErrorText {
				t.Fatalf("error = %q; want %q", err.Error(), sshSandboxRelativePathErrorText)
			}
		})
	}

	for _, invalidPath := range []string{"../secret.txt", "docs/../secret.txt", "../../secret.txt", "s3://bucket/key"} {
		t.Run("reject "+invalidPath, func(t *testing.T) {
			_, err := normalizeSSHSandboxCLIPath(invalidPath, false)
			if err == nil {
				t.Fatal("expected error")
			}
			if err.Error() != sshSandboxRelativePathErrorText {
				t.Fatalf("error = %q; want %q", err.Error(), sshSandboxRelativePathErrorText)
			}
		})
	}

	if got, err := normalizeSSHSandboxCLIPath("docs/report.txt", false); err != nil || got != "docs/report.txt" {
		t.Fatalf("normalizeSSHSandboxCLIPath valid = %q, %v; want docs/report.txt, nil", got, err)
	}
	if _, err := normalizeSSHSandboxCLIPath(".", false); err == nil || err.Error() != sshSandboxRelativePathErrorText {
		t.Fatalf("normalizeSSHSandboxCLIPath dot without root allowance = %v; want %q", err, sshSandboxRelativePathErrorText)
	}
	if got, err := normalizeSSHSandboxCLIPath(".", true); err != nil || got != "." {
		t.Fatalf("normalizeSSHSandboxCLIPath root = %q, %v; want ., nil", got, err)
	}
}
