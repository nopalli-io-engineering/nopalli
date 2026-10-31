package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/nopalli-io/nopalli/internal/operations/domain/intermodal_old"
)

// Query ports for the operations read side (rdt: application/queries.py). The
// read side is CQRS: these are served by flat projections over the tables, not
// by hydrating an aggregate. Defined here (the consumer — the HTTP layer) and
// satisfied by the pgx adapter in operations/infrastructure/db.

// Visibility is read-side authorization: reads scope to what the principal may
// see and return a (possibly empty) subset rather than rejecting the whole
// request. Coarse for now — everything or nothing, driven by a read_all grant.
// The struct leaves room for finer scoping (e.g. by tenant/owner) without
// changing the port.
type Visibility struct {
	// All is true when every dispatch is in scope; false means none are.
	All bool
}

// VisibleAll scopes to every dispatch.
func VisibleAll() Visibility { return Visibility{All: true} }

// VisibleNothing scopes to no dispatch; queries yield empty / not-found.
func VisibleNothing() Visibility { return Visibility{} }

// ListFilter narrows and pages a dispatch listing. A nil Status means all
// statuses; Limit <= 0 and Offset < 0 are normalized by the adapter.
type ListFilter struct {
	Status *intermodal_old.Status
	Limit  int
	Offset int
}

// DispatchQueries is the read-side port over dispatch projections.
type DispatchQueries interface {
	// ListDispatches returns summaries in scope, most recent first. An
	// out-of-scope visibility yields an empty slice.
	ListDispatches(ctx context.Context, vis Visibility, filter ListFilter) ([]DispatchSummary, error)

	// GetDispatch returns one dispatch's detail, or nil when it does not exist
	// or is out of scope (out-of-scope reads as not-found, never leaking that
	// the id is real).
	GetDispatch(ctx context.Context, id uuid.UUID, vis Visibility) (*DispatchDetail, error)
}
