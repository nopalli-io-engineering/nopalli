package domain

// Root is embedded by every aggregate root to accumulate the domain events
// raised while handling a command (rdt: core.domain.entity.Aggregate). The
// application layer pulls them after a successful commit and hands them to the
// outbox. Composition, not inheritance: an aggregate embeds Root and gains the
// event bookkeeping without a base-class hierarchy.
//
//	type Dispatch struct {
//	    domain.Root
//	    // ... aggregate state ...
//	}
type Root struct {
	events []RecordedEvent
}

// RecordEvents stamps each payload with its envelope (id + timestamp) and
// appends it, in order. Aggregate methods pass plain event payloads; the
// envelope is minted here, once, so no method hand-builds one.
func (r *Root) RecordEvents(events ...DomainEvent) {
	for _, e := range events {
		r.events = append(r.events, recordEvent(e))
	}
}

// Events returns the recorded events so far without clearing them (rdt:
// Aggregate.events). The slice is a snapshot; mutating it does not affect the
// aggregate.
func (r *Root) Events() []RecordedEvent {
	out := make([]RecordedEvent, len(r.events))
	copy(out, r.events)
	return out
}

// PullEvents returns the recorded events and clears them in one step — the
// usual move once the work has been persisted (rdt: read Aggregate.events then
// clear_events). After this the aggregate has no pending events.
func (r *Root) PullEvents() []RecordedEvent {
	out := r.events
	r.events = nil
	return out
}

// ClearEvents discards any pending events (rdt: Aggregate.clear_events).
func (r *Root) ClearEvents() {
	r.events = nil
}
