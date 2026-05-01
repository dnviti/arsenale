package connections

import (
	"net/netip"
	"testing"
)

func TestBlockedHostMessage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		policy hostPolicy
		want   string
	}{
		{
			name:   "block loopback and local",
			policy: hostPolicy{},
			want:   "Connections to loopback, local network, wildcard, link-local, metadata, or server interface addresses are not allowed",
		},
		{
			name:   "allow loopback",
			policy: hostPolicy{allowLoopback: true},
			want:   "Connections to local network, wildcard, link-local, metadata, or server interface addresses are not allowed",
		},
		{
			name:   "allow local",
			policy: hostPolicy{allowLocalNetwork: true},
			want:   "Connections to loopback, wildcard, link-local, metadata, or server interface addresses are not allowed",
		},
		{
			name:   "allow both",
			policy: hostPolicy{allowLoopback: true, allowLocalNetwork: true},
			want:   "Connections to wildcard, link-local, metadata, or server interface addresses are not allowed",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := blockedHostMessage(tc.policy); got != tc.want {
				t.Fatalf("blockedHostMessage() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestIsForbiddenConnectionAddr(t *testing.T) {
	t.Parallel()

	local := map[string]struct{}{
		"127.0.0.1": {},
		"10.0.0.7":  {},
	}

	cases := []struct {
		name   string
		addr   string
		policy hostPolicy
		want   bool
	}{
		{name: "wildcard ipv4", addr: "0.0.0.0", want: true},
		{name: "loopback blocked by default", addr: "127.0.0.1", want: true},
		{name: "loopback allowed when configured", addr: "127.0.0.1", policy: hostPolicy{allowLoopback: true}, want: false},
		{name: "private ipv4 blocked when local network disabled", addr: "10.0.0.7", want: true},
		{name: "private ipv4 still blocked when it is a server address", addr: "10.0.0.7", policy: hostPolicy{allowLocalNetwork: true}, want: true},
		{name: "private ipv4 allowed when remote and local network enabled", addr: "10.0.0.8", policy: hostPolicy{allowLocalNetwork: true}, want: false},
		{name: "link local metadata blocked", addr: "169.254.169.254", policy: hostPolicy{allowLoopback: true, allowLocalNetwork: true}, want: true},
		{name: "ipv6 ula blocked by default", addr: "fd00::1", want: true},
		{name: "ipv6 ula allowed when local network enabled", addr: "fd00::1", policy: hostPolicy{allowLocalNetwork: true}, want: false},
		{name: "ipv6 link local always blocked", addr: "fe80::1", policy: hostPolicy{allowLoopback: true, allowLocalNetwork: true}, want: true},
		{name: "ipv4 mapped loopback blocked", addr: "::ffff:127.0.0.1", want: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			addr, err := netip.ParseAddr(tc.addr)
			if err != nil {
				t.Fatalf("ParseAddr(%q): %v", tc.addr, err)
			}
			if got := isForbiddenConnectionAddr(addr, local, tc.policy); got != tc.want {
				t.Fatalf("isForbiddenConnectionAddr(%q) = %v, want %v", tc.addr, got, tc.want)
			}
		})
	}
}

func TestValidateConnectionHostLocalhost(t *testing.T) {
	t.Setenv("ALLOW_LOOPBACK", "false")
	t.Setenv("ALLOW_LOCAL_NETWORK", "true")

	if err := validateConnectionHost(t.Context(), "localhost"); err == nil {
		t.Fatal("validateConnectionHost(localhost) unexpectedly succeeded")
	}
}

func TestValidateConnectionHostBracketedIPv6(t *testing.T) {
	t.Setenv("ALLOW_LOOPBACK", "true")
	t.Setenv("ALLOW_LOCAL_NETWORK", "true")

	if err := validateConnectionHost(t.Context(), "[::1]"); err != nil {
		t.Fatalf("validateConnectionHost([::1]) = %v", err)
	}
}
