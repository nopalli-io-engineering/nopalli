-- +goose Up
-- The Dispatch aggregate: a root table plus one child table per owned
-- collection (tasks, assignments, chassis/container custody episodes),
-- each FK'd back to the root with ON DELETE CASCADE so the aggregate is the
-- transactional boundary. Enums are stored as text (app-level enums), ids as
-- uuid, matching the nopalli domain model. Ported from rdt's operations schema;
-- nopalli owns this migration.

CREATE SCHEMA IF NOT EXISTS operations;

CREATE TABLE operations.intermodal_dispatches (
    id                    uuid PRIMARY KEY,
    -- Human-friendly sequential identifier, assigned by the database.
    number                bigint GENERATED ALWAYS AS IDENTITY NOT NULL UNIQUE,
    -- Nullable: a dispatch may exist without an originating quote. When present
    -- it is unique (at most one dispatch per accepted quote); Postgres treats
    -- NULLs as distinct, so quote-less dispatches are unconstrained.
    source_quote_id       uuid NULL UNIQUE,
    plan_type             text NOT NULL,
    status                text NOT NULL,
    -- Position of the next task to execute; NULL until execution begins.
    task_cursor           integer NULL,
    -- The assignment currently steering the dispatch: a soft reference into
    -- intermodal_dispatch_assignments (same aggregate), not a hard FK, to avoid
    -- a create-order cycle.
    current_assignment_id uuid NULL,
    -- The dispatcher the dispatch is delegated to; NULL until delegated.
    dispatcher_id         uuid NULL,
    -- Optimistic-concurrency token, bumped on each write.
    version               integer NOT NULL DEFAULT 0,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE operations.intermodal_dispatch_tasks (
    id                uuid PRIMARY KEY,
    dispatch_id       uuid NOT NULL REFERENCES operations.intermodal_dispatches (id) ON DELETE CASCADE,
    -- 1-based position within the dispatch's tasks; ordering is significant.
    task_no           integer NOT NULL,
    action            text NOT NULL,
    facility_id       uuid NULL,
    status            text NOT NULL,
    -- The task's details map (e.g. the container number captured at pickup), a
    -- small VO bag with no independent query needs — kept as JSONB.
    details           jsonb NOT NULL DEFAULT '{}',
    -- Who actually executed the task, recorded at execution time.
    attributed_driver uuid NULL,
    attributed_truck  uuid NULL,
    CONSTRAINT uq_intermodal_dispatch_tasks_position UNIQUE (dispatch_id, task_no)
);

CREATE INDEX ix_intermodal_dispatch_tasks_dispatch
    ON operations.intermodal_dispatch_tasks (dispatch_id);

CREATE TABLE operations.intermodal_dispatch_assignments (
    id          uuid PRIMARY KEY,
    dispatch_id uuid NOT NULL REFERENCES operations.intermodal_dispatches (id) ON DELETE CASCADE,
    -- Either side may be null transiently while the pair is built up; approval
    -- is where the aggregate requires both.
    driver_id   uuid NULL,
    truck_id    uuid NULL,
    status      text NOT NULL
);

CREATE INDEX ix_intermodal_dispatch_assignments_dispatch
    ON operations.intermodal_dispatch_assignments (dispatch_id);

CREATE TABLE operations.intermodal_dispatch_chassis_custody_episodes (
    id             uuid PRIMARY KEY,
    dispatch_id    uuid NOT NULL REFERENCES operations.intermodal_dispatches (id) ON DELETE CASCADE,
    status         text NOT NULL,
    close          text NULL,
    sourced_from   uuid NOT NULL,
    stationed_at   uuid NULL,
    -- Nullable: a chassis may be taken into custody before its number is known.
    chassis_number text NULL
);

CREATE INDEX ix_intermodal_dispatch_chassis_custody_episodes_dispatch
    ON operations.intermodal_dispatch_chassis_custody_episodes (dispatch_id);

CREATE TABLE operations.intermodal_dispatch_container_custody_episodes (
    id               uuid PRIMARY KEY,
    dispatch_id      uuid NOT NULL REFERENCES operations.intermodal_dispatches (id) ON DELETE CASCADE,
    status           text NOT NULL,
    close            text NULL,
    sourced_from     uuid NOT NULL,
    stationed_at     uuid NULL,
    -- Required: a container custody episode is always opened with its number.
    container_number text NOT NULL
);

CREATE INDEX ix_intermodal_dispatch_container_custody_episodes_dispatch
    ON operations.intermodal_dispatch_container_custody_episodes (dispatch_id);

-- +goose Down
DROP TABLE operations.intermodal_dispatch_container_custody_episodes;
DROP TABLE operations.intermodal_dispatch_chassis_custody_episodes;
DROP TABLE operations.intermodal_dispatch_assignments;
DROP TABLE operations.intermodal_dispatch_tasks;
DROP TABLE operations.intermodal_dispatches;
DROP SCHEMA IF EXISTS operations;
