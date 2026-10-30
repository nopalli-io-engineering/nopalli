package rest

import (
	"time"

	"github.com/google/uuid"

	"github.com/nopalli-io/nopalli/internal/operations/application"
	"github.com/nopalli-io/nopalli/internal/operations/domain/intermodal"
)

// Wire contracts for the operations command API. Ids are raw UUID strings on
// the wire; the handlers parse them into typed ids at the boundary. The domain
// PlanType enum is reused as a field type — its values are the wire format.

// --- requests ------------------------------------------------------------ //

// CreateDispatchRequest creates a dispatch from scratch (POST /dispatches). The
// handler assembles the matching LiveLoad/LiveUnload from PlanType + these
// fields; AD_HOC is rejected (no standard agenda to build).
type CreateDispatchRequest struct {
	PlanType             intermodal.PlanType `json:"plan_type"`
	ContainerNumber      string              `json:"container_number"`
	ContainerPickupRamp  string              `json:"container_pickup_ramp"`
	ContainerDropoffRamp string              `json:"container_dropoff_ramp"`
	DockDoor             string              `json:"dock_door"`
	Premounted           bool                `json:"premounted"`
	SourceQuoteID        *string             `json:"source_quote_id,omitempty"`
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
	ID            string              `json:"id"`
	SourceQuoteID *string             `json:"source_quote_id,omitempty"`
	PlanType      intermodal.PlanType `json:"plan_type"`
	Status        intermodal.Status   `json:"status"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
}

// DispatchDetailResponse is the single-dispatch view: the summary plus the
// dispatcher (null until the dispatch is delegated).
type DispatchDetailResponse struct {
	ID            string              `json:"id"`
	SourceQuoteID *string             `json:"source_quote_id,omitempty"`
	DispatcherID  *string             `json:"dispatcher_id,omitempty"`
	PlanType      intermodal.PlanType `json:"plan_type"`
	Status        intermodal.Status   `json:"status"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
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

// newDispatchDetailResponse maps a read-model detail to its wire view.
func newDispatchDetailResponse(d application.DispatchDetail) DispatchDetailResponse {
	return DispatchDetailResponse{
		ID:            d.ID.String(),
		SourceQuoteID: optUUIDString(d.SourceQuoteID),
		DispatcherID:  optUUIDString(d.DispatcherID),
		PlanType:      d.PlanType,
		Status:        d.Status,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}
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
