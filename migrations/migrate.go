// Package migrations holds the SQL schema and applies it at startup.
//
// Files are named NNNN_description.sql and run in name order, each in its own
// transaction. Applied versions are recorded in schema_migrations, so every
// file runs exactly once per database. Never edit a file that has shipped —
// add a new one instead.
package migrations

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
)

//go:embed *.sql
var files embed.FS

// lockID is an arbitrary constant: a session-level advisory lock on it keeps two
// server instances from migrating the same database at the same time.
const lockID = 7_301_984_221

// Up applies every migration not yet recorded and returns the versions it ran.
func Up(ctx context.Context, db *sqlx.DB) ([]string, error) {
	conn, err := db.Connx(ctx)
	if err != nil {
		return nil, fmt.Errorf("migrations: connect: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, lockID); err != nil {
		return nil, fmt.Errorf("migrations: lock: %w", err)
	}
	defer conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock($1)`, lockID) //nolint:errcheck

	if _, err := conn.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`); err != nil {
		return nil, fmt.Errorf("migrations: create table: %w", err)
	}

	var done []string
	if err := conn.SelectContext(ctx, &done, `SELECT version FROM schema_migrations`); err != nil {
		return nil, fmt.Errorf("migrations: list applied: %w", err)
	}
	applied := make(map[string]bool, len(done))
	for _, v := range done {
		applied[v] = true
	}

	names, err := fs.Glob(files, "*.sql")
	if err != nil {
		return nil, fmt.Errorf("migrations: list files: %w", err)
	}
	sort.Strings(names)

	var ran []string
	for _, name := range names {
		version := strings.TrimSuffix(name, ".sql")
		if applied[version] {
			continue
		}
		body, err := files.ReadFile(name)
		if err != nil {
			return ran, fmt.Errorf("migrations: read %s: %w", name, err)
		}

		tx, err := conn.BeginTxx(ctx, nil)
		if err != nil {
			return ran, fmt.Errorf("migrations: begin %s: %w", version, err)
		}
		if _, err := tx.ExecContext(ctx, string(body)); err != nil {
			_ = tx.Rollback()
			return ran, fmt.Errorf("migrations: apply %s: %w", version, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			_ = tx.Rollback()
			return ran, fmt.Errorf("migrations: record %s: %w", version, err)
		}
		if err := tx.Commit(); err != nil {
			return ran, fmt.Errorf("migrations: commit %s: %w", version, err)
		}
		ran = append(ran, version)
	}
	return ran, nil
}
