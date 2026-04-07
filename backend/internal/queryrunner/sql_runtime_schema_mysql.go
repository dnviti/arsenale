package queryrunner

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func fetchMySQLSchema(ctx context.Context, target *contracts.DatabaseTarget) (contracts.SchemaInfo, error) {
	sqlConn, err := openSQLTargetConn(ctx, target)
	if err != nil {
		return contracts.SchemaInfo{}, err
	}
	defer sqlConn.Close()

	result := emptySchemaInfo()
	tables, err := loadTableRefs(ctx, sqlConn.conn, `
SELECT table_schema, table_name
FROM information_schema.tables
WHERE table_schema = DATABASE() AND table_type = 'BASE TABLE'
ORDER BY table_schema, table_name
`)
	if err != nil {
		return contracts.SchemaInfo{}, fmt.Errorf("fetch tables: %w", err)
	}
	for _, table := range tables {
		item, err := loadMySQLTable(ctx, sqlConn.conn, table)
		if err != nil {
			return contracts.SchemaInfo{}, err
		}
		result.Tables = append(result.Tables, item)
	}
	if err := loadSchemaViews(ctx, sqlConn.conn, &result, `
SELECT table_schema, table_name, false
FROM information_schema.views
WHERE table_schema = DATABASE()
ORDER BY table_schema, table_name
`); err != nil {
		return contracts.SchemaInfo{}, fmt.Errorf("fetch views: %w", err)
	}
	if err := loadSchemaRoutines(ctx, sqlConn.conn, `
SELECT routine_schema, routine_name, COALESCE(data_type, '')
FROM information_schema.routines
WHERE routine_schema = DATABASE() AND routine_type = 'FUNCTION'
ORDER BY routine_schema, routine_name
`, &result.Functions); err != nil {
		return contracts.SchemaInfo{}, fmt.Errorf("fetch functions: %w", err)
	}
	if err := loadSchemaRoutines(ctx, sqlConn.conn, `
SELECT routine_schema, routine_name, ''
FROM information_schema.routines
WHERE routine_schema = DATABASE() AND routine_type = 'PROCEDURE'
ORDER BY routine_schema, routine_name
`, &result.Procedures); err != nil {
		return contracts.SchemaInfo{}, fmt.Errorf("fetch procedures: %w", err)
	}
	if err := loadSchemaTriggers(ctx, sqlConn.conn, &result, `
SELECT trigger_schema, trigger_name, event_object_table, action_timing, event_manipulation
FROM information_schema.triggers
WHERE trigger_schema = DATABASE()
ORDER BY trigger_schema, trigger_name
`); err != nil {
		return contracts.SchemaInfo{}, fmt.Errorf("fetch triggers: %w", err)
	}
	return result, nil
}

func loadMySQLTable(ctx context.Context, conn *sql.Conn, table objectRef) (contracts.SchemaTable, error) {
	rows, err := conn.QueryContext(ctx, `
SELECT column_name, column_type, is_nullable = 'YES', column_key = 'PRI'
FROM information_schema.columns
WHERE table_schema = ? AND table_name = ?
ORDER BY ordinal_position
`, table.Schema, table.Name)
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
