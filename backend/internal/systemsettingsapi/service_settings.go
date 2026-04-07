package systemsettingsapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
)

func (s Service) listSettings(ctx context.Context, callerRole string) ([]SettingValue, error) {
	if err := ensureRegistryLoaded(); err != nil {
		return nil, err
	}

	dbValues := make(map[string]string)
	if s.DB != nil {
		rows, err := s.DB.Query(ctx, `SELECT key, value FROM "AppConfig"`)
		if err != nil {
			return nil, fmt.Errorf("load system settings: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var key string
			var value string
			if err := rows.Scan(&key, &value); err != nil {
				return nil, fmt.Errorf("scan system settings: %w", err)
			}
			dbValues[key] = value
		}
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("iterate system settings: %w", err)
		}
	}

	results := make([]SettingValue, 0, len(registry))
	for _, def := range registry {
		envRaw, envLocked := os.LookupEnv(def.EnvVar)
		var (
			value  any
			source string
		)
		switch {
		case envLocked:
			value = parseValue(envRaw, def.Type, def.Default)
			source = "env"
		case dbValues[def.Key] != "":
			value = parseValue(dbValues[def.Key], def.Type, def.Default)
			source = "db"
		default:
			if raw, ok := dbValues[def.Key]; ok {
				value = parseValue(raw, def.Type, def.Default)
				source = "db"
			} else {
				value = def.Default
				source = "default"
			}
		}

		if def.Sensitive && value != nil && fmt.Sprint(value) != "" {
			value = sensitiveMask
		}

		results = append(results, SettingValue{
			Key:             def.Key,
			Value:           value,
			Source:          source,
			EnvLocked:       envLocked,
			CanEdit:         !envLocked && roleAtLeast(callerRole, def.MinEditRole),
			Type:            def.Type,
			Default:         def.Default,
			Options:         def.Options,
			Group:           def.Group,
			Label:           def.Label,
			Description:     def.Description,
			RestartRequired: def.RestartRequired,
			Sensitive:       def.Sensitive,
		})
	}

	return results, nil
}

func (s Service) setSetting(ctx context.Context, key string, value any, userID string, callerRole string) (map[string]any, error) {
	if err := ensureRegistryLoaded(); err != nil {
		return nil, err
	}
	def, ok := lookupDef(key)
	if !ok {
		return nil, &requestError{status: http.StatusBadRequest, message: "Unknown setting key."}
	}
	if !roleAtLeast(callerRole, def.MinEditRole) {
		return nil, &requestError{status: http.StatusForbidden, message: "Insufficient role to modify this setting."}
	}
	if _, envLocked := os.LookupEnv(def.EnvVar); envLocked {
		return nil, &requestError{
			status:  http.StatusForbidden,
			message: fmt.Sprintf("Setting %q is locked by environment variable and cannot be changed via the admin panel.", key),
		}
	}
	if def.Sensitive {
		if raw, ok := value.(string); ok && raw == sensitiveMask {
			return map[string]any{"key": key, "value": sensitiveMask, "source": "db"}, nil
		}
	}
	if s.DB == nil {
		return nil, errors.New("database is unavailable")
	}

	serialized, redactedValue, err := serializeValue(value, def)
	if err != nil {
		return nil, &requestError{status: http.StatusBadRequest, message: err.Error()}
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin system setting update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
INSERT INTO "AppConfig" (key, value, "updatedAt")
VALUES ($1, $2, NOW())
ON CONFLICT (key)
DO UPDATE SET value = EXCLUDED.value, "updatedAt" = NOW()
`, key, serialized); err != nil {
		return nil, fmt.Errorf("upsert system setting: %w", err)
	}

	if err := insertAuditLog(ctx, tx, userID, key, redactedValue); err != nil {
		return nil, fmt.Errorf("audit system setting update: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit system setting update: %w", err)
	}

	return map[string]any{"key": key, "value": value, "source": "db"}, nil
}
