package queryrunner

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func explainSQLQuery(ctx context.Context, target *contracts.DatabaseTarget, req contracts.QueryPlanRequest) (contracts.QueryPlanResponse, error) {
	sqlConn, err := openSQLTargetConn(ctx, target)
	if err != nil {
		return contracts.QueryPlanResponse{}, err
	}
	defer sqlConn.Close()

	switch targetProtocol(target) {
	case protocolMySQL:
		rows, err := sqlConn.conn.QueryContext(ctx, `EXPLAIN FORMAT=JSON `+req.SQL)
		if err != nil {
			return contracts.QueryPlanResponse{}, fmt.Errorf("run explain: %w", err)
		}
		defer rows.Close()

		if !rows.Next() {
			if err := rows.Err(); err != nil {
				return contracts.QueryPlanResponse{}, fmt.Errorf("read explain row: %w", err)
			}
			return contracts.QueryPlanResponse{Supported: false}, nil
		}

		var payload any
		if err := rows.Scan(&payload); err != nil {
			return contracts.QueryPlanResponse{}, fmt.Errorf("scan explain values: %w", err)
		}
		plan, raw, err := normalizePlanValue(payload)
		if err != nil {
			return contracts.QueryPlanResponse{}, err
		}
		return contracts.QueryPlanResponse{Supported: true, Plan: plan, Format: "json", Raw: raw}, nil
	default:
		return contracts.QueryPlanResponse{Supported: false}, nil
	}
}

func introspectSQLQuery(ctx context.Context, target *contracts.DatabaseTarget, req contracts.QueryIntrospectionRequest) (contracts.QueryIntrospectionResponse, error) {
	sqlConn, err := openSQLTargetConn(ctx, target)
	if err != nil {
		return contracts.QueryIntrospectionResponse{}, err
	}
	defer sqlConn.Close()

	switch targetProtocol(target) {
	case protocolMySQL:
		return introspectMySQL(ctx, sqlConn.conn, req)
	case protocolMSSQL:
		return introspectMSSQL(ctx, sqlConn.conn, req)
	case protocolOracle:
		return introspectOracle(ctx, sqlConn.conn, req)
	default:
		return contracts.QueryIntrospectionResponse{Supported: false}, nil
	}
}

func introspectMySQL(ctx context.Context, conn *sql.Conn, req contracts.QueryIntrospectionRequest) (contracts.QueryIntrospectionResponse, error) {
	ref := parseObjectRef(req.Target, "")
	switch req.Type {
	case "indexes":
		return queryIntrospectionSQL(ctx, conn, `
SELECT index_name, column_name, non_unique = 0 AS is_unique, seq_in_index, index_type, cardinality
FROM information_schema.statistics
WHERE table_schema = DATABASE() AND table_name = ?
ORDER BY index_name, seq_in_index
`, ref.Name)
	case "statistics":
		if ref.Column != "" {
			return queryIntrospectionSQL(ctx, conn, `
SELECT index_name, column_name, cardinality, sub_part, nullable, index_type
FROM information_schema.statistics
WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?
ORDER BY index_name, seq_in_index
`, ref.Name, ref.Column)
		}
		return queryIntrospectionSQL(ctx, conn, `
SELECT table_name, column_name, data_type, column_type, is_nullable
FROM information_schema.columns
WHERE table_schema = DATABASE() AND table_name = ?
ORDER BY ordinal_position
`, ref.Name)
	case "foreign_keys":
		return queryIntrospectionSQL(ctx, conn, `
SELECT constraint_name, column_name, referenced_table_name AS referenced_table, referenced_column_name AS referenced_column
FROM information_schema.key_column_usage
WHERE table_schema = DATABASE() AND table_name = ? AND referenced_table_name IS NOT NULL
ORDER BY constraint_name, ordinal_position
`, ref.Name)
	case "table_schema":
		return queryIntrospectionSQL(ctx, conn, `
SELECT column_name, data_type, column_type, is_nullable, column_default
FROM information_schema.columns
WHERE table_schema = DATABASE() AND table_name = ?
ORDER BY ordinal_position
`, ref.Name)
	case "row_count":
		return queryIntrospectionSQL(ctx, conn, `
SELECT table_rows AS approximate_count
FROM information_schema.tables
WHERE table_schema = DATABASE() AND table_name = ?
`, ref.Name)
	case "database_version":
		return queryIntrospectionSQL(ctx, conn, `SELECT VERSION() AS version`)
	default:
		return contracts.QueryIntrospectionResponse{Supported: false}, nil
	}
}

func introspectMSSQL(ctx context.Context, conn *sql.Conn, req contracts.QueryIntrospectionRequest) (contracts.QueryIntrospectionResponse, error) {
	ref := parseObjectRef(req.Target, "dbo")
	switch req.Type {
	case "indexes":
		return queryIntrospectionSQL(ctx, conn, `
SELECT i.name AS index_name,
       STRING_AGG(c.name, ', ') WITHIN GROUP (ORDER BY ic.key_ordinal) AS columns,
       i.is_primary_key,
       i.is_unique,
       i.type_desc
FROM sys.indexes i
JOIN sys.tables t ON i.object_id = t.object_id
JOIN sys.schemas s ON t.schema_id = s.schema_id
LEFT JOIN sys.index_columns ic
  ON i.object_id = ic.object_id AND i.index_id = ic.index_id AND ic.is_included_column = 0
LEFT JOIN sys.columns c
  ON ic.object_id = c.object_id AND ic.column_id = c.column_id
WHERE s.name = @p1 AND t.name = @p2 AND i.name IS NOT NULL
GROUP BY i.name, i.is_primary_key, i.is_unique, i.type_desc
ORDER BY i.name
`, ref.Schema, ref.Name)
	case "statistics":
		return queryIntrospectionSQL(ctx, conn, `
SELECT st.name AS statistic_name,
       c.name AS column_name,
       sp.last_updated,
       sp.rows,
       sp.rows_sampled,
       sp.modification_counter
FROM sys.stats st
JOIN sys.tables t ON st.object_id = t.object_id
JOIN sys.schemas s ON t.schema_id = s.schema_id
JOIN sys.stats_columns sc ON st.object_id = sc.object_id AND st.stats_id = sc.stats_id
JOIN sys.columns c ON sc.object_id = c.object_id AND sc.column_id = c.column_id
OUTER APPLY sys.dm_db_stats_properties(st.object_id, st.stats_id) sp
WHERE s.name = @p1 AND t.name = @p2
ORDER BY st.name, sc.stats_column_id
`, ref.Schema, ref.Name)
	case "foreign_keys":
		return queryIntrospectionSQL(ctx, conn, `
SELECT fk.name AS constraint_name,
       pc.name AS column_name,
       rt.name AS referenced_table,
       rc.name AS referenced_column
FROM sys.foreign_keys fk
JOIN sys.foreign_key_columns fkc
  ON fk.object_id = fkc.constraint_object_id
JOIN sys.tables pt ON fk.parent_object_id = pt.object_id
JOIN sys.schemas ps ON pt.schema_id = ps.schema_id
JOIN sys.columns pc ON fkc.parent_object_id = pc.object_id AND fkc.parent_column_id = pc.column_id
JOIN sys.tables rt ON fk.referenced_object_id = rt.object_id
JOIN sys.columns rc ON fkc.referenced_object_id = rc.object_id AND fkc.referenced_column_id = rc.column_id
WHERE ps.name = @p1 AND pt.name = @p2
ORDER BY fk.name, fkc.constraint_column_id
`, ref.Schema, ref.Name)
	case "table_schema":
		return queryIntrospectionSQL(ctx, conn, `
SELECT COLUMN_NAME, DATA_TYPE, CHARACTER_MAXIMUM_LENGTH, COLUMN_DEFAULT, IS_NULLABLE
FROM INFORMATION_SCHEMA.COLUMNS
WHERE TABLE_SCHEMA = @p1 AND TABLE_NAME = @p2
ORDER BY ORDINAL_POSITION
`, ref.Schema, ref.Name)
	case "row_count":
		return queryIntrospectionSQL(ctx, conn, `
SELECT SUM(p.rows) AS approximate_count
FROM sys.partitions p
JOIN sys.tables t ON p.object_id = t.object_id
JOIN sys.schemas s ON t.schema_id = s.schema_id
WHERE s.name = @p1 AND t.name = @p2 AND p.index_id IN (0, 1)
`, ref.Schema, ref.Name)
	case "database_version":
		return queryIntrospectionSQL(ctx, conn, `SELECT @@VERSION AS version`)
	default:
		return contracts.QueryIntrospectionResponse{Supported: false}, nil
	}
}

func introspectOracle(ctx context.Context, conn *sql.Conn, req contracts.QueryIntrospectionRequest) (contracts.QueryIntrospectionResponse, error) {
	ref := parseObjectRef(req.Target, "")
	switch req.Type {
	case "indexes":
		return queryIntrospectionSQL(ctx, conn, `
SELECT idx.index_name, idx.column_name, ind.uniqueness, idx.column_position
FROM user_ind_columns idx
JOIN user_indexes ind ON idx.index_name = ind.index_name
WHERE idx.table_name = :1
ORDER BY idx.index_name, idx.column_position
`, strings.ToUpper(ref.Name))
	case "statistics":
		if ref.Column != "" {
			return queryIntrospectionSQL(ctx, conn, `
SELECT table_name, column_name, num_distinct, num_nulls, density, num_buckets, histogram
FROM user_tab_col_statistics
WHERE table_name = :1 AND column_name = :2
`, strings.ToUpper(ref.Name), strings.ToUpper(ref.Column))
		}
		return queryIntrospectionSQL(ctx, conn, `
SELECT table_name, column_name, num_distinct, num_nulls, density, num_buckets, histogram
FROM user_tab_col_statistics
WHERE table_name = :1
ORDER BY column_name
`, strings.ToUpper(ref.Name))
	case "foreign_keys":
		return queryIntrospectionSQL(ctx, conn, `
SELECT c.constraint_name,
       cc.column_name,
       r.table_name AS referenced_table,
       rc.column_name AS referenced_column
FROM user_constraints c
JOIN user_cons_columns cc
  ON c.constraint_name = cc.constraint_name
JOIN user_constraints r
  ON c.r_constraint_name = r.constraint_name
JOIN user_cons_columns rc
  ON r.constraint_name = rc.constraint_name
 AND rc.position = cc.position
WHERE c.constraint_type = 'R' AND c.table_name = :1
ORDER BY c.constraint_name, cc.position
`, strings.ToUpper(ref.Name))
	case "table_schema":
		return queryIntrospectionSQL(ctx, conn, `
SELECT column_name, data_type, data_length, data_precision, data_scale, data_default, nullable
FROM user_tab_columns
WHERE table_name = :1
ORDER BY column_id
`, strings.ToUpper(ref.Name))
	case "row_count":
		return queryIntrospectionSQL(ctx, conn, `
SELECT num_rows AS approximate_count
FROM user_tables
WHERE table_name = :1
`, strings.ToUpper(ref.Name))
	case "database_version":
		return queryIntrospectionSQL(ctx, conn, `
SELECT banner AS version
FROM v$version
WHERE ROWNUM = 1
`)
	default:
		return contracts.QueryIntrospectionResponse{Supported: false}, nil
	}
}

func queryIntrospectionSQL(ctx context.Context, conn *sql.Conn, query string, args ...any) (contracts.QueryIntrospectionResponse, error) {
	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return contracts.QueryIntrospectionResponse{}, err
	}
	defer rows.Close()

	records, err := rowsToMapsSQL(rows)
	if err != nil {
		return contracts.QueryIntrospectionResponse{}, err
	}
	if len(records) == 1 {
		return contracts.QueryIntrospectionResponse{Supported: true, Data: records[0]}, nil
	}
	return contracts.QueryIntrospectionResponse{Supported: true, Data: records}, nil
}

func rowsToMapsSQL(rows *sql.Rows) ([]map[string]any, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("read columns: %w", err)
	}

	result := make([]map[string]any, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		dest := make([]any, len(columns))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("scan row values: %w", err)
		}
		record := make(map[string]any, len(columns))
		for idx, column := range columns {
			record[column] = normalizeValue(values[idx])
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}
	return result, nil
}

func normalizePlanValue(value any) (any, string, error) {
	normalized := normalizeValue(value)
	switch typed := normalized.(type) {
	case string:
		var decoded any
		if err := json.Unmarshal([]byte(typed), &decoded); err != nil {
			return normalized, typed, nil
		}
		pretty, err := json.MarshalIndent(decoded, "", "  ")
		if err != nil {
			return decoded, typed, nil
		}
		return decoded, string(pretty), nil
	default:
		pretty, err := json.MarshalIndent(normalized, "", "  ")
		if err != nil {
			return normalized, fmt.Sprintf("%v", normalized), nil
		}
		return normalized, string(pretty), nil
	}
}
