package sshproxyapi

import (
	"strings"
	"testing"
)

func TestSSHHostKeyCallbackFallsBackWhenNoKnownHostsExist(t *testing.T) {
	callback, err := sshHostKeyCallbackForPaths("", nil)
	if err != nil {
		t.Fatalf("callback error = %v", err)
	}
	if callback == nil {
		t.Fatal("callback is nil")
	}
}

func TestSSHHostKeyCallbackRejectsMissingConfiguredKnownHosts(t *testing.T) {
	_, err := sshHostKeyCallbackForPaths("/tmp/arsenale-missing-known-hosts", nil)
	if err == nil || !strings.Contains(err.Error(), "configured SSH proxy known_hosts file does not exist") {
		t.Fatalf("error = %v; want missing configured known_hosts", err)
	}
}
