// Package rest is the operations context's HTTP command API over the
// application service (rdt: interfaces/api/v1/operations). Named "rest" rather
// than "http" to avoid shadowing the stdlib net/http import. Each write is a
// state transition modeled as a POST to a sub-resource; ids are raw UUID
// strings on the wire, parsed into typed ids at the boundary.
package rest

import (
	"context"
	"net/http"

	"github.com/nopalli-io/nopalli/internal/core/interface/httpx"
	"github.com/nopalli-io/nopalli/internal/operations/application"
	"github.com/nopalli-io/nopalli/internal/operations/domain/intermodal"
)

// CommandService is the slice of the application service the command handlers
// need. Defined here (the consumer) so handlers can be tested with a fake;
// *application.IntermodalDispatchService satisfies it.
type CommandService interface {
	CreateDispatch(ctx context.Context, move intermodal.MovePlan, sourceQuoteID *intermodal.SourceQuoteID) (intermodal.DispatchID, error)
	Delegate(ctx context.Context, dispatchID intermodal.DispatchID, dispatcherID intermodal.DispatcherID) error
	Assign(ctx context.Context, dispatchID intermodal.DispatchID, driverID *intermodal.DriverID, truckID *intermodal.TruckID) (intermodal.AssignmentID, error)
	Approve(ctx context.Context, dispatchID intermodal.DispatchID) error
	Start(ctx context.Context, dispatchID intermodal.DispatchID) error
	Verify(ctx context.Context, dispatchID intermodal.DispatchID) error
	AcquireChassis(ctx context.Context, dispatchID intermodal.DispatchID, facilityID intermodal.FacilityID, chassisNumber intermodal.ChassisNumber) error
	PickUpContainer(ctx context.Context, dispatchID intermodal.DispatchID) error
	PickUpPremountedContainer(ctx context.Context, dispatchID intermodal.DispatchID, chassisNumber intermodal.ChassisNumber) error
	DeliverContainer(ctx context.Context, dispatchID intermodal.DispatchID) error
	Hold(ctx context.Context, dispatchID intermodal.DispatchID) error
	TakeContainer(ctx context.Context, dispatchID intermodal.DispatchID) error
	DropContainer(ctx context.Context, dispatchID intermodal.DispatchID) error
	DropPremountedContainer(ctx context.Context, dispatchID intermodal.DispatchID) error
	ReturnChassis(ctx context.Context, dispatchID intermodal.DispatchID, facilityID intermodal.FacilityID) error
}

var _ CommandService = (*application.IntermodalDispatchService)(nil)

// Handlers serves the operations command and query endpoints.
type Handlers struct {
	svc     CommandService
	queries application.DispatchQueries
}

// NewHandlers builds the handlers over the command service and the read-side
// query port.
func NewHandlers(svc CommandService, queries application.DispatchQueries) *Handlers {
	return &Handlers{svc: svc, queries: queries}
}

// --- intake --------------------------------------------------------------- //

// CreateDispatch handles POST /dispatches.
func (h *Handlers) CreateDispatch(w http.ResponseWriter, r *http.Request) {
	var body CreateDispatchRequest
	if !httpx.DecodeJSON(w, r, &body) {
		return
	}
	move, ok := h.toMovePlan(w, body)
	if !ok {
		return
	}
	var sourceQuote *intermodal.SourceQuoteID
	if body.SourceQuoteID != nil {
		sq, err := intermodal.ParseSourceQuoteID(*body.SourceQuoteID)
		if err != nil {
			badRequest(w, "invalid source_quote_id")
			return
		}
		sourceQuote = &sq
	}

	id, err := h.svc.CreateDispatch(r.Context(), move, sourceQuote)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	w.Header().Set("Location", "/v1/operations/dispatches/"+id.String())
	httpx.WriteJSON(w, http.StatusCreated, CreatedResponse{ID: id.String()})
}

func (h *Handlers) toMovePlan(w http.ResponseWriter, body CreateDispatchRequest) (intermodal.MovePlan, bool) {
	container, err := intermodal.NewContainerNumber(body.ContainerNumber)
	if err != nil {
		badRequest(w, "invalid container_number")
		return nil, false
	}
	pickup, ok := parseFacility(w, body.ContainerPickupRamp, "container_pickup_ramp")
	if !ok {
		return nil, false
	}
	dropoff, ok := parseFacility(w, body.ContainerDropoffRamp, "container_dropoff_ramp")
	if !ok {
		return nil, false
	}
	dock, ok := parseFacility(w, body.DockDoor, "dock_door")
	if !ok {
		return nil, false
	}

	switch body.PlanType {
	case intermodal.PlanLiveLoad:
		return intermodal.LiveLoad{
			ContainerNumber: container, ContainerPickupRamp: pickup,
			ContainerDropoffRamp: dropoff, DockDoor: dock, Premounted: body.Premounted,
		}, true
	case intermodal.PlanLiveUnload:
		return intermodal.LiveUnload{
			ContainerNumber: container, ContainerPickupRamp: pickup,
			ContainerDropoffRamp: dropoff, DockDoor: dock, Premounted: body.Premounted,
		}, true
	default:
		httpx.WriteProblem(w, http.StatusUnprocessableEntity, "unsupported_plan_type",
			"plan_type cannot be created from scratch")
		return nil, false
	}
}

// --- assignment, readiness, verification ---------------------------------- //

// Delegate handles POST /dispatches/{dispatchID}/delegation.
func (h *Handlers) Delegate(w http.ResponseWriter, r *http.Request) {
	id, ok := h.dispatchID(w, r)
	if !ok {
		return
	}
	var body DelegateRequest
	if !httpx.DecodeJSON(w, r, &body) {
		return
	}
	dispatcher, err := intermodal.ParseDispatcherID(body.DispatcherID)
	if err != nil {
		badRequest(w, "invalid dispatcher_id")
		return
	}
	if err := h.svc.Delegate(r.Context(), id, dispatcher); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Assign handles POST /dispatches/{dispatchID}/assignment.
func (h *Handlers) Assign(w http.ResponseWriter, r *http.Request) {
	id, ok := h.dispatchID(w, r)
	if !ok {
		return
	}
	var body AssignRequest
	if !httpx.DecodeJSON(w, r, &body) {
		return
	}
	var driver *intermodal.DriverID
	if body.DriverID != nil {
		d, err := intermodal.ParseDriverID(*body.DriverID)
		if err != nil {
			badRequest(w, "invalid driver_id")
			return
		}
		driver = &d
	}
	var truck *intermodal.TruckID
	if body.TruckID != nil {
		tk, err := intermodal.ParseTruckID(*body.TruckID)
		if err != nil {
			badRequest(w, "invalid truck_id")
			return
		}
		truck = &tk
	}

	assignmentID, err := h.svc.Assign(r.Context(), id, driver, truck)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, CreatedResponse{ID: assignmentID.String()})
}

// Approve handles POST /dispatches/{dispatchID}/approval.
func (h *Handlers) Approve(w http.ResponseWriter, r *http.Request) {
	h.command(w, r, h.svc.Approve)
}

// Start handles POST /dispatches/{dispatchID}/start.
func (h *Handlers) Start(w http.ResponseWriter, r *http.Request) {
	h.command(w, r, h.svc.Start)
}

// Verify handles POST /dispatches/{dispatchID}/verification.
func (h *Handlers) Verify(w http.ResponseWriter, r *http.Request) {
	h.command(w, r, h.svc.Verify)
}

// --- task execution ----------------------------------------------------- //

// AcquireChassis handles POST /dispatches/{dispatchID}/chassis-acquisition.
func (h *Handlers) AcquireChassis(w http.ResponseWriter, r *http.Request) {
	id, ok := h.dispatchID(w, r)
	if !ok {
		return
	}
	var body AcquireChassisRequest
	if !httpx.DecodeJSON(w, r, &body) {
		return
	}
	facility, ok := parseFacility(w, body.FacilityID, "facility_id")
	if !ok {
		return
	}
	chassis, err := intermodal.NewChassisNumber(body.ChassisNumber)
	if err != nil {
		badRequest(w, "invalid chassis_number")
		return
	}
	if err := h.svc.AcquireChassis(r.Context(), id, facility, chassis); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PickUpContainer handles POST /dispatches/{dispatchID}/container-pickup.
func (h *Handlers) PickUpContainer(w http.ResponseWriter, r *http.Request) {
	h.command(w, r, h.svc.PickUpContainer)
}

// PickUpPremountedContainer handles POST /dispatches/{dispatchID}/premounted-container-pickup.
func (h *Handlers) PickUpPremountedContainer(w http.ResponseWriter, r *http.Request) {
	id, ok := h.dispatchID(w, r)
	if !ok {
		return
	}
	var body PickUpPremountedContainerRequest
	if !httpx.DecodeJSON(w, r, &body) {
		return
	}
	chassis, err := intermodal.NewChassisNumber(body.ChassisNumber)
	if err != nil {
		badRequest(w, "invalid chassis_number")
		return
	}
	if err := h.svc.PickUpPremountedContainer(r.Context(), id, chassis); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeliverContainer handles POST /dispatches/{dispatchID}/container-delivery.
func (h *Handlers) DeliverContainer(w http.ResponseWriter, r *http.Request) {
	h.command(w, r, h.svc.DeliverContainer)
}

// Hold handles POST /dispatches/{dispatchID}/hold.
func (h *Handlers) Hold(w http.ResponseWriter, r *http.Request) {
	h.command(w, r, h.svc.Hold)
}

// TakeContainer handles POST /dispatches/{dispatchID}/container-take.
func (h *Handlers) TakeContainer(w http.ResponseWriter, r *http.Request) {
	h.command(w, r, h.svc.TakeContainer)
}

// DropContainer handles POST /dispatches/{dispatchID}/container-drop.
func (h *Handlers) DropContainer(w http.ResponseWriter, r *http.Request) {
	h.command(w, r, h.svc.DropContainer)
}

// DropPremountedContainer handles POST /dispatches/{dispatchID}/premounted-container-drop.
func (h *Handlers) DropPremountedContainer(w http.ResponseWriter, r *http.Request) {
	h.command(w, r, h.svc.DropPremountedContainer)
}

// ReturnChassis handles POST /dispatches/{dispatchID}/chassis-return.
func (h *Handlers) ReturnChassis(w http.ResponseWriter, r *http.Request) {
	id, ok := h.dispatchID(w, r)
	if !ok {
		return
	}
	var body ReturnChassisRequest
	if !httpx.DecodeJSON(w, r, &body) {
		return
	}
	facility, ok := parseFacility(w, body.FacilityID, "facility_id")
	if !ok {
		return
	}
	if err := h.svc.ReturnChassis(r.Context(), id, facility); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- shared handler plumbing ---------------------------------------------- //

// command runs a no-body transition: parse the path id, invoke run, and write
// 204 on success or the mapped error otherwise.
func (h *Handlers) command(w http.ResponseWriter, r *http.Request, run func(context.Context, intermodal.DispatchID) error) {
	id, ok := h.dispatchID(w, r)
	if !ok {
		return
	}
	if err := run(r.Context(), id); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) dispatchID(w http.ResponseWriter, r *http.Request) (intermodal.DispatchID, bool) {
	id, err := intermodal.ParseDispatchID(r.PathValue("dispatchID"))
	if err != nil {
		badRequest(w, "invalid dispatch id")
		return intermodal.DispatchID{}, false
	}
	return id, true
}

func (h *Handlers) writeErr(w http.ResponseWriter, err error) {
	status, code := statusFor(err)
	message := err.Error()
	if status >= 500 {
		message = "internal error" // don't leak internals
	}
	httpx.WriteProblem(w, status, code, message)
}

func parseFacility(w http.ResponseWriter, raw, field string) (intermodal.FacilityID, bool) {
	id, err := intermodal.ParseFacilityID(raw)
	if err != nil {
		badRequest(w, "invalid "+field)
		return intermodal.FacilityID{}, false
	}
	return id, true
}

func badRequest(w http.ResponseWriter, message string) {
	httpx.WriteProblem(w, http.StatusBadRequest, "invalid_request", message)
}
