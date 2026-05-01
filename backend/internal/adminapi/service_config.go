package adminapi

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/emaildelivery"
	"github.com/jackc/pgx/v5"
)

func (s Service) getSelfSignupEnabled(ctx context.Context) (bool, error) {
	if selfSignupEnvLocked() {
		return false, nil
	}
	if s.DB == nil {
		return true, nil
	}

	var value string
	err := s.DB.QueryRow(ctx, `SELECT value FROM "AppConfig" WHERE key = 'selfSignupEnabled'`).Scan(&value)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return true, nil
		}
		return false, fmt.Errorf("query self-signup flag: %w", err)
	}
	return strings.EqualFold(strings.TrimSpace(value), "true"), nil
}

func (s Service) setSelfSignupEnabled(ctx context.Context, enabled bool, userID string) error {
	if selfSignupEnvLocked() {
		return &requestError{
			status:  403,
			message: "Self-signup is disabled at the environment level and cannot be changed via the admin panel.",
		}
	}
	if s.DB == nil {
		return errors.New("database is unavailable")
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin self-signup update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
INSERT INTO "AppConfig" (key, value, "updatedAt")
VALUES ('selfSignupEnabled', $1, NOW())
ON CONFLICT (key)
DO UPDATE SET value = EXCLUDED.value, "updatedAt" = NOW()
`, fmt.Sprintf("%t", enabled)); err != nil {
		return fmt.Errorf("upsert self-signup config: %w", err)
	}

	if err := insertAuditLog(ctx, tx, userID, "APP_CONFIG_UPDATE", map[string]any{
		"key":   "selfSignupEnabled",
		"value": enabled,
	}); err != nil {
		return fmt.Errorf("audit self-signup update: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit self-signup update: %w", err)
	}
	return nil
}

func selfSignupEnvLocked() bool {
	return os.Getenv("SELF_SIGNUP_ENABLED") != "true"
}

func buildEmailStatus() emailStatusResponse {
	deliveryStatus := emaildelivery.StatusFromEnv()

	status := emailStatusResponse{
		Provider:   deliveryStatus.Provider,
		Configured: deliveryStatus.Configured,
		From:       deliveryStatus.From,
	}

	switch strings.ToLower(strings.TrimSpace(deliveryStatus.Provider)) {
	case "smtp":
		status.Host = strings.TrimSpace(os.Getenv("SMTP_HOST"))
		status.Port = parseInt(getenv("SMTP_PORT", "587"), 587)
		status.Secure = status.Port == 465
	}

	return status
}

func buildAuthProviderDetails() []authProviderDetail {
	return []authProviderDetail{
		{Key: "google", Label: "Google", Enabled: strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")) != ""},
		{Key: "microsoft", Label: "Microsoft", Enabled: strings.TrimSpace(os.Getenv("MICROSOFT_CLIENT_ID")) != ""},
		{Key: "github", Label: "GitHub", Enabled: strings.TrimSpace(os.Getenv("GITHUB_CLIENT_ID")) != ""},
		{
			Key:          "oidc",
			Label:        "OIDC",
			Enabled:      strings.TrimSpace(os.Getenv("OIDC_CLIENT_ID")) != "",
			ProviderName: getenv("OIDC_PROVIDER_NAME", "SSO"),
		},
		{
			Key:          "saml",
			Label:        "SAML",
			Enabled:      strings.TrimSpace(os.Getenv("SAML_ENTRY_POINT")) != "",
			ProviderName: getenv("SAML_PROVIDER_NAME", "SAML SSO"),
		},
		{
			Key:          "ldap",
			Label:        "LDAP",
			Enabled:      os.Getenv("LDAP_ENABLED") == "true" && strings.TrimSpace(os.Getenv("LDAP_SERVER_URL")) != "",
			ProviderName: getenv("LDAP_PROVIDER_NAME", "LDAP"),
		},
	}
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
