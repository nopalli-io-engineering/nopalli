package intermodal_old

// Dispatch lifecycle invariants — cross-plan (rdt: tests/test_dispatch_lifecycle.py).
// These exercise the aggregate's state machine, not any plan's action sequence;
// a LiveLoad is used only to build a valid fixture.

import (
	"errors"
	"testing"
)

// Shared fixtures. Generated once per run (rdt generated these at module load).
var (
	dispatcher         = NewDispatcherID()
	sourceQuote        = NewSourceQuoteID()
	chassis            = mkChassis("CHAS-0001")
	chassisPickupYard  = NewFacilityID()
	chassisDropoffYard = NewFacilityID()
	driver             = NewDriverID()
	truck              = NewTruckID()
	otherDriver        = NewDriverID()
	otherTruck         = NewTruckID()
)

func assignee() Assignee      { return Assignee{Driver: driver, Truck: truck} }
func otherAssignee() Assignee { return Assignee{Driver: otherDriver, Truck: otherTruck} }

func newDispatch() *Dispatch {
	move := LiveLoad{
		ContainerNumber:      mkContainer("MSCU1234567"),
		ContainerPickupRamp:  NewFacilityID(),
		ContainerDropoffRamp: NewFacilityID(),
		DockDoor:             NewFacilityID(),
	}
	return Queue(&sourceQuote, move)
}

func startDispatch(t *testing.T, d *Dispatch) {
	t.Helper()
	d.Delegate(dispatcher)
	_, err := d.Assign(&driver, &truck)
	mustNoErr(t, err)
	mustNoErr(t, d.Approve())
	mustNoErr(t, d.Start())
}

// completeDispatch runs a full live-load through completion; returns the assignment id.
func completeDispatch(t *testing.T, d *Dispatch) AssignmentID {
	t.Helper()
	d.Delegate(dispatcher)
	assignmentID, err := d.Assign(&driver, &truck)
	mustNoErr(t, err)
	mustNoErr(t, d.Approve())
	mustNoErr(t, d.Start())
	_, err = d.AcquireChassis(chassisPickupYard, chassis)
	mustNoErr(t, err)
	_, err = d.PickUpContainer()
	mustNoErr(t, err)
	_, err = d.DeliverContainer()
	mustNoErr(t, err)
	_, err = d.Hold()
	mustNoErr(t, err)
	_, err = d.TakeContainer()
	mustNoErr(t, err)
	_, err = d.DropContainer()
	mustNoErr(t, err)
	_, err = d.ReturnChassis(chassisDropoffYard)
	mustNoErr(t, err)
	return assignmentID
}

func requireNoAssignee(t *testing.T, d *Dispatch) {
	t.Helper()
	if _, ok := d.CurrentAssignee(); ok {
		t.Fatal("expected no current assignee")
	}
}

func requireNoLiveAssignee(t *testing.T, d *Dispatch) {
	t.Helper()
	if _, ok := d.LiveAssignee(); ok {
		t.Fatal("expected no live assignee")
	}
}

// -------------------------------------------------------------------------- //

func TestDelegation(t *testing.T) {
	d := newDispatch()
	track := trackFrom(d)
	d.Delegate(dispatcher)

	if d.Status() != StatusQueued {
		t.Fatalf("status = %v, want queued", d.Status())
	}
	if d.Dispatcher() == nil || *d.Dispatcher() != dispatcher {
		t.Fatal("dispatcher not set")
	}
	ev := track.since()
	assertTypeNames(t, ev, "DispatchDelegated")
	dd := oneEvent[DispatchDelegated](t, ev)
	if dd.DispatchID != d.ID() || dd.ToDispatcherID != dispatcher {
		t.Fatal("DispatchDelegated payload mismatch")
	}
}

func TestApproveRequiresDelegation(t *testing.T) {
	d := newDispatch()
	_, err := d.Assign(&driver, &truck)
	mustNoErr(t, err)
	err = d.Approve()
	if !errors.Is(err, ErrDispatchNotDelegated) {
		t.Fatalf("err = %v, want ErrDispatchNotDelegated", err)
	}
	if err.Error() != "A dispatch needs a dispatcher for approval." {
		t.Fatalf("message = %q", err.Error())
	}
}

func TestAssign(t *testing.T) {
	d := newDispatch()
	track := trackFrom(d)
	assignmentID, err := d.Assign(&driver, &truck)
	mustNoErr(t, err)

	if d.Status() != StatusQueued {
		t.Fatalf("status = %v, want queued", d.Status())
	}
	if got, ok := d.CurrentAssignee(); !ok || got != assignee() {
		t.Fatalf("current assignee = %v,%v", got, ok)
	}
	requireNoLiveAssignee(t, d)
	assertAssigneeSet(t, d, assignee())
	ev := track.since()
	assertTypeNames(t, ev, "AssignmentProposed")
	a := oneEvent[AssignmentProposed](t, ev)
	if a.DispatchID != d.ID() || a.AssignmentID != assignmentID || a.DriverID != driver || a.TruckID != truck {
		t.Fatal("AssignmentProposed payload mismatch")
	}
}

func TestAssigningAfterApproval(t *testing.T) {
	d := newDispatch()
	d.Delegate(dispatcher)
	_, err := d.Assign(&driver, &truck)
	mustNoErr(t, err)
	mustNoErr(t, d.Approve())
	track := trackFrom(d)
	_, err = d.Assign(&otherDriver, &otherTruck)
	mustNoErr(t, err)

	if d.Status() != StatusQueued {
		t.Fatalf("status = %v, want queued", d.Status())
	}
	if got, ok := d.CurrentAssignee(); !ok || got != otherAssignee() {
		t.Fatalf("current assignee = %v,%v", got, ok)
	}
	requireNoLiveAssignee(t, d)
	assertAssigneeSet(t, d, otherAssignee())
	ev := track.since()
	assertTypeNames(t, ev, "DispatchRequeued", "AssignmentProposed")
	ac := oneEvent[AssignmentProposed](t, ev)
	if ac.DriverID != otherDriver || ac.TruckID != otherTruck {
		t.Fatal("AssignmentProposed payload mismatch")
	}
	if oneEvent[DispatchRequeued](t, ev).DispatchID != d.ID() {
		t.Fatal("DispatchRequeued payload mismatch")
	}
}

func TestAssigningEmptyDriver(t *testing.T) {
	d := newDispatch()
	track := trackFrom(d)
	_, err := d.Assign(nil, &truck)
	mustNoErr(t, err)

	want := Assignee{Truck: truck}
	if got, ok := d.CurrentAssignee(); !ok || got != want {
		t.Fatalf("current assignee = %v,%v", got, ok)
	}
	requireNoLiveAssignee(t, d)
	assertAssigneeSet(t, d, want)
	if len(track.since()) != 0 {
		t.Fatal("expected no events")
	}
}

func TestAssigningEmptyTruck(t *testing.T) {
	d := newDispatch()
	track := trackFrom(d)
	_, err := d.Assign(&driver, nil)
	mustNoErr(t, err)

	want := Assignee{Driver: driver}
	if got, ok := d.CurrentAssignee(); !ok || got != want {
		t.Fatalf("current assignee = %v,%v", got, ok)
	}
	if len(track.since()) != 0 {
		t.Fatal("expected no events")
	}
}

func TestAssignClears(t *testing.T) {
	d := newDispatch()
	first, err := d.Assign(&driver, &truck)
	mustNoErr(t, err)
	track := trackFrom(d)
	last, err := d.Assign(nil, nil)
	mustNoErr(t, err)

	if d.Status() != StatusQueued {
		t.Fatalf("status = %v, want queued", d.Status())
	}
	requireNoAssignee(t, d)
	requireNoLiveAssignee(t, d)
	assertAssigneeSet(t, d, Assignee{})
	if first != last {
		t.Fatal("assignment id should be stable across clear")
	}
	if len(track.since()) != 0 {
		t.Fatal("expected no events")
	}
}

func TestInitiallyAssigningRequiresDriverOrTruck(t *testing.T) {
	d := newDispatch()
	_, err := d.Assign(nil, nil)
	if !errors.Is(err, ErrDispatchAssigneeEmpty) {
		t.Fatalf("err = %v, want ErrDispatchAssigneeEmpty", err)
	}
	if err.Error() != "Dispatch needs a valid assignee to carry out action." {
		t.Fatalf("message = %q", err.Error())
	}
}

func TestApprove(t *testing.T) {
	d := newDispatch()
	d.Delegate(dispatcher)
	_, err := d.Assign(&driver, &truck)
	mustNoErr(t, err)
	track := trackFrom(d)
	mustNoErr(t, d.Approve())

	if d.Status() != StatusReady {
		t.Fatalf("status = %v, want ready", d.Status())
	}
	if got, ok := d.CurrentAssignee(); !ok || got != assignee() {
		t.Fatalf("current assignee = %v,%v", got, ok)
	}
	requireNoLiveAssignee(t, d)
	ev := track.since()
	assertTypeNames(t, ev, "DispatchApproved")
	da := oneEvent[DispatchApproved](t, ev)
	if da.DispatchID != d.ID() || da.DriverID != driver || da.TruckID != truck {
		t.Fatal("DispatchApproved payload mismatch")
	}
}

func TestApprovalRequiresAssignedDriver(t *testing.T) {
	d := newDispatch()
	d.Delegate(dispatcher)
	_, err := d.Assign(nil, &truck)
	mustNoErr(t, err)
	err = d.Approve()
	if !errors.Is(err, ErrDispatchDriverMissing) {
		t.Fatalf("err = %v, want ErrDispatchDriverMissing", err)
	}
	if err.Error() != "Dispatch needs a driver." {
		t.Fatalf("message = %q", err.Error())
	}
}

func TestApprovalRequiresAssignedTruck(t *testing.T) {
	d := newDispatch()
	d.Delegate(dispatcher)
	_, err := d.Assign(&driver, nil)
	mustNoErr(t, err)
	err = d.Approve()
	if !errors.Is(err, ErrDispatchTruckMissing) {
		t.Fatalf("err = %v, want ErrDispatchTruckMissing", err)
	}
}

func TestApprovalRequiresAssignment(t *testing.T) {
	d := newDispatch()
	d.Delegate(dispatcher)
	if d.Status() != StatusQueued {
		t.Fatalf("status = %v, want queued", d.Status())
	}
	err := d.Approve()
	if !errors.Is(err, ErrDispatchNotAssigned) {
		t.Fatalf("err = %v, want ErrDispatchNotAssigned", err)
	}
	if err.Error() != "A dispatch needs an assignment for approval." {
		t.Fatalf("message = %q", err.Error())
	}
}

func TestApproveUsesLatestAssignment(t *testing.T) {
	d := newDispatch()
	d.Delegate(dispatcher)
	first, err := d.Assign(&driver, &truck)
	mustNoErr(t, err)
	latest, err := d.Assign(&otherDriver, &otherTruck)
	mustNoErr(t, err)
	track := trackFrom(d)
	mustNoErr(t, d.Approve())

	if d.Status() != StatusReady {
		t.Fatalf("status = %v, want ready", d.Status())
	}
	if got, ok := d.CurrentAssignee(); !ok || got != otherAssignee() {
		t.Fatalf("current assignee = %v,%v", got, ok)
	}
	if first != latest {
		t.Fatal("assignment id should be stable (update in place)")
	}
	da := oneEvent[DispatchApproved](t, track.since())
	if da.DriverID != otherDriver || da.TruckID != otherTruck {
		t.Fatal("DispatchApproved payload mismatch")
	}
}

func TestStart(t *testing.T) {
	d := newDispatch()
	d.Delegate(dispatcher)
	assignmentID, err := d.Assign(&driver, &truck)
	mustNoErr(t, err)
	mustNoErr(t, d.Approve())
	track := trackFrom(d)
	mustNoErr(t, d.Start())

	if d.Status() != StatusInProgress {
		t.Fatalf("status = %v, want in_progress", d.Status())
	}
	if got, ok := d.LiveAssignee(); !ok || got != assignee() {
		t.Fatalf("live assignee = %v,%v", got, ok)
	}
	if a := d.Assignment(assignmentID); a == nil || a.Status() != AssignmentActive {
		t.Fatal("assignment should be active")
	}
	if oneEvent[DispatchStarted](t, track.since()).DispatchID != d.ID() {
		t.Fatal("DispatchStarted payload mismatch")
	}
}

func TestComplete(t *testing.T) {
	d := newDispatch()
	d.Delegate(dispatcher)
	assignmentID, err := d.Assign(&driver, &truck)
	mustNoErr(t, err)
	mustNoErr(t, d.Approve())
	mustNoErr(t, d.Start())
	_, err = d.AcquireChassis(chassisPickupYard, chassis)
	mustNoErr(t, err)
	_, err = d.PickUpContainer()
	mustNoErr(t, err)
	_, err = d.DeliverContainer()
	mustNoErr(t, err)
	_, err = d.Hold()
	mustNoErr(t, err)
	_, err = d.TakeContainer()
	mustNoErr(t, err)
	_, err = d.DropContainer()
	mustNoErr(t, err)
	track := trackFrom(d)
	_, err = d.ReturnChassis(chassisDropoffYard)
	mustNoErr(t, err)

	if d.Status() != StatusCompleted {
		t.Fatalf("status = %v, want completed", d.Status())
	}
	requireNoAssignee(t, d)
	requireNoLiveAssignee(t, d)
	if a := d.Assignment(assignmentID); a == nil || a.Status() != AssignmentConcluded {
		t.Fatal("assignment should be concluded")
	}
	dc := oneEvent[DispatchCompleted](t, track.since())
	if dc.DispatchID != d.ID() || dc.SourceQuoteID != sourceQuote {
		t.Fatal("DispatchCompleted payload mismatch")
	}
}

func TestVerify(t *testing.T) {
	d := newDispatch()
	assignmentID := completeDispatch(t, d)
	track := trackFrom(d)
	mustNoErr(t, d.Verify())

	if d.Status() != StatusVerified {
		t.Fatalf("status = %v, want verified", d.Status())
	}
	if a := d.Assignment(assignmentID); a == nil || a.Status() != AssignmentConcluded {
		t.Fatal("assignment should be concluded")
	}
	dv := oneEvent[DispatchVerified](t, track.since())
	if dv.DispatchID != d.ID() || dv.SourceQuoteID != sourceQuote {
		t.Fatal("DispatchVerified payload mismatch")
	}
}

func TestVerificationRequiresCompletion(t *testing.T) {
	d := newDispatch()
	startDispatch(t, d) // in progress, not completed
	err := d.Verify()
	if !errors.Is(err, ErrDispatchNotCompleted) {
		t.Fatalf("err = %v, want ErrDispatchNotCompleted", err)
	}
	if err.Error() != "Dispatch must be completed for action to be taken." {
		t.Fatalf("message = %q", err.Error())
	}
}

func TestStartRequiresApprovalAfterAnotherAssignment(t *testing.T) {
	d := newDispatch()
	d.Delegate(dispatcher)
	_, err := d.Assign(&driver, &truck)
	mustNoErr(t, err)
	mustNoErr(t, d.Approve())
	_, err = d.Assign(&otherDriver, &otherTruck) // drops back to queued
	mustNoErr(t, err)
	if err := d.Start(); !errors.Is(err, ErrDispatchNotReady) {
		t.Fatalf("err = %v, want ErrDispatchNotReady", err)
	}
}

func TestStartRequiresApproval(t *testing.T) {
	d := newDispatch()
	if err := d.Start(); !errors.Is(err, ErrDispatchNotReady) {
		t.Fatalf("err = %v, want ErrDispatchNotReady", err)
	}
}

func TestAssignRequiresNewAssignee(t *testing.T) {
	d := newDispatch()
	_, err := d.Assign(&driver, &truck)
	mustNoErr(t, err)
	_, err = d.Assign(&driver, &truck)
	if !errors.Is(err, ErrDispatchAlreadyAssignedAssignee) {
		t.Fatalf("err = %v, want ErrDispatchAlreadyAssignedAssignee", err)
	}
}

func TestAssignNotAllowedWhileInProgress(t *testing.T) {
	d := newDispatch()
	startDispatch(t, d)
	_, err := d.Assign(&otherDriver, &otherTruck)
	if !errors.Is(err, ErrDispatchAssignedWhileInProgress) {
		t.Fatalf("err = %v, want ErrDispatchAssignedWhileInProgress", err)
	}
}

func TestAssignNotAllowedAfterCompletion(t *testing.T) {
	d := newDispatch()
	completeDispatch(t, d)
	_, err := d.Assign(&otherDriver, &otherTruck)
	if !errors.Is(err, ErrDispatchAssignedAfterCompletion) {
		t.Fatalf("err = %v, want ErrDispatchAssignedAfterCompletion", err)
	}
}

// Every execution action must be rejected before the dispatch is started.
func TestActionRequiresStartedWhenQueued(t *testing.T) {
	actions := map[string]func(*Dispatch) error{
		"pick_up_container":            func(d *Dispatch) error { _, e := d.PickUpContainer(); return e },
		"pick_up_premounted_container": func(d *Dispatch) error { _, e := d.PickUpPremountedContainer(chassis); return e },
		"deliver_container":            func(d *Dispatch) error { _, e := d.DeliverContainer(); return e },
		"hold":                         func(d *Dispatch) error { _, e := d.Hold(); return e },
		"take_container":               func(d *Dispatch) error { _, e := d.TakeContainer(); return e },
		"acquire_chassis":              func(d *Dispatch) error { _, e := d.AcquireChassis(chassisPickupYard, chassis); return e },
		"drop_container":               func(d *Dispatch) error { _, e := d.DropContainer(); return e },
		"drop_premounted_container":    func(d *Dispatch) error { _, e := d.DropPremountedContainer(); return e },
		"return_chassis":               func(d *Dispatch) error { _, e := d.ReturnChassis(chassisDropoffYard); return e },
	}
	for name, invoke := range actions {
		t.Run(name, func(t *testing.T) {
			d := newDispatch()
			if err := invoke(d); !errors.Is(err, ErrDispatchNotInProgress) {
				t.Fatalf("err = %v, want ErrDispatchNotInProgress", err)
			}
		})
	}
}
