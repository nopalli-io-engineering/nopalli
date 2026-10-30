// Command api is the HTTP entrypoint: it wires the pgx pool, the operations
// unit of work + application service, and the REST command handlers, then
// serves them. Configuration is via environment:
//
//	DB_URL   Postgres DSN (required)
//	PORT     listen port (default 8080)
//
// Auth is not wired yet, so every command runs under application.AllowAll.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	coredb "github.com/nopalli-io/nopalli/internal/core/infrastructure/db"
	"github.com/nopalli-io/nopalli/internal/operations/application"
	opsdb "github.com/nopalli-io/nopalli/internal/operations/infrastructure/db"
	"github.com/nopalli-io/nopalli/internal/operations/interfaces/rest"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	dsn := os.Getenv("DB_URL")
	if dsn == "" {
		return errors.New("DB_URL is required")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx := context.Background()
	if err := coredb.Migrate(ctx, dsn); err != nil {
		return err
	}
	pool, err := coredb.Connect(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()

	// Compose: pool -> unit of work -> service (writes) + query service (reads)
	// -> handlers -> routes.
	uow := opsdb.NewUnitOfWork(pool, nil) // nil recorder: events dropped until the outbox lands
	svc := application.NewIntermodalDispatchService(uow, application.AllowAll{})
	queries := opsdb.NewDispatchQueryService(pool)
	mux := http.NewServeMux()
	rest.RegisterRoutes(mux, rest.NewHandlers(svc, queries))

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Serve until SIGINT/SIGTERM, then drain.
	errc := make(chan error, 1)
	go func() {
		log.Printf("api listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-errc:
		return err
	case <-stop:
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
