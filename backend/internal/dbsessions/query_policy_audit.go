package dbsessions

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
)

func (s Service) interceptQuery(ctx context.Context, userID, connectionID, tenantID, sessionID, queryText string, rowsAffected, executionTimeMS *int, blocked bool, blockReason string, executionPlan any) {
	if s.DB == nil || strings.TrimSpace(userID) == "" || strings.TrimSpace(connectionID) == "" || strings.TrimSpace(queryText) == "" {
		return
	}

	tables := extractTablesAccessed(queryText)
	queryType := classifyDBQuery(queryText)
	planJSON := "null"
	if executionPlan != nil {
		if raw, err := json.Marshal(executionPlan); err == nil {
			planJSON = string(raw)
		}
	}

	_, _ = s.DB.Exec(ctx, `
INSERT INTO "DbAuditLog" (
	id, "userId", "connectionId", "tenantId", "sessionId", "queryText", "queryType", "tablesAccessed",
	"rowsAffected", "executionTimeMs", blocked, "blockReason", "executionPlan", "createdAt"
) VALUES (
	$1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), $6, $7::"DbQueryType", $8::text[],
	$9, $10, $11, NULLIF($12, ''), $13::jsonb, NOW()
)`,
		uuid.NewString(),
		strings.TrimSpace(userID),
		strings.TrimSpace(connectionID),
		strings.TrimSpace(tenantID),
		strings.TrimSpace(sessionID),
		queryText,
		string(queryType),
		tables,
		rowsAffected,
		executionTimeMS,
		blocked,
		strings.TrimSpace(blockReason),
		planJSON,
	)
}

func (s Service) insertQueryAuditEvent(ctx context.Context, userID, action, targetID string, details map[string]any, ipAddress string) {
	if s.DB == nil || strings.TrimSpace(action) == "" {
		return
	}

	payload, err := json.Marshal(details)
	if err != nil {
		return
	}

	_, _ = s.DB.Exec(ctx, `
INSERT INTO "AuditLog" (id, "userId", action, "targetType", "targetId", details, "ipAddress")
VALUES ($1, NULLIF($2, ''), $3::"AuditAction", 'DatabaseQuery', NULLIF($4, ''), $5::jsonb, NULLIF($6, ''))
`,
		uuid.NewString(),
		strings.TrimSpace(userID),
		strings.TrimSpace(action),
		strings.TrimSpace(targetID),
		string(payload),
		strings.TrimSpace(ipAddress),
	)
}
