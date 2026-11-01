package intermodal

import "testing"

var (
	driver = NewDriverID()
	truck  = NewTruckID()
)

func createRun() (*Run, error) {
	return Plan()
}

func TestAssign(t *testing.T) {
	r, _ := createRun()
	r.Assign(&driver, &truck)

	if r.Status() != RunPlanned {
		t.Fatalf("status = %v, want planned", r.Status())
	}
}
