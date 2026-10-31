-- +goose Up
-- The child tables gain the same created_at/updated_at audit columns the root
-- already carries. This is also what makes the repository's child-write
-- strategy move from delete+insert to UPSERT-by-id: a blind delete+insert would
-- reset created_at on every save, so Save now updates rows in place (preserving
-- created_at, bumping updated_at) and deletes only the rows the aggregate no
-- longer owns.

ALTER TABLE operations.intermodal_dispatch_tasks
    ADD COLUMN created_at timestamptz NOT NULL DEFAULT now(),
    ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now();

ALTER TABLE operations.intermodal_dispatch_assignments
    ADD COLUMN created_at timestamptz NOT NULL DEFAULT now(),
    ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now();

ALTER TABLE operations.intermodal_dispatch_chassis_custody_episodes
    ADD COLUMN created_at timestamptz NOT NULL DEFAULT now(),
    ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now();

ALTER TABLE operations.intermodal_dispatch_container_custody_episodes
    ADD COLUMN created_at timestamptz NOT NULL DEFAULT now(),
    ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now();

-- +goose Down
ALTER TABLE operations.intermodal_dispatch_container_custody_episodes
    DROP COLUMN created_at, DROP COLUMN updated_at;
ALTER TABLE operations.intermodal_dispatch_chassis_custody_episodes
    DROP COLUMN created_at, DROP COLUMN updated_at;
ALTER TABLE operations.intermodal_dispatch_assignments
    DROP COLUMN created_at, DROP COLUMN updated_at;
ALTER TABLE operations.intermodal_dispatch_tasks
    DROP COLUMN created_at, DROP COLUMN updated_at;
