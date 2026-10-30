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
	"github.com/nopalli-io/nopalli/internal/operations/domain/intermodal"
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
SELECT source_quote_id, dispatcher_id, current_assignment_id, plan_type, status, item_cursor, version
FROM operations.intermodal_dispatches
WHERE id = $1`

// Get loads a fully-populated Dispatch by id (rdt: repository.get + selectin
// eager load). It returns operations' DispatchNotFound when absent.
func (r *DispatchRepository) Get(ctx context.Context, id intermodal.DispatchID) (*intermodal.Dispatch, error) {
	var (
		sourceQuote, dispatcher, currentAssignment pgtype.UUID
		planType, status                           string
		itemCursor                                 pgtype.Int4
		version                                    int32
	)
	err := r.db.QueryRow(ctx, selectRoot, pgID(id)).
		Scan(&sourceQuote, &dispatcher, &currentAssignment, &planType, &status, &itemCursor, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, intermodal.NewDispatchNotFound(id)
	}
	if err != nil {
		return nil, fmt.Errorf("operations: load dispatch: %w", err)
	}

	snap := intermodal.DispatchSnapshot{
		ID:                  id,
		SourceQuoteID:       optSourceQuote(sourceQuote),
		DispatcherID:        optDispatcher(dispatcher),
		CurrentAssignmentID: optAssignment(currentAssignment),
		PlanType:            intermodal.PlanType(planType),
		Status:              intermodal.Status(status),
		Cursor:              cursorFromDB(itemCursor),
		Version:             int(version),
	}

	var loadErr error
	if snap.AgendaItems, loadErr = r.loadAgendaItems(ctx, id); loadErr != nil {
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

	return intermodal.Reconstitute(snap), nil
}

const insertRoot = `
INSERT INTO operations.intermodal_dispatches
    (id, source_quote_id, plan_type, status, item_cursor, current_assignment_id, dispatcher_id, version)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

// Add inserts a newly-queued dispatch and its children (rdt: repository.add).
// Must run inside a UnitOfWork transaction.
func (r *DispatchRepository) Add(ctx context.Context, d *intermodal.Dispatch) error {
	s := d.Snapshot()
	_, err := r.db.Exec(ctx, insertRoot,
		pgID(s.ID), pgIDOpt(s.SourceQuoteID), string(s.PlanType), string(s.Status),
		cursorToDB(s.Cursor), pgIDOpt(s.CurrentAssignmentID), pgIDOpt(s.DispatcherID), int32(s.Version),
	)
	if err != nil {
		return fmt.Errorf("operations: insert dispatch: %w", err)
	}
	if err := r.insertChildren(ctx, s); err != nil {
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
SET source_quote_id = $1, plan_type = $2, status = $3, item_cursor = $4,
    current_assignment_id = $5, dispatcher_id = $6, version = $7, updated_at = now()
WHERE id = $8 AND version = $9`

// Save persists changes to an already-stored dispatch under optimistic
// concurrency, then replaces its child rows. Must run inside a UnitOfWork
// transaction. Returns ErrConcurrencyConflict if the row moved on since load.
//
// Child rows are deleted and re-inserted from the snapshot each save — the
// simplest correct reconciliation; optimize to a diff only if it ever matters.
func (r *DispatchRepository) Save(ctx context.Context, d *intermodal.Dispatch) error {
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
	if err := r.deleteChildren(ctx, s.ID); err != nil {
		return err
	}
	if err := r.insertChildren(ctx, s); err != nil {
		return err
	}
	if err := r.events.Record(ctx, r.db, d.PullEvents()); err != nil {
		return err
	}
	d.MarkPersisted(newVersion)
	return nil
}

// --- child writes --------------------------------------------------------- //

func (r *DispatchRepository) insertChildren(ctx context.Context, s intermodal.DispatchSnapshot) error {
	for _, i := range s.AgendaItems {
		details, err := encodeDetails(i.Container)
		if err != nil {
			return fmt.Errorf("operations: encode agenda details: %w", err)
		}
		_, err = r.db.Exec(ctx, `
INSERT INTO operations.intermodal_dispatch_agenda_items
    (id, dispatch_id, item_no, action, facility_id, status, details, attributed_driver, attributed_truck)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			pgID(i.ID), pgID(s.ID), int32(i.ItemNo), string(i.Action), pgIDOpt(i.FacilityID),
			string(i.Status), details, pgIDOpt(i.AttributedDriver), pgIDOpt(i.AttributedTruck),
		)
		if err != nil {
			return fmt.Errorf("operations: insert agenda item: %w", err)
		}
	}
	for _, a := range s.Assignments {
		_, err := r.db.Exec(ctx, `
INSERT INTO operations.intermodal_dispatch_assignments (id, dispatch_id, driver_id, truck_id, status)
VALUES ($1, $2, $3, $4, $5)`,
			pgID(a.ID), pgID(s.ID), pgIDOpt(a.DriverID), pgIDOpt(a.TruckID), string(a.Status),
		)
		if err != nil {
			return fmt.Errorf("operations: insert assignment: %w", err)
		}
	}
	for _, e := range s.ChassisEpisodes {
		_, err := r.db.Exec(ctx, `
INSERT INTO operations.intermodal_dispatch_chassis_custody_episodes
    (id, dispatch_id, status, close, sourced_from, stationed_at, chassis_number)
VALUES ($1, $2, $3, NULL, $4, $5, $6)`,
			pgID(e.ID), pgID(s.ID), string(e.Status), pgID(e.SourcedFrom), pgIDOpt(e.StationedAt),
			chassisText(e.ChassisNumber),
		)
		if err != nil {
			return fmt.Errorf("operations: insert chassis episode: %w", err)
		}
	}
	for _, e := range s.ContainerEpisodes {
		_, err := r.db.Exec(ctx, `
INSERT INTO operations.intermodal_dispatch_container_custody_episodes
    (id, dispatch_id, status, close, sourced_from, stationed_at, container_number)
VALUES ($1, $2, $3, NULL, $4, $5, $6)`,
			pgID(e.ID), pgID(s.ID), string(e.Status), pgID(e.SourcedFrom), pgIDOpt(e.StationedAt),
			e.ContainerNumber.String(),
		)
		if err != nil {
			return fmt.Errorf("operations: insert container episode: %w", err)
		}
	}
	return nil
}

func (r *DispatchRepository) deleteChildren(ctx context.Context, id intermodal.DispatchID) error {
	for _, table := range []string{
		"operations.intermodal_dispatch_agenda_items",
		"operations.intermodal_dispatch_assignments",
		"operations.intermodal_dispatch_chassis_custody_episodes",
		"operations.intermodal_dispatch_container_custody_episodes",
	} {
		if _, err := r.db.Exec(ctx, "DELETE FROM "+table+" WHERE dispatch_id = $1", pgID(id)); err != nil {
			return fmt.Errorf("operations: delete children (%s): %w", table, err)
		}
	}
	return nil
}

// --- child reads ---------------------------------------------------------- //

func (r *DispatchRepository) loadAgendaItems(ctx context.Context, id intermodal.DispatchID) ([]intermodal.AgendaItemSnapshot, error) {
	rows, err := r.db.Query(ctx, `
SELECT id, item_no, action, facility_id, status, details, attributed_driver, attributed_truck
FROM operations.intermodal_dispatch_agenda_items
WHERE dispatch_id = $1
ORDER BY item_no`, pgID(id))
	if err != nil {
		return nil, fmt.Errorf("operations: load agenda items: %w", err)
	}
	defer rows.Close()

	var out []intermodal.AgendaItemSnapshot
	for rows.Next() {
		var (
			itemID, facility, driver, truck pgtype.UUID
			itemNo                          int32
			action, status                  string
			details                         []byte
		)
		if err := rows.Scan(&itemID, &itemNo, &action, &facility, &status, &details, &driver, &truck); err != nil {
			return nil, fmt.Errorf("operations: scan agenda item: %w", err)
		}
		container, err := decodeContainerFromDetails(details)
		if err != nil {
			return nil, err
		}
		out = append(out, intermodal.AgendaItemSnapshot{
			ID:               mustAgendaItemID(itemID),
			ItemNo:           int(itemNo),
			Action:           intermodal.Action(action),
			FacilityID:       optFacility(facility),
			Status:           intermodal.AgendaItemStatus(status),
			Container:        container,
			AttributedDriver: optDriver(driver),
			AttributedTruck:  optTruck(truck),
		})
	}
	return out, rows.Err()
}

func (r *DispatchRepository) loadAssignments(ctx context.Context, id intermodal.DispatchID) ([]intermodal.AssignmentSnapshot, error) {
	rows, err := r.db.Query(ctx, `
SELECT id, driver_id, truck_id, status
FROM operations.intermodal_dispatch_assignments
WHERE dispatch_id = $1`, pgID(id))
	if err != nil {
		return nil, fmt.Errorf("operations: load assignments: %w", err)
	}
	defer rows.Close()

	var out []intermodal.AssignmentSnapshot
	for rows.Next() {
		var (
			assignmentID, driver, truck pgtype.UUID
			status                      string
		)
		if err := rows.Scan(&assignmentID, &driver, &truck, &status); err != nil {
			return nil, fmt.Errorf("operations: scan assignment: %w", err)
		}
		out = append(out, intermodal.AssignmentSnapshot{
			ID:       mustAssignmentID(assignmentID),
			DriverID: optDriver(driver),
			TruckID:  optTruck(truck),
			Status:   intermodal.AssignmentStatus(status),
		})
	}
	return out, rows.Err()
}

func (r *DispatchRepository) loadChassisEpisodes(ctx context.Context, id intermodal.DispatchID) ([]intermodal.ChassisEpisodeSnapshot, error) {
	rows, err := r.db.Query(ctx, `
SELECT id, status, sourced_from, stationed_at, chassis_number
FROM operations.intermodal_dispatch_chassis_custody_episodes
WHERE dispatch_id = $1`, pgID(id))
	if err != nil {
		return nil, fmt.Errorf("operations: load chassis episodes: %w", err)
	}
	defer rows.Close()

	var out []intermodal.ChassisEpisodeSnapshot
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
		out = append(out, intermodal.ChassisEpisodeSnapshot{
			ID:            mustChassisEpisodeID(episodeID),
			Status:        intermodal.CustodyEpisodeStatus(status),
			SourcedFrom:   mustFacilityID(sourcedFrom),
			StationedAt:   optFacility(stationedAt),
			ChassisNumber: chassis,
		})
	}
	return out, rows.Err()
}

func (r *DispatchRepository) loadContainerEpisodes(ctx context.Context, id intermodal.DispatchID) ([]intermodal.ContainerEpisodeSnapshot, error) {
	rows, err := r.db.Query(ctx, `
SELECT id, status, sourced_from, stationed_at, container_number
FROM operations.intermodal_dispatch_container_custody_episodes
WHERE dispatch_id = $1`, pgID(id))
	if err != nil {
		return nil, fmt.Errorf("operations: load container episodes: %w", err)
	}
	defer rows.Close()

	var out []intermodal.ContainerEpisodeSnapshot
	for rows.Next() {
		var (
			episodeID, sourcedFrom, stationedAt pgtype.UUID
			status, containerNumber             string
		)
		if err := rows.Scan(&episodeID, &status, &sourcedFrom, &stationedAt, &containerNumber); err != nil {
			return nil, fmt.Errorf("operations: scan container episode: %w", err)
		}
		container, err := intermodal.NewContainerNumber(containerNumber)
		if err != nil {
			return nil, fmt.Errorf("operations: decode container number: %w", err)
		}
		out = append(out, intermodal.ContainerEpisodeSnapshot{
			ID:              mustContainerEpisodeID(episodeID),
			Status:          intermodal.CustodyEpisodeStatus(status),
			SourcedFrom:     mustFacilityID(sourcedFrom),
			StationedAt:     optFacility(stationedAt),
			ContainerNumber: container,
		})
	}
	return out, rows.Err()
}
