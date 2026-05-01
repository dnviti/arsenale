package dbsessions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

func (s Service) GetQueryHistory(ctx context.Context, userID, sessionID string, limit int, search string) ([]QueryHistoryEntry, error) {
	if s.DB == nil {
		return nil, errors.New("postgres is not configured")
	}
	state, err := s.Store.LoadOwnedSessionState(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	args := []any{userID, state.Record.ID, limit}
	where := `WHERE "userId" = $1 AND "sessionId" = $2`
	if search != "" {
		args = append(args, "%"+search+"%")
		where += fmt.Sprintf(` AND "queryText" ILIKE $%d`, len(args))
	}

	rows, err := s.DB.Query(ctx, `
SELECT id, "queryText", "queryType"::text, "executionTimeMs", "rowsAffected", blocked, "createdAt", "blockReason", "connectionId", "tenantId"
FROM "DbAuditLog"
`+where+`
ORDER BY "createdAt" DESC
LIMIT $3
`, args...)
	if err != nil {
		return nil, fmt.Errorf("query db history: %w", err)
	}
	defer rows.Close()

	items := make([]QueryHistoryEntry, 0)
	for rows.Next() {
		var (
			item            QueryHistoryEntry
			executionTimeMS sql.NullInt32
			rowsAffected    sql.NullInt32
			blockReason     sql.NullString
			tenantID        sql.NullString
		)
		if err := rows.Scan(
			&item.ID,
			&item.QueryText,
			&item.QueryType,
			&executionTimeMS,
			&rowsAffected,
			&item.Blocked,
			&item.CreatedAt,
			&blockReason,
			&item.ConnectionID,
			&tenantID,
		); err != nil {
			return nil, fmt.Errorf("scan db history: %w", err)
		}
		if executionTimeMS.Valid {
			value := int(executionTimeMS.Int32)
			item.ExecutionTimeMS = &value
		}
		if rowsAffected.Valid {
			value := int(rowsAffected.Int32)
			item.RowsAffected = &value
		}
		if blockReason.Valid {
			item.BlockReason = &blockReason.String
		}
		if tenantID.Valid {
			item.TenantID = &tenantID.String
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate db history: %w", err)
	}
	return items, nil
}
