// Package db is the operations context's persistence adapter: a hand-written
// Data Mapper for the Dispatch aggregate (rdt: modules/operations/
// infrastructure/db). It maps the aggregate's persistence snapshot to and from
// SQL rows with pgx — no ORM, no reflection into aggregate internals. The
// aggregate owns the snapshot/reconstitute contract (persistence.go); this
// package owns the SQL.
package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	coredb "github.com/nopalli-io/nopalli/internal/core/infrastructure/db"
	"github.com/nopalli-io/nopalli/internal/core/infrastructure/outbox"
	"github.com/nopalli-io/nopalli/internal/operations/domain/intermodal_old"
)

// DispatchRepository loads and stores the Dispatch aggregate. Bind it to a
// DBTX: a *pgxpool.Pool for reads, or the tx from a UnitOfWork for writes
// (Add/Save must run inside a transaction to be atomic across the root and its
// child tables).
type DispatchRepository struct {
	db     coredb.DBTX
	events outbox.Recorder
}

// NewDispatchRepository builds a repository over a DBTX. A nil recorder means
// events are dropped (outbox.Noop).
func NewDispatchRepository(db coredb.DBTX, recorder outbox.Recorder) *DispatchRepository {
	if recorder == nil {
		recorder = outbox.Noop{}
	}
	return &DispatchRepository{db: db, events: recorder}
}

const selectRoot = `
SELECT source_quote_id, dispatcher_id, current_assignment_id, plan_type, status, task_cursor, version
FROM operations.intermodal_dispatches
WHERE id = $1`

// Get loads a fully-populated Dispatch by id (rdt: repository.get + selectin
// eager load). It returns operations' DispatchNotFound when absent.
func (r *DispatchRepository) Get(ctx context.Context, id intermodal_old.DispatchID) (*intermodal_old.Dispatch, error) {
	var (
		sourceQuote, dispatcher, currentAssignment pgtype.UUID
		planType, status                           string
		taskCursor                                 pgtype.Int4
		version                                    int32
	)
	err := r.db.QueryRow(ctx, selectRoot, pgID(id)).
		Scan(&sourceQuote, &dispatcher, &currentAssignment, &planType, &status, &taskCursor, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, intermodal_old.NewDispatchNotFound(id)
	}
	if err != nil {
		return nil, fmt.Errorf("operations: load dispatch: %w", err)
	}

	snap := intermodal_old.DispatchSnapshot{
		ID:                  id,
		SourceQuoteID:       optSourceQuote(sourceQuote),
		DispatcherID:        optDispatcher(dispatcher),
		CurrentAssignmentID: optAssignment(currentAssignment),
		PlanType:            intermodal_old.PlanType(planType),
		Status:              intermodal_old.Status(status),
		Cursor:              cursorFromDB(taskCursor),
		Version:             int(version),
	}

	var loadErr error
	if snap.Tasks, loadErr = r.loadTasks(ctx, id); loadErr != nil {
		return nil, loadErr
	}
	if snap.Assignments, loadErr = r.loadAssignments(ctx, id); loadErr != nil {
		return nil, loadErr
	}
	if snap.ChassisEpisodes, loadErr = r.loadChassisEpisodes(ctx, id); loadErr != nil {
		return nil, loadErr
	}
	if snap.ContainerEpisodes, loadErr = r.loadContainerEpisodes(ctx, id); loadErr != nil {
		return nil, loadErr
	}

	return intermodal_old.Reconstitute(snap), nil
}

const insertRoot = `
INSERT INTO operations.intermodal_dispatches
    (id, source_quote_id, plan_type, status, task_cursor, current_assignment_id, dispatcher_id, version)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

// Add inserts a newly-queued dispatch and its children (rdt: repository.add).
// Must run inside a UnitOfWork transaction.
func (r *DispatchRepository) Add(ctx context.Context, d *intermodal_old.Dispatch) error {
	s := d.Snapshot()
	_, err := r.db.Exec(ctx, insertRoot,
		pgID(s.ID), pgIDOpt(s.SourceQuoteID), string(s.PlanType), string(s.Status),
		cursorToDB(s.Cursor), pgIDOpt(s.CurrentAssignmentID), pgIDOpt(s.DispatcherID), int32(s.Version),
	)
	if err != nil {
		return fmt.Errorf("operations: insert dispatch: %w", err)
	}
	// A fresh aggregate has no stored children, so writeChildren only inserts
	// here; it shares the upsert path with Save for a single source of SQL.
	if err := r.writeChildren(ctx, s); err != nil {
		return err
	}
	if err := r.events.Record(ctx, r.db, d.PullEvents()); err != nil {
		return err
	}
	d.MarkPersisted(s.Version)
	return nil
}

const updateRoot = `
UPDATE operations.intermodal_dispatches
SET source_quote_id = $1, plan_type = $2, status = $3, task_cursor = $4,
    current_assignment_id = $5, dispatcher_id = $6, version = $7, updated_at = now()
WHERE id = $8 AND version = $9`

// Save persists changes to an already-stored dispatch under optimistic
// concurrency, then reconciles its child rows. Must run inside a UnitOfWork
// transaction. Returns ErrConcurrencyConflict if the row moved on since load.
//
// Children are reconciled to the snapshot by id: each snapshot row is upserted
// (insert, or update-in-place on conflict — preserving created_at, bumping
// updated_at), and rows the aggregate no longer owns are deleted. The whole
// aggregate is one transactional unit gated by the root's version, so there is
// no window where the children disagree with the root.
func (r *DispatchRepository) Save(ctx context.Context, d *intermodal_old.Dispatch) error {
	s := d.Snapshot()
	newVersion := s.Version + 1
	tag, err := r.db.Exec(ctx, updateRoot,
		pgIDOpt(s.SourceQuoteID), string(s.PlanType), string(s.Status), cursorToDB(s.Cursor),
		pgIDOpt(s.CurrentAssignmentID), pgIDOpt(s.DispatcherID), int32(newVersion),
		pgID(s.ID), int32(s.Version),
	)
	if err != nil {
		return fmt.Errorf("operations: update dispatch: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return coredb.ErrConcurrencyConflict
	}
	if err := r.writeChildren(ctx, s); err != nil {
		return err
	}
	if err := r.deleteMissingChildren(ctx, s); err != nil {
		return err
	}
	if err := r.events.Record(ctx, r.db, d.PullEvents()); err != nil {
		return err
	}
	d.MarkPersisted(newVersion)
	return nil
}

// --- child writes --------------------------------------------------------- //

// writeChildren upserts every child row from the snapshot by id. On conflict it
// updates the mutable columns in place: created_at is never listed, so it keeps
// the value from the original insert, and updated_at is bumped to now() — the
// same audit semantics the root row has. The custody episodes' close column is
// not part of the snapshot, so it is written once on insert and left untouched
// on update.
func (r *DispatchRepository) writeChildren(ctx context.Context, s intermodal_old.DispatchSnapshot) error {
	for _, t := range s.Tasks {
		details, err := encodeDetails(t.Container)
		if err != nil {
			return fmt.Errorf("operations: encode task details: %w", err)
		}
		_, err = r.db.Exec(ctx, `
INSERT INTO operations.intermodal_dispatch_tasks
    (id, dispatch_id, task_no, action, facility_id, status, details, attributed_driver, attributed_truck)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (id) DO UPDATE SET
    task_no = EXCLUDED.task_no, action = EXCLUDED.action, facility_id = EXCLUDED.facility_id,
    status = EXCLUDED.status, details = EXCLUDED.details,
    attributed_driver = EXCLUDED.attributed_driver, attributed_truck = EXCLUDED.attributed_truck,
    updated_at = now()`,
			pgID(t.ID), pgID(s.ID), int32(t.No), string(t.Action), pgIDOpt(t.FacilityID),
			string(t.Status), details, pgIDOpt(t.AttributedDriver), pgIDOpt(t.AttributedTruck),
		)
		if err != nil {
			return fmt.Errorf("operations: upsert tasks: %w", err)
		}
	}
	for _, a := range s.Assignments {
		_, err := r.db.Exec(ctx, `
INSERT INTO operations.intermodal_dispatch_assignments (id, dispatch_id, driver_id, truck_id, status)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE SET
    driver_id = EXCLUDED.driver_id, truck_id = EXCLUDED.truck_id, status = EXCLUDED.status,
    updated_at = now()`,
			pgID(a.ID), pgID(s.ID), pgIDOpt(a.DriverID), pgIDOpt(a.TruckID), string(a.Status),
		)
		if err != nil {
			return fmt.Errorf("operations: upsert assignment: %w", err)
		}
	}
	for _, e := range s.ChassisEpisodes {
		_, err := r.db.Exec(ctx, `
INSERT INTO operations.intermodal_dispatch_chassis_custody_episodes
    (id, dispatch_id, status, close, sourced_from, stationed_at, chassis_number)
VALUES ($1, $2, $3, NULL, $4, $5, $6)
ON CONFLICT (id) DO UPDATE SET
    status = EXCLUDED.status, sourced_from = EXCLUDED.sourced_from,
    stationed_at = EXCLUDED.stationed_at, chassis_number = EXCLUDED.chassis_number,
    updated_at = now()`,
			pgID(e.ID), pgID(s.ID), string(e.Status), pgID(e.SourcedFrom), pgIDOpt(e.StationedAt),
			chassisText(e.ChassisNumber),
		)
		if err != nil {
			return fmt.Errorf("operations: upsert chassis episode: %w", err)
		}
	}
	for _, e := range s.ContainerEpisodes {
		_, err := r.db.Exec(ctx, `
INSERT INTO operations.intermodal_dispatch_container_custody_episodes
    (id, dispatch_id, status, close, sourced_from, stationed_at, container_number)
VALUES ($1, $2, $3, NULL, $4, $5, $6)
ON CONFLICT (id) DO UPDATE SET
    status = EXCLUDED.status, sourced_from = EXCLUDED.sourced_from,
    stationed_at = EXCLUDED.stationed_at, container_number = EXCLUDED.container_number,
    updated_at = now()`,
			pgID(e.ID), pgID(s.ID), string(e.Status), pgID(e.SourcedFrom), pgIDOpt(e.StationedAt),
			e.ContainerNumber.String(),
		)
		if err != nil {
			return fmt.Errorf("operations: upsert container episode: %w", err)
		}
	}
	return nil
}

// deleteMissingChildren removes rows the aggregate no longer owns: per child
// table, everything under this dispatch whose id is not in the snapshot. An
// empty id set deletes every row of that kind (id <> ALL('{}') is true), which
// is the correct "the aggregate dropped them all" case.
func (r *DispatchRepository) deleteMissingChildren(ctx context.Context, s intermodal_old.DispatchSnapshot) error {
	tables := []struct {
		name string
		keep []string
	}{
		{"operations.intermodal_dispatch_tasks", taskIDs(s)},
		{"operations.intermodal_dispatch_assignments", assignmentIDs(s)},
		{"operations.intermodal_dispatch_chassis_custody_episodes", chassisEpisodeIDs(s)},
		{"operations.intermodal_dispatch_container_custody_episodes", containerEpisodeIDs(s)},
	}
	for _, t := range tables {
		_, err := r.db.Exec(ctx,
			"DELETE FROM "+t.name+" WHERE dispatch_id = $1 AND id <> ALL($2::uuid[])",
			pgID(s.ID), t.keep,
		)
		if err != nil {
			return fmt.Errorf("operations: prune children (%s): %w", t.name, err)
		}
	}
	return nil
}

func taskIDs(s intermodal_old.DispatchSnapshot) []string {
	ids := make([]string, 0, len(s.Tasks))
	for _, t := range s.Tasks {
		ids = append(ids, t.ID.String())
	}
	return ids
}

func assignmentIDs(s intermodal_old.DispatchSnapshot) []string {
	ids := make([]string, 0, len(s.Assignments))
	for _, a := range s.Assignments {
		ids = append(ids, a.ID.String())
	}
	return ids
}

func chassisEpisodeIDs(s intermodal_old.DispatchSnapshot) []string {
	ids := make([]string, 0, len(s.ChassisEpisodes))
	for _, e := range s.ChassisEpisodes {
		ids = append(ids, e.ID.String())
	}
	return ids
}

func containerEpisodeIDs(s intermodal_old.DispatchSnapshot) []string {
	ids := make([]string, 0, len(s.ContainerEpisodes))
	for _, e := range s.ContainerEpisodes {
		ids = append(ids, e.ID.String())
	}
	return ids
}

// --- child reads ---------------------------------------------------------- //

func (r *DispatchRepository) loadTasks(ctx context.Context, id intermodal_old.DispatchID) ([]intermodal_old.TaskSnapshot, error) {
	rows, err := r.db.Query(ctx, `
SELECT id, task_no, action, facility_id, status, details, attributed_driver, attributed_truck
FROM operations.intermodal_dispatch_tasks
WHERE dispatch_id = $1
ORDER BY task_no`, pgID(id))
	if err != nil {
		return nil, fmt.Errorf("operations: load tasks: %w", err)
	}
	defer rows.Close()

	var out []intermodal_old.TaskSnapshot
	for rows.Next() {
		var (
			taskID, facility, driver, truck pgtype.UUID
			taskNo                          int32
			action, status                  string
			details                         []byte
		)
		if err := rows.Scan(&taskID, &taskNo, &action, &facility, &status, &details, &driver, &truck); err != nil {
			return nil, fmt.Errorf("operations: scan task: %w", err)
		}
		container, err := decodeContainerFromDetails(details)
		if err != nil {
			return nil, err
		}
		out = append(out, intermodal_old.TaskSnapshot{
			ID:               mustTaskID(taskID),
			No:               int(taskNo),
			Action:           intermodal_old.Action(action),
			FacilityID:       optFacility(facility),
			Status:           intermodal_old.TaskStatus(status),
			Container:        container,
			AttributedDriver: optDriver(driver),
			AttributedTruck:  optTruck(truck),
		})
	}
	return out, rows.Err()
}

func (r *DispatchRepository) loadAssignments(ctx context.Context, id intermodal_old.DispatchID) ([]intermodal_old.AssignmentSnapshot, error) {
	rows, err := r.db.Query(ctx, `
SELECT id, driver_id, truck_id, status
FROM operations.intermodal_dispatch_assignments
WHERE dispatch_id = $1`, pgID(id))
	if err != nil {
		return nil, fmt.Errorf("operations: load assignments: %w", err)
	}
	defer rows.Close()

	var out []intermodal_old.AssignmentSnapshot
	for rows.Next() {
		var (
			assignmentID, driver, truck pgtype.UUID
			status                      string
		)
		if err := rows.Scan(&assignmentID, &driver, &truck, &status); err != nil {
			return nil, fmt.Errorf("operations: scan assignment: %w", err)
		}
		out = append(out, intermodal_old.AssignmentSnapshot{
			ID:       mustAssignmentID(assignmentID),
			DriverID: optDriver(driver),
			TruckID:  optTruck(truck),
			Status:   intermodal_old.AssignmentStatus(status),
		})
	}
	return out, rows.Err()
}

func (r *DispatchRepository) loadChassisEpisodes(ctx context.Context, id intermodal_old.DispatchID) ([]intermodal_old.ChassisEpisodeSnapshot, error) {
	rows, err := r.db.Query(ctx, `
SELECT id, status, sourced_from, stationed_at, chassis_number
FROM operations.intermodal_dispatch_chassis_custody_episodes
WHERE dispatch_id = $1`, pgID(id))
	if err != nil {
		return nil, fmt.Errorf("operations: load chassis episodes: %w", err)
	}
	defer rows.Close()

	var out []intermodal_old.ChassisEpisodeSnapshot
	for rows.Next() {
		var (
			episodeID, sourcedFrom, stationedAt pgtype.UUID
			status                              string
			chassisNumber                       pgtype.Text
		)
		if err := rows.Scan(&episodeID, &status, &sourcedFrom, &stationedAt, &chassisNumber); err != nil {
			return nil, fmt.Errorf("operations: scan chassis episode: %w", err)
		}
		chassis, err := optChassis(chassisNumber)
		if err != nil {
			return nil, err
		}
		out = append(out, intermodal_old.ChassisEpisodeSnapshot{
			ID:            mustChassisEpisodeID(episodeID),
			Status:        intermodal_old.CustodyEpisodeStatus(status),
			SourcedFrom:   mustFacilityID(sourcedFrom),
			StationedAt:   optFacility(stationedAt),
			ChassisNumber: chassis,
		})
	}
	return out, rows.Err()
}

func (r *DispatchRepository) loadContainerEpisodes(ctx context.Context, id intermodal_old.DispatchID) ([]intermodal_old.ContainerEpisodeSnapshot, error) {
	rows, err := r.db.Query(ctx, `
SELECT id, status, sourced_from, stationed_at, container_number
FROM operations.intermodal_dispatch_container_custody_episodes
WHERE dispatch_id = $1`, pgID(id))
	if err != nil {
		return nil, fmt.Errorf("operations: load container episodes: %w", err)
	}
	defer rows.Close()

	var out []intermodal_old.ContainerEpisodeSnapshot
	for rows.Next() {
		var (
			episodeID, sourcedFrom, stationedAt pgtype.UUID
			status, containerNumber             string
		)
		if err := rows.Scan(&episodeID, &status, &sourcedFrom, &stationedAt, &containerNumber); err != nil {
			return nil, fmt.Errorf("operations: scan container episode: %w", err)
		}
		container, err := intermodal_old.NewContainerNumber(containerNumber)
		if err != nil {
			return nil, fmt.Errorf("operations: decode container number: %w", err)
		}
		out = append(out, intermodal_old.ContainerEpisodeSnapshot{
			ID:              mustContainerEpisodeID(episodeID),
			Status:          intermodal_old.CustodyEpisodeStatus(status),
			SourcedFrom:     mustFacilityID(sourcedFrom),
			StationedAt:     optFacility(stationedAt),
			ContainerNumber: container,
		})
	}
	return out, rows.Err()
}
