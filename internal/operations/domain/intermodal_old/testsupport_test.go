package intermodal_old

import (
	"reflect"
	"testing"

	"github.com/nopalli-io/nopalli/internal/core/domain"
)

// Shared test support, porting rdt's tests/helpers.py. (rdt's *_committed
// helpers referenced a non-existent AssignmentStatus.COMMITTED and are dropped.)

// eventTypeNames returns the concrete payload type name of each recorded event,
// in order (rdt: types_of). The payload is the embedded DomainEvent on the
// envelope.
func eventTypeNames(events []domain.RecordedEvent) []string {
	out := make([]string, len(events))
	for i, e := range events {
		out[i] = reflect.TypeOf(e.DomainEvent).Name()
	}
	return out
}

// oneEvent asserts exactly one recorded event has payload type T and returns
// that payload (rdt: one).
func oneEvent[T domain.DomainEvent](t *testing.T, events []domain.RecordedEvent) T {
	t.Helper()
	var found T
	count := 0
	for _, e := range events {
		if ev, ok := e.DomainEvent.(T); ok {
			found = ev
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one %T, got %d (events: %v)", found, count, eventTypeNames(events))
	}
	return found
}

// eventTracker captures only the events raised after it was created, so each
// test sees just its step's events (rdt: EventTracker).
type eventTracker struct {
	d    *Dispatch
	seen int
}

func trackFrom(d *Dispatch) *eventTracker {
	return &eventTracker{d: d, seen: len(d.Events())}
}

func (tr *eventTracker) since() []domain.RecordedEvent {
	all := tr.d.Events()
	fresh := all[tr.seen:]
	tr.seen = len(all)
	return fresh
}

// assertTaskCompletedBy mirrors assert_task_completion_by_assignee.
func assertTaskCompletedBy(t *testing.T, d *Dispatch, id TaskID, by Assignee) {
	t.Helper()
	task := d.Task(id)
	if task == nil {
		t.Fatalf("task %v not found", id)
	}
	if task.Status() != TaskCompleted {
		t.Fatalf("task status = %v, want completed", task.Status())
	}
	if task.AttributedDriver() != by.Driver {
		t.Fatalf("attributed driver = %v, want %v", task.AttributedDriver(), by.Driver)
	}
	if task.AttributedTruck() != by.Truck {
		t.Fatalf("attributed truck = %v, want %v", task.AttributedTruck(), by.Truck)
	}
}

func assertTypeNames(t *testing.T, events []domain.RecordedEvent, want ...string) {
	t.Helper()
	got := eventTypeNames(events)
	if len(got) != len(want) {
		t.Fatalf("event types = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("event types = %v, want %v", got, want)
		}
	}
}

// assertAssigneeSet checks the dispatch's assignees equal the given set.
func assertAssigneeSet(t *testing.T, d *Dispatch, want ...Assignee) {
	t.Helper()
	got := d.Assignees()
	if len(got) != len(want) {
		t.Fatalf("assignees = %v, want %v", got, want)
	}
	set := make(map[Assignee]struct{}, len(got))
	for _, a := range got {
		set[a] = struct{}{}
	}
	for _, w := range want {
		if _, ok := set[w]; !ok {
			t.Fatalf("assignees = %v, want %v", got, want)
		}
	}
}

func ptrDriver(d DriverID) *DriverID { return &d }
func ptrTruck(tk TruckID) *TruckID   { return &tk }

// mkContainer / mkChassis are panic-on-error constructors for use in package-level
// test fixtures (where *testing.T is not available).
func mkContainer(s string) ContainerNumber {
	c, err := NewContainerNumber(s)
	if err != nil {
		panic(err)
	}
	return c
}

func mkChassis(s string) ChassisNumber {
	c, err := NewChassisNumber(s)
	if err != nil {
		panic(err)
	}
	return c
}

func mustNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
