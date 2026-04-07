package rdgatewayapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (s Service) GetConfig(ctx context.Context) (Config, error) {
	config := defaultConfig()
	if s.DB == nil {
		return config, nil
	}

	var raw string
	err := s.DB.QueryRow(ctx, `SELECT value FROM "AppConfig" WHERE key = 'rdGatewayConfig'`).Scan(&raw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return config, nil
		}
		return Config{}, fmt.Errorf("load rd gateway config: %w", err)
	}

	var partial Config
	if err := json.Unmarshal([]byte(raw), &partial); err != nil {
		return config, nil
	}

	if partial.ExternalHostname != "" {
		config.ExternalHostname = partial.ExternalHostname
	}
	if partial.Port != 0 {
		config.Port = partial.Port
	}
	if partial.IdleTimeoutSeconds != 0 {
		config.IdleTimeoutSeconds = partial.IdleTimeoutSeconds
	}
	config.Enabled = partial.Enabled
	return config, nil
}

func (s Service) UpsertConfig(ctx context.Context, cfg Config) error {
	if s.DB == nil {
		return errors.New("database is unavailable")
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal rd gateway config: %w", err)
	}
	_, err = s.DB.Exec(ctx, `
INSERT INTO "AppConfig" (key, value, "updatedAt")
VALUES ('rdGatewayConfig', $1, NOW())
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, "updatedAt" = NOW()
`, string(raw))
	if err != nil {
		return fmt.Errorf("upsert rd gateway config: %w", err)
	}
	return nil
}

func defaultConfig() Config {
	return Config{
		Enabled:            false,
		ExternalHostname:   "",
		Port:               443,
		IdleTimeoutSeconds: 3600,
	}
}
