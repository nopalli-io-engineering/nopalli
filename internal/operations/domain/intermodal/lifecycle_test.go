package intermodal

import (
	"testing"

	"github.com/nopalli-io/nopalli/internal/core/kernel"
)

var (
	driver = NewDriverID()
	truck  = NewTruckID()
	user   = kernel.NewUserID()
)

func newRun() *Run {
	stops := liveLoadStops()
	return Plan(stops)
}

func TestOfferTrip(t *testing.T) {
	r := newRun()
	err := r.Assign(driverRef, truckRef, user, testNow)

	mustNoError(t, err)
	if r.Status() != RunPlanned {
		t.Fatalf("status = %v, want planned", r.Status())
	}
	// want := Assignee{Driver: driver, Truck: truck}
	// if got, ok := r.Assignee(); !ok || got != expectedAssignee {
	// 	t.Fatalf("current assignee = %v,%v", got, ok)
	// }
	// assertTypeNames(t, ev, "")
}
