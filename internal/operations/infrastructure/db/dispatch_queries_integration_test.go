package db_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	coredb "github.com/nopalli-io/nopalli/internal/core/infrastructure/db"
	"github.com/nopalli-io/nopalli/internal/operations/application"
	"github.com/nopalli-io/nopalli/internal/operations/domain/intermodal_old"
	opsdb "github.com/nopalli-io/nopalli/internal/operations/infrastructure/db"
)

// Integration tests for the read-side projections. The DB persists across tests
// in a run, so assertions find specific seeded ids rather than asserting total
// counts.

func containsSummary(list []application.DispatchSummary, id uuid.UUID) bool {
	for _, s := range list {
		if s.ID == id {
			return true
		}
	}
	return false
}

func TestIntegration_DispatchQueries(t *testing.T) {
	ctx := context.Background()
	dsn := dsnOrSkip(t)
	if err := coredb.Migrate(ctx, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := coredb.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	uow := opsdb.NewUnitOfWork(pool, nil)
	q := opsdb.NewDispatchQueryService(pool)

	// Seed one queued dispatch and one advanced to in-progress.
	queued := liveLoadDispatch(t)
	if err := uow.Do(ctx, func(r application.Repository) error { return r.Add(ctx, queued) }); err != nil {
		t.Fatalf("add queued: %v", err)
	}
	inProgress := liveLoadDispatch(t)
	if err := uow.Do(ctx, func(r application.Repository) error { return r.Add(ctx, inProgress) }); err != nil {
		t.Fatalf("add in-progress: %v", err)
	}
	dispatcher := intermodal_old.NewDispatcherID()
	driver := intermodal_old.NewDriverID()
	truck := intermodal_old.NewTruckID()
	if err := uow.Do(ctx, func(r application.Repository) error {
		d, err := r.Get(ctx, inProgress.ID())
		if err != nil {
			return err
		}
		d.Delegate(dispatcher)
		if _, err := d.Assign(&driver, &truck); err != nil {
			return err
		}
		if err := d.Approve(); err != nil {
			return err
		}
		if err := d.Start(); err != nil {
			return err
		}
		return r.Save(ctx, d)
	}); err != nil {
		t.Fatalf("advance in-progress: %v", err)
	}

	queuedID := queued.ID().UUID()
	inProgressID := inProgress.ID().UUID()

	// --- GetDispatch --------------------------------------------------------

	detail, err := q.GetDispatch(ctx, queuedID, application.VisibleAll())
	if err != nil {
		t.Fatalf("get queued: %v", err)
	}
	if detail == nil {
		t.Fatal("expected the queued dispatch")
	}
	if detail.Status != intermodal_old.StatusQueued || detail.PlanType != intermodal_old.PlanLiveLoad {
		t.Fatalf("detail = %+v, want queued/live_load", detail)
	}
	if detail.DispatcherID != nil {
		t.Fatal("un-delegated dispatch must have nil dispatcher")
	}

	if d, err := q.GetDispatch(ctx, inProgressID, application.VisibleAll()); err != nil || d == nil {
		t.Fatalf("get in-progress: d=%v err=%v", d, err)
	} else if d.DispatcherID == nil || *d.DispatcherID != dispatcher.UUID() {
		t.Fatal("delegated dispatch must expose its dispatcher")
	}

	// Out of scope reads as not-found.
	if d, err := q.GetDispatch(ctx, queuedID, application.VisibleNothing()); err != nil || d != nil {
		t.Fatalf("visible-nothing get = %v,%v, want nil,nil", d, err)
	}
	// Genuinely missing id.
	if d, err := q.GetDispatch(ctx, intermodal_old.NewDispatchID().UUID(), application.VisibleAll()); err != nil || d != nil {
		t.Fatalf("missing get = %v,%v, want nil,nil", d, err)
	}

	// --- ListDispatches -----------------------------------------------------

	all, err := q.ListDispatches(ctx, application.VisibleAll(), application.ListFilter{})
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if !containsSummary(all, queuedID) || !containsSummary(all, inProgressID) {
		t.Fatal("list-all must contain both seeded dispatches")
	}

	none, err := q.ListDispatches(ctx, application.VisibleNothing(), application.ListFilter{})
	if err != nil {
		t.Fatalf("list nothing: %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("visible-nothing list = %d rows, want 0", len(none))
	}

	// Status filter: queued contains the queued one, not the in-progress one,
	// and every returned row is queued.
	queuedStatus := intermodal_old.StatusQueued
	ql, err := q.ListDispatches(ctx, application.VisibleAll(), application.ListFilter{Status: &queuedStatus})
	if err != nil {
		t.Fatalf("list queued: %v", err)
	}
	if !containsSummary(ql, queuedID) || containsSummary(ql, inProgressID) {
		t.Fatal("queued filter must include the queued dispatch and exclude the in-progress one")
	}
	for _, s := range ql {
		if s.Status != intermodal_old.StatusQueued {
			t.Fatalf("status filter leaked a %v row", s.Status)
		}
	}
}

// TestIntegration_DispatchDetailChildren checks that GetDispatch projects the
// owned child collections, each carrying its audit timestamps.
func TestIntegration_DispatchDetailChildren(t *testing.T) {
	ctx := context.Background()
	dsn := dsnOrSkip(t)
	if err := coredb.Migrate(ctx, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := coredb.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	uow := coredb.NewUnitOfWork(pool)
	q := opsdb.NewDispatchQueryService(pool)

	d := liveLoadDispatch(t)
	id := d.ID().UUID()
	if err := uow.Do(ctx, func(tx coredb.DBTX) error {
		return opsdb.NewDispatchRepository(tx, nil).Add(ctx, d)
	}); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := uow.Do(ctx, func(tx coredb.DBTX) error {
		repo := opsdb.NewDispatchRepository(tx, nil)
		loaded, err := repo.Get(ctx, d.ID())
		if err != nil {
			return err
		}
		loaded.Delegate(intermodal_old.NewDispatcherID())
		driver, truck := intermodal_old.NewDriverID(), intermodal_old.NewTruckID()
		if _, err := loaded.Assign(&driver, &truck); err != nil {
			return err
		}
		if err := loaded.Approve(); err != nil {
			return err
		}
		if err := loaded.Start(); err != nil {
			return err
		}
		chassis, _ := intermodal_old.NewChassisNumber("CHAS-1")
		if _, err := loaded.AcquireChassis(intermodal_old.NewFacilityID(), chassis); err != nil {
			return err
		}
		return repo.Save(ctx, loaded)
	}); err != nil {
		t.Fatalf("advance+save: %v", err)
	}

	detail, err := q.GetDispatch(ctx, id, application.VisibleAll())
	if err != nil || detail == nil {
		t.Fatalf("get: detail=%v err=%v", detail, err)
	}

	if len(detail.Tasks) == 0 {
		t.Fatal("expected tasks on the detail")
	}
	for _, i := range detail.Tasks {
		if i.CreatedAt.IsZero() || i.UpdatedAt.IsZero() {
			t.Fatalf("task %v missing timestamps: %+v", i.ID, i)
		}
	}
	if len(detail.Assignments) != 1 {
		t.Fatalf("assignments = %d, want 1", len(detail.Assignments))
	}
	if detail.Assignments[0].CreatedAt.IsZero() {
		t.Fatal("assignment missing created_at")
	}
	if len(detail.ChassisEpisodes) == 0 {
		t.Fatal("expected a chassis custody episode after acquire")
	}
	if ch := detail.ChassisEpisodes[0]; ch.ChassisNumber == nil || *ch.ChassisNumber != "CHAS-1" {
		t.Fatalf("chassis episode number = %v, want CHAS-1", ch.ChassisNumber)
	}
}
