package intermodal

import (
	"time"

	"github.com/nopalli-io/nopalli/internal/core/kernel"
)

// task entity
type Task struct {
	id        TaskID
	kind      TaskKind
	subject   TaskSubject
	status    TaskStatus
	documents []DocumentRef
}

func (t *Task) ID() TaskID         { return t.id }
func (t *Task) Status() TaskStatus { return t.status }

// stop entity
type Stop struct {
	id       StopID
	status   StopStatus
	location LocationRef
	window   AppointmentWindow
	tasks    []Task
}

func (s *Stop) ID() StopID         { return s.id }
func (s *Stop) Status() StopStatus { return s.status }

// assignment aggregate
type Assignment struct {
	id         AssignmentID
	driver     DriverRef
	truck      TruckRef
	status     AssignmentStatus
	acceptedAt time.Time
}

func newAssignment(driver DriverRef, truck TruckRef, status AssignmentStatus, at time.Time) Assignment {
	return Assignment{driver: driver, truck: truck, status: status, acceptedAt: at}
}

func (a Assignment) ID() AssignmentID         { return a.id }
func (a Assignment) Driver() DriverRef        { return a.driver }
func (a Assignment) Truck() TruckRef          { return a.truck }
func (a Assignment) Status() AssignmentStatus { return a.status }
func (a Assignment) HasTruck() bool           { return !a.truck.IsZero() }

// run aggregate
type Run struct {
	kernel.Root

	id          RunID
	status      RunStatus
	stops       []Stop
	assignments []Assignment
	version     int
}

func (r *Run) ID() RunID         { return r.id }
func (r *Run) Status() RunStatus { return r.status }
func (r *Run) CurrentAssignment() (Assignment, bool) {
	for i := len(r.assignments) - 1; i >= 0; i-- {
		if r.assignments[i].status.IsLive() {
			return r.assignments[i], true
		}
	}
	return Assignment{}, false
}

// assignment commands
func (r *Run) OfferTo() error       { return nil }
func (r *Run) WithdrawOffer() error { return nil }
func (r *Run) Release() error       { return nil }
func (r *Run) AssignTruck() error   { return nil }

func (r *Run) Accept() error  { return nil }
func (r *Run) Decline() error { return nil }

func (r *Run) Assign(driver DriverRef, truck TruckRef, by kernel.UserID, at time.Time) error {
	a := newAssignment(driver, truck, AssignmentAccepted, at)
	r.assignments = append(r.assignments, a)
	r.RecordEvents(DriverAssigned{RunID: r.ID(), Driver: driver.ID, Truck: truck.ID, By: by, At: at})
	return nil
}

// lifecycle commands

func (r *Run) Approve() error { return nil }
func (r *Run) Start() error   { return nil }

// task execution commands
func (r *Run) PickUpContainer(taskID TaskID) error                            { return nil }
func (r *Run) DropOffContainer(taskID TaskID) error                           { return nil }
func (r *Run) PickUpChassis(taskID TaskID, chassisNumber ChassisNumber) error { return nil }
func (r *Run) DropOffChassis(taskID TaskID) error                             { return nil }

func Plan(stops []Stop) *Run {
	return &Run{
		Root:   kernel.NewRoot(kernel.NewTenantID()),
		id:     NewRunID(),
		status: RunPlanned,
		stops:  stops,
	}
}
