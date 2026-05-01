package ldapapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s Service) testConnection(ctx context.Context) LdapTestResult {
	cfg := loadConfig()
	if !cfg.isEnabled() {
		return LdapTestResult{Ok: false, Message: "LDAP is not enabled"}
	}

	type testPayload struct {
		Users      []ldapUserEntry
		GroupCount int
	}
	payload, err := withAdminBind(ctx, cfg, func(conn *ldap.Conn) (testPayload, error) {
		entries, err := searchUsers(ctx, conn, cfg, 100)
		if err != nil {
			return testPayload{}, err
		}
		groupCount := 0
		if strings.TrimSpace(cfg.GroupBaseDN) != "" {
			groupCount, err = countGroups(ctx, conn, cfg, 100)
			if err != nil {
				return testPayload{}, err
			}
		}
		return testPayload{Users: entries, GroupCount: groupCount}, nil
	})
	if err != nil {
		return LdapTestResult{Ok: false, Message: "Connection failed: " + err.Error()}
	}

	message := fmt.Sprintf("Connected successfully. Found %d user(s)", len(payload.Users))
	if strings.TrimSpace(cfg.GroupBaseDN) != "" {
		message += fmt.Sprintf(" and %d group(s)", payload.GroupCount)
	}
	return LdapTestResult{
		Ok:         true,
		Message:    message,
		UserCount:  len(payload.Users),
		GroupCount: payload.GroupCount,
	}
}

func (s Service) syncUsers(ctx context.Context) LdapSyncResult {
	result := LdapSyncResult{Errors: []string{}}
	cfg := loadConfig()
	if !cfg.isEnabled() {
		result.Errors = append(result.Errors, "LDAP is not enabled")
		return result
	}
	if s.DB == nil {
		result.Errors = append(result.Errors, "database is unavailable")
		return result
	}

	_ = s.insertAudit(ctx, "LDAP_SYNC_START", nil, nil, map[string]any{"provider": cfg.ProviderName})

	tenantExists := false
	if strings.TrimSpace(cfg.DefaultTenantID) != "" {
		var exists bool
		if err := s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM "Tenant" WHERE id = $1)`, cfg.DefaultTenantID).Scan(&exists); err == nil {
			tenantExists = exists
		}
	}

	ldapUsers, err := withAdminBind(ctx, cfg, func(conn *ldap.Conn) ([]ldapUserEntry, error) {
		entries, err := searchUsers(ctx, conn, cfg, 5000)
		return entries, err
	})
	if err != nil {
		message := err.Error()
		result.Errors = append(result.Errors, message)
		_ = s.insertAudit(ctx, "LDAP_SYNC_ERROR", nil, nil, map[string]any{"error": message})
		return result
	}

	seenEmails := make(map[string]struct{}, len(ldapUsers))
	for _, ldapUser := range ldapUsers {
		email := strings.ToLower(strings.TrimSpace(ldapUser.Email))
		if email == "" {
			continue
		}
		seenEmails[email] = struct{}{}

		if err := s.syncSingleUser(ctx, cfg, tenantExists, ldapUser, &result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", email, err.Error()))
		}
	}

	if err := s.disableMissingUsers(ctx, seenEmails, &result); err != nil {
		result.Errors = append(result.Errors, err.Error())
	}

	_ = s.insertAudit(ctx, "LDAP_SYNC_COMPLETE", nil, nil, map[string]any{
		"created":  result.Created,
		"updated":  result.Updated,
		"disabled": result.Disabled,
		"errors":   len(result.Errors),
	})

	return result
}

func (s Service) syncSingleUser(ctx context.Context, cfg ldapConfig, tenantExists bool, ldapUser ldapUserEntry, result *LdapSyncResult) error {
	attributes, err := json.Marshal(map[string]any{
		"dn":     ldapUser.DN,
		"uid":    ldapUser.UID,
		"groups": ldapUser.Groups,
	})
	if err != nil {
		return err
	}

	var (
		userID       string
		existingUser bool
		currentName  *string
	)
	err = s.DB.QueryRow(ctx, `
SELECT id, username
FROM "User"
WHERE LOWER(email) = LOWER($1)
`, ldapUser.Email).Scan(&userID, &currentName)
	if err == nil {
		existingUser = true
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	if !existingUser {
		if !cfg.AutoProvision {
			return nil
		}

		displayName := strings.TrimSpace(ldapUser.DisplayName)
		if displayName == "" {
			displayName = strings.TrimSpace(ldapUser.UID)
		}

		tx, err := s.DB.Begin(ctx)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback(ctx) }()

		if err := tx.QueryRow(ctx, `
INSERT INTO "User" (id, email, username, "vaultSetupComplete", "emailVerified")
VALUES ($1, $2, $3, $4, $5)
RETURNING id
`, uuid.NewString(), ldapUser.Email, nullableString(displayName), false, true).Scan(&userID); err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, `
INSERT INTO "OAuthAccount" (id, "userId", provider, "providerUserId", "providerEmail", "samlAttributes")
VALUES ($1, $2, 'LDAP', $3, $4, $5::jsonb)
`, uuid.NewString(), userID, ldapUser.ProviderUserID, ldapUser.Email, string(attributes)); err != nil {
			return err
		}

		if cfg.DefaultTenantID != "" && tenantExists {
			if _, err := tx.Exec(ctx, `
INSERT INTO "TenantMember" (id, "tenantId", "userId", role, status, "isActive")
VALUES ($1, $2, $3, 'MEMBER', 'ACCEPTED', false)
ON CONFLICT ("tenantId", "userId") DO NOTHING
`, uuid.NewString(), cfg.DefaultTenantID, userID); err != nil {
				return err
			}
		}

		if err := tx.Commit(ctx); err != nil {
			return err
		}

		createdUserID := userID
		_ = s.insertAudit(ctx, "LDAP_USER_CREATED", &createdUserID, nil, map[string]any{
			"email": ldapUser.Email,
			"uid":   ldapUser.UID,
			"dn":    ldapUser.DN,
		})
		result.Created++
		return nil
	}

	displayName := strings.TrimSpace(ldapUser.DisplayName)
	if displayName == "" {
		displayName = strings.TrimSpace(ldapUser.UID)
	}
	if displayName != "" && (currentName == nil || strings.TrimSpace(*currentName) != displayName) {
		if _, err := s.DB.Exec(ctx, `UPDATE "User" SET username = $2 WHERE id = $1`, userID, displayName); err != nil {
			return err
		}
		result.Updated++
	}

	var accountID string
	err = s.DB.QueryRow(ctx, `
SELECT id
FROM "OAuthAccount"
WHERE "userId" = $1
  AND provider = 'LDAP'
`, userID).Scan(&accountID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if strings.TrimSpace(accountID) == "" {
		if _, err := s.DB.Exec(ctx, `
INSERT INTO "OAuthAccount" (id, "userId", provider, "providerUserId", "providerEmail", "samlAttributes")
VALUES ($1, $2, 'LDAP', $3, $4, $5::jsonb)
`, uuid.NewString(), userID, ldapUser.ProviderUserID, ldapUser.Email, string(attributes)); err != nil {
			return err
		}
		return nil
	}

	_, err = s.DB.Exec(ctx, `
UPDATE "OAuthAccount"
SET "providerUserId" = $2,
    "providerEmail" = $3,
    "samlAttributes" = $4::jsonb
WHERE id = $1
`, accountID, ldapUser.ProviderUserID, ldapUser.Email, string(attributes))
	return err
}

func (s Service) disableMissingUsers(ctx context.Context, seenEmails map[string]struct{}, result *LdapSyncResult) error {
	rows, err := s.DB.Query(ctx, `
SELECT oa.id, u.id, u.email, u.enabled
FROM "OAuthAccount" oa
JOIN "User" u ON u.id = oa."userId"
WHERE oa.provider = 'LDAP'
`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			accountID string
			userID    string
			email     string
			enabled   bool
		)
		if err := rows.Scan(&accountID, &userID, &email, &enabled); err != nil {
			return err
		}
		if !enabled {
			continue
		}
		if _, ok := seenEmails[strings.ToLower(strings.TrimSpace(email))]; ok {
			continue
		}
		if _, err := s.DB.Exec(ctx, `UPDATE "User" SET enabled = false WHERE id = $1`, userID); err != nil {
			return err
		}
		disabledUserID := userID
		_ = s.insertAudit(ctx, "LDAP_USER_DISABLED", &disabledUserID, nil, map[string]any{
			"email":  email,
			"reason": "not_found_in_ldap",
		})
		result.Disabled++
		_ = accountID
	}
	return rows.Err()
}
