package intermodal_old

import (
	"fmt"

	"github.com/nopalli-io/nopalli/internal/core/domain"
)

// Domain errors raised by the Dispatch aggregate (rdt: errors.py). All share
// the "operations" namespace. The parameterless ones are package-level
// *domain.Error values, so callers match them with errors.Is; the message text
// matches rdt exactly.
//
// Two codes are corrected from rdt (see CONVENTIONS.md "fix known quirks"):
//   - ErrDispatchNotDelegated used rdt's "dispatch_not_assigned" (collided with
//     ErrDispatchNotAssigned) and is now "dispatch_not_delegated";
//   - ErrDispatchAssignedAfterCompletion reused rdt's
//     "dispatch_assigned_while_in_progress" and is now
//     "dispatch_assigned_after_completion".
var (
	ErrDispatchNotDelegated = &domain.Error{
		Namespace: "operations",
		Code:      "dispatch_not_delegated",
		Message:   "A dispatch needs a dispatcher for approval.",
	}
	ErrDispatchNotAssigned = &domain.Error{
		Namespace: "operations",
		Code:      "dispatch_not_assigned",
		Message:   "A dispatch needs an assignment for approval.",
	}
	ErrDispatchAssigneeEmpty = &domain.Error{
		Namespace: "operations",
		Code:      "dispatch_assignee_empty",
		Message:   "Dispatch needs a valid assignee to carry out action.",
	}
	ErrDispatchDriverMissing = &domain.Error{
		Namespace: "operations",
		Code:      "dispatch_driver_missing",
		Message:   "Dispatch needs a driver.",
	}
	ErrDispatchTruckMissing = &domain.Error{
		Namespace: "operations",
		Code:      "dispatch_truck_missing",
		Message:   "Dispatch needs a truck.",
	}
	ErrDispatchNotReady = &domain.Error{
		Namespace: "operations",
		Code:      "dispatch_not_ready",
		Message:   "Only a ready dispatch can be started.",
	}
	ErrDispatchNotInProgress = &domain.Error{
		Namespace: "operations",
		Code:      "dispatch_not_in_progress",
		Message:   "Dispatch must be progress for action to be taken.",
	}
	ErrDispatchNotCompleted = &domain.Error{
		Namespace: "operations",
		Code:      "dispatch_not_completed",
		Message:   "Dispatch must be completed for action to be taken.",
	}
	ErrDispatchAlreadyAssignedAssignee = &domain.Error{
		Namespace: "operations",
		Code:      "dispatch_already_assigned_assignee",
		Message:   "Dispatch is already assigned the same driver-truck pair.",
	}
	ErrDispatchAssignedWhileInProgress = &domain.Error{
		Namespace: "operations",
		Code:      "dispatch_assigned_while_in_progress",
		Message:   "Dispatch cannot be assigned while in progress.",
	}
	ErrDispatchAssignedAfterCompletion = &domain.Error{
		Namespace: "operations",
		Code:      "dispatch_assigned_after_completion",
		Message:   "Dispatch cannot be assigned after completion.",
	}
)

// NewDispatchNotFound is raised by a repository when no dispatch has the given
// id (rdt: DispatchNotFound). Parameterized by id, so it is a constructor
// rather than a sentinel; match it on code via errors.As to *domain.Error.
func NewDispatchNotFound(id DispatchID) *domain.Error {
	return &domain.Error{
		Namespace: "operations",
		Code:      "dispatch_not_found",
		Message:   "no dispatch with id " + id.String(),
	}
}

// OutOfSequenceError is raised when the attempted execution action is not the
// one the task cursor is currently at (rdt: DispatchActionOutOfSequence). It
// carries the Expected action so callers can tell the driver what to do next.
// Match it with errors.As; it unwraps to its underlying *domain.Error.
type OutOfSequenceError struct {
	Expected Action
	err      *domain.Error
}

func newOutOfSequence(expected Action) *OutOfSequenceError {
	return &OutOfSequenceError{
		Expected: expected,
		err: &domain.Error{
			Namespace: "operations",
			Code:      "dispatch_action_out_of_sequence",
			Message:   fmt.Sprintf("Expected action '%s' not taken", expected),
		},
	}
}

// Error implements error.
func (e *OutOfSequenceError) Error() string { return e.err.Error() }

// Unwrap exposes the underlying domain error (for errors.As/Is and the code).
func (e *OutOfSequenceError) Unwrap() error { return e.err }
