# nopalli — Go rewrite conventions

nopalli is a ground-up Go rewrite of the `rdt` Python application (the sibling
repo). `rdt` is the behavioral spec: same domains, same database, same wire
formats. We mirror its bounded-context structure but write idiomatic Go rather
than transliterating Python.

Work is proceeding **module by module, starting with `operations`**, and within
a module **domain layer first**.

## Layering (mirrors rdt, trimmed for Go)

```
internal/
  core/domain/                     shared domain primitives (rdt: app/core/domain)
  <context>/domain/<aggregate>/    one package per aggregate (rdt: modules/<ctx>/domain/aggregates/<agg>)
  <context>/application/           services + read models (later)
  <context>/infrastructure/        db repos, messaging, adapters (later)
  <context>/interfaces/            http, bff, pubsub (later)
```

Current: `internal/operations/domain/intermodal` (package named after the move
family, not the aggregate, so `intermodal.Dispatch` reads without stutter and
sibling move types like `flatbed` fit alongside it).

## Naming — trim the stutter

The package qualifies every name, so we drop the aggregate prefix rdt needed for
module-global names: `DispatchStatus` → `Status`, `DispatchAction` → `Action`.
We keep a qualifier only where trimming would collide (three different status
enums → `AssignmentStatus`, `CustodyEpisodeStatus`, `AgendaItemStatus`). Each
ported symbol notes its rdt name in a doc comment for traceability.

## Identity — distinct struct types per id

Each id is a distinct named type embedding `domain.ID`
(`type DispatchID struct{ domain.ID }`). The compiler keeps id types apart; the
UUID plumbing (v7 generation, parse, zero-check, string) lives once in
`domain.ID`. `New<T>ID()` mints; `Parse<T>ID(string)` validates at boundaries.
UUIDv7 via `github.com/google/uuid` (stdlib has none), matching rdt's `uuid7`.

## Value objects & enums

- Value objects with invariants (e.g. `ContainerNumber`) have an unexported
  field and a validating constructor returning `(T, error)`; they stay
  comparable so `==` works.
- Enums are typed string constants with a `Valid()` method. String values are
  **identical to rdt's** so DB rows and event payloads stay wire-compatible.

## Optionality

Optional fields are pointers (`*ChassisNumber`), explicit in signatures — the
Go stand-in for rdt's `X | None`.

## Errors

Domain rule violations are `*domain.Error` carrying a stable
`namespace.code` (e.g. `operations.dispatch_action_out_of_sequence`). Functions
return errors; they do not panic. rdt's `Result` type is dropped in favor of
Go's `(T, error)`.

## Events & aggregates

- `domain.DomainEvent` interface + embeddable `domain.BaseEvent` (event id +
  occurred-at). `SubjectID()` returns the aggregate's raw UUID (the outbox
  `aggregate_id`).
- Aggregate roots embed `domain.Root` for event bookkeeping
  (`RecordEvents` / `Events` / `PullEvents` / `ClearEvents`).

## Fidelity — port behavior, fix known quirks

We port rdt's behavior and its tests, but **fix** defects found along the way
rather than reproducing them (and encode the corrected behavior in tests).
Known items to address when their code is ported:
- duplicate error codes in rdt's operations errors (`dispatch_not_assigned`
  used for two errors; `DispatchAssignedAfterCompletion` reuses the
  `while_in_progress` code);
- `live_assignee` dereferences the current assignment without a nil guard.

## Application layer — thin orchestration over consumer-defined ports

The application service (`operations/application`) holds no business rules: each
command guards by permission, runs a `UnitOfWork`, loads or builds the
aggregate, invokes exactly one domain method, and saves. It depends only on
small interfaces it defines itself — `Repository`, `UnitOfWork` (which yields a
tx-bound `Repository`), `AccessControl` — so the layer has **no infrastructure
imports**; the pgx adapters in `operations/infrastructure/db` satisfy them. Events
recorded by the aggregate are written in the same transaction via the
repository's `EventRecorder` seam, so there is no separate post-commit publish.
(Deferred from rdt for now: the availability-policy check in `Start`, and the
inbox-idempotent `queue_dispatch` intake — the manual `CreateDispatch` path is
ported.)

## Persistence — hand-written Data Mapper, no ORM

nopalli **owns its own database and migrations** (it does not share rdt's DB).

- **Driver**: `jackc/pgx/v5` (+ `pgxpool`). **Migrations**: `pressly/goose` over an
  embedded FS (`internal/migrations`), applied with `coredb.Migrate`.
- **No ORM for aggregates.** GORM/ent fight our unexported fields, distinct id
  types and value objects. Instead each aggregate exposes a persistence
  contract — `Snapshot()` / `Reconstitute()` (the Go analogue of rdt's
  imperative mapping) — and infrastructure maps `Snapshot ↔ rows` with pgx.
  Infra never touches aggregate internals; `Reconstitute` raises no events.
- **Read side (CQRS queries / read models)**: flat pgx SELECTs over the root
  table, shaped straight into read-model DTOs (`application.DispatchSummary` /
  `DispatchDetail`) behind the `application.DispatchQueries` port — never by
  hydrating the aggregate. Hand-written pgx (not sqlc) to stay consistent with
  the write side and keep codegen out of the build; revisit sqlc if the query
  set grows large. Read-side authz is `Visibility` (filter, don't reject).
- **Transactions**: `coredb.UnitOfWork.Do(ctx, fn)` brackets a `pgx.Tx`;
  repositories bind to the `DBTX` (pool for reads, tx for `Add`/`Save`).
- **Optimistic concurrency** via a `version` column: `Save` does
  `UPDATE … WHERE id AND version = $old`; 0 rows → `ErrConcurrencyConflict`.
  The aggregate carries `version` as persistence bookkeeping (not domain state).
- **Child collections**: root + one child table each (FK `ON DELETE CASCADE`);
  `Save` reconciles children by delete-and-reinsert within the tx.
- **Outbox**: `Save` pulls domain events and hands them to an injected
  `EventRecorder` (no-op default) — the seam for the transactional outbox, which
  is its own later increment.

## Testing — in containers, like rdt

Tests run inside a container against the same image/services, mirroring rdt's
`docker compose run app pytest`:

- `make test` / `make test-unit` — `docker compose run --rm app go test -short ./...`
- `make test-integration` — brings up postgres/redis/minio, runs the full suite.
- `make test-host` — host-side `go test ./...` for a fast inner loop.

`-short` is the idiomatic stand-in for rdt's `not integration_test` marker:
service-backed tests guard with `if testing.Short() { t.Skip(...) }`.
