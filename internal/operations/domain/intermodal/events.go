package intermodal

import "github.com/google/uuid"

// Domain events raised by the Dispatch aggregate (rdt: events.py). Each is a
// plain payload carrying only its domain fields and SubjectID — the aggregate
// it concerns (the outbox's aggregate_id). The delivery envelope (event id +
// occurred-at) is stamped at record time by domain.Root, not here, so these
// structs are constructed as bare literals: DispatchQueued{DispatchID: id, ...}.

// DispatchQueued: a dispatch was created and queued.
type DispatchQueued struct {
	DispatchID    DispatchID
	SourceQuoteID SourceQuoteID // zero when the dispatch has no source quote
}

func (e DispatchQueued) SubjectID() uuid.UUID { return e.DispatchID.UUID() }

// DispatchDelegated: a dispatch was delegated to a dispatcher.
type DispatchDelegated struct {
	DispatchID     DispatchID
	ToDispatcherID DispatcherID
}

func (e DispatchDelegated) SubjectID() uuid.UUID { return e.DispatchID.UUID() }

// DispatchApproved: a delegated, fully-assigned dispatch was approved and is
// ready to start. (rdt annotated driver_id as DispatcherID — a typo; it is the
// assigned DriverID, corrected here.)
type DispatchApproved struct {
	DispatchID DispatchID
	DriverID   DriverID
	TruckID    TruckID
}

func (e DispatchApproved) SubjectID() uuid.UUID { return e.DispatchID.UUID() }

// DispatchRequeued: an approved dispatch was reassigned, dropping it back to
// queued.
type DispatchRequeued struct {
	DispatchID DispatchID
}

func (e DispatchRequeued) SubjectID() uuid.UUID { return e.DispatchID.UUID() }

// DispatchVerified: a completed dispatch was verified.
type DispatchVerified struct {
	DispatchID    DispatchID
	SourceQuoteID SourceQuoteID
}

func (e DispatchVerified) SubjectID() uuid.UUID { return e.DispatchID.UUID() }

// DispatchStarted: the assigned driver began executing the agenda.
type DispatchStarted struct {
	DispatchID DispatchID
}

func (e DispatchStarted) SubjectID() uuid.UUID { return e.DispatchID.UUID() }

// DispatchHeldForEmptying: the dispatch is holding at a dock door to be emptied.
type DispatchHeldForEmptying struct {
	DispatchID DispatchID
	FacilityID FacilityID
}

func (e DispatchHeldForEmptying) SubjectID() uuid.UUID { return e.DispatchID.UUID() }

// DispatchHeldForLoading: the dispatch is holding at a dock door to be loaded.
type DispatchHeldForLoading struct {
	DispatchID DispatchID
	FacilityID FacilityID
}

func (e DispatchHeldForLoading) SubjectID() uuid.UUID { return e.DispatchID.UUID() }

// DispatchReleased: the dispatch was released from a hold after taking the
// container back.
type DispatchReleased struct {
	DispatchID DispatchID
	FacilityID FacilityID
}

func (e DispatchReleased) SubjectID() uuid.UUID { return e.DispatchID.UUID() }

// DispatchCompleted: the final agenda item ran; the move is done.
type DispatchCompleted struct {
	DispatchID    DispatchID
	SourceQuoteID SourceQuoteID
}

func (e DispatchCompleted) SubjectID() uuid.UUID { return e.DispatchID.UUID() }

// AssignmentProposed: a driver+truck pair was proposed for the dispatch.
type AssignmentProposed struct {
	DispatchID   DispatchID
	AssignmentID AssignmentID
	DriverID     DriverID
	TruckID      TruckID
}

func (e AssignmentProposed) SubjectID() uuid.UUID { return e.DispatchID.UUID() }

// ChassisAcquired: a chassis was taken into custody.
type ChassisAcquired struct {
	DispatchID    DispatchID
	FacilityID    FacilityID
	ChassisNumber ChassisNumber
}

func (e ChassisAcquired) SubjectID() uuid.UUID { return e.DispatchID.UUID() }

// ChassisReturned: a chassis was returned, closing its custody.
type ChassisReturned struct {
	DispatchID    DispatchID
	FacilityID    FacilityID
	ChassisNumber ChassisNumber
}

func (e ChassisReturned) SubjectID() uuid.UUID { return e.DispatchID.UUID() }

// ChassisHandedOff: a chassis custody was handed off (reserved; not yet raised).
type ChassisHandedOff struct {
	DispatchID    DispatchID
	ChassisNumber ChassisNumber
}

func (e ChassisHandedOff) SubjectID() uuid.UUID { return e.DispatchID.UUID() }

// ChassisReused: a chassis custody was reused (reserved; not yet raised).
type ChassisReused struct {
	DispatchID    DispatchID
	ChassisNumber ChassisNumber
}

func (e ChassisReused) SubjectID() uuid.UUID { return e.DispatchID.UUID() }

// EmptyContainerPickedUp: an empty container was picked up.
type EmptyContainerPickedUp struct {
	DispatchID      DispatchID
	FacilityID      FacilityID
	ContainerNumber ContainerNumber
}

func (e EmptyContainerPickedUp) SubjectID() uuid.UUID { return e.DispatchID.UUID() }

// EmptyContainerDelivered: an empty container was delivered.
type EmptyContainerDelivered struct {
	DispatchID      DispatchID
	FacilityID      FacilityID
	ContainerNumber ContainerNumber
}

func (e EmptyContainerDelivered) SubjectID() uuid.UUID { return e.DispatchID.UUID() }

// EmptyContainerTaken: an empty container was taken back.
type EmptyContainerTaken struct {
	DispatchID      DispatchID
	FacilityID      FacilityID
	ContainerNumber ContainerNumber
}

func (e EmptyContainerTaken) SubjectID() uuid.UUID { return e.DispatchID.UUID() }

// EmptyContainerDropped: an empty container was dropped.
type EmptyContainerDropped struct {
	DispatchID      DispatchID
	FacilityID      FacilityID
	ContainerNumber ContainerNumber
}

func (e EmptyContainerDropped) SubjectID() uuid.UUID { return e.DispatchID.UUID() }

// LoadedContainerPickedUp: a loaded container was picked up.
type LoadedContainerPickedUp struct {
	DispatchID      DispatchID
	FacilityID      FacilityID
	ContainerNumber ContainerNumber
}

func (e LoadedContainerPickedUp) SubjectID() uuid.UUID { return e.DispatchID.UUID() }

// LoadedContainerDelivered: a loaded container was delivered.
type LoadedContainerDelivered struct {
	DispatchID      DispatchID
	FacilityID      FacilityID
	ContainerNumber ContainerNumber
}

func (e LoadedContainerDelivered) SubjectID() uuid.UUID { return e.DispatchID.UUID() }

// LoadedContainerTaken: a loaded container was taken back.
type LoadedContainerTaken struct {
	DispatchID      DispatchID
	FacilityID      FacilityID
	ContainerNumber ContainerNumber
}

func (e LoadedContainerTaken) SubjectID() uuid.UUID { return e.DispatchID.UUID() }

// LoadedContainerDropped: a loaded container was dropped.
type LoadedContainerDropped struct {
	DispatchID      DispatchID
	FacilityID      FacilityID
	ContainerNumber ContainerNumber
}

func (e LoadedContainerDropped) SubjectID() uuid.UUID { return e.DispatchID.UUID() }
