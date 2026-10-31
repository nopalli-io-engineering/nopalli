package rest_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	coredb "github.com/nopalli-io/nopalli/internal/core/infrastructure/db"
	"github.com/nopalli-io/nopalli/internal/operations/application"
	intermodal "github.com/nopalli-io/nopalli/internal/operations/domain/intermodal"
	opsdb "github.com/nopalli-io/nopalli/internal/operations/infrastructure/db"
	"github.com/nopalli-io/nopalli/internal/operations/interfaces/rest"
)

// Full-stack HTTP integration: real handlers → service → unit of work → pgx →
// Postgres, driven over HTTP. Mirrors the wiring in cmd/api. Runs under
// `make test-integration`; `go test -short` skips it.

func dsnOrSkipHTTP(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	dsn := os.Getenv("DB_URL")
	if dsn == "" {
		dsn = os.Getenv("TEST_DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("no DB_URL / TEST_DATABASE_URL set; skipping integration test")
	}
	return dsn
}

func TestIntegration_HTTPStack(t *testing.T) {
	ctx := context.Background()
	dsn := dsnOrSkipHTTP(t)
	if err := coredb.Migrate(ctx, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := coredb.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	uow := opsdb.NewUnitOfWork(pool, nil)
	svc := application.NewIntermodalDispatchService(uow, application.AllowAll{})
	queries := opsdb.NewDispatchQueryService(pool)
	mux := http.NewServeMux()
	rest.RegisterRoutes(mux, rest.NewHandlers(svc, queries))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	post := func(path, body string) *http.Response {
		t.Helper()
		var rdr *strings.Reader
		if body == "" {
			rdr = strings.NewReader("")
		} else {
			rdr = strings.NewReader(body)
		}
		req, _ := http.NewRequest(http.MethodPost, srv.URL+path, rdr)
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("POST %s: %v", path, err)
		}
		return resp
	}

	base := "/v1/operations/dispatches"

	// Create.
	createBody := `{
		"plan_type": "live_load",
		"container_number": "MSCU1234567",
		"container_pickup_ramp": "` + intermodal.NewFacilityID().String() + `",
		"container_dropoff_ramp": "` + intermodal.NewFacilityID().String() + `",
		"dock_door": "` + intermodal.NewFacilityID().String() + `"
	}`
	resp := post(base, createBody)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if loc == "" {
		t.Fatal("create: missing Location header")
	}
	id := strings.TrimPrefix(loc, base+"/")

	// Approving before delegation is a state conflict → 409.
	if r := post(base+"/"+id+"/approval", ""); r.StatusCode != http.StatusConflict {
		t.Fatalf("premature approve status = %d, want 409", r.StatusCode)
	} else {
		r.Body.Close()
	}

	// Drive to in-progress over HTTP.
	steps := []struct {
		path, body string
		want       int
	}{
		{"/delegation", `{"dispatcher_id":"` + intermodal.NewDispatcherID().String() + `"}`, 204},
		{"/assignment", `{"driver_id":"` + intermodal.NewDriverID().String() + `","truck_id":"` + intermodal.NewTruckID().String() + `"}`, 201},
		{"/approval", "", 204},
		{"/start", "", 204},
	}
	for _, s := range steps {
		r := post(base+"/"+id+s.path, s.body)
		if r.StatusCode != s.want {
			t.Fatalf("POST %s status = %d, want %d", s.path, r.StatusCode, s.want)
		}
		r.Body.Close()
	}

	// Confirm the persisted state through the read side.
	did, err := intermodal.ParseDispatchID(id)
	if err != nil {
		t.Fatalf("parse returned id: %v", err)
	}
	detail, err := opsdb.NewDispatchQueryService(pool).GetDispatch(ctx, did.UUID(), application.VisibleAll())
	if err != nil || detail == nil {
		t.Fatalf("read back: detail=%v err=%v", detail, err)
	}
	if detail.Status != intermodal.StatusInProgress {
		t.Fatalf("persisted status = %v, want in_progress", detail.Status)
	}

	// And over HTTP: GET /{id} and GET list both reflect the same state.
	get := func(path string) *http.Response {
		t.Helper()
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		return resp
	}

	gr := get(base + "/" + id)
	if gr.StatusCode != http.StatusOK {
		t.Fatalf("GET detail status = %d, want 200", gr.StatusCode)
	}
	var gotDetail rest.DispatchDetailResponse
	if err := json.NewDecoder(gr.Body).Decode(&gotDetail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	gr.Body.Close()
	if gotDetail.ID != id || gotDetail.Status != intermodal.StatusInProgress {
		t.Fatalf("GET detail = %+v, want id %s in_progress", gotDetail, id)
	}
	// Children are projected onto the detail, carrying audit timestamps.
	if len(gotDetail.Tasks) == 0 {
		t.Fatal("GET detail: expected tasks")
	}
	if gotDetail.Tasks[0].CreatedAt.IsZero() || gotDetail.Tasks[0].UpdatedAt.IsZero() {
		t.Fatalf("GET detail: task missing timestamps: %+v", gotDetail.Tasks[0])
	}
	if len(gotDetail.Assignments) != 1 {
		t.Fatalf("GET detail: assignments = %d, want 1", len(gotDetail.Assignments))
	}

	lr := get(base + "?status=" + string(intermodal.StatusInProgress))
	if lr.StatusCode != http.StatusOK {
		t.Fatalf("GET list status = %d, want 200", lr.StatusCode)
	}
	var list []rest.DispatchSummaryResponse
	if err := json.NewDecoder(lr.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	lr.Body.Close()
	found := false
	for _, s := range list {
		if s.ID == id {
			found = true
		}
	}
	if !found {
		t.Fatalf("GET list did not contain %s: %+v", id, list)
	}

	// 404 for a well-formed but unknown id.
	nr := get(base + "/" + intermodal.NewDispatchID().String())
	if nr.StatusCode != http.StatusNotFound {
		t.Fatalf("GET unknown status = %d, want 404", nr.StatusCode)
	}
	nr.Body.Close()
}
