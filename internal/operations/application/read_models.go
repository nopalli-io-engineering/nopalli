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

// DispatchDetail is the single-dispatch projection (adds the dispatcher).
type DispatchDetail struct {
	ID            uuid.UUID
	SourceQuoteID *uuid.UUID // nil when the dispatch has no source quote
	DispatcherID  *uuid.UUID // nil until delegated
	PlanType      intermodal.PlanType
	Status        intermodal.Status
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
