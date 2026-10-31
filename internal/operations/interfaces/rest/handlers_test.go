package rest_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/nopalli-io/nopalli/internal/core/domain"
	"github.com/nopalli-io/nopalli/internal/operations/application"
	"github.com/nopalli-io/nopalli/internal/operations/domain/intermodal_old"
	"github.com/nopalli-io/nopalli/internal/operations/interfaces/rest"
)

// fakeService is a CommandService that returns canned results/errors and
// captures the dispatch id of the last no-body command, so tests can assert
// routing, decoding, status codes, and error mapping without a DB.
type fakeService struct {
	createID intermodal_old.DispatchID
	assignID intermodal_old.AssignmentID
	err      error
	lastID   intermodal_old.DispatchID
}

func (f *fakeService) CreateDispatch(context.Context, intermodal_old.MovePlan, *intermodal_old.SourceQuoteID) (intermodal_old.DispatchID, error) {
	return f.createID, f.err
}
func (f *fakeService) Delegate(_ context.Context, id intermodal_old.DispatchID, _ intermodal_old.DispatcherID) error {
	f.lastID = id
	return f.err
}
func (f *fakeService) Assign(_ context.Context, id intermodal_old.DispatchID, _ *intermodal_old.DriverID, _ *intermodal_old.TruckID) (intermodal_old.AssignmentID, error) {
	f.lastID = id
	return f.assignID, f.err
}
func (f *fakeService) Approve(_ context.Context, id intermodal_old.DispatchID) error {
	f.lastID = id
	return f.err
}
func (f *fakeService) Start(_ context.Context, id intermodal_old.DispatchID) error {
	f.lastID = id
	return f.err
}
func (f *fakeService) Verify(_ context.Context, id intermodal_old.DispatchID) error {
	f.lastID = id
	return f.err
}
func (f *fakeService) AcquireChassis(_ context.Context, id intermodal_old.DispatchID, _ intermodal_old.FacilityID, _ intermodal_old.ChassisNumber) error {
	f.lastID = id
	return f.err
}
func (f *fakeService) PickUpContainer(_ context.Context, id intermodal_old.DispatchID) error {
	f.lastID = id
	return f.err
}
func (f *fakeService) PickUpPremountedContainer(_ context.Context, id intermodal_old.DispatchID, _ intermodal_old.ChassisNumber) error {
	f.lastID = id
	return f.err
}
func (f *fakeService) DeliverContainer(_ context.Context, id intermodal_old.DispatchID) error {
	f.lastID = id
	return f.err
}
func (f *fakeService) Hold(_ context.Context, id intermodal_old.DispatchID) error {
	f.lastID = id
	return f.err
}
func (f *fakeService) TakeContainer(_ context.Context, id intermodal_old.DispatchID) error {
	f.lastID = id
	return f.err
}
func (f *fakeService) DropContainer(_ context.Context, id intermodal_old.DispatchID) error {
	f.lastID = id
	return f.err
}
func (f *fakeService) DropPremountedContainer(_ context.Context, id intermodal_old.DispatchID) error {
	f.lastID = id
	return f.err
}
func (f *fakeService) ReturnChassis(_ context.Context, id intermodal_old.DispatchID, _ intermodal_old.FacilityID) error {
	f.lastID = id
	return f.err
}

// fakeQueries is a DispatchQueries returning canned read-model results and
// capturing the last visibility/filter, so GET tests need no DB.
type fakeQueries struct {
	summaries []application.DispatchSummary
	detail    *application.DispatchDetail
	err       error

	lastVis    application.Visibility
	lastFilter application.ListFilter
	lastID     uuid.UUID
}

func (q *fakeQueries) ListDispatches(_ context.Context, vis application.Visibility, filter application.ListFilter) ([]application.DispatchSummary, error) {
	q.lastVis, q.lastFilter = vis, filter
	return q.summaries, q.err
}
func (q *fakeQueries) GetDispatch(_ context.Context, id uuid.UUID, vis application.Visibility) (*application.DispatchDetail, error) {
	q.lastID, q.lastVis = id, vis
	return q.detail, q.err
}

func serve(svc rest.CommandService) *http.ServeMux {
	return serveFull(svc, &fakeQueries{})
}

func serveFull(svc rest.CommandService, queries application.DispatchQueries) *http.ServeMux {
	mux := http.NewServeMux()
	rest.RegisterRoutes(mux, rest.NewHandlers(svc, queries))
	return mux
}

func do(t *testing.T, mux *http.ServeMux, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

func decodeProblem(t *testing.T, w *httptest.ResponseRecorder) (code, message string) {
	t.Helper()
	var p struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatalf("decode problem: %v (body=%s)", err, w.Body.String())
	}
	return p.Code, p.Message
}

const dispatchesPath = "/v1/operations/dispatches"

func TestCreateDispatch_Created(t *testing.T) {
	f := &fakeService{createID: intermodal_old.NewDispatchID()}
	mux := serve(f)
	body := `{
		"plan_type": "live_load",
		"container_number": "MSCU1234567",
		"container_pickup_ramp": "` + intermodal_old.NewFacilityID().String() + `",
		"container_dropoff_ramp": "` + intermodal_old.NewFacilityID().String() + `",
		"dock_door": "` + intermodal_old.NewFacilityID().String() + `"
	}`
	w := do(t, mux, http.MethodPost, dispatchesPath, body)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body=%s)", w.Code, w.Body.String())
	}
	wantLoc := dispatchesPath + "/" + f.createID.String()
	if got := w.Header().Get("Location"); got != wantLoc {
		t.Fatalf("Location = %q, want %q", got, wantLoc)
	}
	var resp struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.ID != f.createID.String() {
		t.Fatalf("body id = %q, want %q", resp.ID, f.createID.String())
	}
}

func TestCreateDispatch_UnsupportedPlan(t *testing.T) {
	mux := serve(&fakeService{createID: intermodal_old.NewDispatchID()})
	body := `{
		"plan_type": "ad_hoc",
		"container_number": "MSCU1234567",
		"container_pickup_ramp": "` + intermodal_old.NewFacilityID().String() + `",
		"container_dropoff_ramp": "` + intermodal_old.NewFacilityID().String() + `",
		"dock_door": "` + intermodal_old.NewFacilityID().String() + `"
	}`
	w := do(t, mux, http.MethodPost, dispatchesPath, body)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", w.Code)
	}
	if code, _ := decodeProblem(t, w); code != "unsupported_plan_type" {
		t.Fatalf("code = %q, want unsupported_plan_type", code)
	}
}

func TestCreateDispatch_BadUUID(t *testing.T) {
	mux := serve(&fakeService{})
	body := `{
		"plan_type": "live_load",
		"container_number": "MSCU1234567",
		"container_pickup_ramp": "not-a-uuid",
		"container_dropoff_ramp": "` + intermodal_old.NewFacilityID().String() + `",
		"dock_door": "` + intermodal_old.NewFacilityID().String() + `"
	}`
	w := do(t, mux, http.MethodPost, dispatchesPath, body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestApprove_NoContentAndRoutesID(t *testing.T) {
	f := &fakeService{}
	mux := serve(f)
	id := intermodal_old.NewDispatchID()
	w := do(t, mux, http.MethodPost, dispatchesPath+"/"+id.String()+"/approval", "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
	if f.lastID != id {
		t.Fatalf("service got id %v, want %v", f.lastID, id)
	}
}

func TestApprove_DomainConflict(t *testing.T) {
	f := &fakeService{err: intermodal_old.ErrDispatchNotReady}
	mux := serve(f)
	id := intermodal_old.NewDispatchID()
	w := do(t, mux, http.MethodPost, dispatchesPath+"/"+id.String()+"/approval", "")
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
	if code, _ := decodeProblem(t, w); code != "dispatch_not_ready" {
		t.Fatalf("code = %q, want dispatch_not_ready", code)
	}
}

func TestApprove_NotFound(t *testing.T) {
	id := intermodal_old.NewDispatchID()
	f := &fakeService{err: intermodal_old.NewDispatchNotFound(id)}
	mux := serve(f)
	w := do(t, mux, http.MethodPost, dispatchesPath+"/"+id.String()+"/approval", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestApprove_PermissionDenied(t *testing.T) {
	f := &fakeService{err: domain.PermissionDenied("operations:intermodal_dispatch.approve")}
	mux := serve(f)
	id := intermodal_old.NewDispatchID()
	w := do(t, mux, http.MethodPost, dispatchesPath+"/"+id.String()+"/approval", "")
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestApprove_BadPathID(t *testing.T) {
	mux := serve(&fakeService{})
	w := do(t, mux, http.MethodPost, dispatchesPath+"/not-a-uuid/approval", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestAssign_Created(t *testing.T) {
	f := &fakeService{assignID: intermodal_old.NewAssignmentID()}
	mux := serve(f)
	id := intermodal_old.NewDispatchID()
	body := `{"driver_id": "` + intermodal_old.NewDriverID().String() + `", "truck_id": "` + intermodal_old.NewTruckID().String() + `"}`
	w := do(t, mux, http.MethodPost, dispatchesPath+"/"+id.String()+"/assignment", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body=%s)", w.Code, w.Body.String())
	}
	var resp struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.ID != f.assignID.String() {
		t.Fatalf("assignment id = %q, want %q", resp.ID, f.assignID.String())
	}
}

func TestAcquireChassis_BadChassisNumber(t *testing.T) {
	mux := serve(&fakeService{})
	id := intermodal_old.NewDispatchID()
	body := `{"facility_id": "` + intermodal_old.NewFacilityID().String() + `", "chassis_number": "  "}`
	w := do(t, mux, http.MethodPost, dispatchesPath+"/"+id.String()+"/chassis-acquisition", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestListDispatches_OK(t *testing.T) {
	id := intermodal_old.NewDispatchID()
	q := &fakeQueries{summaries: []application.DispatchSummary{{
		ID:       id.UUID(),
		PlanType: intermodal_old.PlanLiveLoad,
		Status:   intermodal_old.StatusInProgress,
	}}}
	mux := serveFull(&fakeService{}, q)
	w := do(t, mux, http.MethodGet, dispatchesPath, "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", w.Code, w.Body.String())
	}
	var resp []rest.DispatchSummaryResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 1 || resp[0].ID != id.String() {
		t.Fatalf("summaries = %+v, want one with id %s", resp, id.String())
	}
	if resp[0].SourceQuoteID != nil {
		t.Fatalf("source_quote_id = %v, want omitted", resp[0].SourceQuoteID)
	}
}

func TestListDispatches_EmptyIsArray(t *testing.T) {
	mux := serveFull(&fakeService{}, &fakeQueries{})
	w := do(t, mux, http.MethodGet, dispatchesPath, "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got := strings.TrimSpace(w.Body.String()); got != "[]" {
		t.Fatalf("body = %q, want []", got)
	}
}

func TestListDispatches_FilterParsed(t *testing.T) {
	q := &fakeQueries{}
	mux := serveFull(&fakeService{}, q)
	w := do(t, mux, http.MethodGet, dispatchesPath+"?status="+string(intermodal_old.StatusInProgress)+"&limit=10&offset=5", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if q.lastFilter.Status == nil || *q.lastFilter.Status != intermodal_old.StatusInProgress {
		t.Fatalf("filter status = %v, want %v", q.lastFilter.Status, intermodal_old.StatusInProgress)
	}
	if q.lastFilter.Limit != 10 || q.lastFilter.Offset != 5 {
		t.Fatalf("filter limit/offset = %d/%d, want 10/5", q.lastFilter.Limit, q.lastFilter.Offset)
	}
}

func TestListDispatches_BadStatus(t *testing.T) {
	mux := serveFull(&fakeService{}, &fakeQueries{})
	w := do(t, mux, http.MethodGet, dispatchesPath+"?status=not-a-status", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestListDispatches_BadLimit(t *testing.T) {
	mux := serveFull(&fakeService{}, &fakeQueries{})
	w := do(t, mux, http.MethodGet, dispatchesPath+"?limit=-1", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestGetDispatch_OK(t *testing.T) {
	id := intermodal_old.NewDispatchID()
	dispatcher := intermodal_old.NewDispatcherID().UUID()
	q := &fakeQueries{detail: &application.DispatchDetail{
		ID:           id.UUID(),
		DispatcherID: &dispatcher,
		PlanType:     intermodal_old.PlanLiveLoad,
		Status:       intermodal_old.StatusInProgress,
	}}
	mux := serveFull(&fakeService{}, q)
	w := do(t, mux, http.MethodGet, dispatchesPath+"/"+id.String(), "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", w.Code, w.Body.String())
	}
	if q.lastID != id.UUID() {
		t.Fatalf("query got id %v, want %v", q.lastID, id.UUID())
	}
	var resp rest.DispatchDetailResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != id.String() || resp.DispatcherID == nil || *resp.DispatcherID != dispatcher.String() {
		t.Fatalf("detail = %+v, want id %s dispatcher %s", resp, id.String(), dispatcher.String())
	}
	// Child collections with no rows serialize as [] (present, not null).
	if got := w.Body.String(); !strings.Contains(got, `"tasks":[]`) {
		t.Fatalf("empty tasks should be [], body=%s", got)
	}
}

func TestGetDispatch_IncludesChildren(t *testing.T) {
	id := intermodal_old.NewDispatchID()
	taskID := intermodal_old.NewTaskID().UUID()
	assignID := intermodal_old.NewAssignmentID().UUID()
	now := time.Now().UTC()
	q := &fakeQueries{detail: &application.DispatchDetail{
		ID:       id.UUID(),
		PlanType: intermodal_old.PlanLiveLoad,
		Status:   intermodal_old.StatusInProgress,
		Tasks: []application.TaskView{{
			ID: taskID, No: 1, Action: intermodal_old.ActionAcquireChassis,
			Status: intermodal_old.TaskPending, CreatedAt: now, UpdatedAt: now,
		}},
		Assignments: []application.AssignmentView{{
			ID: assignID, Status: intermodal_old.AssignmentPending, CreatedAt: now, UpdatedAt: now,
		}},
	}}
	mux := serveFull(&fakeService{}, q)
	w := do(t, mux, http.MethodGet, dispatchesPath+"/"+id.String(), "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", w.Code, w.Body.String())
	}
	var resp rest.DispatchDetailResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Tasks) != 1 || resp.Tasks[0].ID != taskID.String() {
		t.Fatalf("tasks = %+v, want one with id %s", resp.Tasks, taskID)
	}
	if resp.Tasks[0].CreatedAt.IsZero() {
		t.Fatal("task created_at missing from wire view")
	}
	if len(resp.Assignments) != 1 || resp.Assignments[0].ID != assignID.String() {
		t.Fatalf("assignments = %+v, want one with id %s", resp.Assignments, assignID)
	}
}

func TestGetDispatch_NotFound(t *testing.T) {
	mux := serveFull(&fakeService{}, &fakeQueries{detail: nil})
	id := intermodal_old.NewDispatchID()
	w := do(t, mux, http.MethodGet, dispatchesPath+"/"+id.String(), "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	if code, _ := decodeProblem(t, w); code != "dispatch_not_found" {
		t.Fatalf("code = %q, want dispatch_not_found", code)
	}
}

func TestGetDispatch_BadID(t *testing.T) {
	mux := serveFull(&fakeService{}, &fakeQueries{})
	w := do(t, mux, http.MethodGet, dispatchesPath+"/not-a-uuid", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}
