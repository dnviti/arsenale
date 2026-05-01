package importexportapi

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/google/uuid"
)

func (s Service) insertAuditLog(ctx context.Context, userID, action, targetID string, details map[string]any, ip *string) error {
	var payload any
	if details != nil {
		payload = details
	}
	_, err := s.DB.Exec(ctx, `
INSERT INTO "AuditLog" (id, "userId", action, "targetType", "targetId", details, "ipAddress")
VALUES ($1, $2, $3::"AuditAction", 'Connection', NULLIF($4, ''), $5, $6)
`, uuid.NewString(), userID, action, targetID, payload, ip)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

func writeError(w http.ResponseWriter, err error) {
	var reqErr *requestError
	if errors.As(err, &reqErr) {
		app.ErrorJSON(w, reqErr.status, reqErr.message)
		return
	}
	app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
}

func requestIP(r *http.Request) *string {
	if r == nil {
		return nil
	}
	candidates := []string{
		r.Header.Get("X-Real-IP"),
		firstForwardedFor(r.Header.Get("X-Forwarded-For")),
		stripPort(r.RemoteAddr),
	}
	for _, candidate := range candidates {
		value := strings.TrimSpace(candidate)
		if value != "" {
			return &value
		}
	}
	return nil
}

func firstForwardedFor(value string) string {
	for _, item := range strings.Split(value, ",") {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func stripPort(value string) string {
	if host, _, err := net.SplitHostPort(strings.TrimSpace(value)); err == nil {
		return host
	}
	return strings.TrimSpace(value)
}
