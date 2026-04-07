package oauthapi

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

func (s Service) loadTenantAllowlist(ctx context.Context, userID string) (*tenantAllowlist, error) {
	if s.DB == nil {
		return nil, errors.New("database is unavailable")
	}

	rows, err := s.DB.Query(ctx, `
SELECT tm.status::text, tm."isActive", tm."joinedAt",
       t."ipAllowlistEnabled", t."ipAllowlistMode", t."ipAllowlistEntries"
FROM "TenantMember" tm
JOIN "Tenant" t ON t.id = tm."tenantId"
WHERE tm."userId" = $1
  AND (tm."expiresAt" IS NULL OR tm."expiresAt" > NOW())
ORDER BY tm."joinedAt" ASC
`, userID)
	if err != nil {
		return nil, fmt.Errorf("query oauth memberships: %w", err)
	}
	defer rows.Close()

	type membership struct {
		Status   string
		IsActive bool
		JoinedAt time.Time
		Allow    tenantAllowlist
	}
	var memberships []membership
	for rows.Next() {
		var (
			item    membership
			mode    sql.NullString
			entries []string
		)
		if err := rows.Scan(&item.Status, &item.IsActive, &item.JoinedAt, &item.Allow.Enabled, &mode, &entries); err != nil {
			return nil, fmt.Errorf("scan oauth membership: %w", err)
		}
		item.Allow.Mode = "flag"
		if mode.Valid && strings.TrimSpace(mode.String) != "" {
			item.Allow.Mode = strings.TrimSpace(mode.String)
		}
		item.Allow.Entries = entries
		memberships = append(memberships, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate oauth memberships: %w", err)
	}

	var accepted []membership
	for _, item := range memberships {
		if item.Status == "ACCEPTED" {
			accepted = append(accepted, item)
		}
		if item.Status == "ACCEPTED" && item.IsActive {
			allow := item.Allow
			return &allow, nil
		}
	}
	if len(accepted) == 1 {
		allow := accepted[0].Allow
		return &allow, nil
	}
	return nil, nil
}

func evaluateIPAllowlist(allowlist *tenantAllowlist, ipAddress string) ipAllowlistDecision {
	if allowlist == nil || !allowlist.Enabled {
		return ipAllowlistDecision{}
	}
	if isIPAllowed(ipAddress, allowlist.Entries) {
		return ipAllowlistDecision{}
	}
	if strings.EqualFold(strings.TrimSpace(allowlist.Mode), "block") {
		return ipAllowlistDecision{Blocked: true}
	}
	return ipAllowlistDecision{Flagged: true}
}

func (d ipAllowlistDecision) Flags() []string {
	if !d.Flagged {
		return nil
	}
	return []string{"UNTRUSTED_IP"}
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
