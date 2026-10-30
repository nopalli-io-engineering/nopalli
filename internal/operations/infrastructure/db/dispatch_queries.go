package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	coredb "github.com/nopalli-io/nopalli/internal/core/infrastructure/db"
	"github.com/nopalli-io/nopalli/internal/operations/application"
	"github.com/nopalli-io/nopalli/internal/operations/domain/intermodal"
)

const (
	defaultListLimit = 50
	maxListLimit     = 200
)

// DispatchQueryService is the read-side adapter: flat SELECTs over
// operations.intermodal_dispatches shaped straight into read-model DTOs (rdt:
// SqlAlchemyIntermodalDispatchQueries). Child collections are never part of
// these views. Reads need no transaction, so bind it to the pool.
type DispatchQueryService struct {
	db coredb.DBTX
}

// NewDispatchQueryService builds the read adapter over a DBTX (typically the pool).
func NewDispatchQueryService(db coredb.DBTX) *DispatchQueryService {
	return &DispatchQueryService{db: db}
}

// ListDispatches returns in-scope summaries, most recent first.
func (q *DispatchQueryService) ListDispatches(
	ctx context.Context, vis application.Visibility, filter application.ListFilter,
) ([]application.DispatchSummary, error) {
	if !vis.All {
		return nil, nil // out of scope: empty, without touching the DB
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	// $3 (status) is optional: the predicate is skipped when it is NULL.
	var statusArg any
	if filter.Status != nil {
		statusArg = string(*filter.Status)
	}
	rows, err := q.db.Query(ctx, `
SELECT id, source_quote_id, plan_type, status, created_at, updated_at
FROM operations.intermodal_dispatches
WHERE ($3::text IS NULL OR status = $3::text)
ORDER BY created_at DESC
LIMIT $1 OFFSET $2`, limit, offset, statusArg)
	if err != nil {
		return nil, fmt.Errorf("operations: list dispatches: %w", err)
	}
	defer rows.Close()

	var out []application.DispatchSummary
	for rows.Next() {
		var (
			id, sourceQuote      pgtype.UUID
			planType, status     string
			createdAt, updatedAt pgtype.Timestamptz
		)
		if err := rows.Scan(&id, &sourceQuote, &planType, &status, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("operations: scan dispatch summary: %w", err)
		}
		out = append(out, application.DispatchSummary{
			ID:            uuid.UUID(id.Bytes),
			SourceQuoteID: optUUID(sourceQuote),
			PlanType:      intermodal.PlanType(planType),
			Status:        intermodal.Status(status),
			CreatedAt:     createdAt.Time,
			UpdatedAt:     updatedAt.Time,
		})
	}
	return out, rows.Err()
}

// GetDispatch returns one dispatch's detail, or nil when absent or out of scope.
func (q *DispatchQueryService) GetDispatch(
	ctx context.Context, id uuid.UUID, vis application.Visibility,
) (*application.DispatchDetail, error) {
	if !vis.All {
		return nil, nil // out of scope reads as not-found
	}

	var (
		rowID, sourceQuote, dispatcher pgtype.UUID
		planType, status               string
		createdAt, updatedAt           pgtype.Timestamptz
	)
	err := q.db.QueryRow(ctx, `
SELECT id, source_quote_id, dispatcher_id, plan_type, status, created_at, updated_at
FROM operations.intermodal_dispatches
WHERE id = $1`, pgtype.UUID{Bytes: id, Valid: true}).
		Scan(&rowID, &sourceQuote, &dispatcher, &planType, &status, &createdAt, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("operations: get dispatch: %w", err)
	}
	detail := &application.DispatchDetail{
		ID:            uuid.UUID(rowID.Bytes),
		SourceQuoteID: optUUID(sourceQuote),
		DispatcherID:  optUUID(dispatcher),
		PlanType:      intermodal.PlanType(planType),
		Status:        intermodal.Status(status),
		CreatedAt:     createdAt.Time,
		UpdatedAt:     updatedAt.Time,
	}

	var loadErr error
	if detail.AgendaItems, loadErr = q.agendaItemViews(ctx, id); loadErr != nil {
		return nil, loadErr
	}
	if detail.Assignments, loadErr = q.assignmentViews(ctx, id); loadErr != nil {
		return nil, loadErr
	}
	if detail.ChassisEpisodes, loadErr = q.chassisEpisodeViews(ctx, id); loadErr != nil {
		return nil, loadErr
	}
	if detail.ContainerEpisodes, loadErr = q.containerEpisodeViews(ctx, id); loadErr != nil {
		return nil, loadErr
	}
	return detail, nil
}

// --- child projections ---------------------------------------------------- //

func (q *DispatchQueryService) agendaItemViews(ctx context.Context, id uuid.UUID) ([]application.AgendaItemView, error) {
	rows, err := q.db.Query(ctx, `
SELECT id, item_no, action, facility_id, status, attributed_driver, attributed_truck, created_at, updated_at
FROM operations.intermodal_dispatch_agenda_items
WHERE dispatch_id = $1
ORDER BY item_no`, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("operations: list agenda items: %w", err)
	}
	defer rows.Close()

	var out []application.AgendaItemView
	for rows.Next() {
		var (
			itemID, facility, driver, truck pgtype.UUID
			itemNo                          int32
			action, status                  string
			createdAt, updatedAt            pgtype.Timestamptz
		)
		if err := rows.Scan(&itemID, &itemNo, &action, &facility, &status, &driver, &truck, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("operations: scan agenda item view: %w", err)
		}
		out = append(out, application.AgendaItemView{
			ID:               uuid.UUID(itemID.Bytes),
			ItemNo:           int(itemNo),
			Action:           intermodal.Action(action),
			FacilityID:       optUUID(facility),
			Status:           intermodal.AgendaItemStatus(status),
			AttributedDriver: optUUID(driver),
			AttributedTruck:  optUUID(truck),
			CreatedAt:        createdAt.Time,
			UpdatedAt:        updatedAt.Time,
		})
	}
	return out, rows.Err()
}

func (q *DispatchQueryService) assignmentViews(ctx context.Context, id uuid.UUID) ([]application.AssignmentView, error) {
	rows, err := q.db.Query(ctx, `
SELECT id, driver_id, truck_id, status, created_at, updated_at
FROM operations.intermodal_dispatch_assignments
WHERE dispatch_id = $1
ORDER BY created_at`, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("operations: list assignments: %w", err)
	}
	defer rows.Close()

	var out []application.AssignmentView
	for rows.Next() {
		var (
			assignmentID, driver, truck pgtype.UUID
			status                      string
			createdAt, updatedAt        pgtype.Timestamptz
		)
		if err := rows.Scan(&assignmentID, &driver, &truck, &status, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("operations: scan assignment view: %w", err)
		}
		out = append(out, application.AssignmentView{
			ID:        uuid.UUID(assignmentID.Bytes),
			DriverID:  optUUID(driver),
			TruckID:   optUUID(truck),
			Status:    intermodal.AssignmentStatus(status),
			CreatedAt: createdAt.Time,
			UpdatedAt: updatedAt.Time,
		})
	}
	return out, rows.Err()
}

func (q *DispatchQueryService) chassisEpisodeViews(ctx context.Context, id uuid.UUID) ([]application.ChassisEpisodeView, error) {
	rows, err := q.db.Query(ctx, `
SELECT id, status, sourced_from, stationed_at, chassis_number, created_at, updated_at
FROM operations.intermodal_dispatch_chassis_custody_episodes
WHERE dispatch_id = $1
ORDER BY created_at`, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("operations: list chassis episodes: %w", err)
	}
	defer rows.Close()

	var out []application.ChassisEpisodeView
	for rows.Next() {
		var (
			episodeID, sourcedFrom, stationedAt pgtype.UUID
			status                              string
			chassisNumber                       pgtype.Text
			createdAt, updatedAt                pgtype.Timestamptz
		)
		if err := rows.Scan(&episodeID, &status, &sourcedFrom, &stationedAt, &chassisNumber, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("operations: scan chassis episode view: %w", err)
		}
		out = append(out, application.ChassisEpisodeView{
			ID:            uuid.UUID(episodeID.Bytes),
			Status:        intermodal.CustodyEpisodeStatus(status),
			SourcedFrom:   uuid.UUID(sourcedFrom.Bytes),
			StationedAt:   optUUID(stationedAt),
			ChassisNumber: optText(chassisNumber),
			CreatedAt:     createdAt.Time,
			UpdatedAt:     updatedAt.Time,
		})
	}
	return out, rows.Err()
}

func (q *DispatchQueryService) containerEpisodeViews(ctx context.Context, id uuid.UUID) ([]application.ContainerEpisodeView, error) {
	rows, err := q.db.Query(ctx, `
SELECT id, status, sourced_from, stationed_at, container_number, created_at, updated_at
FROM operations.intermodal_dispatch_container_custody_episodes
WHERE dispatch_id = $1
ORDER BY created_at`, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("operations: list container episodes: %w", err)
	}
	defer rows.Close()

	var out []application.ContainerEpisodeView
	for rows.Next() {
		var (
			episodeID, sourcedFrom, stationedAt pgtype.UUID
			status, containerNumber             string
			createdAt, updatedAt                pgtype.Timestamptz
		)
		if err := rows.Scan(&episodeID, &status, &sourcedFrom, &stationedAt, &containerNumber, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("operations: scan container episode view: %w", err)
		}
		out = append(out, application.ContainerEpisodeView{
			ID:              uuid.UUID(episodeID.Bytes),
			Status:          intermodal.CustodyEpisodeStatus(status),
			SourcedFrom:     uuid.UUID(sourcedFrom.Bytes),
			StationedAt:     optUUID(stationedAt),
			ContainerNumber: containerNumber,
			CreatedAt:       createdAt.Time,
			UpdatedAt:       updatedAt.Time,
		})
	}
	return out, rows.Err()
}

// optUUID maps a scanned nullable uuid to a *uuid.UUID for the read model.
func optUUID(p pgtype.UUID) *uuid.UUID {
	if !p.Valid {
		return nil
	}
	v := uuid.UUID(p.Bytes)
	return &v
}

// optText maps a scanned nullable text to a *string for the read model.
func optText(p pgtype.Text) *string {
	if !p.Valid {
		return nil
	}
	v := p.String
	return &v
}

// Compile-time check that the adapter satisfies the read port.
var _ application.DispatchQueries = (*DispatchQueryService)(nil)
