package cmd

import "testing"

func TestSessionCreateSSHColumnsExposeManagedFileFlags(t *testing.T) {
	if got := sessionCreateColumnsSSH[2].Header; got != "SFTP_SUPPORTED" {
		t.Fatalf("sessionCreateColumnsSSH[2].Header = %q; want SFTP_SUPPORTED", got)
	}
	if got := sessionCreateColumnsSSH[2].Field; got != "sftpSupported" {
		t.Fatalf("sessionCreateColumnsSSH[2].Field = %q; want sftpSupported", got)
	}
	if got := sessionCreateColumnsSSH[3].Header; got != "FILE_BROWSER_SUPPORTED" {
		t.Fatalf("sessionCreateColumnsSSH[3].Header = %q; want FILE_BROWSER_SUPPORTED", got)
	}
	if got := sessionCreateColumnsSSH[3].Field; got != "fileBrowserSupported" {
		t.Fatalf("sessionCreateColumnsSSH[3].Field = %q; want fileBrowserSupported", got)
	}
}

func TestSessionCreateRDPColumnsExposeDriveFlag(t *testing.T) {
	if got := sessionCreateColumnsRDP[1].Header; got != "ENABLE_DRIVE" {
		t.Fatalf("sessionCreateColumnsRDP[1].Header = %q; want ENABLE_DRIVE", got)
	}
	if got := sessionCreateColumnsRDP[1].Field; got != "enableDrive" {
		t.Fatalf("sessionCreateColumnsRDP[1].Field = %q; want enableDrive", got)
	}
}
