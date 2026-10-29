package domain

import (
	"time"

	"github.com/google/uuid"
)

// DomainEvent is a domain event payload: it knows only which aggregate it is
// about (rdt: core.domain.event.DomainEvent.subject_id). The delivery envelope
// — a unique, time-ordered id and the instant it occurred — is NOT the
// aggregate method's concern; it is stamped once, at record time, by
// Root.RecordEvents. So concrete events carry only their domain fields and a
// SubjectID method, never a hand-built envelope.
type DomainEvent interface {
	SubjectID() uuid.UUID
}

// RecordedEvent is a domain event plus the envelope stamped when it was
// recorded. This is what PullEvents returns and what the transactional outbox
// persists: EventID → messages.event_id, SubjectID → messages.aggregate_id,
// OccurredAt → messages.occurred_at, and the embedded DomainEvent is the
// payload to serialize.
type RecordedEvent struct {
	DomainEvent
	EventID    uuid.UUID
	OccurredAt time.Time
}

// recordEvent attaches a fresh envelope (v7 id, current UTC instant) to a
// payload — the single place event metadata is minted.
func recordEvent(e DomainEvent) RecordedEvent {
	return RecordedEvent{
		DomainEvent: e,
		EventID:     uuid.Must(uuid.NewV7()),
		OccurredAt:  time.Now().UTC(),
	}
}
