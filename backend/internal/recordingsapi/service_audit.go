package recordingsapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s Service) GetAuditTrail(ctx context.Context, recordingID string, claims authn.Claims) (auditTrailResponse, error) {
	if _, err := uuid.Parse(strings.TrimSpace(recordingID)); err != nil {
		return auditTrailResponse{}, &requestError{status: http.StatusBadRequest, message: "invalid recording id"}
	}

	var (
		sessionID sql.NullString
		userID    string
	)
	err := s.DB.QueryRow(ctx, `SELECT "sessionId", "userId" FROM "SessionRecording" WHERE id = $1`, recordingID).Scan(&sessionID, &userID)
	if err != nil {
		return auditTrailResponse{}, err
	}

	isOwner := userID == claims.UserID
	isAuditor := strings.TrimSpace(claims.TenantID) != "" && hasAnyRole(claims.TenantRole, "ADMIN", "OWNER", "AUDITOR")
	if !isOwner && !isAuditor {
		return auditTrailResponse{}, pgx.ErrNoRows
	}
	if !sessionID.Valid {
		return auditTrailResponse{Data: []auditTrailEntry{}, HasMore: false}, nil
	}

	args := []any{recordingID, sessionID.String}
	// Older audit entries may reference only the session id, while newer ones can also
	// include the recording id directly. Keep both predicates so historic trails stay visible.
	conditions := []string{`((details ->> 'sessionId') = $2 OR (details ->> 'recordingId') = $1)`}
	if !isAuditor {
		args = append(args, claims.UserID)
		conditions = append(conditions, fmt.Sprintf(`"userId" = $%d`, len(args)))
	}
	querySQL := `
SELECT id, "userId", action::text, "targetType", "targetId",
       CASE WHEN details IS NULL THEN NULL ELSE details::text END,
       "ipAddress", "gatewayId", "geoCountry", "geoCity", "geoCoords", flags, "createdAt"
FROM "AuditLog"
WHERE ` + strings.Join(conditions, " AND ") + `
ORDER BY "createdAt" ASC
LIMIT 201
`
	rows, err := s.DB.Query(ctx, querySQL, args...)
	if err != nil {
		return auditTrailResponse{}, fmt.Errorf("list audit trail: %w", err)
	}
	defer rows.Close()

	items := make([]auditTrailEntry, 0)
	for rows.Next() {
		item, err := scanAuditTrailEntry(rows)
		if err != nil {
			return auditTrailResponse{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return auditTrailResponse{}, fmt.Errorf("iterate audit trail: %w", err)
	}

	hasMore := len(items) > 200
	if hasMore {
		items = items[:200]
	}
	return auditTrailResponse{Data: items, HasMore: hasMore}, nil
}

func scanAuditTrailEntry(row interface{ Scan(dest ...any) error }) (auditTrailEntry, error) {
	var (
		item       auditTrailEntry
		userIDText sql.NullString
		targetType sql.NullString
		targetID   sql.NullString
		details    sql.NullString
		ipAddress  sql.NullString
		gatewayID  sql.NullString
		geoCountry sql.NullString
		geoCity    sql.NullString
		geoCoords  []float64
		flags      []string
	)
	if err := row.Scan(&item.ID, &userIDText, &item.Action, &targetType, &targetID, &details, &ipAddress, &gatewayID, &geoCountry, &geoCity, &geoCoords, &flags, &item.CreatedAt); err != nil {
		return auditTrailEntry{}, fmt.Errorf("scan audit trail: %w", err)
	}
	if userIDText.Valid {
		item.UserID = &userIDText.String
	}
	if targetType.Valid {
		item.TargetType = &targetType.String
	}
	if targetID.Valid {
		item.TargetID = &targetID.String
	}
	if details.Valid {
		item.Details = json.RawMessage(details.String)
	}
	if ipAddress.Valid {
		item.IPAddress = &ipAddress.String
	}
	if gatewayID.Valid {
		item.GatewayID = &gatewayID.String
	}
	if geoCountry.Valid {
		item.GeoCountry = &geoCountry.String
	}
	if geoCity.Valid {
		item.GeoCity = &geoCity.String
	}
	item.GeoCoords = geoCoords
	item.Flags = flags
	return item, nil
}

func hasAnyRole(role string, allowed ...string) bool {
	for _, candidate := range allowed {
		if role == candidate {
			return true
		}
	}
	return false
}

func (s Service) insertAuditLog(ctx context.Context, userID, action, targetID string, details map[string]any, ipAddress string) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("marshal audit details: %w", err)
	}
	var ip *string
	if strings.TrimSpace(ipAddress) != "" {
		ip = &ipAddress
	}
	_, err = s.DB.Exec(ctx, `
INSERT INTO "AuditLog" (id, "userId", action, "targetType", "targetId", details, "ipAddress")
VALUES ($1, $2, $3, 'Recording', $4, $5::jsonb, $6)
`, uuid.NewString(), userID, action, targetID, string(detailsJSON), ip)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}
