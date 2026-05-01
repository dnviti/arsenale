package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	schemamigrations "github.com/dnviti/arsenale/backend/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const migrationTable = `public.arsenale_schema_migrations`

type AppliedMigration struct {
	Version   int64
	Name      string
	AppliedAt time.Time
}

type MigrationReport struct {
	Applied       []AppliedMigration
	AppliedNow    int
	Pending       []schemamigrations.Definition
	LegacyStamped bool
}

func OpenMigrationConn(ctx context.Context) (*pgx.Conn, error) {
	databaseURL, err := DatabaseURLFromEnv()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(databaseURL) == "" {
		return nil, nil
	}

	config, err := pgx.ParseConfig(augmentDatabaseURL(databaseURL, os.Getenv("DATABASE_SSL_ROOT_CERT")))
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	config.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	conn, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("open postgres connection: %w", err)
	}
	if err := conn.Ping(ctx); err != nil {
		conn.Close(ctx)
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return conn, nil
}

func RunMigrations(ctx context.Context) (MigrationReport, error) {
	conn, err := OpenMigrationConn(ctx)
	if err != nil {
		return MigrationReport{}, err
	}
	if conn == nil {
		return MigrationReport{}, nil
	}
	defer conn.Close(ctx)

	return migrateConn(ctx, conn)
}

func MigrationStatus(ctx context.Context) (MigrationReport, error) {
	conn, err := OpenMigrationConn(ctx)
	if err != nil {
		return MigrationReport{}, err
	}
	if conn == nil {
		return MigrationReport{}, nil
	}
	defer conn.Close(ctx)

	return statusConn(ctx, conn)
}

func RequireMigrations(ctx context.Context, db *pgxpool.Pool) error {
	if db == nil || strings.EqualFold(strings.TrimSpace(os.Getenv("ARSENALE_SKIP_MIGRATION_CHECK")), "true") {
		return nil
	}

	exists, err := migrationTableExists(ctx, db)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("database is not migrated: missing %s; run the migrate command before starting services", migrationTable)
	}

	var current int64
	if err := db.QueryRow(ctx, `SELECT COALESCE(MAX(version), 0) FROM `+migrationTable).Scan(&current); err != nil {
		return fmt.Errorf("query migration version: %w", err)
	}

	latest, err := schemamigrations.LatestVersion()
	if err != nil {
		return fmt.Errorf("load migration definitions: %w", err)
	}
	if current < latest {
		return fmt.Errorf("database schema is behind: current=%d latest=%d; run the migrate command before starting services", current, latest)
	}

	return nil
}

func migrateConn(ctx context.Context, conn *pgx.Conn) (MigrationReport, error) {
	defs, err := schemamigrations.Definitions()
	if err != nil {
		return MigrationReport{}, fmt.Errorf("load migration definitions: %w", err)
	}

	if err := ensureMigrationTable(ctx, conn); err != nil {
		return MigrationReport{}, err
	}

	applied, err := loadAppliedMigrations(ctx, conn)
	if err != nil {
		return MigrationReport{}, err
	}

	report := MigrationReport{
		Applied: applied,
	}

	if len(applied) == 0 && len(defs) > 0 {
		legacySchema, err := legacySchemaPresent(ctx, conn)
		if err != nil {
			return MigrationReport{}, err
		}
		if legacySchema {
			baseline := defs[0]
			if err := stampMigration(ctx, conn, baseline.Version, baseline.Name); err != nil {
				return MigrationReport{}, fmt.Errorf("stamp baseline migration: %w", err)
			}
			report.LegacyStamped = true
			applied, err = loadAppliedMigrations(ctx, conn)
			if err != nil {
				return MigrationReport{}, err
			}
			report.Applied = applied
		}
	}

	appliedLookup := make(map[int64]AppliedMigration, len(report.Applied))
	for _, item := range report.Applied {
		appliedLookup[item.Version] = item
	}

	for _, def := range defs {
		if _, ok := appliedLookup[def.Version]; ok {
			continue
		}
		if err := applyMigration(ctx, conn, def); err != nil {
			return MigrationReport{}, fmt.Errorf("apply migration %s: %w", def.Name, err)
		}
		report.AppliedNow++
		appliedItem, err := loadAppliedMigration(ctx, conn, def.Version)
		if err != nil {
			return MigrationReport{}, fmt.Errorf("load applied migration %s: %w", def.Name, err)
		}
		report.Applied = append(report.Applied, appliedItem)
		appliedLookup[def.Version] = appliedItem
	}

	report.Pending = pendingMigrations(defs, report.Applied)
	return report, nil
}

func statusConn(ctx context.Context, conn *pgx.Conn) (MigrationReport, error) {
	defs, err := schemamigrations.Definitions()
	if err != nil {
		return MigrationReport{}, fmt.Errorf("load migration definitions: %w", err)
	}

	exists, err := migrationTableExists(ctx, conn)
	if err != nil {
		return MigrationReport{}, err
	}
	if !exists {
		return MigrationReport{Pending: defs}, nil
	}

	applied, err := loadAppliedMigrations(ctx, conn)
	if err != nil {
		return MigrationReport{}, err
	}

	return MigrationReport{
		Applied: applied,
		Pending: pendingMigrations(defs, applied),
	}, nil
}

func ensureMigrationTable(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, `
CREATE TABLE IF NOT EXISTS public.arsenale_schema_migrations (
  version BIGINT PRIMARY KEY,
  name TEXT NOT NULL,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`)
	if err != nil {
		return fmt.Errorf("ensure migration table: %w", err)
	}
	return nil
}

func applyMigration(ctx context.Context, conn *pgx.Conn, def schemamigrations.Definition) error {
	body, err := schemamigrations.Read(def)
	if err != nil {
		return err
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, string(body)); err != nil {
		return fmt.Errorf("execute migration SQL: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO `+migrationTable+` (version, name) VALUES ($1, $2)`, def.Version, def.Name); err != nil {
		return fmt.Errorf("record migration: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration transaction: %w", err)
	}

	return nil
}

func stampMigration(ctx context.Context, conn *pgx.Conn, version int64, name string) error {
	_, err := conn.Exec(ctx, `INSERT INTO `+migrationTable+` (version, name) VALUES ($1, $2)`, version, name)
	if err != nil {
		return fmt.Errorf("insert migration stamp: %w", err)
	}
	return nil
}

func loadAppliedMigrations(ctx context.Context, conn queryRower) ([]AppliedMigration, error) {
	rows, err := conn.Query(ctx, `SELECT version, name, applied_at FROM `+migrationTable+` ORDER BY version ASC`)
	if err != nil {
		return nil, fmt.Errorf("query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make([]AppliedMigration, 0)
	for rows.Next() {
		var item AppliedMigration
		if err := rows.Scan(&item.Version, &item.Name, &item.AppliedAt); err != nil {
			return nil, fmt.Errorf("scan applied migration: %w", err)
		}
		applied = append(applied, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate applied migrations: %w", err)
	}
	return applied, nil
}

func loadAppliedMigration(ctx context.Context, conn queryRower, version int64) (AppliedMigration, error) {
	var item AppliedMigration
	if err := conn.QueryRow(ctx, `SELECT version, name, applied_at FROM `+migrationTable+` WHERE version = $1`, version).Scan(&item.Version, &item.Name, &item.AppliedAt); err != nil {
		return AppliedMigration{}, fmt.Errorf("query applied migration %d: %w", version, err)
	}
	return item, nil
}

func pendingMigrations(defs []schemamigrations.Definition, applied []AppliedMigration) []schemamigrations.Definition {
	if len(defs) == 0 {
		return nil
	}
	appliedLookup := make(map[int64]struct{}, len(applied))
	for _, item := range applied {
		appliedLookup[item.Version] = struct{}{}
	}

	pending := make([]schemamigrations.Definition, 0)
	for _, def := range defs {
		if _, ok := appliedLookup[def.Version]; ok {
			continue
		}
		pending = append(pending, def)
	}
	return pending
}

func migrationTableExists(ctx context.Context, conn queryRower) (bool, error) {
	var exists bool
	if err := conn.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM information_schema.tables
  WHERE table_schema = 'public'
    AND table_name = 'arsenale_schema_migrations'
)`).Scan(&exists); err != nil {
		return false, fmt.Errorf("check migration table: %w", err)
	}
	return exists, nil
}

func legacySchemaPresent(ctx context.Context, conn queryRower) (bool, error) {
	var exists bool
	if err := conn.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM information_schema.tables
  WHERE table_schema = 'public'
    AND table_name = 'User'
)`).Scan(&exists); err != nil {
		return false, fmt.Errorf("check legacy schema presence: %w", err)
	}
	return exists, nil
}

type queryRower interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func IsMissingRelation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42P01"
}
