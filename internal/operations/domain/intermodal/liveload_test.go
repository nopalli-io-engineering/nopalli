package intermodal

// Live Load — one test per lifecycle transition (rdt: tests/test_live_load.py).
// Each advances a fresh dispatch to just before its step, starts a tracker,
// acts, and asserts only its step's events, status, and live equipment/assignee.

import (
	"testing"
)

// liveLoadFixture bundles a queued dispatch with the move it was built from, so
// tests can assert against the move's facilities.
type liveLoadFixture struct {
	d    *Dispatch
	move LiveLoad
}

var llContainer = mkContainer("MSCU1234567")

func newLiveLoad(t *testing.T) liveLoadFixture {
	t.Helper()
	move := LiveLoad{
		ContainerNumber:      llContainer,
		ContainerPickupRamp:  NewFacilityID(),
		ContainerDropoffRamp: NewFacilityID(),
		DockDoor:             NewFacilityID(),
	}
	return liveLoadFixture{d: Queue(&sourceQuote, move), move: move}
}

// arrange: advance to just before each step (no assertions).
func (f liveLoadFixture) toApprove(t *testing.T) {
	t.Helper()
	f.d.Delegate(dispatcher)
	_, err := f.d.Assign(&driver, &truck)
	mustNoErr(t, err)
}
func (f liveLoadFixture) toStart(t *testing.T) {
	t.Helper()
	f.toApprove(t)
	mustNoErr(t, f.d.Approve())
}
func (f liveLoadFixture) toAcquireChassis(t *testing.T) {
	t.Helper()
	f.toStart(t)
	mustNoErr(t, f.d.Start())
}
func (f liveLoadFixture) toPickUp(t *testing.T) {
	t.Helper()
	f.toAcquireChassis(t)
	_, err := f.d.AcquireChassis(chassisPickupYard, chassis)
	mustNoErr(t, err)
}
func (f liveLoadFixture) toDeliver(t *testing.T) {
	t.Helper()
	f.toPickUp(t)
	_, err := f.d.PickUpContainer()
	mustNoErr(t, err)
}
func (f liveLoadFixture) toHold(t *testing.T) {
	t.Helper()
	f.toDeliver(t)
	_, err := f.d.DeliverContainer()
	mustNoErr(t, err)
}
func (f liveLoadFixture) toTake(t *testing.T) {
	t.Helper()
	f.toHold(t)
	_, err := f.d.Hold()
	mustNoErr(t, err)
}
func (f liveLoadFixture) toDrop(t *testing.T) {
	t.Helper()
	f.toTake(t)
	_, err := f.d.TakeContainer()
	mustNoErr(t, err)
}
func (f liveLoadFixture) toReturnChassis(t *testing.T) {
	t.Helper()
	f.toDrop(t)
	_, err := f.d.DropContainer()
	mustNoErr(t, err)
}

func requireEquipment(t *testing.T, d *Dispatch, want Equipment) {
	t.Helper()
	got, ok := d.LiveEquipment()
	if !ok || got != want {
		t.Fatalf("live equipment = %v,%v, want %v", got, ok, want)
	}
}

func requireNoEquipment(t *testing.T, d *Dispatch) {
	t.Helper()
	if _, ok := d.LiveEquipment(); ok {
		t.Fatal("expected no live equipment")
	}
}

func requireLiveAssignee(t *testing.T, d *Dispatch) {
	t.Helper()
	if got, ok := d.LiveAssignee(); !ok || got != assignee() {
		t.Fatalf("live assignee = %v,%v", got, ok)
	}
}

var (
	fullEquipment    = Equipment{Chassis: chassis, Container: llContainer}
	partialEquipment = Equipment{Chassis: chassis}
)

func TestLiveLoad_Queue(t *testing.T) {
	f := newLiveLoad(t)
	if f.d.Status() != StatusQueued {
		t.Fatalf("status = %v, want queued", f.d.Status())
	}
	dq := oneEvent[DispatchQueued](t, f.d.Events())
	if dq.DispatchID != f.d.ID() || dq.SourceQuoteID != sourceQuote {
		t.Fatal("DispatchQueued payload mismatch")
	}
}

func TestLiveLoad_AcquireChassis(t *testing.T) {
	f := newLiveLoad(t)
	f.toAcquireChassis(t)
	track := trackFrom(f.d)
	id, err := f.d.AcquireChassis(chassisPickupYard, chassis)
	mustNoErr(t, err)

	requireEquipment(t, f.d, partialEquipment)
	requireLiveAssignee(t, f.d)
	assertTaskCompletedBy(t, f.d, id, assignee())
	ev := track.since()
	assertTypeNames(t, ev, "ChassisAcquired")
	acq := oneEvent[ChassisAcquired](t, ev)
	if acq.DispatchID != f.d.ID() || acq.FacilityID != chassisPickupYard || acq.ChassisNumber != chassis {
		t.Fatal("ChassisAcquired payload mismatch")
	}
}

func TestLiveLoad_PickUpContainer(t *testing.T) {
	f := newLiveLoad(t)
	f.toPickUp(t)
	track := trackFrom(f.d)
	id, err := f.d.PickUpContainer()
	mustNoErr(t, err)

	requireEquipment(t, f.d, fullEquipment)
	assertTaskCompletedBy(t, f.d, id, assignee())
	ev := track.since()
	assertTypeNames(t, ev, "EmptyContainerPickedUp")
	picked := oneEvent[EmptyContainerPickedUp](t, ev)
	if picked.FacilityID != f.move.ContainerPickupRamp || picked.ContainerNumber != llContainer {
		t.Fatal("EmptyContainerPickedUp payload mismatch")
	}
}

func TestLiveLoad_DeliverContainer(t *testing.T) {
	f := newLiveLoad(t)
	f.toDeliver(t)
	track := trackFrom(f.d)
	id, err := f.d.DeliverContainer()
	mustNoErr(t, err)

	requireEquipment(t, f.d, fullEquipment)
	assertTaskCompletedBy(t, f.d, id, assignee())
	ev := track.since()
	assertTypeNames(t, ev, "EmptyContainerDelivered")
	del := oneEvent[EmptyContainerDelivered](t, ev)
	if del.FacilityID != f.move.DockDoor || del.ContainerNumber != llContainer {
		t.Fatal("EmptyContainerDelivered payload mismatch")
	}
}

func TestLiveLoad_Hold(t *testing.T) {
	f := newLiveLoad(t)
	f.toHold(t)
	track := trackFrom(f.d)
	id, err := f.d.Hold()
	mustNoErr(t, err)

	requireEquipment(t, f.d, fullEquipment)
	assertTaskCompletedBy(t, f.d, id, assignee())
	ev := track.since()
	assertTypeNames(t, ev, "DispatchHeldForLoading")
	held := oneEvent[DispatchHeldForLoading](t, ev)
	if held.FacilityID != f.move.DockDoor {
		t.Fatal("DispatchHeldForLoading payload mismatch")
	}
}

func TestLiveLoad_TakeReleasesHold(t *testing.T) {
	f := newLiveLoad(t)
	f.toTake(t)
	track := trackFrom(f.d)
	id, err := f.d.TakeContainer()
	mustNoErr(t, err)

	requireEquipment(t, f.d, fullEquipment)
	assertTaskCompletedBy(t, f.d, id, assignee())
	ev := track.since()
	assertTypeNames(t, ev, "LoadedContainerTaken", "DispatchReleased")
	taken := oneEvent[LoadedContainerTaken](t, ev)
	if taken.FacilityID != f.move.DockDoor || taken.ContainerNumber != llContainer {
		t.Fatal("LoadedContainerTaken payload mismatch")
	}
	if oneEvent[DispatchReleased](t, ev).FacilityID != f.move.DockDoor {
		t.Fatal("DispatchReleased payload mismatch")
	}
}

func TestLiveLoad_DropClosesCustody(t *testing.T) {
	f := newLiveLoad(t)
	f.toDrop(t)
	track := trackFrom(f.d)
	id, err := f.d.DropContainer()
	mustNoErr(t, err)

	requireEquipment(t, f.d, partialEquipment) // chassis still open
	assertTaskCompletedBy(t, f.d, id, assignee())
	ev := track.since()
	assertTypeNames(t, ev, "LoadedContainerDropped")
	ld := oneEvent[LoadedContainerDropped](t, ev)
	if ld.FacilityID != f.move.ContainerDropoffRamp || ld.ContainerNumber != llContainer {
		t.Fatal("LoadedContainerDropped payload mismatch")
	}
	if f.d.Status() != StatusInProgress {
		t.Fatalf("status = %v, want in_progress (not complete — chassis open)", f.d.Status())
	}
}

func TestLiveLoad_ReturnChassisCompletes(t *testing.T) {
	f := newLiveLoad(t)
	f.toReturnChassis(t)
	track := trackFrom(f.d)
	id, err := f.d.ReturnChassis(chassisDropoffYard)
	mustNoErr(t, err)

	requireNoEquipment(t, f.d)
	requireNoLiveAssignee(t, f.d)
	requireNoAssignee(t, f.d)
	assertTaskCompletedBy(t, f.d, id, assignee())
	ev := track.since()
	assertTypeNames(t, ev, "ChassisReturned", "DispatchCompleted")
	ret := oneEvent[ChassisReturned](t, ev)
	if ret.FacilityID != chassisDropoffYard || ret.ChassisNumber != chassis {
		t.Fatal("ChassisReturned payload mismatch")
	}
	if oneEvent[DispatchCompleted](t, ev).DispatchID != f.d.ID() {
		t.Fatal("DispatchCompleted payload mismatch")
	}
	if f.d.Status() != StatusCompleted {
		t.Fatalf("status = %v, want completed", f.d.Status())
	}
}

func TestLiveLoad_FullLifecycleReachesCompleted(t *testing.T) {
	f := newLiveLoad(t)
	f.toReturnChassis(t)
	_, err := f.d.ReturnChassis(chassisDropoffYard)
	mustNoErr(t, err)
	requireNoEquipment(t, f.d)
	if f.d.Status() != StatusCompleted {
		t.Fatalf("status = %v, want completed", f.d.Status())
	}
	names := eventTypeNames(f.d.Events())
	if names[len(names)-1] != "DispatchCompleted" {
		t.Fatalf("last event = %q, want DispatchCompleted", names[len(names)-1])
	}
}
