package rest

import (
	"time"

	"github.com/google/uuid"

	"github.com/nopalli-io/nopalli/internal/operations/application"
	"github.com/nopalli-io/nopalli/internal/operations/domain/intermodal_old"
)

// Wire contracts for the operations command API. Ids are raw UUID strings on
// the wire; the handlers parse them into typed ids at the boundary. The domain
// PlanType enum is reused as a field type — its values are the wire format.

// --- requests ------------------------------------------------------------ //

// CreateDispatchRequest creates a dispatch from scratch (POST /dispatches). The
// handler assembles the matching LiveLoad/LiveUnload from PlanType + these
// fields; AD_HOC is rejected (no standard tasks to build).
type CreateDispatchRequest struct {
	PlanType             intermodal_old.PlanType `json:"plan_type"`
	ContainerNumber      string                  `json:"container_number"`
	ContainerPickupRamp  string                  `json:"container_pickup_ramp"`
	ContainerDropoffRamp string                  `json:"container_dropoff_ramp"`
	DockDoor             string                  `json:"dock_door"`
	Premounted           bool                    `json:"premounted"`
	SourceQuoteID        *string                 `json:"source_quote_id,omitempty"`
}

// DelegateRequest assigns a dispatcher.
type DelegateRequest struct {
	DispatcherID string `json:"dispatcher_id"`
}

// AssignRequest proposes a driver and/or truck. Either may be omitted.
type AssignRequest struct {
	DriverID *string `json:"driver_id,omitempty"`
	TruckID  *string `json:"truck_id,omitempty"`
}

// AcquireChassisRequest carries the facility and chassis for acquisition.
type AcquireChassisRequest struct {
	FacilityID    string `json:"facility_id"`
	ChassisNumber string `json:"chassis_number"`
}

// PickUpPremountedContainerRequest carries the chassis the container is on.
type PickUpPremountedContainerRequest struct {
	ChassisNumber string `json:"chassis_number"`
}

// ReturnChassisRequest carries the facility the chassis is returned to.
type ReturnChassisRequest struct {
	FacilityID string `json:"facility_id"`
}

// --- responses ----------------------------------------------------------- //

// CreatedResponse is the body for a resource-creating command (dispatch or
// assignment): the new id as a raw UUID string.
type CreatedResponse struct {
	ID string `json:"id"`
}

// DispatchSummaryResponse is the list-row view of a dispatch. Ids are raw UUID
// strings; nullable ids are omitted when absent. Timestamps are RFC 3339.
type DispatchSummaryResponse struct {
	ID            string                  `json:"id"`
	SourceQuoteID *string                 `json:"source_quote_id,omitempty"`
	PlanType      intermodal_old.PlanType `json:"plan_type"`
	Status        intermodal_old.Status   `json:"status"`
	CreatedAt     time.Time               `json:"created_at"`
	UpdatedAt     time.Time               `json:"updated_at"`
}

// DispatchDetailResponse is the single-dispatch view: the summary fields, the
// dispatcher (null until delegated), and the owned child collections. The child
// arrays are always present (empty, not null) so clients can iterate them
// unconditionally.
type DispatchDetailResponse struct {
	ID                string                     `json:"id"`
	SourceQuoteID     *string                    `json:"source_quote_id,omitempty"`
	DispatcherID      *string                    `json:"dispatcher_id,omitempty"`
	PlanType          intermodal_old.PlanType    `json:"plan_type"`
	Status            intermodal_old.Status      `json:"status"`
	CreatedAt         time.Time                  `json:"created_at"`
	UpdatedAt         time.Time                  `json:"updated_at"`
	Tasks             []TaskResponse             `json:"tasks"`
	Assignments       []AssignmentResponse       `json:"assignments"`
	ChassisEpisodes   []ChassisEpisodeResponse   `json:"chassis_episodes"`
	ContainerEpisodes []ContainerEpisodeResponse `json:"container_episodes"`
}

// TaskResponse is the wire view of one task row.
type TaskResponse struct {
	ID               string                    `json:"id"`
	No               int                       `json:"no"`
	Action           intermodal_old.Action     `json:"action"`
	FacilityID       *string                   `json:"facility_id,omitempty"`
	Status           intermodal_old.TaskStatus `json:"status"`
	AttributedDriver *string                   `json:"attributed_driver,omitempty"`
	AttributedTruck  *string                   `json:"attributed_truck,omitempty"`
	CreatedAt        time.Time                 `json:"created_at"`
	UpdatedAt        time.Time                 `json:"updated_at"`
}

// AssignmentResponse is the wire view of one assignment row.
type AssignmentResponse struct {
	ID        string                          `json:"id"`
	DriverID  *string                         `json:"driver_id,omitempty"`
	TruckID   *string                         `json:"truck_id,omitempty"`
	Status    intermodal_old.AssignmentStatus `json:"status"`
	CreatedAt time.Time                       `json:"created_at"`
	UpdatedAt time.Time                       `json:"updated_at"`
}

// ChassisEpisodeResponse is the wire view of one chassis custody row.
type ChassisEpisodeResponse struct {
	ID            string                              `json:"id"`
	Status        intermodal_old.CustodyEpisodeStatus `json:"status"`
	SourcedFrom   string                              `json:"sourced_from"`
	StationedAt   *string                             `json:"stationed_at,omitempty"`
	ChassisNumber *string                             `json:"chassis_number,omitempty"`
	CreatedAt     time.Time                           `json:"created_at"`
	UpdatedAt     time.Time                           `json:"updated_at"`
}

// ContainerEpisodeResponse is the wire view of one container custody row.
type ContainerEpisodeResponse struct {
	ID              string                              `json:"id"`
	Status          intermodal_old.CustodyEpisodeStatus `json:"status"`
	SourcedFrom     string                              `json:"sourced_from"`
	StationedAt     *string                             `json:"stationed_at,omitempty"`
	ContainerNumber string                              `json:"container_number"`
	CreatedAt       time.Time                           `json:"created_at"`
	UpdatedAt       time.Time                           `json:"updated_at"`
}

// newDispatchSummaryResponse maps a read-model summary to its wire view.
func newDispatchSummaryResponse(s application.DispatchSummary) DispatchSummaryResponse {
	return DispatchSummaryResponse{
		ID:            s.ID.String(),
		SourceQuoteID: optUUIDString(s.SourceQuoteID),
		PlanType:      s.PlanType,
		Status:        s.Status,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}

// newDispatchDetailResponse maps a read-model detail to its wire view, with the
// child collections rendered as present-but-possibly-empty arrays.
func newDispatchDetailResponse(d application.DispatchDetail) DispatchDetailResponse {
	resp := DispatchDetailResponse{
		ID:                d.ID.String(),
		SourceQuoteID:     optUUIDString(d.SourceQuoteID),
		DispatcherID:      optUUIDString(d.DispatcherID),
		PlanType:          d.PlanType,
		Status:            d.Status,
		CreatedAt:         d.CreatedAt,
		UpdatedAt:         d.UpdatedAt,
		Tasks:             make([]TaskResponse, 0, len(d.Tasks)),
		Assignments:       make([]AssignmentResponse, 0, len(d.Assignments)),
		ChassisEpisodes:   make([]ChassisEpisodeResponse, 0, len(d.ChassisEpisodes)),
		ContainerEpisodes: make([]ContainerEpisodeResponse, 0, len(d.ContainerEpisodes)),
	}
	for _, i := range d.Tasks {
		resp.Tasks = append(resp.Tasks, TaskResponse{
			ID:               i.ID.String(),
			No:               i.No,
			Action:           i.Action,
			FacilityID:       optUUIDString(i.FacilityID),
			Status:           i.Status,
			AttributedDriver: optUUIDString(i.AttributedDriver),
			AttributedTruck:  optUUIDString(i.AttributedTruck),
			CreatedAt:        i.CreatedAt,
			UpdatedAt:        i.UpdatedAt,
		})
	}
	for _, a := range d.Assignments {
		resp.Assignments = append(resp.Assignments, AssignmentResponse{
			ID:        a.ID.String(),
			DriverID:  optUUIDString(a.DriverID),
			TruckID:   optUUIDString(a.TruckID),
			Status:    a.Status,
			CreatedAt: a.CreatedAt,
			UpdatedAt: a.UpdatedAt,
		})
	}
	for _, e := range d.ChassisEpisodes {
		resp.ChassisEpisodes = append(resp.ChassisEpisodes, ChassisEpisodeResponse{
			ID:            e.ID.String(),
			Status:        e.Status,
			SourcedFrom:   e.SourcedFrom.String(),
			StationedAt:   optUUIDString(e.StationedAt),
			ChassisNumber: e.ChassisNumber,
			CreatedAt:     e.CreatedAt,
			UpdatedAt:     e.UpdatedAt,
		})
	}
	for _, e := range d.ContainerEpisodes {
		resp.ContainerEpisodes = append(resp.ContainerEpisodes, ContainerEpisodeResponse{
			ID:              e.ID.String(),
			Status:          e.Status,
			SourcedFrom:     e.SourcedFrom.String(),
			StationedAt:     optUUIDString(e.StationedAt),
			ContainerNumber: e.ContainerNumber,
			CreatedAt:       e.CreatedAt,
			UpdatedAt:       e.UpdatedAt,
		})
	}
	return resp
}

// optUUIDString renders a nullable id as a *string for the wire (nil stays nil,
// so the field is omitted).
func optUUIDString(p *uuid.UUID) *string {
	if p == nil {
		return nil
	}
	s := p.String()
	return &s
}
