package postgres

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewDB(databaseURL string) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbpool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}

	if err := dbpool.Ping(ctx); err != nil {
		dbpool.Close()
		return nil, err
	}

	return dbpool, nil
}

func RunMigrations(ctx context.Context, db *pgxpool.Pool) error {
	if _, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return err
	}

	migrationsDir, err := resolveMigrationsDir()
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		var applied bool
		if err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, name).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}

		contents, err := os.ReadFile(filepath.Join(migrationsDir, name))
		if err != nil {
			return err
		}

		tx, err := db.Begin(ctx)
		if err != nil {
			return err
		}

		if _, err = tx.Exec(ctx, string(contents)); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}

		if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES ($1)`, name); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}

		if err = tx.Commit(ctx); err != nil {
			return err
		}
	}

	return nil
}

func resolveMigrationsDir() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", os.ErrNotExist
	}

	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "migrations")), nil
}
