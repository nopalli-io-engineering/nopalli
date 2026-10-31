package db_test

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	coredb "github.com/nopalli-io/nopalli/internal/core/infrastructure/db"
	"github.com/nopalli-io/nopalli/internal/operations/domain/intermodal_old"
	opsdb "github.com/nopalli-io/nopalli/internal/operations/infrastructure/db"
)

// Integration tests for the Dispatch Data Mapper against a real Postgres. They
// run under `make test-integration` (not `-short`); `go test -short` skips them.
// DB connection comes from DB_URL (set by docker-compose for the app service).

func dsnOrSkip(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	dsn := os.Getenv("DB_URL")
	if dsn == "" {
		dsn = os.Getenv("TEST_DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("no DB_URL / TEST_DATABASE_URL set; skipping integration test")
	}
	return dsn
}

func setup(t *testing.T) (*coredb.UnitOfWork, func(intermodal_old.DispatchID) *opsdb.DispatchRepository) {
	t.Helper()
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
	// read repo bound to the pool (reads need no transaction)
	readRepo := func(intermodal_old.DispatchID) *opsdb.DispatchRepository {
		return opsdb.NewDispatchRepository(pool, nil)
	}
	return uow, readRepo
}

func liveLoadDispatch(t *testing.T) *intermodal_old.Dispatch {
	t.Helper()
	container, err := intermodal_old.NewContainerNumber("MSCU1234567")
	if err != nil {
		t.Fatal(err)
	}
	sq := intermodal_old.NewSourceQuoteID()
	return intermodal_old.Queue(&sq, intermodal_old.LiveLoad{
		ContainerNumber:      container,
		ContainerPickupRamp:  intermodal_old.NewFacilityID(),
		ContainerDropoffRamp: intermodal_old.NewFacilityID(),
		DockDoor:             intermodal_old.NewFacilityID(),
	})
}

func TestIntegration_AddAndGet_RoundTrip(t *testing.T) {
	ctx := context.Background()
	uow, readRepo := setup(t)

	d := liveLoadDispatch(t)
	want := d.Snapshot()

	if err := uow.Do(ctx, func(tx coredb.DBTX) error {
		return opsdb.NewDispatchRepository(tx, nil).Add(ctx, d)
	}); err != nil {
		t.Fatalf("add: %v", err)
	}

	loaded, err := readRepo(d.ID()).Get(ctx, d.ID())
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got := loaded.Snapshot(); !reflect.DeepEqual(want, got) {
		t.Fatalf("round-trip mismatch:\n want: %+v\n  got: %+v", want, got)
	}
}

func TestIntegration_SaveAdvances(t *testing.T) {
	ctx := context.Background()
	uow, readRepo := setup(t)

	d := liveLoadDispatch(t)
	id := d.ID()
	if err := uow.Do(ctx, func(tx coredb.DBTX) error {
		return opsdb.NewDispatchRepository(tx, nil).Add(ctx, d)
	}); err != nil {
		t.Fatalf("add: %v", err)
	}

	// Load, advance into execution, and save within one transaction.
	if err := uow.Do(ctx, func(tx coredb.DBTX) error {
		repo := opsdb.NewDispatchRepository(tx, nil)
		loaded, err := repo.Get(ctx, id)
		if err != nil {
			return err
		}
		dispatcher := intermodal_old.NewDispatcherID()
		driver := intermodal_old.NewDriverID()
		truck := intermodal_old.NewTruckID()
		loaded.Delegate(dispatcher)
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

	loaded, err := readRepo(id).Get(ctx, id)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if loaded.Status() != intermodal_old.StatusInProgress {
		t.Fatalf("status = %v, want in_progress", loaded.Status())
	}
	if loaded.Version() != 1 {
		t.Fatalf("version = %d, want 1", loaded.Version())
	}
	if _, ok := loaded.LiveEquipment(); !ok {
		t.Fatal("expected a live chassis after acquire")
	}
	if _, ok := loaded.LiveAssignee(); !ok {
		t.Fatal("expected a live assignee after start")
	}
}

func TestIntegration_OptimisticConcurrency(t *testing.T) {
	ctx := context.Background()
	uow, readRepo := setup(t)

	d := liveLoadDispatch(t)
	id := d.ID()
	if err := uow.Do(ctx, func(tx coredb.DBTX) error {
		return opsdb.NewDispatchRepository(tx, nil).Add(ctx, d)
	}); err != nil {
		t.Fatalf("add: %v", err)
	}

	// Two independent loads at the same version.
	first, err := readRepo(id).Get(ctx, id)
	if err != nil {
		t.Fatalf("load first: %v", err)
	}
	second, err := readRepo(id).Get(ctx, id)
	if err != nil {
		t.Fatalf("load second: %v", err)
	}

	first.Delegate(intermodal_old.NewDispatcherID())
	if err := uow.Do(ctx, func(tx coredb.DBTX) error {
		return opsdb.NewDispatchRepository(tx, nil).Save(ctx, first)
	}); err != nil {
		t.Fatalf("save first: %v", err)
	}

	// The stale second save must be rejected.
	second.Delegate(intermodal_old.NewDispatcherID())
	err = uow.Do(ctx, func(tx coredb.DBTX) error {
		return opsdb.NewDispatchRepository(tx, nil).Save(ctx, second)
	})
	if !errors.Is(err, coredb.ErrConcurrencyConflict) {
		t.Fatalf("stale save err = %v, want ErrConcurrencyConflict", err)
	}
}

// TestIntegration_ChildTimestampsSurviveSave is the regression the move from
// delete+insert to UPSERT-by-id exists for: a child row's created_at must
// survive a Save (it is not re-created), while its updated_at advances. A
// delete+insert strategy would reset created_at on every save.
func TestIntegration_ChildTimestampsSurviveSave(t *testing.T) {
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

	d := liveLoadDispatch(t) // a queued live-load already has tasks
	id := d.ID()
	if err := uow.Do(ctx, func(tx coredb.DBTX) error {
		return opsdb.NewDispatchRepository(tx, nil).Add(ctx, d)
	}); err != nil {
		t.Fatalf("add: %v", err)
	}

	// Read one task row's timestamps straight after insert.
	taskTimes := func() (createdAt, updatedAt time.Time) {
		t.Helper()
		err := pool.QueryRow(ctx, `
SELECT created_at, updated_at
FROM operations.intermodal_dispatch_tasks
WHERE dispatch_id = $1 ORDER BY task_no LIMIT 1`, id.UUID()).Scan(&createdAt, &updatedAt)
		if err != nil {
			t.Fatalf("read task timestamps: %v", err)
		}
		return createdAt, updatedAt
	}
	created1, updated1 := taskTimes()

	// Advance and save in a later transaction, so now() differs.
	if err := uow.Do(ctx, func(tx coredb.DBTX) error {
		repo := opsdb.NewDispatchRepository(tx, nil)
		loaded, err := repo.Get(ctx, id)
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
		return repo.Save(ctx, loaded)
	}); err != nil {
		t.Fatalf("advance+save: %v", err)
	}

	created2, updated2 := taskTimes()
	if !created2.Equal(created1) {
		t.Fatalf("created_at changed across save: %v -> %v (should be preserved)", created1, created2)
	}
	if !updated2.After(updated1) {
		t.Fatalf("updated_at did not advance across save: %v -> %v", updated1, updated2)
	}
}

func TestIntegration_GetNotFound(t *testing.T) {
	ctx := context.Background()
	_, readRepo := setup(t)

	missing := intermodal_old.NewDispatchID()
	_, err := readRepo(missing).Get(ctx, missing)
	if err == nil {
		t.Fatal("expected a not-found error")
	}
}
