package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nopalli-io/nopalli/internal/operations/application"
	"github.com/nopalli-io/nopalli/internal/operations/domain/intermodal_old"
)

// Unit tests for the service's orchestration, using in-memory fakes — no DB.
// They assert the command shape (guard → load → one domain call → save) and
// error propagation, not persistence (that's the repository's integration test).

type fakeRepo struct {
	store  map[intermodal_old.DispatchID]*intermodal_old.Dispatch
	saves  int
	adds   int
	getErr error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{store: map[intermodal_old.DispatchID]*intermodal_old.Dispatch{}}
}

func (r *fakeRepo) Get(_ context.Context, id intermodal_old.DispatchID) (*intermodal_old.Dispatch, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	d, ok := r.store[id]
	if !ok {
		return nil, intermodal_old.NewDispatchNotFound(id)
	}
	return d, nil
}

func (r *fakeRepo) Add(_ context.Context, d *intermodal_old.Dispatch) error {
	r.adds++
	r.store[d.ID()] = d
	return nil
}

func (r *fakeRepo) Save(_ context.Context, d *intermodal_old.Dispatch) error {
	r.saves++
	r.store[d.ID()] = d
	return nil
}

// fakeUoW runs fn immediately against the one repo (a committed transaction).
type fakeUoW struct{ repo application.Repository }

func (u fakeUoW) Do(ctx context.Context, fn func(application.Repository) error) error {
	return fn(u.repo)
}

type denyAll struct{ err error }

func (d denyAll) Require(context.Context, string) error { return d.err }

func liveLoad(t *testing.T) intermodal_old.LiveLoad {
	t.Helper()
	c, err := intermodal_old.NewContainerNumber("MSCU1234567")
	if err != nil {
		t.Fatal(err)
	}
	return intermodal_old.LiveLoad{
		ContainerNumber:      c,
		ContainerPickupRamp:  intermodal_old.NewFacilityID(),
		ContainerDropoffRamp: intermodal_old.NewFacilityID(),
		DockDoor:             intermodal_old.NewFacilityID(),
	}
}

func newService(repo application.Repository, access application.AccessControl) *application.IntermodalDispatchService {
	return application.NewIntermodalDispatchService(fakeUoW{repo: repo}, access)
}

func TestService_FullLifecycle(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	svc := newService(repo, application.AllowAll{})

	id, err := svc.CreateDispatch(ctx, liveLoad(t), nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if repo.adds != 1 {
		t.Fatalf("adds = %d, want 1", repo.adds)
	}

	dispatcher := intermodal_old.NewDispatcherID()
	driver := intermodal_old.NewDriverID()
	truck := intermodal_old.NewTruckID()
	chassis, _ := intermodal_old.NewChassisNumber("CHAS-1")
	yard := intermodal_old.NewFacilityID()

	mustCall := func(label string, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
	}
	mustCall("delegate", svc.Delegate(ctx, id, dispatcher))
	if _, err := svc.Assign(ctx, id, &driver, &truck); err != nil {
		t.Fatalf("assign: %v", err)
	}
	mustCall("approve", svc.Approve(ctx, id))
	mustCall("start", svc.Start(ctx, id))
	mustCall("acquire", svc.AcquireChassis(ctx, id, yard, chassis))
	mustCall("pickup", svc.PickUpContainer(ctx, id))
	mustCall("deliver", svc.DeliverContainer(ctx, id))
	mustCall("hold", svc.Hold(ctx, id))
	mustCall("take", svc.TakeContainer(ctx, id))
	mustCall("drop", svc.DropContainer(ctx, id))
	mustCall("return", svc.ReturnChassis(ctx, id, yard))
	mustCall("verify", svc.Verify(ctx, id))

	if got := repo.store[id].Status(); got != intermodal_old.StatusVerified {
		t.Fatalf("final status = %v, want verified", got)
	}
	if repo.saves == 0 {
		t.Fatal("expected Save to have been called")
	}
}

func TestService_PermissionDeniedShortCircuits(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	sentinel := errors.New("denied")
	svc := newService(repo, denyAll{err: sentinel})

	_, err := svc.CreateDispatch(ctx, liveLoad(t), nil)
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel deny", err)
	}
	if repo.adds != 0 {
		t.Fatal("permission denial must short-circuit before touching the repo")
	}
}

func TestService_DomainErrorPropagates(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	svc := newService(repo, application.AllowAll{})

	id, err := svc.CreateDispatch(ctx, liveLoad(t), nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// Start before approval → the aggregate rejects it; the service propagates.
	if err := svc.Start(ctx, id); !errors.Is(err, intermodal_old.ErrDispatchNotReady) {
		t.Fatalf("err = %v, want ErrDispatchNotReady", err)
	}
}

func TestService_GetNotFoundPropagates(t *testing.T) {
	ctx := context.Background()
	svc := newService(newFakeRepo(), application.AllowAll{})
	err := svc.Approve(ctx, intermodal_old.NewDispatchID())
	if err == nil {
		t.Fatal("expected a not-found error")
	}
}
