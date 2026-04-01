package authservice

import (
	"context"
	"net/netip"
	"strconv"
	"strings"
)

const untrustedIPFlag = "UNTRUSTED_IP"

type ipAllowlistDecision struct {
	Flagged bool
	Blocked bool
}

func evaluateIPAllowlist(active *loginMembership, ipAddress string) ipAllowlistDecision {
	if active == nil || !active.IPAllowlistEnabled {
		return ipAllowlistDecision{}
	}
	if isIPAllowed(ipAddress, active.IPAllowlistEntries) {
		return ipAllowlistDecision{}
	}

	if strings.EqualFold(strings.TrimSpace(active.IPAllowlistMode), "block") {
		return ipAllowlistDecision{Blocked: true}
	}
	return ipAllowlistDecision{Flagged: true}
}

func (d ipAllowlistDecision) Flags() []string {
	if !d.Flagged {
		return nil
	}
	return []string{untrustedIPFlag}
}

func isIPAllowed(ipAddress string, entries []string) bool {
	if len(entries) == 0 {
		return true
	}
	for _, entry := range entries {
		if isIPInCIDR(ipAddress, entry) {
			return true
		}
	}
	return false
}

func isIPInCIDR(ipAddress, cidr string) bool {
	ipAddress = normalizeIP(ipAddress)
	cidr = strings.TrimSpace(cidr)
	if ipAddress == "" || cidr == "" {
		return false
	}

	addr, err := netip.ParseAddr(ipAddress)
	if err != nil {
		return false
	}

	slash := strings.LastIndexByte(cidr, '/')
	if slash == -1 {
		target, err := netip.ParseAddr(normalizeIP(cidr))
		if err != nil {
			return false
		}
		return target == addr
	}

	base := normalizeIP(cidr[:slash])
	prefixLen, err := strconv.Atoi(strings.TrimSpace(cidr[slash+1:]))
	if err != nil {
		return false
	}

	baseAddr, err := netip.ParseAddr(base)
	if err != nil || baseAddr.BitLen() != addr.BitLen() {
		return false
	}
	if prefixLen < 0 || prefixLen > baseAddr.BitLen() {
		return false
	}

	return netip.PrefixFrom(baseAddr, prefixLen).Masked().Contains(addr)
}

func (s Service) rejectBlockedIPAllowlist(ctx context.Context, userID, ipAddress string) error {
	_ = s.insertStandaloneAuditLog(ctx, &userID, "LOGIN_FAILURE", map[string]any{
		"reason": "ip_not_allowed",
	}, ipAddress)
	_ = s.clearVaultSessions(ctx, userID)
	return &requestError{
		status:  403,
		message: "Access denied: IP address not in tenant allowlist",
	}
}
