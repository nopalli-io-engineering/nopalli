package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	coredb "github.com/nopalli-io/nopalli/internal/core/infrastructure/db"
	"github.com/nopalli-io/nopalli/internal/core/infrastructure/outbox"
	"github.com/nopalli-io/nopalli/internal/operations/application"
)

// UnitOfWork adapts the core transaction boundary to the operations application
// layer: it runs the caller's function inside one transaction and hands it a
// DispatchRepository bound to that transaction. This is what lets the service
// stay free of pgx — it sees only application.Repository.
type UnitOfWork struct {
	core     *coredb.UnitOfWork
	recorder outbox.Recorder
}

// NewUnitOfWork builds the adapter over a pool. A nil recorder drops events
// (outbox.Noop) until the transactional outbox is wired in.
func NewUnitOfWork(pool *pgxpool.Pool, recorder outbox.Recorder) *UnitOfWork {
	if recorder == nil {
		recorder = outbox.Noop{}
	}
	return &UnitOfWork{core: coredb.NewUnitOfWork(pool), recorder: recorder}
}

// Do opens a transaction and invokes fn with a transaction-bound repository.
func (u *UnitOfWork) Do(ctx context.Context, fn func(application.Repository) error) error {
	return u.core.Do(ctx, func(tx coredb.DBTX) error {
		return fn(NewDispatchRepository(tx, u.recorder))
	})
}

// Compile-time checks that the adapters satisfy the application ports.
var (
	_ application.UnitOfWork = (*UnitOfWork)(nil)
	_ application.Repository = (*DispatchRepository)(nil)
)
