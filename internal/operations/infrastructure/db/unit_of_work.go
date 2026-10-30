package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	coredb "github.com/nopalli-io/nopalli/internal/core/infrastructure/db"
	"github.com/nopalli-io/nopalli/internal/core/infrastructure/outbox"
	"github.com/nopalli-io/nopalli/internal/operations/application"
)

type UnitOfWork struct {
	core     *coredb.UnitOfWork
	recorder outbox.Recorder
}

func NewUnitOfWork(pool *pgxpool.Pool, recorder outbox.Recorder) *UnitOfWork {
	if recorder == nil {
		recorder = outbox.Noop{}
	}
	return &UnitOfWork{core: coredb.NewUnitOfWork(pool), recorder: recorder}
}

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
