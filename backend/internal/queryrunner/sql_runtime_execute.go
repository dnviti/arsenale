package queryrunner

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func executeSQLReadOnly(ctx context.Context, target *contracts.DatabaseTarget, req contracts.QueryExecutionRequest) (contracts.QueryExecutionResponse, error) {
	if err := ValidateReadOnlySQL(req.SQL); err != nil {
		return contracts.QueryExecutionResponse{}, err
	}
	return executeSQLAny(ctx, target, req)
}

func executeSQLAny(ctx context.Context, target *contracts.DatabaseTarget, req contracts.QueryExecutionRequest) (contracts.QueryExecutionResponse, error) {
	maxRows := req.MaxRows
	switch {
	case maxRows <= 0:
		maxRows = defaultMaxRows
	case maxRows > maxAllowedRows:
		maxRows = maxAllowedRows
	}

	sqlConn, err := openSQLTargetConn(ctx, target)
	if err != nil {
		return contracts.QueryExecutionResponse{}, err
	}
	defer sqlConn.Close()

	statements := splitStatements(req.SQL)
	if len(statements) == 0 {
		return contracts.QueryExecutionResponse{}, fmt.Errorf("sql is required")
	}

	start := time.Now()
	var result contracts.QueryExecutionResponse
	for _, stmt := range statements {
		queryCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
		result, err = executeSingleSQLStatement(queryCtx, sqlConn.conn, stmt, maxRows)
		cancel()
		if err != nil {
			return contracts.QueryExecutionResponse{}, err
		}
	}
	result.DurationMs = time.Since(start).Milliseconds()
	return result, nil
}

func executeSingleSQLStatement(ctx context.Context, conn *sql.Conn, stmt string, maxRows int) (contracts.QueryExecutionResponse, error) {
	if statementReturnsRows(stmt) {
		rows, err := conn.QueryContext(ctx, stmt)
		if err != nil {
			return contracts.QueryExecutionResponse{}, fmt.Errorf("execute query: %w", err)
		}
		defer rows.Close()

		result, err := scanSQLRows(rows, maxRows)
		if err != nil {
			return contracts.QueryExecutionResponse{}, err
		}
		return result, nil
	}

	execResult, err := conn.ExecContext(ctx, stmt)
	if err != nil {
		return contracts.QueryExecutionResponse{}, fmt.Errorf("execute query: %w", err)
	}

	rowsAffected, _ := execResult.RowsAffected()
	return contracts.QueryExecutionResponse{
		Columns:  []string{},
		Rows:     []map[string]any{},
		RowCount: int(rowsAffected),
	}, nil
}

func scanSQLRows(rows *sql.Rows, maxRows int) (contracts.QueryExecutionResponse, error) {
	columns, err := rows.Columns()
	if err != nil {
		return contracts.QueryExecutionResponse{}, fmt.Errorf("read columns: %w", err)
	}

	result := contracts.QueryExecutionResponse{
		Columns: columns,
		Rows:    make([]map[string]any, 0, min(maxRows, 16)),
	}

	for rows.Next() {
		values := make([]any, len(columns))
		dest := make([]any, len(columns))
		for i := range values {
			dest[i] = &values[i]
		}

		if err := rows.Scan(dest...); err != nil {
			return contracts.QueryExecutionResponse{}, fmt.Errorf("scan row values: %w", err)
		}

		result.RowCount++
		if len(result.Rows) < maxRows {
			row := make(map[string]any, len(columns))
			for idx, column := range columns {
				row[column] = normalizeValue(values[idx])
			}
			result.Rows = append(result.Rows, row)
		} else {
			result.Truncated = true
			break
		}
	}

	if err := rows.Err(); err != nil {
		return contracts.QueryExecutionResponse{}, fmt.Errorf("iterate rows: %w", err)
	}

	return result, nil
}

func statementReturnsRows(stmt string) bool {
	switch firstKeyword(normalizeSQL(stmt)) {
	case "select", "with", "show", "describe", "desc", "explain", "call", "exec", "execute":
		return true
	default:
		return false
	}
}
