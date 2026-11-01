package intermodal

import (
	"github.com/nopalli-io/nopalli/internal/core/domain"
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
	tasks    []Task
}

func (s *Stop) ID() StopID         { return s.id }
func (s *Stop) Status() StopStatus { return s.status }

// assignment aggregate
type Assignment struct {
	id       AssignmentID
	driverID DriverID
	truckID  TruckID
	status   AssignmentStatus
}

// run aggregate
type Run struct {
	domain.Root

	id      RunID
	status  RunStatus
	stops   []Stop
	version int
}

func (r *Run) ID() RunID         { return r.id }
func (r *Run) Status() RunStatus { return r.status }

// lifecycle commands
func (r *Run) Assign(driverID *DriverID, truckID *TruckID) (AssignmentID, error) {
	return AssignmentID{}, nil
}
func (r *Run) Approve() error { return nil }
func (r *Run) Start() error   { return nil }
func (r *Run) Verify() error  { return nil }

// Task execution commands
func (r *Run) PickUpContainer(taskID TaskID) error                            { return nil }
func (r *Run) DropOffContainer(taskID TaskID) error                           { return nil }
func (r *Run) PickUpChassis(taskID TaskID, chassisNumber ChassisNumber) error { return nil }
func (r *Run) DropOffChassis(taskID TaskID) error                             { return nil }
