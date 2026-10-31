package intermodal_old

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidValue is wrapped by value-object constructors when an invariant is
// violated, so callers can match with errors.Is.
var ErrInvalidValue = errors.New("invalid value")

// ContainerNumber is an ISO 6346 marking — a natural key assigned by the
// container's owner, captured (not minted) at pickup (rdt: ContainerNumber).
// The field is unexported so the only way to hold one is through
// NewContainerNumber, which enforces non-emptiness; the type stays comparable,
// so two ContainerNumbers compare with ==.
type ContainerNumber struct {
	value string
}

// NewContainerNumber validates and constructs a container marking. rdt stripped
// and rejected empty/whitespace-only input; this does the same and stores the
// trimmed value.
func NewContainerNumber(raw string) (ContainerNumber, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return ContainerNumber{}, fmt.Errorf("%w: container number must not be empty", ErrInvalidValue)
	}
	return ContainerNumber{value: v}, nil
}

// String returns the marking text.
func (c ContainerNumber) String() string { return c.value }

// IsZero reports whether this is the zero value (no marking).
func (c ContainerNumber) IsZero() bool { return c.value == "" }

// ChassisNumber is a chassis identifier. It may be unknown when custody is
// taken and supplied later (rdt: ChassisNumber).
type ChassisNumber struct {
	value string
}

// NewChassisNumber validates and constructs a chassis identifier, rejecting
// empty/whitespace-only input (rdt: ChassisNumber.__post_init__).
func NewChassisNumber(raw string) (ChassisNumber, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return ChassisNumber{}, fmt.Errorf("%w: chassis number must not be empty", ErrInvalidValue)
	}
	return ChassisNumber{value: v}, nil
}

// String returns the chassis identifier text.
func (c ChassisNumber) String() string { return c.value }

// IsZero reports whether this is the zero value (no identifier).
func (c ChassisNumber) IsZero() bool { return c.value == "" }

// Assignee is a driver+truck pair proposed or active on a Dispatch (rdt:
// Assignee). Either side may be unset while an assignment is being built up; a
// zero id means "unset". We use zero-value sentinels rather than pointers here
// (unlike the general optionality convention) so Assignee stays a comparable
// value — equal pairs compare == and it can be a map/set key, matching rdt's
// frozen-dataclass equality. Build pairs through NewAssignee; inspect presence
// with HasDriver / HasTruck.
type Assignee struct {
	Driver DriverID
	Truck  TruckID
}

// NewAssignee builds a pair from optional driver/truck (nil → unset).
func NewAssignee(driver *DriverID, truck *TruckID) Assignee {
	var a Assignee
	if driver != nil {
		a.Driver = *driver
	}
	if truck != nil {
		a.Truck = *truck
	}
	return a
}

// HasDriver reports whether a driver is set.
func (a Assignee) HasDriver() bool { return !a.Driver.IsZero() }

// HasTruck reports whether a truck is set.
func (a Assignee) HasTruck() bool { return !a.Truck.IsZero() }

// IsEmpty reports whether neither driver nor truck is set.
func (a Assignee) IsEmpty() bool { return a.Driver.IsZero() && a.Truck.IsZero() }

// Equipment is the live chassis/container a Dispatch currently has in hand
// (rdt: Equipment). Either may be absent (zero value). Like Assignee it uses
// zero-value sentinels so it stays comparable.
type Equipment struct {
	Container ContainerNumber
	Chassis   ChassisNumber
}

// TaskDetails carries the per-task data an action may need — currently
// just the container marking for container actions (rdt: TaskDetails).
type TaskDetails struct {
	Container *ContainerNumber
}

// TaskSpec is the planned definition of one task, before it
// becomes a live Task on a Dispatch (rdt: TaskSpec). Facility is
// nil for actions that have no location (acquire/return chassis); Details is
// nil for actions that carry no container.
type TaskSpec struct {
	Action   Action
	Facility *FacilityID
	Details  *TaskDetails
}

// --- enums (ported from rdt's value_objects.py as typed string constants) --- //
//
// Go has no enum type. String values are identical to rdt's so persisted rows
// and event payloads stay wire-compatible. Naming follows "trim the stutter":
// because the package qualifies every name (intermodal_old.Status), rdt's
// DispatchStatus / DispatchAction become Status / Action here; the other status
// enums keep their qualifier to avoid three colliding Status types.

// Status is the Dispatch lifecycle state (rdt: DispatchStatus).
type Status string

const (
	StatusQueued     Status = "queued"
	StatusReady      Status = "ready"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
	StatusVerified   Status = "verified"
)

// Valid reports whether s is a known lifecycle state.
func (s Status) Valid() bool {
	switch s {
	case StatusQueued, StatusReady, StatusInProgress, StatusCompleted, StatusVerified:
		return true
	default:
		return false
	}
}

// PlanType is the shape of move a Dispatch was planned from (rdt: PlanType).
type PlanType string

const (
	PlanLiveLoad   PlanType = "live_load"
	PlanLiveUnload PlanType = "live_unload"
	PlanAdHoc      PlanType = "ad_hoc"
)

// Valid reports whether p is a known plan type.
func (p PlanType) Valid() bool {
	switch p {
	case PlanLiveLoad, PlanLiveUnload, PlanAdHoc:
		return true
	default:
		return false
	}
}

// Action is one step the driver performs while executing the tasks (rdt:
// DispatchAction). The series of task is an ordered list of these; the cursor enforces
// that they happen in sequence.
type Action string

const (
	ActionPickUpEmptyContainer            Action = "pick_up_empty_container"
	ActionPickUpLoadedContainer           Action = "pick_up_loaded_container"
	ActionPickUpEmptyPremountedContainer  Action = "pick_up_empty_premounted_container"
	ActionPickUpLoadedPremountedContainer Action = "pick_up_loaded_premounted_container"
	ActionDeliverEmptyContainer           Action = "deliver_empty_container"
	ActionDeliverLoadedContainer          Action = "deliver_loaded_container"
	ActionDropEmptyContainer              Action = "drop_empty_container"
	ActionDropLoadedContainer             Action = "drop_loaded_container"
	ActionDropEmptyPremountedContainer    Action = "drop_empty_premounted_container"
	ActionDropLoadedPremountedContainer   Action = "drop_loaded_premounted_container"
	ActionTakeEmptyContainer              Action = "take_empty_container"
	ActionTakeLoadedContainer             Action = "take_loaded_container"
	ActionAcquireChassis                  Action = "acquire_chassis"
	ActionReturnChassis                   Action = "return_chassis"
	ActionHoldForEmptying                 Action = "hold_for_emptying"
	ActionHoldForLoading                  Action = "hold_for_loading"
	ActionLeaveEmptyContainer             Action = "leave_empty_container"
	ActionLeaveLoadedContainer            Action = "leave_loaded_container"
)

// Valid reports whether a is a known dispatch action.
func (a Action) Valid() bool {
	switch a {
	case ActionPickUpEmptyContainer, ActionPickUpLoadedContainer,
		ActionPickUpEmptyPremountedContainer, ActionPickUpLoadedPremountedContainer,
		ActionDeliverEmptyContainer, ActionDeliverLoadedContainer,
		ActionDropEmptyContainer, ActionDropLoadedContainer,
		ActionDropEmptyPremountedContainer, ActionDropLoadedPremountedContainer,
		ActionTakeEmptyContainer, ActionTakeLoadedContainer,
		ActionAcquireChassis, ActionReturnChassis,
		ActionHoldForEmptying, ActionHoldForLoading,
		ActionLeaveEmptyContainer, ActionLeaveLoadedContainer:
		return true
	default:
		return false
	}
}

// AssignmentStatus is the lifecycle of a driver+truck assignment (rdt:
// AssignmentStatus).
type AssignmentStatus string

const (
	AssignmentPending   AssignmentStatus = "pending"
	AssignmentActive    AssignmentStatus = "active"
	AssignmentConcluded AssignmentStatus = "concluded"
)

// Valid reports whether s is a known assignment status.
func (s AssignmentStatus) Valid() bool {
	switch s {
	case AssignmentPending, AssignmentActive, AssignmentConcluded:
		return true
	default:
		return false
	}
}

// CustodyEpisodeStatus is the lifecycle of a chassis/container custody episode
// (rdt: CustodyEpisodeStatus).
type CustodyEpisodeStatus string

const (
	CustodyPending CustodyEpisodeStatus = "pending"
	CustodyOpened  CustodyEpisodeStatus = "opened"
	CustodyClosed  CustodyEpisodeStatus = "closed"
)

// Valid reports whether s is a known custody episode status.
func (s CustodyEpisodeStatus) Valid() bool {
	switch s {
	case CustodyPending, CustodyOpened, CustodyClosed:
		return true
	default:
		return false
	}
}

// ChassisClose records how a chassis custody episode ended (rdt: ChassisClose).
// Prefixed to avoid colliding with the ChassisReturned / ChassisHandedOff
// domain event types.
type ChassisClose string

const (
	ChassisCloseReturned  ChassisClose = "returned"
	ChassisCloseHandedOff ChassisClose = "handed_off"
)

// Valid reports whether c is a known chassis close reason.
func (c ChassisClose) Valid() bool {
	switch c {
	case ChassisCloseReturned, ChassisCloseHandedOff:
		return true
	default:
		return false
	}
}

// ContainerClose records how a container custody episode ended (rdt:
// ContainerClose).
type ContainerClose string

const (
	ContainerCloseReturned     ContainerClose = "returned"
	ContainerCloseStreetTurned ContainerClose = "street_turned"
)

// Valid reports whether c is a known container close reason.
func (c ContainerClose) Valid() bool {
	switch c {
	case ContainerCloseReturned, ContainerCloseStreetTurned:
		return true
	default:
		return false
	}
}

// TaskStatus is the lifecycle of a single task (rdt:
// TaskStatus).
type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskCompleted TaskStatus = "completed"
	TaskSkipped   TaskStatus = "skipped"
)

// Valid reports whether s is a known task status.
func (s TaskStatus) Valid() bool {
	switch s {
	case TaskPending, TaskCompleted, TaskSkipped:
		return true
	default:
		return false
	}
}
