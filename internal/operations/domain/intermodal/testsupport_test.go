package intermodal

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/nopalli-io/nopalli/internal/core/kernel"
)

func idTransform[T fmt.Stringer]() cmp.Option {
	return cmp.Transformer("id", func(id T) string { return id.String() })
}

var idOpts = cmp.Options{
	idTransform[RunID](),
	idTransform[StopID](),
	idTransform[TaskID](),
	idTransform[AssignmentID](),
	idTransform[DriverID](),
	idTransform[TruckID](),
	idTransform[LocationID](),
	idTransform[DocumentID](),
	idTransform[kernel.TenantID](),
	idTransform[kernel.UserID](),
}

type Aggregate interface {
	Events() []kernel.RecordedEvent
}

func getEvents(aggregate Aggregate) []kernel.DomainEvent {
	domainEvents := []kernel.DomainEvent{}
	for _, r := range aggregate.Events() {
		domainEvents = append(domainEvents, r.DomainEvent)
	}
	return domainEvents
}

// fixedClock is a deterministic clock for tests. Now returns the current instant
// without advancing; Advance steps it forward. A test thus gets stable,
// repeatable timestamps, and can mint ordered ones by advancing between calls.
// type fixedClock struct{ t time.Time }

// func newFixedClock(t time.Time) *fixedClock { return &fixedClock{t: t} }

// func (c *fixedClock) Now() time.Time          { return c.t }
// func (c *fixedClock) Advance(d time.Duration) { c.t = c.t.Add(d) }

// func mustNoError(t *testing.T, err error) {
// 	t.Helper()
// 	if err != nil {
// 		t.Fatalf("unexpected error: %v", err)
// 	}
// }

// func eventTypeNames(events []kernel.RecordedEvent) []string {
// 	out := make([]string, len(events))
// 	for i, e := range events {
// 		out[i] = reflect.TypeOf(e.DomainEvent).Name()
// 	}
// 	return out
// }

// func assertTypeNames(t *testing.T, events []kernel.RecordedEvent, want ...string) {
// 	t.Helper()
// 	got := eventTypeNames(events)
// 	if len(got) != len(want) {
// 		t.Fatalf("event types = %v, want %v", got, want)
// 	}
// 	for i := range want {
// 		if got[i] != want[i] {
// 			t.Fatalf("event types = %v, want %v", got, want)
// 		}
// 	}
// }

// type eventTracker struct {
// 	r    *Run
// 	seen int
// }

// func trackFrom(r *Run) *eventTracker {
// 	return &eventTracker{r: r, seen: len(r.Events())}
// }

// func (tr *eventTracker) since() []kernel.RecordedEvent {
// 	all := tr.r.Events()
// 	fresh := all[tr.seen:]
// 	tr.seen = len(all)
// 	return fresh
// }

// func oneEvent[T kernel.DomainEvent](t *testing.T, events []kernel.RecordedEvent) T {
// 	t.Helper()
// 	var found T
// 	count := 0
// 	for _, e := range events {
// 		if ev, ok := e.DomainEvent.(T); ok {
// 			found = ev
// 			count++
// 		}
// 	}
// 	if count != 1 {
// 		t.Fatalf("expected exactly on %T, got %d, (events: %v)", found, count, eventTypeNames(events))
// 	}
// 	return found
// }

// func assertAssignment(t *testing.T, got Assignment, ok bool, wantDriver DriverID, wantTruck TruckID) {
// 	t.Helper()
// 	if !ok {
// 		t.Fatal("no current assignee")
// 	}
// }
