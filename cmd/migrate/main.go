package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

const advisoryLockID int64 = 845902134

func main() {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("POSTGRES_URL")
	}
	if dsn == "" {
		log.Fatal("DATABASE_URL or POSTGRES_URL is required")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatal("failed to connect to database:", err)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, "SELECT pg_advisory_lock($1)", advisoryLockID); err != nil {
		log.Fatal("failed to acquire advisory lock:", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "SELECT pg_advisory_unlock($1)", advisoryLockID)
	}()

	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version text PRIMARY KEY,
			applied_at timestamptz NOT NULL DEFAULT now()
		)
	`); err != nil {
		log.Fatal("failed to ensure schema_migrations:", err)
	}

	migrationsDir := os.Getenv("MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = "migrations"
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("migrations directory not found: %s", migrationsDir)
			return
		}
		log.Fatal("failed to read migrations directory:", err)
	}

	files := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		files = append(files, entry.Name())
	}

	sort.Strings(files)
	if len(files) == 0 {
		log.Println("no migrations to apply")
		return
	}

	for _, name := range files {
		applied, err := migrationApplied(ctx, pool, name)
		if err != nil {
			log.Fatal("failed to check migrations:", err)
		}
		if applied {
			continue
		}

		path := filepath.Join(migrationsDir, name)
		contents, err := os.ReadFile(path)
		if err != nil {
			log.Fatal("failed to read migration:", err)
		}

		if err := applyMigration(ctx, pool, name, contents); err != nil {
			log.Fatalf("failed to apply %s: %v", name, err)
		}

		log.Printf("applied migration %s", name)
	}
}

func migrationApplied(ctx context.Context, pool *pgxpool.Pool, version string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version).Scan(&exists)
	return exists, err
}

func applyMigration(ctx context.Context, pool *pgxpool.Pool, version string, contents []byte) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	if _, err = tx.Exec(ctx, string(contents)); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	if _, err = tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	return tx.Commit(ctx)
}
