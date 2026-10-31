# REST API shape (operations / intermodal dispatch)

Design note for the HTTP layer over `application.IntermodalDispatchService`.
Mirrors rdt's `interfaces/api/v1/operations`, translated to our Go stack. The
command and query endpoints are both built now (package
`internal/operations/interfaces/rest`); see "Status" for what remains
(publicid codec, auth).

## Conventions

- Everything under `/v1/operations`, served on the `api.` host (once subdomain
  routing lands).
- **Reads** are plain resources served from **read-model projections** (flat
  SELECTs over the root table), never by hydrating the aggregate.
- **Writes** are state transitions on the Dispatch, each modeled as a **POST to
  a sub-resource noun** (not an RPC verb). One sub-resource ⇄ one service method
  ⇄ one permission ⇄ one valid-state set. More discoverable and edge-cacheable
  than a single `{action}` dispatcher, and it is the parity contract with rdt.
- Wire enums reuse the domain typed-string types (`intermodal.PlanType`,
  `intermodal.Status`) — their string values are the contract.

## Endpoints

| Method & path | Service call | Body | Success |
|---|---|---|---|
| `POST /dispatches` | `CreateDispatch` | `plan_type` + plan fields (+ optional `source_quote_id`) | 201 + `Location` + `{id}` |
| `GET /dispatches?status=&limit=&offset=` | `DispatchQueries.ListDispatches` | — | 200 `[summary]` |
| `GET /dispatches/{id}` | `DispatchQueries.GetDispatch` | — | 200 `detail` / 404 |
| `POST /dispatches/{id}/delegation` | `Delegate` | `{dispatcher_id}` | 204 |
| `POST /dispatches/{id}/assignment` | `Assign` | `{driver_id?, truck_id?}` | 201 + `{id}` |
| `POST /dispatches/{id}/approval` | `Approve` | — | 204 |
| `POST /dispatches/{id}/start` | `Start` | — | 204 |
| `POST /dispatches/{id}/verification` | `Verify` | — | 204 |
| `POST /dispatches/{id}/chassis-acquisition` | `AcquireChassis` | `{facility_id, chassis_number}` | 204 |
| `POST /dispatches/{id}/container-pickup` | `PickUpContainer` | — | 204 |
| `POST /dispatches/{id}/premounted-container-pickup` | `PickUpPremountedContainer` | `{chassis_number}` | 204 |
| `POST /dispatches/{id}/container-delivery` | `DeliverContainer` | — | 204 |
| `POST /dispatches/{id}/hold` | `Hold` | — | 204 |
| `POST /dispatches/{id}/container-take` | `TakeContainer` | — | 204 |
| `POST /dispatches/{id}/container-drop` | `DropContainer` | — | 204 |
| `POST /dispatches/{id}/premounted-container-drop` | `DropPremountedContainer` | — | 204 |
| `POST /dispatches/{id}/chassis-return` | `ReturnChassis` | `{facility_id}` | 204 |

`CreateDispatch` assembles a `LiveLoad`/`LiveUnload` from `plan_type` + the plan
fields and rejects `AD_HOC` with 422 (no standard tasks to build).

## Go pieces

- **Package** `internal/operations/interfaces/http`; cross-cutting bits (problem
  writer, error→status mapper, public-id codec, auth middleware) in
  `internal/core/interface/http`.
- **Handlers** are a struct over the ports — `svc *application.IntermodalDispatchService`
  and `read application.DispatchQueries` — each an `http.HandlerFunc` following
  one shape: decode body → convert wire types to domain types at the boundary
  (`intermodal.ParseDispatchID`, `NewContainerNumber`, …; conversion errors ⇒
  400/422 before the service runs) → call the service → map error or write the
  success status.
- **DTOs**: request/response structs with json tags; enums reuse the domain
  typed-string types.
- **Router**: Go 1.22 `http.ServeMux` method+path patterns
  (`"POST /v1/operations/dispatches/{dispatchID}/approval"`), zero deps —
  default; `chi` is a drop-in alternative if we want richer middleware.
- **Composition root** (`cmd/api/main.go`): pool → `db.NewUnitOfWork` →
  `application.NewIntermodalDispatchService` → handlers → routes → `http.Server`.
  Per-request transactions already live inside the service's `UnitOfWork`, so
  handlers are transaction-agnostic. The api does **not** migrate on boot; schema
  migrations are a separate explicit step — run `cmd/migrate` (`make migrate`)
  first.

## Error mapping (central)

Domain errors are machine-readable (`*domain.Error.Qualified()` = `namespace.code`):

- `operations.dispatch_not_found` → **404**
- state conflicts (`dispatch_not_ready`, `dispatch_not_in_progress`,
  `dispatch_already_assigned_assignee`, `dispatch_assigned_while_in_progress` /
  `_after_completion`, `dispatch_action_out_of_sequence`, `dispatch_not_assigned`,
  `dispatch_not_delegated`) → **409**
- `auth.not_authenticated` → **401**, `auth.forbidden` → **403**
- `errors.Is(ErrInvalidValue / ErrInvalidID)` → **422 / 400**

`OutOfSequenceError` unwraps to `*domain.Error`, so it's caught by the same
`errors.As` path. Rendered as problem+JSON carrying the stable `code`, so clients
switch on the code, not the message.

## Read-side authorization — filter, don't reject

Reads scope to a `Visibility` and return a (possibly empty) subset rather than
403-ing. Coarse for now: `VisibleAll` (everything) vs `VisibleNothing` (empty),
driven by an `operations:intermodal_dispatch.read_all` grant. An out-of-scope
`GET /dispatches/{id}` reads as **404** — never leak that the id is real.

## Open decisions

1. **ID encoding on the wire**: raw UUID strings (simplest) vs. rdt's opaque
   base64url `PublicID` (hides the sequential `number` / row counts). Parity =
   port a small `publicid` codec into `core/interface`.
2. **Problem body**: RFC 9457 (`type/title/status/detail/code`) vs. a lean
   `{code, message}`.
3. **Auth** is deferred — the API ships behind `application.AllowAll` until OIDC
   lands.

## Status

- **Built — command endpoints**: `internal/operations/interfaces/rest`
  (package `rest`, named to avoid shadowing `net/http`) — the 15 POST
  transitions over a consumer-defined `CommandService` interface, raw-UUID
  boundary parsing, `CreateDispatchRequest`/`AssignRequest`/etc. DTOs, and the
  `statusFor` error→status map. Shared mechanism in
  `internal/core/interface/httpx` (`WriteProblem`/`WriteJSON`/`DecodeJSON`,
  lean `{code,message}` body). Composition root at `cmd/api` (env: `DB_URL`,
  `PORT`; auth = `AllowAll`). Covered by `httptest` handler tests + a full-stack
  HTTP↔Postgres integration test.
- **Built — read side**: `application.DispatchSummary`/`DispatchDetail`,
  `application.DispatchQueries` + `Visibility`, pgx adapter `db.DispatchQueryService`.
- **Built — query endpoints**: `GET /dispatches` (status/limit/offset query
  params; empty list serialized as `[]`) and `GET /dispatches/{id}` in
  `rest/reads.go`, over the `application.DispatchQueries` port. The detail view
  projects the owned child collections (tasks, assignments, chassis /
  container custody episodes), each carrying its `created_at`/`updated_at` audit
  timestamps; the child arrays are always present (empty, not null). The child
  tables' audit columns are preserved across saves because the repository
  reconciles children by UPSERT-by-id + delete-missing rather than
  delete+insert. Reads scope via
  a `Handlers.visibility()` seam (returns `VisibleAll` until auth lands); an
  out-of-scope or unknown id is a 404, never leaking that the id is real.
  Response DTOs (`DispatchSummaryResponse`/`DispatchDetailResponse`) are
  raw-UUID strings with nullable ids omitted. Covered by `httptest` tests + the
  full-stack HTTP↔Postgres integration test.
- **Not yet wired**: the `publicid` codec decision (currently raw UUIDs) and
  auth (OIDC/session) — the `visibility()` seam is where read scoping will
  derive from the principal's `read_all` grant.
