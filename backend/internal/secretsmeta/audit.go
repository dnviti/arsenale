package secretsmeta

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

func (s Service) insertAuditLog(ctx context.Context, userID, action, targetID string, details map[string]any, ipAddress string) error {
	return s.insertTypedAuditLog(ctx, userID, action, "VaultSecret", targetID, details, ipAddress)
}

func (s Service) insertTypedAuditLog(ctx context.Context, userID, action, targetType, targetID string, details map[string]any, ipAddress string) error {
	if s.DB == nil {
		return fmt.Errorf("database is unavailable")
	}

	var rawDetails []byte
	if details != nil {
		encoded, err := json.Marshal(details)
		if err != nil {
			return fmt.Errorf("marshal audit details: %w", err)
		}
		rawDetails = encoded
	}

	_, err := s.DB.Exec(ctx, `
INSERT INTO "AuditLog" (
	id,
	"userId",
	action,
	"targetType",
	"targetId",
	details,
	"ipAddress",
	"createdAt"
) VALUES ($1, $2, $3::"AuditAction", NULLIF($4, ''), NULLIF($5, ''), $6, NULLIF($7, ''), NOW())
`, uuid.NewString(), userID, action, strings.TrimSpace(targetType), strings.TrimSpace(targetID), rawDetails, strings.TrimSpace(ipAddress))
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

func requestIP(r *http.Request) string {
	for _, value := range []string{
		strings.TrimSpace(r.Header.Get("X-Real-IP")),
		firstForwardedFor(r.Header.Get("X-Forwarded-For")),
		strings.TrimSpace(r.RemoteAddr),
	} {
		if value == "" {
			continue
		}
		if host, _, err := net.SplitHostPort(value); err == nil {
			return strings.TrimPrefix(strings.TrimSpace(host), "::ffff:")
		}
		return strings.TrimPrefix(value, "::ffff:")
	}
	return ""
}

func firstForwardedFor(value string) string {
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			return part
		}
	}
	return ""
}
