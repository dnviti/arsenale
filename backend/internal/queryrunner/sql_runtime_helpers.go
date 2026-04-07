package queryrunner

import (
	"context"
	"database/sql"
	"strings"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func loadTableRefs(ctx context.Context, conn *sql.Conn, query string, args ...any) ([]objectRef, error) {
	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]objectRef, 0)
	for rows.Next() {
		var ref objectRef
		if err := rows.Scan(&ref.Schema, &ref.Name); err != nil {
			return nil, err
		}
		result = append(result, ref)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func loadSchemaViews(ctx context.Context, conn *sql.Conn, result *contracts.SchemaInfo, query string, args ...any) error {
	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var view contracts.SchemaView
		if err := rows.Scan(&view.Schema, &view.Name, &view.Materialized); err != nil {
			return err
		}
		result.Views = append(result.Views, view)
	}
	return rows.Err()
}

func loadSchemaRoutines(ctx context.Context, conn *sql.Conn, query string, out *[]contracts.SchemaRoutine, args ...any) error {
	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item contracts.SchemaRoutine
		if err := rows.Scan(&item.Schema, &item.Name, &item.ReturnType); err != nil {
			return err
		}
		*out = append(*out, item)
	}
	return rows.Err()
}

func loadSchemaTriggers(ctx context.Context, conn *sql.Conn, result *contracts.SchemaInfo, query string, args ...any) error {
	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item contracts.SchemaTrigger
		if err := rows.Scan(&item.Schema, &item.Name, &item.TableName, &item.Timing, &item.Event); err != nil {
			return err
		}
		result.Triggers = append(result.Triggers, item)
	}
	return rows.Err()
}

func loadSchemaSequences(ctx context.Context, conn *sql.Conn, result *contracts.SchemaInfo, query string, args ...any) error {
	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item contracts.SchemaSequence
		if err := rows.Scan(&item.Schema, &item.Name); err != nil {
			return err
		}
		result.Sequences = append(result.Sequences, item)
	}
	return rows.Err()
}

func loadSchemaPackages(ctx context.Context, conn *sql.Conn, result *contracts.SchemaInfo, query string, args ...any) error {
	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item contracts.SchemaPackage
		if err := rows.Scan(&item.Schema, &item.Name, &item.HasBody); err != nil {
			return err
		}
		result.Packages = append(result.Packages, item)
	}
	return rows.Err()
}

func loadSchemaTypes(ctx context.Context, conn *sql.Conn, result *contracts.SchemaInfo, query string, args ...any) error {
	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item contracts.SchemaNamedType
		if err := rows.Scan(&item.Schema, &item.Name, &item.Kind); err != nil {
			return err
		}
		result.Types = append(result.Types, item)
	}
	return rows.Err()
}

func parseObjectRef(target, defaultSchema string) objectRef {
	parts := splitObjectTarget(target)
	switch len(parts) {
	case 0:
		return objectRef{Schema: defaultSchema}
	case 1:
		return objectRef{Schema: defaultSchema, Name: parts[0]}
	case 2:
		return objectRef{Schema: defaultSchema, Name: parts[0], Column: parts[1]}
	default:
		return objectRef{
			Schema: parts[len(parts)-3],
			Name:   parts[len(parts)-2],
			Column: parts[len(parts)-1],
		}
	}
}

func splitObjectTarget(target string) []string {
	rawParts := strings.Split(strings.TrimSpace(target), ".")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		part = normalizeIdentifierToken(part)
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func normalizeIdentifierToken(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	return value
}
