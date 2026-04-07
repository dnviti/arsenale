package queryrunner

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func fetchMSSQLSchema(ctx context.Context, target *contracts.DatabaseTarget) (contracts.SchemaInfo, error) {
	sqlConn, err := openSQLTargetConn(ctx, target)
	if err != nil {
		return contracts.SchemaInfo{}, err
	}
	defer sqlConn.Close()

	result := emptySchemaInfo()
	tables, err := loadTableRefs(ctx, sqlConn.conn, `
SELECT TABLE_SCHEMA, TABLE_NAME
FROM INFORMATION_SCHEMA.TABLES
WHERE TABLE_TYPE = 'BASE TABLE' AND TABLE_SCHEMA NOT IN ('sys', 'INFORMATION_SCHEMA')
ORDER BY TABLE_SCHEMA, TABLE_NAME
`)
	if err != nil {
		return contracts.SchemaInfo{}, fmt.Errorf("fetch tables: %w", err)
	}
	for _, table := range tables {
		item, err := loadMSSQLTable(ctx, sqlConn.conn, table)
		if err != nil {
			return contracts.SchemaInfo{}, err
		}
		result.Tables = append(result.Tables, item)
	}
	if err := loadSchemaViews(ctx, sqlConn.conn, &result, `
SELECT TABLE_SCHEMA, TABLE_NAME, CAST(0 AS bit)
FROM INFORMATION_SCHEMA.VIEWS
ORDER BY TABLE_SCHEMA, TABLE_NAME
`); err != nil {
		return contracts.SchemaInfo{}, fmt.Errorf("fetch views: %w", err)
	}
	if err := loadSchemaRoutines(ctx, sqlConn.conn, `
SELECT ROUTINE_SCHEMA, ROUTINE_NAME, COALESCE(DATA_TYPE, '')
FROM INFORMATION_SCHEMA.ROUTINES
WHERE ROUTINE_TYPE = 'FUNCTION'
ORDER BY ROUTINE_SCHEMA, ROUTINE_NAME
`, &result.Functions); err != nil {
		return contracts.SchemaInfo{}, fmt.Errorf("fetch functions: %w", err)
	}
	if err := loadSchemaRoutines(ctx, sqlConn.conn, `
SELECT ROUTINE_SCHEMA, ROUTINE_NAME, ''
FROM INFORMATION_SCHEMA.ROUTINES
WHERE ROUTINE_TYPE = 'PROCEDURE'
ORDER BY ROUTINE_SCHEMA, ROUTINE_NAME
`, &result.Procedures); err != nil {
		return contracts.SchemaInfo{}, fmt.Errorf("fetch procedures: %w", err)
	}
	if err := loadSchemaTriggers(ctx, sqlConn.conn, &result, `
SELECT OBJECT_SCHEMA_NAME(parent_id), name, OBJECT_NAME(parent_id),
       CASE WHEN is_instead_of_trigger = 1 THEN 'INSTEAD OF' ELSE 'AFTER' END,
       type_desc
FROM sys.triggers
WHERE parent_class_desc = 'OBJECT_OR_COLUMN'
ORDER BY 1, 2
`); err != nil {
		return contracts.SchemaInfo{}, fmt.Errorf("fetch triggers: %w", err)
	}
	if err := loadSchemaSequences(ctx, sqlConn.conn, &result, `
SELECT SCHEMA_NAME(schema_id), name
FROM sys.sequences
ORDER BY 1, 2
`); err != nil {
		return contracts.SchemaInfo{}, fmt.Errorf("fetch sequences: %w", err)
	}
	if err := loadSchemaTypes(ctx, sqlConn.conn, &result, `
SELECT SCHEMA_NAME(schema_id), name,
       CASE WHEN is_table_type = 1 THEN 'TABLE' ELSE 'TYPE' END
FROM sys.types
WHERE is_user_defined = 1
ORDER BY 1, 2
`); err != nil {
		return contracts.SchemaInfo{}, fmt.Errorf("fetch types: %w", err)
	}
	return result, nil
}

func loadMSSQLTable(ctx context.Context, conn *sql.Conn, table objectRef) (contracts.SchemaTable, error) {
	rows, err := conn.QueryContext(ctx, `
SELECT c.COLUMN_NAME,
       c.DATA_TYPE,
       CASE WHEN c.IS_NULLABLE = 'YES' THEN CAST(1 AS bit) ELSE CAST(0 AS bit) END,
       CASE WHEN tc.CONSTRAINT_TYPE = 'PRIMARY KEY' THEN CAST(1 AS bit) ELSE CAST(0 AS bit) END
FROM INFORMATION_SCHEMA.COLUMNS c
LEFT JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE kcu
  ON c.TABLE_SCHEMA = kcu.TABLE_SCHEMA
 AND c.TABLE_NAME = kcu.TABLE_NAME
 AND c.COLUMN_NAME = kcu.COLUMN_NAME
LEFT JOIN INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
  ON kcu.CONSTRAINT_SCHEMA = tc.CONSTRAINT_SCHEMA
 AND kcu.CONSTRAINT_NAME = tc.CONSTRAINT_NAME
 AND tc.CONSTRAINT_TYPE = 'PRIMARY KEY'
WHERE c.TABLE_SCHEMA = @p1 AND c.TABLE_NAME = @p2
ORDER BY c.ORDINAL_POSITION
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
