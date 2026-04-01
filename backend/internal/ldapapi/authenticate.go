package ldapapi

import (
	"context"
	"crypto/tls"
	"strings"

	"github.com/go-ldap/ldap/v3"
)

type AuthUser struct {
	DN             string
	UID            string
	Email          string
	DisplayName    string
	Groups         []string
	ProviderUserID string
}

func AuthenticateUser(ctx context.Context, identifier, password string) (*AuthUser, error) {
	cfg := loadConfig()
	if !cfg.isEnabled() {
		return nil, nil
	}

	entry, err := withAdminBind(ctx, cfg, func(conn *ldap.Conn) (*ldap.Entry, error) {
		req := ldap.NewSearchRequest(
			resolveUserSearchBase(cfg),
			ldap.ScopeWholeSubtree,
			ldap.NeverDerefAliases,
			1,
			0,
			false,
			resolveUserFilter(cfg, identifier),
			[]string{
				cfg.UIDAttr,
				cfg.EmailAttr,
				cfg.DisplayNameAttr,
				"entryUUID",
				"ipauniqueid",
				"nsuniqueid",
			},
			nil,
		)
		res, err := conn.Search(req)
		if err != nil {
			return nil, err
		}
		if len(res.Entries) == 0 {
			return nil, nil
		}

		user := parseUserEntry(res.Entries[0], cfg)
		if strings.TrimSpace(cfg.GroupBaseDN) != "" {
			if groups, err := fetchUserGroups(ctx, conn, cfg, user.DN); err == nil {
				user.Groups = groups
			}
		}
		if len(cfg.AllowedGroups) > 0 && !allowedGroup(user.Groups, cfg.AllowedGroups) {
			return nil, nil
		}

		entryCopy := *res.Entries[0]
		return &entryCopy, nil
	})
	if err != nil || entry == nil {
		return nil, err
	}

	userConn, err := openConnection(cfg)
	if err != nil {
		return nil, err
	}
	defer userConn.Close()

	if cfg.StartTLS && !strings.HasPrefix(strings.ToLower(cfg.ServerURL), "ldaps://") {
		if err := userConn.StartTLS(&tls.Config{InsecureSkipVerify: !cfg.TLSRejectUnauthorized}); err != nil {
			return nil, err
		}
	}
	if err := userConn.Bind(entry.DN, password); err != nil {
		return nil, nil
	}

	user := parseUserEntry(entry, cfg)
	if strings.TrimSpace(cfg.GroupBaseDN) != "" {
		if groups, err := withAdminBind(ctx, cfg, func(conn *ldap.Conn) ([]string, error) {
			return fetchUserGroups(ctx, conn, cfg, user.DN)
		}); err == nil {
			user.Groups = groups
		}
	}
	if len(cfg.AllowedGroups) > 0 && !allowedGroup(user.Groups, cfg.AllowedGroups) {
		return nil, nil
	}

	return &AuthUser{
		DN:             user.DN,
		UID:            user.UID,
		Email:          strings.ToLower(strings.TrimSpace(user.Email)),
		DisplayName:    strings.TrimSpace(user.DisplayName),
		Groups:         append([]string(nil), user.Groups...),
		ProviderUserID: strings.TrimSpace(user.ProviderUserID),
	}, nil
}

func resolveUserSearchBase(cfg ldapConfig) string {
	if strings.TrimSpace(cfg.UserSearchBase) != "" {
		return strings.TrimSpace(cfg.UserSearchBase)
	}
	return cfg.BaseDN
}

func resolveUserFilter(cfg ldapConfig, identifier string) string {
	filter := strings.TrimSpace(cfg.UserSearchFilter)
	if filter == "" {
		filter = "(uid={{username}})"
	}
	escaped := ldap.EscapeFilter(strings.TrimSpace(identifier))
	filter = strings.ReplaceAll(filter, "{{username}}", escaped)
	filter = strings.ReplaceAll(filter, "{{email}}", escaped)
	return filter
}
