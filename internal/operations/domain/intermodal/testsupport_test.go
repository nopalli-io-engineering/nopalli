package intermodal

import (
	"reflect"
	"testing"
	"time"

	"github.com/nopalli-io/nopalli/internal/core/domain"
)

// fixedClock is a deterministic clock for tests. Now returns the current instant
// without advancing; Advance steps it forward. A test thus gets stable,
// repeatable timestamps, and can mint ordered ones by advancing between calls.
type fixedClock struct{ t time.Time }

func newFixedClock(t time.Time) *fixedClock { return &fixedClock{t: t} }

func (c *fixedClock) Now() time.Time          { return c.t }
func (c *fixedClock) Advance(d time.Duration) { c.t = c.t.Add(d) }

func mustNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func eventTypeNames(events []domain.RecordedEvent) []string {
	out := make([]string, len(events))
	for i, e := range events {
		out[i] = reflect.TypeOf(e.DomainEvent).Name()
	}
	return out
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
