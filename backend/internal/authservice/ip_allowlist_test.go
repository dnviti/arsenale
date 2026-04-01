package authservice

import "testing"

func TestIsIPInCIDR(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ip      string
		cidr    string
		matches bool
	}{
		{name: "bare IPv4 match", ip: "10.0.0.1", cidr: "10.0.0.1", matches: true},
		{name: "bare IPv4 mismatch", ip: "10.0.0.2", cidr: "10.0.0.1", matches: false},
		{name: "IPv4 CIDR match", ip: "10.42.1.7", cidr: "10.0.0.0/8", matches: true},
		{name: "IPv4 CIDR mismatch", ip: "192.168.1.7", cidr: "10.0.0.0/8", matches: false},
		{name: "IPv6 CIDR match", ip: "2001:db8::1", cidr: "2001:db8::/32", matches: true},
		{name: "IPv6 CIDR mismatch", ip: "2001:dead::1", cidr: "2001:db8::/32", matches: false},
		{name: "mapped IPv4 match", ip: "::ffff:192.168.1.44", cidr: "192.168.1.0/24", matches: true},
		{name: "invalid prefix rejected", ip: "10.0.0.1", cidr: "10.0.0.0/99", matches: false},
		{name: "family mismatch rejected", ip: "10.0.0.1", cidr: "2001:db8::/32", matches: false},
		{name: "invalid input rejected", ip: "not-an-ip", cidr: "10.0.0.0/8", matches: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isIPInCIDR(tt.ip, tt.cidr); got != tt.matches {
				t.Fatalf("isIPInCIDR(%q, %q) = %v, want %v", tt.ip, tt.cidr, got, tt.matches)
			}
		})
	}
}

func TestEvaluateIPAllowlist(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		active  *loginMembership
		ip      string
		flagged bool
		blocked bool
	}{
		{name: "nil tenant allows", active: nil, ip: "10.0.0.1"},
		{name: "disabled allowlist allows", active: &loginMembership{IPAllowlistEnabled: false}, ip: "10.0.0.1"},
		{name: "empty entries allow all", active: &loginMembership{IPAllowlistEnabled: true, IPAllowlistMode: "block"}, ip: "10.0.0.1"},
		{name: "matching entry allows", active: &loginMembership{IPAllowlistEnabled: true, IPAllowlistMode: "block", IPAllowlistEntries: []string{"10.0.0.0/8"}}, ip: "10.2.3.4"},
		{name: "block mode rejects", active: &loginMembership{IPAllowlistEnabled: true, IPAllowlistMode: "block", IPAllowlistEntries: []string{"10.0.0.0/8"}}, ip: "192.168.1.9", blocked: true},
		{name: "flag mode flags", active: &loginMembership{IPAllowlistEnabled: true, IPAllowlistMode: "flag", IPAllowlistEntries: []string{"10.0.0.0/8"}}, ip: "192.168.1.9", flagged: true},
		{name: "default mode flags", active: &loginMembership{IPAllowlistEnabled: true, IPAllowlistEntries: []string{"10.0.0.0/8"}}, ip: "192.168.1.9", flagged: true},
		{name: "empty client ip blocks in block mode", active: &loginMembership{IPAllowlistEnabled: true, IPAllowlistMode: "block", IPAllowlistEntries: []string{"10.0.0.0/8"}}, ip: "", blocked: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			decision := evaluateIPAllowlist(tt.active, tt.ip)
			if decision.Flagged != tt.flagged {
				t.Fatalf("decision.Flagged = %v, want %v", decision.Flagged, tt.flagged)
			}
			if decision.Blocked != tt.blocked {
				t.Fatalf("decision.Blocked = %v, want %v", decision.Blocked, tt.blocked)
			}
		})
	}
}
