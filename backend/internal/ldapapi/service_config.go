package ldapapi

import (
	"errors"
	"os"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/authn"
)

func loadConfig() ldapConfig {
	return ldapConfig{
		Enabled:               os.Getenv("LDAP_ENABLED") == "true",
		ProviderName:          getenv("LDAP_PROVIDER_NAME", "LDAP"),
		ServerURL:             strings.TrimSpace(os.Getenv("LDAP_SERVER_URL")),
		BaseDN:                strings.TrimSpace(os.Getenv("LDAP_BASE_DN")),
		BindDN:                strings.TrimSpace(os.Getenv("LDAP_BIND_DN")),
		BindPassword:          strings.TrimSpace(os.Getenv("LDAP_BIND_PASSWORD")),
		UserSearchFilter:      getenv("LDAP_USER_SEARCH_FILTER", "(uid={{username}})"),
		UserSearchBase:        strings.TrimSpace(os.Getenv("LDAP_USER_SEARCH_BASE")),
		DisplayNameAttr:       getenv("LDAP_DISPLAY_NAME_ATTR", "displayName"),
		EmailAttr:             getenv("LDAP_EMAIL_ATTR", "mail"),
		UIDAttr:               getenv("LDAP_UID_ATTR", "uid"),
		GroupBaseDN:           strings.TrimSpace(os.Getenv("LDAP_GROUP_BASE_DN")),
		GroupSearchFilter:     getenv("LDAP_GROUP_SEARCH_FILTER", "(objectClass=groupOfNames)"),
		GroupMemberAttr:       getenv("LDAP_GROUP_MEMBER_ATTR", "member"),
		GroupNameAttr:         getenv("LDAP_GROUP_NAME_ATTR", "cn"),
		AllowedGroups:         splitCSV(os.Getenv("LDAP_ALLOWED_GROUPS")),
		StartTLS:              os.Getenv("LDAP_STARTTLS") == "true",
		TLSRejectUnauthorized: os.Getenv("LDAP_TLS_REJECT_UNAUTHORIZED") != "false",
		SyncEnabled:           os.Getenv("LDAP_SYNC_ENABLED") == "true",
		SyncCron:              getenv("LDAP_SYNC_CRON", "0 */6 * * *"),
		AutoProvision:         os.Getenv("LDAP_AUTO_PROVISION") != "false",
		DefaultTenantID:       strings.TrimSpace(os.Getenv("LDAP_DEFAULT_TENANT_ID")),
	}
}

func requireTenantAdmin(claims authn.Claims) error {
	if strings.TrimSpace(claims.TenantID) == "" {
		return errors.New("You must belong to an organization to perform this action")
	}
	if !hasTenantRole(claims.TenantRole, "ADMIN") {
		return errors.New("Insufficient tenant role")
	}
	return nil
}

func hasTenantRole(actual, minimum string) bool {
	hierarchy := map[string]int{
		"GUEST":      1,
		"AUDITOR":    2,
		"CONSULTANT": 3,
		"MEMBER":     4,
		"OPERATOR":   5,
		"ADMIN":      6,
		"OWNER":      7,
	}
	return hierarchy[strings.ToUpper(strings.TrimSpace(actual))] >= hierarchy[minimum]
}

func redactLDAPURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if idx := strings.Index(raw, "://"); idx >= 0 {
		rest := raw[idx+3:]
		if at := strings.LastIndex(rest, "@"); at >= 0 {
			return raw[:idx+3] + "***:***@" + rest[at+1:]
		}
	}
	return raw
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func nullableString(raw string) any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	return raw
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
