package intermodal

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/nopalli-io/nopalli/internal/core/kernel"
)

var (
	driver = NewDriverID()
	truck  = NewTruckID()
	// user   = kernel.NewUserID()
)

func newTestRun() *Run {
	stops := liveLoadStops()
	return Plan(stops)
}

func TestAssign(t *testing.T) {
	r := newTestRun()
	driver := driverRef
	truck := truckRef
	user := kernel.NewUserID()
	at := testNow
	// track := trackFrom(r)

	if err := r.Assign(driver, truck, user, at); err != nil {
		t.Fatal(err)
	}

	got, ok := r.CurrentAssignment()
	if !ok {
		t.Fatal("no current assignment")
	}
	want := Assignment{
		driver:     driver,
		truck:      truck,
		status:     AssignmentAccepted,
		acceptedAt: at,
	}
	if diff := cmp.Diff(want, got,
		cmp.AllowUnexported(Assignment{}),
		cmpopts.IgnoreFields(Assignment{}, "id"),
		idOpts,
	); diff != "" {
		t.Errorf("assignment mismatch (-want +got):\n%s", diff)
	}
	gotEvents := getEvents(r)
	wantEvents := []kernel.DomainEvent{
		DriverAssigned{RunID: r.ID(), Driver: driver.ID, Truck: truck.ID, By: user, At: at},
	}
	if diff := cmp.Diff(wantEvents, gotEvents); diff != "" {
		t.Errorf("events (-want +got):\n%s", diff)
	}
}
