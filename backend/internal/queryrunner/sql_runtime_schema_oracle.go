package queryrunner

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func fetchOracleSchema(ctx context.Context, target *contracts.DatabaseTarget) (contracts.SchemaInfo, error) {
	sqlConn, err := openSQLTargetConn(ctx, target)
	if err != nil {
		return contracts.SchemaInfo{}, err
	}
	defer sqlConn.Close()

	result := emptySchemaInfo()
	tables, err := loadTableRefs(ctx, sqlConn.conn, `
SELECT USER, table_name
FROM user_tables
ORDER BY table_name
`)
	if err != nil {
		return contracts.SchemaInfo{}, fmt.Errorf("fetch tables: %w", err)
	}
	for _, table := range tables {
		item, err := loadOracleTable(ctx, sqlConn.conn, table)
		if err != nil {
			return contracts.SchemaInfo{}, err
		}
		result.Tables = append(result.Tables, item)
	}
	if err := loadSchemaViews(ctx, sqlConn.conn, &result, `
SELECT USER, view_name, 0
FROM user_views
ORDER BY view_name
`); err != nil {
		return contracts.SchemaInfo{}, fmt.Errorf("fetch views: %w", err)
	}
	if err := loadSchemaRoutines(ctx, sqlConn.conn, `
SELECT USER, object_name, ''
FROM user_objects
WHERE object_type = 'FUNCTION'
ORDER BY object_name
`, &result.Functions); err != nil {
		return contracts.SchemaInfo{}, fmt.Errorf("fetch functions: %w", err)
	}
	if err := loadSchemaRoutines(ctx, sqlConn.conn, `
SELECT USER, object_name, ''
FROM user_objects
WHERE object_type = 'PROCEDURE'
ORDER BY object_name
`, &result.Procedures); err != nil {
		return contracts.SchemaInfo{}, fmt.Errorf("fetch procedures: %w", err)
	}
	if err := loadSchemaTriggers(ctx, sqlConn.conn, &result, `
SELECT USER, trigger_name, table_name, trigger_type, triggering_event
FROM user_triggers
ORDER BY trigger_name
`); err != nil {
		return contracts.SchemaInfo{}, fmt.Errorf("fetch triggers: %w", err)
	}
	if err := loadSchemaSequences(ctx, sqlConn.conn, &result, `
SELECT USER, sequence_name
FROM user_sequences
ORDER BY sequence_name
`); err != nil {
		return contracts.SchemaInfo{}, fmt.Errorf("fetch sequences: %w", err)
	}
	if err := loadSchemaPackages(ctx, sqlConn.conn, &result, `
SELECT USER, pkg.object_name,
       CASE WHEN body.object_name IS NULL THEN 0 ELSE 1 END
FROM user_objects pkg
LEFT JOIN user_objects body
  ON body.object_name = pkg.object_name
 AND body.object_type = 'PACKAGE BODY'
WHERE pkg.object_type = 'PACKAGE'
ORDER BY pkg.object_name
`); err != nil {
		return contracts.SchemaInfo{}, fmt.Errorf("fetch packages: %w", err)
	}
	if err := loadSchemaTypes(ctx, sqlConn.conn, &result, `
SELECT USER, type_name, typecode
FROM user_types
ORDER BY type_name
`); err != nil {
		return contracts.SchemaInfo{}, fmt.Errorf("fetch types: %w", err)
	}
	return result, nil
}

func loadOracleTable(ctx context.Context, conn *sql.Conn, table objectRef) (contracts.SchemaTable, error) {
	rows, err := conn.QueryContext(ctx, `
SELECT c.column_name,
       c.data_type,
       CASE WHEN c.nullable = 'Y' THEN 1 ELSE 0 END,
       CASE WHEN pk.column_name IS NULL THEN 0 ELSE 1 END
FROM user_tab_columns c
LEFT JOIN (
  SELECT cols.table_name, cols.column_name
  FROM user_constraints cons
  JOIN user_cons_columns cols
    ON cons.constraint_name = cols.constraint_name
  WHERE cons.constraint_type = 'P'
) pk
  ON pk.table_name = c.table_name
 AND pk.column_name = c.column_name
WHERE c.table_name = :1
ORDER BY c.column_id
`, strings.ToUpper(table.Name))
	if err != nil {
		return contracts.SchemaTable{}, fmt.Errorf("fetch columns for %s.%s: %w", table.Schema, table.Name, err)
	}
	defer rows.Close()

	item := contracts.SchemaTable{Name: table.Name, Schema: table.Schema, Columns: make([]contracts.SchemaColumn, 0)}
	for rows.Next() {
		var column contracts.SchemaColumn
		if err := rows.Scan(&column.Name, &column.DataType, &column.Nullable, &column.IsPrimaryKey); err != nil {
			return contracts.SchemaTable{}, fmt.Errorf("scan column row for %s.%s: %w", table.Schema, table.Name, err)
		}
		item.Columns = append(item.Columns, column)
	}
	if err := rows.Err(); err != nil {
		return contracts.SchemaTable{}, fmt.Errorf("iterate columns for %s.%s: %w", table.Schema, table.Name, err)
	}
	return item, nil
}
