package dbsessions

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

func (s Service) recordSessionError(ctx context.Context, userID, connectionID, ipAddress string, err error) {
	if s.DB == nil || strings.TrimSpace(connectionID) == "" {
		return
	}

	rawDetails, marshalErr := json.Marshal(map[string]any{
		"protocol": "DATABASE",
		"error":    err.Error(),
	})
	if marshalErr != nil {
		return
	}

	_, _ = s.DB.Exec(
		ctx,
		`INSERT INTO "AuditLog" (
			id, "userId", action, "targetType", "targetId", details, "ipAddress", "geoCoords", flags
		) VALUES (
			$1, NULLIF($2, ''), 'SESSION_ERROR'::"AuditAction", 'Connection', NULLIF($3, ''), $4::jsonb, NULLIF($5, ''), ARRAY[]::double precision[], ARRAY[]::text[]
		)`,
		uuid.NewString(),
		strings.TrimSpace(userID),
		connectionID,
		string(rawDetails),
		strings.TrimSpace(ipAddress),
	)
}

func requestIP(r *http.Request) string {
	for _, value := range []string{
		strings.TrimSpace(r.Header.Get("X-Real-IP")),
		firstForwardedHeader(r.Header.Get("X-Forwarded-For")),
		strings.TrimSpace(r.RemoteAddr),
	} {
		value = stripPort(value)
		value = strings.TrimPrefix(value, "::ffff:")
		if value != "" {
			return value
		}
	}
	return ""
}

func stripPort(value string) string {
	host, _, err := net.SplitHostPort(value)
	if err == nil {
		return host
	}
	return value
}

func firstForwardedHeader(value string) string {
	if value == "" {
		return ""
	}
	parts := strings.Split(value, ",")
	return strings.TrimSpace(parts[0])
}
