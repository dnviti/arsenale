package desktopsessions

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpaqueGrantRoundTripAndHash(t *testing.T) {
	t.Parallel()

	id, secret, grant, err := newOpaqueGrant()
	if err != nil {
		t.Fatalf("newOpaqueGrant() error = %v", err)
	}
	gotID, gotSecret, err := splitOpaqueGrant(grant)
	if err != nil {
		t.Fatalf("splitOpaqueGrant() error = %v", err)
	}
	if gotID != id || gotSecret != secret {
		t.Fatalf("split grant = %q/%q; want %q/%q", gotID, gotSecret, id, secret)
	}
	if !opaqueSecretMatches(secret, hashOpaqueSecret(secret)) {
		t.Fatal("expected secret hash to match")
	}
	if opaqueSecretMatches(secret+"x", hashOpaqueSecret(secret)) {
		t.Fatal("expected modified secret not to match")
	}
}

func TestDesktopLaunchURLUsesConfiguredClientURL(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("POST", "http://api.internal/api/cli/connect/desktop/launch", strings.NewReader("{}"))
	service := Service{ClientURL: "https://arsenale.example.com/app/"}

	got := service.desktopLaunchURL(req, "grant-token")
	want := "https://arsenale.example.com/app/cli/desktop-launch?grant=grant-token"
	if got != want {
		t.Fatalf("desktopLaunchURL() = %q; want %q", got, want)
	}
}

func TestNormalizeDesktopLaunchProtocol(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"rdp", " RDP ", "vnc", "VNC"} {
		if _, err := normalizeDesktopLaunchProtocol(raw); err != nil {
			t.Fatalf("normalizeDesktopLaunchProtocol(%q) error = %v", raw, err)
		}
	}
	if _, err := normalizeDesktopLaunchProtocol("ssh"); err == nil {
		t.Fatal("expected unsupported protocol error")
	}
}
