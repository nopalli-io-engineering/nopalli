package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
	"github.com/pressly/goose/v3"

	"github.com/nopalli-io/nopalli/internal/migrations"
)

// migrateLockKey is an arbitrary, stable key for the advisory lock that
// serializes migrators (the ASCII bytes of "nopalli").
const migrateLockKey int64 = 0x6E6F70616C6C69

// Migrate applies all pending migrations (goose "up"). It is the convenience
// wrapper used by the app boot and tests; the standalone migrate command
// (cmd/migrate) reaches the other goose commands through RunMigrations.
func Migrate(ctx context.Context, dsn string) error {
	return RunMigrations(ctx, dsn, "up")
}

// RunMigrations executes a goose migration command against the database at dsn
// over the embedded migration FS. Useful commands: "up", "up-by-one",
// "up-to <version>", "down", "down-to <version>", "redo", "reset", "status",
// "version". It opens its own database/sql connection because goose is built on
// database/sql; the app itself uses the pgx pool.
//
// Concurrent migrators (parallel test packages, several app instances or a
// migrate job booting against a fresh database) are serialized with a session
// advisory lock: the first holder runs the command while the others wait. For
// "up" this means the losers find the migrations already applied; without it,
// two callers can both try to apply the same new migration and the loser fails
// on the now-duplicate DDL. (Read-only commands like "status" hold the lock too,
// which is harmless.)
func RunMigrations(ctx context.Context, dsn, command string, args ...string) error {
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("db: open for migrate: %w", err)
	}
	defer sqlDB.Close()

	// The lock must be taken and released on one pinned session, so acquire a
	// dedicated connection rather than using the pool.
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return fmt.Errorf("db: migrate conn: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", migrateLockKey); err != nil {
		return fmt.Errorf("db: acquire migrate lock: %w", err)
	}
	defer func() {
		// Release on a fresh context so unlock still runs if ctx was cancelled.
		_, _ = conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", migrateLockKey)
	}()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("db: goose dialect: %w", err)
	}
	if err := goose.RunContext(ctx, command, sqlDB, ".", args...); err != nil {
		return fmt.Errorf("db: migrate %q: %w", command, err)
	}
	return nil
}
