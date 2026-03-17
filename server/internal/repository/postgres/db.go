package postgres

import (
	"context"

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

	return ensureSchemaCompatibility(ctx, db)
}

func ensureSchemaCompatibility(ctx context.Context, db *pgxpool.Pool) error {
	compatibilityStatements := []string{
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS department TEXT`,
		`UPDATE users SET department = 'CSE' WHERE department IS NULL`,
		`ALTER TABLE users ALTER COLUMN department SET NOT NULL`,
		`ALTER TABLE notifications ADD COLUMN IF NOT EXISTS priority TEXT NOT NULL DEFAULT 'normal'`,
		`ALTER TABLE notifications ADD COLUMN IF NOT EXISTS target_department TEXT`,
		`UPDATE notifications SET target_department = 'CSE' WHERE target_department IS NULL`,
		`ALTER TABLE notifications ALTER COLUMN target_department SET NOT NULL`,
		`ALTER TABLE notifications ADD COLUMN IF NOT EXISTS idempotency_key TEXT UNIQUE`,
		`ALTER TABLE deliveries ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'pending'`,
		`ALTER TABLE deliveries ADD COLUMN IF NOT EXISTS retry_count INT NOT NULL DEFAULT 0`,
		`ALTER TABLE deliveries ADD COLUMN IF NOT EXISTS last_error TEXT`,
		`ALTER TABLE deliveries ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_deliveries_notification_user ON deliveries (notification_id, user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications (created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_idempotency_key ON notifications (idempotency_key)`,
		`CREATE INDEX IF NOT EXISTS idx_deliveries_status_retry ON deliveries (status, retry_count)`,
		`CREATE INDEX IF NOT EXISTS idx_deliveries_user_id ON deliveries (user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_users_department ON users (department)`,
		`DO $$
		BEGIN
		    IF NOT EXISTS (
		        SELECT 1
		        FROM pg_constraint
		        WHERE conname = 'users_department_check'
		    ) THEN
		        ALTER TABLE users
		            ADD CONSTRAINT users_department_check
		            CHECK (department IN ('CSE', 'ECE', 'ME', 'CIVIL', 'EEE'));
		    END IF;
		END $$`,
		`DO $$
		BEGIN
		    IF NOT EXISTS (
		        SELECT 1
		        FROM pg_constraint
		        WHERE conname = 'notifications_target_department_check'
		    ) THEN
		        ALTER TABLE notifications
		            ADD CONSTRAINT notifications_target_department_check
		            CHECK (target_department IN ('CSE', 'ECE', 'ME', 'CIVIL', 'EEE'));
		    END IF;
		END $$`,
	}

	for _, statement := range compatibilityStatements {
		if _, err := db.Exec(ctx, statement); err != nil {
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
