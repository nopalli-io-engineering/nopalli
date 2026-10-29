// Package outbox is the transactional-outbox seam shared by every bounded
// context's repositories. A repository hands the events it pulled from an
// aggregate to a Recorder within the same transaction as the state change, so
// the intent-to-publish commits or rolls back atomically with the data.
//
// Only the seam lives here for now; the durable table-writer and the delivery
// relay that implement Recorder are a later increment. Repositories depend on
// the Recorder interface, never on a concrete outbox, so nothing context-
// specific leaks in.
package outbox

import (
	"context"

	"github.com/nopalli-io/nopalli/internal/core/domain"
	coredb "github.com/nopalli-io/nopalli/internal/core/infrastructure/db"
)

// Recorder persists the domain events raised by a command, in the transaction
// the repository is running in (rdt: the Outbox port + OutboxPublisher).
type Recorder interface {
	Record(ctx context.Context, q coredb.DBTX, events []domain.RecordedEvent) error
}

// Noop is a Recorder that discards events — the default until the durable
// outbox is wired in (like rdt's unsubscribed InMemoryEventPublisher, which
// logged and dropped).
type Noop struct{}

// Record implements Recorder by doing nothing.
func (Noop) Record(context.Context, coredb.DBTX, []domain.RecordedEvent) error { return nil }
