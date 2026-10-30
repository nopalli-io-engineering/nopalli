// Command migrate applies nopalli's schema migrations as a standalone step,
// separate from the api server's boot. Running schema changes is an explicit,
// serialized operation (a deploy job, a one-off), not a side effect of every
// server start — so the api no longer migrates on startup; run this first.
//
// Usage:
//
//	migrate [command] [args]
//
// Commands (default "up"):
//
//	up                 apply all pending migrations
//	up-by-one          apply the next pending migration only
//	up-to <version>    apply migrations up to and including <version>
//	down               roll back the most recent migration
//	down-to <version>  roll back to (but not including) <version>
//	redo               roll back and re-apply the most recent migration
//	status             print each migration and whether it is applied
//	version            print the current database schema version
//
// Configuration is via environment:
//
//	DB_URL   Postgres DSN (required)
package main

import (
	"context"
	"errors"
	"log"
	"os"

	coredb "github.com/nopalli-io/nopalli/internal/core/infrastructure/db"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(argv []string) error {
	dsn := os.Getenv("DB_URL")
	if dsn == "" {
		return errors.New("DB_URL is required")
	}

	command := "up"
	var args []string
	if len(argv) > 0 {
		command = argv[0]
		args = argv[1:]
	}

	ctx := context.Background()
	if err := coredb.RunMigrations(ctx, dsn, command, args...); err != nil {
		return err
	}
	log.Printf("migrate %q: ok", command)
	return nil
}
