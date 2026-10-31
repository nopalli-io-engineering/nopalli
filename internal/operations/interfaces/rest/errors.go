package rest

import (
	"errors"

	"github.com/nopalli-io/nopalli/internal/core/domain"
	coredb "github.com/nopalli-io/nopalli/internal/core/infrastructure/db"
	"github.com/nopalli-io/nopalli/internal/operations/domain/intermodal_old"
)

// statusFor maps an error returned by the application service to an HTTP status
// and a stable machine-readable code. This mapping is the operations context's
// own concern (rdt kept it in the module, not core). The returned code is what
// clients switch on; the message is carried separately.
func statusFor(err error) (status int, code string) {
	switch {
	// Boundary/value validation that surfaced from the domain.
	case errors.Is(err, intermodal_old.ErrInvalidValue), errors.Is(err, domain.ErrInvalidID):
		return 400, "invalid_request"
	// Optimistic-concurrency loss — safe to retry after re-reading.
	case errors.Is(err, coredb.ErrConcurrencyConflict):
		return 409, "concurrency_conflict"
	}

	var de *domain.Error
	if errors.As(err, &de) {
		switch de.Qualified() {
		case "operations.dispatch_not_found":
			return 404, de.Code
		case "auth.not_authenticated":
			return 401, de.Code
		case "auth.forbidden":
			return 403, de.Code
		case
			// Acting on an aggregate whose state forbids it — retry after
			// re-reading, so these are conflicts (409).
			"operations.dispatch_not_ready",
			"operations.dispatch_not_in_progress",
			"operations.dispatch_already_assigned_assignee",
			"operations.dispatch_assigned_while_in_progress",
			"operations.dispatch_assigned_after_completion",
			"operations.dispatch_action_out_of_sequence",
			"operations.dispatch_not_assigned",
			"operations.dispatch_not_delegated":
			return 409, de.Code
		default:
			// Other domain-rule violations (missing driver/truck, empty
			// assignee): the request is understood but not processable.
			return 422, de.Code
		}
	}

	return 500, "internal_error"
}
