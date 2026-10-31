package db_test

import (
	"context"
	"testing"

	coredb "github.com/nopalli-io/nopalli/internal/core/infrastructure/db"
	"github.com/nopalli-io/nopalli/internal/operations/application"
	"github.com/nopalli-io/nopalli/internal/operations/domain/intermodal_old"
	opsdb "github.com/nopalli-io/nopalli/internal/operations/infrastructure/db"
)

// Full-stack integration test: the application service driving the aggregate
// through the real UnitOfWork + pgx repository against Postgres. Each command is
// its own committed transaction, so this also exercises load/mutate/save
// round-trips between steps. Runs under `make test-integration`.

func TestIntegration_ServiceFullLifecycle(t *testing.T) {
	ctx := context.Background()
	dsn := dsnOrSkip(t) // skips under -short or when no DB is configured
	if err := coredb.Migrate(ctx, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := coredb.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	uow := opsdb.NewUnitOfWork(pool, nil)
	svc := application.NewIntermodalDispatchService(uow, application.AllowAll{})

	container, err := intermodal_old.NewContainerNumber("MSCU1234567")
	if err != nil {
		t.Fatal(err)
	}
	move := intermodal_old.LiveLoad{
		ContainerNumber:      container,
		ContainerPickupRamp:  intermodal_old.NewFacilityID(),
		ContainerDropoffRamp: intermodal_old.NewFacilityID(),
		DockDoor:             intermodal_old.NewFacilityID(),
	}

	id, err := svc.CreateDispatch(ctx, move, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	dispatcher := intermodal_old.NewDispatcherID()
	driver := intermodal_old.NewDriverID()
	truck := intermodal_old.NewTruckID()
	chassis, _ := intermodal_old.NewChassisNumber("CHAS-1")
	yard := intermodal_old.NewFacilityID()

	steps := []struct {
		label string
		run   func() error
	}{
		{"delegate", func() error { return svc.Delegate(ctx, id, dispatcher) }},
		{"assign", func() error { _, e := svc.Assign(ctx, id, &driver, &truck); return e }},
		{"approve", func() error { return svc.Approve(ctx, id) }},
		{"start", func() error { return svc.Start(ctx, id) }},
		{"acquire", func() error { return svc.AcquireChassis(ctx, id, yard, chassis) }},
		{"pickup", func() error { return svc.PickUpContainer(ctx, id) }},
		{"deliver", func() error { return svc.DeliverContainer(ctx, id) }},
		{"hold", func() error { return svc.Hold(ctx, id) }},
		{"take", func() error { return svc.TakeContainer(ctx, id) }},
		{"drop", func() error { return svc.DropContainer(ctx, id) }},
		{"return", func() error { return svc.ReturnChassis(ctx, id, yard) }},
		{"verify", func() error { return svc.Verify(ctx, id) }},
	}
	for _, s := range steps {
		if err := s.run(); err != nil {
			t.Fatalf("%s: %v", s.label, err)
		}
	}

	// Reload from the DB and confirm the persisted end state.
	loaded, err := opsdb.NewDispatchRepository(pool, nil).Get(ctx, id)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if loaded.Status() != intermodal_old.StatusVerified {
		t.Fatalf("persisted status = %v, want verified", loaded.Status())
	}
	if loaded.Version() == 0 {
		t.Fatal("expected version to have advanced through the saves")
	}
}
