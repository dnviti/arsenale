package ldapapi

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
)

func withAdminBind[T any](ctx context.Context, cfg ldapConfig, fn func(*ldap.Conn) (T, error)) (T, error) {
	var zero T
	conn, err := openConnection(cfg)
	if err != nil {
		return zero, err
	}
	defer conn.Close()

	if cfg.StartTLS && !strings.HasPrefix(strings.ToLower(cfg.ServerURL), "ldaps://") {
		if err := conn.StartTLS(&tls.Config{InsecureSkipVerify: !cfg.TLSRejectUnauthorized}); err != nil {
			return zero, err
		}
	}
	if err := conn.Bind(cfg.BindDN, cfg.BindPassword); err != nil {
		return zero, err
	}
	return fn(conn)
}

func openConnection(cfg ldapConfig) (*ldap.Conn, error) {
	conn, err := ldap.DialURL(cfg.ServerURL, ldap.DialWithTLSConfig(&tls.Config{
		InsecureSkipVerify: !cfg.TLSRejectUnauthorized,
	}))
	if err != nil {
		return nil, err
	}
	conn.SetTimeout(15 * time.Second)
	return conn, nil
}

func searchUsers(ctx context.Context, conn *ldap.Conn, cfg ldapConfig, sizeLimit int) ([]ldapUserEntry, error) {
	filter := strings.TrimSpace(cfg.UserSearchFilter)
	filter = strings.ReplaceAll(filter, "{{username}}", "*")
	filter = strings.ReplaceAll(filter, "{{email}}", "*")
	if filter == "" {
		filter = "(objectClass=person)"
	}

	searchBase := cfg.BaseDN
	if strings.TrimSpace(cfg.UserSearchBase) != "" {
		searchBase = cfg.UserSearchBase
	}

	req := ldap.NewSearchRequest(
		searchBase,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		sizeLimit,
		0,
		false,
		filter,
		[]string{cfg.UIDAttr, cfg.EmailAttr, cfg.DisplayNameAttr, "entryUUID", "ipauniqueid", "nsuniqueid"},
		nil,
	)
	res, err := conn.Search(req)
	if err != nil {
		return nil, err
	}

	items := make([]ldapUserEntry, 0, len(res.Entries))
	for _, entry := range res.Entries {
		user := parseUserEntry(entry, cfg)
		if strings.TrimSpace(user.Email) == "" {
			continue
		}
		if strings.TrimSpace(cfg.GroupBaseDN) != "" {
			groups, err := fetchUserGroups(ctx, conn, cfg, user.DN)
			if err == nil {
				user.Groups = groups
			}
		}
		if len(cfg.AllowedGroups) > 0 {
			if !allowedGroup(user.Groups, cfg.AllowedGroups) {
				continue
			}
		}
		items = append(items, user)
	}
	return items, nil
}

func countGroups(ctx context.Context, conn *ldap.Conn, cfg ldapConfig, sizeLimit int) (int, error) {
	req := ldap.NewSearchRequest(
		cfg.GroupBaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		sizeLimit,
		0,
		false,
		cfg.GroupSearchFilter,
		[]string{cfg.GroupNameAttr},
		nil,
	)
	res, err := conn.Search(req)
	if err != nil {
		return 0, err
	}
	return len(res.Entries), nil
}

func fetchUserGroups(ctx context.Context, conn *ldap.Conn, cfg ldapConfig, userDN string) ([]string, error) {
	baseFilter := strings.TrimSpace(cfg.GroupSearchFilter)
	baseFilter = strings.TrimPrefix(baseFilter, "(")
	baseFilter = strings.TrimSuffix(baseFilter, ")")
	filter := fmt.Sprintf("(&(%s)(%s=%s))", baseFilter, cfg.GroupMemberAttr, ldap.EscapeFilter(userDN))
	req := ldap.NewSearchRequest(
		cfg.GroupBaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		1000,
		0,
		false,
		filter,
		[]string{cfg.GroupNameAttr},
		nil,
	)
	res, err := conn.Search(req)
	if err != nil {
		return nil, err
	}
	groups := make([]string, 0, len(res.Entries))
	for _, entry := range res.Entries {
		name := strings.TrimSpace(entry.GetAttributeValue(cfg.GroupNameAttr))
		if name != "" {
			groups = append(groups, name)
		}
	}
	return groups, nil
}

func parseUserEntry(entry *ldap.Entry, cfg ldapConfig) ldapUserEntry {
	providerUserID := firstNonEmpty(
		entry.GetAttributeValue("entryUUID"),
		entry.GetAttributeValue("ipauniqueid"),
		entry.GetAttributeValue("nsuniqueid"),
		entry.GetAttributeValue(cfg.UIDAttr),
	)
	return ldapUserEntry{
		DN:             entry.DN,
		UID:            entry.GetAttributeValue(cfg.UIDAttr),
		Email:          entry.GetAttributeValue(cfg.EmailAttr),
		DisplayName:    entry.GetAttributeValue(cfg.DisplayNameAttr),
		ProviderUserID: providerUserID,
	}
}

func allowedGroup(userGroups, allowed []string) bool {
	for _, userGroup := range userGroups {
		for _, allow := range allowed {
			if strings.EqualFold(strings.TrimSpace(userGroup), strings.TrimSpace(allow)) {
				return true
			}
		}
	}
	return false
}
