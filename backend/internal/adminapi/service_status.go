package adminapi

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/storage"
)

func (s Service) getDBStatus(ctx context.Context) (dbStatusResponse, error) {
	databaseURL, err := storage.DatabaseURLFromEnv()
	if err != nil {
		return dbStatusResponse{}, fmt.Errorf("resolve database url: %w", err)
	}

	status := dbStatusResponse{Port: 5432}
	if databaseURL != "" {
		if parsed, parseErr := url.Parse(databaseURL); parseErr == nil {
			status.Host = parsed.Hostname()
			status.Database = strings.TrimPrefix(parsed.Path, "/")
			if parsed.Port() != "" {
				if port, convErr := strconv.Atoi(parsed.Port()); convErr == nil {
					status.Port = port
				}
			}
		}
	}

	if s.DB == nil {
		return status, nil
	}

	var rawVersion string
	if err := s.DB.QueryRow(ctx, `SELECT version()`).Scan(&rawVersion); err != nil {
		return status, nil
	}
	status.Connected = true
	status.Version = sanitizeDBVersion(rawVersion)
	return status, nil
}

func sanitizeDBVersion(raw string) any {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	fields := strings.Fields(trimmed)
	if len(fields) >= 2 {
		return fields[0] + " " + fields[1]
	}
	return "connected"
}

func parseInt(value string, fallback int) int {
	var parsed int
	if _, err := fmt.Sscanf(strings.TrimSpace(value), "%d", &parsed); err != nil || parsed == 0 {
		return fallback
	}
	return parsed
}
