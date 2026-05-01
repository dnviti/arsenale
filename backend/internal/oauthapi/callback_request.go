package oauthapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

func (s Service) insertStandaloneAuditLog(ctx context.Context, userID *string, action string, details map[string]any, ipAddress string, flags []string) error {
	if s.DB == nil {
		return nil
	}
	rawDetails, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("marshal audit details: %w", err)
	}

	query := `INSERT INTO "AuditLog" (id, "userId", action, details, "ipAddress") VALUES ($1, $2, $3::"AuditAction", $4::jsonb, NULLIF($5, ''))`
	args := []any{uuid.NewString(), userID, action, string(rawDetails), ipAddress}
	if len(flags) > 0 {
		query = `INSERT INTO "AuditLog" (id, "userId", action, details, "ipAddress", flags) VALUES ($1, $2, $3::"AuditAction", $4::jsonb, NULLIF($5, ''), $6::text[])`
		args = append(args, flags)
	}
	if _, err := s.DB.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

func callbackErrorCode(err error) string {
	var reqErr *requestError
	if errors.As(err, &reqErr) && reqErr.status == http.StatusForbidden {
		if strings.Contains(strings.ToLower(reqErr.message), "disabled") {
			return "account_disabled"
		}
		return "registration_disabled"
	}
	return "authentication_failed"
}

func requestIP(r *http.Request) string {
	for _, value := range []string{
		r.Header.Get("X-Real-IP"),
		firstForwardedFor(r.Header.Get("X-Forwarded-For")),
		r.RemoteAddr,
	} {
		if ip := stripIP(value); ip != "" {
			return ip
		}
	}
	return ""
}

func firstForwardedFor(value string) string {
	for i, ch := range value {
		if ch == ',' {
			return value[:i]
		}
	}
	return value
}

func stripIP(value string) string {
	value = normalizeIP(value)
	if value == "" {
		return ""
	}
	return value
}

func normalizeIP(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}
	return strings.TrimPrefix(value, "::ffff:")
}

func nullableString(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}
