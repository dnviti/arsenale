package teams

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s Service) requireMembership(ctx context.Context, teamID, userID, tenantID string) (membership, error) {
	row := s.DB.QueryRow(ctx, `
SELECT tm.role::text, t."tenantId"
FROM "TeamMember" tm
JOIN "Team" t ON t.id = tm."teamId"
WHERE tm."teamId" = $1
  AND tm."userId" = $2
`, teamID, userID)

	var result membership
	if err := row.Scan(&result.Role, &result.TenantID); err != nil {
		return membership{}, err
	}
	if result.TenantID != tenantID {
		return membership{}, &requestError{status: 403, message: "Access denied"}
	}
	return result, nil
}

func sortTeamMembers(items []teamMemberResponse) {
	roleRank := map[string]int{
		"TEAM_VIEWER": 1,
		"TEAM_EDITOR": 2,
		"TEAM_ADMIN":  3,
	}
	sort.Slice(items, func(i, j int) bool {
		leftRank := roleRank[items[i].Role]
		rightRank := roleRank[items[j].Role]
		if leftRank != rightRank {
			return leftRank > rightRank
		}
		return items[i].JoinedAt.Before(items[j].JoinedAt)
	})
}

func writeError(w http.ResponseWriter, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		app.ErrorJSON(w, http.StatusNotFound, "Team not found")
		return
	}
	var reqErr *requestError
	if errors.As(err, &reqErr) {
		app.ErrorJSON(w, reqErr.status, reqErr.message)
		return
	}
	app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
}

func isValidTeamRole(role string) bool {
	switch strings.TrimSpace(role) {
	case "TEAM_ADMIN", "TEAM_EDITOR", "TEAM_VIEWER":
		return true
	default:
		return false
	}
}

func changedTeamFields(payload updateTeamPayload) []string {
	fields := make([]string, 0, 2)
	if payload.Name.Present {
		fields = append(fields, "name")
	}
	if payload.Description.Present {
		fields = append(fields, "description")
	}
	return fields
}

func requestIP(r *http.Request) string {
	for _, value := range []string{
		r.Header.Get("X-Real-IP"),
		firstForwardedFor(r.Header.Get("X-Forwarded-For")),
		r.RemoteAddr,
	} {
		ip := stripIP(value)
		if ip != "" {
			return ip
		}
	}
	return ""
}

func firstForwardedFor(value string) string {
	parts := strings.Split(value, ",")
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

func stripIP(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		return host
	}
	return value
}

func insertAuditLog(ctx context.Context, tx pgx.Tx, userID, action, targetType, targetID string, details map[string]any, ipAddress string) error {
	var payload any
	if details != nil {
		rawDetails, err := json.Marshal(details)
		if err != nil {
			return fmt.Errorf("marshal audit details: %w", err)
		}
		payload = string(rawDetails)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO "AuditLog" (id, "userId", action, "targetType", "targetId", details, "ipAddress")
VALUES ($1, $2, $3::"AuditAction", $4, $5, $6::jsonb, NULLIF($7, ''))
`, uuid.NewString(), userID, action, targetType, targetID, payload, ipAddress); err != nil {
		return err
	}
	return nil
}
