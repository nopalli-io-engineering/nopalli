package application

import (
	"time"

	"github.com/google/uuid"

	"github.com/nopalli-io/nopalli/internal/operations/domain/intermodal"
)

// Read-side DTOs for the operations context (rdt: application/read_models.py).
// These views are served by flat projections over the tables, never by
// hydrating the Dispatch aggregate, so they use plain wire types (raw UUIDs)
// rather than the domain's typed ids. The enums reuse the domain typed-string
// types, whose values already are the wire format.

// DispatchSummary is the list-row projection of a dispatch.
type DispatchSummary struct {
	ID            uuid.UUID
	SourceQuoteID *uuid.UUID // nil when the dispatch has no source quote
	PlanType      intermodal.PlanType
	Status        intermodal.Status
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// DispatchDetail is the single-dispatch projection: the root fields plus the
// owned child collections, each a flat projection of its table (including the
// per-row created_at/updated_at audit timestamps).
type DispatchDetail struct {
	ID                uuid.UUID
	SourceQuoteID     *uuid.UUID // nil when the dispatch has no source quote
	DispatcherID      *uuid.UUID // nil until delegated
	PlanType          intermodal.PlanType
	Status            intermodal.Status
	CreatedAt         time.Time
	UpdatedAt         time.Time
	Tasks             []TaskView
	Assignments       []AssignmentView
	ChassisEpisodes   []ChassisEpisodeView
	ContainerEpisodes []ContainerEpisodeView
}

// TaskView is the read projection of one task row. The details JSON
// (container number) is not projected here; add it if a view needs it.
type TaskView struct {
	ID               uuid.UUID
	No               int
	Action           intermodal.Action
	FacilityID       *uuid.UUID
	Status           intermodal.TaskStatus
	AttributedDriver *uuid.UUID
	AttributedTruck  *uuid.UUID
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// AssignmentView is the read projection of one assignment row.
type AssignmentView struct {
	ID        uuid.UUID
	DriverID  *uuid.UUID
	TruckID   *uuid.UUID
	Status    intermodal.AssignmentStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ChassisEpisodeView is the read projection of one chassis custody row.
type ChassisEpisodeView struct {
	ID            uuid.UUID
	Status        intermodal.CustodyEpisodeStatus
	SourcedFrom   uuid.UUID
	StationedAt   *uuid.UUID
	ChassisNumber *string // nil until the chassis number is known
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ContainerEpisodeView is the read projection of one container custody row.
type ContainerEpisodeView struct {
	ID              uuid.UUID
	Status          intermodal.CustodyEpisodeStatus
	SourcedFrom     uuid.UUID
	StationedAt     *uuid.UUID
	ContainerNumber string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
