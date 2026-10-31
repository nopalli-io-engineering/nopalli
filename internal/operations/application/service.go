// Package application holds the operations context's application services — the
// thin orchestration layer over the domain (rdt: modules/operations/
// application/services). A service holds no business rules; it guards by
// permission, runs a unit of work, loads or builds the aggregate, invokes
// exactly one domain method, and lets the unit of work commit. Domain events
// recorded by the aggregate are written in the same transaction by the
// repository's event seam (the transactional outbox), so there is no separate
// post-commit publish here.
package application

import (
	"context"

	"github.com/nopalli-io/nopalli/internal/operations/domain/intermodal_old"
)

// Repository is the persistence port the service needs (rdt: domain/ports/
// intermodal_dispatch_repository, plus an explicit Save since Go has no
// identity map). Defined here, in the consumer, and satisfied by the pgx
// adapter in operations/infrastructure/db.
type Repository interface {
	Get(ctx context.Context, id intermodal_old.DispatchID) (*intermodal_old.Dispatch, error)
	Add(ctx context.Context, d *intermodal_old.Dispatch) error
	Save(ctx context.Context, d *intermodal_old.Dispatch) error
}

// UnitOfWork runs fn inside one transaction, handing it a Repository bound to
// that transaction, and commits iff fn returns nil (rdt: the UoW the service's
// repositories share). The DB/transaction mechanics live behind this interface
// so the application layer stays free of infrastructure.
type UnitOfWork interface {
	Do(ctx context.Context, fn func(repo Repository) error) error
}

// AccessControl guards a command by permission (rdt: core.domain.ports.
// AccessControl). Require returns a non-nil error (e.g. domain.PermissionDenied)
// when the caller may not perform the action.
type AccessControl interface {
	Require(ctx context.Context, permission string) error
}

// AllowAll is an AccessControl that permits every action — a stand-in for
// composition in dev and tests until a real policy is wired in.
type AllowAll struct{}

// Require always permits.
func (AllowAll) Require(context.Context, string) error { return nil }

// Permission strings, kept identical to rdt for parity.
const (
	permCreate                  = "operations:intermodal_dispatch.create"
	permDelegate                = "operations:intermodal_dispatch.delegate"
	permAssign                  = "operations:intermodal_dispatch.assign"
	permApprove                 = "operations:intermodal_dispatch.approve"
	permStart                   = "operations:intermodal_dispatch.start"
	permVerify                  = "operations:intermodal_dispatch.verify"
	permAcquireChassis          = "operations:intermodal_dispatch.acquire_chassis"
	permPickUpContainer         = "operations:intermodal_dispatch.pick_up_container"
	permPickUpPremounted        = "operations:intermodal_dispatch.pick_up_premounted_container"
	permDeliverContainer        = "operations:intermodal_dispatch.deliver_container"
	permHold                    = "operations:intermodal_dispatch.hold"
	permTakeContainer           = "operations:intermodal_dispatch.take_container"
	permDropContainer           = "operations:intermodal_dispatch.drop_container"
	permDropPremountedContainer = "operations:intermodal_dispatch.drop_premounted_container"
	permReturnChassis           = "operations:intermodal_dispatch.return_chassis"
)

// IntermodalDispatchService orchestrates commands against the Dispatch
// aggregate (rdt: IntermodalDispatchService).
type IntermodalDispatchService struct {
	uow    UnitOfWork
	access AccessControl
}

// NewIntermodalDispatchService wires the service.
func NewIntermodalDispatchService(uow UnitOfWork, access AccessControl) *IntermodalDispatchService {
	return &IntermodalDispatchService{uow: uow, access: access}
}

// --- intake --------------------------------------------------------------- //

// CreateDispatch builds and queues a dispatch from a move plan (rdt:
// create_dispatch — the manual/ad-hoc path). sourceQuoteID is optional.
//
// rdt's event-driven, inbox-idempotent queue_dispatch is intentionally not
// ported yet: it depends on the transactional inbox (ProcessedEventStore),
// which is a later increment.
func (s *IntermodalDispatchService) CreateDispatch(
	ctx context.Context, move intermodal_old.MovePlan, sourceQuoteID *intermodal_old.SourceQuoteID,
) (intermodal_old.DispatchID, error) {
	if err := s.access.Require(ctx, permCreate); err != nil {
		return intermodal_old.DispatchID{}, err
	}
	var id intermodal_old.DispatchID
	err := s.uow.Do(ctx, func(repo Repository) error {
		d := intermodal_old.Queue(sourceQuoteID, move)
		id = d.ID()
		return repo.Add(ctx, d)
	})
	if err != nil {
		return intermodal_old.DispatchID{}, err
	}
	return id, nil
}

// --- assignment, readiness, verification ---------------------------------- //

// Delegate assigns the dispatch to a dispatcher.
func (s *IntermodalDispatchService) Delegate(
	ctx context.Context, dispatchID intermodal_old.DispatchID, dispatcherID intermodal_old.DispatcherID,
) error {
	return s.mutate(ctx, permDelegate, dispatchID, func(d *intermodal_old.Dispatch) error {
		d.Delegate(dispatcherID)
		return nil
	})
}

// Assign proposes or updates the driver+truck pair; returns the current
// assignment id.
func (s *IntermodalDispatchService) Assign(
	ctx context.Context, dispatchID intermodal_old.DispatchID,
	driverID *intermodal_old.DriverID, truckID *intermodal_old.TruckID,
) (intermodal_old.AssignmentID, error) {
	if err := s.access.Require(ctx, permAssign); err != nil {
		return intermodal_old.AssignmentID{}, err
	}
	var assignmentID intermodal_old.AssignmentID
	err := s.uow.Do(ctx, func(repo Repository) error {
		d, err := repo.Get(ctx, dispatchID)
		if err != nil {
			return err
		}
		assignmentID, err = d.Assign(driverID, truckID)
		if err != nil {
			return err
		}
		return repo.Save(ctx, d)
	})
	if err != nil {
		return intermodal_old.AssignmentID{}, err
	}
	return assignmentID, nil
}

// Approve marks a delegated, fully-assigned dispatch ready to start.
func (s *IntermodalDispatchService) Approve(ctx context.Context, dispatchID intermodal_old.DispatchID) error {
	return s.mutate(ctx, permApprove, dispatchID, (*intermodal_old.Dispatch).Approve)
}

// Start begins task execution.
//
// rdt consults the AssigneeAvailabilityPolicy here (rejecting a start when the
// driver/truck is busy on another active dispatch). That check is intentionally
// omitted for now; it returns with a domain service + port that will be
// reintroduced as a dependency of this method.
func (s *IntermodalDispatchService) Start(ctx context.Context, dispatchID intermodal_old.DispatchID) error {
	return s.mutate(ctx, permStart, dispatchID, (*intermodal_old.Dispatch).Start)
}

// Verify marks a completed dispatch verified.
func (s *IntermodalDispatchService) Verify(ctx context.Context, dispatchID intermodal_old.DispatchID) error {
	return s.mutate(ctx, permVerify, dispatchID, (*intermodal_old.Dispatch).Verify)
}

// --- task execution ----------------------------------------------------- //
//
// Each drives the next task; the aggregate enforces action order and
// in-progress status, raising OutOfSequenceError / ErrDispatchNotInProgress if
// the step is not the one due.

// AcquireChassis executes the acquire-chassis step.
func (s *IntermodalDispatchService) AcquireChassis(
	ctx context.Context, dispatchID intermodal_old.DispatchID,
	facilityID intermodal_old.FacilityID, chassisNumber intermodal_old.ChassisNumber,
) error {
	return s.mutate(ctx, permAcquireChassis, dispatchID, func(d *intermodal_old.Dispatch) error {
		_, err := d.AcquireChassis(facilityID, chassisNumber)
		return err
	})
}

// PickUpContainer executes the pick-up-container step.
func (s *IntermodalDispatchService) PickUpContainer(ctx context.Context, dispatchID intermodal_old.DispatchID) error {
	return s.mutate(ctx, permPickUpContainer, dispatchID, func(d *intermodal_old.Dispatch) error {
		_, err := d.PickUpContainer()
		return err
	})
}

// PickUpPremountedContainer executes the premounted pick-up step.
func (s *IntermodalDispatchService) PickUpPremountedContainer(
	ctx context.Context, dispatchID intermodal_old.DispatchID, chassisNumber intermodal_old.ChassisNumber,
) error {
	return s.mutate(ctx, permPickUpPremounted, dispatchID, func(d *intermodal_old.Dispatch) error {
		_, err := d.PickUpPremountedContainer(chassisNumber)
		return err
	})
}

// DeliverContainer executes the deliver-container step.
func (s *IntermodalDispatchService) DeliverContainer(ctx context.Context, dispatchID intermodal_old.DispatchID) error {
	return s.mutate(ctx, permDeliverContainer, dispatchID, func(d *intermodal_old.Dispatch) error {
		_, err := d.DeliverContainer()
		return err
	})
}

// Hold executes the hold step.
func (s *IntermodalDispatchService) Hold(ctx context.Context, dispatchID intermodal_old.DispatchID) error {
	return s.mutate(ctx, permHold, dispatchID, func(d *intermodal_old.Dispatch) error {
		_, err := d.Hold()
		return err
	})
}

// TakeContainer executes the take-container step.
func (s *IntermodalDispatchService) TakeContainer(ctx context.Context, dispatchID intermodal_old.DispatchID) error {
	return s.mutate(ctx, permTakeContainer, dispatchID, func(d *intermodal_old.Dispatch) error {
		_, err := d.TakeContainer()
		return err
	})
}

// DropContainer executes the drop-container step.
func (s *IntermodalDispatchService) DropContainer(ctx context.Context, dispatchID intermodal_old.DispatchID) error {
	return s.mutate(ctx, permDropContainer, dispatchID, func(d *intermodal_old.Dispatch) error {
		_, err := d.DropContainer()
		return err
	})
}

// DropPremountedContainer executes the premounted drop step.
func (s *IntermodalDispatchService) DropPremountedContainer(ctx context.Context, dispatchID intermodal_old.DispatchID) error {
	return s.mutate(ctx, permDropPremountedContainer, dispatchID, func(d *intermodal_old.Dispatch) error {
		_, err := d.DropPremountedContainer()
		return err
	})
}

// ReturnChassis executes the return-chassis step.
func (s *IntermodalDispatchService) ReturnChassis(
	ctx context.Context, dispatchID intermodal_old.DispatchID, facilityID intermodal_old.FacilityID,
) error {
	return s.mutate(ctx, permReturnChassis, dispatchID, func(d *intermodal_old.Dispatch) error {
		_, err := d.ReturnChassis(facilityID)
		return err
	})
}

// --- shared command shape ------------------------------------------------- //

// mutate is the common load-one-method-save command: guard permission, open the
// unit of work, load the aggregate, apply exactly one domain method, and save.
func (s *IntermodalDispatchService) mutate(
	ctx context.Context,
	permission string,
	dispatchID intermodal_old.DispatchID,
	apply func(*intermodal_old.Dispatch) error,
) error {
	if err := s.access.Require(ctx, permission); err != nil {
		return err
	}
	return s.uow.Do(ctx, func(repo Repository) error {
		d, err := repo.Get(ctx, dispatchID)
		if err != nil {
			return err
		}
		if err := apply(d); err != nil {
			return err
		}
		return repo.Save(ctx, d)
	})
}
